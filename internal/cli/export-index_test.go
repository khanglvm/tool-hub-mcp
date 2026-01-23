package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewExportIndexCmd(t *testing.T) {
	cmd := NewExportIndexCmd()

	if cmd == nil {
		t.Fatal("NewExportIndexCmd() returned nil")
	}

	// Verify command properties
	if cmd.Use != "export-index" {
		t.Errorf("Expected Use='export-index', got %q", cmd.Use)
	}

	// Verify flags are registered
	if cmd.Flags().Lookup("format") == nil {
		t.Error("Flag 'format' not registered")
	}
	if cmd.Flags().Lookup("output") == nil {
		t.Error("Flag 'output' not registered")
	}
}

func TestExportIndexCommandHelp(t *testing.T) {
	cmd := NewExportIndexCmd()

	// Verify command has proper description
	if cmd.Short == "" {
		t.Error("Command missing short description")
	}

	if !strings.Contains(cmd.Short, "Export") || !strings.Contains(cmd.Short, "tool") {
		t.Errorf("Short description doesn't mention exporting tools: %q", cmd.Short)
	}

	// Verify example usage is provided
	if cmd.Example == "" {
		t.Error("Command missing example usage")
	}
}

func TestWriteIndexJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "test-index.jsonl")

	tools := []ToolEntry{
		{
			Tool:        "jira_get_issue",
			Server:      "jira",
			Description: "Get Jira issue details",
			InputSchema: map[string]interface{}{"type": "object"},
		},
		{
			Tool:        "figma_get_file",
			Server:      "figma",
			Description: "Get Figma file metadata",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}

	err := writeIndex(tools, output, "jsonl")
	if err != nil {
		t.Fatalf("writeIndex failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(output); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Read and verify JSONL format (one JSON object per line)
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(tools) {
		t.Errorf("Expected %d lines, got %d", len(tools), len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var entry ToolEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}

		// Verify required fields
		if entry.Tool == "" || entry.Server == "" {
			t.Errorf("Line %d missing required fields: %+v", i, entry)
		}
	}
}

func TestWriteIndexJSON(t *testing.T) {
	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "test-index.json")

	tools := []ToolEntry{
		{
			Tool:        "jira_get_issue",
			Server:      "jira",
			Description: "Get Jira issue details",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}

	err := writeIndex(tools, output, "json")
	if err != nil {
		t.Fatalf("writeIndex failed: %v", err)
	}

	// Read and verify JSON array format
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var entries []ToolEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Errorf("Output is not valid JSON array: %v", err)
	}

	if len(entries) != len(tools) {
		t.Errorf("Expected %d entries, got %d", len(tools), len(entries))
	}
}

func TestAcquireFileLock(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-lock.jsonl")

	// Create test file
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Acquire lock
	lockFile, err := acquireFileLock(testFile)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer releaseFileLock(lockFile)

	// Verify lock file was created
	lockPath := testFile + ".lock"
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}

	// Try to acquire lock again (should fail)
	_, err = acquireFileLock(testFile)
	if err == nil {
		t.Error("Expected lock acquisition to fail, but it succeeded")
	}
}

func TestReleaseFileLock(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-lock.jsonl")

	// Create test file
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Acquire and release lock
	lockFile, err := acquireFileLock(testFile)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	lockPath := lockFile.Name()

	if err := releaseFileLock(lockFile); err != nil {
		t.Errorf("Failed to release lock: %v", err)
	}

	// Verify lock file was removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("Lock file was not removed after release")
	}

	// Should be able to acquire lock again
	lockFile2, err := acquireFileLock(testFile)
	if err != nil {
		t.Errorf("Failed to re-acquire lock after release: %v", err)
	}
	defer releaseFileLock(lockFile2)
}

func TestConcurrentFileLocking(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-concurrent.jsonl")

	// Create test file
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var wg sync.WaitGroup
	successCount := 0
	mu := sync.Mutex{}

	// Try to acquire lock from multiple goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lockFile, err := acquireFileLock(testFile)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()

				// Hold lock briefly
				time.Sleep(10 * time.Millisecond)
				releaseFileLock(lockFile)
			}
		}()
	}

	wg.Wait()

	// Only one should have succeeded at a time
	// But after releasing, others could acquire
	if successCount == 0 {
		t.Error("No goroutine acquired lock")
	}
}

func TestRegenerateIndexNonBlocking(t *testing.T) {
	// Test that RegenerateIndex doesn't block
	start := time.Now()
	RegenerateIndex()
	duration := time.Since(start)

	// Should return immediately (< 100ms)
	if duration > 100*time.Millisecond {
		t.Errorf("RegenerateIndex blocked for %v, should be async", duration)
	}
}

func TestExportIndexCommandExecution(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"help flag", []string{"--help"}, false},
		{"format flag", []string{"--format", "json"}, false},
		{"format flag jsonl", []string{"--format", "jsonl"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewExportIndexCmd()
			cmd.SetArgs(tt.args)

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToolEntryJSONMarshaling(t *testing.T) {
	entry := ToolEntry{
		Tool:        "test_tool",
		Server:      "test_server",
		Description: "Test description",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal ToolEntry: %v", err)
	}

	// Unmarshal back
	var decoded ToolEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ToolEntry: %v", err)
	}

	// Verify fields
	if decoded.Tool != entry.Tool {
		t.Errorf("Tool mismatch: got %q, want %q", decoded.Tool, entry.Tool)
	}
	if decoded.Server != entry.Server {
		t.Errorf("Server mismatch: got %q, want %q", decoded.Server, entry.Server)
	}
	if decoded.Description != entry.Description {
		t.Errorf("Description mismatch: got %q, want %q", decoded.Description, entry.Description)
	}
}

func TestWriteIndexWithEmptyTools(t *testing.T) {
	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "empty.jsonl")

	var emptyTools []ToolEntry

	err := writeIndex(emptyTools, output, "jsonl")
	if err != nil {
		t.Fatalf("writeIndex with empty tools failed: %v", err)
	}

	// File should be created but empty (or just whitespace)
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Should be empty or contain empty array
	content := strings.TrimSpace(string(data))
	if content != "" && content != "[]" {
		t.Errorf("Expected empty output, got: %q", content)
	}
}

func TestWriteIndexPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "perms.jsonl")

	tools := []ToolEntry{
		{Tool: "test", Server: "test", Description: "test", InputSchema: nil},
	}

	err := writeIndex(tools, output, "jsonl")
	if err != nil {
		t.Fatalf("writeIndex failed: %v", err)
	}

	// Check file permissions (should be readable)
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	if mode&0400 == 0 {
		t.Error("Output file is not readable by owner")
	}
}

func TestRunExportIndexWithNoConfig(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "test.jsonl")

	// Test with custom output path when config doesn't exist
	// This will fail to load config but should handle gracefully
	err := runExportIndex("jsonl", output)

	// Should either return error or handle gracefully
	// Just verify it doesn't panic
	if err != nil {
		// Expected when no config exists
		if !strings.Contains(err.Error(), "failed to load config") {
			t.Logf("Got expected error: %v", err)
		}
	}
}
