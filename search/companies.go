package search

import (
	"fmt"

	"github.com/go-rod/rod"
)

func FindCompanies(browser *rod.Browser, keyword string, maxPages int) ([]string, error) {

	page, err := OpenSearchPage(browser, "companies", keyword, 1)
	if err != nil {
		return nil, err
	}

	var allLinks []string
	seen := make(map[string]bool)

	for pageNum := 1; pageNum <= maxPages; pageNum++ {
		fmt.Println("DEBUG: waiting for company results")
		page.MustWaitElementsMoreThan(
			`div[data-view-name="search-entity-result-universal-template"]`,
			0,
		)
		fmt.Println("DEBUG: company results appeared")
		links, _ := ExtractCompanyProfiles(page)

		for _, l := range links {
			if !seen[l] {
				seen[l] = true
				allLinks = append(allLinks, l)
			}
		}

		fmt.Printf("🏢 Page %d → %d companies\n", pageNum, len(links))

		hasNext, _ := ClickNextPage(page)
		if !hasNext {
			break
		}
	}

	return allLinks, nil
}
