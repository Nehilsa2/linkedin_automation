package stealth

import (
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
)

// ScrollConfig holds configuration for human-like scrolling
type ScrollConfig struct {
	// Base scroll amount in pixels
	BaseScrollMin int
	BaseScrollMax int

	// Speed variation (delay between scroll steps in ms)
	ScrollSpeedMin int
	ScrollSpeedMax int

	// Probability of scroll-back (0.0 to 1.0)
	ScrollBackChance float64

	// Scroll-back amount (percentage of last scroll)
	ScrollBackMin float64
	ScrollBackMax float64

	// Pause probability after scrolling
	PauseChance float64
	PauseMin    int // milliseconds
	PauseMax    int // milliseconds

	// Acceleration settings
	UseAcceleration bool
	AccelSteps      int // number of steps for acceleration/deceleration
}

// DefaultScrollConfig returns sensible defaults for human-like scrolling
func DefaultScrollConfig() *ScrollConfig {
	return &ScrollConfig{
		BaseScrollMin:    100,
		BaseScrollMax:    400,
		ScrollSpeedMin:   15,
		ScrollSpeedMax:   50,
		ScrollBackChance: 0.15, // 15% chance to scroll back slightly
		ScrollBackMin:    0.1,
		ScrollBackMax:    0.3,
		PauseChance:      0.25, // 25% chance to pause after scroll
		PauseMin:         200,
		PauseMax:         800,
		UseAcceleration:  true,
		AccelSteps:       5,
	}
}

// Global scroll config
var ScrollCfg = DefaultScrollConfig()

// ScrollDown performs a human-like scroll down on the page
func ScrollDown(page *rod.Page) error {
	return ScrollDownWithConfig(page, ScrollCfg)
}

// ScrollDownWithConfig performs scroll with custom configuration
func ScrollDownWithConfig(page *rod.Page, cfg *ScrollConfig) error {
	// Random scroll distance
	distance := rand.Intn(cfg.BaseScrollMax-cfg.BaseScrollMin+1) + cfg.BaseScrollMin

	// Perform the scroll with acceleration
	if cfg.UseAcceleration {
		smoothScroll(page, distance, cfg)
	} else {
		simpleScroll(page, distance, cfg)
	}

	// Occasional scroll-back
	if rand.Float64() < cfg.ScrollBackChance {
		scrollBack(page, distance, cfg)
	}

	// Occasional pause (simulating reading)
	if rand.Float64() < cfg.PauseChance {
		pauseDelay := rand.Intn(cfg.PauseMax-cfg.PauseMin+1) + cfg.PauseMin
		time.Sleep(time.Duration(pauseDelay) * time.Millisecond)
	}

	return nil
}

// smoothScroll implements acceleration and deceleration
func smoothScroll(page *rod.Page, totalDistance int, cfg *ScrollConfig) {
	steps := cfg.AccelSteps * 2 // acceleration + deceleration phases
	if steps < 4 {
		steps = 4
	}

	// Calculate distance per step with easing
	for i := 0; i < steps; i++ {
		// Ease-in-out function (sine-based)
		progress := float64(i) / float64(steps-1)
		easeValue := easeInOutSine(progress)

		// Calculate this step's scroll amount
		stepDistance := int(float64(totalDistance) / float64(steps))

		// Vary the step size based on easing (faster in middle, slower at ends)
		if i < steps/2 {
			// Acceleration phase - gradually increase
			stepDistance = int(float64(stepDistance) * (0.5 + easeValue))
		} else {
			// Deceleration phase - gradually decrease
			stepDistance = int(float64(stepDistance) * (1.5 - easeValue))
		}

		if stepDistance < 10 {
			stepDistance = 10
		}

		// Execute scroll step
		page.Mouse.MustScroll(0, float64(stepDistance))

		// Variable delay between steps (faster in middle)
		delay := cfg.ScrollSpeedMin + rand.Intn(cfg.ScrollSpeedMax-cfg.ScrollSpeedMin+1)
		if i > cfg.AccelSteps/2 && i < steps-cfg.AccelSteps/2 {
			delay = delay / 2 // Faster in the middle
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// simpleScroll performs basic scrolling without acceleration
func simpleScroll(page *rod.Page, distance int, cfg *ScrollConfig) {
	// Break into small steps
	steps := 3 + rand.Intn(4) // 3-6 steps
	stepSize := distance / steps

	for i := 0; i < steps; i++ {
		// Add slight variation to each step
		variation := rand.Intn(21) - 10 // -10 to +10
		actualStep := stepSize + variation
		if actualStep < 10 {
			actualStep = 10
		}

		page.Mouse.MustScroll(0, float64(actualStep))

		delay := cfg.ScrollSpeedMin + rand.Intn(cfg.ScrollSpeedMax-cfg.ScrollSpeedMin+1)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// scrollBack performs a slight scroll back (natural human behavior)
func scrollBack(page *rod.Page, lastDistance int, cfg *ScrollConfig) {
	// Calculate scroll-back amount
	backPercent := cfg.ScrollBackMin + rand.Float64()*(cfg.ScrollBackMax-cfg.ScrollBackMin)
	backDistance := int(float64(lastDistance) * backPercent)

	// Small delay before scrolling back
	time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

	// Scroll up (negative Y)
	page.Mouse.MustScroll(0, float64(-backDistance))

	// Brief pause after scroll-back
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
}

// easeInOutSine provides smooth acceleration/deceleration curve
func easeInOutSine(x float64) float64 {
	return -(math.Cos(math.Pi*x) - 1) / 2
}

// ScrollToBottom scrolls to the bottom of the page with human-like behavior
func ScrollToBottom(page *rod.Page, maxScrolls int) error {
	return ScrollToBottomWithConfig(page, maxScrolls, ScrollCfg)
}

// ScrollToBottomWithConfig scrolls to bottom with custom config
func ScrollToBottomWithConfig(page *rod.Page, maxScrolls int, cfg *ScrollConfig) error {
	lastHeight := 0

	for i := 0; i < maxScrolls; i++ {
		// Get current scroll position
		currentHeight := page.MustEval(`() => document.documentElement.scrollTop`).Int()

		// Check if we've reached the bottom
		if currentHeight == lastHeight && i > 0 {
			// Might be at bottom, try one more scroll to be sure
			ScrollDownWithConfig(page, cfg)
			time.Sleep(500 * time.Millisecond)

			newHeight := page.MustEval(`() => document.documentElement.scrollTop`).Int()
			if newHeight == currentHeight {
				break // Definitely at bottom
			}
		}

		lastHeight = currentHeight

		// Perform human-like scroll
		ScrollDownWithConfig(page, cfg)

		// Random delay between scrolls (reading time)
		readTime := 500 + rand.Intn(1500) // 0.5 to 2 seconds
		time.Sleep(time.Duration(readTime) * time.Millisecond)
	}

	return nil
}

// ScrollIntoView scrolls an element into view with human-like behavior
func ScrollIntoView(page *rod.Page, selector string) error {
	el, err := page.Element(selector)
	if err != nil {
		return err
	}
	return ScrollElementIntoView(page, el)
}

// ScrollElementIntoView scrolls a specific element into view naturally
func ScrollElementIntoView(page *rod.Page, el *rod.Element) error {
	// Get element position
	box, err := el.Shape()
	if err != nil {
		return err
	}

	if box == nil || len(box.Quads) == 0 {
		// Element not visible, use default scroll
		el.MustScrollIntoView()
		return nil
	}

	// Get viewport info
	viewportHeight := page.MustEval(`() => window.innerHeight`).Int()
	currentScroll := page.MustEval(`() => window.scrollY`).Int()

	// Calculate element's position relative to viewport
	elementTop := int(box.Quads[0][1])
	elementInViewport := elementTop - currentScroll

	// Determine if we need to scroll
	if elementInViewport < 0 || elementInViewport > viewportHeight-100 {
		// Calculate target scroll position (element at ~1/3 of viewport)
		targetScroll := elementTop - (viewportHeight / 3)
		scrollNeeded := targetScroll - currentScroll

		if scrollNeeded != 0 {
			// Scroll with human-like behavior
			scrollToPosition(page, scrollNeeded)
		}
	}

	return nil
}

// scrollToPosition scrolls by a specific amount with human-like motion
func scrollToPosition(page *rod.Page, distance int) {
	direction := 1
	if distance < 0 {
		direction = -1
		distance = -distance
	}

	// Break into chunks with acceleration
	chunks := 4 + rand.Intn(4) // 4-7 chunks
	baseChunk := distance / chunks

	for i := 0; i < chunks; i++ {
		// Vary chunk size (larger in middle for natural feel)
		multiplier := 1.0
		if i == 0 || i == chunks-1 {
			multiplier = 0.5 // Slower at start/end
		} else if i == chunks/2 {
			multiplier = 1.5 // Faster in middle
		}

		chunkSize := int(float64(baseChunk) * multiplier)
		if chunkSize < 20 {
			chunkSize = 20
		}

		page.Mouse.MustScroll(0, float64(chunkSize*direction))

		// Variable delay
		delay := 20 + rand.Intn(40)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// RandomScroll performs a random scroll (up or down) to simulate browsing
func RandomScroll(page *rod.Page) error {
	// 70% chance to scroll down, 30% up
	if rand.Float64() < 0.7 {
		return ScrollDown(page)
	}
	return ScrollUp(page)
}

// ScrollUp performs a human-like scroll up
func ScrollUp(page *rod.Page) error {
	distance := rand.Intn(ScrollCfg.BaseScrollMax-ScrollCfg.BaseScrollMin+1) + ScrollCfg.BaseScrollMin

	// Smaller scroll up (feels more natural)
	distance = int(float64(distance) * 0.6)

	steps := 2 + rand.Intn(3)
	stepSize := distance / steps

	for i := 0; i < steps; i++ {
		page.Mouse.MustScroll(0, float64(-stepSize))
		delay := ScrollCfg.ScrollSpeedMin + rand.Intn(ScrollCfg.ScrollSpeedMax-ScrollCfg.ScrollSpeedMin+1)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	return nil
}

// BrowseScroll simulates natural page browsing (mix of scrolls and pauses)
func BrowseScroll(page *rod.Page, iterations int) error {
	for i := 0; i < iterations; i++ {
		// Random action
		action := rand.Float64()

		switch {
		case action < 0.5:
			// 50% - Scroll down
			ScrollDown(page)
		case action < 0.65:
			// 15% - Scroll up
			ScrollUp(page)
		case action < 0.85:
			// 20% - Pause and "read"
			readTime := 1000 + rand.Intn(3000)
			time.Sleep(time.Duration(readTime) * time.Millisecond)
		default:
			// 15% - Quick scroll (impatient user)
			quickScroll(page)
		}

		// Small delay between actions
		time.Sleep(time.Duration(200+rand.Intn(400)) * time.Millisecond)
	}

	return nil
}

// quickScroll simulates an impatient fast scroll
func quickScroll(page *rod.Page) {
	distance := 500 + rand.Intn(300) // Larger, faster scroll

	steps := 2
	stepSize := distance / steps

	for i := 0; i < steps; i++ {
		page.Mouse.MustScroll(0, float64(stepSize))
		time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)
	}
}
