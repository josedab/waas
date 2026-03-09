# Kubernetes Manifests

Raw Kubernetes manifests for deploying WaaS without Helm. For a parameterized, production-ready deployment, see the [Helm chart](../../deploy/helm/waas/).

## Manifests

| File | Purpose |
|------|---------|
| `namespace.yaml` | Creates the `waas` namespace |
| `configmap.yaml` | Application configuration |
| `secrets.yaml` | Sensitive values (database credentials, API keys) |
| `postgres.yaml` | PostgreSQL StatefulSet and Service |
| `redis.yaml` | Redis Deployment and Service |
| `api-service.yaml` | API Service Deployment and Service (port 8080) |
| `delivery-engine.yaml` | Delivery Engine Deployment |
| `analytics-service.yaml` | Analytics Service Deployment and Service (port 8082) |
| `ingress.yaml` | Ingress rules for external access |
| `migration-job.yaml` | One-time Job to run database migrations |
| `backup-cronjob.yaml` | Scheduled CronJob for database backups |
| `monitoring.yaml` | Prometheus ServiceMonitor and alerting rules |
| `logging.yaml` | Logging configuration |

## Usage

```bash
# Apply all manifests
kubectl apply -f backend/k8s/

# Apply in order (recommended for first deployment)
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml -f secrets.yaml
kubectl apply -f postgres.yaml -f redis.yaml
kubectl apply -f migration-job.yaml
kubectl apply -f api-service.yaml -f delivery-engine.yaml -f analytics-service.yaml
kubectl apply -f ingress.yaml -f monitoring.yaml -f logging.yaml -f backup-cronjob.yaml
```

## See Also

- [Helm Chart](../../deploy/helm/waas/) — Parameterized deployment with `values.yaml`
- [Deployment Guide](../docs/deployment-guide.md) — Full production deployment walkthrough
- [Terraform Modules](../../deploy/terraform/) — Cloud infrastructure provisioning (AWS/GCP)
