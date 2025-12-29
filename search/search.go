// opens search page
package search

import (
	"fmt"
	"net/url"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/humanize"
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

	// Apply stealth scripts to mask automation fingerprints
	stealth.ApplyStealthScripts(page)

	page.MustWaitLoad()
	humanize.Sleep(2, 4) // Random page load delay

	return page, nil
}
