// Package persistence provides SQLite-based state persistence for LinkedIn automation
package persistence

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	DefaultDBPath = "linkedin_automation.db"
)

// Store handles all persistence operations using SQLite
type Store struct {
	db     *sql.DB
	dbPath string
}

// NewStore creates a new persistence store
func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	store := &Store{
		db:     db,
		dbPath: dbPath,
	}

	if err := store.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// initTables creates all required tables
func (s *Store) initTables() error {
	tables := []string{
		// Connection requests table
		`CREATE TABLE IF NOT EXISTS connection_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_url TEXT UNIQUE NOT NULL,
			name TEXT,
			headline TEXT,
			company TEXT,
			note TEXT,
			status TEXT DEFAULT 'pending',
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			accepted_at DATETIME,
			source TEXT,
			search_keyword TEXT
		)`,

		// Connections table (accepted connections)
		`CREATE TABLE IF NOT EXISTS connections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_url TEXT UNIQUE NOT NULL,
			name TEXT,
			headline TEXT,
			company TEXT,
			connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			has_messaged BOOLEAN DEFAULT FALSE,
			last_message_at DATETIME,
			message_count INTEGER DEFAULT 0,
			notes TEXT
		)`,

		// Messages table
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id TEXT,
			recipient_url TEXT NOT NULL,
			recipient_name TEXT,
			content TEXT NOT NULL,
			template_name TEXT,
			message_type TEXT,
			status TEXT DEFAULT 'sent',
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			delivered_at DATETIME,
			read_at DATETIME,
			error_message TEXT
		)`,

		// People search results table
		`CREATE TABLE IF NOT EXISTS people_search_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_url TEXT NOT NULL,
			name TEXT,
			headline TEXT,
			company TEXT,
			location TEXT,
			search_keyword TEXT,
			page_number INTEGER,
			discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			processed BOOLEAN DEFAULT FALSE,
			processed_at DATETIME,
			UNIQUE(profile_url, search_keyword)
		)`,

		// Companies search results table
		`CREATE TABLE IF NOT EXISTS company_search_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_url TEXT NOT NULL,
			name TEXT,
			industry TEXT,
			location TEXT,
			employee_count TEXT,
			description TEXT,
			search_keyword TEXT,
			page_number INTEGER,
			discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			processed BOOLEAN DEFAULT FALSE,
			processed_at DATETIME,
			UNIQUE(company_url, search_keyword)
		)`,

		// Workflow state table (for resumption)
		`CREATE TABLE IF NOT EXISTS workflow_state (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workflow_type TEXT NOT NULL,
			status TEXT DEFAULT 'in_progress',
			current_step TEXT,
			current_index INTEGER DEFAULT 0,
			total_items INTEGER DEFAULT 0,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			paused_at DATETIME,
			completed_at DATETIME,
			error_message TEXT,
			metadata TEXT
		)`,

		// Daily stats table
		`CREATE TABLE IF NOT EXISTS daily_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date DATE UNIQUE NOT NULL,
			connections_sent INTEGER DEFAULT 0,
			connections_accepted INTEGER DEFAULT 0,
			messages_sent INTEGER DEFAULT 0,
			profiles_searched INTEGER DEFAULT 0
		)`,
	}

	// Create tables
	for _, table := range tables {
		if _, err := s.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_connection_requests_status ON connection_requests(status)`,
		`CREATE INDEX IF NOT EXISTS idx_connection_requests_sent_at ON connection_requests(sent_at)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(recipient_url)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages(sent_at)`,
		`CREATE INDEX IF NOT EXISTS idx_people_search_processed ON people_search_results(processed)`,
		`CREATE INDEX IF NOT EXISTS idx_people_search_keyword ON people_search_results(search_keyword)`,
		`CREATE INDEX IF NOT EXISTS idx_company_search_processed ON company_search_results(processed)`,
		`CREATE INDEX IF NOT EXISTS idx_company_search_keyword ON company_search_results(search_keyword)`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_state_status ON workflow_state(status)`,
	}

	for _, idx := range indexes {
		if _, err := s.db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// GetDB returns the underlying database connection for advanced queries
func (s *Store) GetDB() *sql.DB {
	return s.db
}

// Transaction executes a function within a database transaction
func (s *Store) Transaction(fn func(*sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Save is a no-op for SQLite (for interface compatibility)
func (s *Store) Save() error {
	return nil
}

// getTodayDate returns today's date in YYYY-MM-DD format
func getTodayDate() string {
	return time.Now().Format("2006-01-02")
}

// ensureDailyStats ensures a record exists for today
func (s *Store) ensureDailyStats() error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO daily_stats (date) VALUES (?)
	`, getTodayDate())
	return err
}

// incrementDailyStat increments a daily statistic
func (s *Store) incrementDailyStat(field string) error {
	s.ensureDailyStats()

	query := fmt.Sprintf(`
		UPDATE daily_stats SET %s = %s + 1 WHERE date = ?
	`, field, field)

	_, err := s.db.Exec(query, getTodayDate())
	return err
}
