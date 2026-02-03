# Contributing to WaaS

Thank you for contributing to the Webhook-as-a-Service platform.

## Getting Started

1. Fork and clone the repository
2. Install prerequisites (see [README.md](README.md#prerequisites))
3. Run `make validate-setup` to verify your environment
4. Run `make dev-setup` to start local infrastructure
5. Run `make test` to confirm everything works

> **VS Code tip:** Open the repo root in VS Code — curated settings in `.vscode/` and `backend/.vscode/` will be picked up automatically. If gopls has trouble finding packages, open `backend/` as the workspace root instead.

## Development Workflow

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/your-feature
   ```
2. Make your changes
3. Run quality checks:
   ```bash
   make check    # runs fmt, vet, lint, and test
   ```
4. Commit with a conventional commit message (see below)
5. Push and open a pull request against `main`

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(delivery): add exponential jitter to retry backoff
fix(auth): validate API key length before database lookup
docs(readme): add golang-migrate install instructions
test(queue): add consumer timeout edge case
refactor(repository): extract pagination helper
ci(github): add coverage threshold to workflow
```

Format: `type(scope): description`

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`, `perf`

## Code Style

- Go code follows `gofmt` formatting (enforced by CI)
- Use `golangci-lint` for static analysis (config in `.golangci.yml`)
- Follow existing naming patterns: `Service` structs with `NewService()` constructors
- Errors should be wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Pass `context.Context` as the first parameter in service/repository methods

## Project Layout

| Directory | Purpose |
|-----------|---------|
| `cmd/` | Service entry points (thin - just wiring) |
| `internal/` | Private service implementations |
| `pkg/` | Shared packages (any service can import) |
| `migrations/` | PostgreSQL schema migrations |
| `test/` | Integration, e2e, performance, chaos tests |
| `sdk/` | Client SDKs (see [SDK Contributing](sdk/CONTRIBUTING.md)) |
| `web/dashboard/` | React frontend (see [Dashboard README](web/dashboard/README.md)) |

## Pull Request Guidelines

- Keep PRs focused - one feature or fix per PR
- Include tests for new functionality
- Update documentation if you change public behavior
- All CI checks must pass before merge
- PRs require at least one review approval

## Testing

```bash
make test                            # Quick core tests (internal + key packages, < 1 min)
make test-all                        # All tests including enterprise packages
make -f Makefile.test test-unit      # Unit tests with coverage
make -f Makefile.test test-integration  # Integration tests (needs Docker)
make -f Makefile.test help           # All available test commands
```

## SDK Development

See [sdk/CONTRIBUTING.md](sdk/CONTRIBUTING.md) for guidelines on developing and maintaining client SDKs.

## Questions?

- Open a GitHub issue for bugs or feature requests
- Use GitHub Discussions for questions
