package storage

import (
	"log"
	"time"
)

// RecordSearch records a search query for analytics.
func (s *SQLiteStorage) RecordSearch(search SearchRecord) error {
	if !s.enabled || s.db == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO search_history (search_id, query_hash, timestamp, results_count)
		VALUES (?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		search.SearchID,
		search.QueryHash,
		search.Timestamp.Format(time.RFC3339),
		search.ResultsCount,
	)

	if err != nil {
		log.Printf("Warning: failed to record search: %v", err)
	}

	return nil
}

// Cleanup removes old records based on retention policy.
func (s *SQLiteStorage) Cleanup(retention time.Duration) error {
	if !s.enabled || s.db == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retention).Format(time.RFC3339)

	// Cleanup tool_usage
	if _, err := s.db.Exec("DELETE FROM tool_usage WHERE timestamp < ?", cutoff); err != nil {
		log.Printf("Warning: failed to cleanup tool_usage: %v", err)
	}

	// Cleanup search_history
	if _, err := s.db.Exec("DELETE FROM search_history WHERE timestamp < ?", cutoff); err != nil {
		log.Printf("Warning: failed to cleanup search_history: %v", err)
	}

	// Vacuum to reclaim space
	if _, err := s.db.Exec("VACUUM"); err != nil {
		log.Printf("Warning: failed to vacuum database: %v", err)
	}

	return nil
}
