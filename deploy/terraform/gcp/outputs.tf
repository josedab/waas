# --- GKE ---

output "cluster_endpoint" {
  description = "GKE cluster endpoint"
  value       = google_container_cluster.main.endpoint
}

output "cluster_name" {
  description = "GKE cluster name"
  value       = google_container_cluster.main.name
}

output "cluster_ca_certificate" {
  description = "GKE cluster CA certificate (base64)"
  value       = google_container_cluster.main.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

# --- Cloud SQL ---

output "database_endpoint" {
  description = "Cloud SQL private IP"
  value       = google_sql_database_instance.postgresql.private_ip_address
}

output "database_connection_name" {
  description = "Cloud SQL connection name (for Cloud SQL Proxy)"
  value       = google_sql_database_instance.postgresql.connection_name
}

output "database_name" {
  description = "PostgreSQL database name"
  value       = google_sql_database.main.name
}

# --- Memorystore ---

output "redis_endpoint" {
  description = "Memorystore Redis host"
  value       = google_redis_instance.main.host
}

output "redis_port" {
  description = "Memorystore Redis port"
  value       = google_redis_instance.main.port
}

# --- VPC ---

output "vpc_id" {
  description = "VPC network ID"
  value       = google_compute_network.main.id
}

output "subnet_id" {
  description = "Subnet ID"
  value       = google_compute_subnetwork.main.id
}
