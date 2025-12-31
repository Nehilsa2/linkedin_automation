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

	var allLinks []string // <-- moved here

	// Check for LinkedIn errors on initial load
	result := stealth.CheckPage(page)
	if result.HasError {
		stealth.PrintDetectionStatus(result)
		if result.Error.Type == stealth.ErrorMonthlySearchLimit {
			fmt.Println("⚠️ Monthly search limit detected on initial load. Attempting to extract any visible profiles...")
			anchors, _ := page.Elements(`a[href^="https://www.linkedin.com/in/"]`)
			for _, a := range anchors {
				href, _ := a.Attribute("href")
				if href == nil {
					continue
				}
				link := strings.Split(*href, "?")[0]
				allLinks = append(allLinks, link)
			}
			fmt.Printf("🔎 Extracted %d profiles despite limit banner.\n", len(allLinks))
			// Do not save to DB here; let caller handle it
			return allLinks, result.Error
		}
		if !result.Error.Recoverable {
			return nil, result.Error
		}
	}

	var seen = make(map[string]bool)

	for pageNum := 1; pageNum <= maxPages; pageNum++ {

		// Human-like browsing: scroll through results naturally
		scrollAndBrowse(page)

		// ALWAYS extract profiles FIRST (even if limit reached, we want current page)
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

		fmt.Printf("👤 Page %d → %d profiles (total: %d)\n", pageNum, pageLinks, len(allLinks))

		// Check if LinkedIn monthly search limit reached AFTER extracting current page
		limitReached := checkSearchLimitReached(page)
		if limitReached {
			fmt.Println("⚠️ LinkedIn monthly search limit reached - extracted current page profiles before stopping")
			fmt.Println("🔎 Extracted profiles on last page:")
			for _, link := range allLinks {
				fmt.Println("   ", link)
			}
			break
		}

		// Try to go to next page (only if not last page requested)
		if pageNum < maxPages {
			hasNext, _ := ClickNextPage(page)
			if !hasNext {
				fmt.Println("ℹ️ No more pages available")
				break
			}
		}
	}

	fmt.Printf("✅ Search complete: found %d total profiles\n", len(allLinks))
	return allLinks, nil
}

// checkSearchLimitReached checks if LinkedIn's monthly search limit message is shown
func checkSearchLimitReached(page *rod.Page) bool {
	result := page.MustEval(`() => {
		const pageText = document.body.innerText || '';
		const limitPhrases = [
			"reached the monthly limit",
			"reached your monthly limit", 
			"you've reached the commercial use limit",
			"commercial use limit",
			"Upgrade to Premium",
			"Get unlimited searches",
			"unlimited search"
		];
		
		for (const phrase of limitPhrases) {
			if (pageText.toLowerCase().includes(phrase.toLowerCase())) {
				return true;
			}
		}
		
		// Also check if pagination is disabled/hidden
		const nextBtn = document.querySelector('button[aria-label="Next"]');
		const paginationDisabled = document.querySelector('.artdeco-pagination--disabled');
		
		if (paginationDisabled || (nextBtn && nextBtn.disabled)) {
			// Check if there's a premium upsell visible
			const premiumUpsell = document.querySelector('[class*="premium"], [class*="upsell"]');
			if (premiumUpsell) {
				return true;
			}
		}
		
		return false;
	}`)

	return result.Bool()
}
