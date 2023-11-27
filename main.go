package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"siteLink/link"
	"strings"
	"sync"
)

var linkMap = make(map[string]int)
var wg sync.WaitGroup
var mu sync.Mutex

func main() {
	url := flag.String("url", "", "the root url to begin the site mapping from (only urls of this domain will be searched)")
	maxDepth := flag.Int("maxDepth", 4, "the amount of pages to search in from the root url")
	flag.Parse()
	// Transfer response.Body (which is an io.ReadCloser) to an io.Reader variable.
	body, err := getHtml(*url)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%d\n", *maxDepth)
	// To test if the transfer worked, let's copy from our reader to stdout (os.Stdout).
	// In a practical scenario, you might use the reader for something else.
	//_, err = io.Copy(os.Stdout, reader)
	// if err != nil {
	// 	fmt.Println("Error reading response body:", err)
	// }
	links, err := gatherLinks(body, *url)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%#v", links)
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
		linkMap[link.Href]++
		if linkMap[link.Href] < 1 {
			mu.Unlock()
			wg.Add(1)
			go mapSite(link.Href, maxDepth)
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

func gatherLinks(body io.ReadCloser, domain string) ([]link.Link, error) {
	defer body.Close()
	links, err := link.Parse(body)
	domainUrls := make([]link.Link, 4)
	if err != nil {
		fmt.Println("Error parsing html", err)
		return links, err
	}
	for _, link := range links {
		if strings.HasPrefix(link.Href, "/") {
			fmt.Printf("%s\n", link.Href)
			link.Href = fmt.Sprintf("%s%s", domain, link.Href)
			domainUrls = append(domainUrls, link)
		} else if strings.HasPrefix(link.Href, domain) {
			fmt.Printf("%s\n", link.Href)

			domainUrls = append(domainUrls, link)
		}
	}
	return domainUrls, nil
}
