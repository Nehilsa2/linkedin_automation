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

	// Scroll to bottom to ensure pagination is loaded
	page.MustEval(`() => window.scrollTo(0, document.body.scrollHeight)`)
	time.Sleep(1 * time.Second)

	// Execute JavaScript to find and click the Next button
	result := page.MustEval(`() => {
		// Debug: log all pagination-related elements
		const debugInfo = [];
		
		// Extended selectors for both people and company search pages
		const selectors = [
			'button[aria-label="Next"]',
			'button[aria-label="View next page"]',
			'button[data-testid="pagination-controls-next-button"]',
			'button.artdeco-pagination__button--next',
			'.artdeco-pagination__button--next',
			'li.artdeco-pagination__indicator--number + li button',
		];

		// Also try to find by button text content
		const allButtons = document.querySelectorAll('button');
		for (const btn of allButtons) {
			const text = btn.textContent?.trim().toLowerCase() || '';
			const ariaLabel = btn.getAttribute('aria-label')?.toLowerCase() || '';
			
			if (text === 'next' || ariaLabel.includes('next')) {
				const isDisabled = btn.disabled || 
				                  btn.getAttribute('aria-disabled') === 'true' ||
				                  btn.classList.contains('artdeco-button--disabled') ||
				                  btn.classList.contains('disabled');
				
				if (isDisabled) {
					return { found: true, disabled: true, clicked: false, method: 'text-search' };
				}
				
				btn.scrollIntoView({ block: "center", behavior: "smooth" });
				btn.click();
				return { found: true, disabled: false, clicked: true, method: 'text-search' };
			}
		}

		// Try standard selectors
		for (const selector of selectors) {
			const btn = document.querySelector(selector);
			if (!btn) continue;

			// Check if button is disabled
			const isDisabled = btn.disabled || 
			                  btn.getAttribute('aria-disabled') === 'true' ||
			                  btn.classList.contains('artdeco-button--disabled') ||
			                  btn.classList.contains('disabled');

			if (isDisabled) {
				return { found: true, disabled: true, clicked: false, selector: selector };
			}

			// Click the button
			btn.scrollIntoView({ block: "center", behavior: "smooth" });
			btn.click();
			return { found: true, disabled: false, clicked: true, selector: selector };
		}

		// Last resort: find pagination container and look for right arrow / next icon
		const paginationContainers = document.querySelectorAll('.artdeco-pagination, [class*="pagination"]');
		for (const container of paginationContainers) {
			const buttons = container.querySelectorAll('button');
			// Usually the last button in pagination is "Next"
			if (buttons.length >= 2) {
				const lastBtn = buttons[buttons.length - 1];
				const isDisabled = lastBtn.disabled || 
				                  lastBtn.getAttribute('aria-disabled') === 'true';
				
				if (!isDisabled) {
					lastBtn.scrollIntoView({ block: "center", behavior: "smooth" });
					lastBtn.click();
					return { found: true, disabled: false, clicked: true, method: 'pagination-container-last' };
				} else {
					return { found: true, disabled: true, clicked: false, method: 'pagination-container-last' };
				}
			}
		}

		// Try finding by pagination list items (LinkedIn's numbered pagination)
		const paginationList = document.querySelector('ul.artdeco-pagination__pages, .artdeco-pagination__pages');
		if (paginationList) {
			const items = paginationList.querySelectorAll('li');
			let currentIdx = -1;
			
			for (let i = 0; i < items.length; i++) {
				const item = items[i];
				if (item.classList.contains('active') || item.classList.contains('selected') ||
				    item.querySelector('button[aria-current="true"]') ||
				    item.querySelector('.active')) {
					currentIdx = i;
					break;
				}
			}
			
			// Click the next page number
			if (currentIdx >= 0 && currentIdx < items.length - 1) {
				const nextItem = items[currentIdx + 1];
				const nextBtn = nextItem.querySelector('button');
				if (nextBtn && !nextBtn.disabled) {
					nextBtn.scrollIntoView({ block: "center", behavior: "smooth" });
					nextBtn.click();
					return { found: true, disabled: false, clicked: true, method: 'pagination-number-next' };
				}
			}
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
