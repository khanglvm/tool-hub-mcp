# tool-hub-mcp: Project Roadmap

**Version:** 1.0.0
**Last Updated:** 2026-01-21
**Status:** Active

## Project Vision

**Mission:** Reduce AI context token consumption by aggregating MCP servers through a single gateway with on-demand tool discovery.

**Strategy:** Serverless architecture with lazy process spawning, eliminating the need to load all tool definitions upfront.

**Success Metrics:**
- Token savings: >90% reduction vs traditional approach
- Latency: <500ms average tool discovery
- Adoption: 1,000+ npm downloads/month
- Community: Active contributions and issues

## Release History

### v1.1.0 (Current - January 2026)

**Status:** ✅ Production Ready

**Features:**
- ✅ 2 meta-tools (hub_search with BM25 + bandit, hub_execute with learning)
- ✅ Semantic search using Bleve BM25 algorithm
- ✅ Learning system with ε-greedy multi-armed bandit (UCB1)
- ✅ SQLite storage at ~/.tool-hub-mcp/history.db
- ✅ CLI learning commands (status, export, clear, enable, disable)
- ✅ searchId tracking for linking search → execution
- ✅ Privacy-preserving (SHA256 context hashing)
- ✅ Config import from 6 AI clients (Claude Code, OpenCode, Google Antigravity, Gemini CLI, Cursor, Windsurf)
- ✅ Lazy process spawning with pool management
- ✅ Zero-install npm distribution (5 platforms)
- ✅ CLI commands (setup, add, remove, list, verify, serve, benchmark)
- ✅ Speed benchmarking
- ✅ Automated release pipeline (GitHub Actions)
- ✅ Comprehensive documentation

**Verified Performance:**
- Token savings: 38.48% reduction (18,613 tokens saved with 6 servers)
- Speed: 307ms average latency (5 servers)
- Intelligence: Bandit ranking improves tool discovery over time

**Known Issues:**
- No remote MCP support (OpenCode "remote" type)
- No tool metadata refresh mechanism
- No pool eviction policy
- Duplicate server names use last-write-wins

**Downloads:** N/A (new release)

---

## Roadmap Phases

### Phase 1: Foundation (Complete - v1.0.0)

**Timeline:** December 2025 - January 2026

**Completed Tasks:**
- ✅ Design meta-tools architecture
- ✅ Implement MCP server (stdio transport)
- ✅ Build process pool with lazy spawning
- ✅ Create CLI commands (setup, add, remove, list, verify, serve)
- ✅ Implement config import from Claude Code, OpenCode
- ✅ Add benchmark system (token + speed)
- ✅ Set up zero-install npm distribution
- ✅ Configure GitHub Actions release pipeline
- ✅ Write comprehensive documentation

**Metrics Achieved:**
- 5 meta-tools implemented
- 6 AI client config sources supported
- 5 platforms supported (npm + Go install)
- 38.48% token reduction verified

---

### Phase 2: Stability & Performance (Complete - v1.0.1)

**Timeline:** January 2026

**Completed Tasks:**
- ✅ Fix JavaScript MCP timeout (safe request IDs)
- ✅ Fix stderr pipe deadlock (draining goroutine)
- ✅ Increase timeout to 60s (handle npx cold starts)
- ✅ Add Google Antigravity, Gemini CLI, Cursor, Windsurf sources
- ✅ Remove hardcoded server names from tool descriptions
- ✅ Add comprehensive documentation (7 docs)
- ✅ Verify 3-server benchmark (88.6% savings)

**Metrics Achieved:**
- 0 timeout errors (JavaScript MCPs working)
- 0 deadlock issues
- 95% token reduction (5 servers, 98 tools)
- 307ms average latency

---

### Phase 3: Tool Reduction & Learning (Complete - v1.1.0)

**Timeline:** January 2026

**Completed Tasks:**
- ✅ Reduce meta-tools from 5 → 2 (hub_search, hub_execute)
- ✅ Implement BM25 semantic search using Bleve
- ✅ Build learning system with ε-greedy bandit algorithm
- ✅ Add SQLite storage for usage tracking
- ✅ Implement searchId for linking search → execution
- ✅ Create CLI learning commands (status, export, clear, enable, disable)
- ✅ Add SHA256 context hashing for privacy
- ✅ Implement UCB1 scoring for intelligent ranking
- ✅ Update all documentation for new architecture

**Metrics Achieved:**
- Token reduction: 38-96% (same or better than before)
- Search quality: BM25 + bandit ranking improves over time
- Learning overhead: <10ms per query
- Privacy: All contexts SHA256 hashed before storage

---

### Phase 4: Enhanced Compatibility (In Progress - v1.2.0)

**Timeline:** February - March 2026

**Target Release:** v1.2.0

#### Priority Features

**P0 - Remote MCP Support:**
- [ ] Parse OpenCode "remote" type config
- [ ] Implement remote MCP connection (HTTP/SSE)
- [ ] Add remote server health checks
- [ ] Document remote vs local differences

**P1 - Config Merge Strategy:**
- [ ] Define merge behavior for duplicate server names
- [ ] Implement merge strategies: first-write-wins, last-write-wins, merge-env
- [ ] Add `--merge-strategy` flag to setup command
- [ ] Document merge behavior in docs

**P2 - Additional Config Sources:**
- [ ] Add Roo Code source
- [ ] Add Continue.dev source
- [ ] Add Tabby source
- [ ] Add generic JSON source (custom path)

**P3 - Metadata Refresh:**
- [ ] Implement `tool-hub-mcp refresh` command
- [ ] Add `--force-refresh` flag to hub_discover
- [ ] Cache metadata with TTL (24h default)
- [ ] Auto-refresh on version change detection

**Success Criteria:**
- Remote MCPs working (OpenCode remote type)
- Duplicate server names handled gracefully
- 10+ config sources supported
- Metadata cache auto-refresh

**Stretch Goals:**
- [ ] Support for SSE (Server-Sent Events) transport
- [ ] Websocket transport support
- [ ] MCP proxy mode (forward all requests)

---

### Phase 5: Performance & Scalability (Planned - v1.3.0)

**Timeline:** April - May 2026

**Target Release:** v1.3.0

#### Priority Features

**P0 - Pool Eviction Policy:**
- [ ] Implement LRU eviction (least recently used)
- [ ] Add TTL-based eviction (configurable, default 1h)
- [ ] Add `--pool-ttl` setting
- [ ] Monitor pool statistics (size, evictions)

**P1 - Parallel Tool Discovery:**
- [ ] Fetch tool lists concurrently (not sequentially)
- [ ] Implement `hub_discover --parallel` flag
- [ ] Add timeout per concurrent request
- [ ] Aggregate results with error handling

**P2 - Connection Keep-Alive:**
- [ ] Persist processes beyond current session
- [ ] Implement daemon mode (`tool-hub-mcp daemon`)
- [ ] Add Unix socket communication
- [ ] Add health check endpoint

**P3 - Performance Optimizations:**
- [ ] Reduce process spawn time (reuse processes)
- [ ] Implement connection pooling per server
- [ ] Cache tool definitions in memory (not just file)
- [ ] Optimize JSON parsing (streaming for large responses)

**Success Criteria:**
- <50ms warm start latency (process reuse)
- <100ms parallel tool discovery (5 servers)
- 0-10MB memory footprint (per process)
- 1,000+ tool calls without memory leaks

**Stretch Goals:**
- [ ] Native code compilation (faster startup)
- [ ] Shared process pool (multiple hub instances)
- [ ] Distributed mode (load balancing)

---

### Phase 6: Observability & Diagnostics (Planned - v1.4.0)

**Timeline:** June - July 2026

**Target Release:** v1.4.0

#### Priority Features

**P0 - Structured Logging:**
- [ ] Implement structured logging (JSON format)
- [ ] Add log levels (debug, info, warn, error)
- [ ] Support log output to file
- [ ] Add `--log-level` and `--log-file` flags

**P1 - Metrics Collection:**
- [ ] Track spawn count per server
- [ ] Track error rate per server
- [ ] Track latency distribution (p50, p95, p99)
- [ ] Export metrics to Prometheus/Promtail

**P2 - Health Checks:**
- [ ] Implement `tool-hub-mcp health` command
- [ ] Check server connectivity
- [ ] Check tool availability
- [ ] Check process pool status

**P3 - Debugging Tools:**
- [ ] Add `tool-hub-mcp debug` command
- [ ] Trace request lifecycle
- [ ] Capture stderr from child processes (optional)
- [ ] Generate diagnostic bundle

**Success Criteria:**
- All operations logged with context
- Metrics dashboard available
- Health checks pass/fail clear
- Debug session <5min to setup

**Stretch Goals:**
- [ ] OpenTelemetry integration
- [ ] Grafana dashboard templates
- [ ] Alerting on error thresholds

---

### Phase 6: Enterprise Features (Planned - v2.0.0)

**Timeline:** August - December 2026

**Target Release:** v2.0.0 (Major Version)

#### Breaking Changes

**BC1 - Remote Configuration:**
- [ ] Support remote config sources (HTTP, S3, Git)
- [ ] Implement config polling (auto-reload)
- [ ] Add config validation schema
- [ ] Support environment-specific configs (dev, staging, prod)

**BC2 - Authentication:**
- [ ] Add authentication to MCP gateway
- [ ] Support API key authentication
- [ ] Add OAuth2 integration
- [ ] Implement per-server auth mapping

**BC3 - Authorization:**
- [ ] Add role-based access control (RBAC)
- [ ] Implement per-tool permissions
- [ ] Add audit logging
- [ ] Support team/organization access

**New Features:**

**P0 - Multi-Tenancy:**
- [ ] Support multiple isolated configurations
- [ ] Implement team workspaces
- [ ] Add per-tenant rate limiting
- [ ] Tenant-specific metrics

**P1 - High Availability:**
- [ ] Support multiple hub instances (load balancing)
- [ ] Implement leader election
- [ ] Add graceful shutdown
- [ ] Support rolling upgrades

**P2 - Advanced Networking:**
- [ ] Support HTTP/HTTPS transport (not just stdio)
- [ ] Add gRPC transport support
- [ ] Implement request batching
- [ ] Add request prioritization

**P3 - Developer Tools:**
- [ ] MCP testing framework
- [ ] Mock MCP server for testing
- [ ] Integration test templates
- [ ] Performance profiling tools

**Success Criteria:**
- 10,000+ concurrent users
- 99.9% uptime
- <100ms p95 latency
- Enterprise security compliance

**Stretch Goals:**
- [ ] Cloud-native deployment (Kubernetes)
- [ ] Plugin system (custom meta-tools)
- [ ] Web UI for management
- [ ] Multi-region deployment

---

## Future Considerations

### Potential Enhancements (Beyond v2.0.0)

**AI Integration:**
- [ ] AI-powered server recommendation
- [ ] Natural language server search
- [ ] Automatic tool chaining
- [ ] Intent-based tool selection

**Advanced Performance:**
- [ ] Binary protocol (MessagePack, protobuf)
- [ ] Compression (gzip, brotli)
- [ ] Edge deployment (CDN)
- [ ] Caching layer (Redis)

**Ecosystem:**
- [ ] Plugin marketplace
- [ ] Community-contributed sources
- [ ] Server certification program
- [ ] Best practices guide

**Platform Expansion:**
- [ ] Python distribution (uvx)
- [ ] Rust distribution (cargo)
- [ ] Docker images
- [ ] Homebrew formula

### Deprecations

**No deprecations planned yet.**

**Future candidates:**
- Old config format versions (after v2.0.0)
- Legacy CLI flags (after 6-month notice)
- Python 2.x (if Python distribution added)

---

## Resource Allocation

### Development Team

**Current:** 1 maintainer (part-time)

**Target (v1.3.0):** 2-3 contributors
- 1 maintainer (core architecture)
- 1 contributor (config sources, docs)
- 1 contributor (testing, benchmarks)

**Target (v2.0.0):** 3-5 contributors
- 1 maintainer (architecture, direction)
- 2 contributors (features, bugs)
- 1 contributor (docs, community)
- 1 contributor (testing, CI/CD)

### Milestone Schedule

| Version | Phase | Target Date | Status |
|---------|-------|-------------|--------|
| v1.0.0 | Foundation | Dec 2025 - Jan 2026 | ✅ Complete |
| v1.0.1 | Stability | Jan 2026 | ✅ Complete |
| v1.1.0 | Tool Reduction & Learning | Jan 2026 | ✅ Complete |
| v1.2.0 | Compatibility | Feb - Mar 2026 | 🚧 In Progress |
| v1.3.0 | Performance | Apr - May 2026 | 📋 Scheduled |
| v1.4.0 | Observability | Jun - Jul 2026 | 📋 Scheduled |
| v2.0.0 | Enterprise | Aug - Dec 2026 | 📋 Scheduled |

---

## Risk Assessment

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| MCP protocol breaking changes | High | Low | Pin protocol version, flexible parsing |
| Child process memory leaks | Medium | Medium | Pool monitoring, eviction policy |
| Config format complexity | Low | High | Auto-detection, validation |
| npm platform support breakage | High | Low | Test on all platforms, fallback to Go install |

### Operational Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Maintainer burnout | High | Medium | Community contributions, clear docs |
| Security vulnerabilities | High | Low | Code reviews, dependency scanning |
| npm account compromise | High | Low | 2FA, limited access tokens |
| GitHub service outage | Low | Low | Manual publish script as backup |

### Adoption Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Low adoption rate | Medium | Medium | Marketing, case studies, examples |
| Competing solutions | High | Low | Focus on simplicity, performance |
| AI clients add native aggregators | High | Low | Differentiate with flexibility, docs |

---

## Success Metrics

### Adoption Metrics

**v1.1.0 Targets:**
- 500 npm downloads/month
- 100 GitHub stars
- 10 community issues/PRs

**v1.2.0 Targets:**
- 1,000 npm downloads/month
- 200 GitHub stars
- 50 community issues/PRs

**v2.0.0 Targets:**
- 5,000 npm downloads/month
- 500 GitHub stars
- 200 community issues/PRs
- 5 enterprise customers

### Performance Metrics

**Current (v1.0.1):**
- Token savings: 38-96%
- Latency: 307ms average
- Memory: ~15MB (pool of 3)

**v1.2.0 Targets:**
- Token savings: >95% (consistent)
- Latency: <100ms (warm), <500ms (cold)
- Memory: <10MB (pool of 3)

**v2.0.0 Targets:**
- Token savings: >95%
- Latency: <50ms (warm), <200ms (cold)
- Memory: <5MB (pool of 3)

### Quality Metrics

**Current:**
- Test coverage: ~70% (critical paths)
- Documentation: 7 comprehensive docs
- Benchmarking: Token + speed tests

**v1.3.0 Targets:**
- Test coverage: >80%
- Documentation: 10+ docs (API reference, tutorials)
- Benchmarking: Automated CI/CD benchmarks

**v2.0.0 Targets:**
- Test coverage: >90%
- Documentation: Complete (API, guides, examples)
- Benchmarking: Performance regression tests

---

## Feedback & Contributions

### How to Contribute

**Report Issues:**
- GitHub Issues: https://github.com/khanglvm/tool-hub-mcp/issues
- Include: Version, platform, steps to reproduce, expected behavior

**Submit PRs:**
- Fork repository
- Create feature branch
- Add tests (if applicable)
- Update docs
- Submit PR with clear description

**Feature Requests:**
- GitHub Issues with "enhancement" label
- Describe use case, proposed solution, alternatives

### Community Channels

**Discussions:**
- GitHub Discussions: https://github.com/khanglvm/tool-hub-mcp/discussions

**Documentation:**
- `/docs/` - Comprehensive guides
- `/README.md` - Quick start
- `/docs/facts/` - Research and analysis

---

## References

- **MCP Protocol:** https://modelcontextprotocol.io/
- **GitHub Repository:** https://github.com/khanglvm/tool-hub-mcp
- **npm Package:** https://www.npmjs.com/package/@khanglvm/tool-hub-mcp
- **Project Overview:** `/docs/project-overview-pdr.md`
- **System Architecture:** `/docs/system-architecture.md`

---

**Owner:** Development Team
**Review Cycle:** Monthly
**Next Review:** 2026-02-21
