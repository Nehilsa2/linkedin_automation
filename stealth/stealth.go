package stealth

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// StealthConfig holds configuration for stealth browser
type StealthConfig struct {
	Headless  bool
	UserAgent string
	Viewport  *Viewport
}

// Viewport represents browser window dimensions
type Viewport struct {
	Width  int
	Height int
}

// Common realistic viewport sizes (desktop)
var commonViewports = []Viewport{
	{1920, 1080}, // Full HD (most common)
	{1366, 768},  // HD (laptops)
	{1536, 864},  // Common laptop
	{1440, 900},  // MacBook
	{1280, 720},  // HD
	{1600, 900},  // HD+
	{1680, 1050}, // WSXGA+
	{1920, 1200}, // WUXGA
}

// Common realistic user agents (updated Chrome versions)
var commonUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
}

// DefaultConfig returns a randomized stealth configuration
func DefaultConfig() *StealthConfig {
	return &StealthConfig{
		Headless:  false,
		UserAgent: randomUserAgent(),
		Viewport:  randomViewport(),
	}
}

// randomUserAgent returns a random realistic user agent
func randomUserAgent() string {
	return commonUserAgents[rand.Intn(len(commonUserAgents))]
}

// randomViewport returns a random realistic viewport
func randomViewport() *Viewport {
	vp := commonViewports[rand.Intn(len(commonViewports))]
	// Add slight randomness to avoid exact matches
	vp.Width += rand.Intn(20) - 10  // ±10 pixels
	vp.Height += rand.Intn(20) - 10 // ±10 pixels
	return &vp
}

// CreateStealthLauncher creates a Chrome launcher with anti-detection flags
//
// WHY THESE FLAGS MATTER:
//   - "disable-blink-features=AutomationControlled": Removes the
//     "Chrome is being controlled by automated software" banner and
//     prevents navigator.webdriver from being set to true
//   - "disable-dev-shm-usage": Prevents crashes in Docker/limited memory
//   - "no-first-run": Skips first-run dialogs that could interfere
//   - "disable-infobars": Removes automation info bars
//   - "excludeSwitches=enable-automation": Removes automation flags from Chrome
func CreateStealthLauncher(config *StealthConfig) *launcher.Launcher {
	l := launcher.New().
		// CRITICAL: Disable automation-controlled feature
		// This is the main flag that prevents navigator.webdriver = true
		Set("disable-blink-features", "AutomationControlled").

		// Remove the "Chrome is being controlled" infobar
		Set("disable-infobars").

		// Prevent first-run dialogs
		Set("no-first-run").
		Set("no-default-browser-check").

		// Memory optimization (also makes fingerprint more realistic)
		Set("disable-dev-shm-usage").

		// Set window size (will be overridden per-page, but good default)
		Set("window-size", fmt.Sprintf("%d,%d", config.Viewport.Width, config.Viewport.Height)).

		// Disable automation extension
		Set("disable-extensions").

		// Use a realistic user agent
		Set("user-agent", config.UserAgent).

		// Control headless mode
		Headless(config.Headless).

		// Don't use leakless (can cause issues)
		Leakless(false)

	return l
}

// CreateStealthBrowser creates a browser with stealth configuration
func CreateStealthBrowser(config *StealthConfig) (*rod.Browser, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Print stealth config for debugging
	fmt.Println("🥷 Stealth Configuration:")
	fmt.Printf("   User-Agent: %s\n", truncate(config.UserAgent, 60))
	fmt.Printf("   Viewport: %dx%d\n", config.Viewport.Width, config.Viewport.Height)

	// Create launcher with stealth flags
	l := CreateStealthLauncher(config)
	url := l.MustLaunch()

	// Connect to browser
	browser := rod.New().ControlURL(url).MustConnect()

	return browser, nil
}

// ApplyStealthToPage applies stealth settings to a specific page
// This injects JavaScript BEFORE any page scripts run
func ApplyStealthToPage(page *rod.Page, config *StealthConfig) error {
	if config == nil {
		config = DefaultConfig()
	}

	// Set viewport with slight randomness
	page.MustSetViewport(config.Viewport.Width, config.Viewport.Height, 1.0, false)

	// Inject stealth JavaScript that runs before page loads
	// This is the MOST IMPORTANT part - it modifies navigator properties
	// before LinkedIn's detection scripts can read them
	_, err := page.EvalOnNewDocument(getStealthScript())
	if err != nil {
		return fmt.Errorf("failed to inject stealth script: %w", err)
	}

	return nil
}

// ApplyStealthScripts injects anti-detection JavaScript into a page
// Call this AFTER navigation to reinforce stealth
func ApplyStealthScripts(page *rod.Page) {
	// Run stealth scripts
	page.MustEval(getStealthScript())
}

// getStealthScript returns JavaScript code that masks automation fingerprints
//
// WHY THIS WORKS:
// LinkedIn's detection script runs something like:
//
//	if (navigator.webdriver) { flagAsBot(); }
//	if (navigator.plugins.length === 0) { flagAsBot(); }
//
// By overriding these properties BEFORE their script runs,
// we make the browser look like a normal user's browser
func getStealthScript() string {
	return `
	// ============================================================
	// STEALTH SCRIPT - Masks automation fingerprints
	// ============================================================

	// 1. Remove webdriver flag (MOST IMPORTANT)
	// LinkedIn checks: if (navigator.webdriver) flagAsBot()
	Object.defineProperty(navigator, 'webdriver', {
		get: () => undefined,
		configurable: true
	});

	// 2. Fix plugins array (real Chrome has plugins)
	// An empty plugins array is suspicious
	Object.defineProperty(navigator, 'plugins', {
		get: () => {
			const plugins = [
				{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
				{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
				{ name: 'Native Client', filename: 'internal-nacl-plugin', description: '' }
			];
			plugins.item = (i) => plugins[i] || null;
			plugins.namedItem = (name) => plugins.find(p => p.name === name) || null;
			plugins.refresh = () => {};
			return plugins;
		},
		configurable: true
	});

	// 3. Fix languages (should match Accept-Language header)
	Object.defineProperty(navigator, 'languages', {
		get: () => ['en-US', 'en'],
		configurable: true
	});

	// 4. Fix hardware concurrency (CPU cores)
	// Undefined or 0 is suspicious - real devices have 2-16 cores
	Object.defineProperty(navigator, 'hardwareConcurrency', {
		get: () => 8,
		configurable: true
	});

	// 5. Fix device memory
	// Undefined is suspicious - real devices report 2-8 GB
	Object.defineProperty(navigator, 'deviceMemory', {
		get: () => 8,
		configurable: true
	});

	// 6. Fix maxTouchPoints (desktop = 0, mobile = 5+)
	Object.defineProperty(navigator, 'maxTouchPoints', {
		get: () => 0,
		configurable: true
	});

	// 7. Fix Chrome object (headless Chrome is missing properties)
	if (!window.chrome) {
		window.chrome = {};
	}
	if (!window.chrome.runtime) {
		window.chrome.runtime = {};
	}

	// 8. Fix permissions query (automation often fails this)
	const originalQuery = window.navigator.permissions?.query;
	if (originalQuery) {
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);
	}

	// 9. Fix WebGL vendor/renderer (for canvas fingerprinting)
	const getParameterProxyHandler = {
		apply: function(target, ctx, args) {
			const param = args[0];
			// UNMASKED_VENDOR_WEBGL
			if (param === 37445) {
				return 'Google Inc. (NVIDIA)';
			}
			// UNMASKED_RENDERER_WEBGL  
			if (param === 37446) {
				return 'ANGLE (NVIDIA, NVIDIA GeForce GTX 1080 Direct3D11 vs_5_0 ps_5_0, D3D11)';
			}
			return Reflect.apply(target, ctx, args);
		}
	};

	// Apply to both WebGL contexts
	try {
		const canvas = document.createElement('canvas');
		const gl = canvas.getContext('webgl');
		if (gl) {
			const originalGetParameter = gl.getParameter.bind(gl);
			gl.__proto__.getParameter = new Proxy(originalGetParameter, getParameterProxyHandler);
		}
		const gl2 = canvas.getContext('webgl2');
		if (gl2) {
			const originalGetParameter2 = gl2.getParameter.bind(gl2);
			gl2.__proto__.getParameter = new Proxy(originalGetParameter2, getParameterProxyHandler);
		}
	} catch (e) {}

	// 10. Fix toString() checks (advanced detection)
	// Some sites check if functions have been tampered with
	const nativeToString = Function.prototype.toString;
	Function.prototype.toString = function() {
		if (this === window.navigator.permissions?.query) {
			return 'function query() { [native code] }';
		}
		return nativeToString.call(this);
	};

	// Mark as stealthed (for debugging)
	window.__stealthed__ = true;
	`
}

// EmulateHumanViewport sets a realistic viewport with slight randomness
func EmulateHumanViewport(page *rod.Page) {
	vp := randomViewport()
	page.MustSetViewport(vp.Width, vp.Height, 1.0, false)
}

// GetRandomUserAgent returns a random realistic user agent
func GetRandomUserAgent() string {
	return randomUserAgent()
}

// Helper to truncate strings for display
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// SetupPageStealth applies all stealth measures to a page
// Call this after creating a new page but BEFORE navigation
func SetupPageStealth(page *rod.Page, config *StealthConfig) error {
	if config == nil {
		config = DefaultConfig()
	}

	// Set viewport
	page.MustSetViewport(config.Viewport.Width, config.Viewport.Height, 1.0, false)

	// Set user agent via CDP
	page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: config.UserAgent,
	})

	// Inject stealth script to run on every new document
	_, err := page.EvalOnNewDocument(getStealthScript())
	if err != nil {
		return fmt.Errorf("failed to setup page stealth: %w", err)
	}

	return nil
}
