# tool-hub-mcp Spawner Fixes

## Issue: JavaScript MCPs Timeout

### Root Cause
JavaScript-based MCP servers (Playwright, Chrome DevTools) silently fail when the JSON-RPC `id` field exceeds JavaScript's `Number.MAX_SAFE_INTEGER` (2^53-1 = 9007199254740991).

The original implementation used `time.Now().UnixNano()` which returns values like `1768868165588394000` (~1.7×10^18), exceeding this limit and causing precision loss in JavaScript JSON parsers.

### Fix Applied
Changed ID generation from `UnixNano` to an atomic counter in `internal/spawner/pool.go`:

```go
// Process now has reqID field
type Process struct {
    // ...
    reqID  int64  // atomic counter for safe IDs
}

// sendRequest uses counter instead of UnixNano
proc.reqID++
reqID := proc.reqID
req := map[string]interface{}{
    "id": reqID,  // Safe integer, starts at 1
    // ...
}
```

### Additional Fixes
1. **Stderr draining**: Added background goroutine to drain stderr and prevent pipe buffer deadlock
2. **60s timeout**: Allows time for npx package downloads on cold start

## Benchmark Results (2026-01-20)

### Token Benchmark (5 servers, 98 tools)
- Traditional: ~15,150 tokens
- tool-hub-mcp: 461 tokens
- **Savings: 95.0%**

### Speed Benchmark
| MCP Server | Tools | Avg Latency |
|------------|-------|-------------|
| Playwright | 33 | 498ms |
| Chrome DevTools | 26 | 295ms |
| mcp-outline | 30 | 202ms |
| shadcn | 7 | 424ms |
| Figma | 2 | 118ms |
| **Total** | **98** | **307ms avg** |
