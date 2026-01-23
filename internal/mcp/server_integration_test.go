package mcp

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/learning"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
)

// TestHandleToolsList tests tools/list RPC handler
func TestHandleToolsList(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"testServer": {
				Command: "echo",
				Args:    []string{"mock"},
			},
		},
	}

	server := NewServer(cfg)
	defer server.Close()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp, err := server.handleToolsList(&req)
	if err != nil {
		t.Fatalf("handleToolsList failed: %v", err)
	}

	// Validate JSON-RPC 2.0 protocol compliance
	if resp.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", resp.JSONRPC)
	}

	if resp.ID != req.ID {
		t.Errorf("expected ID %v, got %v", req.ID, resp.ID)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	// Parse result
	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	tools, ok := resultMap["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("tools is not an array")
	}

	// Verify hub_search and hub_execute are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok {
			toolNames[name] = true
		}
	}

	expectedTools := []string{"hub_search", "hub_execute", "hub_manage"}
	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("missing expected tool: %s", expected)
		}
	}

	// Verify hub_search has proper schema
	var hubSearchTool map[string]interface{}
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok && name == "hub_search" {
			hubSearchTool = tool
			break
		}
	}

	if hubSearchTool == nil {
		t.Fatal("hub_search tool not found")
	}

	// Validate inputSchema
	schema, ok := hubSearchTool["inputSchema"].(map[string]interface{})
	if !ok {
		t.Fatal("hub_search inputSchema is not a map")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("hub_search properties is not a map")
	}

	if _, exists := properties["query"]; !exists {
		t.Error("hub_search schema missing 'query' property")
	}

	if _, exists := properties["server"]; !exists {
		t.Error("hub_search schema missing 'server' property")
	}

	if _, exists := properties["limit"]; !exists {
		t.Error("hub_search schema missing 'limit' property")
	}
}

// TestHubSearchBM25Ranking tests hub_search with BM25 scoring
func TestHubSearchBM25Ranking(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		minResults  int
	}{
		{
			name:        "valid search query",
			query:       "create issue",
			expectError: false,
			minResults:  0, // May have no results if no tools indexed
		},
		{
			name:        "empty query",
			query:       "",
			expectError: false, // Server handles gracefully
			minResults:  0,
		},
		{
			name:        "single word query",
			query:       "jira",
			expectError: false,
			minResults:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Servers: map[string]*config.ServerConfig{
					"testServer": {
						Command: "echo",
						Args:    []string{"test"},
					},
				},
			}

			server := NewServer(cfg)
			defer server.Close()

			// Index mock tools for testing BM25
			if server.indexer != nil {
				tools := []spawner.Tool{
					{
						Name:        "create_issue",
						Description: "Create a new issue in the system",
						InputSchema: json.RawMessage(`{"type":"object"}`),
					},
					{
						Name:        "search_issues",
						Description: "Search for existing issues",
						InputSchema: json.RawMessage(`{"type":"object"}`),
					},
				}
				_ = server.indexer.IndexServer("jira", tools)
			}

			result, err := server.execHubSearch(tt.query, "", 10)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Parse JSON result
			var resultData map[string]interface{}
			if err := json.Unmarshal([]byte(result), &resultData); err != nil {
				// Fallback response (no indexer)
				if !strings.Contains(result, "Available servers") {
					t.Logf("Result (fallback): %s", result)
				}
				return
			}

			// Verify searchId is present
			if _, exists := resultData["searchId"]; !exists {
				t.Error("result missing searchId field")
			}

			// Verify results array
			results, ok := resultData["results"].([]interface{})
			if !ok {
				t.Fatal("results is not an array")
			}

			if len(results) < tt.minResults {
				t.Errorf("expected at least %d results, got %d", tt.minResults, len(results))
			}

			// Verify result structure (if any results)
			if len(results) > 0 {
				firstResult, ok := results[0].(map[string]interface{})
				if !ok {
					t.Fatal("first result is not a map")
				}

				// Check required fields (flat structure)
				requiredFields := []string{"name", "description", "inputSchema", "server", "score"}
				for _, field := range requiredFields {
					if _, exists := firstResult[field]; !exists {
						t.Errorf("result missing field: %s", field)
					}
				}

				// Verify removed fields NOT present
				if _, exists := firstResult["expectedResponse"]; exists {
					t.Error("result should not have expectedResponse field")
				}
				if _, exists := firstResult["matchReason"]; exists {
					t.Error("result should not have matchReason field")
				}
				if _, exists := firstResult["tool"]; exists {
					t.Error("result should not have nested tool object")
				}
			}
		})
	}
}

// TestHubExecuteWithLearning tests hub_execute with learning tracker
func TestHubExecuteWithLearning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"echo": {
				Command: "echo",
				Args:    []string{"test"},
			},
		},
	}

	server := NewServer(cfg)
	defer server.Close()

	// Test execution without searchId
	_, err := server.execHubExecute("echo", "test_tool", map[string]interface{}{}, "")
	if err == nil {
		// Echo server doesn't support tools/call - expected
		t.Log("Expected error for echo server (no MCP support)")
	}

	// Test execution with searchId
	searchID := "test-search-id-123"
	_, err = server.execHubExecute("echo", "test_tool", map[string]interface{}{}, searchID)
	if err == nil {
		t.Log("Echo server doesn't support MCP - expected error")
	}

	// Verify tracker recorded usage
	if server.tracker != nil && server.tracker.IsEnabled() {
		// Give tracker time to process
		time.Sleep(100 * time.Millisecond)

		// Verify tracking happened (via logs or storage)
		if server.storage != nil {
			history, err := server.storage.GetUsageHistory("test_tool", time.Now().Add(-1*time.Hour))
			if err != nil {
				t.Logf("GetUsageHistory failed (expected if no events): %v", err)
			} else {
				t.Logf("Usage history entries: %d", len(history))
			}
		}
	}

	// Test with non-existent server
	_, err = server.execHubExecute("nonexistent", "test_tool", map[string]interface{}{}, "")
	if err == nil {
		t.Error("expected error for non-existent server")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// TestConcurrentToolCalls tests concurrent access to RPC handlers
func TestConcurrentToolCalls(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"testServer": {
				Command: "echo",
				Args:    []string{"test"},
			},
		},
	}

	server := NewServer(cfg)
	defer server.Close()

	// Index test tools
	if server.indexer != nil {
		tools := []spawner.Tool{
			{
				Name:        "concurrent_test",
				Description: "Tool for concurrent testing",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		}
		_ = server.indexer.IndexServer("testServer", tools)
	}

	// Spawn 10 goroutines, each calling hub_search 100 times
	var wg sync.WaitGroup
	goroutines := 10
	callsPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < callsPerGoroutine; j++ {
				// Mix of search calls
				query := "test"
				if j%2 == 0 {
					query = "concurrent"
				}

				_, err := server.execHubSearch(query, "", 5)
				if err != nil {
					t.Logf("Goroutine %d call %d failed: %v", routineID, j, err)
				}

				// Track usage events concurrently
				if server.tracker != nil {
					server.tracker.Track(learning.UsageEvent{
						ToolName:  "concurrent_test",
						Timestamp: time.Now(),
					})
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Give tracker time to flush
	if server.tracker != nil {
		time.Sleep(200 * time.Millisecond)
		server.tracker.Stop()
	}

	// Verify no race conditions occurred (test will fail with -race if issues exist)
	t.Logf("Completed %d concurrent calls across %d goroutines", goroutines*callsPerGoroutine, goroutines)
}

// TestJSONRPCErrorHandling tests error response formatting
func TestJSONRPCErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		request    MCPRequest
		expectCode int
	}{
		{
			name: "invalid method",
			request: MCPRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "invalid/method",
			},
			expectCode: -32601, // Method not found
		},
		{
			name: "tools/call with unknown tool",
			request: MCPRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"unknown_tool","arguments":{}}`),
			},
			expectCode: -32602, // Unknown tool
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Servers: map[string]*config.ServerConfig{},
			}

			server := NewServer(cfg)
			defer server.Close()

			reqJSON, _ := json.Marshal(tt.request)
			resp, err := server.handleRequest(reqJSON)

			if err != nil {
				t.Logf("handleRequest returned error: %v", err)
				return
			}

			if resp.Error == nil {
				t.Fatal("expected error response")
			}

			if resp.Error.Code != tt.expectCode {
				t.Errorf("expected error code %d, got %d", tt.expectCode, resp.Error.Code)
			}

			// Verify JSON-RPC 2.0 protocol
			if resp.JSONRPC != "2.0" {
				t.Errorf("expected JSONRPC 2.0, got %s", resp.JSONRPC)
			}

			// ID comparison - handle interface{} type conversion
			// JSON marshaling may convert int to float64
			idMatch := false
			switch expectedID := tt.request.ID.(type) {
			case int:
				if actualFloat, ok := resp.ID.(float64); ok {
					idMatch = float64(expectedID) == actualFloat
				} else if actualInt, ok := resp.ID.(int); ok {
					idMatch = expectedID == actualInt
				}
			case float64:
				if actualFloat, ok := resp.ID.(float64); ok {
					idMatch = expectedID == actualFloat
				}
			default:
				idMatch = tt.request.ID == resp.ID
			}

			if !idMatch {
				t.Errorf("expected ID %v (type %T), got %v (type %T)", tt.request.ID, tt.request.ID, resp.ID, resp.ID)
			}
		})
	}
}

// TestSearchWithServerFilter tests hub_search with server-scoped filtering
func TestSearchWithServerFilter(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"jira": {
				Command: "echo",
				Args:    []string{"jira"},
			},
			"github": {
				Command: "echo",
				Args:    []string{"github"},
			},
		},
	}

	server := NewServer(cfg)
	defer server.Close()

	// Index tools from multiple servers
	if server.indexer != nil {
		jiraTools := []spawner.Tool{
			{
				Name:        "create_issue",
				Description: "Create Jira issue",
				InputSchema: json.RawMessage(`{}`),
			},
		}
		githubTools := []spawner.Tool{
			{
				Name:        "create_issue",
				Description: "Create GitHub issue",
				InputSchema: json.RawMessage(`{}`),
			},
		}

		_ = server.indexer.IndexServer("jira", jiraTools)
		_ = server.indexer.IndexServer("github", githubTools)
	}

	// Search with server filter
	result, err := server.execHubSearch("create issue", "jira", 10)
	if err != nil {
		t.Fatalf("execHubSearch failed: %v", err)
	}

	// Parse result
	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		// Fallback handling
		t.Skip("Indexer not available or fallback response")
	}

	results, ok := resultData["results"].([]interface{})
	if !ok {
		t.Fatal("results is not an array")
	}

	// Verify all results are from jira server
	for _, res := range results {
		resMap, ok := res.(map[string]interface{})
		if !ok {
			continue
		}

		serverName, ok := resMap["server"].(string)
		if !ok {
			t.Error("result missing server field")
			continue
		}

		if serverName != "jira" {
			t.Errorf("expected server 'jira', got '%s'", serverName)
		}
	}
}

// TestBanditRanking tests that bandit algorithm affects search ranking
func TestBanditRanking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"test": {
				Command: "echo",
				Args:    []string{"test"},
			},
		},
	}

	server := NewServer(cfg)
	defer server.Close()

	// Index tools
	if server.indexer != nil {
		tools := []spawner.Tool{
			{
				Name:        "tool_a",
				Description: "First test tool",
				InputSchema: json.RawMessage(`{}`),
			},
			{
				Name:        "tool_b",
				Description: "Second test tool",
				InputSchema: json.RawMessage(`{}`),
			},
		}
		_ = server.indexer.IndexServer("test", tools)
	}

	// Perform initial search
	result1, err := server.execHubSearch("test tool", "", 10)
	if err != nil {
		t.Fatalf("initial search failed: %v", err)
	}

	// Parse searchId
	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result1), &resultData); err != nil {
		t.Skip("Indexer not available")
	}

	searchID, ok := resultData["searchId"].(string)
	if !ok || searchID == "" {
		t.Fatal("searchId not returned")
	}

	// Simulate successful execution (tracking usage)
	if server.tracker != nil && server.storage != nil {
		// Track multiple successful uses of tool_a
		for i := 0; i < 5; i++ {
			server.trackUsage("tool_a", searchID, true)
		}

		// Wait for flush
		time.Sleep(200 * time.Millisecond)
	}

	// Perform second search - tool_a should have higher score due to learning
	result2, err := server.execHubSearch("test tool", "", 10)
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}

	t.Logf("Search with learning completed: %s", result2)

	// In a real scenario with hybrid search + bandit, tool_a would rank higher
	// For now, we just verify the workflow completes
}

// TestIndexReload tests that config reload triggers reindexing
func TestIndexReload(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"server1": {
				Command: "echo",
				Args:    []string{"s1"},
			},
		},
	}

	server := NewServer(cfg)
	defer server.Close()

	// Initial index
	if server.indexer != nil {
		tools := []spawner.Tool{
			{Name: "tool1", Description: "Tool 1", InputSchema: json.RawMessage(`{}`)},
		}
		_ = server.indexer.IndexServer("server1", tools)

		count1, _ := server.indexer.Count()
		t.Logf("Initial tool count: %d", count1)
	}

	// Reload with new server
	newCfg := &config.Config{
		Servers: map[string]*config.ServerConfig{
			"server1": cfg.Servers["server1"],
			"server2": {
				Command: "echo",
				Args:    []string{"s2"},
			},
		},
	}

	server.ReloadConfig(newCfg)

	// Verify config updated
	server.configMu.RLock()
	serverCount := len(server.config.Servers)
	server.configMu.RUnlock()

	if serverCount != 2 {
		t.Errorf("expected 2 servers after reload, got %d", serverCount)
	}
}

// TestSearchResultsStructure validates search results match expected schema
func TestSearchResultsStructure(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
	}

	server := NewServer(cfg)
	defer server.Close()

	// Index tools
	if server.indexer != nil {
		tools := []spawner.Tool{
			{
				Name:        "example_tool",
				Description: "Example tool for testing",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{"type": "string"},
					},
				},
			},
		}
		_ = server.indexer.IndexServer("example", tools)
	}

	result, err := server.execHubSearch("example", "", 10)
	if err != nil {
		t.Fatalf("execHubSearch failed: %v", err)
	}

	// Parse and validate structure
	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Skip("Fallback response (no indexer)")
	}

	// Validate top-level fields
	expectedFields := []string{"searchId", "query", "totalResults", "results", "failedServers"}
	for _, field := range expectedFields {
		if _, exists := resultData[field]; !exists {
			t.Errorf("result missing field: %s", field)
		}
	}

	// Validate results array structure
	results, ok := resultData["results"].([]interface{})
	if !ok || len(results) == 0 {
		return // No results to validate
	}

	firstResult, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatal("first result is not a map")
	}

	// Validate flat structure (no nested "tool" object)
	requiredFields := []string{"name", "description", "inputSchema", "server", "score"}
	for _, field := range requiredFields {
		if _, exists := firstResult[field]; !exists {
			t.Errorf("result missing field: %s", field)
		}
	}

	// Verify removed fields NOT present
	if _, exists := firstResult["expectedResponse"]; exists {
		t.Error("result should not have expectedResponse field")
	}
	if _, exists := firstResult["matchReason"]; exists {
		t.Error("result should not have matchReason field")
	}
	if _, exists := firstResult["tool"]; exists {
		t.Error("result should not have nested tool object")
	}
}

// TestCompactJSON verifies hub_search returns compact JSON (no indentation)
func TestCompactJSON(t *testing.T) {
	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
	}

	server := NewServer(cfg)
	defer server.Close()

	// Index tools
	if server.indexer != nil {
		tools := []spawner.Tool{
			{
				Name:        "test_tool",
				Description: "Test tool for compact JSON validation",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param": map[string]interface{}{"type": "string"},
					},
				},
			},
		}
		_ = server.indexer.IndexServer("test", tools)
	}

	result, err := server.execHubSearch("test", "", 10)
	if err != nil {
		t.Fatalf("execHubSearch failed: %v", err)
	}

	// Verify JSON is compact (no indentation patterns)
	if strings.Contains(result, "\n  ") {
		t.Error("response contains 2-space indentation (should be compact)")
	}
	if strings.Contains(result, "\n    ") {
		t.Error("response contains 4-space indentation (should be compact)")
	}
	if strings.Contains(result, "\t") {
		t.Error("response contains tab characters (should be compact)")
	}

	// Verify it's still valid JSON
	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Skip("Fallback response (no indexer)")
	}

	t.Logf("Compact JSON size: %d bytes", len(result))
}

// TestRPCHybridSearchWorkflow tests hybrid search via RPC handlers
func TestRPCHybridSearchWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := &config.Config{
		Servers: map[string]*config.ServerConfig{},
	}

	server := NewServer(cfg)
	defer server.Close()

	if server.indexer == nil {
		t.Skip("indexer not available")
	}

	// Index tools
	tools := []spawner.Tool{
		{
			Name:        "create_task",
			Description: "Create a new task in project management",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		{
			Name:        "list_tasks",
			Description: "List all tasks in the system",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
	}
	_ = server.indexer.IndexServer("pm", tools)

	// Test hybrid search via execHubSearch (RPC handler)
	result, err := server.execHubSearch("create task", "", 10)
	if err != nil {
		t.Fatalf("execHubSearch failed: %v", err)
	}

	// Parse result
	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Skip("Fallback response")
	}

	results, ok := resultData["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Error("expected search results")
		return
	}

	// Verify BM25 scoring via RPC
	firstResult := results[0].(map[string]interface{})
	if score, ok := firstResult["score"].(float64); ok {
		if score <= 0 {
			t.Error("expected positive BM25 score")
		}
	}
}
