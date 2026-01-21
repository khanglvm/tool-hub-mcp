package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/learning"
	"github.com/khanglvm/tool-hub-mcp/internal/search"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
)

// TestSearchWorkflow tests the complete search workflow
func TestSearchWorkflow(t *testing.T) {
	// Create test config with mock server
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"test-server": {
				Command: "echo",
				Args:    []string{"test"},
			},
		},
		Settings: &config.Settings{
			ProcessPoolSize: 1,
		},
	}

	// Create server
	server := NewServer(cfg)
	defer func() {
		if server.tracker != nil {
			server.tracker.Stop()
		}
		if server.storage != nil {
			server.storage.Close()
		}
	}()

	// Index tools
	if err := server.IndexTools(); err != nil {
		t.Logf("Warning: IndexTools failed: %v", err)
	}

	// Verify indexer has tools
	if server.indexer != nil {
		count, err := server.indexer.Count()
		if err != nil {
			t.Errorf("failed to get count: %v", err)
		}
		t.Logf("Indexed %d tools", count)
	}
}

// TestServerInitialization tests server creation with various configs
func TestServerInitialization(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		wantNil  bool
	}{
		{
			name: "nil config",
			config: &config.Config{
				Servers: map[string]*config.ServerConfig{},
			},
			wantNil: false,
		},
		{
			name: "with servers",
			config: &config.Config{
				Servers: map[string]*config.ServerConfig{
					"test": {
						Command: "echo",
					},
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.config)
			defer func() {
				if server != nil {
					if server.storage != nil {
						server.storage.Close()
					}
					if server.tracker != nil {
						server.tracker.Stop()
					}
				}
			}()

			if (server == nil) != tt.wantNil {
				t.Errorf("NewServer() = %v, wantNil %v", server, tt.wantNil)
			}
		})
	}
}

// TestLearningIntegration tests that learning system works end-to-end
func TestLearningIntegration(t *testing.T) {
	// Create server with learning enabled
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
		Settings: &config.Settings{
			ProcessPoolSize: 1,
		},
	}

	server := NewServer(cfg)
	defer func() {
		if server.storage != nil {
			server.storage.Close()
		}
		if server.tracker != nil {
			server.tracker.Stop()
		}
	}()

	// Verify tracker was created
	if server.tracker == nil {
		t.Error("expected tracker to be created")
		return
	}

	// Verify tracker is enabled
	if !server.tracker.IsEnabled() {
		t.Error("expected tracker to be enabled")
	}

	// Track a usage event
	event := learning.UsageEvent{
		ToolName:  "test_tool",
		Timestamp: time.Now(),
	}

	server.tracker.Track(event)

	// Give time for background processing
	time.Sleep(200 * time.Millisecond)

	// Verify storage has the event
	if server.storage != nil {
		history, err := server.storage.GetUsageHistory("test_tool", time.Now().Add(-1*time.Hour))
		if err != nil {
			t.Errorf("failed to get history: %v", err)
		}

		if len(history) == 0 {
			t.Error("expected event to be recorded")
		}
	}
}

// TestIndexerIntegration tests search indexing workflow
func TestIndexerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
		Settings: &config.Settings{
			ProcessPoolSize: 1,
		},
	}

	server := NewServer(cfg)
	defer func() {
		if server.storage != nil {
			server.storage.Close()
		}
		if server.tracker != nil {
			server.tracker.Stop()
		}
	}()

	if server.indexer == nil {
		t.Skip("indexer not available")
	}

	// Index mock tools
	tools := []spawner.Tool{
		{
			Name:        "create_issue",
			Description: "Create a Jira issue",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		{
			Name:        "search_issues",
			Description: "Search for Jira issues",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
	}

	err := server.indexer.IndexServer("jira", tools)
	if err != nil {
		t.Fatalf("failed to index tools: %v", err)
	}

	// Search for tools
	results, err := server.indexer.SearchBM25("create issue", 10)
	if err != nil {
		t.Errorf("SearchBM25 failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected search results")
	}

	// Verify first result is relevant
	if len(results) > 0 && results[0].ToolName != "create_issue" {
		t.Errorf("expected 'create_issue', got '%s'", results[0].ToolName)
	}

	// Test search by server
	results, err = server.indexer.SearchByServer("create", "jira", 10)
	if err != nil {
		t.Errorf("SearchByServer failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected search results for server-scoped search")
	}
}

// TestHybridSearchIntegration tests hybrid search functionality
func TestHybridSearchIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
		Settings: &config.Settings{
			ProcessPoolSize: 1,
		},
	}

	server := NewServer(cfg)
	defer func() {
		if server.storage != nil {
			server.storage.Close()
		}
		if server.tracker != nil {
			server.tracker.Stop()
		}
	}()

	if server.indexer == nil {
		t.Skip("indexer not available")
	}

	// Index tools
	tools := []spawner.Tool{
		{
			Name:        "create_ticket",
			Description: "Create a support ticket",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
	}

	err := server.indexer.IndexServer("support", tools)
	if err != nil {
		t.Fatalf("failed to index tools: %v", err)
	}

	// Test hybrid search (should fallback to BM25 since semantic not implemented)
	results, err := server.indexer.SearchHybrid("create ticket", 10, search.DefaultFusionConfig)
	if err != nil {
		t.Errorf("SearchHybrid failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected hybrid search results")
	}
}

// TestGracefulDegradation tests that the system works with missing components
func TestGracefulDegradation(t *testing.T) {
	// Server should work even with nil storage
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
	}

	server := NewServer(cfg)
	defer func() {
		if server.storage != nil {
			server.storage.Close()
		}
		if server.tracker != nil {
			server.tracker.Stop()
		}
	}()

	// Server should still be functional
	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	// Operations should not crash
	if server.indexer != nil {
		_, _ = server.indexer.SearchBM25("test", 10)
	}

	if server.tracker != nil {
		// Should not panic
		server.tracker.Track(learning.UsageEvent{
			ToolName:  "test",
			Timestamp: time.Now(),
		})
	}
}

// TestConcurrentAccess tests concurrent access to server components
func TestConcurrentAccess(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
	}

	server := NewServer(cfg)
	defer func() {
		if server.tracker != nil {
			server.tracker.Stop()
		}
		if server.storage != nil {
			server.storage.Close()
		}
	}()

	// Concurrent tracking
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			if server.tracker != nil {
				for j := 0; j < 100; j++ {
					server.tracker.Track(learning.UsageEvent{
						ToolName:  "test",
						Timestamp: time.Now(),
					})
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Give tracker time to flush
	time.Sleep(200 * time.Millisecond)

	// Should not have crashed
	t.Log("Concurrent access test passed")
}
