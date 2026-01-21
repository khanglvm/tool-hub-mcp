# tool-hub-mcp: Design Guidelines

**Version:** 1.0.0
**Last Updated:** 2026-01-21
**Status:** Active

## Core Design Principles

### 1. Token Efficiency (Primary Goal)

**Principle:** Minimize AI context token consumption

**Implementation:**
- Expose 5 meta-tools instead of all tools from all servers
- Lazy loading: Tool definitions fetched on-demand
- Generic descriptions without hardcoded server names
- Dynamic discovery: AI calls `hub_list` first

**Trade-off:** Slightly more complex AI workflow for massive token savings

### 2. Simplicity (KISS)

**Principle:** Keep designs simple and straightforward

**Guidelines:**
- Prefer simple solutions over clever ones
- Avoid premature optimization
- Use standard library when possible
- Clear code over clever code

**Examples:**
- Use `exec.Command` instead of process management libraries
- Manual JSON parsing instead of code generation
- Direct stdio instead of transport abstractions

### 3. Flexibility Over Hardcoding

**Principle:** Support diverse configurations without code changes

**Implementation:**
- No hardcoded server names in tool definitions
- Format detection: 40+ config variations
- Name normalization: dash-case, snake_case, PascalCase → camelCase
- Extensible source system (easy to add new AI clients)

**Trade-off:** More complex parsing logic, but infinitely more flexible

### 4. Performance

**Principle:** Optimize for real-world usage patterns

**Strategies:**
- Process pooling: Reuse spawned processes
- Safe request IDs: Atomic counter (not UnixNano)
- Stderr draining: Prevent pipe buffer deadlock
- 60s timeout: Handle npx cold starts

**Benchmarks:**
- Token savings: 38-96% reduction
- Latency: 307ms average (5 servers)
- Startup: <100ms config load

### 5. YAGNI (You Aren't Gonna Need It)

**Principle:** Only implement what's needed now

**Guidelines:**
- Don't build features for hypothetical use cases
- Defer optimization until proven necessary
- Avoid abstraction layers without clear need
- Minimal viable implementation over complete solution

**Examples:**
- No remote MCP support yet (not needed)
- No pool eviction (processes stay alive)
- No metrics collection (can add later)

## Architecture Patterns

### Aggregator Pattern

**Purpose:** Single gateway to multiple services

**Implementation:**
```go
// tool-hub-mcp acts as aggregator
type Server struct {
    spawner *spawner.Pool  // Manages child processes
}

// AI interacts with hub, not directly with servers
hub_execute(server="jira", tool="create-issue", ...)
```

**Benefits:**
- Unified interface
- Reduced token consumption
- Centralized management

**Trade-offs:**
- Additional hop (latency)
- Single point of failure (mitigated by simplicity)

### Process Pool Pattern

**Purpose:** Reuse expensive resources

**Implementation:**
```go
type Pool struct {
    maxSize   int
    processes map[string]*Process
}

// Lazy spawn
GetOrSpawn(name, cfg) {
    if exists in pool {
        return cached
    }
    return spawn(name, cfg)
}
```

**Benefits:**
- Faster warm starts (process reuse)
- Lower resource usage
- Bounded concurrency

**Trade-offs:**
- Memory overhead (~5-10MB per process)
- No eviction policy yet

### Lazy Loading Pattern

**Purpose:** Defer expensive operations until needed

**Implementation:**
```go
// Tool definitions not loaded until hub_discover called
func (e *execHubDiscover) Execute(args CallToolResultArgs) string {
    proc, _ := e.spawner.GetOrSpawn(server, cfg)
    tools, _ := e.spawner.GetTools(server, cfg)
    return formatTools(tools)
}
```

**Benefits:**
- Faster startup
- Lower memory footprint
- Only load what's used

**Trade-offs:**
- First call slower (mitigated by pooling)

### Strategy Pattern (Config Sources)

**Purpose:** Pluggable config parsing

**Implementation:**
```go
type Source interface {
    Name() string
    Scan() (*SourceResult, error)
}

sources := []Source{
    &ClaudeCodeSource{},
    &OpenCodeSource{},
    &AntigravitySource{},
    // Easy to add new sources
}
```

**Benefits:**
- Extensible (add new AI clients easily)
- Testable (mock sources)
- Decoupled parsing logic

## API Design Guidelines

### Meta-Tool Naming

**Convention:** `hub_{action}`

**Actions:**
- `list` - Enumerate items
- `discover` - Get detailed info
- `search` - Find by query
- `execute` - Perform action
- `help` - Get guidance

**Rationale:** Clear, consistent, discoverable

### Parameter Naming

**Conventions:**
- `server` - Server name (enum of registered)
- `tool` - Tool name (string)
- `arguments` - Tool parameters (object)
- `query` - Search text (string)

**Example:**
```json
{
  "server": "jira",
  "tool": "create-issue",
  "arguments": {
    "project": "PROJ",
    "summary": "Bug fix"
  }
}
```

### Error Responses

**Format:** JSON-RPC 2.0 error

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32601,
    "message": "Tool not found: unknown-tool"
  }
}
```

**Messages:**
- Descriptive, actionable
- Include context (server name, tool name)
- No stack traces (security)

### Dynamic Enums

**Principle:** Build enums at runtime from config

```go
"server": {
  "type": "string",
  "enum": getServerNames(),  // ["jira", "figma", "playwright"]
  "description": "Currently registered: jira, figma, playwright"
}
```

**Benefits:**
- Always in sync with actual servers
- No stale descriptions
- No code changes needed

## Code Organization Guidelines

### Package Structure

**Principle:** Package by responsibility, not layer

```
internal/
├── cli/         # Command implementations (UI layer)
├── mcp/         # MCP protocol (protocol layer)
├── config/      # Configuration management (data layer)
├── spawner/     # Process management (infrastructure layer)
└── benchmark/   # Performance metrics (analytics layer)
```

**Benefits:**
- Clear separation of concerns
- Easy to locate code
- Testable in isolation

### File Size Guidelines

**Target:** <200 lines per file

**When to Split:**
- File exceeds 200 lines
- Contains multiple distinct responsibilities
- Hard to understand in one sitting

**How to Split:**
- Extract related functions into separate files
- Create helper packages for utilities
- Use composition over inheritance

**Examples:**
- `server.go` (485 lines) → Consider splitting tool handlers
- `add.go` (403 lines) → Could extract JSON parsing logic

### Function Design

**Principles:**
- Single responsibility
- Clear, descriptive names
- Minimal parameters (prefer structs for >3 params)
- Return errors, don't panic

**Example:**
```go
// Good
func (p *Pool) GetTools(name string, cfg *config.ServerConfig) ([]Tool, error)

// Bad (too many params)
func (p *Pool) GetTools(name string, cmd string, args []string, env map[string]string) ([]Tool, error)
```

## Error Handling Design

### Error Wrapping

**Principle:** Always add context

```go
// Good
return fmt.Errorf("failed to spawn server %s: %w", name, err)

// Bad (no context)
return err
```

### Error Categories

1. **User Errors** (clear message, no stack trace)
   - Invalid config format
   - Server not found
   - Invalid parameters

2. **System Errors** (context + wrap)
   - Process spawn failure
   - Timeout
   - JSON parse errors

3. **Protocol Errors** (JSON-RPC format)
   - Invalid request
   - Method not found
   - Invalid params

### Panic Policy

**Principle:** No panics in production code paths

**When to Panic:**
- Truly unrecoverable conditions (config validation at startup)
- Programmer errors (nil pointer in tests)

**When NOT to Panic:**
- User input errors
- Network failures
- Process spawn failures

## Concurrency Design

### Mutex Usage

**Principle:** Protect shared state, minimize lock duration

```go
// Good
func (p *Pool) getOrSpawn(name string) (*Process, error) {
    p.mu.Lock()
    if proc, exists := p.processes[name]; exists {
        p.mu.Unlock()
        return proc, nil
    }
    p.mu.Unlock()

    // Spawn without holding lock
    proc := spawn(name)
    p.mu.Lock()
    p.processes[name] = proc
    p.mu.Unlock()
    return proc, nil
}
```

### Goroutine Safety

**Principles:**
- Always drain stderr (prevent deadlock)
- Use channels for communication
- Timeout long-running operations

**Example:**
```go
// Always drain stderr
stderr, _ := cmd.StderrPipe()
go func() {
    io.Copy(io.Discard, stderr)
}()
```

### Atomic Operations

**Use for:** Simple counters, flags

```go
type Process struct {
    reqID int64  // Atomic counter
}

func (p *Process) nextID() int64 {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.reqID++
    return p.reqID
}
```

## Testing Guidelines

### Unit Tests

**Focus:** Single function logic

**Examples:**
- Name transformation (dash-case → camelCase)
- Config parsing
- Token calculations

**Guidelines:**
- Test edge cases (empty, nil, special characters)
- Use table-driven tests
- Mock external dependencies

### Integration Tests

**Focus:** Component interaction

**Examples:**
- Config import → spawn → tool execution
- CLI command end-to-end

**Guidelines:**
- Use real config files (fixtures)
- Test error paths
- Verify side effects (file writes)

### Missing Tests

**Current Gaps:**
- Spawner lifecycle (hard without real MCPs)
- Source parsing (needs fixtures)
- Process pool edge cases

**Priority:** Medium (functionality verified via manual testing)

## Security Design

### Command Injection Prevention

**Principle:** Never use shell interpolation

```go
// BAD (vulnerable)
cmd := exec.Command("sh", "-c", fmt.Sprintf("%s %s", cfg.Command, args))

// GOOD (safe)
cmd := exec.Command(cfg.Command, cfg.Args...)
```

### Environment Variable Isolation

**Principle:** Isolate per process

```go
cmd.Env = append(os.Environ(), envs...)
// Each process gets own copy
```

### Input Validation

**Principles:**
- Validate before use
- Fail fast (return errors early)
- Sanitize file paths

**Example:**
```go
if cfg.Command == "" {
    return fmt.Errorf("server command cannot be empty")
}
```

## Performance Design

### Optimization Strategy

**Principle:** Measure first, optimize second

**Benchmarks:**
- Token consumption (per configuration)
- Latency (per server)
- Startup time (cold/warm)

**Optimization Targets:**
- Reduce token count (primary)
- Minimize cold start latency (secondary)
- Lower memory usage (tertiary)

### Caching Strategy

**Current:** Metadata caching (optional)

```go
type ServerConfig struct {
    Metadata *ServerMetadata  // Cached tool info
}
```

**Future Considerations:**
- Process pool TTL (evict after idle)
- Tool definition refresh (invalidate cache)
- Response caching (for read-only tools)

### Resource Limits

**Configurable:**
```go
type Settings struct {
    ProcessPoolSize int  // Max 3 concurrent
    TimeoutSeconds  int  // 60s per request
}
```

**Rationale:**
- Prevent runaway resource usage
- Ensure predictable performance
- Allow user tuning

## Extensibility Design

### Adding New Config Sources

**Interface:**
```go
type Source interface {
    Name() string
    Scan() (*SourceResult, error)
}
```

**Steps:**
1. Create new source file (e.g., `cursor.go`)
2. Implement `Source` interface
3. Add to sources list in `setup.go`

**Example:**
```go
type CursorSource struct {}

func (s *CursorSource) Name() string {
    return "cursor"
}

func (s *CursorSource) Scan() (*SourceResult, error) {
    // Parse ~/.cursor/mcp.json
    // Return SourceResult
}
```

### Adding New Meta-Tools

**Steps:**
1. Define tool in `server.go`
2. Create executor function
3. Add to `handleToolsCall` routing
4. Update AI workflow documentation

**Example:**
```go
{
    Name: "hub_schema",
    Description: "Get JSON schema for server...",
    InputSchema: map[string]interface{}{
        "properties": map[string]interface{}{
            "server": map[string]interface{}{
                "enum": getServerNames(),
            },
        },
    },
}

// Executor
func (e *execHubSchema) Execute(args CallToolResultArgs) string {
    // Implementation
}
```

## Documentation Design

### Code Comments

**When to Use:**
- Package documentation (package header)
- Exported functions (what + why)
- Non-obvious decisions (why, not what)
- Warnings (edge cases, security)

**Example:**
```go
/*
Package mcp implements the MCP server that exposes meta-tools.

The server uses stdio transport and exposes 5 meta-tools:
  - hub_list: List all registered MCP servers
  ...
*/
package mcp

// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *config.Config) *Server {
    // ...
}

// CRITICAL: Always drain stderr to prevent pipe buffer deadlock
// (~64KB limit). Some MCPs write to stderr during startup.
go func() {
    io.Copy(io.Discard, stderr)
}()
```

### README Design

**Target Audience:** Users (not developers)

**Sections:**
1. Problem (why this exists)
2. Solution (how it works)
3. Benchmark (proof it works)
4. Installation (how to get it)
5. Quick Start (how to use it)
6. Examples (common use cases)
7. Architecture (high-level diagram)

**Style:**
- Concise (<300 lines)
- User-focused (not internals)
- Actionable (copy-paste commands)

### API Documentation

**Meta-Tools:**
- Purpose (what it does)
- Parameters (with types)
- Examples (real usage)
- Workflow (when to use)

**Example:**
```markdown
### hub_execute

Execute a tool from a specific server.

**Parameters:**
- `server` (string, required): Server name
- `tool` (string, required): Tool name
- `arguments` (object, optional): Tool parameters

**Workflow:**
1. Call hub_list to discover servers
2. Call hub_discover(server) to see tools
3. Call hub_execute with server, tool, args

**Example:**
```json
{
  "server": "jira",
  "tool": "create-issue",
  "arguments": {
    "project": "PROJ",
    "summary": "Bug fix"
  }
}
```

## Anti-Patterns to Avoid

### Don't Hardcode Server Names

**Bad:**
```go
"enum": []string{"jira", "figma", "playwright"}
```

**Good:**
```go
"enum": getServerNames()  // Dynamic from config
```

### Don't Use Shell Commands

**Bad:**
```go
exec.Command("sh", "-c", fmt.Sprintf("%s %s", cmd, args))
```

**Good:**
```go
exec.Command(cmd, args...)
```

### Don't Ignore Stderr

**Bad:**
```go
cmd.Stdout = &stdout
// stderr ignored → deadlock risk
```

**Good:**
```go
stderr, _ := cmd.StderrPipe()
go func() {
    io.Copy(io.Discard, stderr)
}()
```

### Don't Panic in Production

**Bad:**
```go
if cfg.Command == "" {
    panic("command required")  // Crashes hub
}
```

**Good:**
```go
if cfg.Command == "" {
    return fmt.Errorf("server command cannot be empty")
}
```

## Design Review Checklist

Before implementing new features:

- [ ] Aligns with token efficiency goal
- [ ] Follows KISS principle
- [ ] No hardcoding (dynamic/configurable)
- [ ] Performance impact measured
- [ ] Security implications considered
- [ ] Error handling comprehensive
- [ ] Documentation updated
- [ ] Tests added (if applicable)

## References

- **MCP Protocol:** https://modelcontextprotocol.io/
- **Go Best Practices:** https://go.dev/doc/effective_go
- **JSON-RPC 2.0:** https://www.jsonrpc.org/specification
- **Project README:** `/README.md`

---

**Owner:** Development Team
**Review Cycle:** Monthly
**Next Review:** 2026-02-21
