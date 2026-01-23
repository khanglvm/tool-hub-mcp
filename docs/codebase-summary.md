# tool-hub-mcp: Codebase Summary

**Version:** 1.2.0
**Last Updated:** 2026-01-23
**Total Files:** 31 files (+2 new)
**Total Tokens:** ~187,000
**Language:** Go 1.22+

## Executive Summary

tool-hub-mcp is a Go-based CLI application that implements a serverless MCP (Model Context Protocol) aggregator. The codebase is well-structured with clear separation of concerns, modular design, and comprehensive documentation.

**Key Characteristics:**
- **Language:** Go (command-line application)
- **Distribution:** Zero-install npm package + Go install
- **Architecture:** Single binary with lazy process spawning
- **Code Size:** ~5,200 lines of Go code (excluding tests)
- **Largest File:** `internal/mcp/server.go` (972 lines, ~7,000 tokens)

## Codebase Structure

### Directory Layout

```
tool-hub-mcp/
├── cmd/tool-hub-mcp/        # Application entry point
├── internal/                # Private application code
│   ├── cli/                 # CLI command implementations (7 files, 972 LOC)
│   ├── mcp/                 # MCP server core (1 file, 485 LOC)
│   ├── config/              # Configuration management (4 files, 395 LOC)
│   ├── spawner/             # Process pool management (1 file, 327 LOC)
│   └── benchmark/           # Performance metrics (1 file, 274 LOC)
├── npm/                     # NPM distribution package
├── scripts/                 # Build and utility scripts
├── docs/                    # Documentation
└── .github/workflows/       # CI/CD pipelines
```

### File Distribution

| Component | Files | Lines of Code | Tokens | Percentage |
|-----------|-------|---------------|--------|------------|
| MCP Server | 1 | 485 | 3,647 | 2.0% |
| CLI Commands | 7 | 972 | 10,821 | 5.9% |
| Config System | 4 | 395 | 3,259 | 1.8% |
| Spawner | 1 | 327 | 2,102 | 1.1% |
| Benchmark | 1 | 274 | 2,334 | 1.3% |
| Documentation | 5+ | 2,000+ | 8,000+ | 4.3% |
| Release Manifest | 1 | 3,782 | 156,562 | 84.8% |
| Other | 9 | 500 | 2,000+ | 1.1% |

## Core Components

### 1. Entry Point (`cmd/tool-hub-mcp/main.go`)

**Lines:** 83
**Purpose:** CLI application initialization

**Key Responsibilities:**
- Version info injection (via ldflags)
- Root command setup (cobra)
- Subcommand registration (7 commands)
- Error handling and exit codes

**Version Variables:**
```go
var (
    version = "dev"      // Set via -ldflags
    commit  = "none"     // Git commit hash
    date    = "unknown"  // Build date
)
```

**Commands Registered:**
1. `setup` - Import MCP configs from AI tools
2. `serve` - Run MCP server
3. `add` - Add server manually
4. `remove` - Remove server
5. `list` - List servers
6. `verify` - Validate config
7. `benchmark` - Token/speed comparison

### 2. MCP Server (`internal/mcp/server.go`)

**Lines:** 972 (+487 lines from v1.0.1)
**Tokens:** ~7,000
**Purpose:** Implements MCP protocol with 2 meta-tools

**Changes in v1.2.0:**
- Added search indexing system integration
- Optimized response format (removed `expectedResponse`, `matchReason`)
- Compact JSON encoding (no indentation)
- Flat response structure (5 fields instead of nested)

**Architecture:**
```go
type Server struct {
    config  *config.Config      // Server configurations
    spawner *spawner.Pool       // Process pool
}
```

**Request Flow:**
```
stdin → JSON-RPC Request → handleRequest()
  → switch on method
    → initialize → handleInitialize()
    → tools/list → handleToolsList()
    → tools/call → handleToolsCall()
  → sendResponse() → stdout
```

**Meta-Tools Implemented:**

| Tool | Purpose |
|------|---------|
| `hub_search` | Semantic search across all servers (BM25 + bandit ranking) |
| `hub_execute` | Execute tool from specific server (with learning) |

**Key Design Decisions:**
- No hardcoded server names (dynamic discovery)
- JSON-RPC 2.0 over stdio
- Lazy spawning (processes start on-demand)
- 60s timeout for npx cold starts
- Compact response format (v1.2.0)

### 3. CLI Commands (`internal/cli/`)

**Total Lines:** 1,150 across 9 files (+2 new in v1.2.0)

#### 3.1 Setup Command (`setup.go`, 132 lines)

**Purpose:** Import configs from AI CLI tools

**Supported Sources:**
- Claude Code (`~/.claude.json`, `.mcp.json`)
- OpenCode (`~/.opencode.json`)
- Google Antigravity (`~/.gemini/antigravity/mcp_config.json`)
- Gemini CLI (`~/.gemini/settings.json`)
- Cursor (`~/.cursor/mcp.json`)
- Windsurf (`~/.codeium/windsurf/mcp_config.json`)

**Workflow:**
1. Scan for config files
2. Parse JSON (format detection)
3. Transform server names to camelCase
4. Merge into unified config
5. Save to `~/.tool-hub-mcp.json`

#### 3.2 Add Command (`add.go`, 403 lines)

**Purpose:** Manually add MCP servers

**Two Modes:**

**Interactive Mode:**
- Paste any MCP config JSON
- Auto-detects format (10+ variations)
- Shows preview with detected servers
- Prompts for confirmation

**Flag Mode:**
```bash
tool-hub-mcp add jira --command npx --arg -y --arg @lvmk/jira-mcp
```

**Smart Parsing:** Handles 40+ key variations
- Command: `command`, `cmd`, `exec`, `run`, `bin`...
- Args: `args`, `arguments`, `argv`, `params`...
- Env: `env`, `environment`, `envVars`...

#### 3.3 List Command (`list.go`, 64 lines)

**Purpose:** Display all registered servers

**Output Format:**
- Server name
- Command with args
- Source (where imported from)
- Environment variable count

**Aliases:** `ls`

#### 3.4 Remove Command (`remove.go`, 57 lines)

**Purpose:** Remove server from config

**Behavior:**
- Tries both original and camelCase names
- Updates config file
- Returns error if not found

**Aliases:** `rm`

#### 3.5 Verify Command (`verify.go`, 51 lines)

**Purpose:** Validate configuration

**Checks:**
- Config file exists
- Valid JSON structure
- Server count
- Command defined for each server

#### 3.6 Serve Command (`serve.go`, 53 lines)

**Purpose:** Run MCP server (stdio transport)

**Flow:**
1. Load config from `~/.tool-hub-mcp.json`
2. Create MCP server instance
3. Start stdio event loop (blocks until stdin closed)

**Usage:**
```bash
# Direct
tool-hub-mcp serve

# Via Claude Code
claude mcp add tool-hub -- tool-hub-mcp serve
```

#### 3.7 Export Index Command (`export-index.go`, 178 lines) **NEW in v1.2.0**

**Purpose:** Export tool index for bash/grep search

**Output Format:** JSONL (newline-delimited JSON)
```jsonl
{"tool":"jira_search","server":"jira","description":"...","inputSchema":{...}}
```

**Auto-regeneration:** Called automatically by `setup`, `add`, `remove` commands

**File Locking:** Uses `flock` (Unix) to prevent concurrent write corruption

**Usage:**
```bash
tool-hub-mcp export-index
grep '"jira"' ~/.tool-hub-mcp-index.jsonl | jq -r '.tool'
```

#### 3.8 Benchmark Command (`benchmark.go`, 212 lines)

**Purpose:** Compare token consumption

**Subcommands:**
- `benchmark` - Token savings calculation
- `benchmark speed` - Latency measurement

**Token Calculation:**
```
Traditional = N servers × 10 tools/server × 150 tokens/tool
tool-hub-mcp = 1 server × 2 meta-tools × actual token count
Savings = Traditional - tool-hub-mcp
```

**Speed Benchmark:**
- Measures spawn + tools/list latency
- Runs N iterations (default: 3)
- Calculates average per server

### 4. Configuration System (`internal/config/`)

**Total Lines:** 395 across 4 files

#### 4.1 Data Models (`config.go`, 155 lines)

**Structures:**
```go
type Config struct {
    Servers  map[string]*ServerConfig  // camelCase keys
    Settings *Settings
}

type ServerConfig struct {
    Command  string
    Args     []string
    Env      map[string]string
    Source   string  // Import source
    Metadata *ServerMetadata  // Cached tool info
}

type Settings struct {
    CacheToolMetadata bool
    ProcessPoolSize   int  // Default: 3
    TimeoutSeconds    int  // Default: 30
}
```

**Config Path:** `~/.tool-hub-mcp.json`

**Operations:**
- `Load()` - Read from default path
- `LoadFrom(path)` - Custom path
- `Save(cfg, path)` - Write with indentation
- `NewConfig()` - Initialize with defaults

#### 4.2 Name Transformer (`transformer.go`, 105 lines)

**Purpose:** Normalize naming conventions to camelCase

**Supported Formats:**
- `dash-case` → `dashCase`
- `snake_case` → `snakeCase`
- `PascalCase` → `pascalCase`
- Already camelCase → unchanged

**Implementation:**
```go
func ToCamelCase(s string) string {
    words := splitWords(s)  // Split on '-', '_', ' ', case transitions
    if len(words) == 0 {
        return ""
    }
    words[0] = strings.ToLower(words[0])
    for i := 1; i < len(words); i++ {
        words[i] = strings.Title(words[i])
    }
    return strings.Join(words, "")
}
```

**Env Var Normalization:**
```go
func ToEnvVarCase(s string) string {
    // "apiKey" → "API_KEY"
    return strings.ToUpper(ToCamelCase(s))
}
```

#### 4.3 Config Sources (`sources/`)

**Interface:** (`sources.go`, 51 lines)
```go
type Source interface {
    Name() string
    Scan() (*SourceResult, error)
}

type SourceResult struct {
    ConfigPath string
    Servers    map[string]*config.ServerConfig
}
```

**Claude Code Source** (`claude_code.go`, 109 lines):
- Paths: `~/.claude.json`, `.mcp.json`
- Format: `{"mcpServers": {...}}`
- Transform: `mcpServers` → `Servers`

**OpenCode Source** (`opencode.go`, 130 lines):
- Paths: `~/.opencode.json`, `opencode.json`, `~/.config/opencode/opencode.json`
- Format: `{"mcp": {"serverName": {...}}}`
- Transform: `mcp` → `Servers`, skip disabled/remote

### 5. Process Spawner (`internal/spawner/pool.go`)

**Lines:** 327
**Tokens:** 2,102
**Purpose:** Lazy spawning and management of child MCP servers

**Architecture:**
```go
type Pool struct {
    maxSize   int
    mu        sync.Mutex
    processes map[string]*Process
}

type Process struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout *bufio.Reader
    mu     sync.Mutex
    reqID  int64  // Atomic counter (safe for JS)
}
```

**Lifecycle:**

```
GetOrSpawn(name, cfg)
  → Check pool for existing process
  → If not found: Spawn()
    → exec.Command(cfg.Command, cfg.Args...)
    → Create stdin, stdout, stderr pipes
    → Start process
    → Drain stderr in goroutine (prevent deadlock)
    → Initialize() - MCP handshake
      → Send initialize request
      → Send initialized notification
    → Add to pool
  → Return process
```

**Critical Bug Fix:**
```go
// CRITICAL: Create stderr pipe and drain it in background
// to prevent pipe buffer deadlock. Some MCPs write to stderr
// during startup and if the buffer fills up (~64KB), it blocks
// the entire process including stdout.
stderr, err := cmd.StderrPipe()
go func() {
    io.Copy(io.Discard, stderr)
}()
```

**Safe Request IDs:**
```go
// Use atomic counter instead of UnixNano to avoid
// JavaScript precision issues (MAX_SAFE_INTEGER = 2^53-1)
proc.reqID++
reqID := proc.reqID
```

**Timeout:** 60 seconds (handles npx package downloads)

**Public API:**
- `GetTools(name, cfg)` - Call `tools/list`
- `ExecuteTool(name, tool, args)` - Call `tools/call`
- `GetToolHelp(name, tool)` - Get parameter schema

### 6. Benchmark System (`internal/benchmark/benchmark.go`)

**Lines:** 274
**Tokens:** 2,334
**Purpose:** Calculate token consumption and performance

**Token Estimation Constants:**
```go
const (
    AverageToolsPerServer = 10
    AverageTokensPerTool  = 150
    ToolHubTools          = 5
)
```

**Known Tool Counts:** (for accuracy)
- High: chromeDevtools (35), github (40), outline (32)
- Medium: jira (13), linear (15), notion (25)
- Low: sequentialThinking (1), webReader (2)

**Calculation:**
```go
traditional := totalTools * AverageTokensPerTool
toolHub := 5 * actualTokenCount
savings := traditional - toolHub
percentage := (float64(savings) / float64(traditional)) * 100
```

**Display Format:** ASCII art table with borders

## Distribution Strategy

### NPM Zero-Install

**Pattern:** `optionalDependencies` (esbuild/Biome/Turbo pattern)

**Architecture:**
```
@khanglvm/tool-hub-mcp (main package)
├── cli.js (thin wrapper)
├── postinstall.js (fallback)
└── optionalDependencies:
    ├── @khanglvm/tool-hub-mcp-darwin-arm64
    ├── @khanglvm/tool-hub-mcp-darwin-x64
    ├── @khanglvm/tool-hub-mcp-linux-x64
    ├── @khanglvm/tool-hub-mcp-linux-arm64
    └── @khanglvm/tool-hub-mcp-win32-x64
```

**Platform Detection:** (`cli.js`)
```javascript
const platform = os.platform() + '-' + os.arch()
// e.g., "darwin-arm64", "linux-x64", "win32-x64"
```

**Fallback:** If optionalDependencies disabled, download from GitHub Releases

### Build & Release

**GitHub Actions:** (`.github/workflows/release.yml`)
- Trigger: Git tags matching `v*`
- Jobs: Build (matrix) → Release → npm-publish
- Platforms: darwin/arm64, darwin/x64, linux/x64, linux/arm64, windows/x64

**Manual Script:** (`scripts/publish-npm.sh`)
- Build Go binaries with optimization flags
- Version management (semver bumping)
- Publish platform packages first, then main

## Testing

### Test Files

- `internal/config/transformer_test.go` (133 lines)
  - TestToCamelCase (9 cases)
  - TestToEnvVarCase (4 cases)
  - TestNormalizeEnvVars

- `internal/config/config_test.go` (91 lines)
  - TestNewConfig
  - TestSaveAndLoad
  - TestLoadNonExistent

### Coverage

**Current:** Good coverage of critical paths
**Missing:**
- Spawner lifecycle tests (hard without real MCPs)
- Source parsing tests (needs fixtures)
- Integration tests with mock MCP servers

## Dependencies

### External Libraries

1. **github.com/spf13/cobra**
   - Purpose: CLI framework
   - Used in: All commands
   - Why: Industry standard for Go CLIs

### Standard Library

- `encoding/json` - JSON-RPC parsing
- `bufio` - Stdio line reading
- `fmt` - Output formatting
- `os` - File I/O, stdin/stdout
- `os/exec` - Process spawning
- `sync` - Mutex, atomic operations
- `time` - Benchmark timing
- `strings` - String manipulation
- `path/filepath` - Path operations
- `io` - I/O interfaces
- `io/ioutil` - File utilities

## Performance Characteristics

### Startup Time
- Config load: <10ms (JSON parse)
- First tool spawn: ~100ms (warm), ~30s (npx cold)
- Subsequent calls: <50ms (process reuse)

### Memory Usage
- Config: ~1KB per server
- Process: ~5-10MB per active server
- Pool: Configurable (default: 3 processes)

### Scalability
- Tested with 10+ servers
- Linear memory growth
- No global bottlenecks

## Code Quality Metrics

### File Size Distribution

| File | Lines | Status |
|------|-------|--------|
| `internal/mcp/server.go` | 972 | ⚠️ Large but cohesive |
| `internal/cli/add.go` | 403 | ⚠️ Consider splitting |
| `internal/spawner/pool.go` | 327 | ✅ Acceptable |
| `internal/benchmark/benchmark.go` | 274 | ✅ Acceptable |
| `internal/cli/export-index.go` | 178 | ✅ Good |
| All other files | <200 | ✅ Good |

### Token Efficiency

**Top 5 Files by Token Count:**
1. `release-manifest.json` - 156,562 tokens (83.7%)
2. `internal/mcp/server.go` - ~7,000 tokens (3.7%)
3. `internal/cli/add.go` - 2,968 tokens (1.6%)
4. `internal/benchmark/benchmark.go` - 2,334 tokens (1.2%)
5. `internal/spawner/pool.go` - 2,102 tokens (1.1%)

**Total Go Code:** ~17,000 tokens (9.1%)
**Total Project:** ~187,000 tokens

## Key Design Patterns

1. **Process Pool Pattern** - Lazy spawning with reuse
2. **Aggregator Pattern** - Single endpoint, meta-tools
3. **Dynamic Discovery** - No hardcoding, runtime building
4. **Flexible Parsing** - 40+ config format variations
5. **stdio Transport** - Standard MCP JSON-RPC over stdin/stdout

## New in v1.2.0

### Features Added
- ✅ Export index command (`export-index.go`, 178 lines)
- ✅ Auto-regeneration hooks in setup/add/remove commands
- ✅ File locking for concurrent write protection
- ✅ Bash/grep usage examples in CLI help

### Optimizations
- ✅ Compact JSON encoding (no indentation)
- ✅ Removed redundant response fields
- ✅ Flat response structure (5 fields)
- ✅ 43-70% token savings per search

## Known Issues & Limitations

### High Priority
- [ ] Split `add.go` (403 lines) extract JSON parsing
- [ ] Config merge strategy for duplicates

### Medium Priority
- [ ] Metadata refresh mechanism
- [ ] Pool eviction policy (LRU/TTL)
- [ ] Remote MCP support (OpenCode format)

### Low Priority
- [ ] Integration tests with mock MCPs
- [ ] Metrics and observability

## References

- **Scout Reports:** `/plans/reports/scout-*.md`
- **Fact Documentation:** `/docs/facts/*.md`
- **Project README:** `/README.md`
- **MCP Protocol:** https://modelcontextprotocol.io/

---

**Owner:** Development Team
**Review Cycle:** Monthly
**Next Review:** 2026-02-23
