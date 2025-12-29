package persistence

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ConnectionRequest represents a sent connection request
type ConnectionRequest struct {
	ID            int64      `json:"id"`
	ProfileURL    string     `json:"profile_url"`
	Name          string     `json:"name"`
	Headline      string     `json:"headline,omitempty"`
	Company       string     `json:"company,omitempty"`
	Note          string     `json:"note,omitempty"`
	Status        string     `json:"status"` // "pending", "accepted", "declined", "withdrawn"
	SentAt        time.Time  `json:"sent_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	AcceptedAt    *time.Time `json:"accepted_at,omitempty"`
	Source        string     `json:"source,omitempty"` // "search", "suggestions", "manual"
	SearchKeyword string     `json:"search_keyword,omitempty"`
}

// ConnectionRequestStatus constants
const (
	StatusPending   = "pending"
	StatusAccepted  = "accepted"
	StatusDeclined  = "declined"
	StatusWithdrawn = "withdrawn"
)

// SaveConnectionRequest saves or updates a connection request
func (s *Store) SaveConnectionRequest(req *ConnectionRequest) error {
	if req.Status == "" {
		req.Status = StatusPending
	}
	if req.SentAt.IsZero() {
		req.SentAt = time.Now()
	}

	result, err := s.db.Exec(`
		INSERT INTO connection_requests (
			profile_url, name, headline, company, note, status, 
			sent_at, source, search_keyword
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(profile_url) DO UPDATE SET
			name = COALESCE(excluded.name, connection_requests.name),
			headline = COALESCE(excluded.headline, connection_requests.headline),
			company = COALESCE(excluded.company, connection_requests.company),
			status = excluded.status,
			updated_at = CURRENT_TIMESTAMP
	`, req.ProfileURL, req.Name, req.Headline, req.Company, req.Note,
		req.Status, req.SentAt, req.Source, req.SearchKeyword)

	if err != nil {
		return fmt.Errorf("failed to save connection request: %w", err)
	}

	if req.ID == 0 {
		id, _ := result.LastInsertId()
		req.ID = id
	}

	// Update daily stats
	s.incrementDailyStat("connections_sent")

	return nil
}

// GetConnectionRequest retrieves a connection request by profile URL
func (s *Store) GetConnectionRequest(profileURL string) (*ConnectionRequest, error) {
	normalized := normalizeURL(profileURL)

	row := s.db.QueryRow(`
		SELECT id, profile_url, name, headline, company, note, status,
			   sent_at, updated_at, accepted_at, source, search_keyword
		FROM connection_requests
		WHERE profile_url = ? OR profile_url LIKE ?
	`, profileURL, "%"+normalized+"%")

	req := &ConnectionRequest{}
	var acceptedAt sql.NullTime
	var sentAt, updatedAt sql.NullTime
	var headline, company, note, source, searchKeyword sql.NullString

	err := row.Scan(
		&req.ID, &req.ProfileURL, &req.Name, &headline, &company,
		&note, &req.Status, &sentAt, &updatedAt, &acceptedAt,
		&source, &searchKeyword,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get connection request: %w", err)
	}

	if sentAt.Valid {
		req.SentAt = sentAt.Time
	}
	if updatedAt.Valid {
		req.UpdatedAt = updatedAt.Time
	}
	if acceptedAt.Valid {
		req.AcceptedAt = &acceptedAt.Time
	}
	if headline.Valid {
		req.Headline = headline.String
	}
	if company.Valid {
		req.Company = company.String
	}
	if note.Valid {
		req.Note = note.String
	}
	if source.Valid {
		req.Source = source.String
	}
	if searchKeyword.Valid {
		req.SearchKeyword = searchKeyword.String
	}

	return req, nil
}

// HasSentRequest checks if a connection request was already sent
func (s *Store) HasSentRequest(profileURL string) (bool, error) {
	req, err := s.GetConnectionRequest(profileURL)
	if err != nil {
		return false, err
	}
	return req != nil, nil
}

// UpdateRequestStatus updates the status of a connection request
func (s *Store) UpdateRequestStatus(profileURL, status string) error {
	var acceptedAt interface{}
	if status == StatusAccepted {
		acceptedAt = time.Now()
		s.incrementDailyStat("connections_accepted")
	}

	_, err := s.db.Exec(`
		UPDATE connection_requests 
		SET status = ?, updated_at = CURRENT_TIMESTAMP, accepted_at = ?
		WHERE profile_url = ?
	`, status, acceptedAt, profileURL)

	return err
}

// GetPendingRequests returns all pending connection requests
func (s *Store) GetPendingRequests() ([]ConnectionRequest, error) {
	return s.getRequestsByStatus(StatusPending)
}

// GetAcceptedRequests returns all accepted connection requests
func (s *Store) GetAcceptedRequests() ([]ConnectionRequest, error) {
	return s.getRequestsByStatus(StatusAccepted)
}

func (s *Store) getRequestsByStatus(status string) ([]ConnectionRequest, error) {
	rows, err := s.db.Query(`
		SELECT id, profile_url, name, headline, company, note, status,
			   sent_at, updated_at, accepted_at, source, search_keyword
		FROM connection_requests
		WHERE status = ?
		ORDER BY sent_at DESC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConnectionRequests(rows)
}

// GetTodayRequestCount returns the number of requests sent today
func (s *Store) GetTodayRequestCount() (int, error) {
	today := time.Now().Truncate(24 * time.Hour)

	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM connection_requests
		WHERE sent_at >= ?
	`, today).Scan(&count)

	return count, err
}

// GetAllConnectionRequests returns all connection requests with optional filters
func (s *Store) GetAllConnectionRequests(limit, offset int) ([]ConnectionRequest, error) {
	query := `
		SELECT id, profile_url, name, headline, company, note, status,
			   sent_at, updated_at, accepted_at, source, search_keyword
		FROM connection_requests
		ORDER BY sent_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConnectionRequests(rows)
}

func scanConnectionRequests(rows *sql.Rows) ([]ConnectionRequest, error) {
	var requests []ConnectionRequest

	for rows.Next() {
		var req ConnectionRequest
		var acceptedAt sql.NullTime
		var sentAt, updatedAt sql.NullTime
		var headline, company, note, source, searchKeyword sql.NullString

		err := rows.Scan(
			&req.ID, &req.ProfileURL, &req.Name, &headline, &company,
			&note, &req.Status, &sentAt, &updatedAt, &acceptedAt,
			&source, &searchKeyword,
		)
		if err != nil {
			return nil, err
		}

		if sentAt.Valid {
			req.SentAt = sentAt.Time
		}
		if updatedAt.Valid {
			req.UpdatedAt = updatedAt.Time
		}
		if acceptedAt.Valid {
			req.AcceptedAt = &acceptedAt.Time
		}
		if headline.Valid {
			req.Headline = headline.String
		}
		if company.Valid {
			req.Company = company.String
		}
		if note.Valid {
			req.Note = note.String
		}
		if source.Valid {
			req.Source = source.String
		}
		if searchKeyword.Valid {
			req.SearchKeyword = searchKeyword.String
		}

		requests = append(requests, req)
	}

	return requests, rows.Err()
}

// ConnectionRequestStats returns statistics about connection requests
type ConnectionRequestStats struct {
	TotalSent      int
	Pending        int
	Accepted       int
	Declined       int
	SentToday      int
	AcceptanceRate float64
	DailyLimit     int
	RemainingToday int
}

// GetConnectionRequestStats returns connection request statistics
func (s *Store) GetConnectionRequestStats(dailyLimit int) (*ConnectionRequestStats, error) {
	stats := &ConnectionRequestStats{DailyLimit: dailyLimit}

	// Get counts by status
	rows, err := s.db.Query(`
		SELECT status, COUNT(*) FROM connection_requests GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		switch status {
		case StatusPending:
			stats.Pending = count
		case StatusAccepted:
			stats.Accepted = count
		case StatusDeclined:
			stats.Declined = count
		}
		stats.TotalSent += count
	}

	// Get today's count
	stats.SentToday, _ = s.GetTodayRequestCount()
	stats.RemainingToday = dailyLimit - stats.SentToday
	if stats.RemainingToday < 0 {
		stats.RemainingToday = 0
	}

	// Calculate acceptance rate
	completed := stats.Accepted + stats.Declined
	if completed > 0 {
		stats.AcceptanceRate = float64(stats.Accepted) / float64(completed) * 100
	}

	return stats, nil
}

// normalizeURL normalizes LinkedIn URLs for comparison
func normalizeURL(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	return strings.ToLower(url)
}
