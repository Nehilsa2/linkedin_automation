package humanize

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
)

// TypingConfig holds configuration for human-like typing
type TypingConfig struct {
	// Base delay between keystrokes in milliseconds
	BaseDelayMs int
	// Random variation added to base delay (±)
	VariationMs int
	// Probability of a longer "thinking" pause (0-100)
	ThinkPauseProbability int
	// Duration of thinking pause in milliseconds
	ThinkPauseMinMs int
	ThinkPauseMaxMs int
	// Probability of making a typo and correcting it (0-100)
	// Set to 0 to disable typos
	TypoProbability int
}

// DefaultTypingConfig returns sensible defaults for human-like typing
// Average human types 40-60 WPM = 200-300ms per character
func DefaultTypingConfig() *TypingConfig {
	return &TypingConfig{
		BaseDelayMs:           80,  // ~75 WPM base speed
		VariationMs:           40,  // ±40ms variation
		ThinkPauseProbability: 5,   // 5% chance of pause
		ThinkPauseMinMs:       300, // 300-800ms thinking pause
		ThinkPauseMaxMs:       800,
		TypoProbability:       0, // Disabled by default (risky)
	}
}

// FastTypingConfig returns config for faster typing (experienced user)
func FastTypingConfig() *TypingConfig {
	return &TypingConfig{
		BaseDelayMs:           50, // ~100 WPM
		VariationMs:           25,
		ThinkPauseProbability: 3,
		ThinkPauseMinMs:       200,
		ThinkPauseMaxMs:       500,
		TypoProbability:       0,
	}
}

// SlowTypingConfig returns config for slower typing (casual user)
func SlowTypingConfig() *TypingConfig {
	return &TypingConfig{
		BaseDelayMs:           120, // ~50 WPM
		VariationMs:           60,
		ThinkPauseProbability: 8,
		ThinkPauseMinMs:       500,
		ThinkPauseMaxMs:       1200,
		TypoProbability:       0,
	}
}

// TypeText types text character by character with human-like timing
// This simulates real keyboard input with natural delays
//
// WHY THIS IS IMPORTANT:
// - LinkedIn monitors keystroke timing patterns
// - Instant text (innerHTML = "...") has 0 keystroke events = obvious bot
// - Human typing has variable speed: faster for common letters, slower for symbols
// - Natural pauses occur between words and sentences
func TypeText(page *rod.Page, selector string, text string, config *TypingConfig) error {
	if config == nil {
		config = DefaultTypingConfig()
	}

	// Find and focus the element
	element, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Clear existing content first
	element.MustSelectAllText().MustInput("")
	SleepMillis(100, 200)

	// Focus the element
	element.MustFocus()
	SleepMillis(50, 100)

	// Type each character with human-like delays
	for i, char := range text {
		// Calculate delay for this character
		delay := calculateKeystrokeDelay(char, config, i, len(text))

		// Type the character
		element.MustInput(string(char))

		// Wait before next character
		time.Sleep(delay)
	}

	return nil
}

// TypeTextIntoActiveElement types into the currently focused element
// Use this when the element is already focused
func TypeTextIntoActiveElement(page *rod.Page, text string, config *TypingConfig) error {
	if config == nil {
		config = DefaultTypingConfig()
	}

	for i, char := range text {
		// Calculate delay for this character
		delay := calculateKeystrokeDelay(char, config, i, len(text))

		// Type using page.InsertText which simulates typing
		page.InsertText(string(char))

		// Wait before next character
		time.Sleep(delay)
	}

	return nil
}

// TypeTextWithElement types into a rod.Element with human-like timing
func TypeTextWithElement(element *rod.Element, text string, config *TypingConfig) error {
	if config == nil {
		config = DefaultTypingConfig()
	}

	// Clear existing content
	element.MustSelectAllText()
	SleepMillis(50, 100)

	// Focus the element
	element.MustFocus()
	SleepMillis(50, 100)

	// Type each character
	for i, char := range text {
		delay := calculateKeystrokeDelay(char, config, i, len(text))

		// Input single character
		element.MustInput(string(char))

		time.Sleep(delay)
	}

	return nil
}

// calculateKeystrokeDelay returns a human-like delay for a keystroke
// Takes into account:
// - Character type (letters faster, symbols slower)
// - Position in word (first char slower, mid-word faster)
// - Random variation
// - Occasional thinking pauses
func calculateKeystrokeDelay(char rune, config *TypingConfig, position, totalLength int) time.Duration {
	baseDelay := config.BaseDelayMs

	// Adjust based on character type
	switch {
	case char == ' ':
		// Space after word - slightly longer (word boundary)
		baseDelay = int(float64(baseDelay) * 1.3)
	case char == '.' || char == '!' || char == '?':
		// End of sentence - longer pause
		baseDelay = int(float64(baseDelay) * 1.8)
	case char == ',' || char == ';' || char == ':':
		// Punctuation - moderate pause
		baseDelay = int(float64(baseDelay) * 1.4)
	case char >= 'A' && char <= 'Z':
		// Capital letters - slightly slower (shift key)
		baseDelay = int(float64(baseDelay) * 1.2)
	case char >= '0' && char <= '9':
		// Numbers - slightly slower
		baseDelay = int(float64(baseDelay) * 1.15)
	case char == '@' || char == '#' || char == '$' || char == '%':
		// Special characters - slower (shift + key)
		baseDelay = int(float64(baseDelay) * 1.4)
	}

	// First character is often slower (finding the key)
	if position == 0 {
		baseDelay = int(float64(baseDelay) * 1.5)
	}

	// Add random variation
	variation := rand.Intn(config.VariationMs*2) - config.VariationMs
	delay := baseDelay + variation

	// Ensure minimum delay
	if delay < 30 {
		delay = 30
	}

	// Random thinking pause
	if rand.Intn(100) < config.ThinkPauseProbability {
		thinkPause := rand.Intn(config.ThinkPauseMaxMs-config.ThinkPauseMinMs) + config.ThinkPauseMinMs
		delay += thinkPause
	}

	return time.Duration(delay) * time.Millisecond
}

// TypeCredential types a credential (email/password) with human-like timing
// Uses slightly faster config since users often type familiar credentials quickly
func TypeCredential(element *rod.Element, credential string) error {
	config := &TypingConfig{
		BaseDelayMs:           60, // Faster for familiar text
		VariationMs:           30,
		ThinkPauseProbability: 2, // Less pausing
		ThinkPauseMinMs:       150,
		ThinkPauseMaxMs:       400,
		TypoProbability:       0,
	}

	return TypeTextWithElement(element, credential, config)
}

// TypeMessage types a message with natural human timing
// Uses default config with occasional pauses for "thinking"
func TypeMessage(element *rod.Element, message string) error {
	config := DefaultTypingConfig()
	return TypeTextWithElement(element, message, config)
}

// SimulateTypingDelay just adds a delay as if typing occurred
// Use when you need the timing effect without actual typing
func SimulateTypingDelay(textLength int, config *TypingConfig) {
	if config == nil {
		config = DefaultTypingConfig()
	}

	// Estimate total typing time
	avgDelay := config.BaseDelayMs + config.VariationMs/2
	totalDelay := textLength * avgDelay

	// Add some variance
	variance := totalDelay / 5 // ±20%
	totalDelay += rand.Intn(variance*2) - variance

	time.Sleep(time.Duration(totalDelay) * time.Millisecond)
}

// TypeTextJS types text using JavaScript-simulated keyboard events
// This is an alternative approach that may work better with some input fields
func TypeTextJS(page *rod.Page, text string, config *TypingConfig) error {
	if config == nil {
		config = DefaultTypingConfig()
	}

	for i, char := range text {
		delay := calculateKeystrokeDelay(char, config, i, len(text))

		// Simulate keydown, keypress, input, keyup events
		page.MustEval(`(char) => {
			const activeElement = document.activeElement;
			if (!activeElement) return;

			const keyCode = char.charCodeAt(0);
			
			// Create and dispatch keyboard events
			const keydownEvent = new KeyboardEvent('keydown', {
				key: char,
				code: 'Key' + char.toUpperCase(),
				keyCode: keyCode,
				which: keyCode,
				bubbles: true
			});
			
			const keypressEvent = new KeyboardEvent('keypress', {
				key: char,
				code: 'Key' + char.toUpperCase(),
				keyCode: keyCode,
				which: keyCode,
				bubbles: true
			});

			const inputEvent = new InputEvent('input', {
				data: char,
				inputType: 'insertText',
				bubbles: true
			});

			const keyupEvent = new KeyboardEvent('keyup', {
				key: char,
				code: 'Key' + char.toUpperCase(),
				keyCode: keyCode,
				which: keyCode,
				bubbles: true
			});

			activeElement.dispatchEvent(keydownEvent);
			activeElement.dispatchEvent(keypressEvent);
			
			// Insert the character
			if (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA') {
				activeElement.value += char;
			} else if (activeElement.isContentEditable) {
				document.execCommand('insertText', false, char);
			}
			
			activeElement.dispatchEvent(inputEvent);
			activeElement.dispatchEvent(keyupEvent);
		}`, string(char))

		time.Sleep(delay)
	}

	return nil
}
