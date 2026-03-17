.PHONY: build run-api run-delivery run-analytics test test-quick test-all clean \
       docker-up docker-down docker-build \
       migrate-up migrate-down migrate-core migrate-status migrate-reset migrate-rollback dev-setup dev-setup-full ensure-env deps \
       docs docs-generate docs-serve docs-test open-docs \
       lint lint-fast lint-handlers fmt fmt-fix fix vet profile help validate-setup ci-local \
       smoke-test run-all run-dashboard logs \
       install-hooks uninstall-hooks \
       test-coverage test-status test-stubs test-single new-pkg audit-deps validate-env \
       dev dev-all dev-logs check seed quickstart stop _ensure-air _ensure-golangci-lint _ensure-entr _wait-for-services doctor \
       test-failed

# Configuration — DATABASE_URL must be set in the environment or .env file.
# Run "make ensure-env" to create .env from template.
DATABASE_URL ?=

# Default target
.DEFAULT_GOAL := help

help: ## Show available targets
	@echo "WaaS Platform - Development Commands"
	@echo "====================================="
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Additional Makefiles:"
	@echo "  make -f Makefile.test help    Testing commands"
	@echo "  make -f Makefile.prod help    Production operations"

# Build
build: ## Build all service binaries
	go build -o bin/api-service ./cmd/api-service
	go build -o bin/delivery-engine ./cmd/delivery-engine
	go build -o bin/analytics-service ./cmd/analytics-service

build-check: ## Compile-check all packages without producing binaries
	go build ./...

# Run services locally
run-api: ## Run API service (port 8080)
	go run ./cmd/api-service

run-delivery: ## Run delivery engine
	go run ./cmd/delivery-engine

run-analytics: ## Run analytics service (port 8082)
	go run ./cmd/analytics-service

run-all: ## Run all services in parallel (Ctrl+C to stop)
	@echo "Starting all services..."
	@trap 'kill 0' INT TERM; \
		go run ./cmd/api-service & \
		go run ./cmd/delivery-engine & \
		go run ./cmd/analytics-service & \
		wait

dev-logs: ## Run all services with colored, prefixed log output
	@echo "Starting all services with colored prefixed output..."
	@echo "  \033[36m[api]\033[0m = API Service    \033[33m[delivery]\033[0m = Delivery Engine    \033[35m[analytics]\033[0m = Analytics Service"
	@echo ""
	@trap 'kill 0' INT TERM; \
		go run ./cmd/api-service 2>&1       | sed 's/^/\x1b[36m[api]       \x1b[0m /' & \
		go run ./cmd/delivery-engine 2>&1   | sed 's/^/\x1b[33m[delivery]  \x1b[0m /' & \
		go run ./cmd/analytics-service 2>&1 | sed 's/^/\x1b[35m[analytics] \x1b[0m /' & \
		wait

_ensure-golangci-lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint is not installed. It runs static analysis on Go code."; \
		printf "Install via 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'? [y/N] "; \
		read ans; \
		case "$$ans" in [yY]*) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest ;; *) echo "Aborted."; exit 1 ;; esac; \
	}

AIR_VERSION ?= v1.64.5

_ensure-air:
	@command -v air >/dev/null 2>&1 || { \
		echo "Air is not installed. It enables hot-reload for Go services."; \
		printf "Install via 'go install github.com/air-verse/air@$(AIR_VERSION)'? [y/N] "; \
		read ans; \
		case "$$ans" in [yY]*) go install github.com/air-verse/air@$(AIR_VERSION) ;; *) echo "Aborted."; exit 1 ;; esac; \
	}

_ensure-entr:
	@command -v entr >/dev/null 2>&1 || { \
		echo "entr is not installed. It watches files and re-runs commands on changes."; \
		if command -v brew >/dev/null 2>&1; then \
			printf "Install via 'brew install entr'? [y/N] "; \
			read ans; \
			case "$$ans" in [yY]*) brew install entr ;; *) echo "Aborted."; exit 1 ;; esac; \
		elif command -v apt-get >/dev/null 2>&1; then \
			printf "Install via 'sudo apt-get install entr'? [y/N] "; \
			read ans; \
			case "$$ans" in [yY]*) sudo apt-get install -y entr ;; *) echo "Aborted."; exit 1 ;; esac; \
		else \
			echo "Install entr manually: https://github.com/eradman/entr"; \
			exit 1; \
		fi; \
	}

dev: _ensure-air ## Run API service with hot-reload (requires Air)
	air

dev-all: _ensure-air ## Run all services with hot-reload (requires Air)
	@echo "Starting all services with hot-reload..."
	@echo "  \033[36m[api]\033[0m       .air.toml"
	@echo "  \033[33m[delivery]\033[0m  .air.delivery.toml"
	@echo "  \033[35m[analytics]\033[0m .air.analytics.toml"
	@echo ""
	@trap 'kill 0' INT TERM; \
		air -c .air.toml 2>&1              | sed 's/^/\x1b[36m[api]       \x1b[0m /' & \
		air -c .air.delivery.toml 2>&1     | sed 's/^/\x1b[33m[delivery]  \x1b[0m /' & \
		air -c .air.analytics.toml 2>&1    | sed 's/^/\x1b[35m[analytics] \x1b[0m /' & \
		wait

# Testing
test: ## Run core tests with coverage summary
	@go test -short -coverprofile=coverage.out ./internal/... ./pkg/database/... ./pkg/repository/... ./pkg/auth/... ./pkg/queue/... ./pkg/models/... ./pkg/errors/... ./pkg/signatures/...
	@echo ""
	@go tool cover -func=coverage.out | grep total: | awk '{printf "Coverage: %s\n", $$3}'
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total: | awk '{gsub(/%/,""); print $$3}'); \
	if [ "$$(echo "$$COVERAGE < 70" | bc -l 2>/dev/null || echo 0)" = "1" ]; then \
		echo "⚠️  WARNING: Coverage $${COVERAGE}% is below the 70% CI threshold."; \
	fi
	@echo "Detailed report: go tool cover -html=coverage.out"

test-quick: ## Fast inner-loop tests: no coverage, parallel, -short (for TDD)
	@go test -short -count=1 -parallel=4 ./internal/... ./pkg/...

test-all: ## Run all tests including enterprise packages
	go test -v -timeout=15m ./...

test-watch: _ensure-entr ## Watch for changes and re-run tests (auto-installs entr if needed)
	@echo "Watching for changes (entr)... Ctrl+C to stop"
	@find . -name "*.go" | entr -c go test -short -race ./...

test-single: ## Run a single test (usage: make test-single TEST=TestName PKG=./pkg/auth)
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-single TEST=TestName [PKG=./pkg/auth/...]"; \
		echo "Example: make test-single TEST=TestCreateTenant PKG=./pkg/auth/..."; \
		exit 1; \
	fi
	go test -v -run $(TEST) $(or $(PKG),./...)

test-failed: ## Re-run only tests that failed in the last test run
	@if [ ! -f .test-failures ]; then \
		echo "No previous failure log found. Running tests to capture failures..."; \
		go test -short ./internal/... ./pkg/... 2>&1 | tee /dev/stderr | grep '^\-\-\- FAIL:' | awk '{print $$3}' > .test-failures; \
	fi
	@if [ ! -s .test-failures ]; then \
		echo "No test failures to re-run. ✅"; \
		rm -f .test-failures; \
	else \
		FAILED=$$(paste -sd '|' .test-failures); \
		echo "Re-running failed tests: $$FAILED"; \
		go test -v -run "$$FAILED" ./... && rm -f .test-failures; \
	fi

test-coverage: ## Run tests with per-package coverage breakdown
	@echo "Running tests with per-package coverage..."
	@go test -short -coverprofile=coverage.out ./internal/... ./pkg/...
	@echo ""
	@echo "=== Per-Package Coverage ==="
	@go tool cover -func=coverage.out | grep -E '^[a-z]' | awk -F'\t' '{split($$1,a,"/"); pkg=a[length(a)-1]"/"a[length(a)]; coverage[pkg]=$$NF} END {for (p in coverage) printf "  %-45s %s\n", p, coverage[p]}' | sort
	@echo ""
	@go tool cover -func=coverage.out | grep total: | awk '{printf "Total Coverage: %s\n", $$3}'
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out"

test-status: ## Show test file inventory per package (find coverage gaps)
	@echo "=== Test File Inventory ==="
	@echo ""
	@echo "Core packages:"
	@for pkg in internal/api internal/delivery internal/analytics pkg/models pkg/repository pkg/queue pkg/auth pkg/errors pkg/signatures pkg/database pkg/delivery pkg/utils; do \
		count=$$(find $$pkg -name '*_test.go' 2>/dev/null | wc -l | tr -d ' '); \
		if [ "$$count" -eq 0 ]; then status="❌ no tests"; else status="✅ $$count test file(s)"; fi; \
		printf "  %-35s %s\n" "$$pkg" "$$status"; \
	done
	@echo ""
	@echo "Enterprise packages (with tests):"
	@for dir in pkg/*/; do \
		pkg=$$(basename $$dir); \
		case $$pkg in models|repository|queue|auth|errors|signatures|database|delivery|utils) continue;; esac; \
		count=$$(find $$dir -name '*_test.go' 2>/dev/null | wc -l | tr -d ' '); \
		if [ "$$count" -gt 0 ]; then printf "  %-35s ✅ %s test file(s)\n" "pkg/$$pkg" "$$count"; fi; \
	done
	@echo ""
	@total=$$(find . -name '*_test.go' -not -path './vendor/*' | wc -l | tr -d ' '); \
	pkgs_with=$$(find pkg -name '*_test.go' | xargs -n1 dirname 2>/dev/null | sort -u | wc -l | tr -d ' '); \
	pkgs_total=$$(find pkg -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' '); \
	echo "Summary: $$total test files across $$pkgs_with/$$pkgs_total packages"

test-stubs: ## Generate _test.go skeleton for every package directory lacking test files
	@echo "=== Generating test stubs ==="
	@generated=0; \
	for dir in pkg/*/; do \
		pkg=$$(basename "$$dir"); \
		if ls "$$dir"*_test.go >/dev/null 2>&1; then continue; fi; \
		stub="$${dir}$${pkg}_test.go"; \
		printf "  Creating %s\n" "$$stub"; \
		printf 'package %s\n\nimport "testing"\n\nfunc TestPlaceholder(t *testing.T) {\n\tt.Skip("TODO: add tests for package %s")\n}\n' "$$pkg" "$$pkg" > "$$stub"; \
		generated=$$((generated + 1)); \
	done; \
	echo ""; \
	echo "Generated $$generated test stub(s)."

# Package scaffolding
new-pkg: ## Scaffold a new package under pkg/ (usage: make new-pkg NAME=foo)
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make new-pkg NAME=<package-name>"; \
		echo "Example: make new-pkg NAME=notifications"; \
		exit 1; \
	fi
	@if [ -d "pkg/$(NAME)" ]; then \
		echo "❌ pkg/$(NAME) already exists."; \
		exit 1; \
	fi
	@echo "Scaffolding pkg/$(NAME)..."
	@mkdir -p "pkg/$(NAME)"
	@printf 'package $(NAME)\n\nimport "time"\n\n// TODO: Define domain models for the $(NAME) package.\n\n// Entity represents a $(NAME) entity.\ntype Entity struct {\n\tID        string    `json:"id"`\n\tTenantID  string    `json:"tenant_id"`\n\tCreatedAt time.Time `json:"created_at"`\n\tUpdatedAt time.Time `json:"updated_at"`\n}\n' > "pkg/$(NAME)/models.go"
	@printf 'package $(NAME)\n\nimport "context"\n\n// Repository defines the storage interface for $(NAME).\ntype Repository interface {\n\tGet(ctx context.Context, id string) (*Entity, error)\n\tList(ctx context.Context, tenantID string) ([]Entity, error)\n\tSave(ctx context.Context, entity *Entity) error\n\tDelete(ctx context.Context, id string) error\n}\n' > "pkg/$(NAME)/repository.go"
	@printf 'package $(NAME)\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/google/uuid"\n)\n\n// Service implements $(NAME) business logic.\ntype Service struct {\n\trepo Repository\n}\n\n// NewService creates a new $(NAME) service.\nfunc NewService(repo Repository) *Service {\n\treturn &Service{repo: repo}\n}\n\n// Create creates a new entity.\nfunc (s *Service) Create(ctx context.Context, tenantID string) (*Entity, error) {\n\tentity := &Entity{\n\t\tID:        uuid.New().String(),\n\t\tTenantID:  tenantID,\n\t\tCreatedAt: time.Now(),\n\t\tUpdatedAt: time.Now(),\n\t}\n\n\tif err := s.repo.Save(ctx, entity); err != nil {\n\t\treturn nil, fmt.Errorf("create $(NAME) entity: %%w", err)\n\t}\n\n\treturn entity, nil\n}\n' > "pkg/$(NAME)/service.go"
	@printf 'package $(NAME)\n\nimport (\n\t"encoding/json"\n\t"net/http"\n)\n\n// Handler provides HTTP handlers for $(NAME).\ntype Handler struct {\n\tsvc *Service\n}\n\n// NewHandler creates a new $(NAME) handler.\nfunc NewHandler(svc *Service) *Handler {\n\treturn &Handler{svc: svc}\n}\n\n// List handles GET requests.\nfunc (h *Handler) List(w http.ResponseWriter, r *http.Request) {\n\ttenantID := r.URL.Query().Get("tenant_id")\n\tentities, err := h.svc.repo.List(r.Context(), tenantID)\n\tif err != nil {\n\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n\t\treturn\n\t}\n\n\tw.Header().Set("Content-Type", "application/json")\n\tjson.NewEncoder(w).Encode(entities)\n}\n' > "pkg/$(NAME)/handler.go"
	@printf 'package $(NAME)\n\nimport "testing"\n\nfunc TestNewService(t *testing.T) {\n\tsvc := NewService(nil)\n\tif svc == nil {\n\t\tt.Fatal("NewService returned nil")\n\t}\n}\n\nfunc TestPlaceholder(t *testing.T) {\n\tt.Skip("TODO: add tests for package $(NAME)")\n}\n' > "pkg/$(NAME)/$(NAME)_test.go"
	@echo "  pkg/$(NAME)/models.go"
	@echo "  pkg/$(NAME)/repository.go"
	@echo "  pkg/$(NAME)/service.go"
	@echo "  pkg/$(NAME)/handler.go"
	@echo "  pkg/$(NAME)/$(NAME)_test.go"
	@echo ""
	@echo "✅ Package pkg/$(NAME) scaffolded. Next steps:"
	@echo "   1. Define your domain models in models.go"
	@echo "   2. Implement the repository interface"
	@echo "   3. Add business logic to service.go"
	@echo "   4. Wire the handler into cmd/api-service/main.go"

test-integration: ## Run integration tests in Docker (isolated from dev environment)
	@echo "Starting isolated test environment..."
	@docker-compose -f docker-compose.test.yml up -d postgres redis
	@echo "Waiting for test services..."
	@sleep 5
	@docker-compose -f docker-compose.test.yml exec -T postgres pg_isready -U postgres -d webhook_platform_test >/dev/null 2>&1 || sleep 5
	@echo "Running integration tests..."
	@TEST_DATABASE_URL=postgres://postgres:password@localhost:5433/webhook_platform_test?sslmode=disable \
		TEST_REDIS_URL=redis://localhost:6380/10 \
		go test -v -tags integration -timeout=10m ./test/integration/... || (docker-compose -f docker-compose.test.yml down -v && exit 1)
	@docker-compose -f docker-compose.test.yml down -v
	@echo "Integration tests complete. Test environment cleaned up."

# Code quality
lint: _ensure-golangci-lint ## Run golangci-lint
	golangci-lint run ./...

lint-handlers: _ensure-golangci-lint ## Run errcheck specifically on handler files (no exclusions)
	golangci-lint run --enable errcheck --disable-all -e '.*_test\.go' ./internal/api/handlers/

lint-fast: _ensure-golangci-lint ## Run golangci-lint --fast on changed packages only
	@CHANGED=$$(git diff --name-only --diff-filter=ACM HEAD | grep '\.go$$' | xargs -n1 dirname 2>/dev/null | sort -u | sed 's|^|./|'); \
	if [ -z "$$CHANGED" ]; then echo "No changed Go packages to lint."; \
	else echo "Linting changed packages: $$CHANGED"; golangci-lint run --fast $$CHANGED; fi

fmt: ## Check code formatting (fails if unformatted)
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; echo "$$unformatted"; \
		echo "Run 'make fmt-fix' to fix."; exit 1; \
	fi; \
	echo "All files formatted."

fmt-fix: ## Auto-format all Go source files
	gofmt -w .
	@echo "All files formatted."

fix: fmt-fix ## Auto-format and tidy modules
	go mod tidy
	@echo "Format and tidy complete."

vet: ## Run go vet
	go vet ./...

profile: ## Run CPU profile and open pprof in browser (usage: make profile [PKG=./pkg/auth] [BENCH=.])
	@echo "Running CPU profile..."
	@PKG=$(or $(PKG),./pkg/...); \
	BENCH=$(or $(BENCH),.); \
	go test -bench=$$BENCH -cpuprofile=cpu.prof -benchmem $$PKG; \
	echo ""; \
	echo "Profile saved to cpu.prof"; \
	echo "Opening pprof web UI..."; \
	go tool pprof -http=:6060 cpu.prof

check: fmt vet lint test ## Run all quality gates: fmt, vet, lint, test

ci-local: ## Run the full CI pipeline locally (fmt, vet, lint, test+coverage, build, env check)
	@echo "=== CI Local Pipeline ==="
	@echo ""
	@echo "--- Step 1/6: Formatting ---"
	$(MAKE) --no-print-directory fmt
	@echo ""
	@echo "--- Step 2/6: Vet ---"
	$(MAKE) --no-print-directory vet
	@echo ""
	@echo "--- Step 3/6: Lint ---"
	$(MAKE) --no-print-directory lint
	@echo ""
	@echo "--- Step 4/6: Unit tests with coverage threshold ---"
	@go test -race -coverprofile=coverage.out -covermode=atomic ./pkg/... ./internal/...
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total: | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $${COVERAGE}%"; \
	THRESHOLD=70; \
	if [ "$$(echo "$$COVERAGE < $$THRESHOLD" | bc -l 2>/dev/null || echo 0)" = "1" ]; then \
		echo "❌ Coverage $${COVERAGE}% is below the required $${THRESHOLD}% threshold"; \
		exit 1; \
	fi; \
	echo "✅ Coverage $${COVERAGE}% meets the $${THRESHOLD}% threshold"
	@echo ""
	@echo "--- Step 5/6: Build ---"
	$(MAKE) --no-print-directory build
	@go build ./...
	@echo ""
	@echo "--- Step 6/6: .env.example completeness ---"
	@if [ -f .env.example ]; then \
		ENV_VARS_IN_CODE=$$(grep -roh 'os\.Getenv("[^"]*")' --include='*.go' . 2>/dev/null \
			| grep -v '_test\.go' \
			| sed 's/os\.Getenv("//;s/")//' \
			| grep -v -E '^(RUN_|TEST_)' \
			| sort -u); \
		MISSING=""; \
		for var in $$ENV_VARS_IN_CODE; do \
			if ! grep -q "^$${var}=" .env.example 2>/dev/null && ! grep -q "^# *$${var}=" .env.example 2>/dev/null; then \
				MISSING="$$MISSING $$var"; \
			fi; \
		done; \
		if [ -n "$$MISSING" ]; then \
			echo "❌ Missing from .env.example:$$MISSING"; \
			exit 1; \
		fi; \
		echo "✅ .env.example is up to date."; \
	else \
		echo "⚠️  .env.example not found, skipping."; \
	fi
	@echo ""
	@echo "=== ✅ CI Local Pipeline Passed ==="

# Clean
clean: ## Remove build artifacts
	rm -rf bin/

# Docker
docker-up: ## Start development containers (PostgreSQL + Redis)
	docker-compose up -d

docker-down: ## Stop development containers
	docker-compose down

docker-build: ## Build Docker images
	docker-compose build

up: docker-up ## Alias for docker-up

down: docker-down ## Alias for docker-down

logs: ## Stream logs from development containers
	docker-compose logs -f

# Database migrations (uses Docker image if local migrate CLI is not installed)
_require-db-url:
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "❌ DATABASE_URL is not set. Export it or run 'make ensure-env' and source .env first."; \
		exit 1; \
	fi

MIGRATE_CMD = $(shell command -v migrate >/dev/null 2>&1 \
	&& echo 'migrate -path migrations -database "$(DATABASE_URL)"' \
	|| echo 'docker run --rm -v $(CURDIR)/migrations:/migrations --network host migrate/migrate -path=/migrations -database "$(DATABASE_URL)"')

migrate-up: _require-db-url ## Run database migrations
	$(MIGRATE_CMD) up

migrate-down: _require-db-url ## Rollback database migrations
	$(MIGRATE_CMD) down

migrate-core: _require-db-url ## Run core-only migrations (tables 1-5: tenants, endpoints, deliveries, analytics, quotas)
	$(MIGRATE_CMD) up 5

migrate-status: _require-db-url ## Show current migration version
	$(MIGRATE_CMD) version

migrate-rollback-last: _require-db-url ## Rollback the last migration only
	$(MIGRATE_CMD) down 1

migrate-reset: _require-db-url ## Drop all and re-run migrations (DESTRUCTIVE - requires confirmation)
	@echo "⚠️  WARNING: This will DROP all tables and re-run all migrations."
	@echo "   Database: $(DATABASE_URL)"
	@echo ""
	@read -p "Type 'yes' to confirm: " confirm && [ "$$confirm" = "yes" ] || (echo "Aborted."; exit 1)
	$(MIGRATE_CMD) drop -f
	$(MIGRATE_CMD) up
	@echo "✅ Database reset complete."

# Environment file
ensure-env: ## Create .env from template if it doesn't exist
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
		JWT=$$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom 2>/dev/null | head -c 32 || openssl rand -hex 16); \
		if [ -n "$$JWT" ]; then \
			sed -i.bak "s/change-me-in-production/$$JWT/" .env && rm -f .env.bak; \
			echo "  Generated random JWT_SECRET."; \
		fi; \
		echo "  .env ready."; \
	fi

validate-env: ## Validate .env file for required variables and insecure defaults
	@if [ ! -f .env ]; then \
		echo "❌ .env file not found. Run 'make ensure-env' first."; \
		exit 1; \
	fi
	@echo "Checking .env configuration..."
	@errors=0; \
	if ! grep -q '^DATABASE_URL=' .env 2>/dev/null; then echo "  ❌ DATABASE_URL is missing"; errors=$$((errors+1)); fi; \
	if ! grep -q '^REDIS_URL=' .env 2>/dev/null && ! grep -q '^REDIS_ADDR=' .env 2>/dev/null; then echo "  ⚠️  REDIS_URL not set (will use default localhost:6379)"; fi; \
	if ! grep -q '^JWT_SECRET=' .env 2>/dev/null; then echo "  ❌ JWT_SECRET is missing"; errors=$$((errors+1)); fi; \
	if grep -q 'change-me-in-production' .env 2>/dev/null; then echo "  ⚠️  JWT_SECRET still has default value — change before deploying"; fi; \
	if grep -q '^JWT_SECRET=$$' .env 2>/dev/null; then echo "  ❌ JWT_SECRET is empty"; errors=$$((errors+1)); fi; \
	if [ $$errors -gt 0 ]; then echo ""; echo "$$errors error(s) found. Fix .env before running."; exit 1; \
	else echo "  ✅ .env looks good."; fi

# Development setup
_wait-for-services:
	@echo "Waiting for PostgreSQL to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30; do \
		docker-compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1 && break; \
		echo "  Attempt $$i/30 - PostgreSQL not ready, retrying in 2s..."; \
		sleep 2; \
	done
	@docker-compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1 || { echo "PostgreSQL failed to start after 60s"; exit 1; }
	@echo "PostgreSQL is ready."
	@echo "Waiting for Redis to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		docker-compose exec -T redis redis-cli ping >/dev/null 2>&1 && break; \
		sleep 1; \
	done
	@echo "Redis is ready."

dev-setup: ensure-env install-hooks docker-up _wait-for-services ## One-command dev setup: creates .env, installs hooks, starts containers, runs core migrations
	$(MAKE) migrate-core
	@echo ""
	@echo "Development environment is ready! (core migrations only — run 'make dev-setup-full' for all migrations)"
	@echo "  API service:       make run-api        (http://localhost:8080)"
	@echo "  Delivery engine:   make run-delivery"
	@echo "  Analytics service: make run-analytics   (http://localhost:8082)"
	@echo "  All services:      make run-all"
	@echo "  Hot-reload:        make dev             (requires Air)"
	@echo "  React dashboard:   make run-dashboard   (http://localhost:5173)"
	@echo "  Smoke test:        make smoke-test"
	@echo "  API docs:          http://localhost:8080/docs/"

dev-setup-full: ensure-env install-hooks docker-up _wait-for-services ## Full dev setup with all migrations (not just core)
	$(MAKE) migrate-up
	@echo ""
	@echo "Development environment is ready! (all migrations applied)"
	@echo "  API service:       make run-api        (http://localhost:8080)"
	@echo "  Delivery engine:   make run-delivery"
	@echo "  Analytics service: make run-analytics   (http://localhost:8082)"
	@echo "  All services:      make run-all"
	@echo "  Hot-reload:        make dev             (requires Air)"
	@echo "  React dashboard:   make run-dashboard   (http://localhost:5173)"
	@echo "  Smoke test:        make smoke-test"
	@echo "  API docs:          http://localhost:8080/docs/"

# Dependencies
deps: ## Download and tidy Go modules
	go mod tidy
	go mod download

# Documentation
docs: ## Generate Swagger/OpenAPI docs
	@command -v swag >/dev/null 2>&1 || { echo "swag not found. Install: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	swag init -g cmd/api-service/main.go -o docs --parseDependency --parseInternal

docs-generate: ## Generate and validate all documentation
	./scripts/generate-docs.sh

docs-serve: docs ## Generate docs and print access URL
	@echo "API documentation available at: http://localhost:8080/docs/index.html"

open-docs: docs ## Generate docs and open in browser (requires running API)
	@echo "Opening API docs in browser..."
	@if command -v open >/dev/null 2>&1; then \
		open http://localhost:8080/docs/index.html; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open http://localhost:8080/docs/index.html; \
	else \
		echo "Could not detect browser opener. Visit: http://localhost:8080/docs/index.html"; \
	fi

docs-test: ## Run documentation tests
	go test ./docs -v

# Smoke test
smoke-test: ## Quick smoke test against running API (requires make run-api)
	@echo "Running smoke test against http://localhost:8080..."
	@echo ""
	@printf "  %-30s" "Health check:" ; \
		STATUS=$$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health 2>/dev/null); \
		if [ "$$STATUS" = "200" ]; then echo "✅ OK"; else echo "❌ FAIL (HTTP $$STATUS — is the API running?)"; exit 1; fi
	@printf "  %-30s" "Create tenant:" ; \
		RESP=$$(curl -s -X POST http://localhost:8080/api/v1/tenants \
			-H "Content-Type: application/json" \
			-d '{"name": "smoke-test-tenant", "subscription_tier": "free"}' 2>/dev/null); \
		API_KEY=$$(echo "$$RESP" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4); \
		if [ -n "$$API_KEY" ]; then echo "✅ OK"; else echo "❌ FAIL: $$RESP"; exit 1; fi
	@printf "  %-30s" "Create endpoint:" ; \
		RESP=$$(curl -s -X POST http://localhost:8080/api/v1/tenants \
			-H "Content-Type: application/json" \
			-d '{"name": "smoke-ep-tenant", "subscription_tier": "free"}' 2>/dev/null); \
		KEY=$$(echo "$$RESP" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4); \
		EP_RESP=$$(curl -s -X POST http://localhost:8080/api/v1/webhooks/endpoints \
			-H "Content-Type: application/json" \
			-H "Authorization: Bearer $$KEY" \
			-d '{"url": "https://httpbin.org/post"}' 2>/dev/null); \
		if echo "$$EP_RESP" | grep -q '"id"'; then echo "✅ OK"; else echo "❌ FAIL: $$EP_RESP"; exit 1; fi
	@printf "  %-30s" "Auth rejection:" ; \
		STATUS=$$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/webhooks/endpoints 2>/dev/null); \
		if [ "$$STATUS" = "401" ]; then echo "✅ OK (401)"; else echo "❌ FAIL (expected 401, got $$STATUS)"; exit 1; fi
	@printf "  %-30s" "Request ID header:" ; \
		RID=$$(curl -s -D- -o /dev/null http://localhost:8080/health 2>/dev/null | grep -i 'X-Request-ID' | tr -d '\r'); \
		if [ -n "$$RID" ]; then echo "✅ OK"; else echo "⚠️  Missing X-Request-ID header"; fi
	@printf "  %-30s" "API version header:" ; \
		VER=$$(curl -s -D- -o /dev/null http://localhost:8080/health 2>/dev/null | grep -i 'X-API-Version' | tr -d '\r'); \
		if [ -n "$$VER" ]; then echo "✅ OK"; else echo "⚠️  Missing X-API-Version header"; fi
	@printf "  %-30s" "Swagger docs:" ; \
		STATUS=$$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/docs/index.html 2>/dev/null); \
		if [ "$$STATUS" = "200" ]; then echo "✅ OK"; else echo "⚠️  Not available (HTTP $$STATUS)"; fi
	@echo ""
	@echo "Smoke test passed! 🎉"

# Seed demo data
seed: ## Seed sample tenants, endpoints, and deliveries via the API (requires make run-api)
	@./scripts/seed-dev-data.sh

# Quickstart
quickstart: dev-setup ## One-command demo: setup, start API, wait for health, seed data
	@echo ""
	@echo "Starting API service in background..."
	@go run ./cmd/api-service &>/dev/null & API_PID=$$!; \
	echo "$$API_PID" > .api.pid; \
	echo "  API PID: $$API_PID"; \
	echo "Waiting for API to be ready..."; \
	READY=0; \
	for i in $$(seq 1 30); do \
		if curl -sf http://localhost:8080/health >/dev/null 2>&1; then \
			READY=1; break; \
		fi; \
		printf "  Attempt $$i/30 — waiting 2s...\n"; \
		sleep 2; \
	done; \
	if [ "$$READY" -ne 1 ]; then \
		echo "❌ API failed to start after 60s"; \
		kill $$API_PID 2>/dev/null; \
		exit 1; \
	fi; \
	echo "✅ API is healthy"; \
	echo ""; \
	echo "Seeding demo data..."; \
	./scripts/seed-dev-data.sh || true; \
	echo ""; \
	echo "=============================================="; \
	echo "  🚀 WaaS Quick Start — Ready!"; \
	echo "=============================================="; \
	echo ""; \
	echo "  API:          http://localhost:8080"; \
	echo "  Swagger docs: http://localhost:8080/docs/index.html"; \
	echo "  Health:       http://localhost:8080/health"; \
	echo ""; \
	echo "  API is running in background (PID $$API_PID)."; \
	echo "  Stop with: make stop"; \
	echo ""; \
	echo "  Next steps:"; \
	echo "    make run-dashboard   Start React UI (port 5173)"; \
	echo "    make smoke-test      Verify API endpoints"; \
	echo "    make test            Run test suite"; \
	echo "=============================================="

stop: ## Stop background API started by quickstart
	@if [ -f .api.pid ]; then \
		PID=$$(cat .api.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			kill $$PID; \
			echo "Stopped API service (PID $$PID)."; \
		else \
			echo "API process (PID $$PID) is not running."; \
		fi; \
		rm -f .api.pid; \
	else \
		echo "No PID file found. Is the API running via 'make quickstart'?"; \
	fi

# Environment validation
validate-setup: ## Check that all prerequisites are installed
	@echo "Checking prerequisites..."
	@echo ""
	@printf "  %-20s" "Go 1.24+:" ; \
		go version >/dev/null 2>&1 && echo "OK ($$(go version | awk '{print $$3}'))" || echo "MISSING - install from https://go.dev/dl/"
	@printf "  %-20s" "Docker:" ; \
		docker --version >/dev/null 2>&1 && echo "OK" || echo "MISSING - install from https://docs.docker.com/get-docker/"
	@printf "  %-20s" "Docker Compose:" ; \
		(docker compose version >/dev/null 2>&1 || docker-compose --version >/dev/null 2>&1) && echo "OK" || echo "MISSING"
	@printf "  %-20s" "golang-migrate:" ; \
		migrate -version >/dev/null 2>&1 && echo "OK" || echo "MISSING (optional) - migrations run via Docker image as fallback"
	@printf "  %-20s" "golangci-lint:" ; \
		golangci-lint --version >/dev/null 2>&1 && echo "OK" || echo "MISSING (optional) - install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
	@printf "  %-20s" "swag:" ; \
		swag --version >/dev/null 2>&1 && echo "OK" || echo "MISSING (optional) - install: go install github.com/swaggo/swag/cmd/swag@latest"
	@echo ""
	@printf "  %-20s" "Port 5432 (PG):" ; \
		(lsof -i :5432 >/dev/null 2>&1 && echo "IN USE") || echo "Available"
	@printf "  %-20s" "Port 6379 (Redis):" ; \
		(lsof -i :6379 >/dev/null 2>&1 && echo "IN USE") || echo "Available"
	@printf "  %-20s" "Port 8080 (API):" ; \
		(lsof -i :8080 >/dev/null 2>&1 && echo "IN USE") || echo "Available"
	@printf "  %-20s" "Port 8082 (Analytics):" ; \
		(lsof -i :8082 >/dev/null 2>&1 && echo "IN USE") || echo "Available"
	@echo ""
	@printf "  %-20s" "go.work:" ; \
		if [ -f ../go.work ]; then echo "OK"; \
		elif [ -f ../go.work.example ]; then echo "MISSING — copy go.work.example to go.work: cp ../go.work.example ../go.work"; \
		else echo "MISSING"; fi

doctor: ## Full environment diagnostics (prerequisites + env + Docker + ports)
	@echo "🩺 WaaS Doctor — Environment Diagnostics"
	@echo "========================================="
	@echo ""
	@echo "--- Prerequisites ---"
	@$(MAKE) --no-print-directory validate-setup
	@echo "--- Environment File ---"
	@if [ -f .env ]; then \
		$(MAKE) --no-print-directory validate-env; \
	else \
		echo "  ⚠️  .env not found. Run 'make ensure-env' to create it."; \
	fi
	@echo ""
	@echo "--- Docker Connectivity ---"
	@printf "  %-20s" "Docker daemon:" ; \
		docker info >/dev/null 2>&1 && echo "✅ Running" || echo "❌ Not running — start Docker Desktop or 'sudo systemctl start docker'"
	@printf "  %-20s" "Compose services:" ; \
		docker-compose ps --services >/dev/null 2>&1 && echo "✅ Compose file OK" || echo "⚠️  Could not read docker-compose.yml"
	@printf "  %-20s" "PostgreSQL:" ; \
		docker-compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1 && echo "✅ Accepting connections" || echo "⚠️  Not reachable (run 'make docker-up')"
	@printf "  %-20s" "Redis:" ; \
		docker-compose exec -T redis redis-cli ping >/dev/null 2>&1 && echo "✅ PONG" || echo "⚠️  Not reachable (run 'make docker-up')"
	@echo ""
	@echo "--- Port Availability ---"
	@printf "  %-22s" "5432 (PostgreSQL):" ; (lsof -i :5432 >/dev/null 2>&1 && echo "IN USE") || echo "Available"
	@printf "  %-22s" "6379 (Redis):" ; (lsof -i :6379 >/dev/null 2>&1 && echo "IN USE") || echo "Available"
	@printf "  %-22s" "8080 (API):" ; (lsof -i :8080 >/dev/null 2>&1 && echo "IN USE") || echo "Available"
	@printf "  %-22s" "8082 (Analytics):" ; (lsof -i :8082 >/dev/null 2>&1 && echo "IN USE") || echo "Available"
	@echo ""
	@echo "Done. Fix any ❌ items above before running services."

# Git hooks
GIT_HOOKS_DIR := $(shell git rev-parse --git-dir 2>/dev/null)/hooks

install-hooks: ## Install git pre-commit hooks
	@mkdir -p $(GIT_HOOKS_DIR)
	@cp scripts/pre-commit $(GIT_HOOKS_DIR)/pre-commit
	@chmod +x $(GIT_HOOKS_DIR)/pre-commit
	@echo "Pre-commit hook installed. It will run gofmt and go vet on staged files."

uninstall-hooks: ## Remove git pre-commit hooks
	@rm -f $(GIT_HOOKS_DIR)/pre-commit
	@echo "Pre-commit hook removed."

# Dashboard
run-dashboard: ## Run React dashboard (connects to API on :8080)
	@echo "Starting React dashboard..."
	@if command -v pnpm >/dev/null 2>&1; then \
		cd web/dashboard && pnpm install --frozen-lockfile && VITE_API_URL=http://localhost:8080 pnpm dev; \
	elif command -v npm >/dev/null 2>&1; then \
		cd web/dashboard && npm install && VITE_API_URL=http://localhost:8080 npm run dev; \
	else \
		echo "Neither pnpm nor npm found. Install Node.js: https://nodejs.org/"; exit 1; \
	fi

# Dependency Auditing
audit-deps: ## Check for outdated and vulnerable dependencies
	@echo "=== Checking for outdated modules ==="
	@go list -m -u -json all 2>/dev/null | go run golang.org/x/tools/cmd/godeps@latest 2>/dev/null || \
		go list -m -u all 2>/dev/null | grep '\[' || echo "  All modules up to date."
	@echo ""
	@echo "=== Checking for known vulnerabilities ==="
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "  govulncheck not installed. Install: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		echo "  Running 'go mod verify' instead..."; \
		go mod verify; \
	fi
