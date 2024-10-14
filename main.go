package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const MOCK_SOURCE = "sources/mock/NF-1888,15.html"
const SOURCE = "sources/html_original/NF-1888,14.html"
const HKA_SOURCE = "sources/HKA.txt"

// we expect later that all stored strings are already html.
type Entry struct {
	h2   string
	html string
}

// represents a complete eKGW grouping as downloaded from nietzschesource.org
type eKGWDoc struct {
	h1      string
	entries []Entry
}

func ParseWithGoquery(doc *goquery.Document) eKGWDoc {
	var ekgw eKGWDoc

	// clean it first
	doc.Find("div.tooltip").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})
	doc.Find("h2").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})
	doc.Find("span.bold").Each(func(i int, s *goquery.Selection) {
		s.ReplaceWithHtml("<em>" + s.Text() + "</em>")
	})
	doc.Find("span.bolditalic").Each(func(i int, s *goquery.Selection) {
		s.ReplaceWithHtml("<b>" + s.Text() + "</b>")
	})
	// TODO: replace with <ul> ?
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		s.ReplaceWithSelection(s.Find("div.p"))
	})

	// h1
	doc.Find("div.titel").Each(func(i int, s *goquery.Selection) {
		title, err := s.Html()
		if err != nil {
			panic(err)
		}
		ekgw.h1 = title
	})
	p := doc.Find("p.Gruppe").Text()
	ekgw.h1 = "<h1>" + p + "</h1>"

	// entries : h2 and html
	doc.Find("div.txt_block").Each(func(i int, s *goquery.Selection) {
		var e Entry
		id, ok := s.Find("div.div1").Attr("id")
		if !ok || strings.Contains(id, "Gruppe") {
			return
		}
		e.h2 = "<h2>" + id + "</h2>"

		inner, err := s.Html()
		if err != nil {
			panic(err)
		}
		e.html = inner
		ekgw.entries = append(ekgw.entries, e)
	})
	return ekgw
}

func Render(ekgw eKGWDoc) (out string) {
	out += fmt.Sprintln(ekgw.h1)
	for _, e := range ekgw.entries {
		out += fmt.Sprintln(e.h2)
		out += e.html
	}
	return out
}

func PreCleanupHtml(content []byte) []byte {
	// crapdiv, _ := regexp.Compile(`(?s)<div class="tooltip" style="position: absolute;.*?</span>`)
	// content = crapdiv.ReplaceAll(content, []byte("</span>"))
	content = bytes.ReplaceAll(content, []byte("&lt;"), []byte(""))
	content = bytes.ReplaceAll(content, []byte("&gt;"), []byte(""))
	return content
}

func CleanupMd(content string) (out string) {
	out = strings.ReplaceAll(content, `\`, "")
	out = strings.ReplaceAll(out, `#eKGWB`, "eKGWB")
	out = strings.ReplaceAll(out, ` `, " ")
	return out
}

func RunPandoc(content string) string {
	cmd := exec.Command("pandoc", "--wrap=none", "--from=html-native_divs-native_spans", "--to=markdown-smart")

	// https://pkg.go.dev/os/exec#Cmd.StdoutPipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	// pipe it in:
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, content)
	}()

	// get stdout:
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	return string(out)
}

func MapHKA() map[string][]string {
	books := map[string][]string{}
	// [ 30 = Z II 5, 83. Z II 7b. Z II 6b. Herbst 1884 — Anfang 1885 ]
	// [ 31 = Z II 8. Winter 1884 — 85 ]
	book_rx, _ := regexp.Compile(`(?m)^\[(.+)\]$`)
	// Aphorism n=9963 id='VII.31[1]' kgw='VII-3.71' ksa='11.359'
	aphorism_rx, _ := regexp.Compile(`(?m)^Aphorism .* kgw='.*' ksa='.*'$`)

	dat, err := os.ReadFile(HKA_SOURCE)
	if err != nil {
		panic(err)
	}
	s := string(dat)

	res := book_rx.FindAllStringIndex(s, -1)
	for j, indices := range res {
		// current match
		book := s[indices[0]:indices[1]]
		book = strings.TrimPrefix(book, "[ ")
		book = strings.TrimSuffix(book, " ]")

		// slice the whole to look forward for the Aphorism match
		end := len(s)
		if j+1 < len(res) {
			end = res[j+1][0]
		}
		sub := s[indices[1]:end]
		aph := aphorism_rx.FindAllString(sub, -1)
		books[book] = append(books[book], aph...)
	}
	return books
}

// takes the markdown rendered string and replaces the bullshit eKGWB citations with the proper KGW
// numbers mapped from the HKA.
func AnnotateKGW(markdown string, books map[string][]string) string {
	// # [15 = W II 6a. Frühjahr 1888]
	book_rx, _ := regexp.Compile(`(?m)^# \[(.+)\]$`)
	// ## eKGWB/NF-1888,15[1]
	aphorism_rx, _ := regexp.Compile(`(?m)^## eKGWB/.*,(.*)$`)
	book_match := book_rx.FindStringSubmatch(markdown)
	if book_match == nil {
		return markdown
	}

	// get the submatch only:
	aphs, ok := books[book_match[1]]
	if !ok {
		return markdown
	}

	out := markdown
	h2s := aphorism_rx.FindAllString(markdown, -1)
	for i, header := range h2s {
		_, number, ok := strings.Cut(header, ",")
		if !ok {
			continue
		}
		// NOTE: the index here is assumed to match the []string from the map:
		if strings.Contains(aphs[i], number) {
			// '## '
			//  012
			j := strings.Index(out, header) + 3
			aph := strings.TrimPrefix(aphs[i], "Aphorism ")
			// NOTE: j+len(header)-3 : effectively removes the eKGWB header
			// NOTE: we're not building back the markdown string, but interpolating:
			out = out[:j] + aph + out[j+len(header)-3:]
		}
	}
	return out
}

func main() {
	dat, err := os.ReadFile(SOURCE)
	if err != nil {
		panic(err)
	}
	dat = PreCleanupHtml(dat)
	r := bytes.NewReader(dat)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}
	ekgw := ParseWithGoquery(doc)
	out := Render(ekgw)

	md := RunPandoc(out)
	md = CleanupMd(md)

	books := MapHKA()
	md = AnnotateKGW(md, books)

	fmt.Println(md)
}
