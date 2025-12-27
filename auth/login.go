package auth

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

// Login to linkedin using credentials
func Login(browser *rod.Browser) error {
	//take the email from env
	email := os.Getenv("LINKEDIN_EMAIL")
	password := os.Getenv("LINKEDIN_PASSWORD")

	//check if credentials are missing
	if email == "" || password == "" {
		return fmt.Errorf("Linkedin email or password is missing")
	}

	//login page
	page := browser.MustPage("https://www.linkedin.com/login")
	page.MustWaitLoad()

	//fill email
	page.MustElement(`input#username`).MustInput(email)
	time.Sleep(1 * time.Second)

	page.MustElement(`input#password`).MustInput(password)
	time.Sleep(3 * time.Second)

	// page.MustElement(`input#rememberMeOptIn-checkbox`).MustClick()

	page.MustElement(`button[type="submit"]`).MustClick()

	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	currentURL := page.MustInfo().URL

	//login failure handling

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
