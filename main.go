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

	"golang.org/x/net/html"
)

const SOURCE = "sources/mock/NF-1888,15.html"
const HKA_SOURCE = "sources/HKA.txt"

func ProcessNode(n *html.Node) (out string) {
	switch n.Data {

	// title
	case "div":
		for _, a := range n.Attr {
			if a.Key == "id" && (strings.HasSuffix(a.Val, "[Gruppe]") || strings.HasSuffix(a.Val, "[Titel]")) {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Data == "div" {
						for c2 := c.FirstChild; c2 != nil; c2 = c2.NextSibling {
							if c2.Data == "p" && c2.FirstChild.Type == html.TextNode {
								out += fmt.Sprintln("<h1>", c2.FirstChild.Data, "</h1>")
								// HACK: to prevent it from printing again in the "p" case:
								c2.FirstChild = nil
							}
						}
					}
				}
			}
		}

	// aphorism heading
	case "a":
		for _, a := range n.Attr {
			if a.Key == "data-link" {
				out += fmt.Sprintln("<h2>", a.Val, "</h2>")
			}
		}

	case "p":
		out += fmt.Sprintln("<p>")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				out += fmt.Sprintln(c.Data)
			}

			// span : have to catch this from within the p case:
			if c.Type == html.ElementNode && c.Data == "span" {
				for _, a := range c.Attr {
					// bold : <em>
					if a.Key == "class" && a.Val == "bold" {
						for span := c.FirstChild; span != nil; span = span.NextSibling {
							if span.Type == html.TextNode {
								out += fmt.Sprintln("<em>" + span.Data + "</em>")
							}
						}
					}

					// handle <span style="position:relative"><span class="tooltip_corrige">text
					if a.Key == "style" && a.Val == "position:relative" {
						for corrige := c.FirstChild; corrige != nil; corrige = corrige.NextSibling {
							if corrige.Attr != nil && corrige.Attr[0].Val == "tooltip_corrige" && corrige.FirstChild.Type == html.TextNode {
								out += fmt.Sprintln(corrige.FirstChild.Data)
							}
						}
					}
				}
			}
		}
		out += fmt.Sprintln("</p>")
	}

	// Traverse child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out += ProcessNode(c)
	}
	return out
}

func RunPandoc(content string) string {
	cmd := exec.Command("pandoc", "--wrap=none", "--from=html", "--to=markdown-smart")

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

func CleanupHtml(content []byte) []byte {
	crapdiv, _ := regexp.Compile(`(?s)<div class="tooltip" style="position: absolute;.*?</span>`)
	content = crapdiv.ReplaceAll(content, []byte("</span>"))
	content = bytes.ReplaceAll(content, []byte("&lt;"), []byte(""))
	content = bytes.ReplaceAll(content, []byte("&gt;"), []byte(""))
	return content
}

func CleanupMd(content string) (out string) {
	out = strings.ReplaceAll(content, `\`, "")
	out = strings.ReplaceAll(out, `#eKGWB`, "eKGWB")
	return out
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
	dat = CleanupHtml(dat)
	r := bytes.NewReader(dat)
	doc, err := html.Parse(r)
	if err != nil {
		panic(err)
	}
	out := ProcessNode(doc)
	md := RunPandoc(out)
	md = CleanupMd(md)

	books := MapHKA()
	md = AnnotateKGW(md, books)

	fmt.Println(md)
}
