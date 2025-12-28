package message

import "time"

// Message represents a sent message
type Message struct {
	ConversationID string    `json:"conversation_id"`
	RecipientURL   string    `json:"recipient_url"`
	RecipientName  string    `json:"recipient_name"`
	Content        string    `json:"content"`
	TemplateName   string    `json:"template_name,omitempty"`
	SentAt         time.Time `json:"sent_at"`
	Status         string    `json:"status"`       // "sent", "delivered", "read", "failed"
	MessageType    string    `json:"message_type"` // "follow_up", "initial", "reply"
}

// Connection represents a LinkedIn connection
type Connection struct {
	ProfileURL    string    `json:"profile_url"`
	Name          string    `json:"name"`
	Headline      string    `json:"headline,omitempty"`
	Company       string    `json:"company,omitempty"`
	ConnectedAt   time.Time `json:"connected_at"`
	HasMessaged   bool      `json:"has_messaged"`
	LastMessageAt time.Time `json:"last_message_at,omitempty"`
}

// Template represents a message template
type Template struct {
	Name        string   `json:"name"`
	Content     string   `json:"content"`
	Description string   `json:"description,omitempty"`
	Variables   []string `json:"variables"` // List of supported variables
}

// MessageStats holds messaging statistics
type MessageStats struct {
	TotalSent     int `json:"total_sent"`
	SentToday     int `json:"sent_today"`
	FollowUpsSent int `json:"follow_ups_sent"`
	DailyLimit    int `json:"daily_limit"`
	Remaining     int `json:"remaining"`
}
