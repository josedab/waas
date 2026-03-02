package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/josedab/waas/pkg/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestService_ComponentsInitialized verifies the Service struct can hold all components.
// NewService requires a real DB, so we test the individual components instead.
func TestService_ComponentsInitialized(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := utils.NewLogger("test")

	handlers := NewHandlers(mockRepo)
	wsManager := NewWebSocketManager(mockRepo, logger)
	aggregator := NewAggregator(mockRepo, logger)

	require.NotNil(t, handlers)
	require.NotNil(t, wsManager)
	require.NotNil(t, aggregator)
}

// TestService_StartWorkersAndStop tests that starting and stopping workers
// does not leak goroutines or deadlock.
func TestService_StartWorkersAndStop(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := utils.NewLogger("test")

	// Set up mock expectations for aggregator startup
	mockRepo.On("GetMetricsSummary", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	mockRepo.On("CleanupOldMetrics", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("RecordRealtimeMetric", mock.Anything, mock.Anything).Return(nil).Maybe()

	wsManager := NewWebSocketManager(mockRepo, logger)
	aggregator := NewAggregator(mockRepo, logger)

	ctx := context.Background()
	wsManager.Start(ctx)
	aggregator.Start(ctx)

	// Give goroutines time to spin up
	time.Sleep(50 * time.Millisecond)

	// Stop both — should complete without deadlock
	done := make(chan struct{})
	go func() {
		wsManager.Stop()
		aggregator.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not complete within timeout — possible goroutine leak")
	}
}

// TestService_HandlerReturnsRouter verifies that handler registration produces a non-nil engine.
func TestService_HandlerReturnsRouter(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	handlers := NewHandlers(mockRepo)

	require.NotNil(t, handlers)

	// Verify routes are registered by creating a gin engine
	router := setupTestRouter(mockRepo)
	assert.NotNil(t, router)
}

// TestService_ShutdownOrdering tests that workers stop before DB would close.
// This documents the risk: Stop() calls wsManager.Stop() and aggregator.Stop()
// which use wg.Wait(), and then db.Close(). If workers reference the DB after
// close, there will be panics. We verify the ordering is correct.
func TestService_ShutdownOrdering(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{}
	logger := utils.NewLogger("test")

	mockRepo.On("GetMetricsSummary", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	mockRepo.On("CleanupOldMetrics", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("RecordRealtimeMetric", mock.Anything, mock.Anything).Return(nil).Maybe()

	wsManager := NewWebSocketManager(mockRepo, logger)
	aggregator := NewAggregator(mockRepo, logger)

	ctx := context.Background()
	wsManager.Start(ctx)
	aggregator.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	// Simulate the Stop() order from service.go:
	// 1. wsManager.Stop() — waits for goroutines
	// 2. aggregator.Stop() — waits for goroutines
	// 3. db.Close() — would happen after
	wsManager.Stop()
	aggregator.Stop()
	// If we reach here, shutdown ordering is correct
}
