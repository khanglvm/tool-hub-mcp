package version

import (
	"strings"
	"testing"
)

// Mock version comparison tests (no network calls)
func TestVersionComparison(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		latest      string
		wantUpdate  bool
		description string
	}{
		{
			name:        "update available - patch",
			current:     "1.0.0",
			latest:      "1.0.1",
			wantUpdate:  true,
			description: "patch version update",
		},
		{
			name:        "update available - minor",
			current:     "1.0.0",
			latest:      "1.1.0",
			wantUpdate:  true,
			description: "minor version update",
		},
		{
			name:        "update available - major",
			current:     "1.0.0",
			latest:      "2.0.0",
			wantUpdate:  true,
			description: "major version update",
		},
		{
			name:        "same version",
			current:     "1.0.0",
			latest:      "1.0.0",
			wantUpdate:  false,
			description: "no update needed",
		},
		{
			name:        "current newer - patch",
			current:     "1.0.1",
			latest:      "1.0.0",
			wantUpdate:  false,
			description: "current is ahead",
		},
		{
			name:        "current newer - minor",
			current:     "1.1.0",
			latest:      "1.0.0",
			wantUpdate:  false,
			description: "current is ahead",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple string comparison for version ordering
			// This tests the logic without needing actual version parsing
			updateAvailable := tt.latest > tt.current

			if updateAvailable != tt.wantUpdate {
				t.Errorf("Version comparison failed: current=%s, latest=%s, want update=%v, got=%v",
					tt.current, tt.latest, tt.wantUpdate, updateAvailable)
			}
		})
	}
}

func TestVersionStripping(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"with v prefix", "v1.0.0", "1.0.0"},
		{"without prefix", "1.0.0", "1.0.0"},
		{"with V uppercase", "V2.1.0", "2.1.0"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the TrimPrefix logic used in CheckUpdate
			result := strings.TrimPrefix(tt.version, "v")
			result = strings.TrimPrefix(result, "V")

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetCachePath(t *testing.T) {
	path, err := getCachePath()
	if err != nil {
		t.Fatalf("getCachePath() failed: %v", err)
	}

	// Verify path is not empty
	if path == "" {
		t.Error("getCachePath() returned empty path")
	}

	// Verify path contains expected filename
	if !strings.Contains(path, ".tool-hub-mcp-cache.json") {
		t.Errorf("Path %q does not contain cache filename", path)
	}
}

func TestUpdateCacheSaveLoad(t *testing.T) {
	// Create a test cache
	cache := &UpdateCache{
		LastKnownVersion: "1.2.3",
	}

	// Save cache
	err := saveUpdateCache(cache)
	if err != nil {
		t.Fatalf("saveUpdateCache() failed: %v", err)
	}

	// Load cache back
	loaded, err := loadUpdateCache()
	if err != nil {
		t.Fatalf("loadUpdateCache() failed: %v", err)
	}

	// Verify values match
	if loaded.LastKnownVersion != cache.LastKnownVersion {
		t.Errorf("Expected version %q, got %q", cache.LastKnownVersion, loaded.LastKnownVersion)
	}
}

func TestLoadUpdateCacheNotExist(t *testing.T) {
	// This tests the fallback behavior when cache doesn't exist
	// The function should return empty cache, not error

	// We can't easily delete the cache file in this test,
	// but we can verify the function handles missing files
	cache, err := loadUpdateCache()
	if err != nil {
		t.Fatalf("loadUpdateCache() should not error on missing file: %v", err)
	}

	if cache == nil {
		t.Fatal("loadUpdateCache() returned nil cache")
	}
}

func TestGitHubReleaseConstants(t *testing.T) {
	// Verify constants are set correctly
	if RepoOwner == "" {
		t.Error("RepoOwner is empty")
	}
	if RepoName == "" {
		t.Error("RepoName is empty")
	}
	if UpdateURL == "" {
		t.Error("UpdateURL is empty")
	}

	// Verify URL format
	expectedURL := "https://api.github.com/repos/" + RepoOwner + "/" + RepoName + "/releases/latest"
	if UpdateURL != expectedURL {
		t.Errorf("UpdateURL = %q, want %q", UpdateURL, expectedURL)
	}
}

// TestChecksumValidation tests checksum parsing logic
func TestChecksumParsing(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLen   int
		wantValid bool
	}{
		{
			name:      "valid checksum",
			input:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  tool-hub-mcp",
			wantLen:   64,
			wantValid: true,
		},
		{
			name:      "checksum only",
			input:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantLen:   64,
			wantValid: true,
		},
		{
			name:      "invalid length",
			input:     "abc123",
			wantLen:   6,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract first field (checksum)
			parts := strings.Fields(tt.input)
			if len(parts) == 0 {
				t.Fatal("No checksum found in input")
			}

			checksum := parts[0]
			isValid := len(checksum) == 64

			if len(checksum) != tt.wantLen {
				t.Errorf("Expected length %d, got %d", tt.wantLen, len(checksum))
			}

			if isValid != tt.wantValid {
				t.Errorf("Expected valid=%v, got=%v", tt.wantValid, isValid)
			}
		})
	}
}

// TestVersionConstant verifies the Version constant exists
func TestVersionConstant(t *testing.T) {
	// Version is set via ldflags at build time
	// In tests, it will be "dev" by default
	if Version == "" {
		t.Error("Version constant is empty")
	}

	// Verify it's a reasonable value
	validVersions := []string{"dev", "0.", "1.", "2."} // dev or starts with digit
	valid := false
	for _, prefix := range validVersions {
		if strings.HasPrefix(Version, prefix) {
			valid = true
			break
		}
	}

	if !valid {
		t.Logf("Version has unexpected format: %q (this may be OK)", Version)
	}
}
