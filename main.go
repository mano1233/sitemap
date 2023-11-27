package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"siteLink/link"
)

func main() {
	url := flag.String("url", "", "the root url to begin the site mapping from (only urls of this domain will be searched)")
	maxDepth := flag.Int("maxDepth", 4, "the amount of pages to search in from the root url")
	flag.Parse()
	response, err := http.Get(*url)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer response.Body.Close()
	fmt.Printf("%#v", maxDepth)
	var r io.Reader
	link.Parse(r)
}
