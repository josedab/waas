# Tests

Test suites for the WaaS platform, organized by scope and purpose.

## Test Suites

| Suite | Directory | Make Target | Prerequisites | When to Run |
|-------|-----------|-------------|---------------|-------------|
| **Integration** | `integration/` | `make -f Makefile.test test-integration` | Docker (PostgreSQL + Redis) | After changes to database queries or service wiring |
| **End-to-End** | `e2e/` | `make -f Makefile.test test-e2e` | Docker (full stack running) | Before merging PRs; validates complete user workflows |
| **Performance** | `performance/` | `make -f Makefile.test test-performance` | Docker (full stack running) | Before releases; checks for latency/throughput regressions |
| **Chaos** | `chaos/` | `make -f Makefile.test test-chaos` | Docker (full stack running) | Periodically; validates resilience under failure conditions |
| **Runner** | `runner/` | — | — | Shared test runner utilities used by other suites |

## Quick Start

```bash
# Run unit tests (fastest, no Docker required)
make test

# Run integration tests
make -f Makefile.test test-integration

# Run all test suites
make -f Makefile.test test-all

# See all available test commands
make -f Makefile.test help
```

## Directory Layout

- `chaos/` — Chaos engineering tests (fault injection, network partitions)
- `e2e/` — End-to-end tests covering full API request/response cycles
- `integration/` — Integration tests against real PostgreSQL and Redis
- `performance/` — Load and latency benchmarks
- `runner/` — Shared test runner and helper utilities
- `scripts/` — Test-specific helper scripts
