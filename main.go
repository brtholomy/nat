package main

import (
	"bytes"
	"fmt"
	"os"

	"golang.org/x/net/html"
)

const SOURCE = "NF-1888,15.html"

func ProcessNode(n *html.Node) {
	switch n.Data {

	// heading
	case "a":
		for _, a := range n.Attr {
			if a.Key == "data-link" {
				fmt.Println("<h2>", a.Val, "</h2>")
			}
		}

	case "p":
		fmt.Println("<p>")
		for c := n.FirstChild; c != nil; c = c.NextSibling {

			if c.Type == html.TextNode {
				fmt.Println(c.Data)
			}

			// have to catch this from within the p case:
			if c.DataAtom.String() == "span" {
				for _, a := range c.Attr {
					if a.Key == "class" && a.Val == "bold" {
						for span := c.FirstChild; span != nil; span = span.NextSibling {
							if span.Type == html.TextNode {
								fmt.Println("<em>" + span.Data + "</em>")
							}
						}
					}
				}
			}
		}
		fmt.Println("</p>")

	}

	// Traverse child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ProcessNode(c)
	}
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

	ProcessNode(doc)
}
