package search

import (
	"strings"

	"github.com/go-rod/rod"
)

// ExtractPeopleProfiles extracts LinkedIn people profile URLs
func ExtractPeopleProfiles(page *rod.Page) ([]string, error) {
	var results []string
	seen := make(map[string]bool)

	anchors, _ := page.Elements(`a[href^="https://www.linkedin.com/in/"]`)

	for _, a := range anchors {
		href, _ := a.Attribute("href")
		if href == nil {
			continue
		}

		link := strings.Split(*href, "?")[0]
		if !seen[link] {
			seen[link] = true
			results = append(results, link)
		}
	}

	return results, nil
}

func ExtractCompanyProfiles(page *rod.Page) ([]string, error) {

	var results []string
	seen := make(map[string]bool)

	// Each company result card
	cards := page.MustElements(`div[data-view-name="search-entity-result-universal-template"]`)

	for _, card := range cards {

		// Find company link inside the card
		linkEl, err := card.Element(`a[href^="https://www.linkedin.com/company/"]`)
		if err != nil {
			continue
		}

		href, err := linkEl.Attribute("href")
		if err != nil || href == nil {
			continue
		}

		link := strings.Split(*href, "?")[0]

		if !seen[link] {
			seen[link] = true
			results = append(results, link)
		}
	}

	return results, nil
}
