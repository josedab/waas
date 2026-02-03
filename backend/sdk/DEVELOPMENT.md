# SDK Development Guide

How to develop, test, and validate SDK changes locally against the WaaS backend.

## Directory Structure

```
sdk/
├── go/          # Go SDK
├── nodejs/      # Node.js/TypeScript SDK
├── python/      # Python SDK
├── java/        # Java SDK
├── ruby/        # Ruby SDK
├── php/         # PHP SDK
└── CONTRIBUTING.md
```

## Local Development Workflow

### Go SDK

The Go SDK lives in `sdk/go/` with module name `github.com/josedab/waas/sdk/go`.

**To test against local backend changes:**

1. In your consuming project's `go.mod`, add a `replace` directive:
   ```
   replace github.com/josedab/waas/sdk/go => /path/to/waas/backend/sdk/go
   ```

2. Or from within the SDK directory:
   ```bash
   cd sdk/go
   go test ./...
   ```

3. To run SDK examples against a local API:
   ```bash
   # Terminal 1: Start the API
   make run-api

   # Terminal 2: Run SDK examples
   cd sdk/go/examples
   WAAS_API_URL=http://localhost:8080 WAAS_API_KEY=<your-key> go run main.go
   ```

### Node.js SDK

```bash
cd sdk/nodejs
npm install
npm test

# Link for local development in another project
npm link
# In consuming project:
npm link @josedab/waas-sdk
```

Test against local API:
```bash
WAAS_API_URL=http://localhost:8080 WAAS_API_KEY=<key> npm run test:integration
```

### Python SDK

```bash
cd sdk/python
pip install -e .     # Install in editable mode
pytest               # Run tests

# Test against local API
WAAS_API_URL=http://localhost:8080 WAAS_API_KEY=<key> pytest tests/integration/
```

### Java SDK

```bash
cd sdk/java
./gradlew test       # Or: mvn test

# For local dependency in another project, publish to Maven local
./gradlew publishToMavenLocal
```

### Ruby SDK

```bash
cd sdk/ruby
bundle install
bundle exec rspec

# Test against local API
WAAS_API_URL=http://localhost:8080 WAAS_API_KEY=<key> bundle exec rspec spec/integration/
```

### PHP SDK

```bash
cd sdk/php
composer install
./vendor/bin/phpunit

# Test against local API
WAAS_API_URL=http://localhost:8080 WAAS_API_KEY=<key> ./vendor/bin/phpunit tests/Integration/
```

## Testing SDKs Against Local Backend

1. Start the full backend stack:
   ```bash
   cd backend
   make dev-setup
   make run-api
   ```

2. Create a test tenant and get an API key:
   ```bash
   curl -s -X POST http://localhost:8080/api/v1/tenants \
     -H "Content-Type: application/json" \
     -d '{"name": "sdk-test", "email": "sdk@test.com"}'
   ```

3. Export the credentials:
   ```bash
   export WAAS_API_URL=http://localhost:8080
   export WAAS_API_KEY=<api-key-from-step-2>
   ```

4. Run SDK integration tests for your language (see above).

## Adding a New SDK

1. Create `sdk/<language>/` directory
2. Follow the structure of existing SDKs (README, examples, tests)
3. Implement the core client methods:
   - `createTenant` / `getTenant`
   - `createEndpoint` / `listEndpoints` / `updateEndpoint` / `deleteEndpoint`
   - `sendWebhook`
   - `getDelivery` / `listDeliveries`
4. Add integration tests that run against `WAAS_API_URL`
5. Update this file and `sdk/CONTRIBUTING.md`

## CI Validation

SDK tests are not currently part of the main CI pipeline. To validate locally:
```bash
# Quick validation for all SDKs
for dir in sdk/*/; do
  echo "=== Testing $(basename $dir) ==="
  cd "$dir" && make test 2>/dev/null || echo "  (no make test target)"
  cd ../..
done
```

> **TODO:** Add SDK CI jobs to `.github/workflows/ci.yml` to run SDK tests on PR.
