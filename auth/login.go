package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/stealth"
)

// Login to linkedin using credentials
//
// WHY HUMAN-LIKE TYPING FOR LOGIN:
// - MustInput() types at ~1000 chars/second (superhuman speed)
// - LinkedIn monitors: "Email typed in 10ms" = bot flag
// - Human typing: 40-60 WPM with natural variation
// - Credentials are typed slightly faster (familiar text)
func Login(browser *rod.Browser) error {
	// Take the email from env
	email := os.Getenv("LINKEDIN_EMAIL")
	password := os.Getenv("LINKEDIN_PASSWORD")

	// Check if credentials are missing
	if email == "" || password == "" {
		return fmt.Errorf("Linkedin email or password is missing")
	}

	// Login page
	page := browser.MustPage("https://www.linkedin.com/login")
	page.MustWaitLoad()
	stealth.Sleep(2, 3) // Wait for page to fully render

	// Wait for email input to be ready
	fmt.Println("⌨️ Typing email...")
	emailInput := page.MustElement(`input#username`)
	emailInput.MustWaitVisible()
	emailInput.MustFocus()
	stealth.SleepMillis(300, 500) // Pause before typing
	err := stealth.TypeCredential(emailInput, email)
	if err != nil {
		return fmt.Errorf("failed to type email: %w", err)
	}
	stealth.Sleep(1, 2) // Pause between fields (like a human tabbing)

	// Fill password with human-like typing
	fmt.Println("⌨️ Typing password...")
	passwordInput := page.MustElement(`input#password`)
	passwordInput.MustWaitVisible()
	passwordInput.MustFocus()
	stealth.SleepMillis(300, 500)
	err = stealth.TypeCredential(passwordInput, password)
	if err != nil {
		return fmt.Errorf("failed to type password: %w", err)
	}
	stealth.Sleep(1, 2) // Pause before clicking submit

	// Click submit
	page.MustElement(`button[type="submit"]`).MustClick()

	page.MustWaitLoad()
	stealth.Sleep(2, 4) // Wait for login to process

	currentURL := page.MustInfo().URL

	// Login failure handling
	if strings.Contains(currentURL, "/checkpoint") {
		return fmt.Errorf("checkpoint detected (captcha or 2FA required)")
	}

	if strings.Contains(currentURL, "/login") {
		return fmt.Errorf("login failed: invalid credentials")
	}

	// ---- VERIFY AUTHENTICATION ----
	page.MustNavigate("https://www.linkedin.com/feed/")
	page.MustWaitLoad()

	if strings.Contains(page.MustInfo().URL, "/login") {
		return fmt.Errorf("authentication failed: redirected back to login")
	}

	fmt.Println("✅ Authenticated Successfully")
	return nil
}
