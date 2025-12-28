// opens search page
package search

import (
	"fmt"
	"net/url"
	"time"

	"github.com/go-rod/rod"
)

func OpenSearchPage(browser *rod.Browser, searchType, keyword string, pageNum int) (*rod.Page, error) {
	encoded := url.QueryEscape(keyword)

	searchURL := fmt.Sprintf(
		"https://www.linkedin.com/search/results/%s/?keywords=%s",
		searchType,
		encoded,
	)

	if pageNum > 1 {
		searchURL += fmt.Sprintf("&page=%d", pageNum)
	}

	page := browser.MustPage(searchURL)
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	return page, nil
}
