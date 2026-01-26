# Quick Contributing Guide

> Get your first PR merged in under 30 minutes. For the full guide, see [CONTRIBUTING.md](CONTRIBUTING.md).

## TL;DR

```bash
# 1. Fork & clone
git clone <your-fork-url> && cd waas/backend

# 2. Setup
make dev-setup

# 3. Branch
git checkout -b fix/your-change

# 4. Make changes, then verify
make build-check && make test

# 5. Commit & push
git add -A && git commit -m "fix: brief description"
git push origin fix/your-change

# 6. Open a PR against main
```

## What You Need

- **Go 1.24+** — `go version`
- **Docker** — for `make dev-setup` (PostgreSQL + Redis)
- **~10 min** — to clone, setup, and run tests

## Branch Naming

| Type | Pattern | Example |
|------|---------|---------|
| Bug fix | `fix/short-description` | `fix/retry-counter-overflow` |
| Feature | `feat/short-description` | `feat/batch-endpoint-create` |
| Docs | `docs/short-description` | `docs/update-api-examples` |
| Refactor | `refactor/short-description` | `refactor/delivery-engine` |

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
fix: correct retry backoff calculation
feat: add batch endpoint creation API
docs: update deployment guide for Helm v3
test: add integration tests for delivery engine
refactor: extract queue connection logic
```

## Code Checklist

Before opening your PR:

- [ ] `make build-check` passes (compiles all packages)
- [ ] `make test` passes (core tests)
- [ ] `make vet` passes (static analysis)
- [ ] New code follows existing patterns (see [ARCHITECTURE.md](ARCHITECTURE.md))
- [ ] Error handling uses `pkg/errors` helpers where applicable

## Where to Find Things

| I want to... | Look in... |
|--------------|-----------|
| Fix an API endpoint | `internal/api/` or `pkg/<feature>/handlers.go` |
| Fix a data model | `pkg/models/` |
| Fix database queries | `pkg/repository/` or `pkg/<feature>/repository.go` |
| Fix delivery logic | `pkg/delivery/` |
| Add a migration | `migrations/` (use `migrate create`) |
| Fix auth | `pkg/auth/` |
| Fix error responses | `pkg/errors/` |

## Getting Help

- Check [docs/QUICK_REFERENCE.md](docs/QUICK_REFERENCE.md) for common commands
- Read [ARCHITECTURE.md](ARCHITECTURE.md) for how packages are organized
- Open a draft PR early if you want feedback on your approach
