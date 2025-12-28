package main

import (
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/joho/godotenv"

	"github.com/Nehilsa2/linkedin_automation/auth"
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
