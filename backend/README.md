# Webhook Service Platform

A comprehensive webhook-as-a-service platform built with Go microservices architecture.

## Project Structure

```
webhook-platform/
├── cmd/                    # Application entry points
│   ├── api-service/        # Webhook API service
│   ├── delivery-engine/    # Webhook delivery engine
│   └── analytics-service/  # Analytics and monitoring service
├── internal/               # Private application code
│   ├── api/               # API service implementation
│   ├── delivery/          # Delivery engine implementation
│   └── analytics/         # Analytics service implementation
├── pkg/                   # Shared packages
│   ├── models/           # Data models
│   ├── database/         # Database utilities
│   └── utils/            # Common utilities
├── migrations/           # Database migrations
├── docker-compose.yml    # Local development environment
└── Makefile             # Development commands
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- golang-migrate CLI tool

### Development Setup

1. Clone the repository
2. Start the development environment:
   ```bash
   make dev-setup
   ```

This will:
- Start PostgreSQL and Redis containers
- Run database migrations
- Set up the development environment

### Running Services

Start all services with Docker:
```bash
make docker-up
```

Or run individual services locally:
```bash
make run-api        # API service on :8080
make run-delivery   # Delivery engine
make run-analytics  # Analytics service on :8082
```

### Database Migrations

Run migrations:
```bash
make migrate-up
```

Rollback migrations:
```bash
make migrate-down
```

### Building

Build all services:
```bash
make build
```

### Testing

Run tests:
```bash
make test
```

## Services

### API Service (Port 8080)
- Webhook endpoint management
- Authentication and rate limiting
- Webhook sending API

### Delivery Engine
- Webhook delivery processing
- Retry logic with exponential backoff
- Queue management

### Analytics Service (Port 8082)
- Metrics collection and aggregation
- Dashboard APIs
- Real-time monitoring

## Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string
- `API_PORT`: API service port (default: 8080)
- `ANALYTICS_PORT`: Analytics service port (default: 8082)
- `JWT_SECRET`: JWT signing secret
- `ENVIRONMENT`: Environment (development/production)