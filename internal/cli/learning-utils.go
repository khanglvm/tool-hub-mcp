/*
Package cli provides utility functions for the learning system.

These helpers support CLI commands with tracker creation and formatting.
*/
package cli

import (
	"encoding/json"

	"github.com/khanglvm/tool-hub-mcp/internal/learning"
	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

// NewTrackerForCLI creates a tracker instance for CLI usage.
func NewTrackerForCLI() *learning.Tracker {
	store := storage.NewStorage()
	return learning.NewTracker(store)
}

// formatJSON pretty-prints JSON for export.
func formatJSON(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
