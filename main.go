package main

import (
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/joho/godotenv"

	"github.com/Nehilsa2/linkedin_automation/auth"
	"github.com/Nehilsa2/linkedin_automation/persistence"
	"github.com/Nehilsa2/linkedin_automation/stealth"
)

// Configuration - Set these to control which workflows run
const (
	// Enable/disable workflows
	RunSearchWorkflow     = true
	RunConnectionWorkflow = true
	RunMessagingWorkflow  = true

	// Dry run mode (set to false to perform real actions)
	DryRunMode = true

	// Schedule enforcement (set to false to ignore work hours)
	EnforceSchedule = false // TEMPORARILY DISABLED FOR TESTING

	// Search settings
	SearchKeywordPeople    = "software engineer"
	SearchKeywordCompanies = "fintech"
	SearchMaxPages         = 2

	// Organic browsing settings
	EnableOrganicBrowsing = true // Browse profiles/feed between connections

	// Messaging settings
	MessageTemplate     = "follow_up_simple"
	MaxFollowUpMessages = 3

	// Database settings
	DatabasePath = "linkedin_automation.db"

	// Safety level for rate limiting (all limits controlled from stealth/ratelimit.go)
	// Options: SafetyUltraConservative, SafetyConservative, SafetyModerate, SafetyAggressive
	DefaultSafetyLevel = stealth.SafetyConservative
)

// Global store instance
var store *persistence.Store

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Unable to load .env file; falling back to existing environment variables")
	}

	// ==================== RATE LIMIT CONFIG ====================
	// Initialize rate limiting with the chosen safety level
	stealth.SetSafetyLevel(DefaultSafetyLevel)
	stealth.PrintConfig()

	// ==================== SCHEDULE CHECK ====================
	if EnforceSchedule {
		scheduler := stealth.NewScheduler()
		fmt.Println("📅 Schedule status:", scheduler.GetStatus())

		if !scheduler.CanOperate() {
			fmt.Println("⏰ Outside work hours or on break")
			if !scheduler.WaitUntilCanOperate() {
				fmt.Println("🌙 Work day ended - exiting")
				return
			}
		}

		// Start activity burst
		scheduler.StartBurst()
	}

	// Initialize persistence store
	var err error
	store, err = persistence.NewStore(DatabasePath)
	if err != nil {
		log.Fatal("❌ Failed to initialize database:", err)
	}
	defer store.Close()

	fmt.Println("✅ Database initialized:", DatabasePath)

	// Migrate existing JSON data if present
	store.MigrateFromJSON()

	// Check for resumable workflows
	checkResumableWorkflows()

	// ==================== BROWSER SETUP ====================
	// Simple browser setup with minimal anti-detection
	// Only disables the webdriver flag - nothing else
	u := launcher.New().
		Set("disable-blink-features", "AutomationControlled").
		Headless(false). // VISIBLE browser window
		Leakless(false).
		MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()

	defer browser.MustClose()

	err = auth.EnsureAuthenticated(browser)
	if err != nil {
		log.Fatal("❌ Authentication failed:", err)
	}

	// Use the authenticated feed page for organic browsing
	pages, err := browser.Pages()
	if err != nil || len(pages) == 0 {
		log.Fatal("❌ Could not get feed page after authentication")
	}
	feedPage := pages[len(pages)-1]
	organicBrowser := stealth.NewOrganicBrowser(feedPage)
	organicBrowser.BrowseFeed()
	organicBrowser.RandomDelay()

	var people []string

	// ==================== SEARCH WORKFLOW ====================
	if RunSearchWorkflow {
		var companies []string
		people, companies = RunSearch(browser)
		fmt.Printf("\n📋 Search Summary: %d people, %d companies\n", len(people), len(companies))
	}

	// ==================== CONNECTION WORKFLOW ====================
	if RunConnectionWorkflow {
		RunConnections(browser, people)
	}

	// // ==================== MESSAGING WORKFLOW ====================
	if RunMessagingWorkflow {
		RunMessaging(browser)
	}

	// Print session summary
	printSessionSummary()

	fmt.Println("\n✅ All workflows completed!")
}

// checkResumableWorkflows checks for any paused workflows that can be resumed
func checkResumableWorkflows() {
	fmt.Println("\n🔍 Checking for resumable workflows...")

	workflowTypes := []string{
		persistence.WorkflowTypeSearch,
		persistence.WorkflowTypeConnect,
		persistence.WorkflowTypeMessage,
	}

	for _, wfType := range workflowTypes {
		state, err := store.GetActiveWorkflow(wfType)
		if err != nil {
			continue
		}
		if state != nil && state.Status == persistence.WorkflowStatusPaused {
			fmt.Printf("⏸️ Found paused %s workflow (progress: %d/%d)\n",
				wfType, state.CurrentIndex, state.TotalItems)
			fmt.Printf("   Started: %s\n", state.StartedAt.Format("2006-01-02 15:04:05"))
			if state.PausedAt != nil {
				fmt.Printf("   Paused: %s\n", state.PausedAt.Format("2006-01-02 15:04:05"))
			}
		}
	}
}

// printSessionSummary prints a summary of today's activity
func printSessionSummary() {
	fmt.Println("\n==================================================")
	fmt.Println("📊 SESSION SUMMARY")
	fmt.Println("==================================================")

	stats, err := store.GetDailyStats("")
	if err != nil {
		fmt.Printf("⚠️ Could not get daily stats: %v\n", err)
		return
	}

	fmt.Printf("\n📅 Today's Activity:\n")
	fmt.Printf("   🔍 Profiles discovered: %d\n", stats.ProfilesSearched)
	fmt.Printf("   🔗 Connection requests sent: %d\n", stats.ConnectionsSent)
	fmt.Printf("   ✅ Connections accepted: %d\n", stats.ConnectionsAccepted)
	fmt.Printf("   📬 Messages sent: %d\n", stats.MessagesSent)

	// Connection stats
	connStats, err := store.GetConnectionRequestStats(100)
	if err == nil {
		fmt.Printf("\n📈 Overall Connection Stats:\n")
		fmt.Printf("   Total requests: %d\n", connStats.TotalSent)
		fmt.Printf("   Pending: %d\n", connStats.Pending)
		fmt.Printf("   Accepted: %d (%.1f%% rate)\n", connStats.Accepted, connStats.AcceptanceRate)
	}

	// Message stats
	msgStats, err := store.GetMessageStats(100)
	if err == nil {
		fmt.Printf("\n📬 Overall Message Stats:\n")
		fmt.Printf("   Total sent: %d\n", msgStats.TotalSent)
		fmt.Printf("   Initial: %d\n", msgStats.InitialSent)
		fmt.Printf("   Follow-ups: %d\n", msgStats.FollowUpsSent)
	}
}

// GetStore returns the global store instance for use in other packages
func GetStore() *persistence.Store {
	return store
}
