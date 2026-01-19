/*
Package mcp implements the MCP server that exposes meta-tools.

The server uses stdio transport and exposes 5 meta-tools:
  - hub_list: List all registered MCP servers
  - hub_discover: Get tool definitions from a specific server
  - hub_search: Semantic search for tools across all servers
  - hub_execute: Execute a tool from a specific server
  - hub_help: Get detailed help/schema for a tool
*/
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/khanglvm/tool-hub-mcp/internal/config"
	"github.com/khanglvm/tool-hub-mcp/internal/spawner"
)

// Server represents the tool-hub-mcp MCP server.
type Server struct {
	config  *config.Config
	spawner *spawner.Pool
}

// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *config.Config) *Server {
	poolSize := 3
	if cfg.Settings != nil && cfg.Settings.ProcessPoolSize > 0 {
		poolSize = cfg.Settings.ProcessPoolSize
	}

	return &Server{
		config:  cfg,
		spawner: spawner.NewPool(poolSize),
	}
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
				"version": "0.1.0",
			},
		},
	}, nil
}

// handleToolsList returns the list of available meta-tools with AI-native descriptions.
func (s *Server) handleToolsList(req *MCPRequest) (*MCPResponse, error) {
	// Build dynamic server catalog for AI context
	serverCatalog := s.buildServerCatalog()
	
	tools := []map[string]interface{}{
		{
			"name": "hub_list",
			"description": fmt.Sprintf(`List all available MCP servers and their capabilities.

WHEN TO USE: Call this first to discover what integrations are available.

AVAILABLE SERVERS:
%s

Returns: List of server names with their sources.`, serverCatalog),
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name": "hub_discover",
			"description": fmt.Sprintf(`Get detailed tool definitions from a specific MCP server.

WHEN TO USE: 
- When user mentions: figma, design, jira, issues, outline, documents, playwright, browser
- Before executing any tool, to see available operations and required parameters

AVAILABLE SERVERS: %s

Example: To get Figma design info, first call hub_discover with server="figma" to see available tools.`, s.getServerNames()),
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server": map[string]interface{}{
						"type":        "string",
						"description": "Server name from available servers list",
						"enum":        s.getServerNamesList(),
					},
				},
				"required": []string{"server"},
			},
		},
		{
			"name": "hub_search",
			"description": `Search for tools across ALL servers using natural language.

WHEN TO USE: When you need to find a capability but don't know which server has it.

TRIGGERS: 
- "get design", "extract figma", "design info" → searches figma tools
- "create issue", "jira ticket", "bug report" → searches jira tools
- "search documents", "find in wiki", "knowledge base" → searches outline tools
- "take screenshot", "browser automation" → searches playwright/chrome tools

Example queries: "get figma data", "search documents", "create jira issue"`,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Natural language search query describing what you want to do",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name": "hub_execute",
			"description": fmt.Sprintf(`Execute a tool from an MCP server with the given arguments.

WHEN TO USE: After discovering tools with hub_discover, use this to run them.

WORKFLOW:
1. hub_discover(server) → see available tools and their parameters
2. hub_execute(server, tool, arguments) → run the tool

AVAILABLE SERVERS: %s

Example - Get Figma design:
  hub_execute(server="figma", tool="get_figma_data", arguments={"fileKey": "abc123", "nodeId": "1:2"})`, s.getServerNames()),
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
						"description": "Tool name (get from hub_discover)",
					},
					"arguments": map[string]interface{}{
						"type":        "object",
						"description": "Tool arguments (get schema from hub_discover)",
					},
				},
				"required": []string{"server", "tool"},
			},
		},
		{
			"name": "hub_help",
			"description": `Get detailed help, schema, and examples for a specific tool.

WHEN TO USE: When you need parameter details before calling hub_execute.

Returns: Full JSON schema with parameter types, descriptions, and required fields.`,
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
						"description": "Tool name",
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

// buildServerCatalog creates a formatted list of servers with semantic descriptions.
func (s *Server) buildServerCatalog() string {
	catalog := ""
	
	// Semantic descriptions for known servers
	serverDescriptions := map[string]string{
		"figma":              "Design files, UI components, extract design data from Figma links",
		"playwright":         "Browser automation, screenshots, web interactions, testing",
		"chromeDevtools":     "Chrome debugging, DOM inspection, network analysis",
		"mcpOutline":         "Documentation, wiki search, knowledge base queries",
		"outline":            "Documentation, wiki search, knowledge base queries",
		"jira":               "Issue tracking, create/search tickets, project management",
		"github":             "Repositories, pull requests, issues, code search",
		"shadcn":             "UI components, React component generation",
		"sequentialThinking": "Step-by-step reasoning, complex problem solving",
	}
	
	for name := range s.config.Servers {
		desc := serverDescriptions[name]
		if desc == "" {
			desc = "MCP server integration"
		}
		catalog += fmt.Sprintf("  • %s: %s\n", name, desc)
	}
	
	return catalog
}

// getServerNames returns a comma-separated list of server names.
func (s *Server) getServerNames() string {
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
	case "hub_list":
		result, err = s.execHubList()
	case "hub_discover":
		serverName, _ := params.Arguments["server"].(string)
		result, err = s.execHubDiscover(serverName)
	case "hub_search":
		query, _ := params.Arguments["query"].(string)
		result, err = s.execHubSearch(query)
	case "hub_execute":
		serverName, _ := params.Arguments["server"].(string)
		toolName, _ := params.Arguments["tool"].(string)
		args, _ := params.Arguments["arguments"].(map[string]interface{})
		result, err = s.execHubExecute(serverName, toolName, args)
	case "hub_help":
		serverName, _ := params.Arguments["server"].(string)
		toolName, _ := params.Arguments["tool"].(string)
		result, err = s.execHubHelp(serverName, toolName)
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

// execHubList returns a list of registered servers.
func (s *Server) execHubList() (string, error) {
	if len(s.config.Servers) == 0 {
		return "No servers registered. Run 'tool-hub-mcp setup' to import configurations.", nil
	}

	var result string
	result = fmt.Sprintf("Registered MCP Servers (%d):\n", len(s.config.Servers))
	for name, server := range s.config.Servers {
		result += fmt.Sprintf("  • %s (source: %s)\n", name, server.Source)
	}
	return result, nil
}

// execHubDiscover returns tools from a specific server.
func (s *Server) execHubDiscover(serverName string) (string, error) {
	server, exists := s.config.Servers[serverName]
	if !exists {
		return "", fmt.Errorf("server '%s' not found", serverName)
	}

	// Spawn the server and get its tools
	tools, err := s.spawner.GetTools(serverName, server)
	if err != nil {
		return "", fmt.Errorf("failed to discover tools: %w", err)
	}

	result := fmt.Sprintf("Tools from '%s':\n", serverName)
	for _, tool := range tools {
		result += fmt.Sprintf("  • %s: %s\n", tool.Name, tool.Description)
	}
	return result, nil
}

// execHubSearch searches for tools across all servers using keyword matching.
func (s *Server) execHubSearch(query string) (string, error) {
	query = strings.ToLower(query)
	
	// Keyword to server mapping for intelligent routing
	keywordMap := map[string][]string{
		"figma":       {"figma"},
		"design":      {"figma"},
		"ui":          {"figma", "shadcn"},
		"component":   {"figma", "shadcn"},
		"screenshot":  {"playwright", "chromeDevtools"},
		"browser":     {"playwright", "chromeDevtools"},
		"automation":  {"playwright"},
		"test":        {"playwright"},
		"debug":       {"chromeDevtools"},
		"devtools":    {"chromeDevtools"},
		"dom":         {"chromeDevtools"},
		"network":     {"chromeDevtools"},
		"jira":        {"jira"},
		"issue":       {"jira"},
		"ticket":      {"jira"},
		"bug":         {"jira"},
		"sprint":      {"jira"},
		"document":    {"mcpOutline", "outline"},
		"wiki":        {"mcpOutline", "outline"},
		"knowledge":   {"mcpOutline", "outline"},
		"search":      {"mcpOutline", "outline"},
		"github":      {"github"},
		"repo":        {"github"},
		"pull":        {"github"},
		"pr":          {"github"},
		"code":        {"github"},
		"thinking":    {"sequentialThinking"},
		"reasoning":   {"sequentialThinking"},
		"problem":     {"sequentialThinking"},
	}
	
	// Find matching servers
	matchedServers := make(map[string]bool)
	for keyword, servers := range keywordMap {
		if strings.Contains(query, keyword) {
			for _, server := range servers {
				if _, exists := s.config.Servers[server]; exists {
					matchedServers[server] = true
				}
			}
		}
	}
	
	if len(matchedServers) == 0 {
		// No keyword match, return all servers as suggestions
		var result strings.Builder
		result.WriteString(fmt.Sprintf("No direct match for '%s'. Available servers:\n\n", query))
		for name := range s.config.Servers {
			result.WriteString(fmt.Sprintf("  • %s\n", name))
		}
		result.WriteString("\nTry hub_discover(server) to see tools from a specific server.")
		return result.String(), nil
	}
	
	// Return matched servers with recommendation
	var result strings.Builder
	result.WriteString(fmt.Sprintf("For '%s', recommended servers:\n\n", query))
	
	for server := range matchedServers {
		result.WriteString(fmt.Sprintf("  • %s\n", server))
	}
	
	result.WriteString("\nNext step: Call hub_discover(server) to see available tools, then hub_execute to run them.")
	return result.String(), nil
}

// execHubExecute executes a tool from a server.
func (s *Server) execHubExecute(serverName, toolName string, args map[string]interface{}) (string, error) {
	server, exists := s.config.Servers[serverName]
	if !exists {
		return "", fmt.Errorf("server '%s' not found", serverName)
	}

	result, err := s.spawner.ExecuteTool(serverName, server, toolName, args)
	if err != nil {
		return "", fmt.Errorf("failed to execute tool: %w", err)
	}

	return result, nil
}

// execHubHelp returns help for a specific tool.
func (s *Server) execHubHelp(serverName, toolName string) (string, error) {
	server, exists := s.config.Servers[serverName]
	if !exists {
		return "", fmt.Errorf("server '%s' not found", serverName)
	}

	help, err := s.spawner.GetToolHelp(serverName, server, toolName)
	if err != nil {
		return "", fmt.Errorf("failed to get help: %w", err)
	}

	return help, nil
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
