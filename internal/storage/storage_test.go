/*
Package storage provides tests for the storage layer.
*/
package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewStorage verifies storage initialization.
func TestNewStorage(t *testing.T) {
	storage := NewStorage()
	if storage == nil {
		t.Fatal("NewStorage returned nil")
	}

	// Clean up
	if storage.dbPath != "" {
		os.Remove(storage.dbPath)
	}
}

// TestInit verifies database initialization and schema creation.
func TestInit(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create storage with custom path
	storage := &SQLiteStorage{
		dbPath:  dbPath,
		enabled: true,
	}

	// Initialize
	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file not created")
	}

	// Clean up
	storage.Close()
}

// TestRecordUsage verifies recording usage events.
func TestRecordUsage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage := &SQLiteStorage{
		dbPath:  dbPath,
		enabled: true,
	}

	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer storage.Close()

	// Record usage
	event := UsageEvent{
		ToolName:       "hub_execute",
		ContextHash:    HashQuery("test query"),
		Timestamp:      time.Now(),
		Selected:       true,
		Rating:         5,
		WasRecommended: true,
	}

	if err := storage.RecordUsage(event); err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}

	// Verify retrieval
	history, err := storage.GetUsageHistory("hub_execute", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("GetUsageHistory failed: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 usage event, got %d", len(history))
	}

	if history[0].ToolName != "hub_execute" {
		t.Errorf("Expected tool_name 'hub_execute', got '%s'", history[0].ToolName)
	}
}

// TestHashQuery verifies query hashing consistency.
func TestHashQuery(t *testing.T) {
	query := "test query for hashing"

	hash1 := HashQuery(query)
	hash2 := HashQuery(query)

	if hash1 != hash2 {
		t.Error("HashQuery produced inconsistent results")
	}

	if len(hash1) != 64 { // SHA256 hex = 64 chars
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}

// TestGracefulDegradation verifies behavior when DB is unavailable.
func TestGracefulDegradation(t *testing.T) {
	// Create storage with invalid path
	storage := &SQLiteStorage{
		dbPath:  "/invalid/path/that/does/not/exist/test.db",
		enabled: true,
	}

	// Init should fail gracefully
	if err := storage.Init(); err != nil {
		// Expected to fail
	}

	// Operations should not panic
	event := UsageEvent{
		ToolName:    "test",
		ContextHash: "abc123",
		Timestamp:   time.Now(),
		Selected:    true,
	}

	if err := storage.RecordUsage(event); err != nil {
		t.Errorf("RecordUsage should return nil on disabled storage, got: %v", err)
	}

	history, err := storage.GetUsageHistory("test", time.Now())
	if err != nil {
		t.Errorf("GetUsageHistory should not error on disabled storage, got: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected empty history on disabled storage, got %d events", len(history))
	}
}
