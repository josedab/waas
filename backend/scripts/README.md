# Scripts

Utility scripts for building, deploying, and seeding the WaaS platform.

## Script Reference

| Script | Purpose | Usage |
|--------|---------|-------|
| `build-images.sh` | Build and push production Docker images | `./scripts/build-images.sh` — uses `DOCKER_REGISTRY` and `PROJECT_NAME` env vars |
| `deploy.sh` | Production deployment verification and rollback | `./scripts/deploy.sh` — targets the `webhook-platform` k8s namespace |
| `generate-docs.sh` | Generate and validate OpenAPI documentation | `./scripts/generate-docs.sh` — runs `swag` and validates output |
| `pre-commit` | Git pre-commit hook (blocks secrets, runs lints) | Install with `make install-hooks` |
| `seed-dev-data.sh` | Seed development data via the running API | `./scripts/seed-dev-data.sh [BASE_URL]` — defaults to `http://localhost:8080` |
| `seed_quickstart.sh` | Seed quickstart demo data directly into the database | `./scripts/seed_quickstart.sh` — uses `DATABASE_URL` env var |

## Notes

- All scripts use `set -e` (or `set -euo pipefail`) to fail fast on errors.
- Run scripts from the `backend/` directory unless otherwise noted.
- The `pre-commit` hook is installed automatically by `make install-hooks` and does not need to be run manually.
