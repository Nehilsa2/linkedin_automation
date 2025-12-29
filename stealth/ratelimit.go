package stealth

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

// ActionType represents different rate-limited actions
type ActionType string

const (
	ActionConnection ActionType = "connection"
	ActionMessage    ActionType = "message"
	ActionSearch     ActionType = "search"
)

// RateLimitConfig defines limits for a specific action type
type RateLimitConfig struct {
	// Hard limits
	DailyLimit  int `json:"daily_limit"`
	HourlyLimit int `json:"hourly_limit"`

	// Spacing requirements
	MinIntervalSeconds int `json:"min_interval_seconds"` // Minimum time between actions
	MaxIntervalSeconds int `json:"max_interval_seconds"` // Maximum time between actions

	// Cooldown settings
	CooldownThreshold int `json:"cooldown_threshold"` // Actions before cooldown
	CooldownDuration  int `json:"cooldown_minutes"`   // Cooldown length in minutes

	// Burst settings (allow short bursts, then enforce gaps)
	BurstLimit    int `json:"burst_limit"`    // Max actions in a burst
	BurstCooldown int `json:"burst_cooldown"` // Seconds to wait after burst
}

// DefaultLimits returns conservative limits for each action type
func DefaultLimits() map[ActionType]*RateLimitConfig {
	return map[ActionType]*RateLimitConfig{
		ActionConnection: {
			DailyLimit:         14, // LinkedIn's unofficial limit is ~100/week
			HourlyLimit:        3,
			MinIntervalSeconds: 30,
			MaxIntervalSeconds: 120,
			CooldownThreshold:  15,
			CooldownDuration:   30,
			BurstLimit:         5,
			BurstCooldown:      300, // 5 min after 5 connections
		},
		ActionMessage: {
			DailyLimit:         20,
			HourlyLimit:        4,
			MinIntervalSeconds: 20,
			MaxIntervalSeconds: 90,
			CooldownThreshold:  20,
			CooldownDuration:   20,
			BurstLimit:         8,
			BurstCooldown:      180, // 3 min after 8 messages
		},
		ActionSearch: {
			DailyLimit:         15, // LinkedIn limits searches
			HourlyLimit:        5,
			MinIntervalSeconds: 5,
			MaxIntervalSeconds: 30,
			CooldownThreshold:  10,
			CooldownDuration:   10,
			BurstLimit:         5,
			BurstCooldown:      60,
		},
	}
}

// ActionRecord tracks when an action occurred
type ActionRecord struct {
	Timestamp time.Time  `json:"timestamp"`
	Type      ActionType `json:"type"`
}

// RateLimiter manages rate limiting across all action types
type RateLimiter struct {
	mu sync.RWMutex

	// Configuration
	limits map[ActionType]*RateLimitConfig

	// State tracking
	actions     []ActionRecord           // All actions (pruned to 24h)
	lastAction  map[ActionType]time.Time // Last action per type
	burstCount  map[ActionType]int       // Current burst count
	burstStart  map[ActionType]time.Time // When burst started
	inCooldown  map[ActionType]bool      // Currently in cooldown
	cooldownEnd map[ActionType]time.Time // When cooldown ends

	// Persistence
	stateFile string
}

// RateLimiterState for JSON persistence
type RateLimiterState struct {
	Actions     []ActionRecord       `json:"actions"`
	LastAction  map[string]time.Time `json:"last_action"`
	BurstCount  map[string]int       `json:"burst_count"`
	BurstStart  map[string]time.Time `json:"burst_start"`
	CooldownEnd map[string]time.Time `json:"cooldown_end"`
	SavedAt     time.Time            `json:"saved_at"`
}

// NewRateLimiter creates a new rate limiter with default limits
func NewRateLimiter() *RateLimiter {
	return NewRateLimiterWithConfig(DefaultLimits(), "rate_limiter_state.json")
}

// NewRateLimiterWithConfig creates a rate limiter with custom configuration
func NewRateLimiterWithConfig(limits map[ActionType]*RateLimitConfig, stateFile string) *RateLimiter {
	rl := &RateLimiter{
		limits:      limits,
		actions:     make([]ActionRecord, 0),
		lastAction:  make(map[ActionType]time.Time),
		burstCount:  make(map[ActionType]int),
		burstStart:  make(map[ActionType]time.Time),
		inCooldown:  make(map[ActionType]bool),
		cooldownEnd: make(map[ActionType]time.Time),
		stateFile:   stateFile,
	}

	// Load persisted state
	rl.loadState()

	return rl
}

// CanPerform checks if an action can be performed now
func (rl *RateLimiter) CanPerform(action ActionType) (bool, string) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	cfg, exists := rl.limits[action]
	if !exists {
		return true, "" // No limits configured
	}

	now := time.Now()

	// Check cooldown
	if rl.inCooldown[action] && now.Before(rl.cooldownEnd[action]) {
		remaining := rl.cooldownEnd[action].Sub(now)
		return false, fmt.Sprintf("in cooldown (%v remaining)", remaining.Round(time.Second))
	}

	// Clear expired cooldown
	if rl.inCooldown[action] && now.After(rl.cooldownEnd[action]) {
		rl.mu.RUnlock()
		rl.mu.Lock()
		rl.inCooldown[action] = false
		rl.burstCount[action] = 0
		rl.mu.Unlock()
		rl.mu.RLock()
	}

	// Check daily limit
	dailyCount := rl.countActionsSince(action, now.Add(-24*time.Hour))
	if dailyCount >= cfg.DailyLimit {
		return false, fmt.Sprintf("daily limit reached (%d/%d)", dailyCount, cfg.DailyLimit)
	}

	// Check hourly limit
	hourlyCount := rl.countActionsSince(action, now.Add(-1*time.Hour))
	if hourlyCount >= cfg.HourlyLimit {
		return false, fmt.Sprintf("hourly limit reached (%d/%d)", hourlyCount, cfg.HourlyLimit)
	}

	// Check minimum interval
	if lastTime, exists := rl.lastAction[action]; exists {
		elapsed := now.Sub(lastTime)
		minInterval := time.Duration(cfg.MinIntervalSeconds) * time.Second
		if elapsed < minInterval {
			wait := minInterval - elapsed
			return false, fmt.Sprintf("too soon (wait %v)", wait.Round(time.Second))
		}
	}

	return true, ""
}

// RecordAction records that an action was performed
func (rl *RateLimiter) RecordAction(action ActionType) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Record the action
	rl.actions = append(rl.actions, ActionRecord{
		Timestamp: now,
		Type:      action,
	})
	rl.lastAction[action] = now

	// Update burst tracking
	cfg := rl.limits[action]
	if cfg == nil {
		return
	}

	// Check if burst window expired (reset burst count)
	if burstStart, exists := rl.burstStart[action]; exists {
		burstWindow := time.Duration(cfg.BurstCooldown) * time.Second
		if now.Sub(burstStart) > burstWindow {
			rl.burstCount[action] = 0
		}
	}

	// Increment burst count
	if rl.burstCount[action] == 0 {
		rl.burstStart[action] = now
	}
	rl.burstCount[action]++

	// Check if burst limit reached -> trigger cooldown
	if rl.burstCount[action] >= cfg.BurstLimit {
		rl.inCooldown[action] = true
		rl.cooldownEnd[action] = now.Add(time.Duration(cfg.BurstCooldown) * time.Second)
		fmt.Printf("⏸️ Burst limit reached for %s - cooldown until %s\n",
			action, rl.cooldownEnd[action].Format("15:04:05"))
	}

	// Check if cooldown threshold reached
	recentCount := rl.countActionsSinceUnlocked(action, now.Add(-1*time.Hour))
	if recentCount >= cfg.CooldownThreshold && !rl.inCooldown[action] {
		rl.inCooldown[action] = true
		rl.cooldownEnd[action] = now.Add(time.Duration(cfg.CooldownDuration) * time.Minute)
		fmt.Printf("⏸️ Cooldown threshold reached for %s - resting until %s\n",
			action, rl.cooldownEnd[action].Format("15:04:05"))
	}

	// Prune old actions (keep 24h)
	rl.pruneOldActions()

	// Save state
	rl.saveStateUnlocked()
}

// WaitForAction waits until action can be performed, returns false if should stop
func (rl *RateLimiter) WaitForAction(action ActionType) bool {
	for {
		can, reason := rl.CanPerform(action)
		if can {
			return true
		}

		// Calculate wait time
		waitTime := rl.getWaitTime(action)
		if waitTime > 30*time.Minute {
			fmt.Printf("⏰ Long wait required for %s (%s): %v\n", action, reason, waitTime.Round(time.Minute))
			return false // Too long, let caller decide
		}

		fmt.Printf("⏳ Waiting for %s (%s): %v\n", action, reason, waitTime.Round(time.Second))
		time.Sleep(waitTime)
	}
}

// GetRecommendedDelay returns the recommended delay before next action
func (rl *RateLimiter) GetRecommendedDelay(action ActionType) time.Duration {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	cfg, exists := rl.limits[action]
	if !exists {
		return time.Duration(5+rand.Intn(10)) * time.Second
	}

	// Random delay between min and max interval
	minSec := cfg.MinIntervalSeconds
	maxSec := cfg.MaxIntervalSeconds
	delaySec := minSec + rand.Intn(maxSec-minSec+1)

	return time.Duration(delaySec) * time.Second
}

// GetStats returns current statistics for an action type
func (rl *RateLimiter) GetStats(action ActionType) ActionStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	cfg := rl.limits[action]

	stats := ActionStats{
		Action:      action,
		DailyCount:  rl.countActionsSince(action, now.Add(-24*time.Hour)),
		HourlyCount: rl.countActionsSince(action, now.Add(-1*time.Hour)),
		InCooldown:  rl.inCooldown[action],
		BurstCount:  rl.burstCount[action],
	}

	if cfg != nil {
		stats.DailyLimit = cfg.DailyLimit
		stats.HourlyLimit = cfg.HourlyLimit
		stats.DailyRemaining = cfg.DailyLimit - stats.DailyCount
		stats.HourlyRemaining = cfg.HourlyLimit - stats.HourlyCount
		stats.BurstLimit = cfg.BurstLimit
	}

	if rl.inCooldown[action] {
		stats.CooldownRemaining = rl.cooldownEnd[action].Sub(now)
	}

	if lastTime, exists := rl.lastAction[action]; exists {
		stats.LastAction = lastTime
		stats.TimeSinceLastAction = now.Sub(lastTime)
	}

	return stats
}

// ActionStats holds statistics for an action type
type ActionStats struct {
	Action              ActionType
	DailyCount          int
	DailyLimit          int
	DailyRemaining      int
	HourlyCount         int
	HourlyLimit         int
	HourlyRemaining     int
	InCooldown          bool
	CooldownRemaining   time.Duration
	BurstCount          int
	BurstLimit          int
	LastAction          time.Time
	TimeSinceLastAction time.Duration
}

// PrintStats prints formatted statistics
func (rl *RateLimiter) PrintStats(action ActionType) {
	stats := rl.GetStats(action)

	fmt.Printf("\n📊 Rate Limit Stats for %s:\n", action)
	fmt.Printf("   Daily:  %d/%d (remaining: %d)\n", stats.DailyCount, stats.DailyLimit, stats.DailyRemaining)
	fmt.Printf("   Hourly: %d/%d (remaining: %d)\n", stats.HourlyCount, stats.HourlyLimit, stats.HourlyRemaining)
	fmt.Printf("   Burst:  %d/%d\n", stats.BurstCount, stats.BurstLimit)

	if stats.InCooldown {
		fmt.Printf("   ⏸️ IN COOLDOWN: %v remaining\n", stats.CooldownRemaining.Round(time.Second))
	}

	if !stats.LastAction.IsZero() {
		fmt.Printf("   Last action: %v ago\n", stats.TimeSinceLastAction.Round(time.Second))
	}
}

// PrintAllStats prints statistics for all action types
func (rl *RateLimiter) PrintAllStats() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 RATE LIMITER STATUS")
	fmt.Println(strings.Repeat("=", 50))

	for action := range rl.limits {
		rl.PrintStats(action)
	}
}

// === Internal helpers ===

func (rl *RateLimiter) countActionsSince(action ActionType, since time.Time) int {
	count := 0
	for _, record := range rl.actions {
		if record.Type == action && record.Timestamp.After(since) {
			count++
		}
	}
	return count
}

func (rl *RateLimiter) countActionsSinceUnlocked(action ActionType, since time.Time) int {
	return rl.countActionsSince(action, since)
}

func (rl *RateLimiter) getWaitTime(action ActionType) time.Duration {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	cfg := rl.limits[action]
	if cfg == nil {
		return 5 * time.Second
	}

	now := time.Now()

	// If in cooldown, wait for cooldown to end
	if rl.inCooldown[action] && now.Before(rl.cooldownEnd[action]) {
		return rl.cooldownEnd[action].Sub(now) + time.Second
	}

	// If minimum interval not met, wait for it
	if lastTime, exists := rl.lastAction[action]; exists {
		minInterval := time.Duration(cfg.MinIntervalSeconds) * time.Second
		elapsed := now.Sub(lastTime)
		if elapsed < minInterval {
			return minInterval - elapsed + time.Second
		}
	}

	// Default wait
	return time.Duration(cfg.MinIntervalSeconds) * time.Second
}

func (rl *RateLimiter) pruneOldActions() {
	cutoff := time.Now().Add(-24 * time.Hour)
	filtered := make([]ActionRecord, 0, len(rl.actions))

	for _, record := range rl.actions {
		if record.Timestamp.After(cutoff) {
			filtered = append(filtered, record)
		}
	}

	rl.actions = filtered
}

// === Persistence ===

func (rl *RateLimiter) loadState() {
	data, err := os.ReadFile(rl.stateFile)
	if err != nil {
		return // No state file yet
	}

	var state RateLimiterState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	// Only load if saved recently (within 24h)
	if time.Since(state.SavedAt) > 24*time.Hour {
		return
	}

	rl.actions = state.Actions

	// Convert string keys back to ActionType
	for k, v := range state.LastAction {
		rl.lastAction[ActionType(k)] = v
	}
	for k, v := range state.BurstCount {
		rl.burstCount[ActionType(k)] = v
	}
	for k, v := range state.BurstStart {
		rl.burstStart[ActionType(k)] = v
	}
	for k, v := range state.CooldownEnd {
		if time.Now().Before(v) {
			rl.inCooldown[ActionType(k)] = true
			rl.cooldownEnd[ActionType(k)] = v
		}
	}

	// Prune old actions
	rl.pruneOldActions()

	fmt.Println("📂 Loaded rate limiter state from", rl.stateFile)
}

func (rl *RateLimiter) saveStateUnlocked() {
	state := RateLimiterState{
		Actions:     rl.actions,
		LastAction:  make(map[string]time.Time),
		BurstCount:  make(map[string]int),
		BurstStart:  make(map[string]time.Time),
		CooldownEnd: make(map[string]time.Time),
		SavedAt:     time.Now(),
	}

	for k, v := range rl.lastAction {
		state.LastAction[string(k)] = v
	}
	for k, v := range rl.burstCount {
		state.BurstCount[string(k)] = v
	}
	for k, v := range rl.burstStart {
		state.BurstStart[string(k)] = v
	}
	for k, v := range rl.cooldownEnd {
		state.CooldownEnd[string(k)] = v
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(rl.stateFile, data, 0644)
}

// SetLimit allows adjusting limits at runtime
func (rl *RateLimiter) SetLimit(action ActionType, cfg *RateLimitConfig) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limits[action] = cfg
}

// Reset clears all tracking for an action type
func (rl *RateLimiter) Reset(action ActionType) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.lastAction, action)
	delete(rl.burstCount, action)
	delete(rl.burstStart, action)
	delete(rl.inCooldown, action)
	delete(rl.cooldownEnd, action)

	// Remove actions of this type
	filtered := make([]ActionRecord, 0)
	for _, record := range rl.actions {
		if record.Type != action {
			filtered = append(filtered, record)
		}
	}
	rl.actions = filtered

	rl.saveStateUnlocked()
}

// === Global rate limiter instance ===

var globalRateLimiter *RateLimiter
var rateLimiterOnce sync.Once

// GetRateLimiter returns the global rate limiter instance
func GetRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		globalRateLimiter = NewRateLimiter()
	})
	return globalRateLimiter
}
