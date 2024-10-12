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
			if a.Key == "id" && strings.HasSuffix(a.Val, "[Gruppe]") {
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

			// have to catch this from within the p case:
			if c.DataAtom.String() == "span" {
				for _, a := range c.Attr {
					if a.Key == "class" && a.Val == "bold" {
						for span := c.FirstChild; span != nil; span = span.NextSibling {
							if span.Type == html.TextNode {
								out += fmt.Sprintln("<em>" + span.Data + "</em>")
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
	// cmd := exec.Command("pandoc", "--wrap=none", "--from=html", "--to=markdown-smart", "--output=test.md")
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

func Cleanup(content string) (out string) {
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

func main() {
	dat, err := os.ReadFile(SOURCE)
	if err != nil {
		panic(err)
	}
	r := bytes.NewReader(dat)

	doc, err := html.Parse(r)
	if err != nil {
		panic(err)
	}
	out := ProcessNode(doc)
	md := RunPandoc(out)
	md = Cleanup(md)

	books := MapHKA()
	fmt.Println(books["15 = W II 6a. Frühjahr 1888"])
	panic("foo")

	fmt.Println(md)
}
