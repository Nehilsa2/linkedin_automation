package message

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	TrackerFile       = "message_tracker.json"
	DefaultDailyLimit = 100
)

// Tracker tracks sent messages and connections
type Tracker struct {
	Messages    []Message    `json:"messages"`
	Connections []Connection `json:"connections"`
	DailyLimit  int          `json:"daily_limit"`
	DryRun      bool         `json:"-"` // Don't persist
}

// LoadTracker loads the tracker from file
func LoadTracker() (*Tracker, error) {
	tracker := &Tracker{
		Messages:    []Message{},
		Connections: []Connection{},
		DailyLimit:  DefaultDailyLimit,
	}

	data, err := os.ReadFile(TrackerFile)
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
func (t *Tracker) Save() error {
	if t.DryRun {
		fmt.Println("🧪 [DRY RUN] Would save tracker (skipping)")
		return nil
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(TrackerFile, data, 0644)
}

// SetDryRun enables or disables dry run mode
func (t *Tracker) SetDryRun(enabled bool) {
	t.DryRun = enabled
	if enabled {
		fmt.Println("🧪 DRY RUN MODE ENABLED - No actual messages will be sent")
	}
}

// SetDailyLimit updates the daily message limit
func (t *Tracker) SetDailyLimit(limit int) {
	if limit > 0 {
		t.DailyLimit = limit
	}
}

// GetTodayMessageCount returns messages sent today
func (t *Tracker) GetTodayMessageCount() int {
	today := time.Now().Truncate(24 * time.Hour)
	count := 0
	for _, msg := range t.Messages {
		if msg.SentAt.After(today) {
			count++
		}
	}
	return count
}

// CanSendMore checks if more messages can be sent today
func (t *Tracker) CanSendMore() bool {
	return t.GetTodayMessageCount() < t.DailyLimit
}

// RemainingToday returns remaining message quota
func (t *Tracker) RemainingToday() int {
	return t.DailyLimit - t.GetTodayMessageCount()
}

// AddMessage adds a message to the tracker
func (t *Tracker) AddMessage(msg Message) {
	t.Messages = append(t.Messages, msg)
}

// AddConnection adds or updates a connection
func (t *Tracker) AddConnection(conn Connection) {
	// Check if already exists
	for i, existing := range t.Connections {
		if normalizeURL(existing.ProfileURL) == normalizeURL(conn.ProfileURL) {
			t.Connections[i] = conn
			return
		}
	}
	t.Connections = append(t.Connections, conn)
}

// GetConnection retrieves a connection by profile URL
func (t *Tracker) GetConnection(profileURL string) *Connection {
	normalized := normalizeURL(profileURL)
	for i, conn := range t.Connections {
		if normalizeURL(conn.ProfileURL) == normalized {
			return &t.Connections[i]
		}
	}
	return nil
}

// HasMessaged checks if we've already messaged this person
func (t *Tracker) HasMessaged(profileURL string) bool {
	normalized := normalizeURL(profileURL)
	for _, msg := range t.Messages {
		if normalizeURL(msg.RecipientURL) == normalized {
			return true
		}
	}
	return false
}

// GetUnmessagedConnections returns connections we haven't messaged yet
func (t *Tracker) GetUnmessagedConnections() []Connection {
	var unmessaged []Connection
	for _, conn := range t.Connections {
		if !t.HasMessaged(conn.ProfileURL) {
			unmessaged = append(unmessaged, conn)
		}
	}
	return unmessaged
}

// MarkConnectionMessaged marks a connection as having been messaged
func (t *Tracker) MarkConnectionMessaged(profileURL string) {
	normalized := normalizeURL(profileURL)
	for i, conn := range t.Connections {
		if normalizeURL(conn.ProfileURL) == normalized {
			t.Connections[i].HasMessaged = true
			t.Connections[i].LastMessageAt = time.Now()
			return
		}
	}
}

// GetStats returns messaging statistics
func (t *Tracker) GetStats() MessageStats {
	followUps := 0
	for _, msg := range t.Messages {
		if msg.MessageType == "follow_up" {
			followUps++
		}
	}

	return MessageStats{
		TotalSent:     len(t.Messages),
		SentToday:     t.GetTodayMessageCount(),
		FollowUpsSent: followUps,
		DailyLimit:    t.DailyLimit,
		Remaining:     t.RemainingToday(),
	}
}

// normalizeURL normalizes LinkedIn URLs for comparison
func normalizeURL(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	return strings.ToLower(url)
}
