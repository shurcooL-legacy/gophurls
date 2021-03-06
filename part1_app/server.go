package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"

	. "gist.github.com/5286084.git"
)

var _ = CheckError

var httpAddr = flag.String("http", ":7000", "HTTP service address")

var data struct {
	Urls []UrlTitle
	sync.RWMutex
}

var t = template.Must(template.New("name").Parse(`<html><h1>GophURLs (shurcooL)</h1>
<h2>Links</h2>
<ol>
  {{range .Urls}}<li><a href="{{.URL}}">{{.Title}}</a></li>
{{end}}
</ol></html>`))

func home(w http.ResponseWriter, r *http.Request) {
	//io.WriteString(w, "Links example.com")
	data.RLock()
	defer data.RUnlock()
	t.Execute(w, data)
}

type UrlTitle struct {
	URL   string
	Title string
}

// extract returns the recursive concatenation of the raw text contents of an html node, with Markdown tags.
func extract(n *html.Node) (out string) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			out += c.Data
		} else if c.Type == html.ElementNode && c.DataAtom == atom.Blockquote {
			out += "> " + extract(c)
		} else if c.Type == html.ElementNode && (c.DataAtom == atom.B || c.DataAtom == atom.Strong) {
			out += "**" + extract(c) + "**"
		} else {
			out += extract(c)
		}
	}
	return out
}

func lookupTitle(url string) (title string) {
	r, err := http.Get(url)
	if err != nil {
		return "<Couldn't connect.>"
	}
	defer r.Body.Close()
	/*b, err := ioutil.ReadAll(r.Body)
	CheckError(err)
	if len(b) > 30 {
		b = b[:30]
	}
	return string(b)*/

	title = "<Untitled page.>"

	doc, err := html.Parse(r.Body)
	if err != nil {
		return "<Failed to parse HTML.>"
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Title {
			title = extract(n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return
}

func addLink(url UrlTitle) {
	if url.Title == "" {
		url.Title = lookupTitle(url.URL)
	}

	data.Lock()
	defer data.Unlock()
	data.Urls = append(data.Urls, url)
}

func links(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		dec := json.NewDecoder(r.Body)
		var url UrlTitle
		if err := dec.Decode(&url); err != io.EOF && err != nil {
			log.Fatal(err)
		}

		addLink(url)
	}
}

func init() {
	// Set up the HTTP handler in init (not main) so we can test it. (This main
	// doesn't run when testing.)
	http.HandleFunc("/", home)
	http.HandleFunc("/links", links)
}

func main() {
	flag.Parse()
	if err := http.ListenAndServe(*httpAddr, nil); err != nil {
		log.Fatal(err)
	}
}
