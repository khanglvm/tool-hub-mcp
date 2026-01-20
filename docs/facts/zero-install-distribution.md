# Zero-Install Distribution Research

## Summary
Research on zero-install distribution mechanisms for Go-based CLI tools, focused on enabling `npx`-like UX without requiring Go runtime.

## Zero-Install Mechanisms Evaluated

| Mechanism | Ecosystem | Requirements | Best For |
|-----------|-----------|--------------|----------|
| **npx/bunx** | Node.js | Node.js installed | JS developers (widest reach) |
| **uvx** | Python | uv installed | Python developers |
| **go run** | Go | Go installed | Go developers only |
| **deno dx** | Deno | Deno 2.6+ | Deno ecosystem |

## Recommended: npm with `optionalDependencies` (esbuild/Biome Pattern)

**Used by**: esbuild, Biome, Turbo, SWC, Parcel, Rolldown

**Architecture**:
```
@khanglvm/tool-hub-mcp (main package)
├── cli.js (thin wrapper → spawns binary)
├── postinstall.js (fallback binary download)
└── optionalDependencies:
    ├── @khanglvm/tool-hub-mcp-darwin-arm64
    ├── @khanglvm/tool-hub-mcp-darwin-x64
    ├── @khanglvm/tool-hub-mcp-linux-x64
    ├── @khanglvm/tool-hub-mcp-linux-arm64
    └── @khanglvm/tool-hub-mcp-win32-x64
```

**Usage**:
```bash
npx @khanglvm/tool-hub-mcp setup
npx @khanglvm/tool-hub-mcp serve
```

## Key Implementation Notes

1. **npm optionalDependencies**: Package managers only install platform-matching packages, reducing download size
2. **postinstall fallback**: Downloads from GitHub Releases if optionalDeps disabled (some CI environments)
3. **cli.js wrapper**: Thin JS script that locates and spawns the Go binary with stdio passthrough
4. **6 packages total**: 1 main + 5 platform-specific sub-packages

## References
- esbuild source: https://github.com/evanw/esbuild
- Biome source: https://github.com/biomejs/biome
- go-npm tool: https://github.com/nicholasklick/go-npm

## Implementation Status

**Completed (Jan 2026):**
- Main npm package with `optionalDependencies`
- 5 platform-specific sub-packages (darwin-arm64/x64, linux-x64/arm64, win32-x64)
- CLI wrapper (`cli.js`) with stdio passthrough
- Postinstall fallback for GitHub Releases download
- GitHub Actions workflow for automated npm publishing

**Next Steps:**
1. Create npm org: `npm org create khanglvm`
2. Add `NPM_TOKEN` GitHub Secret
3. Tag first release: `git tag v1.0.0 && git push --tags`
