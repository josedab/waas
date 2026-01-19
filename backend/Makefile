.PHONY: build run test clean docker-up docker-down migrate-up migrate-down

# Build all services
build:
	go build -o bin/api-service ./cmd/api-service
	go build -o bin/delivery-engine ./cmd/delivery-engine
	go build -o bin/analytics-service ./cmd/analytics-service

# Run services locally
run-api:
	go run ./cmd/api-service

run-delivery:
	go run ./cmd/delivery-engine

run-analytics:
	go run ./cmd/analytics-service

# Test
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

# Database migrations
migrate-up:
	migrate -path migrations -database "postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable" down

# Development setup
dev-setup: docker-up
	sleep 5
	make migrate-up

# Install dependencies
deps:
	go mod tidy
	go mod download

# Generate API documentation
docs:
	~/go/bin/swag init -g cmd/api-service/main.go -o docs --parseDependency --parseInternal

# Generate and validate all documentation
docs-generate:
	./scripts/generate-docs.sh

# Generate and serve docs
docs-serve: docs
	@echo "API documentation available at: http://localhost:8080/docs/index.html"

# Test documentation
docs-test:
	go test ./docs -v