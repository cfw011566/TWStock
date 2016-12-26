package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"net/http"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var enc = traditionalchinese.Big5

/* Revenue URL
http://mops.twse.com.tw/nas/t21/sii/t21sc03_105_10_0.html
http://mops.twse.com.tw/nas/t21/sii/t21sc03_105_10_1.html
http://mops.twse.com.tw/nas/t21/otc/t21sc03_105_10_0.html
http://mops.twse.com.tw/nas/t21/otc/t21sc03_105_10_1.html
http://mops.twse.com.tw/nas/t21/rotc/t21sc03_105_10_0.html
http://mops.twse.com.tw/nas/t21/rotc/t21sc03_105_10_1.html
http://mops.twse.com.tw/nas/t21/pub/t21sc03_105_10_0.html
http://mops.twse.com.tw/nas/t21/pub/t21sc03_105_10_1.html
*/

const (
	Domestic = iota
	Foreign
)

var markets = [...]string{"sii", "otc", "rotc", "pub"}

var revenueURLBase = "http://mops.twse.com.tw/nas/t21/%s/t21sc03_%d_%d_%d.html"

func main() {
	var revenueURLs []string

	year := time.Now().Year()
	month := time.Now().Month()
	day := time.Now().Day()
	if day < 10 {
		month -= 1
		if month <= 0 {
			month += 12
			year -= 1
		}
	}
	year -= 1911

	for _, market := range markets {
		url := fmt.Sprintf(revenueURLBase, market, year, month, 0)
		revenueURLs = append(revenueURLs, url)
		url = fmt.Sprintf(revenueURLBase, market, year, month, 1)
		revenueURLs = append(revenueURLs, url)
	}

	for _, url := range revenueURLs {
		//fmt.Println(url)
		revenue(url)
	}
}

func revenue(url string) {
	resp, err := http.Get(url)
	if err != nil {
		//panic(err)
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("status no OK")
		return
	}

	r := transform.NewReader(resp.Body, enc.NewDecoder())

	doc, err := html.Parse(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "htmlparser: %v\n", err)
		os.Exit(1)
	}
	//outline(nil, doc)
	//showText(nil, doc)
	noTotal(nil, doc)
}

func outline(stack []string, n *html.Node) {
	if n.Type == html.ElementNode {
		stack = append(stack, n.Data) // push tag
		fmt.Println(stack)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		outline(stack, c)
	}
}

func showText(stack []string, n *html.Node) {
	if n.Type == html.ElementNode {
		stack = append(stack, n.Data)
	}
	if n.Type == html.TextNode {
		fmt.Println(stack)
		fmt.Println(n.Data)
		/*
			if pathFound(stack) {
				fmt.Println(n.Data)
			}
		*/
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		showText(stack, c)
	}
}

func noTotal(stack []string, n *html.Node) {
	if n.Type == html.ElementNode {
		stack = append(stack, n.Data)
	}
	if pathFound(stack) && n.FirstChild != nil {
		firstChild := n.FirstChild
		if firstChild.Type == html.ElementNode && firstChild.Data == "th" {
			return
		}
		if firstChild.Type == html.ElementNode && firstChild.Data == "td" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.FirstChild != nil && c.FirstChild.Type == html.TextNode {
					text := strings.Replace(c.FirstChild.Data, ",", "", -1)
					fmt.Print(strings.TrimSpace(text))
				}
				if c.NextSibling != nil {
					fmt.Print(",")
				}
			}
			fmt.Println()
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		noTotal(stack, c)
	}
}

// [html body center center table tbody tr td table tbody tr td table tbody tr td]

func pathFound(stack []string) bool {
	//	path := [...]string { "html", "body", "center", "center", "table", "tbody", "tr", "td", "table", "tbody", "tr", "td", "table", "tbody", "tr", "td" }
	path := [...]string{"html", "body", "center", "center", "table", "tbody", "tr", "td", "table", "tbody", "tr", "td", "table", "tbody", "tr"}
	if len(path) != len(stack) {
		return false
	}
	for i := range path {
		if path[i] != stack[i] {
			return false
		}
	}
	return true
}
