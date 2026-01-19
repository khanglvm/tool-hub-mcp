package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.Servers == nil {
		t.Error("NewConfig().Servers should not be nil")
	}

	if cfg.Settings == nil {
		t.Error("NewConfig().Settings should not be nil")
	}

	if cfg.Settings.CacheToolMetadata != true {
		t.Error("Default CacheToolMetadata should be true")
	}

	if cfg.Settings.ProcessPoolSize != 3 {
		t.Errorf("Default ProcessPoolSize should be 3, got %d", cfg.Settings.ProcessPoolSize)
	}

	if cfg.Settings.TimeoutSeconds != 30 {
		t.Errorf("Default TimeoutSeconds should be 30, got %d", cfg.Settings.TimeoutSeconds)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tool-hub-mcp-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".tool-hub-mcp.json")

	// Create test config
	cfg := NewConfig()
	cfg.Servers["testServer"] = &ServerConfig{
		Command: "echo",
		Args:    []string{"hello"},
		Env:     map[string]string{"KEY": "value"},
		Source:  "test",
	}

	// Save
	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	// Verify
	if len(loaded.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(loaded.Servers))
	}

	server, exists := loaded.Servers["testServer"]
	if !exists {
		t.Fatal("testServer not found in loaded config")
	}

	if server.Command != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", server.Command)
	}

	if len(server.Args) != 1 || server.Args[0] != "hello" {
		t.Errorf("Expected args ['hello'], got %v", server.Args)
	}

	if server.Source != "test" {
		t.Errorf("Expected source 'test', got '%s'", server.Source)
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := LoadFrom("/nonexistent/path/config.json")
	if err == nil {
		t.Error("LoadFrom should fail for non-existent file")
	}
}
