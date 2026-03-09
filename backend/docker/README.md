# Dockerfiles

This directory contains Dockerfiles for building WaaS service images.

## Overview

| Dockerfile | Purpose | Usage |
|------------|---------|-------|
| `Dockerfile.api` | API Service (development) | `docker build -f docker/Dockerfile.api -t waas-api .` |
| `Dockerfile.api.prod` | API Service (production, multi-stage) | `docker build -f docker/Dockerfile.api.prod -t waas-api:prod .` |
| `Dockerfile.delivery` | Delivery Engine (development) | `docker build -f docker/Dockerfile.delivery -t waas-delivery .` |
| `Dockerfile.delivery.prod` | Delivery Engine (production, multi-stage) | `docker build -f docker/Dockerfile.delivery.prod -t waas-delivery:prod .` |
| `Dockerfile.analytics` | Analytics Service (development) | `docker build -f docker/Dockerfile.analytics -t waas-analytics .` |
| `Dockerfile.analytics.prod` | Analytics Service (production, multi-stage) | `docker build -f docker/Dockerfile.analytics.prod -t waas-analytics:prod .` |
| `Dockerfile.quickstart` | All-in-one quickstart image | `docker build -f docker/Dockerfile.quickstart -t waas-quickstart .` |
| `Dockerfile.test` | Test runner image | `docker build -f docker/Dockerfile.test -t waas-test .` |

## Dev vs Prod Images

- **Dev images** (`Dockerfile.<service>`) include development tools and mount source code for hot-reload via [Air](https://github.com/air-verse/air).
- **Prod images** (`Dockerfile.<service>.prod`) use multi-stage builds to produce minimal images with only the compiled binary.

All build commands should be run from the `backend/` directory.

## See Also

- [Docker Compose files](../) — `docker-compose.yml`, `docker-compose.prod.yml`, `docker-compose.quickstart.yml`
- [Production Makefile](../Makefile.prod) — `make -f Makefile.prod help` for image build targets
