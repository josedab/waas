# WaaS Deployment Guide

This directory contains everything needed to deploy WaaS (Webhooks as a Service) to production Kubernetes clusters on AWS or GCP.

## Directory Structure

```
deploy/
├── helm/waas/           # Helm chart for Kubernetes deployment
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/       # Parameterized K8s manifests
├── terraform/
│   ├── aws/             # AWS infrastructure (EKS + RDS + ElastiCache)
│   └── gcp/             # GCP infrastructure (GKE + Cloud SQL + Memorystore)
└── README.md            # This file
```

---

## Quick Start with Helm

### Prerequisites

- [Helm](https://helm.sh/) v3+
- `kubectl` configured with cluster access
- A running Kubernetes cluster (v1.25+)

### Install (development)

```bash
helm install waas ./deploy/helm/waas \
  --namespace waas --create-namespace \
  --set global.environment=development \
  --set secrets.jwtSecret=dev-jwt-secret \
  --set secrets.encryptionKey=dev-encryption-key-32bytes \
  --set secrets.webhookSigningSecret=dev-signing-secret \
  --set postgresql.password=devpassword \
  --set ingress.enabled=false
```

### Install (production with external databases)

```bash
helm install waas ./deploy/helm/waas \
  --namespace waas --create-namespace \
  --set postgresql.enabled=false \
  --set postgresql.host=your-rds-endpoint.amazonaws.com \
  --set postgresql.password=<DB_PASSWORD> \
  --set redis.enabled=false \
  --set redis.host=your-redis-endpoint.amazonaws.com \
  --set ingress.host=webhooks.yourdomain.com \
  --set secrets.jwtSecret=<JWT_SECRET> \
  --set secrets.encryptionKey=<ENCRYPTION_KEY> \
  --set secrets.webhookSigningSecret=<SIGNING_SECRET>
```

### Upgrade

```bash
helm upgrade waas ./deploy/helm/waas --namespace waas --reuse-values
```

### Uninstall

```bash
helm uninstall waas --namespace waas
```

---

## AWS Deployment with Terraform

Provisions an EKS cluster with managed node groups, Aurora PostgreSQL, and ElastiCache Redis.

### Prerequisites

- [Terraform](https://www.terraform.io/) v1.0+
- AWS CLI configured with appropriate credentials
- Permissions for EKS, RDS, ElastiCache, VPC, IAM

### Steps

```bash
cd deploy/terraform/aws

# Initialize
terraform init

# Review the plan
terraform plan -var="db_password=YOUR_SECURE_PASSWORD"

# Apply
terraform apply -var="db_password=YOUR_SECURE_PASSWORD"

# Configure kubectl
aws eks update-kubeconfig --name waas --region us-east-1

# Deploy WaaS via Helm
helm install waas ../../helm/waas \
  --namespace waas --create-namespace \
  --set postgresql.enabled=false \
  --set postgresql.host=$(terraform output -raw database_endpoint) \
  --set postgresql.password=YOUR_SECURE_PASSWORD \
  --set redis.enabled=false \
  --set redis.host=$(terraform output -raw redis_endpoint) \
  --set secrets.jwtSecret=<JWT_SECRET> \
  --set secrets.encryptionKey=<ENCRYPTION_KEY> \
  --set secrets.webhookSigningSecret=<SIGNING_SECRET>
```

### Key Variables

| Variable | Default | Description |
|---|---|---|
| `region` | `us-east-1` | AWS region |
| `cluster_name` | `waas` | EKS cluster name |
| `node_instance_type` | `t3.large` | EC2 instance type |
| `node_count` | `3` | Desired node count |
| `node_min` / `node_max` | `2` / `10` | Autoscaling bounds |
| `db_instance_class` | `db.r6g.large` | RDS instance class |
| `db_multi_az` | `true` | Multi-AZ for RDS |
| `redis_node_type` | `cache.r6g.large` | ElastiCache node type |
| `redis_num_nodes` | `2` | Redis cluster nodes |

---

## GCP Deployment with Terraform

Provisions a GKE cluster, Cloud SQL PostgreSQL, and Memorystore Redis.

### Prerequisites

- [Terraform](https://www.terraform.io/) v1.0+
- `gcloud` CLI authenticated
- A GCP project with billing enabled
- APIs enabled: `container.googleapis.com`, `sqladmin.googleapis.com`, `redis.googleapis.com`, `servicenetworking.googleapis.com`

### Steps

```bash
cd deploy/terraform/gcp

# Initialize
terraform init

# Review the plan
terraform plan \
  -var="project_id=YOUR_PROJECT_ID" \
  -var="db_password=YOUR_SECURE_PASSWORD"

# Apply
terraform apply \
  -var="project_id=YOUR_PROJECT_ID" \
  -var="db_password=YOUR_SECURE_PASSWORD"

# Configure kubectl
gcloud container clusters get-credentials waas --region us-central1

# Deploy WaaS via Helm
helm install waas ../../helm/waas \
  --namespace waas --create-namespace \
  --set postgresql.enabled=false \
  --set postgresql.host=$(terraform output -raw database_endpoint) \
  --set postgresql.password=YOUR_SECURE_PASSWORD \
  --set redis.enabled=false \
  --set redis.host=$(terraform output -raw redis_endpoint) \
  --set secrets.jwtSecret=<JWT_SECRET> \
  --set secrets.encryptionKey=<ENCRYPTION_KEY> \
  --set secrets.webhookSigningSecret=<SIGNING_SECRET>
```

### Key Variables

| Variable | Default | Description |
|---|---|---|
| `project_id` | — | GCP project ID (required) |
| `region` | `us-central1` | GCP region |
| `cluster_name` | `waas` | GKE cluster name |
| `node_machine_type` | `e2-standard-4` | GCE machine type |
| `node_count` | `1` | Nodes per zone |
| `node_min` / `node_max` | `1` / `5` | Autoscaling bounds per zone |
| `db_tier` | `db-custom-2-8192` | Cloud SQL tier |
| `db_ha` | `true` | High availability |
| `redis_tier` | `STANDARD_HA` | Memorystore tier |
| `redis_memory_size_gb` | `2` | Redis memory in GB |

---

## Configuration Reference

### Helm Values

#### Global

| Value | Default | Description |
|---|---|---|
| `global.environment` | `production` | Environment name |
| `global.imagePullPolicy` | `IfNotPresent` | Image pull policy |
| `global.imageTag` | `latest` | Image tag for all services |

#### Services (api, delivery, analytics)

| Value | Description |
|---|---|
| `<svc>.replicaCount` | Base replica count (ignored when HPA enabled) |
| `<svc>.image` | Container image name |
| `<svc>.port` | Container port |
| `<svc>.resources.requests/limits` | CPU/memory resources |
| `<svc>.autoscaling.enabled` | Enable HPA |
| `<svc>.autoscaling.minReplicas` | HPA minimum replicas |
| `<svc>.autoscaling.maxReplicas` | HPA maximum replicas |
| `<svc>.autoscaling.targetCPUUtilization` | CPU target % |

#### Database & Cache

| Value | Description |
|---|---|
| `postgresql.enabled` | Deploy in-cluster PostgreSQL (set `false` for managed) |
| `postgresql.host` | External PostgreSQL host |
| `postgresql.existingSecret` | Use existing K8s secret |
| `redis.enabled` | Deploy in-cluster Redis (set `false` for managed) |
| `redis.host` | External Redis host |
| `redis.existingSecret` | Use existing K8s secret |

---

## Production Best Practices

### TLS / Certificates

- Use [cert-manager](https://cert-manager.io/) with Let's Encrypt for automatic TLS
- The ingress template includes `cert-manager.io/cluster-issuer: letsencrypt-prod` by default
- Install cert-manager before deploying:
  ```bash
  helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set crds.enabled=true
  ```

### Secrets Management

- **Never** pass secrets via `--set` in CI/CD — use a values file or external secrets
- Consider [External Secrets Operator](https://external-secrets.io/) for AWS Secrets Manager / GCP Secret Manager integration
- Use `existingSecret` values to reference pre-created Kubernetes secrets

### Backups

- **AWS**: Aurora automatic backups (7-day retention configured in Terraform)
- **GCP**: Cloud SQL point-in-time recovery enabled
- Verify backups regularly with restore tests

### Monitoring

- Prometheus scraping is enabled on all service pods via annotations
- Deploy the Prometheus stack:
  ```bash
  helm install prometheus prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace
  ```
- Alert rules for error rates, latency, queue depth, and pod crashes are defined in the existing `monitoring.yaml` manifests

### Resource Tuning

- Start with the default resource requests/limits and adjust based on observed usage
- Use Vertical Pod Autoscaler (VPA) in recommendation mode to right-size pods
- Monitor HPA behavior and adjust `targetCPUUtilization` thresholds as needed

### Network Security

- Enable network policies (the existing k8s manifests include a `NetworkPolicy`)
- Use private endpoints for databases (configured in both Terraform modules)
- Restrict ingress to known IP ranges in production if applicable
