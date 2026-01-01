package message

import (
	"fmt"

	"github.com/go-rod/rod"
)

// MessagingService orchestrates all messaging operations
type MessagingService struct {
	Page      *rod.Page
	Tracker   *Tracker
	Templates *TemplateManager
}

// NewMessagingService creates a new messaging service
func NewMessagingService(page *rod.Page) (*MessagingService, error) {
	tracker, err := LoadTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to load tracker: %w", err)
	}

	templates, err := LoadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &MessagingService{
		Page:      page,
		Tracker:   tracker,
		Templates: templates,
	}, nil
}

// SetDryRun enables/disables dry run mode
func (ms *MessagingService) SetDryRun(enabled bool) {
	ms.Tracker.SetDryRun(enabled)
}

// SetDailyLimit sets the daily message limit
func (ms *MessagingService) SetDailyLimit(limit int) {
	ms.Tracker.SetDailyLimit(limit)
}

// SyncConnections detects and syncs new connections
func (ms *MessagingService) SyncConnections(maxToScan int) (int, error) {
	return SyncNewConnections(ms.Page, ms.Tracker, maxToScan)
}

// GetUnmessagedConnections returns connections that haven't been messaged
func (ms *MessagingService) GetUnmessagedConnections() []Connection {
	return ms.Tracker.GetUnmessagedConnections()
}

// GetRecentUnmessaged returns recent connections that haven't been messaged
func (ms *MessagingService) GetRecentUnmessaged(days int) []Connection {
	return GetRecentConnections(ms.Tracker, days)
}

// GetConnectionsDaysAgo returns connections that connected exactly `days` ago
func (ms *MessagingService) GetConnectionsDaysAgo(days int) []Connection {
	return GetConnectionsDaysAgo(ms.Tracker, days)
}

// SendFollowUp sends a follow-up message to a connection
func (ms *MessagingService) SendFollowUp(conn Connection, templateName string) error {
	return SendTemplatedFollowUp(ms.Page, conn, templateName, ms.Templates, ms.Tracker)
}

// SendBatchFollowUps sends follow-up messages to multiple connections
func (ms *MessagingService) SendBatchFollowUps(connections []Connection, templateName string, delayMinSec, delayMaxSec int) (int, int, error) {
	return BatchFollowUp(ms.Page, connections, templateName, ms.Templates, ms.Tracker, delayMinSec, delayMaxSec)
}

// SendCustomMessage sends a custom message to a connection
func (ms *MessagingService) SendCustomMessage(conn Connection, content string) error {
	return SendFollowUpMessage(ms.Page, conn, content, ms.Tracker)
}

// GetStats returns messaging statistics
func (ms *MessagingService) GetStats() MessageStats {
	return ms.Tracker.GetStats()
}

// PrintStats prints current statistics
func (ms *MessagingService) PrintStats() {
	stats := ms.GetStats()
	fmt.Println("\n📊 Messaging Statistics:")
	fmt.Printf("   Total messages sent: %d\n", stats.TotalSent)
	fmt.Printf("   Sent today: %d/%d\n", stats.SentToday, stats.DailyLimit)
	fmt.Printf("   Follow-ups sent: %d\n", stats.FollowUpsSent)
	fmt.Printf("   Remaining today: %d\n", stats.Remaining)
	fmt.Printf("   Tracked connections: %d\n", len(ms.Tracker.Connections))
	fmt.Printf("   Unmessaged connections: %d\n", len(ms.GetUnmessagedConnections()))
}

// ListTemplates prints available templates
func (ms *MessagingService) ListTemplates() {
	ms.Templates.PrintTemplates()
}

// AddCustomTemplate adds a new template
func (ms *MessagingService) AddCustomTemplate(name, description, content string) error {
	template := Template{
		Name:        name,
		Description: description,
		Content:     content,
	}
	return ms.Templates.AddTemplate(template)
}

// AutoFollowUp automatically sends follow-ups to recent unmessaged connections
func (ms *MessagingService) AutoFollowUp(templateName string, maxMessages int, delayMinSec, delayMaxSec int) (int, int, error) {
	// Get unmessaged connections
	unmessaged := ms.GetUnmessagedConnections()

	if len(unmessaged) == 0 {
		fmt.Println("ℹ️ No unmessaged connections found")
		return 0, 0, nil
	}

	// Limit to maxMessages
	if len(unmessaged) > maxMessages {
		unmessaged = unmessaged[:maxMessages]
	}

	fmt.Printf("📨 Starting auto follow-up for %d connections...\n", len(unmessaged))
	return ms.SendBatchFollowUps(unmessaged, templateName, delayMinSec, delayMaxSec)
}

// FullWorkflow runs the complete messaging workflow
// 1. Detect new connections
// 2. Send follow-up messages
func (ms *MessagingService) FullWorkflow(templateName string, maxMessages int, delayMinSec, delayMaxSec int) error {
	fmt.Println("\n🚀 Starting Full Messaging Workflow...")

	// Step 1: Sync new connections
	fmt.Println("\n📡 Step 1: Detecting new connections...")
	newCount, err := ms.SyncConnections(50)
	if err != nil {
		fmt.Printf("⚠️ Error syncing connections: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d new connections\n", newCount)
	}

	// Step 2: Print stats
	ms.PrintStats()

	// Step 3: Send follow-ups
	fmt.Println("\n📨 Step 2: Sending follow-up messages...")
	// Send follow-ups to all unmessaged connections (no days filter)
	targets := ms.GetUnmessagedConnections()
	if len(targets) == 0 {
		fmt.Println("ℹ️ No unmessaged connections found to message")
		return nil
	}

	if len(targets) > maxMessages {
		targets = targets[:maxMessages]
	}

	fmt.Printf("📨 Sending follow-ups to %d connections...\n", len(targets))
	success, failed, err := ms.SendBatchFollowUps(targets, templateName, delayMinSec, delayMaxSec)
	if err != nil {
		return err
	}

	fmt.Printf("\n✅ Workflow Complete: %d sent, %d failed\n", success, failed)
	return nil
}

// Close saves the tracker state
func (ms *MessagingService) Close() error {
	return ms.Tracker.Save()
}
