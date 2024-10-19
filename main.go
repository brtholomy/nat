package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const HKA_SOURCE = "sources/HKA.txt"

const NF_GLOB = "sources/html/NF-*.html"
const WERKE_GLOB = "sources/html/W-*.html"

const MOCK_NF = "sources/mock/NF-1885,39.html"
const MOCK_WERKE = "sources/mock/WA.html"

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
	doc.Find("div.head").Each(func(i int, s *goquery.Selection) {
		// leaving h2 as it appears in the div.titel
		s.Find("h2").Remove()
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
	// in the case of published works:
	title := doc.Find("div.titel")
	// remove anchors
	title.Find("a").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})
	// we get the whole block, since it might contain h2s
	title_html, err := doc.Find("div.titel").Html()
	if err != nil {
		panic(err)
	}

	// or the Nachlass:
	p := doc.Find("p.Gruppe").Last().Text()
	if title_html != "" {
		ekgw.h1 = title_html
	} else {
		ekgw.h1 = "<h1>" + p + "</h1>"
	}

	// entries : h2 and html
	doc.Find("div.txt_block").Each(func(i int, s *goquery.Selection) {
		var e Entry
		// the first anchor seems to be the most reliable place to get the h2:
		// .Attr() will just get the value from the first element:
		id, ok := s.Find("a[name]").Attr("name")
		if !ok || strings.Contains(id, "Gruppe") {
			return
		}
		e.h2 = "<h2>" + id + "</h2>"

		// remove all extraneous stuff now that we have what we want:
		s.Find("a").Each(func(i int, s *goquery.Selection) {
			s.Remove()
		})
		s.Find("h2").Each(func(i int, s *goquery.Selection) {
			s.Remove()
		})
		s.Find("h3").Each(func(i int, s *goquery.Selection) {
			s.Remove()
		})

		// then get the rest:
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
	content = bytes.ReplaceAll(content, []byte("&lt;"), []byte(""))
	content = bytes.ReplaceAll(content, []byte("&gt;"), []byte(""))
	return content
}

func CleanupMd(content string) (out string) {
	out = strings.ReplaceAll(content, `\`, "")
	out = strings.ReplaceAll(out, `#eKGWB`, "eKGWB")
	out = strings.ReplaceAll(out, ` `, " ")
	// left behind by empty divs and whatnot:
	out = strings.ReplaceAll(out, "\n\n\n\n", "\n\n")
	return out
}

func RunPandoc(content string) string {
	cmd := exec.Command("pandoc", "--wrap=none", "--from=html-native_divs-native_spans", "--to=markdown-smart")

	// https://pkg.go.dev/os/exec#Cmd.StdoutPipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	// pipe it in:
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, content)
	}()

	// get stdout:
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	return string(out)
}

// Creates a map from a notebook name to a set of aphorism citations.
//
// The somwhat odd and arbitrary notebook naming, which might date back to the original Nietzsche
// archive, is the only reliable link between the HKA and the eKGWB. Here we're trying to create a
// map from those names to the aphorism citations of the HKA, which are systematic and consistent,
// so that the eKGWB mappings, which are consistent but incomplete and largely useless, can be
// replaced.
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
		book = TrimBookName(book)

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

// Transforms both the HKA and eKGWB book listings into a minimal string, ignoring:
// wrapping [], all whitespace, and case. Cuts at first "."
//
// so this:
// [ 31 = Z II 8. Winter 1884 — 85 ]
// becomes:
// 31=zii8
func TrimBookName(book string) string {
	// 31 = Z II 8. Winter 1884 — 85
	book = strings.Trim(book, "[] ")
	// up to the first period:
	// 31 = Z II 8
	book, _, _ = strings.Cut(book, ".")
	// collapse whitespace:
	book = strings.ReplaceAll(book, " ", "")
	book = strings.ToLower(book)
	return book
}

// Find the unique id mapping the eKGWB to the HKA by progressively shortening the key. See MapHKA()
//
// TODO: manually fix the outliers. eKGWB differs in about 20 books. Unless I can fuzzy match.
// https://github.com/lithammer/fuzzysearch
func FindAphorisms(books map[string][]string, book string) (aphs []string, found bool) {
	origbook := book
	book = TrimBookName(book)

	short_book := book
	aphs, ok := books[book]
	for i := 1; !ok && len(short_book) > 2; i++ {
		short_book = book[:len(book)-i]
		for key, _ := range books {
			j := len(key) - i
			if j < 2 {
				continue
			}
			short_key := key[:j]
			if short_book == short_key {
				aphs, ok = books[key]
				log.Println("INFO: found book by shortening:", short_key, "original:", origbook)
				break
			}
		}
	}
	if !ok {
		log.Println("WARN: didn't find the book within the books map:", origbook)
		return nil, false
	}
	return aphs, true
}

// takes the markdown rendered string and replaces the bullshit eKGWB citations with the proper KGW
// numbers mapped from the HKA.
func AnnotateKGW(markdown string, books map[string][]string, book_rx *regexp.Regexp, aphorism_rx *regexp.Regexp) string {
	book_match := book_rx.FindStringSubmatch(markdown)
	if book_match == nil || len(book_match) < 2 {
		log.Println("WARN: didn't find the book within the markdown", markdown[:10])
		return markdown
	}
	// get the submatch only:
	book := book_match[1]
	aphs, ok := FindAphorisms(books, book)
	if !ok {
		return markdown
	}

	out := markdown
	h2s := aphorism_rx.FindAllString(markdown, -1)
	for i, header := range h2s {
		_, number, ok := strings.Cut(header, ",")
		if !ok {
			log.Println("WARN: didn't find the header number in the header", header)
			continue
		}

		// NOTE: happens when the eKGWB combines multiple books, eg NF-1884,28.html:
		if i >= len(aphs) {
			log.Printf("WARN: more eKGW h2 headers: %v found than HKA headers: %v. %v", len(h2s), len(aphs), book)
			break
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

func ProcessGlob(glob string, outdir string) {
	books := MapHKA()
	// # [15 = W II 6a. Frühjahr 1888]
	md_book_rx, _ := regexp.Compile(`(?m)^# \[(.+)\]$`)
	// ## eKGWB/NF-1888,15[1]
	// not:
	// ## eKGWB/NF-1888,15[Titel]
	md_aphorism_rx, _ := regexp.Compile(`(?m)^## eKGWB/.*,[0-9]+\[[0-9]+\]$`)

	files, err := filepath.Glob(glob)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		dat, err := os.ReadFile(f)
		if err != nil {
			panic(err)
		}

		log.Println("INFO: processing", f)
		dat = PreCleanupHtml(dat)
		r := bytes.NewReader(dat)

		doc, err := goquery.NewDocumentFromReader(r)
		if err != nil {
			panic(err)
		}
		ekgw := ParseWithGoquery(doc)
		out := Render(ekgw)

		md := RunPandoc(out)
		md = CleanupMd(md)
		md = AnnotateKGW(md, books, md_book_rx, md_aphorism_rx)

		mdname := filepath.Join(outdir, strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))+".md")
		f, err := os.Create(mdname)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		_, err = f.WriteString(md)
		if err != nil {
			panic(err)
		}
		log.Println("INFO: wrote", mdname)
	}
}

func main() {
	log.SetFlags(log.LstdFlags ^ log.Ldate ^ log.Ltime)
	ProcessGlob(NF_GLOB, "./output/")
	ProcessGlob(WERKE_GLOB, "./output/")
}
