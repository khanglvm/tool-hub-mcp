# tool-hub-mcp: Code Standards & Conventions

**Version:** 1.0.0
**Last Updated:** 2026-01-21
**Status:** Active

## File Organization

### Project Structure

```
tool-hub-mcp/
├── cmd/
│   └── tool-hub-mcp/
│       └── main.go                    # Entry point (83 lines)
├── internal/
│   ├── cli/
│   │   ├── setup.go                   # Setup command (132 lines)
│   │   ├── add.go                     # Add command (403 lines)
│   │   ├── remove.go                  # Remove command (57 lines)
│   │   ├── list.go                    # List command (64 lines)
│   │   ├── verify.go                  # Verify command (51 lines)
│   │   ├── serve.go                   # Serve command (53 lines)
│   │   ├── benchmark.go               # Benchmark command (212 lines)
│   │   ├── learning.go                # Learning command group (40 lines)
│   │   ├── learning-commands.go       # Learning subcommands
│   │   └── learning-utils.go          # Learning utilities
│   ├── mcp/
│   │   └── server.go                  # MCP server core (485 lines)
│   ├── config/
│   │   ├── config.go                  # Config models (155 lines)
│   │   ├── transformer.go             # Name normalization (105 lines)
│   │   └── sources/
│   │       ├── sources.go             # Source interface (51 lines)
│   │       ├── claude_code.go         # Claude Code parser (109 lines)
│   │       └── opencode.go            # OpenCode parser (130 lines)
│   ├── spawner/
│   │   └── pool.go                    # Process pool (327 lines)
│   ├── search/
│   │   ├── indexer.go                 # BM25 indexer (Bleve)
│   │   ├── semantic.go                # Semantic search
│   │   ├── hybrid.go                  # Hybrid ranking
│   │   └── results.go                 # Search result structures
│   ├── learning/
│   │   ├── tracker.go                 # Usage event tracker
│   │   ├── bandit.go                  # ε-greedy bandit algorithm
│   │   ├── scorer.go                  # UCB1 scoring
│   │   └── events.go                  # Event types
│   ├── storage/
│   │   ├── sqlite.go                  # SQLite storage
│   │   └── migrations.go              # Schema migrations
│   └── benchmark/
│       └── benchmark.go               # Token calculations (274 lines)
├── npm/
│   ├── package.json                   # Main npm package
│   ├── cli.js                         # Platform detection wrapper
│   ├── postinstall.js                 # GitHub Releases fallback
│   └── platforms/
│       └── {platform}/bin/
│           └── tool-hub-mcp           # Platform-specific binaries
├── scripts/
│   ├── publish-npm.sh                 # Manual npm publish
│   └── speed_benchmark.sh             # Speed testing
├── docs/
│   ├── project-overview-pdr.md
│   ├── code-standards.md
│   ├── codebase-summary.md
│   ├── system-architecture.md
│   ├── design-guidelines.md
│   ├── deployment-guide.md
│   ├── project-roadmap.md
│   └── facts/                         # Research and analysis docs
├── .github/
│   └── workflows/
│       └── release.yml                # Automated release pipeline
└── README.md                          # User-facing documentation
```

### File Naming Conventions

**Go Files:**
- Use `snake_case` for Go filenames (Go standard)
- Example: `server.go`, `config.go`, `pool.go`

**Documentation:**
- Use `kebab-case` for Markdown files
- Example: `project-overview-pdr.md`, `system-architecture.md`

**Scripts:**
- Use `kebab-case` with descriptive names
- Example: `publish-npm.sh`, `speed_benchmark.sh`

## Code Style Guidelines

### General Principles

1. **YAGNI** (You Aren't Gonna Need It) - Implement only what's needed now
2. **KISS** (Keep It Simple, Stupid) - Prefer simple solutions
3. **DRY** (Don't Repeat Yourself) - Extract common logic

### File Size Management

**Target:** Keep individual code files under 200 lines

**Rationale:**
- Easier to understand and navigate
- Better for token efficiency (LLM context)
- Encourages modular design

**Strategy:**
- Split large files by functional responsibility
- Use composition over inheritance
- Extract utility functions to separate modules
- Create dedicated structs for complex logic

**Examples:**
- `internal/mcp/server.go` (485 lines) - Consider splitting tool handlers
- `internal/cli/add.go` (403 lines) - Could extract JSON parsing logic
- `internal/benchmark/benchmark.go` (274 lines) - Acceptable for related calculations

### Package Organization

**internal/** - Private application code
- `cli/` - Command implementations
- `mcp/` - MCP server logic
- `config/` - Configuration management
- `spawner/` - Process spawning
- `search/` - Semantic search (BM25)
- `learning/` - Usage tracking and bandit algorithm
- `storage/` - SQLite persistence
- `benchmark/` - Performance metrics

**cmd/** - Application entry points
- `tool-hub-mcp/` - Main CLI binary

### Naming Conventions

**Go:**
- **Packages:** `lowercase`, single word when possible
- **Constants:** `PascalCase` or `UPPER_SNAKE_CASE`
- **Variables:** `camelCase`
- **Functions:** `PascalCase` (exported), `camelCase` (private)
- **Interfaces:** `PascalCase` (usually -er suffix)
- **Structs:** `PascalCase`

**Example:**
```go
type Config struct {          // Struct
    Servers map[string]*ServerConfig
    Settings *Settings
}

func NewConfig() *Config {    // Constructor
    return &Config{...}
}

func (c *Config) Load() error { // Method
    // ...
}
```

## Documentation Standards

### Go Doc Comments

**Format:** Standard Go doc comments (preceding declarations)

```go
/*
Package mcp implements the MCP server that exposes meta-tools.

The server uses stdio transport and exposes 2 meta-tools:
  - hub_search: Semantic search for tools across all servers (BM25 + bandit)
  - hub_execute: Execute a tool from a specific server (with learning)
*/
package mcp
```

**Function Documentation:**
```go
// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *config.Config) *Server {
    // ...
}
```

### Inline Comments

**When to Use:**
- Explain **why**, not **what** (code shows what)
- Document non-obvious decisions
- Reference issues or RFCs
- Warn about edge cases

**Example:**
```go
// We use a counter instead of UnixNano to avoid JavaScript precision issues
// (JS Number.MAX_SAFE_INTEGER = 2^53-1 = 9007199254740991)
proc.reqID++
```

### README.md

**Purpose:** User-facing documentation

**Target Size:** <300 lines (main README)

**Sections:**
1. Problem statement
2. Solution overview
3. Benchmark results
4. Installation instructions
5. Quick start guide
6. Usage examples
7. Supported config sources
8. Architecture diagram
9. License

## Testing Standards

### Test File Organization

**Naming:** `{source}_test.go` in same package

**Example:**
- `config.go` → `config_test.go`
- `transformer.go` → `transformer_test.go`

### Test Coverage

**Target:**
- Critical paths: 100% coverage
- Core logic: >80% coverage
- Overall: >70% coverage

**Test Types:**
1. **Unit Tests:** Single function/logic testing
2. **Integration Tests:** Component interaction
3. **End-to-End Tests:** Full workflow (via CLI commands)

**Example:**
```go
func TestToCamelCase(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"dash-case", "dashCase"},
        {"snake_case", "snakeCase"},
        {"PascalCase", "pascalCase"},
    }

    for _, tt := range tests {
        result := ToCamelCase(tt.input)
        if result != tt.expected {
            t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
        }
    }
}
```

### Benchmark Tests

**When to Use:**
- Performance-critical code
- Token calculation logic
- JSON parsing/encoding

**Example:**
```go
func BenchmarkCountTokens(b *testing.B) {
    for i := 0; i < b.N; i++ {
        CountActualToolHubTokens()
    }
}
```

## Error Handling

### Error Wrapping

**Use `fmt.Errorf` with `%w`:**
```go
if err := json.Unmarshal(data, &req); err != nil {
    return nil, fmt.Errorf("invalid JSON-RPC request: %w", err)
}
```

### Error Messages

**Format:** Descriptive, includes context

**Good:**
```go
return fmt.Errorf("failed to spawn server %s: %w", name, err)
```

**Bad:**
```go
return err
```

### Panic Usage

**Avoid panics in production code.** Use only for:
- Truly unrecoverable conditions (e.g., config validation at startup)
- Programmer errors (e.g., nil pointer dereference in tests)

## Concurrency Patterns

### Mutex Usage

**Protect shared state with `sync.Mutex`:**
```go
type Pool struct {
    mu        sync.Mutex
    processes map[string]*Process
}

func (p *Pool) getOrSpawn(name string, cfg *config.ServerConfig) (*Process, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    if proc, exists := p.processes[name]; exists {
        return proc, nil
    }

    // Spawn new process...
}
```

### Atomic Operations

**Use atomic counters for request IDs:**
```go
type Process struct {
    mu    sync.Mutex
    reqID int64  // Atomic counter
}

func (p *Process) nextRequestID() int64 {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.reqID++
    return p.reqID
}
```

### Goroutine Safety

**Always drain stderr to prevent deadlock:**
```go
// Critical: Create stderr pipe and drain it in background
stderr, err := cmd.StderrPipe()
if err != nil {
    return nil, err
}

go func() {
    io.Copy(io.Discard, stderr)
}()
```

## JSON & Serialization

### JSON Tag Conventions

**Use `json` tags with camelCase:**
```go
type Config struct {
    Servers map[string]*ServerConfig `json:"servers"`
    Settings *Settings                `json:"settings,omitempty"`
}
```

### JSON-RPC Messages

**Follow MCP JSON-RPC 2.0 format:**
```go
type MCPRequest struct {
    JSONRPC string          `json:"jsonrpc"`  // Must be "2.0"
    ID      interface{}     `json:"id"`       // Can be string or number
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}
```

## Security Best Practices

### Command Execution

**NEVER use shell string interpolation:**
```go
// BAD - command injection vulnerable
cmd := exec.Command("sh", "-c", fmt.Sprintf("%s %s", cfg.Command, strings.Join(cfg.Args, " ")))

// GOOD - safe separation
cmd := exec.Command(cfg.Command, cfg.Args...)
```

### Environment Variables

**Isolate per process:**
```go
cmd.Env = append(os.Environ(), envs...)
```

### Input Validation

**Validate before use:**
```go
if cfg.Command == "" {
    return fmt.Errorf("server command cannot be empty")
}
```

## Version Management

### Git Tags

**Format:** `v{major}.{minor}.{patch}`

**Example:**
- `v1.0.0` - Initial release
- `v1.0.1` - Bug fix
- `v1.1.0` - New feature
- `v2.0.0` - Breaking change

### Build Information

**Inject via ldflags:**
```bash
go build -ldflags="-X main.version=1.0.1 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%d)"
```

**Access in code:**
```go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
```

## Code Review Checklist

Before committing code, verify:

- [ ] File under 200 lines (or justification for larger)
- [ ] Functions have clear, descriptive names
- [ ] Public types/functions have doc comments
- [ ] Errors are wrapped with context
- [ ] No hardcoded values (use constants)
- [ ] Tests added for new functionality
- [ ] No `panic()` in production code paths
- [ ] Concurrent access protected by mutex
- [ ] JSON tags use camelCase
- [ ] Security considerations addressed (no injection vectors)

## References

- **Go Code Review Comments:** https://github.com/golang/go/wiki/CodeReviewComments
- **Effective Go:** https://go.dev/doc/effective_go
- **MCP Protocol:** https://modelcontextprotocol.io/
- **Project CLAUDE.md:** `/CLAUDE.md`

---

**Owner:** Development Team
**Review Cycle:** Monthly
**Next Review:** 2026-02-21
