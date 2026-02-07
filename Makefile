# Root Makefile — delegates to backend/Makefile so contributors can run
# common targets from the repository root without `cd backend` first.

BACKEND := backend

.DEFAULT_GOAL := help

# ─── Development setup ──────────────────────────────────────────────
.PHONY: dev-setup dev-setup-full validate-setup ensure-env validate-env doctor

dev-setup: ## One-command dev setup (env, containers, core migrations)
	$(MAKE) -C $(BACKEND) dev-setup

dev-setup-full: ## Full dev setup with all migrations
	$(MAKE) -C $(BACKEND) dev-setup-full

validate-setup: ## Check that all prerequisites are installed
	$(MAKE) -C $(BACKEND) validate-setup

ensure-env: ## Create backend/.env from template if missing
	$(MAKE) -C $(BACKEND) ensure-env

validate-env: ## Validate backend/.env for required variables
	$(MAKE) -C $(BACKEND) validate-env

doctor: ## Full environment diagnostics in one shot
	$(MAKE) -C $(BACKEND) doctor

# ─── Run services ───────────────────────────────────────────────────
.PHONY: run-api run-delivery run-analytics run-all run-dashboard dev dev-all dev-logs

run-api: ## Run API service (port 8080)
	$(MAKE) -C $(BACKEND) run-api

run-delivery: ## Run delivery engine
	$(MAKE) -C $(BACKEND) run-delivery

run-analytics: ## Run analytics service (port 8082)
	$(MAKE) -C $(BACKEND) run-analytics

run-all: ## Run all services in parallel
	$(MAKE) -C $(BACKEND) run-all

run-dashboard: ## Run React dashboard (port 5173)
	$(MAKE) -C $(BACKEND) run-dashboard

dev: ## Run API with hot-reload (requires Air)
	$(MAKE) -C $(BACKEND) dev

dev-all: ## Run all services with hot-reload (requires Air)
	$(MAKE) -C $(BACKEND) dev-all

dev-logs: ## Run all services with colored log output
	$(MAKE) -C $(BACKEND) dev-logs

# ─── Testing & quality ──────────────────────────────────────────────
.PHONY: test test-all test-watch test-coverage test-integration lint fmt vet check

test: ## Run core tests with coverage summary
	$(MAKE) -C $(BACKEND) test

test-all: ## Run all tests including enterprise packages
	$(MAKE) -C $(BACKEND) test-all

test-watch: ## Watch for changes and re-run tests (requires entr or fswatch)
	$(MAKE) -C $(BACKEND) test-watch

test-coverage: ## Per-package coverage breakdown
	$(MAKE) -C $(BACKEND) test-coverage

test-integration: ## Integration tests in Docker
	$(MAKE) -C $(BACKEND) test-integration

lint: ## Run golangci-lint
	$(MAKE) -C $(BACKEND) lint

fmt: ## Check code formatting
	$(MAKE) -C $(BACKEND) fmt

vet: ## Run go vet
	$(MAKE) -C $(BACKEND) vet

check: ## Run all quality gates (fmt, vet, lint, test)
	$(MAKE) -C $(BACKEND) check

# ─── Build ──────────────────────────────────────────────────────────
.PHONY: build build-check clean

build: ## Build all service binaries
	$(MAKE) -C $(BACKEND) build

build-check: ## Compile-check all packages
	$(MAKE) -C $(BACKEND) build-check

clean: ## Remove build artifacts
	$(MAKE) -C $(BACKEND) clean

# ─── Database ───────────────────────────────────────────────────────
.PHONY: migrate-up migrate-down migrate-core migrate-status migrate-reset

migrate-up: ## Run all database migrations
	$(MAKE) -C $(BACKEND) migrate-up

migrate-down: ## Rollback database migrations
	$(MAKE) -C $(BACKEND) migrate-down

migrate-core: ## Run core-only migrations (5 tables)
	$(MAKE) -C $(BACKEND) migrate-core

migrate-status: ## Show current migration version
	$(MAKE) -C $(BACKEND) migrate-status

migrate-reset: ## Drop all and re-run migrations (DESTRUCTIVE)
	$(MAKE) -C $(BACKEND) migrate-reset

# ─── Docker ─────────────────────────────────────────────────────────
.PHONY: docker-up docker-down docker-build up down

docker-up: ## Start development containers (PostgreSQL + Redis)
	$(MAKE) -C $(BACKEND) docker-up

docker-down: ## Stop development containers
	$(MAKE) -C $(BACKEND) docker-down

docker-build: ## Build Docker images
	$(MAKE) -C $(BACKEND) docker-build

up: docker-up ## Alias for docker-up
down: docker-down ## Alias for docker-down

# ─── Documentation ──────────────────────────────────────────────────
.PHONY: docs docs-serve smoke-test seed

docs: ## Generate Swagger/OpenAPI docs
	$(MAKE) -C $(BACKEND) docs

docs-serve: ## Generate docs and print access URL
	$(MAKE) -C $(BACKEND) docs-serve

smoke-test: ## Quick smoke test against running API
	$(MAKE) -C $(BACKEND) smoke-test

seed: ## Seed sample tenants, endpoints, and deliveries via the API
	$(MAKE) -C $(BACKEND) seed

# ─── Dependencies & hooks ──────────────────────────────────────────
.PHONY: deps install-hooks uninstall-hooks

deps: ## Download and tidy Go modules
	$(MAKE) -C $(BACKEND) deps

install-hooks: ## Install git pre-commit hooks
	$(MAKE) -C $(BACKEND) install-hooks

uninstall-hooks: ## Remove git pre-commit hooks
	$(MAKE) -C $(BACKEND) uninstall-hooks

# ─── Help ───────────────────────────────────────────────────────────
help: ## Show available targets
	@echo "WaaS Platform — Root Makefile (delegates to backend/)"
	@echo "====================================================="
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "All targets run inside backend/. You can also run:"
	@echo "  cd backend && make help                   Full backend target list"
	@echo "  cd backend && make -f Makefile.test help   Testing commands"
	@echo "  cd backend && make -f Makefile.prod help   Production operations"
