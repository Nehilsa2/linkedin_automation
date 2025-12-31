package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/connect"
	"github.com/Nehilsa2/linkedin_automation/message"
	"github.com/Nehilsa2/linkedin_automation/persistence"
	"github.com/Nehilsa2/linkedin_automation/search"
	"github.com/Nehilsa2/linkedin_automation/stealth"
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
	if len(people) > 0 {
		fmt.Printf("✅ Found %d profiles\n", len(people))
		savePeopleResultsToDB(people, SearchKeywordPeople)
	}
	if err != nil {
		log.Printf("⚠️ People search error: %v\n", err)
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

// RunConnections sends connection requests to found profiles with organic browsing
// Flow: Browse random profile -> Feed -> Quick view target -> Connect
func RunConnections(browser *rod.Browser, profileURLs []string) {
	fmt.Println("\n==================================================")
	fmt.Println("🔗 CONNECTION WORKFLOW (with organic browsing)")
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

	// Set dry run mode and safe daily limit from central config
	tracker.SetDryRun(DryRunMode)
	tracker.SetDailyLimit(stealth.GetConnectionDailyLimit())

	// Print stats from database
	connStats, err := store.GetConnectionRequestStats(stealth.GetConnectionDailyLimit())
	if err == nil {
		fmt.Printf("\n📊 Connection Stats:\n")
		fmt.Printf("   Total sent: %d\n", connStats.TotalSent)
		fmt.Printf("   Today: %d/%d\n", connStats.SentToday, connStats.DailyLimit)
		fmt.Printf("   Remaining: %d\n", connStats.RemainingToday)
		fmt.Printf("   Acceptance rate: %.1f%%\n", connStats.AcceptanceRate)
	}

	if len(profileURLs) == 0 {
		// Try to get unprocessed profiles from database
		// Get extra profiles for browsing (3x the daily limit)
		unprocessed, _ := store.GetUnprocessedSearchResults(SearchKeywordPeople, stealth.GetConnectionDailyLimit()*3)
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

	// Limit requests based on central config
	maxRequests := stealth.GetConnectionDailyLimit()
	if len(profileURLs) < maxRequests {
		maxRequests = len(profileURLs)
	}

	fmt.Printf("\n🔗 Will send up to %d connection requests with organic browsing...\n", maxRequests)

	successCount := 0
	failCount := 0
	browseIndex := maxRequests // Start browsing from profiles after targets

	// Create scheduler for break management
	var scheduler *stealth.Scheduler
	if EnforceSchedule {
		scheduler = stealth.NewScheduler()
		scheduler.StartBurst()
	}

	// Get rate limiter for connection requests
	rateLimiter := stealth.GetRateLimiter()
	rateLimiter.PrintStats(stealth.ActionConnection)

	// Create organic browser for human-like behavior
	organicBrowser := stealth.NewOrganicBrowser(page)

	for i := 0; i < maxRequests; i++ {
		targetURL := profileURLs[i]

		// Check rate limits first
		if can, reason := rateLimiter.CanPerform(stealth.ActionConnection); !can {
			fmt.Printf("⏸️ Rate limited: %s\n", reason)
			if !rateLimiter.WaitForAction(stealth.ActionConnection) {
				fmt.Println("⏰ Rate limit wait too long - stopping workflow")
				workflowState.Status = persistence.WorkflowStatusPaused
				store.PauseWorkflow(workflowState.ID)
				break
			}
		}

		// Check schedule before each action
		if EnforceSchedule && scheduler != nil {
			if !scheduler.CanOperate() {
				fmt.Println("⏰ Work hours ended or on break - pausing workflow")
				workflowState.Status = persistence.WorkflowStatusPaused
				store.PauseWorkflow(workflowState.ID)
				break
			}

			// Check if we should take a break
			if scheduler.ShouldTakeBreak() {
				fmt.Println("☕ Taking a break...")
				workflowState.Status = persistence.WorkflowStatusPaused
				store.SaveWorkflowState(workflowState)
				scheduler.TakeBreak()
				scheduler.StartBurst()
				workflowState.Status = persistence.WorkflowStatusInProgress
				store.SaveWorkflowState(workflowState)
			}
		}

		// Check if already sent (in database)
		sent, _ := store.HasSentRequest(targetURL)
		if sent {
			fmt.Printf("⏭️ Skipping %s (already sent)\n", targetURL)
			continue
		}

		fmt.Printf("\n========== [%d/%d] Connection Cycle ==========\n", i+1, maxRequests)

		// Update workflow progress
		workflowState.CurrentIndex = i
		store.UpdateWorkflowProgress(workflowState.ID, i, "organic_browsing")

		// ==================== ORGANIC BROWSING PHASE ====================
		if EnableOrganicBrowsing {
			// Step 1: Browse a random profile (not the target) for ~10 seconds
			var browseURL string
			if browseIndex < len(profileURLs) {
				browseURL = profileURLs[browseIndex]
				browseIndex++
			}

			if browseURL != "" && browseURL != targetURL {
				fmt.Println("\n📖 Step 1: Browsing random profile...")
				if err := organicBrowser.BrowseProfile(browseURL); err != nil {
					fmt.Printf("   ⚠️ Browse failed: %v (continuing)\n", err)
				}
				organicBrowser.RandomDelay()
			}

			// Step 2: Go to feed and scroll for 5-6 seconds
			fmt.Println("\n📰 Step 2: Checking LinkedIn feed...")
			if err := organicBrowser.BrowseFeed(); err != nil {
				fmt.Printf("   ⚠️ Feed browse failed: %v (continuing)\n", err)
			}
			organicBrowser.RandomDelay()
		}

		// ==================== CONNECTION PHASE ====================
		// Step 3: Quick view target profile (~5 sec) then connect
		fmt.Printf("\n🎯 Step 3: Target profile: %s\n", targetURL)

		store.UpdateWorkflowProgress(workflowState.ID, i, "connecting")

		// Quick browse the target before connecting
		if EnableOrganicBrowsing {
			if err := organicBrowser.BrowseProfileQuick(targetURL); err != nil {
				fmt.Printf("   ⚠️ Target browse failed: %v\n", err)
				// Check if critical error
				if stealth.IsCritical(err) {
					fmt.Println("🛑 Critical error detected - stopping workflow")
					workflowState.Status = persistence.WorkflowStatusPaused
					store.PauseWorkflow(workflowState.ID)
					break
				}
			}
		}

		// Now send the connection request (page is already on target profile)
		err := connect.ConnectWithTracking(page, targetURL, "", noteTemplate, tracker)
		if err != nil {
			fmt.Printf("❌ Connection failed: %v\n", err)
			failCount++

			// Check if this is a critical LinkedIn error
			if stealth.IsCritical(err) {
				fmt.Println("🛑 Critical error detected - stopping workflow")
				workflowState.Status = persistence.WorkflowStatusPaused
				store.PauseWorkflow(workflowState.ID)
				break
			}

			// For non-recoverable errors, may need longer cooldown
			if !stealth.IsRecoverable(err) {
				fmt.Println("⏸️ Non-recoverable error - taking extended break...")
				stealth.Sleep(60, 120) // 1-2 minute break
			}
		} else {
			successCount++
			fmt.Printf("✅ Connection request sent!\n")

			// Record action for rate limiting
			rateLimiter.RecordAction(stealth.ActionConnection)

			// Save to database (track stats even in dry run mode)
			req := &persistence.ConnectionRequest{
				ProfileURL:    targetURL,
				Note:          noteTemplate,
				Status:        persistence.StatusPending,
				SentAt:        time.Now(),
				Source:        "search",
				SearchKeyword: SearchKeywordPeople,
			}

			if DryRunMode {
				fmt.Println("   📝 [DRY RUN] Would save connection request to database")
				// Still increment the daily stat for tracking purposes
				store.IncrementConnectionsSent()
			} else {
				store.SaveConnectionRequest(req)
			}

			// Mark search result as processed
			store.MarkSearchResultProcessed(targetURL)
		}

		// ==================== DELAY BEFORE NEXT CYCLE ====================
		if i < maxRequests-1 {
			// Use centralized delay configuration
			delay := stealth.GetRandomDelay(stealth.ActionConnection)

			fmt.Printf("\n⏳ Waiting %v before next connection cycle...\n", delay.Round(time.Second))
			time.Sleep(delay)
		}
	}

	// Print final rate limit stats
	rateLimiter.PrintStats(stealth.ActionConnection)

	// Mark workflow complete
	store.CompleteWorkflow(workflowState.ID)

	fmt.Printf("\n✅ Connection Results: %d sent, %d failed\n", successCount, failCount)
	if EnableOrganicBrowsing {
		fmt.Println("   (Organic browsing was enabled for stealth)")
	}
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

	// Set dry run mode and use central config for limits
	msgService.SetDryRun(DryRunMode)
	msgService.SetDailyLimit(stealth.GetMessageDailyLimit())

	// Show available templates
	msgService.ListTemplates()

	// Print stats from database
	msgStats, err := store.GetMessageStats(stealth.GetMessageDailyLimit())
	if err == nil {
		fmt.Printf("\n📊 Message Stats (from database):\n")
		fmt.Printf("   Total sent: %d\n", msgStats.TotalSent)
		fmt.Printf("   Today: %d/%d\n", msgStats.SentToday, msgStats.DailyLimit)
		fmt.Printf("   Follow-ups: %d\n", msgStats.FollowUpsSent)
	}

	// Get unmessaged connections from database (4 days old or older)
	unmessaged, err := store.GetUnmessagedConnections()
	if err == nil && len(unmessaged) > 0 {
		fmt.Printf("\n📋 Found %d unmessaged connections (4 days old or older) in database\n", len(unmessaged))
		workflowState.TotalItems = len(unmessaged)
		store.SaveWorkflowState(workflowState)
	} else if err == nil && len(unmessaged) == 0 {
		fmt.Println("ℹ️ No connections from 4 days ago or earlier to message")
	}

	// Run full workflow (detect connections + send follow-ups)
	// Use centralized delay config
	err = msgService.FullWorkflow(
		MessageTemplate,
		MaxFollowUpMessages,
		stealth.GetMessageDelayMin(),
		stealth.GetMessageDelayMax(),
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
