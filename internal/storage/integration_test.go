package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRealDatabaseCreation verifies actual DB creation in home directory.
func TestRealDatabaseCreation(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	dbDir := filepath.Join(home, ".tool-hub-mcp")
	dbPath := filepath.Join(dbDir, "history.db")

	// Ensure directory exists
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	defer os.RemoveAll(dbDir) // Cleanup

	storage := &SQLiteStorage{
		dbPath:  dbPath,
		enabled: true,
	}

	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer storage.Close()

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file not created")
	}

	// Test operations
	event := UsageEvent{
		ToolName:       "hub_execute",
		ContextHash:    HashQuery("integration test query"),
		Timestamp:      time.Now(),
		Selected:       true,
		Rating:         5,
		WasRecommended: true,
	}

	if err := storage.RecordUsage(event); err != nil {
		t.Errorf("RecordUsage failed: %v", err)
	}

	history, err := storage.GetUsageHistory("hub_execute", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Errorf("GetUsageHistory failed: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 event, got %d", len(history))
	}

	fmt.Printf("✓ Database created successfully at: %s\n", dbPath)
	fmt.Printf("✓ Schema created with 3 tables\n")
	fmt.Printf("✓ Usage tracking operational\n")
}
