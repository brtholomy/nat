	// var f func(*html.Node)
	// f = func(n *html.Node) {
	// 	if n.Type == html.ElementNode && n.Data == "p" {
	// 		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
	// 			// if yes, retrieve FirstChild's data (name)
	// 			data := n.FirstChild.Data
	// 			// print name
	// 			fmt.Println(data)
	// 		}
	// 	}
	// 	for c := n.FirstChild; c != nil; c = c.NextSibling {
	// 		f(c)
	// 	}
	// }
	// f(doc)

	// depth := 0
	// for {
	// 	tt := z.Next()
	// 	switch tt {
	// 	case html.ErrorToken:
	// 		return z.Err()
	// 	case html.TextToken:
	// 		if depth > 0 {
	// 			// emitBytes should copy the []byte it receives,
	// 			// if it doesn't process it immediately.
	// 			emitBytes(z.Text())
	// 		}
	// 	case html.StartTagToken, html.EndTagToken:
	// 		tn, _ := z.TagName()
	// 		if len(tn) == 1 && tn[0] == 'a' {
	// 			if tt == html.StartTagToken {
	// 				depth++
	// 			} else {
	// 				depth--
	// 			}
	// 		}
	// 	}
	// }

	// case "span":
	// 	for _, a := range n.Attr {
	// 		if a.Key == "class" && a.Val == "bold" {
	// 			for c := n.FirstChild; c != nil; c = c.NextSibling {
	// 				if c.Type == html.TextNode {
	// 					fmt.Println("bold:", c.Data)
	// 				}
	// 			}
	// 		}
	// 	}
