# Test Workflow Guide

**Version:** 1.0.0
**Last Updated:** 2026-01-22
**Status:** Active

## Overview

Comprehensive testing strategy for tool-hub-mcp with mandatory pre-deployment testing. Ensures code quality through automated testing at multiple stages: local development, pre-commit, pre-push, and CI/CD.

## Test Types

### Unit Tests

**Location:** `*_test.go` files alongside source code
**Coverage Target:** 80% overall minimum
**Purpose:** Test individual functions and components in isolation

**Run Commands:**
```bash
go test ./...                    # All tests
go test ./internal/benchmark    # Specific package
go test -v ./...                # Verbose output
```

**Current Coverage:**
- `internal/benchmark`: 97.8%
- `internal/learning`: 89.3%
- `internal/config`: 80.7%
- `internal/search`: 69.1%
- `internal/storage`: 48.7%
- `internal/mcp`: 43.5%
- `internal/cli`: 21.1%
- `internal/spawner`: 13.2%

### Integration Tests

**Location:** `internal/mcp/server_integration_test.go`
**Coverage Target:** 90% for MCP handlers
**Purpose:** Test component interactions and MCP protocol compliance

**Run Commands:**
```bash
go test ./internal/mcp -v       # MCP integration tests
go test -run Integration ./...  # All integration tests
```

**Test Scenarios:**
- MCP server initialization and handshake
- hub_search semantic search with BM25 ranking
- hub_execute tool execution with learning tracking
- Concurrent access and process pool management
- Error handling and timeout behavior

### End-to-End Tests

**Location:** `test/e2e/workflow_test.go`
**Status:** Incomplete (compilation errors present)
**Purpose:** Full workflow testing from AI client perspective

**Future:** Will test complete request-response cycles simulating real AI client interactions

## Development Workflow

### Standard Development Flow

```
1. Write tests first (TDD encouraged)
   ↓
2. Implement feature
   ↓
3. Run local tests: make test
   ↓
4. Check coverage: make test-coverage
   ↓
5. Commit changes (pre-commit hook runs automatically)
   ↓
6. Push to remote (pre-push hook runs automatically)
   ↓
7. CI validates (GitHub Actions)
   ↓
8. Merge when all checks pass
```

## Writing Tests

### Table-Driven Test Pattern

**Standard Go pattern for comprehensive test coverage:**

```go
func TestToCamelCase(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"dash-case", "dash-case", "dashCase"},
        {"snake_case", "snake_case", "snakeCase"},
        {"PascalCase", "PascalCase", "pascalCase"},
        {"already camelCase", "camelCase", "camelCase"},
        {"empty string", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ToCamelCase(tt.input)
            if got != tt.expected {
                t.Errorf("ToCamelCase(%q) = %q, want %q",
                    tt.input, got, tt.expected)
            }
        })
    }
}
```

### Subprocess Mocking Pattern

**For testing code that spawns external processes:**

See `internal/spawner/pool_test.go` for complete TestHelperProcess implementation.

```go
func TestSpawner(t *testing.T) {
    // Override command to use test helper
    cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
    cmd.Env = []string{"GO_TEST_PROCESS=1", "HELPER_MODE=success"}

    // Test spawning logic
}

func TestHelperProcess(t *testing.T) {
    if os.Getenv("GO_TEST_PROCESS") != "1" {
        return
    }
    // Mock process behavior based on env vars
}
```

### Integration Test Example

```go
func TestMCPServerIntegration(t *testing.T) {
    // Setup
    cfg := &config.Config{
        Servers: map[string]*config.ServerConfig{
            "testServer": {
                Command: "npx",
                Args:    []string{"-y", "@test/server"},
            },
        },
    }

    server := mcp.NewServer(cfg)

    // Test hub_search
    result := server.execHubSearch("test query")

    // Assertions
    if len(result.Results) == 0 {
        t.Error("Expected search results")
    }
}
```

## Git Hooks

### Pre-commit Hook

**Purpose:** Fast feedback on changed code
**Location:** `.git/hooks/pre-commit` (auto-generated)
**Script:** `scripts/test-pre-commit.sh`
**Execution Time:** <10s (target)

**What it does:**
1. Detects staged Go files
2. Identifies affected packages
3. Runs `go test -short` on changed packages only
4. Fails commit if tests fail

**Install:**
```bash
make setup-hooks
```

**Bypass (emergencies only):**
```bash
git commit --no-verify
```

### Pre-push Hook

**Purpose:** Full validation before remote push
**Location:** `.git/hooks/pre-push` (auto-generated)
**Script:** `scripts/test-pre-push.sh`
**Execution Time:** <60s (target)

**What it does:**
1. Runs full test suite with race detector
2. Generates coverage report
3. Checks 80% coverage threshold
4. Fails push if coverage below threshold

**Manual run:**
```bash
make test-coverage
```

## CI/CD Pipeline

### GitHub Actions Workflow

**File:** `.github/workflows/test.yml`
**Trigger:** Push to main/develop, PRs to main

**Matrix Testing:**
- Go versions: 1.21, 1.22, 1.23.x
- OS: Ubuntu Latest (Linux)
- Parallel execution for speed

**Steps:**
1. Checkout code
2. Setup Go (with caching)
3. Install dependencies
4. Run tests with race detector (`-race`)
5. Generate coverage report (`-coverprofile`)
6. Check 80% threshold (fails if below)
7. Upload to Codecov (Go 1.23.x only)

**Coverage Badge:** Available via Codecov integration

## Coverage Analysis

### Generate HTML Report

```bash
# Generate coverage data
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
```

### View Function-Level Coverage

```bash
go tool cover -func=coverage.out

# Show only uncovered or partially covered functions
go tool cover -func=coverage.out | grep -v "100.0%"
```

### Package-Level Coverage

```bash
go test -cover ./...
```

**Output:**
```
ok  	github.com/khanglvm/tool-hub-mcp/internal/benchmark	0.339s	coverage: 97.8% of statements
ok  	github.com/khanglvm/tool-hub-mcp/internal/cli	4.287s	coverage: 21.1% of statements
...
```

## Makefile Targets

### Available Commands

```bash
make test              # Run all tests
make test-race         # Run with race detector
make test-fast         # Pre-commit fast tests
make test-coverage     # Pre-push full suite + coverage check
make setup-hooks       # Install git hooks
```

### Implementation Details

**test:** Standard test run
```bash
go test -v ./...
```

**test-race:** Detect race conditions
```bash
go test -race -v ./...
```

**test-fast:** Changed packages only
```bash
./scripts/test-pre-commit.sh
```

**test-coverage:** Full validation
```bash
./scripts/test-pre-push.sh
```

## New Feature Requirements

**ALL new features MUST include:**

1. **Unit Tests** (80%+ coverage of new code)
   - Happy path scenarios
   - Error cases
   - Edge cases

2. **Integration Tests** (if API/CLI changes)
   - Component interaction
   - Protocol compliance
   - End-to-end flows

3. **Error Scenario Coverage**
   - Invalid inputs
   - Network failures
   - Timeout handling
   - Resource exhaustion

4. **Documentation**
   - Test scenarios explained
   - Known limitations
   - Future test improvements

## Troubleshooting

### Tests Fail Locally

**Check specific package:**
```bash
go test -v ./internal/package
```

**Run with verbose output:**
```bash
go test -v -run TestName ./...
```

**Debug with race detector:**
```bash
go test -race ./internal/spawner
```

### Coverage Below 80%

**Identify low-coverage packages:**
```bash
go test -cover ./... | grep -v "100.0%"
```

**Analyze specific package:**
```bash
go test -coverprofile=coverage.out ./internal/cli
go tool cover -func=coverage.out
```

**Generate HTML for visual analysis:**
```bash
go tool cover -html=coverage.out
```

### Pre-commit Hook Fails

**See detailed output:**
```bash
./scripts/test-pre-commit.sh
```

**Check which tests failed:**
```bash
go test -short ./path/to/failing/package
```

**Emergency bypass (use sparingly):**
```bash
git commit --no-verify
```

### Pre-push Hook Fails

**Run manually to debug:**
```bash
./scripts/test-pre-push.sh
```

**Check coverage threshold:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1
```

### CI Tests Pass Locally But Fail in GitHub Actions

**Common causes:**
- Race conditions (detected by `-race` flag)
- Platform-specific behavior (Linux vs macOS)
- Timing issues in tests
- Missing environment setup

**Debug strategy:**
```bash
# Run exactly like CI does
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Check race detector output
go test -race ./...
```

## Best Practices

### Test Organization

- **One test file per source file:** `config.go` → `config_test.go`
- **Package-level tests:** Use `package_test` for black-box testing
- **Helper functions:** Extract common setup to `testing.go` or `helpers_test.go`

### Test Naming

- **Test functions:** `TestFunctionName`
- **Subtests:** Descriptive names in `t.Run()`
- **Benchmark tests:** `BenchmarkFunctionName`

### Assertions

- **Use `t.Errorf`** for failures with context
- **Provide actual vs expected** in error messages
- **Use `t.Fatal`** to stop test immediately on critical failures

### Test Data

- **Temp directories:** Use `t.TempDir()` for file operations
- **Isolated state:** Each test should be independent
- **Cleanup:** Use `t.Cleanup()` or defer for resource cleanup

## Performance Considerations

### Test Speed

**Current benchmarks:**
- Pre-commit hook: ~3-8s (changed packages only)
- Pre-push hook: ~10-15s (full suite + coverage)
- CI/CD pipeline: ~2-3 min (matrix testing)

**Optimization tips:**
- Use `testing.Short()` for long-running tests
- Run changed packages only in pre-commit
- Parallelize independent tests with `t.Parallel()`

### Coverage vs Speed Trade-off

- **80% coverage target** balances thoroughness and development speed
- **Focus on critical paths** (spawner, MCP handlers, CLI commands)
- **Accept lower coverage** for simple getters/setters
- **Prioritize integration tests** for complex interactions

## References

- **Go Testing Package:** https://pkg.go.dev/testing
- **Table-Driven Tests:** https://go.dev/wiki/TableDrivenTests
- **Subtests:** https://go.dev/blog/subtests
- **Code Coverage:** https://go.dev/blog/cover
- **Race Detector:** https://go.dev/doc/articles/race_detector
- **Research:** [Go Testing Strategies](../plans/260122-1319-test-workflow-implementation/research/researcher-01-go-testing-strategies.md)
- **Code Standards:** [code-standards.md](./code-standards.md)

---

**Owner:** Development Team
**Review Cycle:** Quarterly
**Next Review:** 2026-04-22
