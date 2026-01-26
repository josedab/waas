# Troubleshooting

Common issues and solutions when developing WaaS locally.

## Setup Issues

### `make dev-setup` fails with "docker: command not found"

**Problem:** Docker is not installed or not in PATH.

**Fix:** Install Docker Desktop from https://docker.com and ensure it's running:
```bash
docker --version   # Should show Docker version
docker info        # Should not show errors
```

### PostgreSQL connection refused

**Problem:** `connection refused` or `could not connect to server` when running tests or the API.

**Fix:**
```bash
# Check if PostgreSQL container is running
docker ps | grep postgres

# If not running, start the dev stack
docker-compose up -d postgres

# Verify connectivity
docker-compose exec postgres pg_isready -U postgres
```

If PostgreSQL is running but you still get errors, check your `DATABASE_URL` in `.env`:
```
DATABASE_URL=postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable
```

### Redis connection refused

**Problem:** `dial tcp 127.0.0.1:6379: connect: connection refused`

**Fix:**
```bash
docker-compose up -d redis
docker-compose exec redis redis-cli ping  # Should return PONG
```

### Migrations fail with "dirty database"

**Problem:** A migration partially ran, leaving the database in a dirty state.

**Fix:**
```bash
# Check current version
make migrate-status

# Force the version (replace N with the last clean version)
migrate -path migrations -database "$DATABASE_URL" force N

# Re-run migrations
make migrate-up
```

If all else fails:
```bash
make migrate-reset    # ⚠️ DESTRUCTIVE: drops all tables and re-migrates
```

### "too many open files" during tests

**Problem:** macOS has a low default file descriptor limit.

**Fix:**
```bash
ulimit -n 10240    # Increase for current shell
```

To make permanent, add to `~/.zshrc` or `~/.bashrc`.

## Runtime Issues

### API returns 401 for all requests

**Checklist:**
1. Check that you're passing the API key: `Authorization: Bearer <key>`
2. Verify the key exists in the database: check `api_keys` table
3. Check that `JWT_SECRET` in `.env` matches what was used to generate the key
4. Try creating a new tenant: `curl -X POST http://localhost:8080/api/v1/tenants -H "Content-Type: application/json" -d '{"name":"test","email":"test@example.com"}'`

### Delivery engine not processing webhooks

**Checklist:**
1. Is the delivery engine running? `make run-delivery`
2. Is Redis running? `docker-compose exec redis redis-cli ping`
3. Check the Redis queue: `docker-compose exec redis redis-cli LLEN webhook_deliveries`
4. Check delivery engine logs for errors

### Slow API responses (>500ms)

**Checklist:**
1. Run `make migrate-status` — are all migrations applied?
2. Check PostgreSQL connection pool: look for `max_open_conns` in logs
3. Check if the database has indexes: `\di` in psql
4. Try with fewer enterprise packages loaded (they add middleware overhead)

## Test Issues

### Tests fail with "sql: database is closed"

**Problem:** Tests are trying to use a real database connection that doesn't exist.

**Fix:** Many tests require a running PostgreSQL instance:
```bash
docker-compose up -d postgres redis
make test
```

For truly unit tests that don't need a database, they should use mocks. Check if the test file imports `pkg/database` — if so, it needs a live database.

### `make test-integration` hangs

**Problem:** Docker containers take time to start.

**Fix:**
```bash
# Clean up old test containers
docker-compose -f docker-compose.test.yml down -v

# Start fresh
make test-integration
```

## Build Issues

### `go build` takes >5 minutes

**Problem:** First build compiles all ~90 packages. Subsequent builds use the Go build cache.

**Fix:**
- First build is slow; subsequent builds should be fast (~seconds for incremental changes)
- Check build cache: `go env GOCACHE`
- If developing on a specific feature, build just that package: `go build ./pkg/repository/...`

### "package not found" after `git pull`

**Fix:**
```bash
go mod tidy
go mod download
```

## Getting More Help

- Run `make help` to see all available commands
- Check [QUICK_REFERENCE.md](QUICK_REFERENCE.md) for common commands
- Check [ARCHITECTURE.md](../ARCHITECTURE.md) for how the system is organized
- Review `make validate-setup` output for missing prerequisites
