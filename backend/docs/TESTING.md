# Testing Guide

How to write, run, and debug tests in the WaaS codebase.

> See also: [Testing Tools](testing_tools.md) for webhook testing and debugging utilities.

## Quick Commands

```bash
make test               # Core package tests with coverage
make test-all           # All tests (includes enterprise packages)
make test-coverage      # Per-package coverage breakdown
make test-status        # Show which packages have test files
make test-integration   # Integration tests in Docker (isolated DB)

# Run a specific package
go test -v ./pkg/repository/...

# Run a specific test
go test -v -run TestValidateTenant ./pkg/models/...

# Run with race detection
go test -race ./pkg/queue/...
```

## Test Conventions

### File naming
- Unit tests: `*_test.go` in the same package
- Unit tests (external): `*_unit_test.go` for black-box testing
- Integration tests: `*_integration_test.go` (need live DB/Redis)

### Test structure
We use `testify/assert` and table-driven subtests:

```go
package mypackage

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMyFunction(t *testing.T) {
    t.Parallel()

    t.Run("success case", func(t *testing.T) {
        result, err := MyFunction("valid-input")
        assert.NoError(t, err)
        assert.Equal(t, "expected", result)
    })

    t.Run("error case", func(t *testing.T) {
        _, err := MyFunction("")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "input required")
    })
}
```

### Table-driven tests (preferred for multiple cases)

```go
func TestValidateURL(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid https", "https://example.com/webhook", false},
        {"valid with port", "https://example.com:8443/hook", false},
        {"http rejected", "http://example.com/webhook", true},
        {"empty string", "", true},
        {"no scheme", "example.com/webhook", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateURL(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Test Layers

### 1. Unit tests (no external dependencies)

Test pure functions, models, validation logic. These run fast and need no Docker.

**Example:** `pkg/models/validation_test.go` — tests model validation rules.

```go
// Good: tests pure logic, no DB needed
func TestValidateTenant(t *testing.T) {
    tenant := &Tenant{Name: "test", APIKeyHash: "hash", ...}
    err := ValidateTenant(tenant)
    assert.NoError(t, err)
}
```

### 2. Repository/integration tests (need PostgreSQL)

Test database queries against a real database. These skip if `TEST_DATABASE_URL` is not set.

**Example:** `pkg/repository/test_helpers_test.go` — provides `setupTestDB()` helper.

```go
func setupTestDB(t *testing.T) *database.DB {
    t.Helper()
    dbURL := os.Getenv("TEST_DATABASE_URL")
    if dbURL == "" {
        t.Skip("TEST_DATABASE_URL not set")
    }
    db, err := database.NewTestConnection()
    if err != nil {
        t.Fatalf("failed to connect: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}
```

To run integration tests:
```bash
# Start test database
docker-compose up -d postgres

# Set test DB URL and run
TEST_DATABASE_URL="postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable" \
  go test -v ./pkg/repository/...
```

### 3. Handler tests (HTTP layer)

Test API handlers using `httptest` and Gin's test mode:

```go
func TestHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)

    c.Set("tenant_id", "test-tenant-id")
    c.Request = httptest.NewRequest("GET", "/api/v1/resource", nil)

    handler.GetResource(c)
    assert.Equal(t, http.StatusOK, w.Code)
}
```

**Example:** `internal/api/handlers/tenant_handler_test.go`

### 4. End-to-end tests

Full flow tests using real HTTP calls against a running server. Found in `test/e2e/`.

```bash
# Start the full stack first
make run-all

# Then run e2e tests
go test -v ./test/e2e/...
```

## Mocking

Use interfaces for dependency injection. All service constructors accept an interface:

```go
type Repository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*Model, error)
    Create(ctx context.Context, m *Model) error
}

// In tests, create a mock:
type mockRepo struct{}
func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*Model, error) {
    return &Model{ID: id, Name: "test"}, nil
}
func (m *mockRepo) Create(ctx context.Context, model *Model) error {
    return nil
}

func TestService(t *testing.T) {
    svc := NewService(&mockRepo{})
    result, err := svc.GetByID(context.Background(), uuid.New())
    assert.NoError(t, err)
    assert.Equal(t, "test", result.Name)
}
```

## CI

Tests run automatically on every push via `.github/workflows/ci.yml`. The CI pipeline:

1. **Code Quality**: checks `gofmt` formatting, runs `go vet ./...`, and runs `golangci-lint`
2. **Environment Validation**: verifies `.env.example` covers all `os.Getenv()` calls in source
3. **Unit Tests**: runs tests with race detection and enforces **70% minimum coverage** threshold
4. **Build**: builds all binaries (`make build`) and all packages (`go build ./...`)
5. **License Check**: validates dependency licenses via `go-licenses`
6. **Integration Tests**: runs integration tests against live PostgreSQL and Redis services (requires unit tests and build to pass)

To replicate CI locally: `gofmt -s -l . && go vet ./... && golangci-lint run && make test`

## Coverage Targets

The project enforces a **70% minimum coverage** threshold in CI (`make ci-local`). Per-package coverage can be viewed with:

```bash
make test-coverage      # Per-package breakdown with total
```

**Recommended targets by package type:**

| Package type | Minimum | Aspirational |
|---|---|---|
| Core (`pkg/models`, `pkg/auth`, `pkg/repository`) | 50% | 80%+ |
| Internal services (`internal/api`, `internal/delivery`) | 40% | 70%+ |
| Enterprise packages (`pkg/billing`, `pkg/signatures`) | 30% | 60%+ |

Generate an HTML report for visual inspection:
```bash
go test -coverprofile=coverage.out ./pkg/... ./internal/...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

## Benchmarking

Use Go's built-in benchmark support to measure performance-critical paths:

```bash
# Run all benchmarks
go test -bench=. -benchmem ./pkg/...

# Run benchmarks in a specific package
go test -bench=. -benchmem ./pkg/signatures/...

# Run a specific benchmark
go test -bench=BenchmarkSign -benchmem ./pkg/signatures/...

# Compare before/after with benchstat
go install golang.org/x/perf/cmd/benchstat@latest
go test -bench=. -count=10 ./pkg/signatures/... > old.txt
# ... make changes ...
go test -bench=. -count=10 ./pkg/signatures/... > new.txt
benchstat old.txt new.txt
```

Write benchmarks alongside tests:
```go
func BenchmarkSign(b *testing.B) {
    svc := setupSignatureService(b)
    req := &SignatureRequest{Payload: []byte("test")}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        svc.Sign(context.Background(), "tenant-1", req)
    }
}
```

## Profiling with pprof

For deeper performance analysis, use Go's built-in pprof:

```bash
# CPU profile
go test -cpuprofile=cpu.out -bench=. ./pkg/signatures/...
go tool pprof cpu.out

# Memory profile
go test -memprofile=mem.out -bench=. ./pkg/signatures/...
go tool pprof mem.out

# Common pprof commands (inside interactive prompt):
#   top 20          — top 20 functions by CPU/memory
#   list FuncName   — annotated source for a function
#   web             — open a call graph in browser (requires graphviz)
```

Install graphviz for visual call graphs:
```bash
# macOS
brew install graphviz

# Linux
apt install graphviz
```

## Test Environment Variables

The following environment variables control which test suites run:

| Variable | Default | Description |
|----------|---------|-------------|
| `RUN_ALL_TESTS` | unset | Set to `true` to run all test suites, including chaos and load tests |
| `RUN_CHAOS_TESTS` | unset | Set to `true` to enable chaos engineering tests (fault injection, failure scenarios) |
| `RUN_LOAD_TESTS` | unset | Set to `true` to enable load/performance tests (may take several minutes) |

These are checked by `Makefile.test` targets. Example usage:

```bash
RUN_CHAOS_TESTS=true make -f Makefile.test test-all
RUN_ALL_TESTS=true make -f Makefile.test test-all
```

## Troubleshooting

- **Tests skip with "TEST_DATABASE_URL not set"** — These are integration tests. Start Docker: `docker-compose up -d`
- **"connection refused"** — PostgreSQL/Redis not running. Run `docker-compose up -d`
- **Slow tests** — Use `-short` flag to skip slow tests: `go test -short ./...`
- **Race conditions** — Run with `-race`: `go test -race ./pkg/queue/...`
