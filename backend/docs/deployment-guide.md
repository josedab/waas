# Production Deployment Guide

This guide covers the production deployment of the Webhook Service Platform using Kubernetes with security hardening, monitoring, and automated backup strategies.

## Prerequisites

### Infrastructure Requirements

- Kubernetes cluster (v1.24+) with RBAC enabled
- Container registry (Docker Hub, ECR, GCR, etc.)
- Persistent storage with fast SSD storage class
- Load balancer with SSL termination support
- Monitoring stack (Prometheus, Grafana)
- Logging stack (Elasticsearch, Kibana)

### Tools Required

- `kubectl` (v1.24+)
- `docker` (v20.10+)
- `helm` (v3.8+) - optional but recommended
- `trivy` - for vulnerability scanning

### Access Requirements

- Kubernetes cluster admin access
- Container registry push permissions
- DNS management for domain configuration

## Security Hardening Features

### Container Security

- **Distroless base images**: Minimal attack surface with no shell or package manager
- **Non-root user**: All containers run as non-root user (UID 65532)
- **Read-only root filesystem**: Prevents runtime modifications
- **Security contexts**: Drops all capabilities, prevents privilege escalation
- **Resource limits**: CPU and memory limits to prevent resource exhaustion

### Network Security

- **Network policies**: Restricts pod-to-pod communication
- **TLS encryption**: All external traffic encrypted with Let's Encrypt certificates
- **Service mesh ready**: Compatible with Istio for additional security layers

### Data Security

- **Secrets management**: Kubernetes secrets for sensitive data
- **Encryption at rest**: Database and Redis data encrypted
- **Audit logging**: All administrative operations logged

## Deployment Process

### Step 1: Prepare Environment

1. **Set environment variables**:
```bash
export DOCKER_REGISTRY="your-registry.com"
export PROJECT_NAME="webhook-platform"
export VERSION="v1.0.0"
export KUBECONFIG="/path/to/your/kubeconfig"
```

2. **Verify cluster access**:
```bash
kubectl cluster-info
kubectl get nodes
```

### Step 2: Build and Push Images

1. **Build production images**:
```bash
cd backend
./scripts/build-images.sh build
```

2. **Scan for vulnerabilities and push**:
```bash
./scripts/build-images.sh push
```

This will:
- Build hardened production images
- Scan for security vulnerabilities
- Push to container registry
- Generate image manifest

### Step 3: Deploy Infrastructure

1. **Deploy the platform**:
```bash
cd backend
./scripts/deploy.sh deploy
```

This automated script will:
- Verify prerequisites
- Deploy namespace and RBAC
- Deploy PostgreSQL and Redis
- Run database migrations
- Deploy application services
- Verify deployment health
- Deploy monitoring and logging

### Step 4: Configure DNS and SSL

1. **Update DNS records**:
```
api.webhook-platform.com     -> Load Balancer IP
analytics.webhook-platform.com -> Load Balancer IP
```

2. **Verify SSL certificates**:
```bash
kubectl get certificate -n webhook-platform
```

## Scaling Configuration

### Horizontal Pod Autoscaling (HPA)

The platform includes HPA configurations for automatic scaling:

#### API Service
- **Min replicas**: 3
- **Max replicas**: 20
- **CPU target**: 70%
- **Memory target**: 80%

#### Delivery Engine
- **Min replicas**: 5
- **Max replicas**: 50
- **CPU target**: 70%
- **Memory target**: 80%
- **Custom metric**: Queue depth (target: 100 messages)

#### Analytics Service
- **Min replicas**: 2
- **Max replicas**: 10
- **CPU target**: 70%
- **Memory target**: 80%

### Vertical Pod Autoscaling (VPA)

For VPA support, install the VPA controller and apply VPA resources:

```bash
# VPA requires the Vertical Pod Autoscaler controller to be installed in your cluster.
# See: https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler
# Create a VPA resource for each service, for example:
cat <<EOF | kubectl apply -f -
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: waas-api-vpa
  namespace: webhook-platform
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-service
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: "*"
      minAllowed:
        cpu: "100m"
        memory: "128Mi"
      maxAllowed:
        cpu: "2"
        memory: "2Gi"
EOF
```

## Monitoring and Alerting

### Metrics Collection

- **Prometheus**: Scrapes metrics from all services
- **Custom metrics**: Business metrics (delivery rates, queue depth)
- **Infrastructure metrics**: CPU, memory, disk, network

### Alert Rules

Critical alerts configured:
- High error rate (>10% for 5 minutes)
- High latency (95th percentile >1 second)
- Queue backlog (>1000 messages for 2 minutes)
- Database connection issues
- Pod crash loops
- High memory usage (>90%)

### Dashboards

Grafana dashboards available for:
- Application performance metrics
- Infrastructure monitoring
- Business KPIs
- Error tracking and debugging

## Backup and Recovery

### Automated Backups

#### PostgreSQL Backup
- **Schedule**: Daily at 2:00 AM UTC
- **Retention**: 7 days
- **Format**: Compressed SQL dump
- **Storage**: Persistent volume (100GB)

#### Redis Backup
- **Schedule**: Daily at 3:00 AM UTC
- **Retention**: 7 days
- **Format**: RDB snapshot
- **Storage**: Same persistent volume as PostgreSQL

### Recovery Procedures

#### Database Recovery
```bash
# List available backups
kubectl exec -n webhook-platform deployment/postgres -- ls -la /backups/

# Restore from backup
kubectl exec -n webhook-platform deployment/postgres -- \
  psql $DATABASE_URL < /backups/backup-YYYYMMDD-HHMMSS.sql
```

#### Redis Recovery
```bash
# Stop Redis
kubectl scale deployment redis --replicas=0 -n webhook-platform

# Restore RDB file
kubectl cp backup-file.rdb webhook-platform/redis-pod:/data/dump.rdb

# Start Redis
kubectl scale deployment redis --replicas=1 -n webhook-platform
```

## Deployment Verification

### Automated Verification

The deployment script includes automated verification:

1. **Pod readiness**: All pods must be ready
2. **Health checks**: HTTP health endpoints must respond
3. **Database connectivity**: Services can connect to databases
4. **Queue connectivity**: Message queue is accessible

### Manual Verification

```bash
# Check deployment status
./scripts/deploy.sh verify

# Run health checks
./scripts/deploy.sh health

# Check specific service
kubectl get pods -n webhook-platform
kubectl logs -f deployment/api-service -n webhook-platform
```

## Rollback Procedures

### Automatic Rollback

If deployment verification fails, automatic rollback is triggered:

```bash
# Automatic rollback on failure
./scripts/deploy.sh deploy  # Will rollback if verification fails
```

### Manual Rollback

```bash
# Manual rollback to previous version
./scripts/deploy.sh rollback

# Rollback specific service
kubectl rollout undo deployment/api-service -n webhook-platform
```

### Rollback Verification

After rollback:
1. Verify all pods are running
2. Run health checks
3. Check application functionality
4. Monitor error rates and performance

## Troubleshooting

### Common Issues

#### Pod Startup Issues
```bash
# Check pod status
kubectl describe pod <pod-name> -n webhook-platform

# Check logs
kubectl logs <pod-name> -n webhook-platform --previous
```

#### Database Connection Issues
```bash
# Test database connectivity
kubectl run db-test --image=postgres:15-alpine --rm -i --restart=Never -n webhook-platform -- \
  psql $DATABASE_URL -c "SELECT 1"
```

#### Performance Issues
```bash
# Check resource usage
kubectl top pods -n webhook-platform
kubectl top nodes

# Check HPA status
kubectl get hpa -n webhook-platform
```

### Emergency Procedures

#### Scale Down Services
```bash
# Emergency scale down
kubectl scale deployment api-service --replicas=1 -n webhook-platform
kubectl scale deployment delivery-engine --replicas=1 -n webhook-platform
```

#### Database Maintenance Mode
```bash
# Put system in maintenance mode
kubectl patch ingress webhook-platform-ingress -n webhook-platform \
  --type='json' -p='[{"op": "add", "path": "/metadata/annotations/nginx.ingress.kubernetes.io~1default-backend", "value": "maintenance-page"}]'
```

## Security Considerations

### Regular Security Tasks

1. **Update base images**: Rebuild images monthly with latest security patches
2. **Rotate secrets**: Rotate database passwords and API keys quarterly
3. **Review access**: Audit RBAC permissions monthly
4. **Scan vulnerabilities**: Run security scans weekly

### Security Monitoring

- Monitor failed authentication attempts
- Track unusual API usage patterns
- Alert on privilege escalation attempts
- Monitor network policy violations

## Performance Optimization

### Database Optimization

- Connection pooling configured via environment variables (see below)
- Query optimization with indexes
- Regular VACUUM and ANALYZE operations
- Monitoring slow queries

#### Connection Pool Tuning

The API and delivery engine maintain a PostgreSQL connection pool. Tune these
settings based on your workload:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_MAX_CONNS` | 30 | Maximum open connections per service instance |
| `DB_MIN_CONNS` | 5 | Minimum idle connections kept warm |
| `DB_CONN_MAX_LIFETIME` | `1h` | Maximum time a connection is reused |
| `DB_CONN_MAX_IDLE_TIME` | `30m` | Close idle connections after this duration |

**Sizing guidance:**

```
max_conns = (num_service_instances × DB_MAX_CONNS) ≤ PostgreSQL max_connections
```

For a 3-replica API deployment with the default 30, you need at least 90
server-side connections. Managed databases (RDS, Cloud SQL) typically default to
100–500; adjust accordingly.

**Monitoring query:**

```sql
SELECT count(*) AS active FROM pg_stat_activity WHERE state = 'active';
SELECT count(*) AS idle   FROM pg_stat_activity WHERE state = 'idle';
```

### Application Optimization

- HTTP/2 enabled for better performance
- Gzip compression for API responses
- Connection keep-alive for webhook delivery
- Batch processing for analytics

### Infrastructure Optimization

- Node affinity for database pods
- Pod disruption budgets for high availability
- Resource requests and limits tuned
- Network policies optimized for performance

## Maintenance Windows

### Planned Maintenance

- **Schedule**: Monthly, first Sunday 2:00-4:00 AM UTC
- **Duration**: Maximum 2 hours
- **Notification**: 48 hours advance notice
- **Rollback plan**: Always prepared

### Maintenance Checklist

1. Notify users of maintenance window
2. Create database backup
3. Scale down non-critical services
4. Apply updates with rolling deployment
5. Verify functionality
6. Scale services back up
7. Monitor for issues

---

## Development → Staging → Production Path

This section covers how to promote WaaS from local development to production.

### Environment Overview

| Environment | Purpose | Infrastructure | Database |
|-------------|---------|---------------|----------|
| **Local dev** | Feature development | `docker-compose up` | Local PostgreSQL + Redis |
| **CI** | Automated testing | GitHub Actions | Ephemeral test containers |
| **Staging** | Pre-release validation | Single VM or small K8s cluster | Managed PostgreSQL |
| **Production** | Live traffic | Kubernetes cluster (see above) | Managed PostgreSQL + Redis |

### Step 1: Local Development

```bash
cd backend
make dev-setup          # Creates .env, starts Docker, runs migrations
make run-all            # Runs API + delivery + analytics

# Optional: Add observability
docker-compose -f docker-compose.yml -f docker-compose.observability.yml up -d
```

Validate with:
```bash
make test               # Core tests
make build-check        # Compile all packages
make lint               # Code quality
```

### Step 2: CI Validation

Push to a branch to trigger CI (`.github/workflows/ci.yml`):
- Runs `go build ./...` and `go test ./...`
- Integration tests use ephemeral Docker containers
- Lint and vet checks run in parallel

### Step 3: Staging Deployment (Simple)

For teams that don't need Kubernetes yet, deploy to a single VM:

```bash
# On the staging server:
# 1. Install Go 1.24+, PostgreSQL 15+, Redis 7+

# 2. Build the binaries
cd backend
go build -o bin/api ./cmd/api-service/
go build -o bin/delivery ./cmd/delivery-engine/
go build -o bin/analytics ./cmd/analytics-service/

# 3. Set environment variables
export DATABASE_URL="postgres://user:pass@localhost:5432/waas_staging?sslmode=require"
export REDIS_URL="localhost:6379"
export JWT_SECRET="<staging-secret>"
export API_PORT=8080

# 4. Run migrations
make migrate-up

# 5. Start services (use systemd or supervisor in practice)
./bin/api &
./bin/delivery &
./bin/analytics &
```

For Docker-based staging:
```bash
# Build and push images
docker build -t your-registry/waas-api:staging .
docker push your-registry/waas-api:staging

# Deploy with docker-compose on staging server
DATABASE_URL=... REDIS_URL=... docker-compose up -d
```

### Step 4: Staging Validation Checklist

Before promoting to production:

- [ ] All CI tests pass on the release branch
- [ ] `make smoke-test` passes against staging
- [ ] Webhook delivery round-trip works (create endpoint → send event → confirm delivery)
- [ ] Dashboard loads and shows correct data
- [ ] Rate limiting and quotas enforce correctly
- [ ] Migrations ran cleanly (`make migrate-status`)
- [ ] No error spikes in logs for 1 hour

### Step 5: Production Deployment

Follow the Kubernetes deployment sections above, or use the Helm chart (located at the **repository root** under `deploy/helm/waas`):

```bash
# Using Helm (recommended for production)
# Run from the repository root directory
helm upgrade --install waas ./deploy/helm/waas \
  --namespace waas \
  --set database.url="$PROD_DATABASE_URL" \
  --set redis.url="$PROD_REDIS_URL" \
  --set api.replicas=3 \
  --set delivery.replicas=3

# Verify
kubectl -n waas rollout status deployment/waas-api
kubectl -n waas logs -l app=waas-api --tail=50
```

### Environment Variable Reference

#### Core (Required)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `REDIS_URL` | Yes | `localhost:6379` | Redis connection string |
| `JWT_SECRET` | Yes | — | JWT signing key (use different per env) |

#### Application

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `API_PORT` | No | `8080` | API service HTTP listen port |
| `ANALYTICS_PORT` | No | `8082` | Analytics service HTTP listen port |
| `ENVIRONMENT` | No | `development` | `development` or `production` |
| `LOG_LEVEL` | No | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | No | `text` | Log output format: `json` or `text` |

#### Security & Access Control

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ADMIN_TENANT_IDS` | No | — | Comma-separated tenant IDs with admin privileges |
| `CORS_ALLOWED_ORIGINS` | No | — | Comma-separated allowed CORS origins |
| `ALLOW_INSECURE_TLS` | No | `false` | Skip TLS certificate verification for webhook delivery |
| `WEBSOCKET_ALLOWED_ORIGINS` | No | — | Comma-separated allowed WebSocket origins |

#### AI / LLM Integration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | No | — | OpenAI API key for AI-powered features |
| `ANTHROPIC_API_KEY` | No | — | Anthropic API key for AI-powered features |

#### Billing

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `STRIPE_API_KEY` | No | — | Stripe secret key for billing integration |

#### Observability (OpenTelemetry)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | — | OTLP collector URL for tracing |
| `OTEL_SERVICE_NAME` | No | — | Service name for tracing |
| `OTEL_SERVICE_VERSION` | No | — | Service version tag for telemetry |
| `OTEL_ENVIRONMENT` | No | — | Environment tag for telemetry |
| `OTEL_INSECURE` | No | — | Use insecure connection to OTLP collector |

#### Platform Webhooks

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `WEBHOOK_PLATFORM_API_KEY` | No | — | API key for platform-level webhook notifications |

#### Test Environment

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TEST_DATABASE_URL` | No | — | PostgreSQL connection string for test database |
| `TEST_REDIS_URL` | No | — | Redis connection string for test database (use a separate DB index) |

#### Database Connection Pool

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_MAX_CONNS` | No | 30 | Maximum open connections per instance |
| `DB_MIN_CONNS` | No | 5 | Minimum idle connections kept warm |
| `DB_CONN_MAX_LIFETIME` | No | `1h` | Maximum time a connection is reused (Go duration) |
| `DB_CONN_MAX_IDLE_TIME` | No | `30m` | Close idle connections after this duration (Go duration) |

#### Delivery Engine

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DELIVERY_HEALTH_PORT` | No | 8081 | Health check HTTP port for the delivery engine |


This deployment guide ensures a secure, scalable, and maintainable production deployment of the Webhook Service Platform.