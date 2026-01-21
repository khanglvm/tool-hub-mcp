/*
Package storage implements a persistent storage layer for learning and history.

This package provides SQLite-based storage for tool usage tracking, search history,
and embedding caching with graceful degradation if the database is unavailable.

The database is stored at ~/.tool-hub-mcp/history.db and uses modernc.org/sqlite
(a pure Go, CGo-free implementation).
*/
package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Storage defines the interface for persistent storage operations.
type Storage interface {
	// Init initializes the database and runs migrations.
	Init() error

	// RecordUsage records a tool usage event.
	RecordUsage(event UsageEvent) error

	// GetUsageHistory retrieves usage history for a tool since a given time.
	GetUsageHistory(toolName string, since time.Time) ([]UsageEvent, error)

	// RecordSearch records a search query for analytics.
	RecordSearch(search SearchRecord) error

	// SaveEmbedding caches an embedding vector for a tool.
	SaveEmbedding(toolName string, vector []float32, version string) error

	// GetEmbedding retrieves a cached embedding for a tool.
	GetEmbedding(toolName string) ([]float32, string, error)

	// Cleanup removes old records based on retention policy.
	Cleanup(retention time.Duration) error

	// Close closes the database connection.
	Close() error
}

// SQLiteStorage implements the Storage interface using SQLite.
type SQLiteStorage struct {
	db       *sql.DB
	dbPath   string
	enabled  bool
	mu       sync.Mutex
	initOnce sync.Once
}

// NewStorage creates a new SQLite storage instance.
//
// The database is created at ~/.tool-hub-mcp/history.db.
// If the directory doesn't exist, it will be created.
// If the database cannot be opened, the storage will be disabled but operations will not fail.
func NewStorage() *SQLiteStorage {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: failed to get home directory: %v", err)
		return &SQLiteStorage{enabled: false}
	}

	dbDir := filepath.Join(home, ".tool-hub-mcp")
	dbPath := filepath.Join(dbDir, "history.db")

	return &SQLiteStorage{
		dbPath:  dbPath,
		enabled: true,
	}
}

// Init initializes the database and runs migrations.
//
// If initialization fails, storage is disabled and subsequent operations
// become no-ops (graceful degradation).
func (s *SQLiteStorage) Init() error {
	if !s.enabled {
		return nil
	}

	var initErr error
	s.initOnce.Do(func() {
		// Ensure directory exists
		dbDir := filepath.Dir(s.dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create db directory: %w", err)
			s.enabled = false
			return
		}

		// Open database
		db, err := sql.Open("sqlite", s.dbPath)
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			s.enabled = false
			log.Printf("Warning: %v", initErr)
			return
		}
		s.db = db

		// Test connection
		if err := db.Ping(); err != nil {
			initErr = fmt.Errorf("failed to ping database: %w", err)
			s.enabled = false
			log.Printf("Warning: %v", initErr)
			return
		}

		// Run migrations
		if err := s.runMigrations(); err != nil {
			initErr = fmt.Errorf("failed to run migrations: %w", err)
			s.enabled = false
			log.Printf("Warning: %v", initErr)
			return
		}
	})

	return initErr
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	if !s.enabled || s.db == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	s.db = nil
	return nil
}

// HashQuery creates a SHA256 hash of a query string for privacy.
func HashQuery(query string) string {
	hash := sha256.Sum256([]byte(query))
	return hex.EncodeToString(hash[:])
}
