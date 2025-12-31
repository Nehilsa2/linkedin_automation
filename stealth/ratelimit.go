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

// =============================================================================
// SINGLE SOURCE OF TRUTH: All rate limits and throttling configuration
// =============================================================================

// SafetyLevel controls how aggressive the automation is
type SafetyLevel string

const (
	SafetyUltraConservative SafetyLevel = "ultra_conservative" // For accounts at risk
	SafetyConservative      SafetyLevel = "conservative"       // Recommended for main accounts
	SafetyModerate          SafetyLevel = "moderate"           // For established accounts
	SafetyAggressive        SafetyLevel = "aggressive"         // High risk - not recommended
)

// GlobalConfig holds all automation configuration - SINGLE SOURCE OF TRUTH
type GlobalConfig struct {
	// Safety level
	SafetyLevel SafetyLevel `json:"safety_level"`

	// Connection limits
	ConnectionDailyLimit  int `json:"connection_daily_limit"`
	ConnectionHourlyLimit int `json:"connection_hourly_limit"`
	ConnectionDelayMin    int `json:"connection_delay_min_sec"` // seconds
	ConnectionDelayMax    int `json:"connection_delay_max_sec"` // seconds

	// Message limits
	MessageDailyLimit  int `json:"message_daily_limit"`
	MessageHourlyLimit int `json:"message_hourly_limit"`
	MessageDelayMin    int `json:"message_delay_min_sec"` // seconds
	MessageDelayMax    int `json:"message_delay_max_sec"` // seconds

	// Search limits
	SearchDailyLimit  int `json:"search_daily_limit"`
	SearchHourlyLimit int `json:"search_hourly_limit"`
	SearchDelayMin    int `json:"search_delay_min_sec"` // seconds
	SearchDelayMax    int `json:"search_delay_max_sec"` // seconds

	// Burst settings
	BurstLimit    int `json:"burst_limit"`        // Actions before forced cooldown
	BurstCooldown int `json:"burst_cooldown_sec"` // Cooldown after burst (seconds)

	// Session settings
	MaxSessionDuration int `json:"max_session_duration_min"` // Max runtime in minutes
	BreakAfterActions  int `json:"break_after_actions"`      // Take break after N actions
	BreakDurationMin   int `json:"break_duration_min_sec"`   // Min break length (seconds)
	BreakDurationMax   int `json:"break_duration_max_sec"`   // Max break length (seconds)
}

// Pre-defined safety configurations
var safetyConfigs = map[SafetyLevel]*GlobalConfig{
	SafetyUltraConservative: {
		SafetyLevel:           SafetyUltraConservative,
		ConnectionDailyLimit:  5,
		ConnectionHourlyLimit: 2,
		ConnectionDelayMin:    60,  // 1 minute minimum
		ConnectionDelayMax:    180, // 3 minutes maximum
		MessageDailyLimit:     3,
		MessageHourlyLimit:    1,
		MessageDelayMin:       45,
		MessageDelayMax:       120,
		SearchDailyLimit:      10,
		SearchHourlyLimit:     3,
		SearchDelayMin:        10,
		SearchDelayMax:        30,
		BurstLimit:            3,
		BurstCooldown:         600, // 10 min cooldown
		MaxSessionDuration:    60,  // 1 hour max
		BreakAfterActions:     5,
		BreakDurationMin:      120,
		BreakDurationMax:      300,
	},
	SafetyConservative: {
		SafetyLevel:           SafetyConservative,
		ConnectionDailyLimit:  10,
		ConnectionHourlyLimit: 3,
		ConnectionDelayMin:    30, //sec
		ConnectionDelayMax:    90, //sec
		MessageDailyLimit:     3,
		MessageHourlyLimit:    1,
		MessageDelayMin:       30,
		MessageDelayMax:       60,
		SearchDailyLimit:      15,
		SearchHourlyLimit:     5,
		SearchDelayMin:        5,
		SearchDelayMax:        20,
		BurstLimit:            5,
		BurstCooldown:         300, // 5 min cooldown
		MaxSessionDuration:    90,  // 1.5 hours max
		BreakAfterActions:     8,
		BreakDurationMin:      60,
		BreakDurationMax:      180,
	},
	SafetyModerate: {
		SafetyLevel:           SafetyModerate,
		ConnectionDailyLimit:  15,
		ConnectionHourlyLimit: 4,
		ConnectionDelayMin:    20,
		ConnectionDelayMax:    60,
		MessageDailyLimit:     8,
		MessageHourlyLimit:    3,
		MessageDelayMin:       20,
		MessageDelayMax:       45,
		SearchDailyLimit:      20,
		SearchHourlyLimit:     6,
		SearchDelayMin:        3,
		SearchDelayMax:        15,
		BurstLimit:            8,
		BurstCooldown:         180, // 3 min cooldown
		MaxSessionDuration:    120, // 2 hours max
		BreakAfterActions:     12,
		BreakDurationMin:      30,
		BreakDurationMax:      90,
	},
	SafetyAggressive: {
		SafetyLevel:           SafetyAggressive,
		ConnectionDailyLimit:  25,
		ConnectionHourlyLimit: 6,
		ConnectionDelayMin:    10,
		ConnectionDelayMax:    30,
		MessageDailyLimit:     10,
		MessageHourlyLimit:    5,
		MessageDelayMin:       10,
		MessageDelayMax:       30,
		SearchDailyLimit:      30,
		SearchHourlyLimit:     10,
		SearchDelayMin:        2,
		SearchDelayMax:        10,
		BurstLimit:            12,
		BurstCooldown:         120, // 2 min cooldown
		MaxSessionDuration:    180, // 3 hours max
		BreakAfterActions:     20,
		BreakDurationMin:      15,
		BreakDurationMax:      45,
	},
}

// Global configuration instance - SINGLETON
var (
	globalConfig     *GlobalConfig
	globalConfigOnce sync.Once
	globalConfigMu   sync.RWMutex
	configFile       = "rate_config.json"
)

// GetConfig returns the global configuration (singleton)
func GetConfig() *GlobalConfig {
	globalConfigOnce.Do(func() {
		// Try to load from file first
		globalConfig = loadConfigFromFile()
		if globalConfig == nil {
			// Default to conservative
			globalConfig = safetyConfigs[SafetyConservative].clone()
		}
		fmt.Printf("⚙️ Rate limiter initialized: %s mode\n", globalConfig.SafetyLevel)
	})
	return globalConfig
}

// SetSafetyLevel changes the global safety level
func SetSafetyLevel(level SafetyLevel) {
	globalConfigMu.Lock()
	defer globalConfigMu.Unlock()

	if cfg, exists := safetyConfigs[level]; exists {
		globalConfig = cfg.clone()
		saveConfigToFile(globalConfig)
		fmt.Printf("⚙️ Safety level changed to: %s\n", level)
	}
}

// clone creates a copy of the config
func (c *GlobalConfig) clone() *GlobalConfig {
	return &GlobalConfig{
		SafetyLevel:           c.SafetyLevel,
		ConnectionDailyLimit:  c.ConnectionDailyLimit,
		ConnectionHourlyLimit: c.ConnectionHourlyLimit,
		ConnectionDelayMin:    c.ConnectionDelayMin,
		ConnectionDelayMax:    c.ConnectionDelayMax,
		MessageDailyLimit:     c.MessageDailyLimit,
		MessageHourlyLimit:    c.MessageHourlyLimit,
		MessageDelayMin:       c.MessageDelayMin,
		MessageDelayMax:       c.MessageDelayMax,
		SearchDailyLimit:      c.SearchDailyLimit,
		SearchHourlyLimit:     c.SearchHourlyLimit,
		SearchDelayMin:        c.SearchDelayMin,
		SearchDelayMax:        c.SearchDelayMax,
		BurstLimit:            c.BurstLimit,
		BurstCooldown:         c.BurstCooldown,
		MaxSessionDuration:    c.MaxSessionDuration,
		BreakAfterActions:     c.BreakAfterActions,
		BreakDurationMin:      c.BreakDurationMin,
		BreakDurationMax:      c.BreakDurationMax,
	}
}

// =============================================================================
// CONVENIENCE GETTERS - Use these throughout the codebase
// =============================================================================

// Connection getters
func GetConnectionDailyLimit() int  { return GetConfig().ConnectionDailyLimit }
func GetConnectionHourlyLimit() int { return GetConfig().ConnectionHourlyLimit }
func GetConnectionDelayMin() int    { return GetConfig().ConnectionDelayMin }
func GetConnectionDelayMax() int    { return GetConfig().ConnectionDelayMax }

// Message getters
func GetMessageDailyLimit() int  { return GetConfig().MessageDailyLimit }
func GetMessageHourlyLimit() int { return GetConfig().MessageHourlyLimit }
func GetMessageDelayMin() int    { return GetConfig().MessageDelayMin }
func GetMessageDelayMax() int    { return GetConfig().MessageDelayMax }

// Search getters
func GetSearchDailyLimit() int  { return GetConfig().SearchDailyLimit }
func GetSearchHourlyLimit() int { return GetConfig().SearchHourlyLimit }
func GetSearchDelayMin() int    { return GetConfig().SearchDelayMin }
func GetSearchDelayMax() int    { return GetConfig().SearchDelayMax }

// Burst/Break getters
func GetBurstLimit() int        { return GetConfig().BurstLimit }
func GetBurstCooldown() int     { return GetConfig().BurstCooldown }
func GetBreakAfterActions() int { return GetConfig().BreakAfterActions }
func GetBreakDurationMin() int  { return GetConfig().BreakDurationMin }
func GetBreakDurationMax() int  { return GetConfig().BreakDurationMax }

// GetRandomDelay returns a random delay for the given action type
func GetRandomDelay(action ActionType) time.Duration {
	cfg := GetConfig()
	var min, max int

	switch action {
	case ActionConnection:
		min, max = cfg.ConnectionDelayMin, cfg.ConnectionDelayMax
	case ActionMessage:
		min, max = cfg.MessageDelayMin, cfg.MessageDelayMax
	case ActionSearch:
		min, max = cfg.SearchDelayMin, cfg.SearchDelayMax
	default:
		min, max = 5, 15
	}

	if min >= max {
		return time.Duration(min) * time.Second
	}
	delay := min + rand.Intn(max-min+1)
	return time.Duration(delay) * time.Second
}

// GetRandomBreakDuration returns a random break duration
func GetRandomBreakDuration() time.Duration {
	cfg := GetConfig()
	min, max := cfg.BreakDurationMin, cfg.BreakDurationMax
	if min >= max {
		return time.Duration(min) * time.Second
	}
	duration := min + rand.Intn(max-min+1)
	return time.Duration(duration) * time.Second
}

// PrintConfig prints the current configuration
func PrintConfig() {
	cfg := GetConfig()
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("⚙️ RATE LIMIT CONFIGURATION (%s)\n", cfg.SafetyLevel)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Connections: %d/day, %d/hour (delay: %d-%ds)\n",
		cfg.ConnectionDailyLimit, cfg.ConnectionHourlyLimit,
		cfg.ConnectionDelayMin, cfg.ConnectionDelayMax)
	fmt.Printf("Messages:    %d/day, %d/hour (delay: %d-%ds)\n",
		cfg.MessageDailyLimit, cfg.MessageHourlyLimit,
		cfg.MessageDelayMin, cfg.MessageDelayMax)
	fmt.Printf("Searches:    %d/day, %d/hour (delay: %d-%ds)\n",
		cfg.SearchDailyLimit, cfg.SearchHourlyLimit,
		cfg.SearchDelayMin, cfg.SearchDelayMax)
	fmt.Printf("Burst: %d actions then %ds cooldown\n",
		cfg.BurstLimit, cfg.BurstCooldown)
	fmt.Printf("Breaks: every %d actions (%d-%ds)\n",
		cfg.BreakAfterActions, cfg.BreakDurationMin, cfg.BreakDurationMax)
	fmt.Println(strings.Repeat("=", 50))
}

// Config persistence
func loadConfigFromFile() *GlobalConfig {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil
	}
	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

func saveConfigToFile(cfg *GlobalConfig) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(configFile, data, 0644)
}

// =============================================================================
// ACTION TYPES AND RATE LIMITER (uses GlobalConfig)
// =============================================================================

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

// DefaultLimits returns limits based on GlobalConfig
func DefaultLimits() map[ActionType]*RateLimitConfig {
	cfg := GetConfig()
	return map[ActionType]*RateLimitConfig{
		ActionConnection: {
			DailyLimit:         cfg.ConnectionDailyLimit,
			HourlyLimit:        cfg.ConnectionHourlyLimit,
			MinIntervalSeconds: cfg.ConnectionDelayMin,
			MaxIntervalSeconds: cfg.ConnectionDelayMax,
			CooldownThreshold:  cfg.ConnectionDailyLimit,
			CooldownDuration:   cfg.BurstCooldown / 60, // Convert to minutes
			BurstLimit:         cfg.BurstLimit,
			BurstCooldown:      cfg.BurstCooldown,
		},
		ActionMessage: {
			DailyLimit:         cfg.MessageDailyLimit,
			HourlyLimit:        cfg.MessageHourlyLimit,
			MinIntervalSeconds: cfg.MessageDelayMin,
			MaxIntervalSeconds: cfg.MessageDelayMax,
			CooldownThreshold:  cfg.MessageDailyLimit,
			CooldownDuration:   cfg.BurstCooldown / 60,
			BurstLimit:         cfg.BurstLimit,
			BurstCooldown:      cfg.BurstCooldown,
		},
		ActionSearch: {
			DailyLimit:         cfg.SearchDailyLimit,
			HourlyLimit:        cfg.SearchHourlyLimit,
			MinIntervalSeconds: cfg.SearchDelayMin,
			MaxIntervalSeconds: cfg.SearchDelayMax,
			CooldownThreshold:  cfg.SearchDailyLimit,
			CooldownDuration:   cfg.BurstCooldown / 60,
			BurstLimit:         cfg.BurstLimit,
			BurstCooldown:      cfg.BurstCooldown,
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
