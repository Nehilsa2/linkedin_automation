// opens search page
package search

import (
	"fmt"
	"net/url"
	"time"

	"github.com/go-rod/rod"
)

func OpenSearchPage(browser *rod.Browser, searchType, keyword string) (*rod.Page, error) {
	encodedKeyword := url.QueryEscape(keyword)

	//build linkedin search url
	searchURL := fmt.Sprintf(
		"https://www.linkedin.com/search/results/%s/?keywords=%s",
		searchType,
		encodedKeyword,
	)

	//open search page
	page := browser.MustPage(searchURL)
	page.MustWaitLoad()

	time.Sleep(3 * time.Second)

	return page, nil
}
