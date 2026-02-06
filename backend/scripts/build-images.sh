#!/bin/bash

# Production Image Build Script for Webhook Platform
# This script builds and pushes production Docker images

set -euo pipefail

# Configuration
REGISTRY=${DOCKER_REGISTRY:-"your-registry.com"}
PROJECT=${PROJECT_NAME:-"webhook-platform"}
VERSION=${VERSION:-$(git rev-parse --short HEAD)}
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git rev-parse HEAD)

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
    
    # Check if docker is available
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir &> /dev/null; then
        log_error "Not in a git repository"
        exit 1
    fi
    
    # Check if there are uncommitted changes
    if ! git diff-index --quiet HEAD --; then
        log_warn "There are uncommitted changes in the repository"
    fi
    
    log_info "Prerequisites check passed"
}

# Build a single service image
build_service_image() {
    local service=$1
    local dockerfile=$2
    local image_name="$REGISTRY/$PROJECT/$service:$VERSION"
    local latest_name="$REGISTRY/$PROJECT/$service:latest"
    
    log_info "Building $service image..."
    
    # Build the image with build args
    docker build \
        --file "$dockerfile" \
        --tag "$image_name" \
        --tag "$latest_name" \
        --build-arg VERSION="$VERSION" \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        --build-arg GIT_COMMIT="$GIT_COMMIT" \
        --label "version=$VERSION" \
        --label "build-date=$BUILD_DATE" \
        --label "git-commit=$GIT_COMMIT" \
        --label "service=$service" \
        .
    
    if [ $? -eq 0 ]; then
        log_info "$service image built successfully: $image_name"
    else
        log_error "Failed to build $service image"
        exit 1
    fi
}

# Push image to registry
push_image() {
    local service=$1
    local image_name="$REGISTRY/$PROJECT/$service:$VERSION"
    local latest_name="$REGISTRY/$PROJECT/$service:latest"
    
    log_info "Pushing $service image to registry..."
    
    # Push versioned image
    docker push "$image_name"
    if [ $? -ne 0 ]; then
        log_error "Failed to push $service image: $image_name"
        exit 1
    fi
    
    # Push latest tag
    docker push "$latest_name"
    if [ $? -ne 0 ]; then
        log_error "Failed to push $service latest image: $latest_name"
        exit 1
    fi
    
    log_info "$service image pushed successfully"
}

# Scan image for vulnerabilities
scan_image() {
    local service=$1
    local image_name="$REGISTRY/$PROJECT/$service:$VERSION"
    
    log_info "Scanning $service image for vulnerabilities..."
    
    # Use trivy for vulnerability scanning if available
    if command -v trivy &> /dev/null; then
        trivy image --exit-code 1 --severity HIGH,CRITICAL "$image_name"
        if [ $? -ne 0 ]; then
            log_error "Vulnerability scan failed for $service image"
            return 1
        fi
        log_info "$service image vulnerability scan passed"
    else
        log_warn "Trivy not available, skipping vulnerability scan"
    fi
    
    return 0
}

# Build all service images
build_all_images() {
    log_info "Building all production images..."
    
    # Define services and their corresponding Dockerfiles
    declare -A services=(
        ["api-service"]="docker/Dockerfile.api.prod"
        ["delivery-engine"]="docker/Dockerfile.delivery.prod"
        ["analytics-service"]="docker/Dockerfile.analytics.prod"
    )
    
    # Build each service
    for service in "${!services[@]}"; do
        build_service_image "$service" "${services[$service]}"
    done
    
    log_info "All images built successfully"
}

# Push all images
push_all_images() {
    log_info "Pushing all images to registry..."
    
    local services=("api-service" "delivery-engine" "analytics-service")
    
    for service in "${services[@]}"; do
        push_image "$service"
    done
    
    log_info "All images pushed successfully"
}

# Scan all images
scan_all_images() {
    log_info "Scanning all images for vulnerabilities..."
    
    local services=("api-service" "delivery-engine" "analytics-service")
    local scan_failed=false
    
    for service in "${services[@]}"; do
        if ! scan_image "$service"; then
            scan_failed=true
        fi
    done
    
    if [ "$scan_failed" = true ]; then
        log_error "One or more vulnerability scans failed"
        return 1
    fi
    
    log_info "All vulnerability scans passed"
    return 0
}

# Generate image manifest
generate_manifest() {
    log_info "Generating image manifest..."
    
    local manifest_file="image-manifest-$VERSION.json"
    
    cat > "$manifest_file" << EOF
{
  "version": "$VERSION",
  "buildDate": "$BUILD_DATE",
  "gitCommit": "$GIT_COMMIT",
  "registry": "$REGISTRY",
  "project": "$PROJECT",
  "images": {
    "api-service": "$REGISTRY/$PROJECT/api-service:$VERSION",
    "delivery-engine": "$REGISTRY/$PROJECT/delivery-engine:$VERSION",
    "analytics-service": "$REGISTRY/$PROJECT/analytics-service:$VERSION"
  }
}
EOF
    
    log_info "Image manifest generated: $manifest_file"
}

# Clean up local images
cleanup_images() {
    log_info "Cleaning up local images..."
    
    local services=("api-service" "delivery-engine" "analytics-service")
    
    for service in "${services[@]}"; do
        # Remove local images to save space
        docker rmi "$REGISTRY/$PROJECT/$service:$VERSION" 2>/dev/null || true
        docker rmi "$REGISTRY/$PROJECT/$service:latest" 2>/dev/null || true
    done
    
    # Clean up dangling images
    docker image prune -f
    
    log_info "Local images cleaned up"
}

# Main function
main() {
    local command=${1:-"build"}
    
    case $command in
        "build")
            log_info "Building production images..."
            check_prerequisites
            build_all_images
            generate_manifest
            log_info "Build completed successfully!"
            ;;
        
        "push")
            log_info "Building and pushing production images..."
            check_prerequisites
            build_all_images
            
            if scan_all_images; then
                push_all_images
                generate_manifest
                cleanup_images
                log_info "Build and push completed successfully!"
            else
                log_error "Vulnerability scans failed. Images not pushed."
                exit 1
            fi
            ;;
        
        "scan")
            log_info "Scanning existing images..."
            check_prerequisites
            scan_all_images
            ;;
        
        "clean")
            log_info "Cleaning up local images..."
            cleanup_images
            ;;
        
        *)
            echo "Usage: $0 {build|push|scan|clean}"
            echo "  build - Build all production images locally"
            echo "  push  - Build, scan, and push all images to registry"
            echo "  scan  - Scan existing images for vulnerabilities"
            echo "  clean - Clean up local images"
            echo ""
            echo "Environment variables:"
            echo "  DOCKER_REGISTRY - Docker registry URL (default: your-registry.com)"
            echo "  PROJECT_NAME    - Project name (default: webhook-platform)"
            echo "  VERSION         - Image version (default: git short hash)"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"