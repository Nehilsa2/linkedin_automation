package humanize

import (
	"fmt"
	"math/rand"
	"time"
)

// ScheduleConfig defines work schedule parameters
type ScheduleConfig struct {
	// Work hours (24-hour format)
	WorkStartHour int // e.g., 9 for 9 AM
	WorkEndHour   int // e.g., 17 for 5 PM

	// Daily variation (minutes) - shifts start/end randomly
	StartVariation int // e.g., 30 means 8:30-9:30 start
	EndVariation   int // e.g., 30 means 4:30-5:30 end

	// Work days (0=Sunday, 1=Monday, ..., 6=Saturday)
	WorkDays []time.Weekday

	// Break settings
	LunchStartHour   int // e.g., 12
	LunchDurationMin int // e.g., 45-60 minutes
	LunchDurationMax int

	// Short breaks
	ShortBreakChance      float64 // Probability per activity cycle
	ShortBreakDurationMin int     // minutes
	ShortBreakDurationMax int

	// Activity bursts (work in focused periods, not constant)
	BurstDurationMin int // minutes of activity
	BurstDurationMax int
	BurstGapMin      int // minutes between bursts
	BurstGapMax      int
}

// DefaultScheduleConfig returns a realistic work schedule
func DefaultScheduleConfig() *ScheduleConfig {
	return &ScheduleConfig{
		WorkStartHour:  9,
		WorkEndHour:    17,
		StartVariation: 30, // Start between 8:30-9:30
		EndVariation:   30, // End between 4:30-5:30

		WorkDays: []time.Weekday{
			time.Monday,
			time.Tuesday,
			time.Wednesday,
			time.Thursday,
			time.Friday,
		},

		LunchStartHour:   12,
		LunchDurationMin: 30,
		LunchDurationMax: 60,

		ShortBreakChance:      0.15, // 15% chance after each burst
		ShortBreakDurationMin: 5,
		ShortBreakDurationMax: 15,

		BurstDurationMin: 15, // Work for 15-45 min
		BurstDurationMax: 45,
		BurstGapMin:      5, // Then rest 5-20 min
		BurstGapMax:      20,
	}
}

// Global schedule config
var ScheduleCfg = DefaultScheduleConfig()

// Scheduler manages activity timing
type Scheduler struct {
	config *ScheduleConfig

	// Daily state (recalculated each day)
	todayStart    time.Time
	todayEnd      time.Time
	todayLunch    time.Time
	lunchDuration time.Duration

	// Current burst tracking
	burstStart    time.Time
	burstDuration time.Duration
	inBurst       bool

	lastActivity time.Time
	initialized  bool
	currentDay   int // day of year
}

// NewScheduler creates a scheduler with default config
func NewScheduler() *Scheduler {
	return NewSchedulerWithConfig(ScheduleCfg)
}

// NewSchedulerWithConfig creates a scheduler with custom config
func NewSchedulerWithConfig(cfg *ScheduleConfig) *Scheduler {
	s := &Scheduler{config: cfg}
	s.initDay()
	return s
}

// initDay sets up today's schedule with variation
func (s *Scheduler) initDay() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Calculate today's start time with variation
	startVariation := rand.Intn(s.config.StartVariation*2+1) - s.config.StartVariation
	s.todayStart = today.Add(time.Duration(s.config.WorkStartHour)*time.Hour +
		time.Duration(startVariation)*time.Minute)

	// Calculate today's end time with variation
	endVariation := rand.Intn(s.config.EndVariation*2+1) - s.config.EndVariation
	s.todayEnd = today.Add(time.Duration(s.config.WorkEndHour)*time.Hour +
		time.Duration(endVariation)*time.Minute)

	// Calculate lunch time with slight variation
	lunchVariation := rand.Intn(31) - 15 // ±15 minutes
	s.todayLunch = today.Add(time.Duration(s.config.LunchStartHour)*time.Hour +
		time.Duration(lunchVariation)*time.Minute)

	// Random lunch duration
	lunchMins := s.config.LunchDurationMin +
		rand.Intn(s.config.LunchDurationMax-s.config.LunchDurationMin+1)
	s.lunchDuration = time.Duration(lunchMins) * time.Minute

	s.currentDay = now.YearDay()
	s.initialized = true
	s.inBurst = false

	fmt.Printf("📅 Today's schedule: %s - %s (lunch at %s for %d min)\n",
		s.todayStart.Format("3:04 PM"),
		s.todayEnd.Format("3:04 PM"),
		s.todayLunch.Format("3:04 PM"),
		lunchMins)
}

// refreshIfNewDay checks if we need to recalculate today's schedule
func (s *Scheduler) refreshIfNewDay() {
	now := time.Now()
	if now.YearDay() != s.currentDay || !s.initialized {
		s.initDay()
	}
}

// IsWorkDay returns true if today is a work day
func (s *Scheduler) IsWorkDay() bool {
	today := time.Now().Weekday()
	for _, wd := range s.config.WorkDays {
		if wd == today {
			return true
		}
	}
	return false
}

// IsWorkHours returns true if current time is within work hours
func (s *Scheduler) IsWorkHours() bool {
	s.refreshIfNewDay()

	if !s.IsWorkDay() {
		return false
	}

	now := time.Now()
	return now.After(s.todayStart) && now.Before(s.todayEnd)
}

// IsLunchTime returns true if it's currently lunch break
func (s *Scheduler) IsLunchTime() bool {
	s.refreshIfNewDay()

	now := time.Now()
	lunchEnd := s.todayLunch.Add(s.lunchDuration)
	return now.After(s.todayLunch) && now.Before(lunchEnd)
}

// CanOperate returns true if it's appropriate to perform actions
func (s *Scheduler) CanOperate() bool {
	return s.IsWorkHours() && !s.IsLunchTime()
}

// WaitUntilCanOperate blocks until it's appropriate to operate
// Returns false if should stop (e.g., end of day approaching)
func (s *Scheduler) WaitUntilCanOperate() bool {
	for {
		s.refreshIfNewDay()

		if s.CanOperate() {
			return true
		}

		now := time.Now()

		// If it's lunch, wait for lunch to end
		if s.IsLunchTime() {
			lunchEnd := s.todayLunch.Add(s.lunchDuration)
			waitTime := lunchEnd.Sub(now) + time.Duration(rand.Intn(300))*time.Second
			fmt.Printf("🍽️ Lunch break - waiting %v\n", waitTime.Round(time.Minute))
			time.Sleep(waitTime)
			continue
		}

		// If before work hours, wait until start
		if now.Before(s.todayStart) {
			waitTime := s.todayStart.Sub(now)
			fmt.Printf("⏰ Before work hours - waiting %v\n", waitTime.Round(time.Minute))
			time.Sleep(waitTime)
			continue
		}

		// If after work hours
		if now.After(s.todayEnd) {
			fmt.Println("🌙 After work hours - stopping for today")
			return false
		}

		// If not a work day
		if !s.IsWorkDay() {
			fmt.Println("📅 Not a work day - stopping")
			return false
		}

		// Safety sleep
		time.Sleep(time.Minute)
	}
}

// StartBurst begins a new activity burst
func (s *Scheduler) StartBurst() {
	burstMins := s.config.BurstDurationMin +
		rand.Intn(s.config.BurstDurationMax-s.config.BurstDurationMin+1)
	s.burstDuration = time.Duration(burstMins) * time.Minute
	s.burstStart = time.Now()
	s.inBurst = true

	fmt.Printf("🚀 Starting activity burst (%d min)\n", burstMins)
}

// ShouldTakeBreak returns true if it's time for a break
func (s *Scheduler) ShouldTakeBreak() bool {
	if !s.inBurst {
		return false
	}

	// Check if burst duration exceeded
	if time.Since(s.burstStart) > s.burstDuration {
		return true
	}

	// Random short break chance
	return rand.Float64() < s.config.ShortBreakChance/10 // Per-check probability
}

// TakeBreak pauses for an appropriate break duration
func (s *Scheduler) TakeBreak() {
	s.inBurst = false

	// Determine break type
	if rand.Float64() < 0.3 { // 30% chance of short break
		breakMins := s.config.ShortBreakDurationMin +
			rand.Intn(s.config.ShortBreakDurationMax-s.config.ShortBreakDurationMin+1)
		fmt.Printf("☕ Short break (%d min)\n", breakMins)
		time.Sleep(time.Duration(breakMins) * time.Minute)
	} else {
		// Normal gap between bursts
		gapMins := s.config.BurstGapMin +
			rand.Intn(s.config.BurstGapMax-s.config.BurstGapMin+1)
		fmt.Printf("💤 Resting between activities (%d min)\n", gapMins)
		time.Sleep(time.Duration(gapMins) * time.Minute)
	}
}

// RecordActivity logs that an activity was performed
func (s *Scheduler) RecordActivity() {
	s.lastActivity = time.Now()
}

// TimeSinceLastActivity returns duration since last recorded activity
func (s *Scheduler) TimeSinceLastActivity() time.Duration {
	if s.lastActivity.IsZero() {
		return 0
	}
	return time.Since(s.lastActivity)
}

// GetStatus returns a human-readable status string
func (s *Scheduler) GetStatus() string {
	s.refreshIfNewDay()

	if !s.IsWorkDay() {
		return "🏠 Weekend/Holiday"
	}

	now := time.Now()

	if now.Before(s.todayStart) {
		return fmt.Sprintf("⏰ Before work (starts %s)", s.todayStart.Format("3:04 PM"))
	}

	if now.After(s.todayEnd) {
		return "🌙 After work hours"
	}

	if s.IsLunchTime() {
		lunchEnd := s.todayLunch.Add(s.lunchDuration)
		return fmt.Sprintf("🍽️ Lunch break (until %s)", lunchEnd.Format("3:04 PM"))
	}

	if s.inBurst {
		remaining := s.burstDuration - time.Since(s.burstStart)
		return fmt.Sprintf("🚀 Active burst (%v remaining)", remaining.Round(time.Minute))
	}

	return "✅ Ready to work"
}

// === Convenience functions for simple usage ===

// ShouldRunNow returns true if automation should run right now
func ShouldRunNow() bool {
	s := NewScheduler()
	return s.CanOperate()
}

// WaitForWorkHours blocks until work hours, returns false if should stop
func WaitForWorkHours() bool {
	s := NewScheduler()
	return s.WaitUntilCanOperate()
}

// GetScheduleStatus returns current schedule status
func GetScheduleStatus() string {
	s := NewScheduler()
	return s.GetStatus()
}
