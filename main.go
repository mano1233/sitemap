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

var mu sync.Mutex

func main() {
	urlSet := URLSet{
		Xmlns: xmlns,
	}
	url := flag.String("url", "", "the root url to begin the site mapping from (only urls of this domain will be searched)")
	maxDepth := flag.Int("depth", 4, "the amount of pages to search in from the root url")
	flag.Parse()
	var wg sync.WaitGroup
	// Transfer response.Body (which is an io.ReadCloser) to an io.Reader variable.
	wg.Add(1)
	go mapSite(*url, *url, *maxDepth, &wg)
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

func mapSite(url, base string, maxDepth int, wg *sync.WaitGroup) {
	defer wg.Done()
	if maxDepth == 0 {
		return
	}
	maxDepth = maxDepth - 1
	body, err := getHtml(url)
	if err != nil {
		fmt.Println(err)
	}
	links, err := gatherLinks(body, base)
	if err != nil {
		fmt.Println(err)
	}
	for _, link := range links {
		if link != "" {
			mu.Lock()
			linkMap[link] = linkMap[link] + 1
			if linkMap[link] == 1 {
				mu.Unlock()
				wg.Add(1)
				go mapSite(link, base, maxDepth, wg)
			} else {
				mu.Unlock()
			}

		}
	}
	fmt.Printf("%#v\n", links)
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
	links, _ := link.Parse(body)
	domainUrls := make([]string, 1)
	for _, link := range links {
		if link.Href != "" && strings.HasPrefix(link.Href, "/") {
			link.Href = fmt.Sprintf("%s%s", domain, link.Href)
			domainUrls = append(domainUrls, link.Href)
		} else if link.Href != "" && strings.HasPrefix(link.Href, domain) {
			fmt.Printf("%s\n", link)
			domainUrls = append(domainUrls, link.Href)
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
