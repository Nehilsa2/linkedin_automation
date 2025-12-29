package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// MigrateFromJSON migrates existing JSON tracker files to the unified store
func (s *Store) MigrateFromJSON() error {
	fmt.Println("📦 Checking for existing JSON data to migrate...")

	migrated := 0

	// Migrate connection requests
	if count, err := s.migrateConnectionRequests(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to migrate connection requests: %v\n", err)
	} else if count > 0 {
		fmt.Printf("✅ Migrated %d connection requests\n", count)
		migrated += count
	}

	// Migrate message tracker
	if count, err := s.migrateMessageTracker(); err != nil {
		fmt.Printf("⚠️ Warning: Failed to migrate message tracker: %v\n", err)
	} else if count > 0 {
		fmt.Printf("✅ Migrated %d messages and connections\n", count)
		migrated += count
	}

	if migrated > 0 {
		fmt.Printf("📦 Migration complete: %d total records migrated\n", migrated)
		// Save after migration
		s.Save()
	} else {
		fmt.Println("ℹ️ No existing JSON data found to migrate")
	}

	return nil
}

// migrateConnectionRequests migrates from connection_requests.json
func (s *Store) migrateConnectionRequests() (int, error) {
	const jsonFile = "connection_requests.json"

	data, err := os.ReadFile(jsonFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	type OldConnectionRequest struct {
		ProfileURL string    `json:"profile_url"`
		Name       string    `json:"name"`
		Note       string    `json:"note,omitempty"`
		SentAt     time.Time `json:"sent_at"`
		Status     string    `json:"status"`
	}

	type OldTracker struct {
		Requests   []OldConnectionRequest `json:"requests"`
		DailyLimit int                    `json:"daily_limit"`
	}

	var oldTracker OldTracker
	if err := json.Unmarshal(data, &oldTracker); err != nil {
		return 0, err
	}

	count := 0
	for _, req := range oldTracker.Requests {
		newReq := &ConnectionRequest{
			ProfileURL: req.ProfileURL,
			Name:       req.Name,
			Note:       req.Note,
			Status:     req.Status,
			SentAt:     req.SentAt,
			Source:     "migrated_json",
		}

		// Map old status to new status
		switch req.Status {
		case "sent", "pending":
			newReq.Status = StatusPending
		case "accepted":
			newReq.Status = StatusAccepted
			now := time.Now()
			newReq.AcceptedAt = &now
		case "declined":
			newReq.Status = StatusDeclined
		default:
			newReq.Status = StatusPending
		}

		if err := s.SaveConnectionRequest(newReq); err != nil {
			fmt.Printf("⚠️ Failed to migrate request for %s: %v\n", req.ProfileURL, err)
			continue
		}
		count++
	}

	// Backup the old file
	if count > 0 {
		backupPath := jsonFile + ".migrated"
		os.Rename(jsonFile, backupPath)
		fmt.Printf("📁 Backed up %s to %s\n", jsonFile, backupPath)
	}

	return count, nil
}

// migrateMessageTracker migrates from message_tracker.json
func (s *Store) migrateMessageTracker() (int, error) {
	const jsonFile = "message_tracker.json"

	data, err := os.ReadFile(jsonFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	type OldMessage struct {
		ConversationID string    `json:"conversation_id"`
		RecipientURL   string    `json:"recipient_url"`
		RecipientName  string    `json:"recipient_name"`
		Content        string    `json:"content"`
		TemplateName   string    `json:"template_name,omitempty"`
		SentAt         time.Time `json:"sent_at"`
		Status         string    `json:"status"`
		MessageType    string    `json:"message_type"`
	}

	type OldConnection struct {
		ProfileURL    string    `json:"profile_url"`
		Name          string    `json:"name"`
		Headline      string    `json:"headline,omitempty"`
		Company       string    `json:"company,omitempty"`
		ConnectedAt   time.Time `json:"connected_at"`
		HasMessaged   bool      `json:"has_messaged"`
		LastMessageAt time.Time `json:"last_message_at,omitempty"`
	}

	type OldTracker struct {
		Messages    []OldMessage    `json:"messages"`
		Connections []OldConnection `json:"connections"`
		DailyLimit  int             `json:"daily_limit"`
	}

	var oldTracker OldTracker
	if err := json.Unmarshal(data, &oldTracker); err != nil {
		return 0, err
	}

	count := 0

	// Migrate messages
	for _, msg := range oldTracker.Messages {
		newMsg := &Message{
			ConversationID: msg.ConversationID,
			RecipientURL:   msg.RecipientURL,
			RecipientName:  msg.RecipientName,
			Content:        msg.Content,
			TemplateName:   msg.TemplateName,
			MessageType:    msg.MessageType,
			Status:         msg.Status,
			SentAt:         msg.SentAt,
		}

		if err := s.SaveMessage(newMsg); err != nil {
			fmt.Printf("⚠️ Failed to migrate message to %s: %v\n", msg.RecipientURL, err)
			continue
		}
		count++
	}

	// Migrate connections
	for _, conn := range oldTracker.Connections {
		var lastMsgAt *time.Time
		if !conn.LastMessageAt.IsZero() {
			lastMsgAt = &conn.LastMessageAt
		}

		newConn := &Connection{
			ProfileURL:    conn.ProfileURL,
			Name:          conn.Name,
			Headline:      conn.Headline,
			Company:       conn.Company,
			ConnectedAt:   conn.ConnectedAt,
			HasMessaged:   conn.HasMessaged,
			LastMessageAt: lastMsgAt,
		}

		if err := s.SaveConnection(newConn); err != nil {
			fmt.Printf("⚠️ Failed to migrate connection %s: %v\n", conn.ProfileURL, err)
			continue
		}
		count++
	}

	// Backup the old file
	if count > 0 {
		backupPath := jsonFile + ".migrated"
		os.Rename(jsonFile, backupPath)
		fmt.Printf("📁 Backed up %s to %s\n", jsonFile, backupPath)
	}

	return count, nil
}

// ExportToJSON exports current database state to JSON for backup
func (s *Store) ExportToJSON(outputPath string) error {
	// Get all data
	requests, err := s.GetAllConnectionRequests(0, 0)
	if err != nil {
		return fmt.Errorf("failed to get connection requests: %w", err)
	}

	connections, err := s.GetAllConnections(0, 0)
	if err != nil {
		return fmt.Errorf("failed to get connections: %w", err)
	}

	messages, err := s.GetRecentMessages(0)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	export := struct {
		ExportedAt         time.Time           `json:"exported_at"`
		ConnectionRequests []ConnectionRequest `json:"connection_requests"`
		Connections        []Connection        `json:"connections"`
		Messages           []Message           `json:"messages"`
	}{
		ExportedAt:         time.Now(),
		ConnectionRequests: requests,
		Connections:        connections,
		Messages:           messages,
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export: %w", err)
	}

	if outputPath == "" {
		outputPath = fmt.Sprintf("linkedin_export_%s.json", time.Now().Format("20060102_150405"))
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export: %w", err)
	}

	fmt.Printf("📤 Exported data to %s\n", outputPath)
	return nil
}
