package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Test successful atomic write
	data := []byte(`{"test": "data"}`)
	err := atomicWrite(testPath, data)
	if err != nil {
		t.Fatalf("atomicWrite failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Verify temp file was cleaned up
	tmpPath := testPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file was not cleaned up")
	}

	// Verify content
	readData, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(readData) != string(data) {
		t.Errorf("content mismatch: got %q, want %q", string(readData), string(data))
	}
}

func TestAtomicWriteCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "subdir", "config.json")

	data := []byte(`{"test": "data"}`)
	err := atomicWrite(testPath, data)
	if err != nil {
		t.Fatalf("atomicWrite failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestBackupConfig(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Create original config
	originalData := []byte(`{"original": true}`)
	if err := os.WriteFile(testPath, originalData, 0644); err != nil {
		t.Fatalf("failed to create original config: %v", err)
	}

	// Create backup
	err := backupConfig(testPath)
	if err != nil {
		t.Fatalf("backupConfig failed: %v", err)
	}

	// Verify backup exists
	bakPath := testPath + ".bak"
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}

	// Verify backup content matches original
	if string(bakData) != string(originalData) {
		t.Errorf("backup content mismatch: got %q, want %q", string(bakData), string(originalData))
	}

	// Verify backup permissions
	info, err := os.Stat(bakPath)
	if err != nil {
		t.Fatalf("failed to stat backup: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("backup permissions incorrect: got %v, want 0644", info.Mode().Perm())
	}
}

func TestBackupConfigFirstRun(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// No original config - should not error
	err := backupConfig(testPath)
	if err != nil {
		t.Fatalf("backupConfig failed on first run: %v", err)
	}

	// Verify no backup was created
	bakPath := testPath + ".bak"
	if _, err := os.Stat(bakPath); !os.IsNotExist(err) {
		t.Error("backup should not exist on first run")
	}
}

func TestValidateJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			data:    []byte(`{"servers": {"test": {"command": "node"}}}`),
			wantErr: false,
		},
		{
			name:    "missing servers field",
			data:    []byte(`{"settings": {}}`),
			wantErr: true,
			errMsg:  "missing 'servers' field",
		},
		{
			name:    "empty server command",
			data:    []byte(`{"servers": {"test": {"command": ""}}}`),
			wantErr: true,
			errMsg:  "empty command field",
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{invalid json}`),
			wantErr: true,
			errMsg:  "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSON(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestSaveCreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Create initial config
	cfg := NewConfig()
	cfg.Servers["test"] = &ServerConfig{
		Command: "node",
		Args:    []string{"server.js"},
	}

	// First save
	if err := Save(cfg, testPath); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	// Modify config
	cfg.Servers["test"].Args = []string{"server.js", "--updated"}

	// Second save (should create backup)
	if err := Save(cfg, testPath); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}

	// Verify backup exists
	bakPath := testPath + ".bak"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}

	// Verify backup has old content
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if !contains(string(bakData), `"server.js"`) || contains(string(bakData), `"--updated"`) {
		t.Error("backup should contain old config, not new config")
	}
}

func TestSaveValidatesBeforeWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Create invalid config (empty command)
	cfg := NewConfig()
	cfg.Servers["invalid"] = &ServerConfig{
		Command: "",
	}

	// Save should fail
	err := Save(cfg, testPath)
	if err == nil {
		t.Error("Save should fail validation for empty command")
	}
	if !contains(err.Error(), "invalid config") {
		t.Errorf("error should mention invalid config, got: %v", err)
	}

	// Verify no file was created
	if _, err := os.Stat(testPath); !os.IsNotExist(err) {
		t.Error("config file should not exist after failed validation")
	}
}

func TestSaveConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent write test in short mode")
	}

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cfg := NewConfig()
			cfg.Servers["server"] = &ServerConfig{
				Command: "node",
				Args:    []string{"--id", string(rune('0' + idx))},
			}
			if err := Save(cfg, testPath); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors - some failures are expected in concurrent scenario
	errorCount := 0
	for err := range errors {
		t.Logf("concurrent save error: %v", err)
		errorCount++
	}

	// Critical: verify final file is valid JSON (not corrupted)
	data, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("failed to read config after concurrent writes: %v", err)
	}

	if err := validateJSON(data); err != nil {
		t.Errorf("config file is corrupted after concurrent writes: %v", err)
	}

	// Verify backup exists (at least one successful write should have created it)
	bakPath := testPath + ".bak"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		t.Log("backup does not exist (all writes may have failed)")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
