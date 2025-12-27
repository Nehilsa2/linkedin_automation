package auth

import (
	"fmt"
	"strings"

	"github.com/go-rod/rod"
)

// EnsureAuthenticated guarantees a logged-in LinkedIn session
func EnsureAuthenticated(browser *rod.Browser) error {

	// Try loading cookies
	if err := LoadCookies(browser); err == nil {
		fmt.Println("🍪 Cookies loaded")

		// Verify if cookies are still valid
		page := browser.MustPage("https://www.linkedin.com/feed/")
		page.MustWaitLoad()

		if !strings.Contains(page.MustInfo().URL, "/login") {
			fmt.Println("✅ Authenticated using existing cookies")
			return nil
		}

		fmt.Println("⚠️ Cookies expired or invalid")
	}

	// Perform fresh login
	fmt.Println("🔐 Performing fresh login...")
	if err := Login(browser); err != nil {
		return err
	}

	// Save cookies after successful login
	if err := SaveCookies(browser); err != nil {
		return fmt.Errorf("failed to save cookies: %v", err)
	}

	fmt.Println("🍪 Cookies saved successfully")
	return nil
}
