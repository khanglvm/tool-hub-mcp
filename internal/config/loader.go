package config

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
)

// LoadFrom reads config with enhanced error handling
func LoadFrom(path string) (*Config, error) {
	// Check file existence first
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, &ConfigNotFoundError{
				Path: path,
				Hint: "Run 'tool-hub-mcp setup' to create configuration",
			}
		}
		return nil, fmt.Errorf("failed to access config: %w", err)
	}

	// Check read permissions
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsPermission(err) {
			return nil, &PermissionError{
				Path:    path,
				Op:      "read",
				Fix:     getReadPermissionFix(path),
				Details: getPermissionDetails(path),
			}
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, &InvalidConfigError{
			Path:    path,
			Message: fmt.Sprintf("JSON parse error: %v", err),
			Hint:    "Restore from .bak file if available",
		}
	}

	// Initialize nil maps
	if cfg.Servers == nil {
		cfg.Servers = make(map[string]*ServerConfig)
	}

	return &cfg, nil
}

// getReadPermissionFix returns platform-specific fix command
func getReadPermissionFix(path string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("Right-click %s → Properties → Security → Edit permissions", path)
	default: // unix-like
		return fmt.Sprintf("Run: chmod 644 %s", path)
	}
}

// getPermissionDetails checks file ownership and permissions
func getPermissionDetails(path string) string {
	if runtime.GOOS == "windows" {
		return "" // Not applicable on Windows
	}

	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	mode := info.Mode()
	perms := mode.Perm()

	return fmt.Sprintf("Current permissions: %04o", perms)
}
