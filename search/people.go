package search

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/stealth"
)

func FindPeople(browser *rod.Browser, keyword string, maxPages int) ([]string, error) {

	searchURL := "https://www.linkedin.com/search/results/people/?keywords=" +
		url.QueryEscape(keyword)

	page := browser.MustPage(searchURL)
	stealth.Sleep(3, 5) // Random initial page load

	// Check for LinkedIn errors on initial load
	result := stealth.CheckPage(page)
	if result.HasError {
		stealth.PrintDetectionStatus(result)
		if !result.Error.Recoverable {
			return nil, result.Error
		}
	}

	var allLinks []string
	seen := make(map[string]bool)

	for pageNum := 1; pageNum <= maxPages; pageNum++ {

		// Human-like browsing: scroll through results naturally
		scrollAndBrowse(page)

		// Extract profiles (even if zero)
		pageLinks := 0
		anchors, _ := page.Elements(`a[href^="https://www.linkedin.com/in/"]`)

		for _, a := range anchors {
			href, _ := a.Attribute("href")
			if href == nil {
				continue
			}

			link := strings.Split(*href, "?")[0]
			if !seen[link] {
				seen[link] = true
				allLinks = append(allLinks, link)
				pageLinks++
			}
		}

		fmt.Printf("👤 Page %d → %d profiles\n", pageNum, pageLinks)

		// Use shared pagination function
		hasNext, _ := ClickNextPage(page)
		if !hasNext {
			break
		}
	}

	return allLinks, nil
}
