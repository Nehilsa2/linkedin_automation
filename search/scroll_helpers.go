package search

import (
	"math/rand"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/humanize"
)

// scrollAndBrowse simulates natural human browsing behavior on search results
// It scrolls through the page with variable speeds, pauses, and occasional scroll-backs
func scrollAndBrowse(page *rod.Page) {
	// Random number of scroll actions (3-6 times)
	scrollActions := 3 + rand.Intn(4)

	for i := 0; i < scrollActions; i++ {
		// Random action type
		action := rand.Float64()

		switch {
		case action < 0.6:
			// 60% - Normal scroll down
			humanize.ScrollDown(page)

		case action < 0.75:
			// 15% - Quick scroll (impatient behavior)
			humanize.ScrollDown(page)
			humanize.SleepMillis(100, 300)
			humanize.ScrollDown(page)

		case action < 0.85:
			// 10% - Scroll up (re-reading something)
			humanize.ScrollUp(page)

		default:
			// 15% - Pause and "read" (longer delay)
			humanize.Sleep(1, 3)
		}

		// Variable delay between scroll actions
		humanize.SleepMillis(300, 800)
	}

	// Final scroll to ensure we've seen most results
	humanize.ScrollDown(page)
	humanize.SleepMillis(500, 1000)
}

// scrollToElement scrolls an element into view with human-like behavior
func scrollToElement(page *rod.Page, selector string) error {
	return humanize.ScrollIntoView(page, selector)
}

// browseResults simulates a user casually browsing search results
// Good for pages where you want to appear like you're actually reading
func browseResults(page *rod.Page) {
	// Simulate natural reading pattern
	humanize.BrowseScroll(page, 4+rand.Intn(3)) // 4-6 browse actions
}
