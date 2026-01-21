# tool-hub-mcp: Serverless MCP Aggregator

## Overview

**tool-hub-mcp** is a serverless MCP (Model Context Protocol) aggregator that solves the **context token consumption problem** when using multiple MCP servers. Instead of exposing dozens of individual MCP tools (which consumes 60k+ tokens just for tool definitions), it provides a **single unified MCP endpoint** that proxies requests to child MCP servers on-demand.

## The Problem

When AI clients (Claude Code, OpenCode, Gemini CLI, etc.) connect to multiple MCP servers:
1. **Token Bloat**: Each conversation sends ALL tool definitions to the LLM, consuming significant context window
2. **Startup Latency**: Loading many servers increases initialization time
3. **Management Overhead**: Configuring 10+ MCP servers across different AI clients is tedious

## The Solution

**tool-hub-mcp** acts as a **serverless proxy/aggregator**:
- Exposes only **3-5 meta-tools** (discover, search, execute) instead of 100+ individual tools
- Spawns child MCP servers **on-demand** when tools are actually needed
- **Imports configuration** from existing AI CLI tools (Claude Code, OpenCode) for quick setup
- Runs via `npx`/`uvx` - **no live server required**

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     AI Client (Claude Code, etc.)               │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                   tool-hub-mcp (Single MCP)                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Meta Tools (3-5 tools exposed to AI):                   │   │
│  │  • hub_discover - List available servers/tools           │   │
│  │  • hub_search   - Semantic search for tools              │   │
│  │  • hub_execute  - Execute tool from specific server      │   │
│  │  • hub_list     - List all registered servers            │   │
│  │  • hub_help     - Get help for specific tool             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Config Store (~/.tool-hub-mcp.json)                       │ │
│  │  • Imported from Claude Code / OpenCode                    │ │
│  │  • Unified camelCase format                                │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  On-Demand Server Spawner                                  │ │
│  │  • Spawns child MCP via stdio transport                    │ │
│  │  • Lazy loading - only when tool is executed               │ │
│  │  • Process pooling for frequently used servers             │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                                │
         ┌──────────────────────┼──────────────────────┐
         ▼                      ▼                      ▼
    ┌─────────┐           ┌─────────┐           ┌─────────┐
    │ Jira    │           │ Outline │           │ GitLab  │
    │ MCP     │           │ MCP     │           │ MCP     │
    └─────────┘           └─────────┘           └─────────┘
    (spawned on-demand)
```

## Features

### 1. Setup Wizard (`uvx tool-hub-mcp setup`)

```bash
$ uvx tool-hub-mcp setup

🔍 Scanning for AI CLI tools...

Found configurations:
  ✓ Claude Code (~/.claude.json)          - 5 MCP servers
  ✓ OpenCode (~/.opencode.json)           - 3 MCP servers
  ✓ Gemini CLI (~/.gemini/settings.json)  - 2 MCP servers

Select configurations to import:
  [x] Claude Code: jira-mcp, outline-mcp, gitlab-mcp, filesystem, brave-search
  [x] OpenCode: jira, outline, memory
  [ ] Gemini CLI: (all duplicates)

✓ Imported 8 unique MCP servers to ~/.tool-hub-mcp.json
✓ Transformed to unified camelCase format

Next steps:
  Add tool-hub-mcp to your AI client:
  
  Claude Code:
    claude mcp add tool-hub-mcp -- npx @tool-hub-mcp/cli
  
  OpenCode:
    opencode mcp add tool-hub-mcp --command "npx @tool-hub-mcp/cli"
```

### 2. Config Import Sources

| Source | Config Location | Format |
|--------|-----------------|--------|
| Claude Code | `~/.claude.json`, `.mcp.json` | `mcpServers` with dash-case |
| OpenCode | `~/.opencode.json`, `opencode.json` | `mcp` with camelCase |
| Gemini CLI | `~/.gemini/settings.json` | `mcpServers` |
| Cursor | `~/.cursor/mcp.json` | `mcpServers` |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` | `mcpServers` |

### 3. Config Transformation

All imported configs are normalized to a **unified camelCase standard**:

**Input (Claude Code - dash-case):**
```json
{
  "mcpServers": {
    "jira-mcp": {
      "command": "npx",
      "args": ["-y", "@lvmk/jira-mcp"],
      "env": {
        "JIRA_BASE_URL": "http://jira.example.com"
      }
    }
  }
}
```

**Output (~/.tool-hub-mcp.json - camelCase):**
```json
{
  "servers": {
    "jiraMcp": {
      "source": "claude-code",
      "command": "npx", 
      "args": ["-y", "@lvmk/jira-mcp"],
      "env": {
        "jiraBaseUrl": "http://jira.example.com"
      },
      "metadata": {
        "description": "Jira MCP Server",
        "tools": ["search_issues", "create_issue", "update_issue"]
      }
    }
  },
  "settings": {
    "cacheToolMetadata": true,
    "processPoolSize": 3,
    "timeoutSeconds": 30
  }
}
```

### 4. Meta-Tools Exposed to AI

Instead of exposing 50+ tools, only these **5 meta-tools** are exposed:

| Tool | Description | Example |
|------|-------------|---------|
| `hub_list` | List all registered servers | `hub_list()` → `["jiraMcp", "outlineMcp", "gitlab"]` |
| `hub_discover` | Get tools from a specific server | `hub_discover("jiraMcp")` → `[{name: "search_issues", ...}]` |
| `hub_search` | Semantic search across all tools | `hub_search("find jira tickets")` → `["jiraMcp.search_issues"]` |
| `hub_execute` | Execute a tool from a server | `hub_execute("jiraMcp", "search_issues", {jql: "..."})` |
| `hub_help` | Get detailed help for a tool | `hub_help("jiraMcp", "search_issues")` → `{schema, examples}` |

### 5. Token Savings

**Before (10 MCP servers, ~100 tools):**
- Tool definitions: ~60,000 tokens per request
- Context window consumed: 50%+

**After (tool-hub-mcp, 5 meta-tools):**
- Meta-tool definitions: ~2,000 tokens
- Context window consumed: <5%
- **Savings: 38% token reduction** (measured with 6 MCPs in Claude Code)

## CLI Commands

```bash
# Initial setup - import from AI CLI tools
uvx tool-hub-mcp setup

# List imported servers
uvx tool-hub-mcp list

# Add a server manually
uvx tool-hub-mcp add jira --command "npx -y @lvmk/jira-mcp" \
  --env JIRA_BASE_URL=http://jira.example.com \
  --env JIRA_USERNAME=user --env JIRA_PASSWORD=pass

# Remove a server
uvx tool-hub-mcp remove jira

# Run as MCP server (stdio transport)
uvx tool-hub-mcp serve

# Verify configuration
uvx tool-hub-mcp verify

# Export config for specific AI client
uvx tool-hub-mcp export --client claude-code

# Sync - re-import from AI CLI tools
uvx tool-hub-mcp sync
```

## Implementation Choices

### Language: Go (Primary)

**Rationale (based on benchmarks):**
1. **Startup performance**: 0.88ms (vs Node.js 41.70ms = 47x faster)
2. **Cold start**: ~45ms in serverless (vs Node.js ~1500ms)
3. **Official MCP SDK**: Maintained with Google collaboration
4. **Single binary**: Easy distribution, no runtime dependencies
5. **Cross-compilation**: Simple `GOOS=linux go build` for all platforms

### Package Distribution

```bash
# Direct binary (primary)
curl -fsSL https://github.com/tool-hub-mcp/releases/latest/download/tool-hub-mcp-$(uname -s)-$(uname -m) -o tool-hub-mcp
chmod +x tool-hub-mcp
./tool-hub-mcp setup

# Or via Go install
go install github.com/tool-hub-mcp/cli/cmd/tool-hub-mcp@latest

# Python wrapper (for uvx compatibility)
uvx tool-hub-mcp setup
```

### Project Structure

```
tool-hub-mcp/
├── cmd/
│   └── tool-hub-mcp/           # Main CLI entry point
│       └── main.go
├── internal/
│   ├── config/                 # Configuration management
│   │   ├── loader.go
│   │   ├── transformer.go
│   │   ├── schema.go
│   │   └── sources/
│   │       ├── claude_code.go
│   │       ├── opencode.go
│   │       ├── antigravity.go  # Google Antigravity (new)
│   │       └── sources.go
│   ├── mcp/                    # MCP server implementation
│   │   ├── server.go
│   │   └── tools/
│   │       ├── hub_list.go
│   │       ├── hub_discover.go
│   │       ├── hub_search.go
│   │       ├── hub_execute.go
│   │       └── hub_help.go
│   └── spawner/                # On-demand server spawning
│       ├── pool.go
│       └── spawn.go
├── python/                     # Python uvx wrapper
│   ├── tool_hub_mcp/
│   │   └── __main__.py
│   └── pyproject.toml
├── go.mod
├── go.sum
├── docs/
│   └── README.md
└── plans/
    └── tool-hub-mcp.md
```

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] Project setup (Go module, MCP SDK, Cobra CLI)
- [ ] Configuration schema and loader
- [ ] Config source readers (Claude Code, OpenCode)
- [ ] Unified config transformer (to camelCase)

### Phase 2: MCP Server
- [ ] Basic MCP server with stdio transport
- [ ] `hub_list` meta-tool
- [ ] `hub_discover` meta-tool  
- [ ] `hub_execute` meta-tool

### Phase 3: Advanced Features
- [ ] On-demand server spawner with process pool
- [ ] `hub_search` with semantic matching
- [ ] `hub_help` with cached tool metadata
- [ ] Tool metadata caching

### Phase 4: CLI & Distribution
- [ ] CLI commands (setup, add, remove, list, verify)
- [ ] GitHub releases (multi-platform binaries)
- [ ] Python uvx wrapper
- [ ] Documentation

### Phase 5: Extended Sources
- [ ] Google Antigravity config source
- [ ] Gemini CLI config source
- [ ] Cursor config source
- [ ] Windsurf config source
- [ ] Roo Code config source
