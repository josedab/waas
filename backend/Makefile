.PHONY: build run-api run-delivery run-analytics test test-all clean \
       docker-up docker-down docker-build \
       migrate-up migrate-down migrate-core dev-setup ensure-env deps \
       docs docs-generate docs-serve docs-test \
       lint fmt vet help validate-setup \
       smoke-test run-all \
       install-hooks uninstall-hooks

# Configuration
DATABASE_URL ?= postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable

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

# Testing
test: ## Run core tests (fast — internal + key packages)
	go test -short ./internal/... ./pkg/database/... ./pkg/repository/... ./pkg/auth/... ./pkg/queue/... ./pkg/models/... ./pkg/errors/... ./pkg/signatures/...

test-all: ## Run all tests including enterprise packages
	go test -v ./...

# Code quality
lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

fmt: ## Check code formatting (fails if unformatted)
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; echo "$$unformatted"; \
		echo "Run 'gofmt -w .' to fix."; exit 1; \
	fi; \
	echo "All files formatted."

vet: ## Run go vet
	go vet ./...

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

# Database migrations (uses Docker image if local migrate CLI is not installed)
MIGRATE_CMD = $(shell command -v migrate >/dev/null 2>&1 \
	&& echo 'migrate -path migrations -database "$(DATABASE_URL)"' \
	|| echo 'docker run --rm -v $(CURDIR)/migrations:/migrations --network host migrate/migrate -path=/migrations -database "$(DATABASE_URL)"')

migrate-up: ## Run database migrations
	$(MIGRATE_CMD) up

migrate-down: ## Rollback database migrations
	$(MIGRATE_CMD) down

migrate-core: ## Run core-only migrations (tables 1-5: tenants, endpoints, deliveries, analytics, quotas)
	$(MIGRATE_CMD) up 5

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

# Development setup
dev-setup: ensure-env docker-up ## One-command dev setup: creates .env, starts containers, runs migrations
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
	$(MAKE) migrate-up
	@echo ""
	@echo "Development environment is ready!"
	@echo "  API service:       make run-api        (http://localhost:8080)"
	@echo "  Delivery engine:   make run-delivery"
	@echo "  Analytics service: make run-analytics   (http://localhost:8082)"
	@echo "  All services:      make run-all"
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
			-d '{"name": "smoke-test-tenant", "email": "smoke@test.com"}' 2>/dev/null); \
		if echo "$$RESP" | grep -q "api_key\|id"; then echo "✅ OK"; else echo "❌ FAIL: $$RESP"; exit 1; fi
	@printf "  %-30s" "Swagger docs:" ; \
		STATUS=$$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/docs/index.html 2>/dev/null); \
		if [ "$$STATUS" = "200" ]; then echo "✅ OK"; else echo "⚠️  Not available (HTTP $$STATUS)"; fi
	@echo ""
	@echo "Smoke test passed! 🎉"

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
