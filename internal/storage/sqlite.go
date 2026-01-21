/*
Package storage provides SQLite database migrations and helper functions.

This file contains schema definitions, migration logic, and vector serialization
utilities for the storage layer.
*/
package storage

import (
	"encoding/json"
	"fmt"
	"log"
)

// runMigrations executes database schema migrations.
func (s *SQLiteStorage) runMigrations() error {
	if !s.enabled || s.db == nil {
		return nil
	}

	// Create migrations table
	if err := s.createMigrationsTable(); err != nil {
		return err
	}

	// Get current version
	version, err := s.getCurrentMigrationVersion()
	if err != nil {
		return err
	}

	// Run migrations in order
	migrations := []migration{
		{version: 1, name: "initial_schema", up: s.migration001InitialSchema},
	}

	for _, m := range migrations {
		if version < m.version {
			log.Printf("Running migration %d: %s", m.version, m.name)
			if err := m.up(); err != nil {
				return fmt.Errorf("migration %d failed: %w", m.version, err)
			}
			if err := s.setMigrationVersion(m.version); err != nil {
				return err
			}
		}
	}

	return nil
}

// migration represents a single database migration.
type migration struct {
	version int
	name    string
	up      func() error
}

// createMigrationsTable creates the schema_migrations table.
func (s *SQLiteStorage) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`
	_, err := s.db.Exec(query)
	return err
}

// getCurrentMigrationVersion returns the highest applied migration version.
func (s *SQLiteStorage) getCurrentMigrationVersion() (int, error) {
	query := "SELECT COALESCE(MAX(version), 0) FROM schema_migrations"
	row := s.db.QueryRow(query)

	var version int
	if err := row.Scan(&version); err != nil {
		return 0, err
	}

	return version, nil
}

// setMigrationVersion records a migration as applied.
func (s *SQLiteStorage) setMigrationVersion(version int) error {
	query := "INSERT INTO schema_migrations (version, name) VALUES (?, ?)"
	_, err := s.db.Exec(query, version, fmt.Sprintf("migration_%d", version))
	return err
}

// migration001InitialSchema creates the initial database schema.
func (s *SQLiteStorage) migration001InitialSchema() error {
	// Create tool_usage table
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tool_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tool_name TEXT NOT NULL,
			context_hash TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			selected INTEGER NOT NULL,
			rating INTEGER,
			was_recommended INTEGER
		)
	`); err != nil {
		return fmt.Errorf("failed to create tool_usage table: %w", err)
	}

	// Create indexes for tool_usage
	if _, err := s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_tool_usage_tool
		ON tool_usage(tool_name)
	`); err != nil {
		return fmt.Errorf("failed to create tool_usage tool index: %w", err)
	}

	if _, err := s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_tool_usage_context
		ON tool_usage(context_hash)
	`); err != nil {
		return fmt.Errorf("failed to create tool_usage context index: %w", err)
	}

	if _, err := s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_tool_usage_timestamp
		ON tool_usage(timestamp DESC)
	`); err != nil {
		return fmt.Errorf("failed to create tool_usage timestamp index: %w", err)
	}

	// Create search_history table
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			search_id TEXT NOT NULL UNIQUE,
			query_hash TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			results_count INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("failed to create search_history table: %w", err)
	}

	// Create index for search_history
	if _, err := s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_search_history_timestamp
		ON search_history(timestamp DESC)
	`); err != nil {
		return fmt.Errorf("failed to create search_history timestamp index: %w", err)
	}

	// Create tool_embeddings table
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tool_embeddings (
			tool_name TEXT PRIMARY KEY,
			vector BLOB NOT NULL,
			version TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create tool_embeddings table: %w", err)
	}

	return nil
}

// vectorToJSON converts a float32 vector to JSON for storage.
func vectorToJSON(vector []float32) string {
	data, err := json.Marshal(vector)
	if err != nil {
		log.Printf("Warning: failed to marshal vector: %v", err)
		return "[]"
	}
	return string(data)
}

// jsonToVector parses JSON storage back to a float32 vector.
func jsonToVector(jsonStr string) ([]float32, error) {
	var vector []float32
	if err := json.Unmarshal([]byte(jsonStr), &vector); err != nil {
		return nil, err
	}
	return vector, nil
}
