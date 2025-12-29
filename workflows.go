package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/connect"
	"github.com/Nehilsa2/linkedin_automation/humanize"
	"github.com/Nehilsa2/linkedin_automation/message"
	"github.com/Nehilsa2/linkedin_automation/persistence"
	"github.com/Nehilsa2/linkedin_automation/search"
)

// RunSearch searches for people and companies on LinkedIn
func RunSearch(browser *rod.Browser) ([]string, []string) {
	fmt.Println("\n==================================================")
	fmt.Println("🔍 SEARCH WORKFLOW")
	fmt.Println("==================================================")

	// Create workflow state for resumption
	workflowState := &persistence.WorkflowState{
		WorkflowType: persistence.WorkflowTypeSearch,
		Status:       persistence.WorkflowStatusInProgress,
		CurrentStep:  "searching_people",
		Metadata: map[string]interface{}{
			"keyword_people":    SearchKeywordPeople,
			"keyword_companies": SearchKeywordCompanies,
			"max_pages":         SearchMaxPages,
		},
	}

	// Check for existing active workflow
	existing, _ := store.GetActiveWorkflow(persistence.WorkflowTypeSearch)
	if existing != nil && existing.Status == persistence.WorkflowStatusPaused {
		fmt.Printf("📌 Resuming previous search workflow from page %d\n", existing.CurrentIndex)
		workflowState = existing
		workflowState.Status = persistence.WorkflowStatusInProgress
	}

	store.SaveWorkflowState(workflowState)

	// Search for people
	fmt.Printf("\n👤 Searching for people: %s\n", SearchKeywordPeople)
	people, err := search.FindPeople(browser, SearchKeywordPeople, SearchMaxPages)
	if err != nil {
		log.Printf("⚠️ People search error: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d profiles\n", len(people))

		// Save search results to database
		savePeopleResultsToDB(people, SearchKeywordPeople)
	}

	workflowState.CurrentStep = "searching_companies"
	workflowState.CurrentIndex = SearchMaxPages
	store.SaveWorkflowState(workflowState)

	// Search for companies
	fmt.Printf("\n🏢 Searching for companies: %s\n", SearchKeywordCompanies)
	companies, err := search.FindCompanies(browser, SearchKeywordCompanies, SearchMaxPages)
	if err != nil {
		log.Printf("⚠️ Company search error: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d companies\n", len(companies))

		// Save company search results
		saveCompanyResultsToDB(companies, SearchKeywordCompanies)
	}

	// Mark workflow as complete
	store.CompleteWorkflow(workflowState.ID)

	return people, companies
}

// savePeopleResultsToDB saves people search results to the database
func savePeopleResultsToDB(urls []string, keyword string) {
	results := make([]persistence.PersonSearchResult, 0, len(urls))

	for i, url := range urls {
		// Check if already exists
		exists, _ := store.HasPersonResult(url)
		if exists {
			continue
		}

		results = append(results, persistence.PersonSearchResult{
			ProfileURL:    url,
			SearchKeyword: keyword,
			PageNumber:    (i / 10) + 1, // Estimate page number
			DiscoveredAt:  time.Now(),
		})
	}

	if len(results) > 0 {
		if err := store.SavePersonSearchResults(results); err != nil {
			fmt.Printf("⚠️ Failed to save people search results: %v\n", err)
		} else {
			fmt.Printf("💾 Saved %d new people profiles to database\n", len(results))
		}
	}
}

// saveCompanyResultsToDB saves company search results to the database
func saveCompanyResultsToDB(urls []string, keyword string) {
	results := make([]persistence.CompanySearchResult, 0, len(urls))

	for i, url := range urls {
		// Check if already exists
		exists, _ := store.HasCompanyResult(url)
		if exists {
			continue
		}

		results = append(results, persistence.CompanySearchResult{
			CompanyURL:    url,
			SearchKeyword: keyword,
			PageNumber:    (i / 10) + 1, // Estimate page number
			DiscoveredAt:  time.Now(),
		})
	}

	if len(results) > 0 {
		if err := store.SaveCompanySearchResults(results); err != nil {
			fmt.Printf("⚠️ Failed to save company search results: %v\n", err)
		} else {
			fmt.Printf("💾 Saved %d new companies to database\n", len(results))
		}
	}
}

// RunConnections sends connection requests to found profiles
func RunConnections(browser *rod.Browser, profileURLs []string) {
	fmt.Println("\n==================================================")
	fmt.Println("🔗 CONNECTION WORKFLOW")
	fmt.Println("==================================================")

	// Create workflow state
	workflowState := &persistence.WorkflowState{
		WorkflowType: persistence.WorkflowTypeConnect,
		Status:       persistence.WorkflowStatusInProgress,
		CurrentStep:  "sending_requests",
		TotalItems:   len(profileURLs),
	}

	// Check for resumable workflow
	existing, _ := store.GetActiveWorkflow(persistence.WorkflowTypeConnect)
	if existing != nil && existing.Status == persistence.WorkflowStatusPaused {
		fmt.Printf("📌 Resuming connection workflow from index %d/%d\n",
			existing.CurrentIndex, existing.TotalItems)
		workflowState = existing
		workflowState.Status = persistence.WorkflowStatusInProgress

		// Adjust profileURLs to skip already processed
		if existing.CurrentIndex < len(profileURLs) {
			profileURLs = profileURLs[existing.CurrentIndex:]
		}
	}

	store.SaveWorkflowState(workflowState)

	// Load legacy tracker (for backward compatibility)
	tracker, err := connect.LoadTracker()
	if err != nil {
		log.Printf("⚠️ Failed to load tracker: %v\n", err)
		return
	}

	// Set dry run mode
	tracker.SetDryRun(DryRunMode)
	tracker.SetDailyLimit(50)

	// Print stats from database
	connStats, err := store.GetConnectionRequestStats(50)
	if err == nil {
		fmt.Printf("\n📊 Connection Stats:\n")
		fmt.Printf("   Total sent: %d\n", connStats.TotalSent)
		fmt.Printf("   Today: %d/%d\n", connStats.SentToday, connStats.DailyLimit)
		fmt.Printf("   Remaining: %d\n", connStats.RemainingToday)
		fmt.Printf("   Acceptance rate: %.1f%%\n", connStats.AcceptanceRate)
	}

	if len(profileURLs) == 0 {
		// Try to get unprocessed profiles from database
		unprocessed, _ := store.GetUnprocessedSearchResults(SearchKeywordPeople, MaxConnectionRequests)
		if len(unprocessed) > 0 {
			fmt.Printf("📋 Found %d unprocessed profiles in database\n", len(unprocessed))
			for _, r := range unprocessed {
				profileURLs = append(profileURLs, r.ProfileURL)
			}
		} else {
			fmt.Println("ℹ️ No profiles to connect with")
			return
		}
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

	successCount := 0
	failCount := 0

	for i, profileURL := range profileURLs[:maxRequests] {
		// Check if already sent (in database)
		sent, _ := store.HasSentRequest(profileURL)
		if sent {
			fmt.Printf("⏭️ Skipping %s (already sent)\n", profileURL)
			continue
		}

		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, maxRequests, profileURL)

		// Update workflow progress
		workflowState.CurrentIndex = i
		store.UpdateWorkflowProgress(workflowState.ID, i, "processing_profile")

		err := connect.ConnectWithTracking(page, profileURL, "", noteTemplate, tracker)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			failCount++
		} else {
			successCount++

			// Save to database
			req := &persistence.ConnectionRequest{
				ProfileURL:    profileURL,
				Note:          noteTemplate,
				Status:        persistence.StatusPending,
				SentAt:        time.Now(),
				Source:        "search",
				SearchKeyword: SearchKeywordPeople,
			}
			if !DryRunMode {
				store.SaveConnectionRequest(req)
			}

			// Mark search result as processed
			store.MarkSearchResultProcessed(profileURL)
		}

		// Randomized delay between requests (human-like)
		if i < maxRequests-1 {
			humanize.Sleep(ConnectionDelayMin, ConnectionDelayMax)
		}
	}

	// Mark workflow complete
	store.CompleteWorkflow(workflowState.ID)

	fmt.Printf("\n✅ Connection Results: %d sent, %d failed\n", successCount, failCount)
}

// RunMessaging sends follow-up messages to connections
func RunMessaging(browser *rod.Browser) {
	fmt.Println("\n==================================================")
	fmt.Println("📬 MESSAGING WORKFLOW")
	fmt.Println("==================================================")

	// Create workflow state
	workflowState := &persistence.WorkflowState{
		WorkflowType: persistence.WorkflowTypeMessage,
		Status:       persistence.WorkflowStatusInProgress,
		CurrentStep:  "sending_messages",
	}

	// Check for resumable workflow
	existing, _ := store.GetActiveWorkflow(persistence.WorkflowTypeMessage)
	if existing != nil && existing.Status == persistence.WorkflowStatusPaused {
		fmt.Printf("📌 Resuming messaging workflow from index %d/%d\n",
			existing.CurrentIndex, existing.TotalItems)
		workflowState = existing
		workflowState.Status = persistence.WorkflowStatusInProgress
	}

	store.SaveWorkflowState(workflowState)

	// Get a page to work with
	page := browser.MustPage()
	defer page.Close()

	// Create messaging service
	msgService, err := message.NewMessagingService(page)
	if err != nil {
		log.Printf("⚠️ Failed to create messaging service: %v\n", err)
		store.FailWorkflow(workflowState.ID, err.Error())
		return
	}
	defer msgService.Close()

	// Set dry run mode
	msgService.SetDryRun(DryRunMode)
	msgService.SetDailyLimit(50)

	// Show available templates
	msgService.ListTemplates()

	// Print stats from database
	msgStats, err := store.GetMessageStats(50)
	if err == nil {
		fmt.Printf("\n📊 Message Stats (from database):\n")
		fmt.Printf("   Total sent: %d\n", msgStats.TotalSent)
		fmt.Printf("   Today: %d/%d\n", msgStats.SentToday, msgStats.DailyLimit)
		fmt.Printf("   Follow-ups: %d\n", msgStats.FollowUpsSent)
	}

	// Get unmessaged connections from database
	unmessaged, err := store.GetUnmessagedConnections()
	if err == nil && len(unmessaged) > 0 {
		fmt.Printf("\n📋 Found %d unmessaged connections in database\n", len(unmessaged))
		workflowState.TotalItems = len(unmessaged)
		store.SaveWorkflowState(workflowState)
	}

	// Run full workflow (detect connections + send follow-ups)
	err = msgService.FullWorkflow(
		MessageTemplate,
		MaxFollowUpMessages,
		MessageDelayMin,
		MessageDelayMax,
	)
	if err != nil {
		log.Printf("⚠️ Workflow error: %v\n", err)
		store.FailWorkflow(workflowState.ID, err.Error())
	} else {
		store.CompleteWorkflow(workflowState.ID)
	}

	// Final stats
	fmt.Println("\n📊 Final Messaging Statistics:")
	msgService.PrintStats()
}
