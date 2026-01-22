/*
Package mcp implements the MCP server that exposes meta-tools.

The server uses stdio transport and exposes 2 meta-tools:
  - hub_search: Semantic search for tools across all servers (with discovery)
  - hub_execute: Execute a tool from a specific server (with learning)
*/
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/learning"
	"github.com/khanglvm/tool-hub-mcp/internal/search"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
	"github.com/khanglvm/tool-hub-mcp/internal/storage"
	"github.com/khanglvm/tool-hub-mcp/internal/version"
)

// Server represents the tool-hub-mcp MCP server.
type Server struct {
	config   *config.Config
	configMu sync.RWMutex
	spawner  *spawner.Pool
	indexer  *search.Indexer
	storage  *storage.SQLiteStorage
	tracker  *learning.Tracker
}

// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *config.Config) *Server {
	poolSize := 3
	if cfg.Settings != nil && cfg.Settings.ProcessPoolSize > 0 {
		poolSize = cfg.Settings.ProcessPoolSize
	}

	// Create search indexer
	indexer, err := search.NewIndexer()
	if err != nil {
		log.Printf("Warning: failed to create search indexer: %v", err)
		indexer = nil
	}

	// Create storage layer
	str := storage.NewStorage()
	if err := str.Init(); err != nil {
		log.Printf("Warning: failed to initialize storage: %v", err)
		// Storage is optional, continue without it
	}

	// Create learning tracker
	var tracker *learning.Tracker
	if str != nil {
		tracker = learning.NewTracker(str)
	}

	return &Server{
		config:  cfg,
		spawner: spawner.NewPool(poolSize),
		indexer: indexer,
		storage: str,
		tracker: tracker,
	}
}

// IndexTools indexes all tools from all servers for search.
// Thread-safe: acquires read lock before accessing config.
func (s *Server) IndexTools() error {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.indexToolsUnsafe()
}

// indexToolsUnsafe indexes tools without locking (caller must hold lock).
// This prevents recursive locking when called from ReloadConfig.
func (s *Server) indexToolsUnsafe() error {
	if s.indexer == nil {
		return fmt.Errorf("search indexer not available")
	}

	// Index each server's tools
	for serverName, serverCfg := range s.config.Servers {
		tools, err := s.spawner.GetTools(serverName, serverCfg)
		if err != nil {
			log.Printf("Warning: failed to get tools from %s: %v", serverName, err)
			continue
		}

		if err := s.indexer.IndexServer(serverName, tools); err != nil {
			log.Printf("Warning: failed to index tools from %s: %v", serverName, err)
		}

		log.Printf("Indexed %d tools from %s", len(tools), serverName)
	}

	// Log total indexed count
	if count, err := s.indexer.Count(); err == nil {
		log.Printf("Total tools indexed: %d", count)
	}

	return nil
}

// StartBackgroundDiscovery starts tool indexing in background goroutine.
// Server accepts requests immediately; search improves as indexing completes.
func (s *Server) StartBackgroundDiscovery() {
	go func() {
		if s.indexer == nil {
			return
		}
		if err := s.IndexTools(); err != nil {
			log.Printf("Background indexing failed: %v", err)
		}
	}()
}

// ReloadConfig atomically reloads configuration and reindexes tools.
// Thread-safe for concurrent use from background goroutines.
func (s *Server) ReloadConfig(newCfg *config.Config) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.config = newCfg

	// Re-index tools with new server list using unsafe version (already holding lock)
	if s.indexer != nil {
		if err := s.indexToolsUnsafe(); err != nil {
			log.Printf("Warning: failed to reindex tools after config reload: %v", err)
		}
	}

	log.Printf("Config reloaded: %d servers registered", len(newCfg.Servers))
}

// Run starts the MCP server using stdio transport.
// This blocks until stdin is closed.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Bytes()

		response, err := s.handleRequest(line)
		if err != nil {
			// Send error response
			s.sendError(err)
			continue
		}

		if response != nil {
			s.sendResponse(response)
		}
	}

	return scanner.Err()
}

// MCPRequest represents an incoming MCP JSON-RPC request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an outgoing MCP JSON-RPC response.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// handleRequest processes an incoming MCP request.
func (s *Server) handleRequest(data []byte) (*MCPResponse, error) {
	var req MCPRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC request: %w", err)
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(&req)
	case "tools/list":
		return s.handleToolsList(&req)
	case "tools/call":
		return s.handleToolsCall(&req)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32601, Message: "Method not found"},
		}, nil
	}
}

// handleInitialize handles the MCP initialize request.
func (s *Server) handleInitialize(req *MCPRequest) (*MCPResponse, error) {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "tool-hub-mcp",
				"version": version.Version,
			},
		},
	}, nil
}

// handleToolsList returns the list of available meta-tools with AI-native descriptions.
func (s *Server) handleToolsList(req *MCPRequest) (*MCPResponse, error) {
	// Build dynamic server list for AI context (runtime, not hardcoded)
	serverList := s.getServerNames()

	tools := []map[string]interface{}{
		{
			"name": "hub_search",
			"description": fmt.Sprintf(`Gateway to external tools and integrations. Use semantic search to discover capabilities.

CALL THIS FIRST when:
• User describes WHAT they want to do (not HOW)
• User mentions ANY external service/platform by name (Figma, Jira, etc.)
• User asks about available tools or capabilities
• You're unsure which tool to use for a task
• User wants to interact with external APIs, services, or data sources

WORKFLOW:
1. Describe capability in plain English
2. Get ranked tools with full schemas
3. Use hub_execute to run selected tool

EXAMPLES (capability → tool discovery):
• "extract figma design" → finds Figma extract tools
• "create jira ticket" → finds Jira create_issue with schema
• "take screenshot of website" → finds screenshot tools with parameters
• "search documents" → finds doc search tools
• "all tools" → lists all tools from all servers

ANTI-PATTERNS (don't do this):
• Don't use web search when user mentions external services
• Don't guess tool names - search first
• Don't assume tools don't exist - search will tell you

CURRENTLY REGISTERED: %s

Returns: JSON with searchId (for tracking), results array with tool details (name, description, inputSchema, expectedResponse), server, score, matchReason.`, serverList),
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "What you want to do in plain English",
					},
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Optional: filter to specific server",
						"enum":        s.getServerNamesList(),
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Optional: max results (default 10)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name": "hub_execute",
			"description": fmt.Sprintf(`Run a tool from an external integration.

USE THIS TOOL AFTER:
• You've used hub_search to discover available tools
• You know which tool to run and have its schema
• You have the tool name and required arguments ready

IMPORTANT: Always call hub_search first to discover tools and get their schemas.
Only call hub_execute when you have the tool details from hub_search.

LEARNING: Optionally pass searchId from hub_search to improve tool recommendations.
This helps the system learn which tools work best for specific queries.

CURRENTLY REGISTERED: %s`, serverList),
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Server name",
						"enum":        s.getServerNamesList(),
					},
					"tool": map[string]interface{}{
						"type":        "string",
						"description": "Tool name (from hub_search)",
					},
					"arguments": map[string]interface{}{
						"type":        "object",
						"description": "Tool arguments (schema from hub_search)",
					},
					"searchId": map[string]interface{}{
						"type":        "string",
						"description": "Optional: search session ID from hub_search to link this execution for learning",
					},
				},
				"required": []string{"server", "tool"},
			},
		},
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}, nil
}

// getServerNames returns a comma-separated list of server names.
func (s *Server) getServerNames() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	names := []string{}
	for name := range s.config.Servers {
		names = append(names, name)
	}
	result := ""
	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += name
	}
	return result
}

// getServerNamesList returns server names as a slice for enum.
func (s *Server) getServerNamesList() []string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	names := []string{}
	for name := range s.config.Servers {
		names = append(names, name)
	}
	return names
}

// handleToolsCall handles tool execution requests.
func (s *Server) handleToolsCall(req *MCPRequest) (*MCPResponse, error) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	var result interface{}
	var err error

	switch params.Name {
	case "hub_search":
		query, _ := params.Arguments["query"].(string)
		server, _ := params.Arguments["server"].(string)
		limitFloat, _ := params.Arguments["limit"].(float64)
		limit := int(limitFloat)
		result, err = s.execHubSearch(query, server, limit)
	case "hub_execute":
		serverName, _ := params.Arguments["server"].(string)
		toolName, _ := params.Arguments["tool"].(string)
		args, _ := params.Arguments["arguments"].(map[string]interface{})
		searchId, _ := params.Arguments["searchId"].(string)
		result, err = s.execHubExecute(serverName, toolName, args, searchId)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32602, Message: fmt.Sprintf("Unknown tool: %s", params.Name)},
		}, nil
	}

	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32000, Message: err.Error()},
		}, nil
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}, nil
}

// execHubSearch searches for tools across all servers using BM25 semantic search.
// Returns rich JSON response with searchId, tool details, and schemas.
func (s *Server) execHubSearch(query, serverFilter string, limit int) (string, error) {
	// Generate unique searchId for tracking
	searchID := uuid.New().String()

	// Default limit if not specified
	if limit <= 0 {
		limit = 10
	}

	// If indexer is not available, fall back to simple server name matching
	if s.indexer == nil {
		return s.execHubSearchFallback(query, searchID)
	}

	var results []search.SearchResult
	var err error

	// Perform search with optional server filter
	if serverFilter != "" {
		// Search within specific server
		results, err = s.indexer.SearchByServer(query, serverFilter, limit)
	} else {
		// Search across all servers
		results, err = s.indexer.SearchBM25(query, limit)
	}

	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	// Store search in history for learning
	if s.storage != nil {
		searchRecord := storage.SearchRecord{
			SearchID:     searchID,
			QueryHash:    storage.HashQuery(query),
			Timestamp:    time.Now(),
			ResultsCount: len(results),
		}
		if err := s.storage.RecordSearch(searchRecord); err != nil {
			log.Printf("Warning: failed to record search: %v", err)
		}
	}

	// Build rich response
	response := map[string]interface{}{
		"searchId":     searchID,
		"query":        query,
		"totalResults": len(results),
		"results":      s.formatSearchResults(results),
	}

	// Convert to JSON
	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonBytes), nil
}

// formatSearchResults converts search results to rich format with tool details.
func (s *Server) formatSearchResults(results []search.SearchResult) []map[string]interface{} {
	formatted := make([]map[string]interface{}, 0, len(results))

	for _, result := range results {
		// Generate expected response description from schema
		expectedResponse := s.generateExpectedResponse(result.InputSchema)

		toolDetail := map[string]interface{}{
			"tool": map[string]interface{}{
				"name":             result.ToolName,
				"description":      result.Description,
				"inputSchema":      result.InputSchema,
				"expectedResponse": expectedResponse,
			},
			"server":      result.ServerName,
			"score":       result.Score,
			"matchReason": s.generateMatchReason(result),
		}

		formatted = append(formatted, toolDetail)
	}

	return formatted
}

// generateExpectedResponse creates a human-readable description of the expected response.
func (s *Server) generateExpectedResponse(schema interface{}) string {
	if schema == nil {
		return "Returns tool execution result"
	}

	// Parse schema as map
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return "Returns tool execution result"
	}

	// Try to extract output description
	// This is a simple heuristic - in production, you'd parse the schema more carefully
	var responseTypes []string

	if props, ok := schemaMap["properties"].(map[string]interface{}); ok {
		for propName, propDef := range props {
			if propDefMap, ok := propDef.(map[string]interface{}); ok {
				if propType, ok := propDefMap["type"].(string); ok {
					responseTypes = append(responseTypes, fmt.Sprintf("%s (%s)", propName, propType))
				}
			}
		}
	}

	if len(responseTypes) > 0 {
		return fmt.Sprintf("Returns: %s", strings.Join(responseTypes, ", "))
	}

	return "Returns tool execution result"
}

// generateMatchReason creates a human-readable explanation of why this tool matched.
func (s *Server) generateMatchReason(result search.SearchResult) string {
	if result.Score > 5.0 {
		return "Strong keyword match in tool name or description"
	} else if result.Score > 2.0 {
		return "Partial keyword match"
	} else {
		return "Low relevance match"
	}
}

// execHubSearchFallback is the fallback when indexer is not available.
func (s *Server) execHubSearchFallback(query, searchID string) (string, error) {
	query = strings.ToLower(query)

	s.configMu.RLock()
	defer s.configMu.RUnlock()

	// Match against actual registered server names (dynamic, no hardcoding)
	matchedServers := []string{}
	for name := range s.config.Servers {
		nameLower := strings.ToLower(name)
		// Match if query contains server name or server name contains query
		if strings.Contains(query, nameLower) || strings.Contains(nameLower, query) {
			matchedServers = append(matchedServers, name)
		}
	}

	if len(matchedServers) == 0 {
		// No match, return all servers as suggestions
		var result strings.Builder
		result.WriteString(fmt.Sprintf("No direct match for '%s'. Available servers:\n\n", query))
		for name := range s.config.Servers {
			result.WriteString(fmt.Sprintf("  • %s\n", name))
		}
		result.WriteString("\nTry hub_search with a server name to see tools from that server.")
		return result.String(), nil
	}

	// Return matched servers with recommendation
	var result strings.Builder
	result.WriteString(fmt.Sprintf("For '%s', matching servers:\n\n", query))

	for _, server := range matchedServers {
		result.WriteString(fmt.Sprintf("  • %s\n", server))
	}

	result.WriteString("\nNext step: Use hub_search to find specific tools, then hub_execute to run them.")
	return result.String(), nil
}

// execHubExecute executes a tool from a server.
func (s *Server) execHubExecute(serverName, toolName string, args map[string]interface{}, searchId string) (string, error) {
	s.configMu.RLock()
	server, exists := s.config.Servers[serverName]
	s.configMu.RUnlock()

	if !exists {
		return "", fmt.Errorf("server '%s' not found", serverName)
	}

	// Execute tool
	result, err := s.spawner.ExecuteTool(serverName, server, toolName, args)
	if err != nil {
		// Track failed execution
		s.trackUsage(toolName, searchId, false)
		return "", fmt.Errorf("failed to execute tool: %w", err)
	}

	// Track successful execution
	s.trackUsage(toolName, searchId, true)

	return result, nil
}

// trackUsage records tool usage for learning (non-blocking).
func (s *Server) trackUsage(toolName, searchId string, success bool) {
	if s.tracker == nil {
		return
	}

	// Hash searchId for privacy
	hashedSearchId := ""
	if searchId != "" {
		hashedSearchId = storage.HashQuery(searchId)
	}

	// Create usage event
	event := learning.UsageEvent{
		ToolName:    toolName,
		ContextHash: hashedSearchId,
		Timestamp:   time.Now(),
		Selected:    true,
		Rating:      0,
	}

	// Non-blocking track
	s.tracker.Track(event)

	// Log if tracking fails (tracker already handles errors internally)
	if s.tracker.IsEnabled() && len(hashedSearchId) > 0 {
		log.Printf("Tracked tool usage: %s (searchId: %s, success: %v)", toolName, searchId, success)
	}
}

// sendResponse writes a JSON-RPC response to stdout.
func (s *Server) sendResponse(resp *MCPResponse) {
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

// sendError writes an error response to stdout.
func (s *Server) sendError(err error) {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      nil,
		Error:   &MCPError{Code: -32700, Message: err.Error()},
	}
	s.sendResponse(resp)
}
