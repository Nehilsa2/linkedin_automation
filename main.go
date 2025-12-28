package main

import (
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/joho/godotenv"

	"github.com/Nehilsa2/linkedin_automation/auth"
	"github.com/Nehilsa2/linkedin_automation/connect"
	"github.com/Nehilsa2/linkedin_automation/message"
	"github.com/Nehilsa2/linkedin_automation/search"
)

// Configuration - Set these to control which workflows run
const (
	// Enable/disable workflows
	RunSearchWorkflow     = true
	RunConnectionWorkflow = true
	RunMessagingWorkflow  = true

	// Dry run mode (set to false to perform real actions)
	DryRunMode = true

	// Search settings
	SearchKeywordPeople    = "software engineer"
	SearchKeywordCompanies = "fintech"
	SearchMaxPages         = 2

	// Connection settings
	MaxConnectionRequests = 3
	ConnectionDelay       = 10 // seconds

	// Messaging settings
	MessageTemplate     = "follow_up_simple"
	MaxFollowUpMessages = 3
	MessageDelay        = 10 // seconds
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Unable to load .env file; falling back to existing environment variables")
	}

	u := launcher.New().
		Leakless(false).
		Headless(false).
		MustLaunch()

	browser := rod.New().
		ControlURL(u).
		MustConnect()

	defer browser.MustClose()

	err := auth.EnsureAuthenticated(browser)
	if err != nil {
		log.Fatal("❌ Authentication failed:", err)
	}

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

	// ==================== MESSAGING WORKFLOW ====================
	if RunMessagingWorkflow {
		RunMessaging(browser)
	}

	fmt.Println("\n✅ All workflows completed!")
}

// RunSearch searches for people and companies on LinkedIn
func RunSearch(browser *rod.Browser) ([]string, []string) {
	fmt.Println("\n==================================================")
	fmt.Println("🔍 SEARCH WORKFLOW")
	fmt.Println("==================================================")

	// Search for people
	fmt.Printf("\n👤 Searching for people: %s\n", SearchKeywordPeople)
	people, err := search.FindPeople(browser, SearchKeywordPeople, SearchMaxPages)
	if err != nil {
		log.Printf("⚠️ People search error: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d profiles\n", len(people))
	}

	// Search for companies
	fmt.Printf("\n🏢 Searching for companies: %s\n", SearchKeywordCompanies)
	companies, err := search.FindCompanies(browser, SearchKeywordCompanies, SearchMaxPages)
	if err != nil {
		log.Printf("⚠️ Company search error: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d companies\n", len(companies))
	}

	return people, companies
}

// RunConnections sends connection requests to found profiles
func RunConnections(browser *rod.Browser, profileURLs []string) {
	fmt.Println("\n==================================================")
	fmt.Println("🔗 CONNECTION WORKFLOW")
	fmt.Println("==================================================")

	// Load tracker
	tracker, err := connect.LoadTracker()
	if err != nil {
		log.Printf("⚠️ Failed to load tracker: %v\n", err)
		return
	}

	// Set dry run mode
	tracker.SetDryRun(DryRunMode)
	tracker.SetDailyLimit(50)

	// Print current stats
	stats := tracker.GetStats()
	fmt.Printf("\n📊 Connection Stats:\n")
	fmt.Printf("   Total sent: %d\n", stats["total"])
	fmt.Printf("   Today: %d/%d\n", stats["today"], stats["daily_limit"])
	fmt.Printf("   Remaining: %d\n", stats["remaining"])

	if len(profileURLs) == 0 {
		fmt.Println("ℹ️ No profiles to connect with")
		return
	}

	// Personalized note template
	noteTemplate := "Hi! I came across your profile and would love to connect. Looking forward to learning from your experience!"

	// Get a page to work with
	page := browser.MustPage()
	defer page.Close()

	// Limit requests
	maxRequests := MaxConnectionRequests
	if len(profileURLs) < maxRequests {
		maxRequests = len(profileURLs)
	}

	fmt.Printf("\n🔗 Sending %d connection requests...\n", maxRequests)

	success, failed, _ := connect.BatchConnect(
		page,
		profileURLs[:maxRequests],
		noteTemplate,
		tracker,
		ConnectionDelay,
	)

	fmt.Printf("\n✅ Connection Results: %d sent, %d failed\n", success, failed)
}

// RunMessaging sends follow-up messages to connections
func RunMessaging(browser *rod.Browser) {
	fmt.Println("\n==================================================")
	fmt.Println("📬 MESSAGING WORKFLOW")
	fmt.Println("==================================================")

	// Get a page to work with
	page := browser.MustPage()
	defer page.Close()

	// Create messaging service
	msgService, err := message.NewMessagingService(page)
	if err != nil {
		log.Printf("⚠️ Failed to create messaging service: %v\n", err)
		return
	}
	defer msgService.Close()

	// Set dry run mode
	msgService.SetDryRun(DryRunMode)
	msgService.SetDailyLimit(50)

	// Show available templates
	msgService.ListTemplates()

	// Print current stats
	msgService.PrintStats()

	// Run full workflow (detect connections + send follow-ups)
	err = msgService.FullWorkflow(
		MessageTemplate,
		MaxFollowUpMessages,
		MessageDelay,
	)
	if err != nil {
		log.Printf("⚠️ Workflow error: %v\n", err)
	}

	// Final stats
	fmt.Println("\n📊 Final Messaging Statistics:")
	msgService.PrintStats()
}
