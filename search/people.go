package search

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

func FindPeople(browser *rod.Browser, keyword string) ([]string, error) {
	page, err := OpenSearchPage(browser, "people", keyword)
	if err != nil {
		return nil, err
	}

	// Scroll to load results
	for i := 0; i < 5; i++ {
		page.Mouse.MustScroll(0, 2000)
		time.Sleep(2 * time.Second)
	}

	links, err := ExtractProfileLinks(page)
	if err != nil {
		return nil, err
	}

	fmt.Printf("👤 Found %d people profiles\n", len(links))
	return links, nil
}
