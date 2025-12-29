package search

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/stealth"
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

	// Human-like scroll to bottom to ensure pagination is loaded
	stealth.ScrollDown(page)
	stealth.SleepMillis(500, 1000)
	stealth.ScrollDown(page)
	stealth.SleepMillis(300, 600)

	// Execute JavaScript to find and click the Next button
	result := page.MustEval(`() => {
		// Check for LinkedIn search limit message first
		const pageText = document.body.innerText || '';
		const limitPhrases = [
			"reached the monthly limit",
			"reached your monthly limit",
			"Upgrade to Premium",
			"unlimited search",
			"You've reached the",
			"search limit"
		];
		
		for (const phrase of limitPhrases) {
			if (pageText.toLowerCase().includes(phrase.toLowerCase())) {
				return { found: false, disabled: false, clicked: false, limitReached: true };
			}
		}
		
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

		return { found: false, disabled: false, clicked: false, limitReached: false };
	}`)

	// Parse result using Get method for gson.JSON
	found := result.Get("found").Bool()
	disabled := result.Get("disabled").Bool()
	clicked := result.Get("clicked").Bool()
	limitReached := result.Get("limitReached").Bool()

	// Check if LinkedIn search limit was reached
	if limitReached {
		fmt.Println("⚠️ LinkedIn monthly search limit reached - cannot continue pagination")
		return false, nil
	}

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
	stealth.Sleep(2, 4) // Random wait for page to load
	page.MustWaitStable()

	return true, nil
}
