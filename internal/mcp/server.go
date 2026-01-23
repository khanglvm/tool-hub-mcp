/*
Package mcp implements the MCP server that exposes meta-tools.

The server uses stdio transport and exposes 3 meta-tools:
  - hub_search: Semantic search for tools across all servers (with discovery)
  - hub_execute: Execute a tool from a specific server (with learning)
  - hub_manage: Add or remove MCP servers from configuration
*/
package mcp

import (
	"bufio"
	"context"
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
	config        *config.Config
	configMu      sync.RWMutex
	spawner       *spawner.Pool
	indexer       *search.Indexer
	storage       *storage.SQLiteStorage
	tracker       *learning.Tracker
	failedServers map[string]string // serverName → error message

	// Context for background goroutines (update checker, discovery)
	ctx    context.Context
	cancel context.CancelFunc

	// closeOnce ensures Close() is idempotent (safe to call multiple times)
	closeOnce sync.Once
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

	// Create cancellable context for background tasks
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config:        cfg,
		spawner:       spawner.NewPool(poolSize),
		indexer:       indexer,
		storage:       str,
		tracker:       tracker,
		failedServers: make(map[string]string),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Close gracefully shuts down the server and cleans up all resources.
// Resources closed in dependency order: tracker → storage → indexer → spawner.
// Safe to call multiple times (idempotent via sync.Once).
func (s *Server) Close() error {
	var errs []error

	s.closeOnce.Do(func() {
		log.Println("Shutting down server...")

		// Cancel background goroutines first
		if s.cancel != nil {
			s.cancel()
		}

		// 1. Stop tracker (flushes event queue to storage)
		if s.tracker != nil {
			log.Println("Stopping tracker...")
			s.tracker.Stop()
		}

		// 2. Close storage (commits SQLite transactions)
		if s.storage != nil {
			log.Println("Closing storage...")
			if err := s.storage.Close(); err != nil {
				errs = append(errs, fmt.Errorf("storage: %w", err))
			}
		}

		// 3. Close indexer (closes Bleve index files)
		if s.indexer != nil {
			log.Println("Closing indexer...")
			if err := s.indexer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("indexer: %w", err))
			}
		}

		// 4. Close spawner pool (terminates child processes)
		if s.spawner != nil {
			log.Println("Closing spawner pool...")
			if err := s.spawner.Close(); err != nil {
				errs = append(errs, fmt.Errorf("spawner: %w", err))
			}
		}

		log.Println("Server shutdown complete")
	})

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
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

	// Clear previous failed servers (fresh state each reindex)
	s.failedServers = make(map[string]string)

	// Index each server's tools
	for serverName, serverCfg := range s.config.Servers {
		tools, err := s.spawner.GetTools(serverName, serverCfg)
		if err != nil {
			// Capture error for this server
			s.failedServers[serverName] = err.Error()
			log.Printf("Warning: failed to get tools from %s: %v", serverName, err)
			continue
		}

		if err := s.indexer.IndexServer(serverName, tools); err != nil {
			// Capture indexing error
			s.failedServers[serverName] = fmt.Sprintf("indexing failed: %v", err)
			log.Printf("Warning: failed to index tools from %s: %v", serverName, err)
			continue
		}

		log.Printf("Indexed %d tools from %s", len(tools), serverName)
	}

	// Log total indexed count
	if count, err := s.indexer.Count(); err == nil {
		log.Printf("Total tools indexed: %d", count)
	}

	// Log summary of failed servers
	if len(s.failedServers) > 0 {
		log.Printf("Failed servers: %d", len(s.failedServers))
	}

	return nil
}

// StartBackgroundDiscovery starts tool indexing in background goroutine.
// Server accepts requests immediately; search improves as indexing completes.
// Goroutine exits when server context is cancelled.
func (s *Server) StartBackgroundDiscovery() {
	go func() {
		if s.indexer == nil {
			return
		}

		// Check if context cancelled before starting
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		if err := s.IndexTools(); err != nil {
			log.Printf("Background indexing failed: %v", err)
		}
	}()
}

// Context returns the server's context for background tasks.
func (s *Server) Context() context.Context {
	return s.ctx
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
		{
			"name": "hub_manage",
			"description": `Manage MCP servers by adding or removing them from configuration.

USE THIS TOOL when:
• User asks to "add a server" or "register an MCP server"
• User asks to "remove a server" or "unregister a server"
• User provides server configuration details

OPERATIONS:
1. add - Register a new MCP server
   - Required: name, command, args
   - Optional: env (environment variables)

2. remove - Unregister an MCP server
   - Required: name

IMPORTANT:
• Server names will be normalized to camelCase
• Config is validated before saving
• Changes trigger automatic reindexing
• Backup created before config modification

EXAMPLES:
• Add: {"operation": "add", "name": "jira", "command": "npx", "args": ["-y", "@lvmk/jira-mcp"], "env": {"API_KEY": "..."}}
• Remove: {"operation": "remove", "name": "jira"}

CURRENTLY REGISTERED: ` + serverList,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operation": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"add", "remove"},
						"description": "Operation to perform (add or remove)",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Server name (will be normalized to camelCase)",
					},
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Command to execute (required for add operation)",
					},
					"args": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Command arguments (required for add operation)",
					},
					"env": map[string]interface{}{
						"type": "object",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
						"description": "Environment variables (optional for add operation)",
					},
				},
				"required": []string{"operation", "name"},
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

// getFailedServers returns a list of failed servers with error messages.
// Thread-safe: acquires read lock.
func (s *Server) getFailedServers() []map[string]interface{} {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	if len(s.failedServers) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(s.failedServers))
	for name, errorMsg := range s.failedServers {
		result = append(result, map[string]interface{}{
			"server": name,
			"error":  errorMsg,
		})
	}
	return result
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
	case "hub_manage":
		operation, _ := params.Arguments["operation"].(string)
		name, _ := params.Arguments["name"].(string)
		command, _ := params.Arguments["command"].(string)

		// Parse args array
		var args []string
		if argsInterface, ok := params.Arguments["args"].([]interface{}); ok {
			args = make([]string, len(argsInterface))
			for i, v := range argsInterface {
				if str, ok := v.(string); ok {
					args[i] = str
				}
			}
		}

		// Parse env map
		var env map[string]string
		if envInterface, ok := params.Arguments["env"].(map[string]interface{}); ok {
			env = make(map[string]string)
			for k, v := range envInterface {
				if str, ok := v.(string); ok {
					env[k] = str
				}
			}
		}

		result, err = s.execHubManage(operation, name, command, args, env)
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
// Returns rich JSON response with searchId, tool details, schemas, and failed servers.
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

	// Add failed servers (always include for consistent schema)
	failedServers := s.getFailedServers()
	if failedServers != nil && len(failedServers) > 0 {
		response["failedServers"] = failedServers
	} else {
		response["failedServers"] = []map[string]interface{}{}
	}

	// Convert to JSON (compact format for token efficiency)
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(jsonBytes), nil
}

// formatSearchResults converts search results to compact format with tool details.
func (s *Server) formatSearchResults(results []search.SearchResult) []map[string]interface{} {
	formatted := make([]map[string]interface{}, 0, len(results))

	for _, result := range results {
		toolDetail := map[string]interface{}{
			"name":        result.ToolName,
			"description": result.Description,
			"inputSchema": result.InputSchema,
			"server":      result.ServerName,
			"score":       result.Score,
		}

		formatted = append(formatted, toolDetail)
	}

	return formatted
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

	var result strings.Builder

	if len(matchedServers) == 0 {
		// No match, return all servers as suggestions
		result.WriteString(fmt.Sprintf("No direct match for '%s'. Available servers:\n\n", query))
		for name := range s.config.Servers {
			result.WriteString(fmt.Sprintf("  • %s\n", name))
		}
		result.WriteString("\nTry hub_search with a server name to see tools from that server.")
	} else {
		// Return matched servers with recommendation
		result.WriteString(fmt.Sprintf("For '%s', matching servers:\n\n", query))

		for _, server := range matchedServers {
			result.WriteString(fmt.Sprintf("  • %s\n", server))
		}

		result.WriteString("\nNext step: Use hub_search to find specific tools, then hub_execute to run them.")
	}

	// Add failed servers info
	failedServers := s.getFailedServers()
	if len(failedServers) > 0 {
		result.WriteString("\n\n⚠️  Failed Servers:\n")
		for _, fs := range failedServers {
			serverName := fs["server"].(string)
			errorMsg := fs["error"].(string)
			result.WriteString(fmt.Sprintf("  • %s: %s\n", serverName, errorMsg))
		}
	}

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

// execHubManage handles server management operations (add/remove).
func (s *Server) execHubManage(operation, name, command string, args []string, env map[string]string) (string, error) {
	// Acquire write lock for config modification
	s.configMu.Lock()
	defer s.configMu.Unlock()

	// Validate operation
	if operation != "add" && operation != "remove" {
		return "", fmt.Errorf("invalid operation '%s'. Must be 'add' or 'remove'", operation)
	}

	// Validate name
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("server name cannot be empty")
	}

	name = strings.TrimSpace(name)

	// Handle operations
	switch operation {
	case "add":
		return s.addServer(name, command, args, env)
	case "remove":
		return s.removeServer(name)
	default:
		return "", fmt.Errorf("unsupported operation: %s", operation)
	}
}

// addServer adds a new MCP server to the configuration.
func (s *Server) addServer(name, command string, args []string, env map[string]string) (string, error) {
	// Validate command
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command cannot be empty for add operation")
	}

	// Validate args
	if args == nil {
		args = []string{} // Default to empty array
	}

	// Check if server already exists
	if _, exists := s.config.Servers[name]; exists {
		return "", fmt.Errorf("server '%s' already exists. Use hub_execute to list servers or remove first", name)
	}

	// Create server config
	serverCfg := &config.ServerConfig{
		Command: strings.TrimSpace(command),
		Args:    args,
		Env:     env,
		Source:  "hub_manage",
	}

	// Add to config
	s.config.Servers[name] = serverCfg

	// Save config atomically
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		// Rollback
		delete(s.config.Servers, name)
		return "", fmt.Errorf("failed to get config path: %w", err)
	}

	if err := config.Save(s.config, configPath); err != nil {
		// Rollback
		delete(s.config.Servers, name)
		return "", fmt.Errorf("failed to save config: %w. Config rolled back", err)
	}

	// Trigger reindexing (must hold lock)
	if s.indexer != nil {
		if err := s.indexToolsUnsafe(); err != nil {
			log.Printf("Warning: failed to reindex after adding server '%s': %v", name, err)
		}
	}

	return fmt.Sprintf("✓ Server '%s' added successfully.\n\nCommand: %s\nArgs: %v\n\nConfig saved to: %s\nIndexing triggered.",
		name, command, args, configPath), nil
}

// removeServer removes an MCP server from the configuration.
func (s *Server) removeServer(name string) (string, error) {
	// Check if server exists
	if _, exists := s.config.Servers[name]; !exists {
		availableServers := make([]string, 0, len(s.config.Servers))
		for serverName := range s.config.Servers {
			availableServers = append(availableServers, serverName)
		}
		return "", fmt.Errorf("server '%s' not found. Available servers: %v", name, availableServers)
	}

	// Backup server config for potential rollback
	backupCfg := s.config.Servers[name]

	// Remove from config
	delete(s.config.Servers, name)

	// Save config atomically
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		// Rollback
		s.config.Servers[name] = backupCfg
		return "", fmt.Errorf("failed to get config path: %w", err)
	}

	if err := config.Save(s.config, configPath); err != nil {
		// Rollback
		s.config.Servers[name] = backupCfg
		return "", fmt.Errorf("failed to save config: %w. Config rolled back", err)
	}

	// Remove from indexer if available
	if s.indexer != nil {
		if err := s.indexer.RemoveServer(name); err != nil {
			log.Printf("Warning: failed to remove server '%s' from index: %v", name, err)
		}
	}

	// Trigger reindexing (must hold lock)
	if s.indexer != nil {
		if err := s.indexToolsUnsafe(); err != nil {
			log.Printf("Warning: failed to reindex after removing server '%s': %v", name, err)
		}
	}

	return fmt.Sprintf("✓ Server '%s' removed successfully.\n\nConfig saved to: %s\nIndexing triggered.",
		name, configPath), nil
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
