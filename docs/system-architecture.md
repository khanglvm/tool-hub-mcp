# tool-hub-mcp: System Architecture

**Version:** 1.0.1
**Last Updated:** 2026-01-21
**Status:** Active

## Overview

tool-hub-mcp implements a serverless aggregator pattern for the Model Context Protocol (MCP). The system exposes **2 meta-tools** instead of loading all tool definitions from registered MCP servers, achieving 38-96% reduction in AI context token consumption through semantic search and intelligent learning-based ranking.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         AI Client                                │
│              (Claude Code, OpenCode, Cursor, etc.)               │
└────────────────────────────┬────────────────────────────────────┘
                             │ JSON-RPC 2.0 over stdio
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      tool-hub-mcp                                │
│  ┌────────────────────────────────────────────────────────┐     │
│  │              MCP Server (stdio transport)              │     │
│  │  ┌─────────────────────┬───────────────────────────┐   │     │
│  │  │   hub_search        │   hub_execute            │   │     │
│  │  │ (BM25 + bandit)     │ (with learning tracker)  │   │     │
│  │  └─────────────────────┴───────────────────────────┘   │     │
│  └────────────────────────────────────────────────────────┘     │
│  ┌────────────────────────────────────────────────────────┐     │
│  │               Process Pool Manager                     │     │
│  │        (max 3 concurrent child processes)              │     │
│  └────────────────────────────────────────────────────────┘     │
└────────────────────────────┬────────────────────────────────────┘
                             │ On-demand lazy spawn
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
    ┌──────────┐        ┌──────────┐        ┌──────────┐
    │   Jira   │        │  Figma   │        │Playwright│
    │   MCP    │        │   MCP    │        │   MCP    │
    │(npx pkg) │        │(npx pkg) │        │(npx pkg) │
    └──────────┘        └──────────┘        └──────────┘
```

## Core Components

### 1. CLI Layer (`internal/cli/`)

**Purpose:** User-facing command interface

**Commands:**

| Command | File | Lines | Responsibility |
|---------|------|-------|----------------|
| `setup` | setup.go | 132 | Import configs from AI tools |
| `add` | add.go | 403 | Manual server addition |
| `remove` | remove.go | 57 | Remove server from config |
| `list` | list.go | 64 | Display registered servers |
| `verify` | verify.go | 51 | Validate configuration |
| `serve` | serve.go | 53 | Start MCP server |
| `benchmark` | benchmark.go | 212 | Performance analysis |
| `learning` | learning.go | 40 | Learning system management |

**Framework:** spf13/cobra
**Pattern:** One command per file, returns `*cobra.Command`

### 2. MCP Server Layer (`internal/mcp/server.go`)

**Purpose:** Implements MCP protocol with 2 meta-tools

**Transport:** JSON-RPC 2.0 over stdio
**Protocol Version:** 2024-11-05

**Request Processing Flow:**

```
stdin (JSON-RPC Request)
  ↓
bufio.Scanner (line-by-line)
  ↓
handleRequest(data []byte)
  ↓
  ├── initialize → handleInitialize()
  ├── tools/list → handleToolsList()
  └── tools/call → handleToolsCall()
      ↓
      Parse tool name
      ↓
      Route to executor:
        ├── hub_search → execHubSearch()
        └── hub_execute → execHubExecute()
  ↓
sendResponse() → stdout (JSON-RPC Response)
```

**Server Structure:**
```go
type Server struct {
    config  *config.Config      // Server configurations
    spawner *spawner.Pool       // Process pool (default: 3)
    indexer *search.Indexer     // BM25 search indexer
    storage *storage.SQLiteStorage // Learning data storage
    tracker *learning.Tracker   // Usage tracking & ranking
}
```

### 3. Process Spawner (`internal/spawner/pool.go`)

**Purpose:** Manage lifecycle of child MCP processes

**Architecture:** Lazy spawning with process pool

**Pool Structure:**
```go
type Pool struct {
    maxSize   int                       // Max concurrent processes
    mu        sync.Mutex                // Protects processes map
    processes map[string]*Process       // Active processes by name
}

type Process struct {
    cmd    *exec.Cmd                   // OS command
    stdin  io.WriteCloser              // JSON-RPC requests
    stdout *bufio.Reader               // JSON-RPC responses
    mu     sync.Mutex                  // Protects reqID & I/O
    reqID  int64                       // Atomic request ID counter
}
```

**Lifecycle:**

```
GetOrSpawn(name, cfg)
  ↓
Check pool for existing process
  ↓ (not found)
Spawn()
  ↓
exec.Command(cfg.Command, cfg.Args...)
  ↓
Create pipes: stdin, stdout, stderr
  ↓
Start process
  ↓
Drain stderr (background goroutine)
  ↓
Initialize() - MCP handshake
  ├── Send: {"method":"initialize", ...}
  └── Send: {"method":"notifications/initialized"}
  ↓
Add to pool
  ↓
Return process
```

**Critical Design Decisions:**

1. **Safe Request IDs:**
   ```go
   // Atomic counter, NOT UnixNano
   // Avoids JS precision loss (MAX_SAFE_INTEGER = 2^53-1)
   proc.reqID++
   reqID := proc.reqID
   ```

2. **Stderr Draining:**
   ```go
   // Prevents pipe buffer deadlock (~64KB limit)
   go func() {
       io.Copy(io.Discard, stderr)
   }()
   ```

3. **60s Timeout:**
   - Handles npx package downloads on cold start
   - Configurable via `Settings.TimeoutSeconds`

### 4. Search System (`internal/search/`)

**Purpose:** Semantic search and ranking for tool discovery

**Components:**
- **BM25 Indexer** (`indexer.go`): Full-text search using Bleve
- **Semantic Search** (`semantic.go`): Query matching and scoring
- **Hybrid Ranking** (`hybrid.go`): Combined BM25 + bandit scores
- **Results** (`results.go`): Search result structures with tracking

**Indexing Flow:**
```
Server.IndexTools()
  ↓
For each server:
  Spawn process → tools/list
  Extract tool metadata (name, description, schema)
  Build Bleve index fields
  Store in-memory index
  ↓
Total indexed count logged
```

**Search Flow:**
```
hub_search(query)
  ↓
BM25 search → candidate tools
  ↓
Apply bandit scores (if learning enabled)
  ↓
Re-rank by combined score
  ↓
Return ranked results + searchId
```

**Storage:** In-memory Bleve index (rebuilt on restart)

### 5. Learning System (`internal/learning/`)

**Purpose:** Track tool usage and provide intelligent ranking

**Components:**
- **Tracker** (`tracker.go`): Event recording and query interface
- **Bandit** (`bandit.go`): ε-greedy multi-armed bandit algorithm
- **Scorer** (`scorer.go`): Calculate UCB scores for tools
- **Events** (`events.go`): Event types (Search, Execute, Feedback)

**Algorithm:** ε-greedy with UCB1 (Upper Confidence Bound)
```
score = average_reward + exploration_bonus
exploration_bonus = sqrt(2 * ln(total_trials) / trials)

ε = 0.1 (10% exploration, 90% exploitation)
```

**Storage:** SQLite database at `~/.tool-hub-mcp/history.db`

**Schema:**
```sql
CREATE TABLE usage_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    context_hash TEXT NOT NULL,
    search_id TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tool_name ON usage_events(tool_name);
CREATE INDEX idx_context_hash ON usage_events(context_hash);
```

**Privacy:** Contexts hashed with SHA256 before storage

### 6. Storage Layer (`internal/storage/`)

**Purpose:** Persistent storage for learning data

**Components:**
- **SQLite Storage** (`sqlite.go`): Database operations
- **Migrations** (`migrations.go`): Schema versioning

**Location:** `~/.tool-hub-mcp/history.db`

**Features:**
- Auto-initialization on first use
- Optional (graceful degradation if unavailable)
- Thread-safe access

### 7. CLI Learning Commands (`internal/cli/learning*.go`)

**Purpose:** User management of learning system

**Commands:**
- `learning status` - Show usage statistics and top tools
- `learning export` - Export usage history as JSON
- `learning clear` - Delete all learning data
- `learning disable` - Turn off tracking (temporary)
- `learning enable` - Turn on tracking

**Status Output:**
```
Learning System Status: ENABLED
Database: ~/.tool-hub-mcp/history.db
Total Events: 1,234
Unique Tools: 45

Top Tools (last 30 days):
1. jira.search_issues (89 uses, 4.2 avg score)
2. figma.get_file (67 uses, 4.5 avg score)
...
```

### 8. Configuration System (`internal/config/`)

**Purpose:** Unified storage and transformation of MCP server configs

**Storage Path:** `~/.tool-hub-mcp.json`

**Schema:**
```json
{
  "servers": {
    "serverName": {
      "command": "npx",
      "args": ["-y", "@package/name"],
      "env": {"KEY": "value"},
      "source": "claude-code",
      "metadata": {
        "description": "Server description",
        "tools": ["tool1", "tool2"],
        "lastUpdated": "2026-01-21"
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

**Name Normalization:**
- All server names converted to camelCase
- Input formats: dash-case, snake_case, PascalCase
- Output: camelCase (unified)

**Config Sources:**

| Source | Path(s) | Format |
|--------|---------|--------|
| Claude Code | `~/.claude.json`, `.mcp.json` | `{"mcpServers": {...}}` |
| OpenCode | `~/.opencode.json`, `opencode.json` | `{"mcp": {...}}` |
| Google Antigravity | `~/.gemini/antigravity/mcp_config.json` | Custom |
| Gemini CLI | `~/.gemini/settings.json` | Custom |
| Cursor | `~/.cursor/mcp.json` | Custom |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` | Custom |

## Meta-Tools Architecture

### Tool Definitions

The 2 meta-tools are defined at runtime with rich input schemas:

```go
tools := []Tool{
    {
        Name: "hub_search",
        Description: "Find the right tool for any task across all registered MCP servers using semantic search...",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "query": map[string]interface{}{
                    "type": "string",
                    "description": "Natural language description of what you want to do",
                },
            },
        },
    },
    {
        Name: "hub_execute",
        Description: "Execute a tool from a specific MCP server...",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "server": map[string]interface{}{
                    "type": "string",
                    "description": "Server name (from hub_search result)",
                },
                "tool": map[string]interface{}{
                    "type": "string",
                    "description": "Tool name (from hub_search result)",
                },
                "arguments": map[string]interface{}{
                    "type": "object",
                    "description": "Tool arguments (schema from hub_search result)",
                },
                "searchId": map[string]interface{}{
                    "type": "string",
                    "description": "Optional: search session ID from hub_search to link execution for learning",
                },
            },
        },
    },
}
```

**Key Design:** No hardcoded server names. AI must call `hub_search` to discover tools with intelligent ranking.

### AI Interaction Workflow

```
1. User: "Search Jira issues"
   ↓
2. AI: hub_search(query="search jira issues")
   ↓
3. tool-hub: BM25 search → jira.search_issues (score: 8.5)
   ↓
4. tool-hub: Return ranked results + searchId ("uuid-123")
   ↓
5. AI: hub_execute(
       server="jira",
       tool="search_issues",
       args={jql: "..."},
       searchId="uuid-123"
   )
   ↓
6. tool-hub: Record execution event (for learning)
   ↓
7. tool-hub: [Spawn Jira MCP if needed] → tools/call
   ↓
8. tool-hub: [Results]
   ↓
9. AI: Response to user
```

**Learning Feedback Loop:**
```
Tool execution recorded
  ↓
Update bandit statistics (trials, rewards)
  ↓
Next search: better ranking for successful tools
  ↓
Improved tool discovery over time
```

## Data Flow

### 1. Setup Flow

```
tool-hub-mcp setup
  ↓
Scan for config files:
  ├── ~/.claude.json
  ├── ~/.opencode.json
  └── ...
  ↓
Parse each format (auto-detect)
  ↓
Transform server names → camelCase
  ↓
Merge into unified Config
  ↓
Save to ~/.tool-hub-mcp.json
  ↓
Display next steps (add to AI client)
```

### 2. Serve Flow

```
tool-hub-mcp serve
  ↓
Load config from ~/.tool-hub-mcp.json
  ↓
Create MCP Server with Process Pool (size: 3)
  ↓
Start stdio event loop:
  ├── Read line from stdin
  ├── Parse JSON-RPC request
  ├── Route to handler
  ├── Generate response
  └── Write to stdout
  ↓
Loop until stdin closed
```

### 3. Tool Execution Flow

```
AI: tools/call with hub_execute
  ↓
Parse args: server, tool, arguments
  ↓
spawner.GetOrSpawn(server, cfg)
  ├── Check pool for existing process
  ├── If not found: spawn new process
  └── Return process
  ↓
process.sendRequest("tools/call", {
  "name": tool,
  "arguments": arguments
})
  ↓
Wait for response (60s timeout)
  ↓
Return result to AI
```

## Performance Optimization

### Token Efficiency

**Traditional Approach:**
```
N MCP Servers
  × ~10 Tools/Server
  × ~150 Tokens/Tool
  = 60,000+ Tokens
```

**tool-hub-mcp Approach:**
```
1 Server (tool-hub-mcp)
  × 5 Meta-Tools
  × ~150 Tokens/Tool
  = ~750 Tokens
```

**Verified Results:**
- 6 servers: 48,371 → 29,758 tokens (38.48% reduction)
- 5 servers: 95.0% reduction (15,150 → 461 tokens)

### Latency Optimization

**Cold Start:** First tool call per server
```
Spawn process: ~100ms
Initialize MCP: ~200ms
tools/list: ~500ms
Total: ~800ms
```

**Warm Start:** Subsequent calls (process reuse)
```
tools/list: ~50ms (process already running)
```

**Average:** 307ms across 5 servers (includes cold/warm mix)

### Process Pool Strategy

**Default Size:** 3 concurrent processes

**Eviction:** None (processes stay alive until hub shutdown)
**Future:** LRU/TTL-based eviction

**Memory:**
- Per process: ~5-10MB
- Pool (3 processes): ~15-30MB

## Security Architecture

### Command Injection Prevention

**Bad Approach (shell interpolation):**
```go
cmd := exec.Command("sh", "-c", fmt.Sprintf("%s %s", command, args))
// VULNERABLE to injection
```

**Good Approach (separate args):**
```go
cmd := exec.Command(cfg.Command, cfg.Args...)
// SAFE - no shell involved
```

### Environment Variable Isolation

```go
cmd.Env = append(os.Environ(), envs...)
// Each process gets isolated env vars
// No cross-contamination
```

### Resource Limits

**Process Pool:** Max concurrent processes (default: 3)
**Timeout:** 60s per request (prevents hanging)
**Auto-cleanup:** Failed processes killed immediately

## Error Handling

### Request/Response Errors

**JSON-RPC Error Format:**
```go
type MCPError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}
```

**Error Categories:**
1. **Parse Errors:** Invalid JSON-RPC
2. **Method Errors:** Unknown method
3. **Tool Errors:** Tool not found, execution failed
4. **Timeout Errors:** 60s limit exceeded
5. **Spawn Errors:** Process creation failed

### Process Failure Handling

```
Spawn failure
  ↓
Kill process immediately
  ↓
Return error to AI
  ↓
AI can retry or report to user
```

**No Automatic Retry:** Failed processes not retried (future enhancement)

## Concurrency Model

### Mutex Protection

**Pool-level:** Protects `processes` map
```go
p.mu.Lock()
defer p.mu.Unlock()
// Access p.processes
```

**Process-level:** Protects request ID generation and I/O
```go
proc.mu.Lock()
defer proc.mu.Unlock()
proc.reqID++
```

### Goroutine Usage

**Stderr Draining:**
```go
go func() {
    io.Copy(io.Discard, stderr)
}()
```

**Response Reading:**
```go
responseCh := make(chan []byte, 1)
errorCh := make(chan error, 1)

go func() {
    data, err := readResponse()
    if err != nil {
        errorCh <- err
        return
    }
    responseCh <- data
}()

select {
case data := <-responseCh:
    return data
case err := <-errorCh:
    return err
case <-time.After(60 * time.Second):
    return timeoutError
}
```

## Distribution Architecture

### Zero-Install Pattern

**Reference:** esbuild, Biome, Turbo, SWC, Parcel

```
@khanglvm/tool-hub-mcp (main package)
├── package.json
├── cli.js (thin wrapper)
├── postinstall.js (fallback)
└── optionalDependencies:
    ├── @khanglvm/tool-hub-mcp-darwin-arm64
    ├── @khanglvm/tool-hub-mcp-darwin-x64
    ├── @khanglvm/tool-hub-mcp-linux-x64
    ├── @khanglvm/tool-hub-mcp-linux-arm64
    └── @khanglvm/tool-hub-mcp-win32-x64
```

**How It Works:**

1. **Installation (`npm install` / `npx`)**
   - npm auto-selects matching platform package
   - Only matching platform downloaded (bandwidth savings)

2. **Fallback (postinstall.js)**
   - Checks if binary exists
   - If missing: Downloads from GitHub Releases
   - Extracts to `npm/bin/`

3. **CLI Execution (`npx @khanglvm/tool-hub-mcp serve`)**
   - Node.js runs `cli.js`
   - Platform detection → package name
   - Searches multiple paths for binary
   - Spawns Go binary with `stdio: 'inherit'`

### Build Pipeline

**GitHub Actions** (`.github/workflows/release.yml`)

```
Git tag pushed (v*)
  ↓
Job 1: Build (matrix strategy)
  ├── darwin-arm64: GOOS=darwin GOARCH=arm64 go build
  ├── darwin-x64: GOOS=darwin GOARCH=amd64 go build
  ├── linux-x64: GOOS=linux GOARCH=amd64 go build
  ├── linux-arm64: GOOS=linux GOARCH=arm64 go build
  └── win32-x64: GOOS=windows GOARCH=amd64 go build
  ↓
Upload artifacts (binaries)
  ↓
Job 2: Release
  ├── Download all artifacts
  ├── Create GitHub Release
  └── Attach binaries
  ↓
Job 3: npm-publish
  ├── Setup Node.js 20
  ├── Extract version from tag (strip 'v')
  ├── Copy binaries to npm/platforms/{platform}/bin/
  ├── npm version {version} --no-git-tag-version (all 6 packages)
  ├── npm publish (platform packages)
  └── npm publish (main package)
```

**Manual Override:** `scripts/publish-npm.sh` for local testing

## Monitoring & Observability

### Current Capabilities

- `tool-hub-mcp verify` - Config validation
- `tool-hub-mcp list` - Server overview
- `tool-hub-mcp benchmark` - Token analysis
- `tool-hub-mcp benchmark speed` - Latency measurement

### Missing Features

- [ ] Structured logging
- [ ] Metrics collection (spawn count, error rate)
- [ ] Health checks
- [ ] Process pool statistics

## Scalability Considerations

### Current Limits

- **Max Concurrent Processes:** 3 (configurable)
- **Timeout:** 60s per request
- **Servers Tested:** 10+

### Bottlenecks

1. **Process Spawn Time:** ~100ms (unavoidable with external processes)
2. **Sequential tool/list:** No parallelization
3. **No connection pooling:** Each request spawns if needed

### Future Improvements

1. **Parallel Discovery:** Fetch tool lists concurrently
2. **Connection Keep-Alive:** Reuse processes beyond current session
3. **Adaptive Pool Size:** Adjust based on load

## Architecture Diagrams

### Component Interaction

```
┌─────────────────────────────────────────────────────────────┐
│                        AI Client                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  MCP Client Library (JSON-RPC over stdio)           │   │
│  └──────────────┬───────────────────────────────────────┘   │
└─────────────────┼───────────────────────────────────────────┘
                  │ stdin/stdout
┌─────────────────▼───────────────────────────────────────────┐
│                    tool-hub-mcp                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  MCP Server (server.go)                              │   │
│  │  ┌────────┬────────┬────────┬─────────┬─────────┐   │   │
│  │  │hub_list│discover│ search │execute  │  help   │   │   │
│  │  └────────┴────────┴────────┴─────────┴─────────┘   │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         ▼                                     │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Process Pool (pool.go)                              │   │
│  │  ┌──────────┬──────────┬──────────┐                 │   │
│  │  │ Process  │ Process  │ Process  │                 │   │
│  │  │ (jira)   │ (figma)  │ (chrome) │                 │   │
│  │  └──────────┴──────────┴──────────┘                 │   │
│  └──────────────────────┬───────────────────────────────┘   │
└─────────────────────────┼───────────────────────────────────┘
                        │ stdin/stdout per process
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │  Jira   │    │  Figma  │    │ Chrome  │
   │  MCP    │    │   MCP   │    │   MCP   │
   │ Server  │    │ Server  │    │ Server  │
   └─────────┘    └─────────┘    └─────────┘
```

### Request Flow

```
AI Client                    tool-hub-mcp               Child MCP
    │                              │                        │
    │─── JSON-RPC Request ────────>│                        │
    │  (tools/call, hub_execute)   │                        │
    │                              │                        │
    │                              ├─── Spawn Process ─────>│
    │                              │    (if not running)    │
    │                              │                        │
    │                              │<─ Ready ───────────────│
    │                              │                        │
    │                              ├─── JSON-RPC ──────────>│
    │                              │  (tools/call)          │
    │                              │                        │
    │                              │<─ JSON-RPC Response ───│
    │                              │                        │
    │<─ JSON-RPC Response ─────────│                        │
    │  (tool result)               │                        │
```

## References

- **MCP Protocol:** https://modelcontextprotocol.io/
- **JSON-RPC 2.0:** https://www.jsonrpc.org/specification
- **Cobra CLI:** https://github.com/spf13/cobra
- **Scout Reports:** `/plans/reports/scout-*.md`

---

**Owner:** Development Team
**Review Cycle:** Quarterly
**Next Review:** 2026-04-21
