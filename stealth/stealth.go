package stealth

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// StealthConfig holds configuration for stealth browser
type StealthConfig struct {
	Headless bool
}

// DefaultConfig returns a minimal stealth configuration
// LESSON LEARNED: Too much modification = more detection!
// LinkedIn can detect fingerprint tampering. Keep it simple.
func DefaultConfig() *StealthConfig {
	return &StealthConfig{
		Headless: false,
	}
}

// CreateStealthLauncher creates a Chrome launcher with minimal anti-detection
//
// KEY INSIGHT: LinkedIn's detection is sophisticated. Trying to fake
// fingerprints (plugins, WebGL, etc.) often makes things WORSE because
// they can detect the faking. The best approach is minimal modification.
func CreateStealthLauncher(config *StealthConfig) *launcher.Launcher {
	l := launcher.New().
		// CRITICAL: This is the ONLY important flag
		// It prevents navigator.webdriver = true
		Set("disable-blink-features", "AutomationControlled").
		// Don't use headless - it's easily detected
		Headless(config.Headless).
		// Leakless can cause issues
		Leakless(false)

	return l
}

// CreateStealthBrowser creates a browser with minimal stealth configuration
func CreateStealthBrowser(config *StealthConfig) (*rod.Browser, error) {
	if config == nil {
		config = DefaultConfig()
	}

	fmt.Println("🥷 Minimal Stealth Mode (webdriver flag disabled)")

	// Create launcher with minimal stealth flags
	l := CreateStealthLauncher(config)
	url := l.MustLaunch()

	// Connect to browser
	browser := rod.New().ControlURL(url).MustConnect()

	return browser, nil
}

// ApplyStealthScripts applies minimal stealth to a page
// IMPORTANT: We only remove the webdriver flag. That's it.
// Faking plugins, WebGL, etc. actually increases detection risk!
func ApplyStealthScripts(page *rod.Page) {
	page.MustEval(`() => {
		// Only remove webdriver flag - nothing else!
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
			configurable: true
		});
	}`)
}
