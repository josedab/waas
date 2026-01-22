#!/bin/bash

# wait-for-services.sh - Wait for required services to be ready

set -e

# Default values
POSTGRES_HOST=${POSTGRES_HOST:-postgres}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-postgres}
POSTGRES_DB=${POSTGRES_DB:-webhook_platform_test}

REDIS_HOST=${REDIS_HOST:-redis}
REDIS_PORT=${REDIS_PORT:-6379}

TIMEOUT=${TIMEOUT:-60}

echo "Waiting for services to be ready..."

# Function to wait for PostgreSQL
wait_for_postgres() {
    echo "Waiting for PostgreSQL at $POSTGRES_HOST:$POSTGRES_PORT..."
    
    for i in $(seq 1 $TIMEOUT); do
        if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
            echo "PostgreSQL is ready!"
            return 0
        fi
        echo "PostgreSQL is not ready yet... ($i/$TIMEOUT)"
        sleep 1
    done
    
    echo "PostgreSQL failed to become ready within $TIMEOUT seconds"
    return 1
}

# Function to wait for Redis
wait_for_redis() {
    echo "Waiting for Redis at $REDIS_HOST:$REDIS_PORT..."
    
    for i in $(seq 1 $TIMEOUT); do
        if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
            echo "Redis is ready!"
            return 0
        fi
        echo "Redis is not ready yet... ($i/$TIMEOUT)"
        sleep 1
    done
    
    echo "Redis failed to become ready within $TIMEOUT seconds"
    return 1
}

# Function to wait for HTTP service
wait_for_http() {
    local host=$1
    local port=$2
    local service_name=$3
    
    echo "Waiting for $service_name at $host:$port..."
    
    for i in $(seq 1 $TIMEOUT); do
        if curl -f "http://$host:$port/health" >/dev/null 2>&1; then
            echo "$service_name is ready!"
            return 0
        fi
        echo "$service_name is not ready yet... ($i/$TIMEOUT)"
        sleep 1
    done
    
    echo "$service_name failed to become ready within $TIMEOUT seconds"
    return 1
}

# Main execution
main() {
    local exit_code=0
    
    # Wait for PostgreSQL
    if ! wait_for_postgres; then
        exit_code=1
    fi
    
    # Wait for Redis
    if ! wait_for_redis; then
        exit_code=1
    fi
    
    # If additional services are specified, wait for them
    if [ -n "$WAIT_FOR_SERVICES" ]; then
        IFS=',' read -ra SERVICES <<< "$WAIT_FOR_SERVICES"
        for service in "${SERVICES[@]}"; do
            IFS=':' read -ra SERVICE_PARTS <<< "$service"
            local service_host=${SERVICE_PARTS[0]}
            local service_port=${SERVICE_PARTS[1]}
            local service_name=${SERVICE_PARTS[2]:-"Service"}
            
            if ! wait_for_http "$service_host" "$service_port" "$service_name"; then
                exit_code=1
            fi
        done
    fi
    
    if [ $exit_code -eq 0 ]; then
        echo "All services are ready!"
    else
        echo "Some services failed to become ready"
    fi
    
    return $exit_code
}

# Run main function
main "$@"