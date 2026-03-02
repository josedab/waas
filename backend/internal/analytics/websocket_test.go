package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestWSManager() (*WebSocketManager, *MockAnalyticsRepository) {
	mockRepo := &MockAnalyticsRepository{}
	logger := utils.NewLogger("test")
	wsm := NewWebSocketManager(mockRepo, logger)
	return wsm, mockRepo
}

// =====================
// NewWebSocketManager
// =====================

func TestNewWebSocketManager(t *testing.T) {
	wsm, _ := newTestWSManager()

	require.NotNil(t, wsm)
	assert.NotNil(t, wsm.connections)
	assert.NotNil(t, wsm.stopCh)
	assert.NotNil(t, wsm.analyticsRepo)
	assert.NotNil(t, wsm.logger)
	assert.Empty(t, wsm.connections)
}

// =====================
// Client Registration / Deregistration
// =====================

func TestWSManager_AddRemoveConnection(t *testing.T) {
	wsm, _ := newTestWSManager()
	tenantID := uuid.New()

	// Use a nil conn for unit testing add/remove logic
	// (addConnection/removeConnection operate on the map)
	// We can't use real websocket.Conn without a server, so test the map logic

	wsm.mutex.Lock()
	if wsm.connections[tenantID] == nil {
		wsm.connections[tenantID] = make(map[*websocket.Conn]bool)
	}
	wsm.mutex.Unlock()

	// Test the connections map
	wsm.mutex.RLock()
	assert.Contains(t, wsm.connections, tenantID)
	wsm.mutex.RUnlock()

	// Remove tenant
	wsm.mutex.Lock()
	delete(wsm.connections, tenantID)
	wsm.mutex.Unlock()

	wsm.mutex.RLock()
	assert.NotContains(t, wsm.connections, tenantID)
	wsm.mutex.RUnlock()
}

// =====================
// Start / Stop
// =====================

func TestWSManager_StartStop(t *testing.T) {
	wsm, mockRepo := newTestWSManager()

	mockRepo.On("GetRealtimeMetrics", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]models.RealtimeMetric{}, nil).Maybe()
	mockRepo.On("GetDashboardMetrics", mock.Anything, mock.Anything, mock.Anything).
		Return(&models.DashboardMetrics{}, nil).Maybe()

	ctx := context.Background()
	wsm.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		wsm.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Stop timed out")
	}
}

func TestWSManager_StopClosesAllConnections(t *testing.T) {
	wsm, _ := newTestWSManager()

	// Simulate some connections in the map (using nil values — Stop will handle gracefully)
	wsm.mutex.Lock()
	tid1 := uuid.New()
	tid2 := uuid.New()
	wsm.connections[tid1] = make(map[*websocket.Conn]bool)
	wsm.connections[tid2] = make(map[*websocket.Conn]bool)
	wsm.mutex.Unlock()

	// Stop should clear all connections
	wsm.Stop()

	wsm.mutex.RLock()
	assert.Empty(t, wsm.connections)
	wsm.mutex.RUnlock()
}

// =====================
// broadcastMetrics with zero clients
// =====================

func TestWSManager_BroadcastToZeroClients(t *testing.T) {
	wsm, _ := newTestWSManager()

	// No connections — broadcastMetrics should complete without error
	wsm.broadcastMetrics()
}

func TestWSManager_BroadcastToEmptyTenantConnections(t *testing.T) {
	wsm, _ := newTestWSManager()

	// Add tenant with no connections
	wsm.mutex.Lock()
	tid := uuid.New()
	wsm.connections[tid] = make(map[*websocket.Conn]bool)
	wsm.mutex.Unlock()

	// Should skip this tenant (len(connections) == 0)
	wsm.broadcastMetrics()
}

// =====================
// getRealtimeMetricsForTenant
// =====================

func TestWSManager_GetRealtimeMetrics(t *testing.T) {
	wsm, mockRepo := newTestWSManager()
	tenantID := uuid.New()

	mockRepo.On("GetRealtimeMetrics", mock.Anything, tenantID, "delivery_rate", mock.Anything).
		Return([]models.RealtimeMetric{{MetricValue: 15.5}}, nil)
	mockRepo.On("GetRealtimeMetrics", mock.Anything, tenantID, "error_rate", mock.Anything).
		Return([]models.RealtimeMetric{{MetricValue: 2.3}}, nil)
	mockRepo.On("GetRealtimeMetrics", mock.Anything, tenantID, "latency", mock.Anything).
		Return([]models.RealtimeMetric{{MetricValue: 250.0}}, nil)

	metrics, err := wsm.getRealtimeMetricsForTenant(tenantID)
	require.NoError(t, err)

	assert.Equal(t, 15.5, metrics["delivery_rate"])
	assert.Equal(t, 2.3, metrics["error_rate"])
	assert.Equal(t, 250.0, metrics["avg_latency"])
}

func TestWSManager_GetRealtimeMetrics_EmptyMetrics(t *testing.T) {
	wsm, mockRepo := newTestWSManager()
	tenantID := uuid.New()

	mockRepo.On("GetRealtimeMetrics", mock.Anything, tenantID, mock.Anything, mock.Anything).
		Return([]models.RealtimeMetric{}, nil)

	metrics, err := wsm.getRealtimeMetricsForTenant(tenantID)
	require.NoError(t, err)

	assert.Equal(t, 0.0, metrics["delivery_rate"])
	assert.Equal(t, 0.0, metrics["error_rate"])
	assert.Equal(t, 0.0, metrics["avg_latency"])
}

// =====================
// getLatestMetricValue helper
// =====================

func TestGetLatestMetricValue(t *testing.T) {
	tests := []struct {
		name    string
		metrics []models.RealtimeMetric
		want    float64
	}{
		{"empty", []models.RealtimeMetric{}, 0.0},
		{"single", []models.RealtimeMetric{{MetricValue: 42.0}}, 42.0},
		{"multiple returns first", []models.RealtimeMetric{
			{MetricValue: 10.0},
			{MetricValue: 20.0},
		}, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getLatestMetricValue(tt.metrics))
		})
	}
}

// =====================
// broadcastToTenant with no connections
// =====================

func TestWSManager_BroadcastToTenant_NoConnections(t *testing.T) {
	wsm, _ := newTestWSManager()
	tenantID := uuid.New()

	msg := &models.WebSocketMessage{
		Type:      "test",
		TenantID:  tenantID,
		Timestamp: time.Now(),
	}

	// Should not panic when tenant has no connections
	wsm.broadcastToTenant(tenantID, msg)
}
