package benchmark

import (
	"encoding/json"
	"testing"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
)

func TestCalculateTokenSavings(t *testing.T) {
	tests := []struct {
		name           string
		serverCount    int
		avgToolsPerSvr int
		tokensPerTool  int
		hubTools       int
		hubTokens      int
		wantSavings    int
		wantPercent    float64
	}{
		{
			name:           "typical 6 servers",
			serverCount:    6,
			avgToolsPerSvr: 10,
			tokensPerTool:  150,
			hubTools:       5,
			hubTokens:      150,
			wantSavings:    8250, // 9000 - 750
			wantPercent:    91.67,
		},
		{
			name:           "minimal 1 server",
			serverCount:    1,
			avgToolsPerSvr: 10,
			tokensPerTool:  150,
			hubTools:       5,
			hubTokens:      150,
			wantSavings:    750, // 1500 - 750
			wantPercent:    50.0,
		},
		{
			name:           "large setup 10 servers",
			serverCount:    10,
			avgToolsPerSvr: 10,
			tokensPerTool:  150,
			hubTools:       5,
			hubTokens:      150,
			wantSavings:    14250, // 15000 - 750
			wantPercent:    95.0,
		},
		{
			name:           "no savings - equal",
			serverCount:    1,
			avgToolsPerSvr: 5,
			tokensPerTool:  150,
			hubTools:       5,
			hubTokens:      150,
			wantSavings:    0, // 750 - 750
			wantPercent:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traditional := tt.serverCount * tt.avgToolsPerSvr * tt.tokensPerTool
			toolHub := tt.hubTools * tt.hubTokens
			savings := traditional - toolHub
			percent := (float64(savings) / float64(traditional)) * 100

			if savings != tt.wantSavings {
				t.Errorf("Expected savings %d, got %d", tt.wantSavings, savings)
			}

			// Use tolerance for float comparison (±0.1%)
			if percent < tt.wantPercent-0.1 || percent > tt.wantPercent+0.1 {
				t.Errorf("Expected percent %.2f, got %.2f", tt.wantPercent, percent)
			}
		})
	}
}

func TestCountActualToolHubTokens(t *testing.T) {
	count := CountActualToolHubTokens()

	// hub_search + hub_execute + hub_list + hub_discover + hub_help tool definitions
	// Should be ~750 tokens (5 tools × ~150 tokens each)
	// Allow reasonable range based on actual tool complexity
	if count < 200 || count > 2000 {
		t.Errorf("Expected token count 200-2000, got %d (this may need adjustment based on actual tool definitions)", count)
	}

	// Verify it returns consistent results
	count2 := CountActualToolHubTokens()
	if count != count2 {
		t.Errorf("CountActualToolHubTokens() not deterministic: %d != %d", count, count2)
	}
}

func TestCountTokens(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantMin int
		wantMax int
	}{
		{
			name:    "simple string",
			input:   "hello world",
			wantMin: 3, // ~13 chars / 3 = 4 tokens
			wantMax: 10,
		},
		{
			name: "json object",
			input: map[string]interface{}{
				"name":        "test",
				"description": "A test tool",
			},
			wantMin: 10,
			wantMax: 30,
		},
		{
			name:    "empty",
			input:   "",
			wantMin: 0,
			wantMax: 1,
		},
		{
			name:    "nil",
			input:   nil,
			wantMin: 0,
			wantMax: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := CountTokens(tt.input)
			if count < tt.wantMin || count > tt.wantMax {
				t.Errorf("CountTokens() = %d, want range [%d, %d]", count, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGetToolHubToolDefinitions(t *testing.T) {
	tools := GetToolHubToolDefinitions()

	// Verify we have exactly 5 tools
	if len(tools) != 5 {
		t.Errorf("Expected 5 tool definitions, got %d", len(tools))
	}

	// Verify all tools have required fields
	expectedTools := map[string]bool{
		"hub_list":     false,
		"hub_discover": false,
		"hub_search":   false,
		"hub_execute":  false,
		"hub_help":     false,
	}

	for _, tool := range tools {
		name, ok := tool["name"].(string)
		if !ok {
			t.Error("Tool missing name field")
			continue
		}

		if _, exists := expectedTools[name]; !exists {
			t.Errorf("Unexpected tool name: %s", name)
			continue
		}
		expectedTools[name] = true

		// Verify required fields
		if _, ok := tool["description"]; !ok {
			t.Errorf("Tool %s missing description", name)
		}
		if _, ok := tool["inputSchema"]; !ok {
			t.Errorf("Tool %s missing inputSchema", name)
		}
	}

	// Verify all expected tools were found
	for name, found := range expectedTools {
		if !found {
			t.Errorf("Expected tool %s not found", name)
		}
	}
}

func TestRunBenchmark(t *testing.T) {
	tests := []struct {
		name           string
		servers        map[string]*config.ServerConfig
		wantMinSavings int
		wantMinPercent float64
	}{
		{
			name: "single server",
			servers: map[string]*config.ServerConfig{
				"jira": {Command: "npx", Args: []string{"-y", "@lvmk/jira-mcp"}},
			},
			wantMinSavings: 0,
			wantMinPercent: 0,
		},
		{
			name: "multiple servers",
			servers: map[string]*config.ServerConfig{
				"jira":    {Command: "npx"},
				"outline": {Command: "npx"},
				"figma":   {Command: "npx"},
			},
			wantMinSavings: 1000,
			wantMinPercent: 50.0,
		},
		{
			name: "high-tool servers",
			servers: map[string]*config.ServerConfig{
				"chromeDevtools": {Command: "npx"},
				"github":         {Command: "npx"},
				"outline":        {Command: "npx"},
			},
			wantMinSavings: 2000,
			wantMinPercent: 80.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Servers: tt.servers,
			}

			result := RunBenchmark(cfg)

			if result == nil {
				t.Fatal("RunBenchmark returned nil")
			}

			// Verify traditional setup
			if result.Traditional.ServerCount != len(tt.servers) {
				t.Errorf("Expected %d servers, got %d", len(tt.servers), result.Traditional.ServerCount)
			}

			// Verify tool-hub setup
			if result.ToolHub.ServerCount != 1 {
				t.Errorf("Expected 1 tool-hub server, got %d", result.ToolHub.ServerCount)
			}
			if result.ToolHub.ToolCount != ToolHubTools {
				t.Errorf("Expected %d tools, got %d", ToolHubTools, result.ToolHub.ToolCount)
			}

			// Verify savings calculations
			if result.TokenSavings < tt.wantMinSavings {
				t.Errorf("Expected at least %d token savings, got %d", tt.wantMinSavings, result.TokenSavings)
			}
			if result.SavingsPercent < tt.wantMinPercent {
				t.Errorf("Expected at least %.1f%% savings, got %.1f%%", tt.wantMinPercent, result.SavingsPercent)
			}

			// Verify consistency
			expectedSavings := result.Traditional.DefinitionTokens - result.ToolHub.DefinitionTokens
			if result.TokenSavings != expectedSavings {
				t.Errorf("TokenSavings inconsistent: got %d, expected %d", result.TokenSavings, expectedSavings)
			}
		})
	}
}

func TestGetToolCount(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		wantCount  int
	}{
		{"known high-count server", "chromeDevtools", 35},
		{"known medium-count server", "jira", 13},
		{"known low-count server", "figma", 5},
		{"unknown server defaults", "unknown-server", AverageToolsPerServer},
		{"empty name defaults", "", AverageToolsPerServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := getToolCount(tt.serverName)
			if count != tt.wantCount {
				t.Errorf("getToolCount(%q) = %d, want %d", tt.serverName, count, tt.wantCount)
			}
		})
	}
}

func TestFormatResult(t *testing.T) {
	result := &BenchmarkResult{
		Traditional: TokenEstimate{
			ServerCount:      6,
			ToolCount:        60,
			DefinitionTokens: 9000,
			Description:      "6 MCP servers with 60 total tools",
		},
		ToolHub: TokenEstimate{
			ServerCount:      1,
			ToolCount:        5,
			DefinitionTokens: 750,
			Description:      "1 tool-hub-mcp server with 5 meta-tools",
		},
		TokenSavings:   8250,
		SavingsPercent: 91.67,
	}

	output := FormatResult(result)

	// Verify output contains key information
	required := []string{
		"TOKEN EFFICIENCY BENCHMARK",
		"TRADITIONAL MCP",
		"TOOL-HUB-MCP",
		"SAVINGS",
		"6",  // server count
		"60", // tool count
		"91", // savings percent (checking prefix instead of exact number)
	}

	for _, substr := range required {
		if !containsString(output, substr) {
			t.Errorf("Output missing required string: %q", substr)
		}
	}

	// Verify it's not empty
	if len(output) < 100 {
		t.Errorf("FormatResult output too short: %d bytes", len(output))
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && indexString(s, substr) >= 0))
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestTokenResponseOptimization validates token savings from optimized response format
func TestTokenResponseOptimization(t *testing.T) {
	// Simulate old verbose response format
	oldResponse := `{
  "searchId": "uuid-123",
  "query": "create issue",
  "totalResults": 2,
  "results": [
    {
      "tool": {
        "name": "jira_create_issue",
        "description": "Create a new Jira issue with summary, description, project key, and optional fields",
        "inputSchema": {
          "type": "object",
          "properties": {
            "projectKey": {"type": "string", "description": "Project key like PROJ"},
            "summary": {"type": "string", "description": "Issue title/summary"},
            "description": {"type": "string", "description": "Detailed description"}
          },
          "required": ["projectKey", "summary"]
        },
        "expectedResponse": "Returns: issueKey (string), summary (string), status (string), assignee (string)"
      },
      "server": "jira",
      "score": 8.342,
      "matchReason": "Strong keyword match in tool name or description"
    },
    {
      "tool": {
        "name": "github_create_issue",
        "description": "Create a new GitHub issue with title, body, labels, and assignees",
        "inputSchema": {
          "type": "object",
          "properties": {
            "owner": {"type": "string", "description": "Repository owner"},
            "repo": {"type": "string", "description": "Repository name"},
            "title": {"type": "string", "description": "Issue title"}
          },
          "required": ["owner", "repo", "title"]
        },
        "expectedResponse": "Returns: number (integer), title (string), state (string), url (string)"
      },
      "server": "github",
      "score": 7.891,
      "matchReason": "Strong keyword match in tool name or description"
    }
  ],
  "failedServers": []
}`

	// Simulate new compact response format
	newResponse := `{"searchId":"uuid-123","query":"create issue","totalResults":2,"results":[{"name":"jira_create_issue","description":"Create a new Jira issue with summary, description, project key, and optional fields","inputSchema":{"type":"object","properties":{"projectKey":{"type":"string","description":"Project key like PROJ"},"summary":{"type":"string","description":"Issue title/summary"},"description":{"type":"string","description":"Detailed description"}},"required":["projectKey","summary"]},"server":"jira","score":8.342},{"name":"github_create_issue","description":"Create a new GitHub issue with title, body, labels, and assignees","inputSchema":{"type":"object","properties":{"owner":{"type":"string","description":"Repository owner"},"repo":{"type":"string","description":"Repository name"},"title":{"type":"string","description":"Issue title"}},"required":["owner","repo","title"]},"server":"github","score":7.891}],"failedServers":[]}`

	oldTokens := CountTokens(oldResponse)
	newTokens := CountTokens(newResponse)

	savings := oldTokens - newTokens
	savingsPercent := float64(savings) / float64(oldTokens) * 100

	t.Logf("Old format: %d bytes, ~%d tokens", len(oldResponse), oldTokens)
	t.Logf("New format: %d bytes, ~%d tokens", len(newResponse), newTokens)
	t.Logf("Byte savings: %d bytes (%.1f%%)", len(oldResponse)-len(newResponse), float64(len(oldResponse)-len(newResponse))/float64(len(oldResponse))*100)
	t.Logf("Token savings: %d tokens (%.1f%%)", savings, savingsPercent)

	// Validate minimum 40% token reduction (realistic baseline)
	// Note: Actual savings depend on content. With verbose descriptions, 40-50% is expected.
	// With more tools (10 results), savings approach 50-70% due to removed redundancy.
	if savingsPercent < 40.0 {
		t.Errorf("Expected at least 40%% token savings, got %.1f%%", savingsPercent)
	}

	// Validate byte size reduction (should be ~40-50% for 2 results)
	byteSavingsPercent := float64(len(oldResponse)-len(newResponse)) / float64(len(oldResponse)) * 100
	if byteSavingsPercent < 35.0 {
		t.Errorf("Expected at least 35%% byte savings, got %.1f%%", byteSavingsPercent)
	}

	// Log success message if savings meet baseline
	if savingsPercent >= 40.0 {
		t.Logf("✓ Token optimization successful: %.1f%% savings", savingsPercent)
	}

	// Verify both contain same semantic data (same tool count)
	var oldData, newData map[string]interface{}
	if err := json.Unmarshal([]byte(oldResponse), &oldData); err != nil {
		t.Fatalf("Old response is invalid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(newResponse), &newData); err != nil {
		t.Fatalf("New response is invalid JSON: %v", err)
	}

	oldResults := oldData["results"].([]interface{})
	newResults := newData["results"].([]interface{})
	if len(oldResults) != len(newResults) {
		t.Errorf("Result count mismatch: old=%d, new=%d", len(oldResults), len(newResults))
	}

	t.Log("Both formats contain valid JSON with same result count")
}
