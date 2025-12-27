package search

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

func FindCompanies(browser *rod.Browser, keyword string) ([]string, error) {
	page, err := OpenSearchPage(browser, "companies", keyword)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 5; i++ {
		page.Mouse.MustScroll(0, 2000)
		time.Sleep(2 * time.Second)
	}

	links, err := ExtractProfileLinks(page)
	if err != nil {
		return nil, err
	}

	fmt.Printf("🏢 Found %d companies\n", len(links))
	return links, nil
}
