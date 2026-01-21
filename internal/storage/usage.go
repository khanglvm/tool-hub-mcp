package storage

import (
	"log"
	"time"
)

// RecordUsage records a tool usage event.
func (s *SQLiteStorage) RecordUsage(event UsageEvent) error {
	if !s.enabled || s.db == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	selected := 0
	if event.Selected {
		selected = 1
	}
	wasRecommended := 0
	if event.WasRecommended {
		wasRecommended = 1
	}

	query := `
		INSERT INTO tool_usage (tool_name, context_hash, timestamp, selected, rating, was_recommended)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		event.ToolName,
		event.ContextHash,
		event.Timestamp.Format(time.RFC3339),
		selected,
		event.Rating,
		wasRecommended,
	)

	if err != nil {
		log.Printf("Warning: failed to record usage: %v", err)
	}

	return nil
}

// GetUsageHistory retrieves usage history for a tool since a given time.
func (s *SQLiteStorage) GetUsageHistory(toolName string, since time.Time) ([]UsageEvent, error) {
	if !s.enabled || s.db == nil {
		return []UsageEvent{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		SELECT tool_name, context_hash, timestamp, selected, rating, was_recommended
		FROM tool_usage
		WHERE tool_name = ? AND timestamp >= ?
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query, toolName, since.Format(time.RFC3339))
	if err != nil {
		log.Printf("Warning: failed to query usage history: %v", err)
		return []UsageEvent{}, nil
	}
	defer rows.Close()

	var events []UsageEvent
	for rows.Next() {
		var event UsageEvent
		var timestampStr string
		var selected, wasRecommended int
		var rating int

		if err := rows.Scan(
			&event.ToolName,
			&event.ContextHash,
			&timestampStr,
			&selected,
			&rating,
			&wasRecommended,
		); err != nil {
			log.Printf("Warning: failed to scan usage row: %v", err)
			continue
		}

		event.Selected = selected == 1
		event.WasRecommended = wasRecommended == 1
		event.Rating = rating

		event.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			log.Printf("Warning: failed to parse timestamp: %v", err)
			continue
		}

		events = append(events, event)
	}

	return events, nil
}
