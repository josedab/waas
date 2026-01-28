package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
	"webhook-platform/pkg/metrics"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	wsReadBufferSize       = 1024
	wsWriteBufferSize      = 1024
	wsReadDeadline         = 60 * time.Second
	wsPingInterval         = 30 * time.Second
	wsWriteDeadline        = 10 * time.Second
	wsBroadcastInterval    = 10 * time.Second
	wsCleanupInterval      = 5 * time.Minute
	wsCleanupPingTimeout   = 5 * time.Second
	wsRealtimeMetricWindow = 1 * time.Minute
	wsDashboardWindow      = time.Hour
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     utils.CheckWebSocketOrigin(),
	ReadBufferSize:  wsReadBufferSize,
	WriteBufferSize: wsWriteBufferSize,
}

// WebSocketManager manages WebSocket connections for real-time metrics
type WebSocketManager struct {
	connections   map[uuid.UUID]map[*websocket.Conn]bool // tenant_id -> connections
	mutex         sync.RWMutex
	analyticsRepo repository.AnalyticsRepositoryInterface
	logger        *utils.Logger
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(analyticsRepo repository.AnalyticsRepositoryInterface, logger *utils.Logger) *WebSocketManager {
	return &WebSocketManager{
		connections:   make(map[uuid.UUID]map[*websocket.Conn]bool),
		analyticsRepo: analyticsRepo,
		logger:        logger,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the WebSocket manager
func (wsm *WebSocketManager) Start(ctx context.Context) {
	wsm.logger.Info("Starting WebSocket manager", nil)

	// Start metrics broadcaster
	wsm.wg.Add(1)
	go wsm.metricsbroadcaster(ctx)

	// Start connection cleanup worker
	wsm.wg.Add(1)
	go wsm.connectionCleanup(ctx)
}

// Stop gracefully stops the WebSocket manager
func (wsm *WebSocketManager) Stop() {
	wsm.logger.Info("Stopping WebSocket manager", nil)
	close(wsm.stopCh)

	// Close all connections
	wsm.mutex.Lock()
	for tenantID, connections := range wsm.connections {
		for conn := range connections {
			conn.Close()
		}
		delete(wsm.connections, tenantID)
	}
	wsm.mutex.Unlock()

	wsm.wg.Wait()
}

// HandleWebSocket handles WebSocket connection upgrades
func (wsm *WebSocketManager) HandleWebSocket(c *gin.Context) {
	// Get tenant ID from context (set by auth middleware)
	tenantID, err := wsm.getTenantIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		wsm.logger.Error("Failed to upgrade WebSocket connection", map[string]interface{}{
			"error":     err.Error(),
			"tenant_id": tenantID,
		})
		return
	}

	// Register connection
	wsm.addConnection(tenantID, conn)
	defer wsm.removeConnection(tenantID, conn)

	wsm.logger.Info("WebSocket connection established", map[string]interface{}{
		"tenant_id": tenantID,
	})

	// Update active connections metric
	wsm.updateConnectionsMetric()

	// Send initial dashboard data
	wsm.sendInitialData(conn, tenantID)

	// Handle connection
	wsm.handleConnection(conn, tenantID)
}

// RegisterWebSocketRoutes registers WebSocket routes
func (wsm *WebSocketManager) RegisterWebSocketRoutes(router *gin.Engine) {
	router.GET("/analytics/ws", wsm.HandleWebSocket)
}

// addConnection adds a new WebSocket connection for a tenant
func (wsm *WebSocketManager) addConnection(tenantID uuid.UUID, conn *websocket.Conn) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	if wsm.connections[tenantID] == nil {
		wsm.connections[tenantID] = make(map[*websocket.Conn]bool)
	}
	wsm.connections[tenantID][conn] = true
}

// removeConnection removes a WebSocket connection for a tenant
func (wsm *WebSocketManager) removeConnection(tenantID uuid.UUID, conn *websocket.Conn) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	if connections, exists := wsm.connections[tenantID]; exists {
		delete(connections, conn)
		if len(connections) == 0 {
			delete(wsm.connections, tenantID)
		}
	}

	conn.Close()
	wsm.updateConnectionsMetric()
}

// handleConnection manages a single WebSocket connection
func (wsm *WebSocketManager) handleConnection(conn *websocket.Conn, tenantID uuid.UUID) {
	// Set read deadline and pong handler for keepalive
	conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Send ping
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				wsm.logger.Error("Failed to send ping", map[string]interface{}{
					"error":     err.Error(),
					"tenant_id": tenantID,
				})
				return
			}
		default:
			// Read message (mainly for pong responses and connection management)
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					wsm.logger.Error("WebSocket connection error", map[string]interface{}{
						"error":     err.Error(),
						"tenant_id": tenantID,
					})
				}
				return
			}
		}
	}
}

// metricsbroadcaster periodically broadcasts metrics to connected clients
func (wsm *WebSocketManager) metricsbroadcaster(ctx context.Context) {
	defer wsm.wg.Done()

	ticker := time.NewTicker(wsBroadcastInterval) // Broadcast every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wsm.stopCh:
			return
		case <-ticker.C:
			wsm.broadcastMetrics()
		}
	}
}

// connectionCleanup periodically cleans up stale connections
func (wsm *WebSocketManager) connectionCleanup(ctx context.Context) {
	defer wsm.wg.Done()

	ticker := time.NewTicker(wsCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wsm.stopCh:
			return
		case <-ticker.C:
			wsm.cleanupStaleConnections()
		}
	}
}

// broadcastMetrics sends real-time metrics to all connected clients
func (wsm *WebSocketManager) broadcastMetrics() {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	for tenantID, connections := range wsm.connections {
		if len(connections) == 0 {
			continue
		}

		// Get real-time metrics for this tenant
		metrics, err := wsm.getRealtimeMetricsForTenant(tenantID)
		if err != nil {
			wsm.logger.Error("Failed to get real-time metrics", map[string]interface{}{
				"error":     err.Error(),
				"tenant_id": tenantID,
			})
			continue
		}

		// Broadcast to all connections for this tenant
		message := &models.WebSocketMessage{
			Type:      "metrics_update",
			TenantID:  tenantID,
			Data:      metrics,
			Timestamp: time.Now(),
		}

		wsm.broadcastToTenant(tenantID, message)
	}
}

// sendInitialData sends initial dashboard data to a new connection
func (wsm *WebSocketManager) sendInitialData(conn *websocket.Conn, tenantID uuid.UUID) {
	// Get dashboard metrics for the last hour
	dashboard, err := wsm.analyticsRepo.GetDashboardMetrics(context.Background(), tenantID, wsDashboardWindow)
	if err != nil {
		wsm.logger.Error("Failed to get initial dashboard data", map[string]interface{}{
			"error":     err.Error(),
			"tenant_id": tenantID,
		})
		return
	}

	message := &models.WebSocketMessage{
		Type:      "initial_data",
		TenantID:  tenantID,
		Data:      dashboard,
		Timestamp: time.Now(),
	}

	wsm.sendMessage(conn, message)
}

// broadcastToTenant sends a message to all connections for a specific tenant
func (wsm *WebSocketManager) broadcastToTenant(tenantID uuid.UUID, message *models.WebSocketMessage) {
	connections, exists := wsm.connections[tenantID]
	if !exists {
		return
	}

	var staleConnections []*websocket.Conn

	for conn := range connections {
		err := wsm.sendMessage(conn, message)
		if err != nil {
			wsm.logger.Error("Failed to send WebSocket message", map[string]interface{}{
				"error":     err.Error(),
				"tenant_id": tenantID,
			})
			staleConnections = append(staleConnections, conn)
		}
	}

	// Remove stale connections
	for _, conn := range staleConnections {
		wsm.removeConnection(tenantID, conn)
	}
}

// sendMessage sends a message to a specific WebSocket connection
func (wsm *WebSocketManager) sendMessage(conn *websocket.Conn, message *models.WebSocketMessage) error {
	conn.SetWriteDeadline(time.Now().Add(wsWriteDeadline))

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// getRealtimeMetricsForTenant retrieves current real-time metrics for a tenant
func (wsm *WebSocketManager) getRealtimeMetricsForTenant(tenantID uuid.UUID) (map[string]interface{}, error) {
	ctx := context.Background()
	since := time.Now().Add(-wsRealtimeMetricWindow)

	// Get various real-time metrics
	deliveryRateMetrics, err := wsm.analyticsRepo.GetRealtimeMetrics(ctx, tenantID, "delivery_rate", since)
	if err != nil {
		return nil, err
	}

	errorRateMetrics, err := wsm.analyticsRepo.GetRealtimeMetrics(ctx, tenantID, "error_rate", since)
	if err != nil {
		return nil, err
	}

	latencyMetrics, err := wsm.analyticsRepo.GetRealtimeMetrics(ctx, tenantID, "latency", since)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"delivery_rate": getLatestMetricValue(deliveryRateMetrics),
		"error_rate":    getLatestMetricValue(errorRateMetrics),
		"avg_latency":   getLatestMetricValue(latencyMetrics),
		"timestamp":     time.Now(),
	}, nil
}

// cleanupStaleConnections removes connections that are no longer responsive
func (wsm *WebSocketManager) cleanupStaleConnections() {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	for tenantID, connections := range wsm.connections {
		var staleConnections []*websocket.Conn

		for conn := range connections {
			// Try to write a ping message to test connection
			conn.SetWriteDeadline(time.Now().Add(wsCleanupPingTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				staleConnections = append(staleConnections, conn)
			}
		}

		// Remove stale connections
		for _, conn := range staleConnections {
			delete(connections, conn)
			conn.Close()
		}

		if len(connections) == 0 {
			delete(wsm.connections, tenantID)
		}
	}

	wsm.updateConnectionsMetric()
}

// updateConnectionsMetric updates the Prometheus metric for active connections
func (wsm *WebSocketManager) updateConnectionsMetric() {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	totalConnections := 0
	for _, connections := range wsm.connections {
		totalConnections += len(connections)
	}

	metrics.UpdateActiveConnections(float64(totalConnections))
}

// Helper functions

func (wsm *WebSocketManager) getTenantIDFromContext(c *gin.Context) (uuid.UUID, error) {
	tenantIDInterface, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, gin.Error{Err: gin.Error{}, Type: gin.ErrorTypePublic}
	}

	tenantIDStr, ok := tenantIDInterface.(string)
	if !ok {
		return uuid.Nil, gin.Error{Err: gin.Error{}, Type: gin.ErrorTypePublic}
	}

	return uuid.Parse(tenantIDStr)
}

func getLatestMetricValue(metrics []models.RealtimeMetric) float64 {
	if len(metrics) == 0 {
		return 0.0
	}
	return metrics[0].MetricValue // Metrics are ordered by timestamp DESC
}
