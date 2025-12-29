package persistence

import (
	"database/sql"
	"fmt"
	"time"
)

// Message represents a sent message
type Message struct {
	ID             int64      `json:"id"`
	ConversationID string     `json:"conversation_id,omitempty"`
	RecipientURL   string     `json:"recipient_url"`
	RecipientName  string     `json:"recipient_name,omitempty"`
	Content        string     `json:"content"`
	TemplateName   string     `json:"template_name,omitempty"`
	MessageType    string     `json:"message_type,omitempty"` // "initial", "follow_up", "reply"
	Status         string     `json:"status"`                 // "sent", "delivered", "read", "failed"
	SentAt         time.Time  `json:"sent_at"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
}

// MessageType constants
const (
	MessageTypeInitial  = "initial"
	MessageTypeFollowUp = "follow_up"
	MessageTypeReply    = "reply"
)

// MessageStatus constants
const (
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusRead      = "read"
	MessageStatusFailed    = "failed"
)

// SaveMessage saves a new message
func (s *Store) SaveMessage(msg *Message) error {
	if msg.Status == "" {
		msg.Status = MessageStatusSent
	}
	if msg.SentAt.IsZero() {
		msg.SentAt = time.Now()
	}

	result, err := s.db.Exec(`
		INSERT INTO messages (
			conversation_id, recipient_url, recipient_name, content,
			template_name, message_type, status, sent_at, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ConversationID, msg.RecipientURL, msg.RecipientName, msg.Content,
		msg.TemplateName, msg.MessageType, msg.Status, msg.SentAt, msg.ErrorMessage)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	id, _ := result.LastInsertId()
	msg.ID = id

	// Update daily stats
	s.incrementDailyStat("messages_sent")

	// Update connection's message status
	s.updateConnectionMessageStatus(msg.RecipientURL)

	return nil
}

// updateConnectionMessageStatus updates the connection record when a message is sent
func (s *Store) updateConnectionMessageStatus(profileURL string) {
	s.db.Exec(`
		UPDATE connections 
		SET has_messaged = TRUE, 
			last_message_at = CURRENT_TIMESTAMP,
			message_count = message_count + 1
		WHERE profile_url = ?
	`, profileURL)
}

// GetMessagesByRecipient returns all messages sent to a specific recipient
func (s *Store) GetMessagesByRecipient(profileURL string) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT id, conversation_id, recipient_url, recipient_name, content,
			   template_name, message_type, status, sent_at, delivered_at,
			   read_at, error_message
		FROM messages
		WHERE recipient_url = ?
		ORDER BY sent_at DESC
	`, profileURL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetRecentMessages returns recent messages with optional limit
func (s *Store) GetRecentMessages(limit int) ([]Message, error) {
	query := `
		SELECT id, conversation_id, recipient_url, recipient_name, content,
			   template_name, message_type, status, sent_at, delivered_at,
			   read_at, error_message
		FROM messages
		ORDER BY sent_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetTodayMessageCount returns the number of messages sent today
func (s *Store) GetTodayMessageCount() (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE date(sent_at) = date('now') AND status != ?
	`, MessageStatusFailed).Scan(&count)
	return count, err
}

// HasMessaged checks if we've already messaged this person
func (s *Store) HasMessaged(profileURL string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE recipient_url = ? AND status != ?
	`, profileURL, MessageStatusFailed).Scan(&count)
	return count > 0, err
}

// GetLastMessageTo returns the last message sent to a recipient
func (s *Store) GetLastMessageTo(profileURL string) (*Message, error) {
	row := s.db.QueryRow(`
		SELECT id, conversation_id, recipient_url, recipient_name, content,
			   template_name, message_type, status, sent_at, delivered_at,
			   read_at, error_message
		FROM messages
		WHERE recipient_url = ?
		ORDER BY sent_at DESC
		LIMIT 1
	`, profileURL)

	return scanMessage(row)
}

// UpdateMessageStatus updates the status of a message
func (s *Store) UpdateMessageStatus(messageID int64, status string) error {
	var query string
	switch status {
	case MessageStatusDelivered:
		query = `UPDATE messages SET status = ?, delivered_at = CURRENT_TIMESTAMP WHERE id = ?`
	case MessageStatusRead:
		query = `UPDATE messages SET status = ?, read_at = CURRENT_TIMESTAMP WHERE id = ?`
	default:
		query = `UPDATE messages SET status = ? WHERE id = ?`
	}

	_, err := s.db.Exec(query, status, messageID)
	return err
}

// MessageStats holds messaging statistics
type MessageStats struct {
	TotalSent      int
	SentToday      int
	InitialSent    int
	FollowUpsSent  int
	FailedMessages int
	DailyLimit     int
	RemainingToday int
}

// GetMessageStats returns messaging statistics
func (s *Store) GetMessageStats(dailyLimit int) (*MessageStats, error) {
	stats := &MessageStats{DailyLimit: dailyLimit}

	// Get total stats
	row := s.db.QueryRow(`
		SELECT 
			COALESCE(SUM(CASE WHEN status != ? THEN 1 ELSE 0 END), 0) as total_sent,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN message_type = ? AND status != ? THEN 1 ELSE 0 END), 0) as initial,
			COALESCE(SUM(CASE WHEN message_type = ? AND status != ? THEN 1 ELSE 0 END), 0) as follow_up
		FROM messages
	`, MessageStatusFailed, MessageStatusFailed,
		MessageTypeInitial, MessageStatusFailed,
		MessageTypeFollowUp, MessageStatusFailed)

	err := row.Scan(&stats.TotalSent, &stats.FailedMessages, &stats.InitialSent, &stats.FollowUpsSent)
	if err != nil {
		return nil, err
	}

	stats.SentToday, _ = s.GetTodayMessageCount()
	stats.RemainingToday = dailyLimit - stats.SentToday
	if stats.RemainingToday < 0 {
		stats.RemainingToday = 0
	}

	return stats, nil
}

// GetUnmessagedConnections returns connections we haven't messaged yet
func (s *Store) GetUnmessagedConnections() ([]Connection, error) {
	rows, err := s.db.Query(`
		SELECT id, profile_url, name, headline, company, connected_at,
			   has_messaged, last_message_at, message_count, notes
		FROM connections
		WHERE has_messaged = FALSE
		ORDER BY connected_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConnections(rows)
}

// Connection represents an accepted LinkedIn connection
type Connection struct {
	ID            int64      `json:"id"`
	ProfileURL    string     `json:"profile_url"`
	Name          string     `json:"name"`
	Headline      string     `json:"headline,omitempty"`
	Company       string     `json:"company,omitempty"`
	ConnectedAt   time.Time  `json:"connected_at"`
	HasMessaged   bool       `json:"has_messaged"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	MessageCount  int        `json:"message_count"`
	Notes         string     `json:"notes,omitempty"`
}

// SaveConnection saves or updates a connection
func (s *Store) SaveConnection(conn *Connection) error {
	if conn.ConnectedAt.IsZero() {
		conn.ConnectedAt = time.Now()
	}

	result, err := s.db.Exec(`
		INSERT INTO connections (
			profile_url, name, headline, company, connected_at,
			has_messaged, last_message_at, message_count, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(profile_url) DO UPDATE SET
			name = COALESCE(excluded.name, connections.name),
			headline = COALESCE(excluded.headline, connections.headline),
			company = COALESCE(excluded.company, connections.company),
			has_messaged = excluded.has_messaged,
			last_message_at = COALESCE(excluded.last_message_at, connections.last_message_at),
			message_count = excluded.message_count,
			notes = COALESCE(excluded.notes, connections.notes)
	`, conn.ProfileURL, conn.Name, conn.Headline, conn.Company,
		conn.ConnectedAt, conn.HasMessaged, conn.LastMessageAt,
		conn.MessageCount, conn.Notes)

	if err != nil {
		return fmt.Errorf("failed to save connection: %w", err)
	}

	if conn.ID == 0 {
		id, _ := result.LastInsertId()
		conn.ID = id
	}

	return nil
}

// GetConnection retrieves a connection by profile URL
func (s *Store) GetConnection(profileURL string) (*Connection, error) {
	row := s.db.QueryRow(`
		SELECT id, profile_url, name, headline, company, connected_at,
			   has_messaged, last_message_at, message_count, notes
		FROM connections
		WHERE profile_url = ?
	`, profileURL)

	return scanConnection(row)
}

// GetAllConnections returns all connections
func (s *Store) GetAllConnections(limit, offset int) ([]Connection, error) {
	query := `
		SELECT id, profile_url, name, headline, company, connected_at,
			   has_messaged, last_message_at, message_count, notes
		FROM connections
		ORDER BY connected_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConnections(rows)
}

func scanMessages(rows *sql.Rows) ([]Message, error) {
	var messages []Message

	for rows.Next() {
		var msg Message
		var conversationID, recipientName, templateName, messageType, errorMessage sql.NullString
		var deliveredAt, readAt sql.NullTime

		err := rows.Scan(
			&msg.ID, &conversationID, &msg.RecipientURL, &recipientName,
			&msg.Content, &templateName, &messageType, &msg.Status,
			&msg.SentAt, &deliveredAt, &readAt, &errorMessage,
		)
		if err != nil {
			return nil, err
		}

		if conversationID.Valid {
			msg.ConversationID = conversationID.String
		}
		if recipientName.Valid {
			msg.RecipientName = recipientName.String
		}
		if templateName.Valid {
			msg.TemplateName = templateName.String
		}
		if messageType.Valid {
			msg.MessageType = messageType.String
		}
		if errorMessage.Valid {
			msg.ErrorMessage = errorMessage.String
		}
		if deliveredAt.Valid {
			msg.DeliveredAt = &deliveredAt.Time
		}
		if readAt.Valid {
			msg.ReadAt = &readAt.Time
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func scanMessage(row *sql.Row) (*Message, error) {
	msg := &Message{}
	var conversationID, recipientName, templateName, messageType, errorMessage sql.NullString
	var deliveredAt, readAt sql.NullTime

	err := row.Scan(
		&msg.ID, &conversationID, &msg.RecipientURL, &recipientName,
		&msg.Content, &templateName, &messageType, &msg.Status,
		&msg.SentAt, &deliveredAt, &readAt, &errorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if conversationID.Valid {
		msg.ConversationID = conversationID.String
	}
	if recipientName.Valid {
		msg.RecipientName = recipientName.String
	}
	if templateName.Valid {
		msg.TemplateName = templateName.String
	}
	if messageType.Valid {
		msg.MessageType = messageType.String
	}
	if errorMessage.Valid {
		msg.ErrorMessage = errorMessage.String
	}
	if deliveredAt.Valid {
		msg.DeliveredAt = &deliveredAt.Time
	}
	if readAt.Valid {
		msg.ReadAt = &readAt.Time
	}

	return msg, nil
}

func scanConnections(rows *sql.Rows) ([]Connection, error) {
	var connections []Connection

	for rows.Next() {
		var conn Connection
		var headline, company, notes sql.NullString
		var lastMessageAt sql.NullTime

		err := rows.Scan(
			&conn.ID, &conn.ProfileURL, &conn.Name, &headline, &company,
			&conn.ConnectedAt, &conn.HasMessaged, &lastMessageAt,
			&conn.MessageCount, &notes,
		)
		if err != nil {
			return nil, err
		}

		if headline.Valid {
			conn.Headline = headline.String
		}
		if company.Valid {
			conn.Company = company.String
		}
		if notes.Valid {
			conn.Notes = notes.String
		}
		if lastMessageAt.Valid {
			conn.LastMessageAt = &lastMessageAt.Time
		}

		connections = append(connections, conn)
	}

	return connections, rows.Err()
}

func scanConnection(row *sql.Row) (*Connection, error) {
	conn := &Connection{}
	var headline, company, notes sql.NullString
	var lastMessageAt sql.NullTime

	err := row.Scan(
		&conn.ID, &conn.ProfileURL, &conn.Name, &headline, &company,
		&conn.ConnectedAt, &conn.HasMessaged, &lastMessageAt,
		&conn.MessageCount, &notes,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if headline.Valid {
		conn.Headline = headline.String
	}
	if company.Valid {
		conn.Company = company.String
	}
	if notes.Valid {
		conn.Notes = notes.String
	}
	if lastMessageAt.Valid {
		conn.LastMessageAt = &lastMessageAt.Time
	}

	return conn, nil
}
