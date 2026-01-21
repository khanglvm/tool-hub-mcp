/*
Package storage provides data models for the learning and history system.

These models represent tool usage events, search history, and cached embeddings
used by the learning system and semantic search functionality.
*/
package storage

import "time"

// UsageEvent represents a single tool invocation event.
type UsageEvent struct {
	// ToolName is the name of the tool that was invoked.
	ToolName string `json:"tool_name"`

	// ContextHash is the SHA256 hash of the user's query/context for privacy.
	ContextHash string `json:"context_hash"`

	// Timestamp is when the tool was invoked.
	Timestamp time.Time `json:"timestamp"`

	// Selected indicates whether the tool was selected (1) or just shown (0).
	Selected bool `json:"selected"`

	// Rating is the user's feedback rating (1-5), or 0 if not rated.
	Rating int `json:"rating"`

	// WasRecommended indicates if the tool was recommended by the learning system.
	WasRecommended bool `json:"was_recommended"`
}

// SearchRecord represents a search query for analytics.
type SearchRecord struct {
	// SearchID is a unique identifier for this search (UUID).
	SearchID string `json:"search_id"`

	// QueryHash is the SHA256 hash of the search query for privacy.
	QueryHash string `json:"query_hash"`

	// Timestamp is when the search was performed.
	Timestamp time.Time `json:"timestamp"`

	// ResultsCount is the number of results returned.
	ResultsCount int `json:"results_count"`
}

// ToolEmbedding represents a cached embedding vector for a tool.
type ToolEmbedding struct {
	// ToolName is the name of the tool.
	ToolName string `json:"tool_name"`

	// Vector is the embedding vector (serialized as JSON).
	Vector []float32 `json:"vector"`

	// Version is the model version used to generate the embedding.
	Version string `json:"version"`

	// CreatedAt is when the embedding was generated.
	CreatedAt time.Time `json:"created_at"`
}
