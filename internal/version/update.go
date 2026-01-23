package version

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	RepoOwner = "khanglvm"
	RepoName  = "tool-hub-mcp"
	UpdateURL = "https://api.github.com/repos/" + RepoOwner + "/" + RepoName + "/releases/latest"
)

var (
	lastCheckTime time.Time
	checkMu       sync.Mutex
)

// GitHubRelease represents a GitHub release API response.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// UpdateCache stores update check state.
type UpdateCache struct {
	LastUpdateCheck  time.Time `json:"lastUpdateCheck"`
	LastKnownVersion string    `json:"lastKnownVersion"`
}

// CheckUpdate checks for new version (cached for 24h).
func CheckUpdate(ctx context.Context) (string, error) {
	checkMu.Lock()
	defer checkMu.Unlock()

	// Check cache
	cache, err := loadUpdateCache()
	if err == nil && time.Since(cache.LastUpdateCheck) < 24*time.Hour {
		// Already checked recently
		return "", nil
	}

	// Create HTTP request with timeout
	req, err := http.NewRequestWithContext(ctx, "GET", UpdateURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Make request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Strip 'v' prefix if present
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// Update cache
	cache.LastUpdateCheck = time.Now()
	cache.LastKnownVersion = latestVersion
	if err := saveUpdateCache(cache); err != nil {
		log.Printf("Warning: failed to save update cache: %v", err)
	}

	// If current version is different from latest
	if latestVersion != Version {
		return latestVersion, nil
	}

	return "", nil
}

// DownloadUpdate downloads new binary to temp location with SHA256 verification.
func DownloadUpdate(ctx context.Context, version string) (string, error) {
	// Determine binary name for platform
	binaryName := "tool-hub-mcp"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	// Get SHA256 checksum from release assets first
	checksumURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s.sha256",
		RepoOwner, RepoName, version, binaryName)

	expectedChecksum, err := fetchChecksum(ctx, checksumURL)
	if err != nil {
		log.Printf("Warning: could not fetch checksum, skipping verification: %v", err)
		expectedChecksum = ""
	}

	// Download URL
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s",
		RepoOwner, RepoName, version, binaryName)

	// Create temp file
	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, "tool-hub-mcp-"+version+"-"+binaryName)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Download
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file and calculate checksum while downloading
	hasher := sha256.New()
	out, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// Tee reader to calculate hash while writing
	multiWriter := io.MultiWriter(out, hasher)

	if _, err := io.Copy(multiWriter, resp.Body); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	// Verify checksum if we have one
	if expectedChecksum != "" {
		actualChecksum := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(actualChecksum, expectedChecksum) {
			os.Remove(tempPath)
			return "", fmt.Errorf("checksum verification failed: expected %s, got %s",
				expectedChecksum, actualChecksum)
		}
		log.Printf("Checksum verified: %s", actualChecksum)
	}

	// Make executable
	if err := os.Chmod(tempPath, 0755); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to make executable: %w", err)
	}

	return tempPath, nil
}

// fetchChecksum retrieves the SHA256 checksum from the checksum file.
func fetchChecksum(ctx context.Context, checksumURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum file not found (status %d)", resp.StatusCode)
	}

	// Checksum file format: "sha256  filename"
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract just the checksum (first 64 hex chars)
	checksum := strings.Fields(string(data))[0]
	if len(checksum) != 64 {
		return "", fmt.Errorf("invalid checksum format")
	}

	return checksum, nil
}

// ApplyUpdate atomically replaces binary with downloaded version.
func ApplyUpdate(tempPath string) error {
	// Get current binary path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Backup current binary
	backupPath := execPath + ".bak"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary to final location
	if err := os.Rename(tempPath, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(execPath, 0755); err != nil {
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	return nil
}

// getCachePath returns the path to the update cache file.
func getCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tool-hub-mcp-cache.json"), nil
}

// loadUpdateCache loads the update cache from disk.
func loadUpdateCache() (*UpdateCache, error) {
	cachePath, err := getCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &UpdateCache{}, nil
		}
		return nil, err
	}

	var cache UpdateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return &UpdateCache{}, nil
	}

	return &cache, nil
}

// saveUpdateCache saves the update cache to disk.
func saveUpdateCache(cache *UpdateCache) error {
	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}
