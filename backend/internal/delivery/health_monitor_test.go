package delivery

import (
	"context"
	"testing"
	"time"

	"webhook-platform/pkg/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEndpointHealthMonitor_RecordDeliveryResult_Success(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)

	endpointID := uuid.New()

	// Record successful delivery
	monitor.RecordDeliveryResult(endpointID, true, intPtr(200))

	// Check health status
	status := monitor.GetEndpointHealth(endpointID)
	assert.True(t, status.IsHealthy)
	assert.Equal(t, 1, status.SuccessCount)
	assert.Equal(t, 0, status.FailureCount)
	assert.Equal(t, 0, status.ConsecutiveFailures)
	assert.NotNil(t, status.LastSuccessAt)
	assert.Nil(t, status.LastFailureAt)
}

func TestEndpointHealthMonitor_RecordDeliveryResult_Failure(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)

	endpointID := uuid.New()

	// Record failed delivery
	monitor.RecordDeliveryResult(endpointID, false, intPtr(500))

	// Check health status
	status := monitor.GetEndpointHealth(endpointID)
	assert.True(t, status.IsHealthy) // Still healthy after one failure
	assert.Equal(t, 0, status.SuccessCount)
	assert.Equal(t, 1, status.FailureCount)
	assert.Equal(t, 1, status.ConsecutiveFailures)
	assert.Nil(t, status.LastSuccessAt)
	assert.NotNil(t, status.LastFailureAt)
}

func TestEndpointHealthMonitor_RecordDeliveryResult_DisableAfterFailures(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)
	monitor.failureThreshold = 3 // Lower threshold for testing

	endpointID := uuid.New()

	// Mock the UpdateStatus call that should happen when endpoint is disabled
	mockRepo.On("UpdateStatus", mock.Anything, endpointID, false).Return(nil)

	// Record multiple failures
	for i := 0; i < 3; i++ {
		monitor.RecordDeliveryResult(endpointID, false, intPtr(500))
	}

	// Check that endpoint is marked as unhealthy
	status := monitor.GetEndpointHealth(endpointID)
	assert.False(t, status.IsHealthy)
	assert.Equal(t, 3, status.ConsecutiveFailures)

	// Wait a bit for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	mockRepo.AssertExpectations(t)
}

func TestEndpointHealthMonitor_RecordDeliveryResult_RecoveryAfterFailures(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)
	monitor.failureThreshold = 2
	monitor.recoveryThreshold = 2

	endpointID := uuid.New()

	// Mock the disable and enable calls
	mockRepo.On("UpdateStatus", mock.Anything, endpointID, false).Return(nil)
	mockRepo.On("UpdateStatus", mock.Anything, endpointID, true).Return(nil)

	// Record failures to disable endpoint
	monitor.RecordDeliveryResult(endpointID, false, intPtr(500))
	monitor.RecordDeliveryResult(endpointID, false, intPtr(500))

	// Verify endpoint is disabled
	status := monitor.GetEndpointHealth(endpointID)
	assert.False(t, status.IsHealthy)

	// Record successes to re-enable endpoint
	monitor.RecordDeliveryResult(endpointID, true, intPtr(200))
	monitor.RecordDeliveryResult(endpointID, true, intPtr(200))

	// Verify endpoint is re-enabled
	status = monitor.GetEndpointHealth(endpointID)
	assert.True(t, status.IsHealthy)
	assert.Equal(t, 0, status.ConsecutiveFailures)

	// Wait for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	mockRepo.AssertExpectations(t)
}

func TestEndpointHealthMonitor_GetEndpointHealth_NonExistent(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)

	endpointID := uuid.New()

	// Get health for non-existent endpoint
	status := monitor.GetEndpointHealth(endpointID)
	assert.True(t, status.IsHealthy) // Should default to healthy
	assert.Equal(t, endpointID, status.EndpointID)
	assert.Equal(t, 0, status.SuccessCount)
	assert.Equal(t, 0, status.FailureCount)
}

func TestEndpointHealthMonitor_GetAllEndpointHealth(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)

	endpointID1 := uuid.New()
	endpointID2 := uuid.New()

	// Record some results
	monitor.RecordDeliveryResult(endpointID1, true, intPtr(200))
	monitor.RecordDeliveryResult(endpointID2, false, intPtr(500))

	// Get all health statuses
	allHealth := monitor.GetAllEndpointHealth()

	assert.Len(t, allHealth, 2)
	assert.Contains(t, allHealth, endpointID1)
	assert.Contains(t, allHealth, endpointID2)

	assert.True(t, allHealth[endpointID1].IsHealthy)
	assert.Equal(t, 1, allHealth[endpointID1].SuccessCount)

	assert.True(t, allHealth[endpointID2].IsHealthy) // Still healthy after one failure
	assert.Equal(t, 1, allHealth[endpointID2].FailureCount)
}

func TestEndpointHealthMonitor_ResetEndpointHealth(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)

	endpointID := uuid.New()

	// Record some results
	monitor.RecordDeliveryResult(endpointID, false, intPtr(500))
	monitor.RecordDeliveryResult(endpointID, false, intPtr(500))

	// Verify stats exist
	status := monitor.GetEndpointHealth(endpointID)
	assert.Equal(t, 2, status.FailureCount)

	// Reset health
	monitor.ResetEndpointHealth(endpointID)

	// Verify stats are reset
	status = monitor.GetEndpointHealth(endpointID)
	assert.True(t, status.IsHealthy) // Should default to healthy
	assert.Equal(t, 0, status.SuccessCount)
	assert.Equal(t, 0, status.FailureCount)
}

func TestEndpointHealthMonitor_PerformHealthCheck_Cleanup(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)

	endpointID := uuid.New()

	// Record a result
	monitor.RecordDeliveryResult(endpointID, true, intPtr(200))

	// Manually set last checked time to old value
	monitor.mutex.Lock()
	monitor.healthStats[endpointID].LastCheckedAt = time.Now().Add(-25 * time.Hour)
	monitor.mutex.Unlock()

	// Perform health check (should clean up old stats)
	monitor.performHealthCheck(context.Background())

	// Verify stats were cleaned up
	allHealth := monitor.GetAllEndpointHealth()
	assert.Len(t, allHealth, 0)
}

func TestEndpointHealthMonitor_Start_Stop(t *testing.T) {
	mockRepo := &MockWebhookRepository{}
	logger := utils.NewLogger("test")
	monitor := NewEndpointHealthMonitor(mockRepo, logger)
	monitor.checkInterval = 100 * time.Millisecond // Short interval for testing

	ctx, cancel := context.WithCancel(context.Background())

	// Start monitor in goroutine
	done := make(chan struct{})
	go func() {
		monitor.Start(ctx)
		close(done)
	}()

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Stop monitor
	cancel()

	// Wait for it to stop
	select {
	case <-done:
		// Monitor stopped successfully
	case <-time.After(1 * time.Second):
		t.Fatal("Monitor did not stop within timeout")
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}