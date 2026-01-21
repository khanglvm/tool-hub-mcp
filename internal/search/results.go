/*
Package search implements semantic search across all MCP server tools.

This package provides BM25-based keyword search with optional semantic
search via embeddings and hybrid fusion for ranked results.
*/
package search

// SearchResult represents a single search result with relevance score.
type SearchResult struct {
	ToolName    string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
	ServerName  string      `json:"server"`
	Score       float64     `json:"score"`
}

// ToolDocument represents a tool as stored in the search index.
type ToolDocument struct {
	ID          string
	Name        string
	Description string
	ServerName  string
	InputSchema interface{}
}
