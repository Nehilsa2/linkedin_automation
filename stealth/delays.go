package stealth

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// DelayConfig holds configuration for different delay types
type DelayConfig struct {
	// Action delays (between major actions like sending connection requests)
	ActionDelayMin int // seconds
	ActionDelayMax int // seconds

	// Page load delays
	PageLoadMin int // seconds
	PageLoadMax int // seconds

	// Short delays (between clicks, scrolls)
	ShortDelayMin int // milliseconds
	ShortDelayMax int // milliseconds

	// Think time (simulating reading content)
	ThinkTimeMin int // seconds
	ThinkTimeMax int // seconds
}

// DefaultConfig returns sensible default delay configuration
func DefaultConfig() *DelayConfig {
	return &DelayConfig{
		ActionDelayMin: 8,
		ActionDelayMax: 15,
		PageLoadMin:    2,
		PageLoadMax:    5,
		ShortDelayMin:  300,
		ShortDelayMax:  800,
		ThinkTimeMin:   1,
		ThinkTimeMax:   3,
	}
}

// Global config - can be modified at runtime
var Config = DefaultConfig()

// RandomSeconds returns a random duration between min and max seconds
func RandomSeconds(min, max int) time.Duration {
	if min >= max {
		return time.Duration(min) * time.Second
	}
	n := rand.Intn(max-min+1) + min
	return time.Duration(n) * time.Second
}

// RandomMillis returns a random duration between min and max milliseconds
func RandomMillis(min, max int) time.Duration {
	if min >= max {
		return time.Duration(min) * time.Millisecond
	}
	n := rand.Intn(max-min+1) + min
	return time.Duration(n) * time.Millisecond
}

// RandomSecondsFloat returns a random duration with sub-second precision
func RandomSecondsFloat(min, max float64) time.Duration {
	if min >= max {
		return time.Duration(min * float64(time.Second))
	}
	n := min + rand.Float64()*(max-min)
	return time.Duration(n * float64(time.Second))
}

// GaussianSeconds returns a normally distributed random duration
// centered around mean with given standard deviation
func GaussianSeconds(mean, stdDev float64) time.Duration {
	n := rand.NormFloat64()*stdDev + mean
	// Clamp to reasonable bounds (mean ± 3*stdDev)
	minVal := math.Max(0.5, mean-3*stdDev)
	maxVal := mean + 3*stdDev
	n = math.Max(minVal, math.Min(maxVal, n))
	return time.Duration(n * float64(time.Second))
}

// Sleep pauses for a random duration between min and max seconds
func Sleep(min, max int) {
	d := RandomSeconds(min, max)
	fmt.Printf("⏳ Waiting %.1f seconds...\n", d.Seconds())
	time.Sleep(d)
}

// SleepMillis pauses for a random duration between min and max milliseconds
func SleepMillis(min, max int) {
	time.Sleep(RandomMillis(min, max))
}

// SleepQuiet pauses without printing (for micro-delays)
func SleepQuiet(min, max int) {
	time.Sleep(RandomSeconds(min, max))
}

// ActionDelay waits between major actions (connection requests, messages)
func ActionDelay() {
	Sleep(Config.ActionDelayMin, Config.ActionDelayMax)
}

// PageLoadDelay waits for page to load
func PageLoadDelay() {
	d := RandomSeconds(Config.PageLoadMin, Config.PageLoadMax)
	time.Sleep(d)
}

// ShortDelay waits briefly between UI interactions
func ShortDelay() {
	SleepMillis(Config.ShortDelayMin, Config.ShortDelayMax)
}

// ThinkTime simulates user reading/thinking
func ThinkTime() {
	d := RandomSeconds(Config.ThinkTimeMin, Config.ThinkTimeMax)
	time.Sleep(d)
}

// ThinkTimeForContent returns a delay based on content length
// Simulates reading time (~200-250 words per minute)
func ThinkTimeForContent(content string) time.Duration {
	words := len(content) / 5                  // Average word length
	readingTimeSeconds := float64(words) / 4.0 // ~240 words per minute = 4 per second

	// Add some variance (±30%)
	variance := readingTimeSeconds * 0.3
	minTime := math.Max(0.5, readingTimeSeconds-variance)
	maxTime := readingTimeSeconds + variance

	return RandomSecondsFloat(minTime, maxTime)
}

// SleepForContent waits based on content length (simulates reading)
func SleepForContent(content string) {
	d := ThinkTimeForContent(content)
	if d.Seconds() > 1 {
		fmt.Printf("👀 Reading... (%.1fs)\n", d.Seconds())
	}
	time.Sleep(d)
}

// JitterMillis adds small random jitter (for more natural timing)
func JitterMillis(baseMs int, jitterPercent int) time.Duration {
	jitter := baseMs * jitterPercent / 100
	if jitter == 0 {
		return time.Duration(baseMs) * time.Millisecond
	}
	actual := baseMs + rand.Intn(2*jitter) - jitter
	if actual < 50 {
		actual = 50
	}
	return time.Duration(actual) * time.Millisecond
}

// MaybeExtraDelay randomly adds an extra delay (simulates distraction)
// probability is 0-100 (percentage chance of extra delay)
func MaybeExtraDelay(probability int, minSec, maxSec int) {
	if rand.Intn(100) < probability {
		fmt.Println("☕ Taking a short break...")
		Sleep(minSec, maxSec)
	}
}

// ActionBurst tracks actions and adds periodic longer breaks
type ActionBurst struct {
	actionCount int
	burstSize   int // Actions before taking a break
	breakMinSec int
	breakMaxSec int
}

// NewActionBurst creates a new burst tracker
func NewActionBurst(burstSize, breakMin, breakMax int) *ActionBurst {
	return &ActionBurst{
		burstSize:   burstSize,
		breakMinSec: breakMin,
		breakMaxSec: breakMax,
	}
}

// DefaultActionBurst returns a burst tracker with default settings
func DefaultActionBurst() *ActionBurst {
	return NewActionBurst(
		3+rand.Intn(3), // 3-5 actions per burst
		5,              // 5-15 second breaks
		15,
	)
}

// Track records an action and may trigger a break
func (ab *ActionBurst) Track() {
	ab.actionCount++
	if ab.actionCount >= ab.burstSize {
		fmt.Println("🧠 Taking a moment to think...")
		Sleep(ab.breakMinSec, ab.breakMaxSec)
		ab.actionCount = 0
		ab.burstSize = 3 + rand.Intn(3) // Randomize next burst size
	}
}

// Reset resets the action counter
func (ab *ActionBurst) Reset() {
	ab.actionCount = 0
}
