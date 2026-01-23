# --- General ---

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "cluster_name" {
  description = "Name of the GKE cluster"
  type        = string
  default     = "waas"
}

variable "environment" {
  description = "Deployment environment (production, staging, development)"
  type        = string
  default     = "production"
}

# --- Networking ---

variable "vpc_cidr" {
  description = "Primary CIDR range for the VPC subnet"
  type        = string
  default     = "10.0.0.0/20"
}

variable "pods_cidr" {
  description = "Secondary CIDR range for pods"
  type        = string
  default     = "10.16.0.0/14"
}

variable "services_cidr" {
  description = "Secondary CIDR range for services"
  type        = string
  default     = "10.20.0.0/20"
}

# --- GKE Node Pool ---

variable "node_machine_type" {
  description = "GCE machine type for GKE nodes"
  type        = string
  default     = "e2-standard-4"
}

variable "node_count" {
  description = "Desired number of GKE nodes per zone"
  type        = number
  default     = 1
}

variable "node_min" {
  description = "Minimum number of GKE nodes per zone"
  type        = number
  default     = 1
}

variable "node_max" {
  description = "Maximum number of GKE nodes per zone"
  type        = number
  default     = 5
}

# --- Cloud SQL PostgreSQL ---

variable "db_tier" {
  description = "Cloud SQL machine tier"
  type        = string
  default     = "db-custom-2-8192"
}

variable "db_ha" {
  description = "Enable high availability for Cloud SQL"
  type        = bool
  default     = true
}

variable "db_name" {
  description = "PostgreSQL database name"
  type        = string
  default     = "webhook_platform"
}

variable "db_username" {
  description = "PostgreSQL username"
  type        = string
  default     = "postgres"
  sensitive   = true
}

variable "db_password" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

# --- Memorystore Redis ---

variable "redis_tier" {
  description = "Memorystore Redis tier (BASIC or STANDARD_HA)"
  type        = string
  default     = "STANDARD_HA"
}

variable "redis_memory_size_gb" {
  description = "Memorystore Redis memory size in GB"
  type        = number
  default     = 2
}
