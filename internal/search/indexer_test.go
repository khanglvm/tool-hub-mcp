package search

import (
	"testing"

	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
)

func TestNewIndexer(t *testing.T) {
	indexer, err := NewIndexer()
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}
	defer indexer.Close()

	if indexer == nil {
		t.Fatal("indexer is nil")
	}
}

func TestIndexServer(t *testing.T) {
	indexer, err := NewIndexer()
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}
	defer indexer.Close()

	// Create mock tools
	tools := []spawner.Tool{
		{
			Name:        "test_tool",
			Description: "A test tool for searching",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "create_ticket",
			Description: "Create a Jira ticket",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
	}

	// Index the tools
	err = indexer.IndexServer("test-server", tools)
	if err != nil {
		t.Fatalf("failed to index server: %v", err)
	}

	// Verify count
	count, err := indexer.Count()
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 indexed tools, got %d", count)
	}
}

func TestSearchBM25(t *testing.T) {
	indexer, err := NewIndexer()
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}
	defer indexer.Close()

	// Index test tools
	tools := []spawner.Tool{
		{
			Name:        "create_jira_ticket",
			Description: "Create a new Jira ticket",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "take_screenshot",
			Description: "Take a screenshot of the current page",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "search_documents",
			Description: "Search documents in the outline",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
	}

	err = indexer.IndexServer("test-server", tools)
	if err != nil {
		t.Fatalf("failed to index server: %v", err)
	}

	// Test search for "jira"
	results, err := indexer.SearchBM25("jira", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected at least one result for 'jira'")
	}

	// Verify the first result is create_jira_ticket
	if results[0].ToolName != "create_jira_ticket" {
		t.Errorf("expected first result to be 'create_jira_ticket', got '%s'", results[0].ToolName)
	}

	// Test search for "screenshot"
	results, err = indexer.SearchBM25("screenshot", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected at least one result for 'screenshot'")
	}

	if results[0].ToolName != "take_screenshot" {
		t.Errorf("expected first result to be 'take_screenshot', got '%s'", results[0].ToolName)
	}
}

func TestSearchBM25NoResults(t *testing.T) {
	indexer, err := NewIndexer()
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}
	defer indexer.Close()

	// Index test tools
	tools := []spawner.Tool{
		{
			Name:        "create_ticket",
			Description: "Create a ticket",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
	}

	err = indexer.IndexServer("test-server", tools)
	if err != nil {
		t.Fatalf("failed to index server: %v", err)
	}

	// Search for something that doesn't exist
	results, err := indexer.SearchBM25("nonexistent_tool_xyz", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for non-existent query, got %d", len(results))
	}
}

func TestGetAllTools(t *testing.T) {
	indexer, err := NewIndexer()
	if err != nil {
		t.Fatalf("failed to create indexer: %v", err)
	}
	defer indexer.Close()

	// Index test tools
	tools := []spawner.Tool{
		{
			Name:        "tool1",
			Description: "Tool 1",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "tool2",
			Description: "Tool 2",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
	}

	err = indexer.IndexServer("test-server", tools)
	if err != nil {
		t.Fatalf("failed to index server: %v", err)
	}

	// Get all tools
	results, err := indexer.GetAllTools(10)
	if err != nil {
		t.Fatalf("get all tools failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 tools, got %d", len(results))
	}
}
