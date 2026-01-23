# --- EKS ---

output "cluster_endpoint" {
  description = "EKS cluster API endpoint"
  value       = aws_eks_cluster.main.endpoint
}

output "cluster_name" {
  description = "EKS cluster name"
  value       = aws_eks_cluster.main.name
}

output "cluster_certificate_authority" {
  description = "EKS cluster CA certificate (base64)"
  value       = aws_eks_cluster.main.certificate_authority[0].data
  sensitive   = true
}

# --- RDS ---

output "database_endpoint" {
  description = "RDS cluster endpoint"
  value       = aws_rds_cluster.postgresql.endpoint
}

output "database_reader_endpoint" {
  description = "RDS cluster reader endpoint"
  value       = aws_rds_cluster.postgresql.reader_endpoint
}

output "database_name" {
  description = "RDS database name"
  value       = aws_rds_cluster.postgresql.database_name
}

# --- ElastiCache ---

output "redis_endpoint" {
  description = "ElastiCache Redis primary endpoint"
  value       = aws_elasticache_replication_group.redis.primary_endpoint_address
}

# --- VPC ---

output "vpc_id" {
  description = "VPC ID"
  value       = aws_vpc.main.id
}

output "public_subnet_ids" {
  description = "Public subnet IDs"
  value       = aws_subnet.public[*].id
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = aws_subnet.private[*].id
}
