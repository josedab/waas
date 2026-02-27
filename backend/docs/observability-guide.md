# Observability Guide

Getting started with tracing, metrics, and dashboards for the WaaS platform.

## Quick Start

Start the full stack with the observability add-on:

```bash
cd backend
docker compose -f docker-compose.yml -f docker-compose.observability.yml up
```

## Service URLs

| Service | URL | Credentials |
|---------|-----|-------------|
| **Jaeger UI** (tracing) | [http://localhost:16686](http://localhost:16686) | None |
| **Prometheus** (metrics) | [http://localhost:9090](http://localhost:9090) | None |
| **Grafana** (dashboards) | [http://localhost:3001](http://localhost:3001) | `admin` / `admin` |

## Viewing Traces in Jaeger

1. Open [http://localhost:16686](http://localhost:16686)
2. Select a service from the **Service** dropdown (`waas-api`, `waas-delivery`, or `waas-analytics`)
3. Click **Find Traces** to view recent requests
4. Click a trace to see the full span breakdown including latency per operation

## Viewing Metrics in Prometheus

1. Open [http://localhost:9090](http://localhost:9090)
2. Use the **Expression** field to query metrics, e.g.:
   - `http_request_duration_seconds` — API request latency
   - `webhook_deliveries_total` — delivery attempt counts
3. Switch to the **Graph** tab for time-series visualization

## Using Grafana Dashboards

1. Open [http://localhost:3001](http://localhost:3001) and log in with `admin` / `admin`
2. Navigate to **Dashboards** in the sidebar
3. Pre-provisioned dashboards are loaded from `deploy/observability/grafana/provisioning/`

## Connecting Services

Each WaaS service exports telemetry when the following environment variables are set:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=waas-api
```

The `docker-compose.observability.yml` overlay configures these automatically for all services. For standalone use, set the variables in your `.env` file or export them before running a service.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint (gRPC, typically `:4317`) |
| `OTEL_SERVICE_NAME` | Service identifier shown in Jaeger and Grafana |
| `OTEL_SERVICE_VERSION` | Version tag for telemetry |
| `OTEL_ENVIRONMENT` | Environment tag (`development`, `production`) |
| `OTEL_INSECURE` | Use insecure connection to the collector (`true`/`false`) |

## Further Reading

- [Deployment Guide — Environment Variable Reference](deployment-guide.md#environment-variable-reference)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
