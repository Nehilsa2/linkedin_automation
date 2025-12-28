// package search

// import (
// 	"fmt"
// 	"time"

// 	"github.com/go-rod/rod"
// )

// // ClickNextPage clicks LinkedIn pagination "Next" button

// func ClickNextPage(page *rod.Page) (bool, error) {
// 	fmt.Println("DEBUG: Looking for Next button via JS")

// 	ok := page.MustEval(`() => {
// 	const selectors = [
// 		'button[data-testid="pagination-controls-next-button"]',
// 		'button[aria-label="Next"]',
// 	];

// 	for (const selector of selectors) {
// 		const btn = document.querySelector(selector);
// 		if (!btn) continue;
// 		if (btn.disabled) return false;
// 		btn.scrollIntoView({ block: "center" });
// 		btn.click();
// 		return true;
// 	}

// 	return false;
// }`).Bool()

// 	if !ok {
// 		fmt.Println("ℹ️ No Next button found or disabled, stopping")
// 		return false, nil
// 	}

// 	fmt.Println("➡️ Clicking Next page")
// 	time.Sleep(5 * time.Second) // React re-render
// 	return true, nil
// }

package search

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

// ClickNextPage clicks LinkedIn pagination "Next" button
// Returns (hasMorePages bool, error)
// - hasMorePages: true if successfully clicked and more pages exist
// - error: any error that occurred during the operation
func ClickNextPage(page *rod.Page) (bool, error) {
	fmt.Println("🔍 Looking for Next button...")

	// Set timeout to prevent hanging
	page = page.Timeout(10 * time.Second)
	defer page.CancelTimeout()

	// Execute JavaScript to find and click the Next button
	result := page.MustEval(`() => {
		const selectors = [
			'button[data-testid="pagination-controls-next-button"]',
			'button[aria-label="Next"]',
			'button.artdeco-pagination__button--next',
			'button[aria-label="View next page"]',
		];

		for (const selector of selectors) {
			const btn = document.querySelector(selector);
			if (!btn) continue;

			// Check if button is disabled
			const isDisabled = btn.disabled || 
			                  btn.getAttribute('aria-disabled') === 'true' ||
			                  btn.classList.contains('artdeco-button--disabled');

			if (isDisabled) {
				return { found: true, disabled: true, clicked: false };
			}

			// Click the button
			btn.scrollIntoView({ block: "center", behavior: "smooth" });
			btn.click();
			return { found: true, disabled: false, clicked: true };
		}

		return { found: false, disabled: false, clicked: false };
	}`)

	// Parse result using Get method for gson.JSON
	found := result.Get("found").Bool()
	disabled := result.Get("disabled").Bool()
	clicked := result.Get("clicked").Bool()

	// No button found - end of results
	if !found {
		fmt.Println("ℹ️ No Next button found - reached end")
		return false, nil
	}

	// Button disabled - last page
	if disabled {
		fmt.Println("ℹ️ Next button disabled - last page")
		return false, nil
	}

	// Button found but click failed
	if !clicked {
		return false, fmt.Errorf("next button found but click failed")
	}

	// Success
	fmt.Println("✅ Clicked Next button")
	time.Sleep(3 * time.Second) // Wait for page to load
	page.MustWaitStable()

	return true, nil
}
