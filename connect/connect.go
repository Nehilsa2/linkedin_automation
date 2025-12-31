package connect

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/stealth"
)

const (
	// LinkedIn limits
	MaxNoteLength = 300

	// Storage file for tracking
	RequestsFile = "connection_requests.json"
)

// GetDefaultDailyLimit returns the daily limit from central config
func GetDefaultDailyLimit() int {
	return stealth.GetConnectionDailyLimit()
}

// ConnectionRequest represents a sent connection request
type ConnectionRequest struct {
	ProfileURL string    `json:"profile_url"`
	Name       string    `json:"name"`
	Note       string    `json:"note,omitempty"`
	SentAt     time.Time `json:"sent_at"`
	Status     string    `json:"status"` // "sent", "pending", "accepted", "declined"
}

// ConnectionTracker tracks sent requests and enforces limits
type ConnectionTracker struct {
	Requests   []ConnectionRequest `json:"requests"`
	DailyLimit int                 `json:"daily_limit"`
	DryRun     bool                `json:"-"` // Don't persist this flag
}

// LoadTracker loads the tracker from file
func LoadTracker() (*ConnectionTracker, error) {
	tracker := &ConnectionTracker{
		Requests:   []ConnectionRequest{},
		DailyLimit: stealth.GetConnectionDailyLimit(), // Use central config
	}

	data, err := os.ReadFile(RequestsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return tracker, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, tracker); err != nil {
		return nil, err
	}

	return tracker, nil
}

// Save saves the tracker to file
func (t *ConnectionTracker) Save() error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(RequestsFile, data, 0644)
}

// GetTodayCount returns the number of requests sent today
func (t *ConnectionTracker) GetTodayCount() int {
	today := time.Now().Truncate(24 * time.Hour)
	count := 0
	for _, req := range t.Requests {
		if req.SentAt.After(today) {
			count++
		}
	}
	return count
}

// CanSendMore checks if more requests can be sent today
func (t *ConnectionTracker) CanSendMore() bool {
	return t.GetTodayCount() < t.DailyLimit
}

// RemainingToday returns how many more requests can be sent today
func (t *ConnectionTracker) RemainingToday() int {
	return t.DailyLimit - t.GetTodayCount()
}

// AddRequest adds a new request to the tracker
func (t *ConnectionTracker) AddRequest(req ConnectionRequest) {
	t.Requests = append(t.Requests, req)
}

// AlreadySent checks if a request was already sent to this profile
func (t *ConnectionTracker) AlreadySent(profileURL string) bool {
	normalized := normalizeProfileURL(profileURL)
	for _, req := range t.Requests {
		if normalizeProfileURL(req.ProfileURL) == normalized {
			return true
		}
	}
	return false
}

// normalizeProfileURL normalizes LinkedIn profile URLs for comparison
func normalizeProfileURL(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	return strings.ToLower(url)
}

// NavigateToProfile navigates to a LinkedIn profile
func NavigateToProfile(page *rod.Page, profileURL string) error {
	fmt.Printf("📍 Navigating to profile: %s\n", profileURL)

	// Set timeout to prevent hanging
	timeoutPage := page.Timeout(15 * time.Second)

	err := timeoutPage.Navigate(profileURL)
	if err != nil {
		timeoutPage.CancelTimeout()
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Wait for page to load with timeout
	err = timeoutPage.WaitStable(time.Second)
	timeoutPage.CancelTimeout()
	if err != nil {
		fmt.Println("⚠️ Page stability wait timed out, continuing anyway...")
	}

	stealth.Sleep(1, 3) // Random delay after page load

	// Check for LinkedIn errors after navigation
	result := stealth.CheckPage(page)
	if result.HasError {
		stealth.PrintDetectionStatus(result)
		return result.Error
	}

	fmt.Println("✅ Profile page loaded")
	return nil
}

// SendConnectionRequest sends a connection request to the current profile
// If note is empty, sends without a note
func SendConnectionRequest(page *rod.Page, note string) error {
	fmt.Println("🔗 Looking for Connect button...")

	// Set timeout to prevent hanging
	page = page.Timeout(15 * time.Second)
	defer page.CancelTimeout()

	// First, try to find and click the Connect button
	result := page.MustEval(`() => {
		// Various Connect button selectors
		const connectSelectors = [
			'button[aria-label*="Invite"][aria-label*="connect"]',
			'button.pvs-profile-actions__action[aria-label*="connect" i]',
			'button[aria-label="Connect"]',
			'button:has(span.artdeco-button__text):has(span:contains("Connect"))',
			'main button[aria-label*="connect" i]',
		];

		// Try each selector
		for (const selector of connectSelectors) {
			try {
				const btn = document.querySelector(selector);
				if (btn && !btn.disabled) {
					const text = btn.innerText.toLowerCase();
					if (text.includes('connect') && !text.includes('message')) {
						btn.scrollIntoView({ block: "center" });
						btn.click();
						return { found: true, clicked: true, error: null };
					}
				}
			} catch (e) {}
		}

		// Try finding by text content
		const buttons = document.querySelectorAll('button');
		for (const btn of buttons) {
			const text = btn.innerText.trim().toLowerCase();
			if (text === 'connect' && !btn.disabled) {
				btn.scrollIntoView({ block: "center" });
				btn.click();
				return { found: true, clicked: true, error: null };
			}
		}

		// Check if already connected or pending
		for (const btn of buttons) {
			const text = btn.innerText.trim().toLowerCase();
			if (text === 'pending' || text === 'message') {
				return { found: false, clicked: false, error: 'already_connected_or_pending' };
			}
		}

		return { found: false, clicked: false, error: 'connect_button_not_found' };
	}`)

	found := result.Get("found").Bool()
	clicked := result.Get("clicked").Bool()
	errorMsg := result.Get("error").Str()

	if !found {
		if errorMsg == "already_connected_or_pending" {
			return fmt.Errorf("already connected or request pending")
		}
		return fmt.Errorf("connect button not found")
	}

	if !clicked {
		return fmt.Errorf("failed to click connect button")
	}

	// Wait for modal to appear
	stealth.SleepMillis(800, 1500)

	// Check for any error responses after clicking connect
	detectionResult := stealth.QuickCheck(page)
	if detectionResult.HasError {
		stealth.PrintDetectionStatus(detectionResult)
		return detectionResult.Error
	}

	// Handle the connection modal
	if note != "" {
		// Truncate note if too long
		if len(note) > MaxNoteLength {
			note = note[:MaxNoteLength-3] + "..."
			fmt.Printf("⚠️ Note truncated to %d characters\n", MaxNoteLength)
		}

		// Click "Add a note" button if present
		err := clickAddNote(page)
		if err != nil {
			fmt.Println("⚠️ Could not add note, sending without note")
		} else {
			// Type the note
			err = typeNote(page, note)
			if err != nil {
				return fmt.Errorf("failed to type note: %w", err)
			}
		}
	}

	// Click Send button
	err := clickSendButton(page)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	fmt.Println("✅ Connection request sent!")
	return nil
}

// clickAddNote clicks the "Add a note" button in the modal
func clickAddNote(page *rod.Page) error {
	result := page.MustEval(`() => {
		const selectors = [
			'button[aria-label="Add a note"]',
			'button:contains("Add a note")',
		];

		// Try selectors
		for (const selector of selectors) {
			try {
				const btn = document.querySelector(selector);
				if (btn) {
					btn.click();
					return true;
				}
			} catch (e) {}
		}

		// Try by text
		const buttons = document.querySelectorAll('button');
		for (const btn of buttons) {
			if (btn.innerText.toLowerCase().includes('add a note')) {
				btn.click();
				return true;
			}
		}

		return false;
	}`)

	if !result.Bool() {
		return fmt.Errorf("add note button not found")
	}

	stealth.SleepMillis(400, 700)
	return nil
}

// typeNote types the personalized note
func typeNote(page *rod.Page, note string) error {
	result := page.MustEval(`(note) => {
		const selectors = [
			'textarea[name="message"]',
			'textarea#custom-message',
			'textarea.connect-button-send-invite__custom-message',
			'textarea[placeholder*="personalize"]',
			'div[role="dialog"] textarea',
		];

		for (const selector of selectors) {
			const textarea = document.querySelector(selector);
			if (textarea) {
				textarea.focus();
				textarea.value = note;
				textarea.dispatchEvent(new Event('input', { bubbles: true }));
				return true;
			}
		}

		return false;
	}`, note)

	if !result.Bool() {
		return fmt.Errorf("note textarea not found")
	}

	return nil
}

// clickSendButton clicks the Send/Connect button in the modal
func clickSendButton(page *rod.Page) error {
	stealth.SleepMillis(400, 700)

	result := page.MustEval(`() => {
		const selectors = [
			'button[aria-label="Send now"]',
			'button[aria-label="Send invitation"]',
			'button.artdeco-button--primary[type="submit"]',
		];

		// Try selectors
		for (const selector of selectors) {
			try {
				const btn = document.querySelector(selector);
				if (btn && !btn.disabled) {
					btn.click();
					return { clicked: true, error: null };
				}
			} catch (e) {}
		}

		// Try by text content in modal
		const modal = document.querySelector('div[role="dialog"]');
		if (modal) {
			const buttons = modal.querySelectorAll('button');
			for (const btn of buttons) {
				const text = btn.innerText.toLowerCase();
				if ((text.includes('send') || text === 'connect') && !btn.disabled) {
					btn.click();
					return { clicked: true, error: null };
				}
			}
		}

		// Fallback to any send button
		const allButtons = document.querySelectorAll('button');
		for (const btn of allButtons) {
			const text = btn.innerText.toLowerCase();
			if (text === 'send' || text === 'send invitation' || text === 'send now') {
				btn.click();
				return { clicked: true, error: null };
			}
		}

		return { clicked: false, error: 'send_button_not_found' };
	}`)

	if !result.Get("clicked").Bool() {
		return fmt.Errorf("send button not found or disabled")
	}

	stealth.SleepMillis(800, 1500)
	return nil

}

// ConnectWithTracking sends a connection request and tracks it
func ConnectWithTracking(page *rod.Page, profileURL string, personName string, note string, tracker *ConnectionTracker) error {
	// Check daily limit
	if !tracker.CanSendMore() {
		return fmt.Errorf("daily limit reached (%d requests). Try again tomorrow", tracker.DailyLimit)
	}

	// Check if already sent
	if tracker.AlreadySent(profileURL) {
		return fmt.Errorf("connection request already sent to this profile")
	}

	// Navigate to profile
	err := NavigateToProfile(page, profileURL)
	if err != nil {
		return err
	}

	// DRY RUN MODE - just log what would happen
	if tracker.DryRun {
		fmt.Println("🧪 [DRY RUN] Would send connection request")
		fmt.Printf("   📍 Profile: %s\n", profileURL)
		fmt.Printf("   👤 Name: %s\n", personName)
		if note != "" {
			fmt.Printf("   📝 Note (%d chars): %s\n", len(note), note)
		} else {
			fmt.Println("   📝 Note: (none)")
		}
		fmt.Println("✅ [DRY RUN] Connection request simulated successfully!")
	} else {
		// Send request (actual mode)
		err = SendConnectionRequest(page, note)
		if err != nil {
			return err
		}
	}

	// Track the request
	request := ConnectionRequest{
		ProfileURL: profileURL,
		Name:       personName,
		Note:       note,
		SentAt:     time.Now(),
		Status:     "sent",
	}

	// In dry run mode, don't actually save
	if tracker.DryRun {
		fmt.Println("🧪 [DRY RUN] Would track request (not saving)")
	} else {
		tracker.AddRequest(request)
		// Save tracker
		if err := tracker.Save(); err != nil {
			fmt.Printf("⚠️ Failed to save tracker: %v\n", err)
		}
	}

	remaining := tracker.RemainingToday()
	fmt.Printf("📊 Requests sent today: %d/%d (remaining: %d)\n",
		tracker.GetTodayCount(), tracker.DailyLimit, remaining)

	return nil
}

// BatchConnect sends connection requests to multiple profiles
func BatchConnect(page *rod.Page, profiles []string, noteTemplate string, tracker *ConnectionTracker, delaySeconds int) (int, int, error) {
	successCount := 0
	failCount := 0

	for i, profileURL := range profiles {
		if !tracker.CanSendMore() {
			fmt.Printf("⚠️ Daily limit reached after %d requests\n", successCount)
			break
		}

		if tracker.AlreadySent(profileURL) {
			fmt.Printf("⏭️ Skipping %s (already sent)\n", profileURL)
			continue
		}

		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(profiles), profileURL)

		err := ConnectWithTracking(page, profileURL, "", noteTemplate, tracker)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			failCount++
		} else {
			successCount++
		}

		// Randomized delay between requests (human-like)
		if i < len(profiles)-1 && tracker.CanSendMore() {
			stealth.Sleep(delaySeconds-2, delaySeconds+5) // Vary around the base delay
		}
	}

	return successCount, failCount, nil
}

// GeneratePersonalizedNote generates a personalized note from a template
// Supported placeholders: {name}, {company}, {title}
func GeneratePersonalizedNote(template string, name string, company string, title string) string {
	note := template
	note = strings.ReplaceAll(note, "{name}", name)
	note = strings.ReplaceAll(note, "{company}", company)
	note = strings.ReplaceAll(note, "{title}", title)

	// Truncate if needed
	if len(note) > MaxNoteLength {
		note = note[:MaxNoteLength-3] + "..."
	}

	return note
}

// SetDailyLimit updates the daily limit
func (t *ConnectionTracker) SetDailyLimit(limit int) {
	if limit > 0 {
		t.DailyLimit = limit
	}
}

// SetDryRun enables or disables dry run mode
func (t *ConnectionTracker) SetDryRun(enabled bool) {
	t.DryRun = enabled
	if enabled {
		fmt.Println("🧪 DRY RUN MODE ENABLED - No actual connection requests will be sent")
	}
}

// GetStats returns connection statistics
func (t *ConnectionTracker) GetStats() map[string]int {
	stats := map[string]int{
		"total":       len(t.Requests),
		"today":       t.GetTodayCount(),
		"remaining":   t.RemainingToday(),
		"daily_limit": t.DailyLimit,
	}

	// Count by status
	for _, req := range t.Requests {
		stats[req.Status]++
	}

	return stats
}
