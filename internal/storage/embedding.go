package storage

import (
	"log"
	"time"
)

// SaveEmbedding caches an embedding vector for a tool.
func (s *SQLiteStorage) SaveEmbedding(toolName string, vector []float32, version string) error {
	if !s.enabled || s.db == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert vector to JSON for storage
	vectorJSON := vectorToJSON(vector)

	query := `
		INSERT OR REPLACE INTO tool_embeddings (tool_name, vector, version, created_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		toolName,
		vectorJSON,
		version,
		time.Now().Format(time.RFC3339),
	)

	if err != nil {
		log.Printf("Warning: failed to save embedding: %v", err)
	}

	return nil
}

// GetEmbedding retrieves a cached embedding for a tool.
func (s *SQLiteStorage) GetEmbedding(toolName string) ([]float32, string, error) {
	if !s.enabled || s.db == nil {
		return nil, "", nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		SELECT vector, version
		FROM tool_embeddings
		WHERE tool_name = ?
	`

	rows, err := s.db.Query(query, toolName)
	if err != nil {
		log.Printf("Warning: failed to query embedding: %v", err)
		return nil, "", nil
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, "", nil
	}

	var vectorJSON, version string
	if err := rows.Scan(&vectorJSON, &version); err != nil {
		log.Printf("Warning: failed to scan embedding: %v", err)
		return nil, "", nil
	}

	vector, err := jsonToVector(vectorJSON)
	if err != nil {
		log.Printf("Warning: failed to parse embedding vector: %v", err)
		return nil, "", nil
	}

	return vector, version, nil
}
