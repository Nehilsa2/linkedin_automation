// opens search page
package search

import (
	"fmt"
	"net/url"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/stealth"
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
	stealth.Sleep(2, 4) // Random page load delay

	// Check for LinkedIn errors after loading search page
	result := stealth.CheckPage(page)
	if result.HasError {
		stealth.PrintDetectionStatus(result)
		return page, result.Error
	}

	return page, nil
}
