package message

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"

	"github.com/Nehilsa2/linkedin_automation/stealth"
)

// SendMessage sends a message to a profile (must be on their profile page or in messaging)
func SendMessage(page *rod.Page, content string, dryRun bool) error {
	fmt.Println("💬 Attempting to send message...")

	if dryRun {
		fmt.Println("🧪 [DRY RUN] Would send message:")
		fmt.Printf("   📝 Content (%d chars): %s\n", len(content), truncateString(content, 100))
		fmt.Println("✅ [DRY RUN] Message simulated successfully!")
		return nil
	}

	// Set timeout
	timeoutPage := page.Timeout(15 * time.Second)
	defer timeoutPage.CancelTimeout()

	// Try to find and click the Message button on profile
	result := timeoutPage.MustEval(`() => {
		// Find Message button on profile
		const messageSelectors = [
			'button[aria-label*="Message"]',
			'button.pvs-profile-actions__action[aria-label*="Message"]',
			'a[href*="/messaging/"]',
		];

		for (const selector of messageSelectors) {
			const btn = document.querySelector(selector);
			if (btn && !btn.disabled) {
				btn.scrollIntoView({ block: "center" });
				btn.click();
				return { found: true, clicked: true };
			}
		}

		// Try finding by text
		const buttons = document.querySelectorAll('button');
		for (const btn of buttons) {
			if (btn.innerText.trim().toLowerCase() === 'message') {
				btn.scrollIntoView({ block: "center" });
				btn.click();
				return { found: true, clicked: true };
			}
		}

		return { found: false, clicked: false };
	}`)

	if !result.Get("found").Bool() {
		return fmt.Errorf("message button not found on profile")
	}

	// Wait for message modal/conversation to open
	stealth.Sleep(1, 3)

	// Check for any errors after opening message modal
	detectionResult := stealth.CheckPage(timeoutPage)
	if detectionResult.HasError {
		stealth.PrintDetectionStatus(detectionResult)
		return detectionResult.Error
	}

	// Type the message
	err := typeMessage(timeoutPage, content)
	if err != nil {
		return fmt.Errorf("failed to type message: %w", err)
	}

	// Send the message
	err = clickSendMessage(timeoutPage)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Println("✅ Message sent!")
	return nil
}

// typeMessage types content into the message input using human-like typing
//
// WHY HUMAN-LIKE TYPING MATTERS:
// - LinkedIn monitors keystroke patterns and timing
// - Instant text paste (innerHTML = "...") has ZERO keystroke events
// - This is an obvious bot signal: "200 characters appeared in 0ms"
// - Human typing generates keydown/keypress/input/keyup events
// - Natural timing varies: faster for common letters, slower for symbols
func typeMessage(page *rod.Page, content string) error {
	// First, find and focus the message input
	result := page.MustEval(`() => {
		const inputSelectors = [
			'div[role="textbox"][contenteditable="true"]',
			'div.msg-form__contenteditable',
			'textarea.msg-form__textarea',
			'div[data-placeholder*="Write a message"]',
		];

		for (const selector of inputSelectors) {
			const input = document.querySelector(selector);
			if (input) {
				input.focus();
				// Clear existing content
				if (input.tagName === 'TEXTAREA') {
					input.value = '';
				} else {
					input.innerHTML = '';
				}
				return { found: true, selector: selector };
			}
		}

		return { found: false };
	}`)

	if !result.Get("found").Bool() {
		return fmt.Errorf("message input not found")
	}

	stealth.SleepMillis(200, 400)

	// Type the message character by character with human-like timing
	fmt.Printf("⌨️ Typing message (%d chars)...\n", len(content))
	err := stealth.TypeTextJS(page, content, stealth.DefaultTypingConfig())
	if err != nil {
		return fmt.Errorf("failed to type message: %w", err)
	}

	stealth.SleepMillis(400, 700)
	return nil
}

// clickSendMessage clicks the send button
func clickSendMessage(page *rod.Page) error {
	stealth.SleepMillis(400, 700)

	result := page.MustEval(`() => {
		const sendSelectors = [
			'button[type="submit"].msg-form__send-button',
			'button.msg-form__send-button',
			'button[aria-label="Send"]',
			'button.msg-form__send-btn',
		];

		for (const selector of sendSelectors) {
			const btn = document.querySelector(selector);
			if (btn && !btn.disabled) {
				btn.click();
				return true;
			}
		}

		// Fallback: find by text
		const buttons = document.querySelectorAll('button');
		for (const btn of buttons) {
			const text = btn.innerText.toLowerCase().trim();
			if (text === 'send' && !btn.disabled) {
				btn.click();
				return true;
			}
		}

		return false;
	}`)

	if !result.Bool() {
		return fmt.Errorf("send button not found or disabled")
	}

	stealth.SleepMillis(800, 1500)
	return nil
}

// SendFollowUpMessage navigates to profile and sends a follow-up message
func SendFollowUpMessage(page *rod.Page, conn Connection, content string, tracker *Tracker) error {
	fmt.Printf("📨 Sending follow-up to: %s\n", conn.Name)

	// Check daily limit
	if !tracker.CanSendMore() {
		return fmt.Errorf("daily message limit reached")
	}

	// Check if already messaged
	if tracker.HasMessaged(conn.ProfileURL) {
		return fmt.Errorf("already messaged this connection")
	}

	// Navigate to profile
	fmt.Printf("📍 Navigating to: %s\n", conn.ProfileURL)
	timeoutPage := page.Timeout(15 * time.Second)

	err := timeoutPage.Navigate(conn.ProfileURL)
	if err != nil {
		timeoutPage.CancelTimeout()
		return fmt.Errorf("failed to navigate: %w", err)
	}

	err = timeoutPage.WaitStable(time.Second)
	timeoutPage.CancelTimeout()
	if err != nil {
		fmt.Println("⚠️ Page stability wait timed out, continuing...")
	}

	stealth.Sleep(1, 3)

	// Send the message
	err = SendMessage(page, content, tracker.DryRun)
	if err != nil {
		return err
	}

	// Track the message
	msg := Message{
		RecipientURL:  conn.ProfileURL,
		RecipientName: conn.Name,
		Content:       content,
		SentAt:        time.Now(),
		Status:        "sent",
		MessageType:   "follow_up",
	}

	if !tracker.DryRun {
		tracker.AddMessage(msg)
		tracker.MarkConnectionMessaged(conn.ProfileURL)
		if err := tracker.Save(); err != nil {
			fmt.Printf("⚠️ Failed to save tracker: %v\n", err)
		}
	} else {
		fmt.Println("🧪 [DRY RUN] Would track message (not saving)")
	}

	return nil
}

// SendTemplatedFollowUp sends a follow-up using a template
func SendTemplatedFollowUp(page *rod.Page, conn Connection, templateName string, templates *TemplateManager, tracker *Tracker) error {
	// Build variables map
	vars := map[string]string{
		"{name}":     conn.Name,
		"{company}":  conn.Company,
		"{headline}": conn.Headline,
	}

	// Extract first name
	nameParts := splitName(conn.Name)
	vars["{first_name}"] = nameParts[0]
	if len(nameParts) > 1 {
		vars["{last_name}"] = nameParts[len(nameParts)-1]
	}

	// Render template
	content, err := templates.RenderTemplate(templateName, vars)
	if err != nil {
		return err
	}

	fmt.Printf("📝 Using template: %s\n", templateName)
	return SendFollowUpMessage(page, conn, content, tracker)
}

// BatchFollowUp sends follow-up messages to multiple connections
func BatchFollowUp(page *rod.Page, connections []Connection, templateName string, templates *TemplateManager, tracker *Tracker, delayMinSec, delayMaxSec int) (int, int, error) {
	successCount := 0
	failCount := 0

	// Get rate limiter for messaging
	rateLimiter := stealth.GetRateLimiter()
	rateLimiter.PrintStats(stealth.ActionMessage)

	for i, conn := range connections {
		// Check rate limits first
		if can, reason := rateLimiter.CanPerform(stealth.ActionMessage); !can {
			fmt.Printf("⏸️ Rate limited: %s\n", reason)
			if !rateLimiter.WaitForAction(stealth.ActionMessage) {
				fmt.Println("⏰ Rate limit wait too long - stopping batch")
				break
			}
		}

		if !tracker.CanSendMore() {
			fmt.Printf("⚠️ Daily limit reached after %d messages\n", successCount)
			break
		}

		if tracker.HasMessaged(conn.ProfileURL) {
			fmt.Printf("⏭️ Skipping %s (already messaged)\n", conn.Name)
			continue
		}

		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(connections), conn.Name)

		err := SendTemplatedFollowUp(page, conn, templateName, templates, tracker)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			failCount++
		} else {
			successCount++
			// Record action for rate limiting
			rateLimiter.RecordAction(stealth.ActionMessage)
		}

		// Use rate limiter's recommended delay
		if i < len(connections)-1 && tracker.CanSendMore() {
			delay := rateLimiter.GetRecommendedDelay(stealth.ActionMessage)
			fmt.Printf("⏳ Waiting %v before next message...\n", delay.Round(time.Second))
			time.Sleep(delay)
		}
	}

	// Print final stats
	rateLimiter.PrintStats(stealth.ActionMessage)

	return successCount, failCount, nil
}

// splitName splits a full name into parts
func splitName(name string) []string {
	var parts []string
	for _, p := range splitString(name, " ") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return []string{name}
	}
	return parts
}

// splitString is a simple string splitter
func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

// truncateString truncates a string with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
