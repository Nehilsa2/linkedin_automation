package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ==================== PEOPLE SEARCH RESULTS ====================

// PersonSearchResult represents a discovered person profile from search
type PersonSearchResult struct {
	ID            int64      `json:"id"`
	ProfileURL    string     `json:"profile_url"`
	Name          string     `json:"name,omitempty"`
	Headline      string     `json:"headline,omitempty"`
	Company       string     `json:"company,omitempty"`
	Location      string     `json:"location,omitempty"`
	SearchKeyword string     `json:"search_keyword,omitempty"`
	PageNumber    int        `json:"page_number,omitempty"`
	DiscoveredAt  time.Time  `json:"discovered_at"`
	Processed     bool       `json:"processed"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

// SavePersonSearchResult saves a person search result
func (s *Store) SavePersonSearchResult(result *PersonSearchResult) error {
	if result.DiscoveredAt.IsZero() {
		result.DiscoveredAt = time.Now()
	}

	res, err := s.db.Exec(`
		INSERT INTO people_search_results (
			profile_url, name, headline, company, location,
			search_keyword, page_number, discovered_at, processed
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(profile_url, search_keyword) DO UPDATE SET
			name = COALESCE(excluded.name, people_search_results.name),
			headline = COALESCE(excluded.headline, people_search_results.headline),
			company = COALESCE(excluded.company, people_search_results.company),
			location = COALESCE(excluded.location, people_search_results.location)
	`, result.ProfileURL, result.Name, result.Headline, result.Company,
		result.Location, result.SearchKeyword, result.PageNumber,
		result.DiscoveredAt, result.Processed)

	if err != nil {
		return fmt.Errorf("failed to save person search result: %w", err)
	}

	if result.ID == 0 {
		id, _ := res.LastInsertId()
		result.ID = id
	}

	s.incrementDailyStat("profiles_searched")
	return nil
}

// SavePersonSearchResults saves multiple person search results in a batch
func (s *Store) SavePersonSearchResults(results []PersonSearchResult) error {
	return s.Transaction(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`
			INSERT INTO people_search_results (
				profile_url, name, headline, company, location,
				search_keyword, page_number, discovered_at, processed
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(profile_url, search_keyword) DO UPDATE SET
				name = COALESCE(excluded.name, people_search_results.name),
				headline = COALESCE(excluded.headline, people_search_results.headline),
				company = COALESCE(excluded.company, people_search_results.company),
				location = COALESCE(excluded.location, people_search_results.location)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		now := time.Now()
		for i := range results {
			if results[i].DiscoveredAt.IsZero() {
				results[i].DiscoveredAt = now
			}
			_, err := stmt.Exec(
				results[i].ProfileURL, results[i].Name, results[i].Headline,
				results[i].Company, results[i].Location, results[i].SearchKeyword,
				results[i].PageNumber, results[i].DiscoveredAt, results[i].Processed,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetUnprocessedPeopleResults returns people search results that haven't been processed
func (s *Store) GetUnprocessedPeopleResults(searchKeyword string, limit int) ([]PersonSearchResult, error) {
	query := `
		SELECT id, profile_url, name, headline, company, location,
			   search_keyword, page_number, discovered_at, processed, processed_at
		FROM people_search_results
		WHERE processed = FALSE
	`
	args := []interface{}{}

	if searchKeyword != "" {
		query += " AND search_keyword = ?"
		args = append(args, searchKeyword)
	}

	query += " ORDER BY discovered_at ASC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPersonResults(rows)
}

// MarkPersonProcessed marks a person search result as processed
func (s *Store) MarkPersonProcessed(profileURL string) error {
	_, err := s.db.Exec(`
		UPDATE people_search_results 
		SET processed = TRUE, processed_at = CURRENT_TIMESTAMP
		WHERE profile_url = ?
	`, profileURL)
	return err
}

// GetPeopleByKeyword returns all people results for a search keyword
func (s *Store) GetPeopleByKeyword(keyword string) ([]PersonSearchResult, error) {
	rows, err := s.db.Query(`
		SELECT id, profile_url, name, headline, company, location,
			   search_keyword, page_number, discovered_at, processed, processed_at
		FROM people_search_results
		WHERE search_keyword = ?
		ORDER BY page_number ASC, discovered_at ASC
	`, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPersonResults(rows)
}

// HasPersonResult checks if a profile URL exists in people search results
func (s *Store) HasPersonResult(profileURL string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM people_search_results WHERE profile_url = ?
	`, profileURL).Scan(&count)
	return count > 0, err
}

// GetPeopleSearchProgress returns the last page number searched for people
func (s *Store) GetPeopleSearchProgress(keyword string) (int, error) {
	var maxPage sql.NullInt64
	err := s.db.QueryRow(`
		SELECT MAX(page_number) FROM people_search_results
		WHERE search_keyword = ?
	`, keyword).Scan(&maxPage)

	if err != nil || !maxPage.Valid {
		return 0, err
	}
	return int(maxPage.Int64), nil
}

// GetPeopleSearchStats returns statistics for people search
func (s *Store) GetPeopleSearchStats(keyword string) (total int, processed int, err error) {
	query := `SELECT COUNT(*), COALESCE(SUM(CASE WHEN processed THEN 1 ELSE 0 END), 0) FROM people_search_results`
	args := []interface{}{}

	if keyword != "" {
		query += " WHERE search_keyword = ?"
		args = append(args, keyword)
	}

	err = s.db.QueryRow(query, args...).Scan(&total, &processed)
	return
}

func scanPersonResults(rows *sql.Rows) ([]PersonSearchResult, error) {
	var results []PersonSearchResult

	for rows.Next() {
		var result PersonSearchResult
		var processedAt sql.NullTime
		var name, headline, company, location sql.NullString

		err := rows.Scan(
			&result.ID, &result.ProfileURL, &name, &headline, &company, &location,
			&result.SearchKeyword, &result.PageNumber,
			&result.DiscoveredAt, &result.Processed, &processedAt,
		)
		if err != nil {
			return nil, err
		}

		if name.Valid {
			result.Name = name.String
		}
		if headline.Valid {
			result.Headline = headline.String
		}
		if company.Valid {
			result.Company = company.String
		}
		if location.Valid {
			result.Location = location.String
		}
		if processedAt.Valid {
			result.ProcessedAt = &processedAt.Time
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// ==================== COMPANY SEARCH RESULTS ====================

// CompanySearchResult represents a discovered company from search
type CompanySearchResult struct {
	ID            int64      `json:"id"`
	CompanyURL    string     `json:"company_url"`
	Name          string     `json:"name,omitempty"`
	Industry      string     `json:"industry,omitempty"`
	Location      string     `json:"location,omitempty"`
	EmployeeCount string     `json:"employee_count,omitempty"`
	Description   string     `json:"description,omitempty"`
	SearchKeyword string     `json:"search_keyword,omitempty"`
	PageNumber    int        `json:"page_number,omitempty"`
	DiscoveredAt  time.Time  `json:"discovered_at"`
	Processed     bool       `json:"processed"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

// SaveCompanySearchResult saves a company search result
func (s *Store) SaveCompanySearchResult(result *CompanySearchResult) error {
	if result.DiscoveredAt.IsZero() {
		result.DiscoveredAt = time.Now()
	}

	res, err := s.db.Exec(`
		INSERT INTO company_search_results (
			company_url, name, industry, location, employee_count,
			description, search_keyword, page_number, discovered_at, processed
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(company_url, search_keyword) DO UPDATE SET
			name = COALESCE(excluded.name, company_search_results.name),
			industry = COALESCE(excluded.industry, company_search_results.industry),
			location = COALESCE(excluded.location, company_search_results.location),
			employee_count = COALESCE(excluded.employee_count, company_search_results.employee_count),
			description = COALESCE(excluded.description, company_search_results.description)
	`, result.CompanyURL, result.Name, result.Industry, result.Location,
		result.EmployeeCount, result.Description, result.SearchKeyword,
		result.PageNumber, result.DiscoveredAt, result.Processed)

	if err != nil {
		return fmt.Errorf("failed to save company search result: %w", err)
	}

	if result.ID == 0 {
		id, _ := res.LastInsertId()
		result.ID = id
	}

	return nil
}

// SaveCompanySearchResults saves multiple company search results in a batch
func (s *Store) SaveCompanySearchResults(results []CompanySearchResult) error {
	return s.Transaction(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`
			INSERT INTO company_search_results (
				company_url, name, industry, location, employee_count,
				description, search_keyword, page_number, discovered_at, processed
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(company_url, search_keyword) DO UPDATE SET
				name = COALESCE(excluded.name, company_search_results.name),
				industry = COALESCE(excluded.industry, company_search_results.industry),
				location = COALESCE(excluded.location, company_search_results.location),
				employee_count = COALESCE(excluded.employee_count, company_search_results.employee_count),
				description = COALESCE(excluded.description, company_search_results.description)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		now := time.Now()
		for i := range results {
			if results[i].DiscoveredAt.IsZero() {
				results[i].DiscoveredAt = now
			}
			_, err := stmt.Exec(
				results[i].CompanyURL, results[i].Name, results[i].Industry,
				results[i].Location, results[i].EmployeeCount, results[i].Description,
				results[i].SearchKeyword, results[i].PageNumber,
				results[i].DiscoveredAt, results[i].Processed,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetUnprocessedCompanyResults returns company search results that haven't been processed
func (s *Store) GetUnprocessedCompanyResults(searchKeyword string, limit int) ([]CompanySearchResult, error) {
	query := `
		SELECT id, company_url, name, industry, location, employee_count,
			   description, search_keyword, page_number, discovered_at, processed, processed_at
		FROM company_search_results
		WHERE processed = FALSE
	`
	args := []interface{}{}

	if searchKeyword != "" {
		query += " AND search_keyword = ?"
		args = append(args, searchKeyword)
	}

	query += " ORDER BY discovered_at ASC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCompanyResults(rows)
}

// MarkCompanyProcessed marks a company search result as processed
func (s *Store) MarkCompanyProcessed(companyURL string) error {
	_, err := s.db.Exec(`
		UPDATE company_search_results 
		SET processed = TRUE, processed_at = CURRENT_TIMESTAMP
		WHERE company_url = ?
	`, companyURL)
	return err
}

// GetCompaniesByKeyword returns all company results for a search keyword
func (s *Store) GetCompaniesByKeyword(keyword string) ([]CompanySearchResult, error) {
	rows, err := s.db.Query(`
		SELECT id, company_url, name, industry, location, employee_count,
			   description, search_keyword, page_number, discovered_at, processed, processed_at
		FROM company_search_results
		WHERE search_keyword = ?
		ORDER BY page_number ASC, discovered_at ASC
	`, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCompanyResults(rows)
}

// HasCompanyResult checks if a company URL exists in company search results
func (s *Store) HasCompanyResult(companyURL string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM company_search_results WHERE company_url = ?
	`, companyURL).Scan(&count)
	return count > 0, err
}

// GetCompanySearchProgress returns the last page number searched for companies
func (s *Store) GetCompanySearchProgress(keyword string) (int, error) {
	var maxPage sql.NullInt64
	err := s.db.QueryRow(`
		SELECT MAX(page_number) FROM company_search_results
		WHERE search_keyword = ?
	`, keyword).Scan(&maxPage)

	if err != nil || !maxPage.Valid {
		return 0, err
	}
	return int(maxPage.Int64), nil
}

// GetCompanySearchStats returns statistics for company search
func (s *Store) GetCompanySearchStats(keyword string) (total int, processed int, err error) {
	query := `SELECT COUNT(*), COALESCE(SUM(CASE WHEN processed THEN 1 ELSE 0 END), 0) FROM company_search_results`
	args := []interface{}{}

	if keyword != "" {
		query += " WHERE search_keyword = ?"
		args = append(args, keyword)
	}

	err = s.db.QueryRow(query, args...).Scan(&total, &processed)
	return
}

func scanCompanyResults(rows *sql.Rows) ([]CompanySearchResult, error) {
	var results []CompanySearchResult

	for rows.Next() {
		var result CompanySearchResult
		var processedAt sql.NullTime
		var name, industry, location, employeeCount, description sql.NullString

		err := rows.Scan(
			&result.ID, &result.CompanyURL, &name, &industry, &location,
			&employeeCount, &description, &result.SearchKeyword, &result.PageNumber,
			&result.DiscoveredAt, &result.Processed, &processedAt,
		)
		if err != nil {
			return nil, err
		}

		if name.Valid {
			result.Name = name.String
		}
		if industry.Valid {
			result.Industry = industry.String
		}
		if location.Valid {
			result.Location = location.String
		}
		if employeeCount.Valid {
			result.EmployeeCount = employeeCount.String
		}
		if description.Valid {
			result.Description = description.String
		}
		if processedAt.Valid {
			result.ProcessedAt = &processedAt.Time
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// ==================== BACKWARD COMPATIBILITY ====================

// SearchResult is kept for backward compatibility (deprecated)
type SearchResult struct {
	ID            int64      `json:"id"`
	ProfileURL    string     `json:"profile_url"`
	Name          string     `json:"name,omitempty"`
	Headline      string     `json:"headline,omitempty"`
	Company       string     `json:"company,omitempty"`
	SearchType    string     `json:"search_type,omitempty"`
	SearchKeyword string     `json:"search_keyword,omitempty"`
	PageNumber    int        `json:"page_number,omitempty"`
	DiscoveredAt  time.Time  `json:"discovered_at"`
	Processed     bool       `json:"processed"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

// SaveSearchResult saves a search result (backward compatibility - routes to appropriate table)
func (s *Store) SaveSearchResult(result *SearchResult) error {
	if result.SearchType == "companies" {
		return s.SaveCompanySearchResult(&CompanySearchResult{
			CompanyURL:    result.ProfileURL,
			Name:          result.Name,
			SearchKeyword: result.SearchKeyword,
			PageNumber:    result.PageNumber,
			DiscoveredAt:  result.DiscoveredAt,
			Processed:     result.Processed,
		})
	}
	return s.SavePersonSearchResult(&PersonSearchResult{
		ProfileURL:    result.ProfileURL,
		Name:          result.Name,
		Headline:      result.Headline,
		Company:       result.Company,
		SearchKeyword: result.SearchKeyword,
		PageNumber:    result.PageNumber,
		DiscoveredAt:  result.DiscoveredAt,
		Processed:     result.Processed,
	})
}

// SaveSearchResults saves multiple search results (backward compatibility)
func (s *Store) SaveSearchResults(results []SearchResult) error {
	var people []PersonSearchResult
	var companies []CompanySearchResult

	for _, r := range results {
		if r.SearchType == "companies" {
			companies = append(companies, CompanySearchResult{
				CompanyURL:    r.ProfileURL,
				Name:          r.Name,
				SearchKeyword: r.SearchKeyword,
				PageNumber:    r.PageNumber,
				DiscoveredAt:  r.DiscoveredAt,
				Processed:     r.Processed,
			})
		} else {
			people = append(people, PersonSearchResult{
				ProfileURL:    r.ProfileURL,
				Name:          r.Name,
				Headline:      r.Headline,
				Company:       r.Company,
				SearchKeyword: r.SearchKeyword,
				PageNumber:    r.PageNumber,
				DiscoveredAt:  r.DiscoveredAt,
				Processed:     r.Processed,
			})
		}
	}

	if len(people) > 0 {
		if err := s.SavePersonSearchResults(people); err != nil {
			return err
		}
	}
	if len(companies) > 0 {
		if err := s.SaveCompanySearchResults(companies); err != nil {
			return err
		}
	}
	return nil
}

// GetUnprocessedSearchResults returns unprocessed people results (backward compatibility)
func (s *Store) GetUnprocessedSearchResults(searchKeyword string, limit int) ([]SearchResult, error) {
	people, err := s.GetUnprocessedPeopleResults(searchKeyword, limit)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, p := range people {
		results = append(results, SearchResult{
			ID:            p.ID,
			ProfileURL:    p.ProfileURL,
			Name:          p.Name,
			Headline:      p.Headline,
			Company:       p.Company,
			SearchType:    "people",
			SearchKeyword: p.SearchKeyword,
			PageNumber:    p.PageNumber,
			DiscoveredAt:  p.DiscoveredAt,
			Processed:     p.Processed,
			ProcessedAt:   p.ProcessedAt,
		})
	}
	return results, nil
}

// MarkSearchResultProcessed marks a search result as processed (backward compatibility)
func (s *Store) MarkSearchResultProcessed(profileURL string) error {
	return s.MarkPersonProcessed(profileURL)
}

// HasSearchResult checks if a profile URL exists (backward compatibility)
func (s *Store) HasSearchResult(profileURL string) (bool, error) {
	return s.HasPersonResult(profileURL)
}

// GetSearchProgress returns search progress (backward compatibility)
func (s *Store) GetSearchProgress(searchType, keyword string) (int, error) {
	if searchType == "companies" {
		return s.GetCompanySearchProgress(keyword)
	}
	return s.GetPeopleSearchProgress(keyword)
}

// ==================== WORKFLOW STATE ====================

// WorkflowState represents the state of a running workflow
type WorkflowState struct {
	ID           int64                  `json:"id"`
	WorkflowType string                 `json:"workflow_type"` // "search", "connect", "message"
	Status       string                 `json:"status"`        // "in_progress", "paused", "completed", "failed"
	CurrentStep  string                 `json:"current_step,omitempty"`
	CurrentIndex int                    `json:"current_index"`
	TotalItems   int                    `json:"total_items"`
	StartedAt    time.Time              `json:"started_at"`
	PausedAt     *time.Time             `json:"paused_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowStatus constants
const (
	WorkflowStatusInProgress = "in_progress"
	WorkflowStatusPaused     = "paused"
	WorkflowStatusCompleted  = "completed"
	WorkflowStatusFailed     = "failed"
)

// WorkflowType constants
const (
	WorkflowTypeSearch  = "search"
	WorkflowTypeConnect = "connect"
	WorkflowTypeMessage = "message"
)

// SaveWorkflowState saves or updates workflow state
func (s *Store) SaveWorkflowState(state *WorkflowState) error {
	if state.StartedAt.IsZero() {
		state.StartedAt = time.Now()
	}
	if state.Status == "" {
		state.Status = WorkflowStatusInProgress
	}

	metadataJSON, err := json.Marshal(state.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	if state.ID == 0 {
		// Insert new
		result, err := s.db.Exec(`
			INSERT INTO workflow_state (
				workflow_type, status, current_step, current_index,
				total_items, started_at, error_message, metadata
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, state.WorkflowType, state.Status, state.CurrentStep, state.CurrentIndex,
			state.TotalItems, state.StartedAt, state.ErrorMessage, string(metadataJSON))

		if err != nil {
			return fmt.Errorf("failed to save workflow state: %w", err)
		}

		id, _ := result.LastInsertId()
		state.ID = id
	} else {
		// Update existing
		_, err := s.db.Exec(`
			UPDATE workflow_state SET
				status = ?, current_step = ?, current_index = ?,
				total_items = ?, paused_at = ?, completed_at = ?,
				error_message = ?, metadata = ?
			WHERE id = ?
		`, state.Status, state.CurrentStep, state.CurrentIndex,
			state.TotalItems, state.PausedAt, state.CompletedAt,
			state.ErrorMessage, string(metadataJSON), state.ID)

		if err != nil {
			return fmt.Errorf("failed to update workflow state: %w", err)
		}
	}

	return nil
}

// GetActiveWorkflow returns the active workflow of a given type
func (s *Store) GetActiveWorkflow(workflowType string) (*WorkflowState, error) {
	row := s.db.QueryRow(`
		SELECT id, workflow_type, status, current_step, current_index,
			   total_items, started_at, paused_at, completed_at, 
			   error_message, metadata
		FROM workflow_state
		WHERE workflow_type = ? AND status IN (?, ?)
		ORDER BY started_at DESC
		LIMIT 1
	`, workflowType, WorkflowStatusInProgress, WorkflowStatusPaused)

	return scanWorkflowState(row)
}

// GetLastWorkflow returns the most recent workflow of a given type
func (s *Store) GetLastWorkflow(workflowType string) (*WorkflowState, error) {
	row := s.db.QueryRow(`
		SELECT id, workflow_type, status, current_step, current_index,
			   total_items, started_at, paused_at, completed_at, 
			   error_message, metadata
		FROM workflow_state
		WHERE workflow_type = ?
		ORDER BY started_at DESC
		LIMIT 1
	`, workflowType)

	return scanWorkflowState(row)
}

// PauseWorkflow pauses an active workflow
func (s *Store) PauseWorkflow(workflowID int64) error {
	_, err := s.db.Exec(`
		UPDATE workflow_state 
		SET status = ?, paused_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, WorkflowStatusPaused, workflowID)
	return err
}

// CompleteWorkflow marks a workflow as completed
func (s *Store) CompleteWorkflow(workflowID int64) error {
	_, err := s.db.Exec(`
		UPDATE workflow_state 
		SET status = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, WorkflowStatusCompleted, workflowID)
	return err
}

// FailWorkflow marks a workflow as failed with an error message
func (s *Store) FailWorkflow(workflowID int64, errMsg string) error {
	_, err := s.db.Exec(`
		UPDATE workflow_state 
		SET status = ?, error_message = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, WorkflowStatusFailed, errMsg, workflowID)
	return err
}

// UpdateWorkflowProgress updates the current progress of a workflow
func (s *Store) UpdateWorkflowProgress(workflowID int64, currentIndex int, currentStep string) error {
	_, err := s.db.Exec(`
		UPDATE workflow_state 
		SET current_index = ?, current_step = ?
		WHERE id = ?
	`, currentIndex, currentStep, workflowID)
	return err
}

func scanWorkflowState(row *sql.Row) (*WorkflowState, error) {
	state := &WorkflowState{}
	var pausedAt, completedAt sql.NullTime
	var currentStep, errorMessage sql.NullString
	var metadataJSON sql.NullString

	err := row.Scan(
		&state.ID, &state.WorkflowType, &state.Status, &currentStep,
		&state.CurrentIndex, &state.TotalItems, &state.StartedAt,
		&pausedAt, &completedAt, &errorMessage, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if currentStep.Valid {
		state.CurrentStep = currentStep.String
	}
	if errorMessage.Valid {
		state.ErrorMessage = errorMessage.String
	}
	if pausedAt.Valid {
		state.PausedAt = &pausedAt.Time
	}
	if completedAt.Valid {
		state.CompletedAt = &completedAt.Time
	}
	if metadataJSON.Valid && metadataJSON.String != "" {
		json.Unmarshal([]byte(metadataJSON.String), &state.Metadata)
	}
	if state.Metadata == nil {
		state.Metadata = make(map[string]interface{})
	}

	return state, nil
}

// DailyStats represents daily statistics
type DailyStats struct {
	Date                string `json:"date"`
	ConnectionsSent     int    `json:"connections_sent"`
	ConnectionsAccepted int    `json:"connections_accepted"`
	MessagesSent        int    `json:"messages_sent"`
	ProfilesSearched    int    `json:"profiles_searched"`
}

// GetDailyStats returns statistics for a specific date
func (s *Store) GetDailyStats(date string) (*DailyStats, error) {
	if date == "" {
		date = getTodayDate()
	}

	row := s.db.QueryRow(`
		SELECT date, connections_sent, connections_accepted, 
			   messages_sent, profiles_searched
		FROM daily_stats
		WHERE date = ?
	`, date)

	stats := &DailyStats{}
	err := row.Scan(
		&stats.Date, &stats.ConnectionsSent, &stats.ConnectionsAccepted,
		&stats.MessagesSent, &stats.ProfilesSearched,
	)

	if err == sql.ErrNoRows {
		return &DailyStats{Date: date}, nil
	}
	return stats, err
}

// GetWeeklyStats returns statistics for the last 7 days
func (s *Store) GetWeeklyStats() ([]DailyStats, error) {
	rows, err := s.db.Query(`
		SELECT date, connections_sent, connections_accepted, 
			   messages_sent, profiles_searched
		FROM daily_stats
		WHERE date >= date('now', '-7 days')
		ORDER BY date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStats
	for rows.Next() {
		var s DailyStats
		if err := rows.Scan(&s.Date, &s.ConnectionsSent, &s.ConnectionsAccepted,
			&s.MessagesSent, &s.ProfilesSearched); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}
