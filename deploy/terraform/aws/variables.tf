# --- General ---

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "waas"
}

variable "environment" {
  description = "Deployment environment (production, staging, development)"
  type        = string
  default     = "production"
}

variable "kubernetes_version" {
  description = "Kubernetes version for EKS"
  type        = string
  default     = "1.29"
}

# --- Networking ---

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "List of availability zones (auto-detected if empty)"
  type        = list(string)
  default     = []
}

# --- EKS Node Group ---

variable "node_instance_type" {
  description = "EC2 instance type for EKS nodes"
  type        = string
  default     = "t3.large"
}

variable "node_count" {
  description = "Desired number of EKS nodes"
  type        = number
  default     = 3
}

variable "node_min" {
  description = "Minimum number of EKS nodes"
  type        = number
  default     = 2
}

variable "node_max" {
  description = "Maximum number of EKS nodes"
  type        = number
  default     = 10
}

# --- RDS PostgreSQL ---

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.r6g.large"
}

variable "db_storage" {
  description = "Allocated storage in GB for RDS"
  type        = number
  default     = 50
}

variable "db_multi_az" {
  description = "Enable Multi-AZ for RDS"
  type        = bool
  default     = true
}

variable "db_name" {
  description = "PostgreSQL database name"
  type        = string
  default     = "webhook_platform"
}

variable "db_username" {
  description = "PostgreSQL master username"
  type        = string
  default     = "postgres"
  sensitive   = true
}

variable "db_password" {
  description = "PostgreSQL master password"
  type        = string
  sensitive   = true
}

# --- ElastiCache Redis ---

variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.r6g.large"
}

variable "redis_num_nodes" {
  description = "Number of Redis cache nodes"
  type        = number
  default     = 2
}
