package config

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Save writes config with atomic write + backup
func Save(cfg *Config, path string) error {
	// Check write permissions before attempting write
	if err := checkWritePermission(path); err != nil {
		return err
	}

	// 1. Backup existing config
	if err := backupConfig(path); err != nil {
		// Log warning but continue (first run = no backup needed)
		fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)
	}

	// 2. Marshal JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 3. Validate JSON
	if err := validateJSON(data); err != nil {
		return &InvalidConfigError{
			Path:    path,
			Message: err.Error(),
			Hint:    "Check server configuration and try again",
		}
	}

	// 4. Atomic write
	return atomicWrite(path, data)
}

func backupConfig(path string) error {
	// Read existing file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // First run, no backup needed
		}
		return err
	}

	// Write to .bak
	bakPath := path + ".bak"
	return os.WriteFile(bakPath, data, 0644)
}

func validateJSON(data []byte) error {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// Check required fields
	if cfg.Servers == nil {
		return fmt.Errorf("missing 'servers' field")
	}

	// Validate each server config
	for name, srv := range cfg.Servers {
		if srv.Command == "" {
			return fmt.Errorf("server %s: empty command field", name)
		}
	}

	return nil
}

func atomicWrite(path string, data []byte) error {
	// Write to temp file in same directory
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpPath, path)
}

// checkWritePermission verifies we can write to the config path
func checkWritePermission(path string) error {
	dir := filepath.Dir(path)

	// Check if directory exists and is writable
	if err := checkDirectoryWritable(dir); err != nil {
		return &PermissionError{
			Path:    dir,
			Op:      "write",
			Fix:     getWritePermissionFix(dir),
			Details: "Cannot write to config directory",
		}
	}

	// If file exists, check if we can overwrite it
	if _, err := os.Stat(path); err == nil {
		if err := checkFileWritable(path); err != nil {
			return &PermissionError{
				Path:    path,
				Op:      "write",
				Fix:     getWritePermissionFix(path),
				Details: "Config file is read-only",
			}
		}
	}

	return nil
}

func checkDirectoryWritable(dir string) error {
	// Try to create a temp file in the directory
	tmpFile := dir + "/.write-test-" + randomString(8)
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	f.Close()
	os.Remove(tmpFile)
	return nil
}

func checkFileWritable(path string) error {
	// Try to open file with write access
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func getWritePermissionFix(path string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("Right-click %s → Properties → Security → Grant 'Write' permission", path)
	default: // unix-like
		return fmt.Sprintf("Run: chmod u+w %s", path)
	}
}

func randomString(n int) string {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Simple random string generator
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
