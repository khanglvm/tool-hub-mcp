package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromEnhancedErrors(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "nonexistent.json")

		_, err := LoadFrom(testPath)
		if err == nil {
			t.Fatal("LoadFrom should error for nonexistent file")
		}
		if !strings.Contains(err.Error(), "config file not found") {
			t.Errorf("error should mention file not found, got: %v", err)
		}
		if !strings.Contains(err.Error(), "tool-hub-mcp setup") {
			t.Errorf("error should mention setup command, got: %v", err)
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "config.json")

		// Create file with no read permissions
		if err := os.WriteFile(testPath, []byte(`{"servers": {}}`), 0000); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := LoadFrom(testPath)
		if err == nil {
			t.Fatal("LoadFrom should error for permission denied")
		}
		if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("error should mention permission denied, got: %v", err)
		}
		if !strings.Contains(err.Error(), "chmod 644") {
			t.Errorf("error should suggest chmod fix, got: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "config.json")

		// Create file with invalid JSON
		if err := os.WriteFile(testPath, []byte(`{invalid json}`), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := LoadFrom(testPath)
		if err == nil {
			t.Fatal("LoadFrom should error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("error should mention invalid JSON, got: %v", err)
		}
		if !strings.Contains(err.Error(), ".bak") {
			t.Errorf("error should mention .bak file, got: %v", err)
		}
	})
}

func TestLoadFromInitializesNilMaps(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Create config with empty servers
	if err := os.WriteFile(testPath, []byte(`{"servers": {}}`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg, err := LoadFrom(testPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if cfg.Servers == nil {
		t.Error("Servers map should be initialized, not nil")
	}
}

func TestLoadFromValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	validJSON := `{
		"servers": {
			"testServer": {
				"command": "node",
				"args": ["server.js"],
				"env": {"NODE_ENV": "production"},
				"source": "manual"
			}
		},
		"settings": {
			"cacheToolMetadata": true,
			"processPoolSize": 5,
			"timeoutSeconds": 60
		}
	}`

	if err := os.WriteFile(testPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg, err := LoadFrom(testPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	// Verify servers
	if len(cfg.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(cfg.Servers))
	}
	server := cfg.Servers["testServer"]
	if server.Command != "node" {
		t.Errorf("expected command 'node', got %q", server.Command)
	}
	if len(server.Args) != 1 || server.Args[0] != "server.js" {
		t.Errorf("unexpected args: %v", server.Args)
	}
	if server.Env["NODE_ENV"] != "production" {
		t.Errorf("unexpected env: %v", server.Env)
	}
	if server.Source != "manual" {
		t.Errorf("expected source 'manual', got %q", server.Source)
	}

	// Verify settings
	if cfg.Settings == nil {
		t.Fatal("Settings should not be nil")
	}
	if !cfg.Settings.CacheToolMetadata {
		t.Error("expected CacheToolMetadata to be true")
	}
	if cfg.Settings.ProcessPoolSize != 5 {
		t.Errorf("expected ProcessPoolSize 5, got %d", cfg.Settings.ProcessPoolSize)
	}
	if cfg.Settings.TimeoutSeconds != 60 {
		t.Errorf("expected TimeoutSeconds 60, got %d", cfg.Settings.TimeoutSeconds)
	}
}

func TestLoadFromWithEmptyCommand(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "config.json")

	// Config with empty command should load successfully
	// (validation happens during Save, not Load)
	invalidJSON := `{
		"servers": {
			"testServer": {
				"command": ""
			}
		}
	}`

	if err := os.WriteFile(testPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg, err := LoadFrom(testPath)
	if err != nil {
		t.Fatalf("LoadFrom should not fail on empty command: %v", err)
	}

	if cfg.Servers["testServer"].Command != "" {
		t.Errorf("expected empty command, got %q", cfg.Servers["testServer"].Command)
	}
}
