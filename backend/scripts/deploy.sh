#!/bin/bash

# Production Deployment Script for Webhook Platform
# This script handles deployment verification and rollback procedures

set -euo pipefail

# Configuration
NAMESPACE="webhook-platform"
TIMEOUT="600s"
ROLLBACK_TIMEOUT="300s"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we can connect to the cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if namespace exists
    if ! kubectl get namespace $NAMESPACE &> /dev/null; then
        log_error "Namespace $NAMESPACE does not exist"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Deploy infrastructure components
deploy_infrastructure() {
    log_info "Deploying infrastructure components..."
    
    # Apply namespace and RBAC
    kubectl apply -f k8s/namespace.yaml
    kubectl apply -f k8s/secrets.yaml
    kubectl apply -f k8s/configmap.yaml
    
    # Deploy databases
    kubectl apply -f k8s/postgres.yaml
    kubectl apply -f k8s/redis.yaml
    
    # Wait for databases to be ready
    log_info "Waiting for databases to be ready..."
    kubectl wait --for=condition=ready pod -l app=postgres -n $NAMESPACE --timeout=$TIMEOUT
    kubectl wait --for=condition=ready pod -l app=redis -n $NAMESPACE --timeout=$TIMEOUT
    
    log_info "Infrastructure deployment completed"
}

# Run database migrations
run_migrations() {
    log_info "Running database migrations..."
    
    # Create migration job
    kubectl apply -f k8s/migration-job.yaml
    
    # Wait for migration to complete
    kubectl wait --for=condition=complete job/db-migration -n $NAMESPACE --timeout=$TIMEOUT
    
    # Check if migration was successful
    if kubectl get job db-migration -n $NAMESPACE -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' | grep -q "True"; then
        log_info "Database migration completed successfully"
    else
        log_error "Database migration failed"
        kubectl logs job/db-migration -n $NAMESPACE
        exit 1
    fi
}

# Deploy application services
deploy_services() {
    log_info "Deploying application services..."
    
    # Deploy services
    kubectl apply -f k8s/api-service.yaml
    kubectl apply -f k8s/delivery-engine.yaml
    kubectl apply -f k8s/analytics-service.yaml
    
    # Deploy ingress
    kubectl apply -f k8s/ingress.yaml
    
    log_info "Application services deployment initiated"
}

# Verify deployment
verify_deployment() {
    log_info "Verifying deployment..."
    
    # Wait for deployments to be ready
    local services=("api-service" "delivery-engine" "analytics-service")
    
    for service in "${services[@]}"; do
        log_info "Waiting for $service to be ready..."
        kubectl rollout status deployment/$service -n $NAMESPACE --timeout=$TIMEOUT
        
        # Check if all pods are ready
        local ready_pods=$(kubectl get deployment $service -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
        local desired_pods=$(kubectl get deployment $service -n $NAMESPACE -o jsonpath='{.spec.replicas}')
        
        if [ "$ready_pods" != "$desired_pods" ]; then
            log_error "$service deployment verification failed: $ready_pods/$desired_pods pods ready"
            return 1
        fi
        
        log_info "$service deployment verified: $ready_pods/$desired_pods pods ready"
    done
    
    return 0
}

# Health check
health_check() {
    log_info "Performing health checks..."
    
    # Get service endpoints
    local api_service_ip=$(kubectl get service api-service -n $NAMESPACE -o jsonpath='{.spec.clusterIP}')
    local analytics_service_ip=$(kubectl get service analytics-service -n $NAMESPACE -o jsonpath='{.spec.clusterIP}')
    
    # Create a temporary pod for health checks
    kubectl run health-check --image=curlimages/curl:latest --rm -i --restart=Never -n $NAMESPACE -- /bin/sh -c "
        echo 'Testing API service health...'
        if curl -f http://$api_service_ip/health; then
            echo 'API service health check passed'
        else
            echo 'API service health check failed'
            exit 1
        fi
        
        echo 'Testing Analytics service health...'
        if curl -f http://$analytics_service_ip/health; then
            echo 'Analytics service health check passed'
        else
            echo 'Analytics service health check failed'
            exit 1
        fi
    "
    
    if [ $? -eq 0 ]; then
        log_info "Health checks passed"
        return 0
    else
        log_error "Health checks failed"
        return 1
    fi
}

# Rollback deployment
rollback_deployment() {
    log_warn "Initiating rollback..."
    
    local services=("api-service" "delivery-engine" "analytics-service")
    
    for service in "${services[@]}"; do
        log_info "Rolling back $service..."
        kubectl rollout undo deployment/$service -n $NAMESPACE
        kubectl rollout status deployment/$service -n $NAMESPACE --timeout=$ROLLBACK_TIMEOUT
    done
    
    log_info "Rollback completed"
}

# Deploy monitoring and logging
deploy_monitoring() {
    log_info "Deploying monitoring and logging..."
    
    kubectl apply -f k8s/monitoring.yaml
    kubectl apply -f k8s/logging.yaml
    kubectl apply -f k8s/backup-cronjob.yaml
    
    log_info "Monitoring and logging deployment completed"
}

# Main deployment function
main() {
    local command=${1:-"deploy"}
    
    case $command in
        "deploy")
            log_info "Starting production deployment..."
            
            check_prerequisites
            deploy_infrastructure
            run_migrations
            deploy_services
            
            if verify_deployment && health_check; then
                deploy_monitoring
                log_info "Deployment completed successfully!"
            else
                log_error "Deployment verification failed. Initiating rollback..."
                rollback_deployment
                exit 1
            fi
            ;;
        
        "rollback")
            log_warn "Starting rollback procedure..."
            check_prerequisites
            rollback_deployment
            ;;
        
        "verify")
            log_info "Running deployment verification..."
            check_prerequisites
            if verify_deployment && health_check; then
                log_info "Verification passed"
            else
                log_error "Verification failed"
                exit 1
            fi
            ;;
        
        "health")
            log_info "Running health checks..."
            check_prerequisites
            if health_check; then
                log_info "Health checks passed"
            else
                log_error "Health checks failed"
                exit 1
            fi
            ;;
        
        *)
            echo "Usage: $0 {deploy|rollback|verify|health}"
            echo "  deploy   - Full deployment with verification"
            echo "  rollback - Rollback to previous version"
            echo "  verify   - Verify current deployment"
            echo "  health   - Run health checks"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"