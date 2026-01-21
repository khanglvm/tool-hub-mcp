package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnhancedErrorMessages tests that error messages are helpful
func TestEnhancedErrorMessages(t *testing.T) {
	t.Run("config_not_found_has_hint", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "not-found.json")

		_, err := LoadFrom(testPath)
		if err == nil {
			t.Fatal("LoadFrom should error for missing file")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "💡") {
			t.Errorf("error should contain helpful hint, got: %v", err)
		}
		if !strings.Contains(errMsg, "setup") {
			t.Errorf("error should mention setup command, got: %v", err)
		}
	})

	t.Run("permission_error_has_fix", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "readonly.json")
		os.WriteFile(testPath, []byte(`{"servers": {}}`), 0000)
		defer os.Chmod(testPath, 0644)

		_, err := LoadFrom(testPath)
		if err == nil {
			t.Fatal("LoadFrom should error for permission denied")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "permission denied") {
			t.Errorf("error should mention permission, got: %v", err)
		}
		if !strings.Contains(errMsg, "💡 Fix:") {
			t.Errorf("error should contain fix hint, got: %v", err)
		}
		// Check for platform-specific fix
		if !strings.Contains(errMsg, "chmod") && !strings.Contains(errMsg, "Properties → Security") {
			t.Errorf("error should contain platform-specific fix, got: %v", err)
		}
	})

	t.Run("invalid_json_mentions_backup", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "invalid.json")
		os.WriteFile(testPath, []byte(`{invalid json}`), 0644)

		_, err := LoadFrom(testPath)
		if err == nil {
			t.Fatal("LoadFrom should error for invalid JSON")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "invalid") {
			t.Errorf("error should mention invalid JSON, got: %v", err)
		}
		if !strings.Contains(errMsg, ".bak") {
			t.Errorf("error should mention backup file, got: %v", err)
		}
		if !strings.Contains(errMsg, "💡") {
			t.Errorf("error should contain helpful hint, got: %v", err)
		}
	})

	t.Run("write_permission_checked_before_save", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "readonly-save.json")
		os.WriteFile(testPath, []byte(`{"servers": {}}`), 0000)
		defer os.Chmod(testPath, 0644)

		cfg := NewConfig()
		err := Save(cfg, testPath)
		if err == nil {
			t.Fatal("Save should error for read-only file")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "permission denied") {
			t.Errorf("error should mention permission, got: %v", err)
		}
		if !strings.Contains(errMsg, "💡 Fix:") {
			t.Errorf("error should contain fix hint, got: %v", err)
		}
	})
}
