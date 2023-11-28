package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"siteLink/link"
	"strings"
	"sync"
)

// URLSet is a struct that holds a slice of URL entries
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL is a struct representing a single URL entry
type URL struct {
	Loc string `xml:"loc"`
}

const xmlns = "http://www.sitemaps.org/schemas/sitemap/0.9"

var linkMap = make(map[string]int)
var wg sync.WaitGroup
var mu sync.Mutex

func main() {
	urlSet := URLSet{
		Xmlns: xmlns,
	}
	url := flag.String("url", "", "the root url to begin the site mapping from (only urls of this domain will be searched)")
	maxDepth := flag.Int("maxDepth", 4, "the amount of pages to search in from the root url")
	flag.Parse()
	// Transfer response.Body (which is an io.ReadCloser) to an io.Reader variable.
	wg.Add(1)
	go mapSite(*url, *maxDepth)
	wg.Wait()
	for link, _ := range linkMap {
		urlSet.URLs = append(urlSet.URLs, URL{Loc: link})
	}

	xmlBytes, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling XML: %v\n", err)
		os.Exit(1)
	}

	// Output the XML declaration and the XML data
	xmlString := xml.Header + string(xmlBytes)
	fmt.Print(xmlString)
	// fmt.Printf("%#v", linkMap)
	// fmt.Printf("%#v", link)

}

func mapSite(url string, maxDepth int) {
	if maxDepth == 0 {
		return
	}
	maxDepth--
	body, err := getHtml(url)
	if err != nil {
		fmt.Println(err)
	}
	links, err := gatherLinks(body, url)
	if err != nil {
		fmt.Println(err)
	}
	for _, link := range links {
		mu.Lock()
		linkMap[link]++
		if linkMap[link] < 1 {
			mu.Unlock()
			wg.Add(1)
			go mapSite(link, maxDepth)
		} else {
			mu.Unlock()
		}
	}
	wg.Done()
}

func getHtml(url string) (io.ReadCloser, error) {
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Failed to get webpage with status code: %d\n", response.StatusCode)
		response.Body.Close()
		return nil, err
	}
	reader := response.Body
	return reader, nil
}

func gatherLinks(body io.ReadCloser, domain string) ([]string, error) {
	defer body.Close()
	links := hrefs(body, domain)
	domainUrls := make([]string, 4)
	for _, link := range links {
		if strings.HasPrefix(link, "/") {
			link = fmt.Sprintf("%s%s", domain, link)
			domainUrls = append(domainUrls, link)
		} else if strings.HasPrefix(link, domain) {
			fmt.Printf("%s\n", link)

			domainUrls = append(domainUrls, link)
		}
	}
	return domainUrls, nil
}

func hrefs(r io.Reader, base string) []string {
	links, _ := link.Parse(r)
	var ret []string
	for _, l := range links {
		switch {
		case strings.HasPrefix(l.Href, "/"):
			ret = append(ret, base+l.Href)
		case strings.HasPrefix(l.Href, "http"):
			ret = append(ret, l.Href)
		}
	}
	return ret
}
