package config

import "fmt"

// PermissionError represents a permission-related config error
type PermissionError struct {
	Path    string
	Op      string // "read" or "write"
	Fix     string // Suggested fix command
	Details string // Additional context
}

func (e *PermissionError) Error() string {
	msg := fmt.Sprintf("permission denied (cannot %s config): %s\n", e.Op, e.Path)
	if e.Details != "" {
		msg += e.Details + "\n"
	}
	msg += "💡 Fix: " + e.Fix
	return msg
}

// ConfigNotFoundError represents missing config file
type ConfigNotFoundError struct {
	Path string
	Hint string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf("config file not found: %s\n\n💡 %s", e.Path, e.Hint)
}

// InvalidConfigError represents malformed config
type InvalidConfigError struct {
	Path    string
	Message string
	Hint    string
}

func (e *InvalidConfigError) Error() string {
	msg := fmt.Sprintf("invalid config: %s\n", e.Path)
	if e.Message != "" {
		msg += e.Message + "\n"
	}
	if e.Hint != "" {
		msg += "💡 " + e.Hint
	}
	return msg
}
