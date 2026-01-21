/*
Package learning implements usage tracking and tool ranking with ε-greedy bandit.

This package provides background tracking of tool usage, scoring algorithms
for ranking tools by frequency/recency/ratings, and exploration strategies
for discovering optimal tool selections.
*/
package learning

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

// UsageEvent represents a tool invocation with context for learning.
type UsageEvent struct {
	// ToolName is the name of the tool that was invoked.
	ToolName string

	// ContextHash is the SHA256 hash of the user's query/context for privacy.
	ContextHash string

	// Timestamp is when the tool was invoked.
	Timestamp time.Time

	// Selected indicates whether the tool was selected (true) or just shown (false).
	Selected bool

	// Rating is the user's feedback rating (1-5), or 0 if not rated.
	Rating int

	// WasRecommended indicates if the tool was recommended by the learning system.
	WasRecommended bool

	// SearchID is the search session identifier (optional).
	SearchID string
}

// NewUsageEvent creates a new usage event for tracking.
func NewUsageEvent(toolName, context string, selected bool, rating int, wasRecommended bool, searchID string) UsageEvent {
	return UsageEvent{
		ToolName:        toolName,
		ContextHash:     hashContext(context),
		Timestamp:       time.Now(),
		Selected:        selected,
		Rating:         rating,
		WasRecommended: wasRecommended,
		SearchID:       searchID,
	}
}

// ToStorage converts learning event to storage model.
func (e UsageEvent) ToStorage() storage.UsageEvent {
	return storage.UsageEvent{
		ToolName:        e.ToolName,
		ContextHash:     e.ContextHash,
		Timestamp:       e.Timestamp,
		Selected:        e.Selected,
		Rating:         e.Rating,
		WasRecommended: e.WasRecommended,
	}
}

// hashContext creates a SHA256 hash of context for privacy.
func hashContext(context string) string {
	if context == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(context))
	return hex.EncodeToString(hash[:])
}
