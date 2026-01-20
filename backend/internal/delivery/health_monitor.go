package delivery

import (
	"context"
	"sync"
	"time"

	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"

	"github.com/google/uuid"
)

// EndpointHealthStatus represents the health status of an endpoint
type EndpointHealthStatus struct {
	EndpointID        uuid.UUID `json:"endpoint_id"`
	SuccessCount      int       `json:"success_count"`
	FailureCount      int       `json:"failure_count"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	LastSuccessAt     *time.Time `json:"last_success_at"`
	LastFailureAt     *time.Time `json:"last_failure_at"`
	IsHealthy         bool      `json:"is_healthy"`
	LastCheckedAt     time.Time `json:"last_checked_at"`
}

// EndpointHealthMonitor monitors endpoint health and auto-disables unhealthy endpoints
type EndpointHealthMonitor struct {
	webhookRepo     repository.WebhookEndpointRepository
	logger          *utils.Logger
	healthStats     map[uuid.UUID]*EndpointHealthStatus
	mutex           sync.RWMutex
	checkInterval   time.Duration
	failureThreshold int
	recoveryThreshold int
}

// NewEndpointHealthMonitor creates a new endpoint health monitor
func NewEndpointHealthMonitor(webhookRepo repository.WebhookEndpointRepository, logger *utils.Logger) *EndpointHealthMonitor {
	return &EndpointHealthMonitor{
		webhookRepo:       webhookRepo,
		logger:            logger,
		healthStats:       make(map[uuid.UUID]*EndpointHealthStatus),
		checkInterval:     5 * time.Minute,
		failureThreshold:  10, // Disable after 10 consecutive failures
		recoveryThreshold: 5,  // Re-enable after 5 consecutive successes
	}
}

// Start begins the health monitoring process
func (hm *EndpointHealthMonitor) Start(ctx context.Context) {
	hm.logger.Info("Starting endpoint health monitor", nil)
	
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hm.logger.Info("Endpoint health monitor stopped", nil)
			return
		case <-ticker.C:
			hm.performHealthCheck(ctx)
		}
	}
}

// RecordDeliveryResult records the result of a delivery attempt for health monitoring
func (hm *EndpointHealthMonitor) RecordDeliveryResult(endpointID uuid.UUID, success bool, httpStatus *int) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	status, exists := hm.healthStats[endpointID]
	if !exists {
		status = &EndpointHealthStatus{
			EndpointID: endpointID,
			IsHealthy:  true,
		}
		hm.healthStats[endpointID] = status
	}

	now := time.Now()
	status.LastCheckedAt = now

	if success {
		status.SuccessCount++
		status.ConsecutiveFailures = 0
		status.LastSuccessAt = &now
		
		// Check if endpoint should be re-enabled
		if !status.IsHealthy && status.SuccessCount >= hm.recoveryThreshold {
			hm.logger.Info("Endpoint recovered, re-enabling", map[string]interface{}{
				"endpoint_id":    endpointID,
				"success_count":  status.SuccessCount,
			})
			status.IsHealthy = true
			go hm.enableEndpoint(endpointID)
		}
	} else {
		status.FailureCount++
		status.ConsecutiveFailures++
		status.LastFailureAt = &now
		
		// Check if endpoint should be disabled
		if status.IsHealthy && status.ConsecutiveFailures >= hm.failureThreshold {
			hm.logger.Warn("Endpoint unhealthy, disabling", map[string]interface{}{
				"endpoint_id":         endpointID,
				"consecutive_failures": status.ConsecutiveFailures,
				"http_status":         httpStatus,
			})
			status.IsHealthy = false
			go hm.disableEndpoint(endpointID)
		}
	}
}

// GetEndpointHealth returns the health status for an endpoint
func (hm *EndpointHealthMonitor) GetEndpointHealth(endpointID uuid.UUID) *EndpointHealthStatus {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	if status, exists := hm.healthStats[endpointID]; exists {
		// Return a copy to avoid race conditions
		statusCopy := *status
		return &statusCopy
	}

	return &EndpointHealthStatus{
		EndpointID: endpointID,
		IsHealthy:  true,
	}
}

// GetAllEndpointHealth returns health status for all monitored endpoints
func (hm *EndpointHealthMonitor) GetAllEndpointHealth() map[uuid.UUID]*EndpointHealthStatus {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	result := make(map[uuid.UUID]*EndpointHealthStatus)
	for id, status := range hm.healthStats {
		statusCopy := *status
		result[id] = &statusCopy
	}

	return result
}

// performHealthCheck performs periodic health checks and cleanup
func (hm *EndpointHealthMonitor) performHealthCheck(ctx context.Context) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour) // Clean up stats older than 24 hours

	for endpointID, status := range hm.healthStats {
		// Clean up old stats
		if status.LastCheckedAt.Before(cutoff) {
			delete(hm.healthStats, endpointID)
			continue
		}

		// Log health status for monitoring
		hm.logger.Debug("Endpoint health status", map[string]interface{}{
			"endpoint_id":         endpointID,
			"is_healthy":          status.IsHealthy,
			"success_count":       status.SuccessCount,
			"failure_count":       status.FailureCount,
			"consecutive_failures": status.ConsecutiveFailures,
		})
	}

	hm.logger.Info("Health check completed", map[string]interface{}{
		"monitored_endpoints": len(hm.healthStats),
	})
}

// disableEndpoint disables an unhealthy endpoint
func (hm *EndpointHealthMonitor) disableEndpoint(endpointID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := hm.webhookRepo.UpdateStatus(ctx, endpointID, false); err != nil {
		hm.logger.Error("Failed to disable unhealthy endpoint", map[string]interface{}{
			"endpoint_id": endpointID,
			"error":       err.Error(),
		})
		return
	}

	hm.logger.Info("Disabled unhealthy endpoint", map[string]interface{}{
		"endpoint_id": endpointID,
	})
}

// enableEndpoint re-enables a recovered endpoint
func (hm *EndpointHealthMonitor) enableEndpoint(endpointID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := hm.webhookRepo.UpdateStatus(ctx, endpointID, true); err != nil {
		hm.logger.Error("Failed to enable recovered endpoint", map[string]interface{}{
			"endpoint_id": endpointID,
			"error":       err.Error(),
		})
		return
	}

	hm.logger.Info("Enabled recovered endpoint", map[string]interface{}{
		"endpoint_id": endpointID,
	})
}

// ResetEndpointHealth resets the health statistics for an endpoint
func (hm *EndpointHealthMonitor) ResetEndpointHealth(endpointID uuid.UUID) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	delete(hm.healthStats, endpointID)
	
	hm.logger.Info("Reset endpoint health statistics", map[string]interface{}{
		"endpoint_id": endpointID,
	})
}