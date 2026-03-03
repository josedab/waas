package playground

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for playground
	},
}

// WSMessage represents a message sent over the WebSocket connection
type WSMessage struct {
	Type      string      `json:"type"`      // event, error, ping, pong, connected
	SessionID string      `json:"session_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// WSEventPayload is the data for a webhook event message
type WSEventPayload struct {
	Direction  string            `json:"direction"` // inbound, outbound
	EventType  string            `json:"event_type,omitempty"`
	Method     string            `json:"method,omitempty"`
	URL        string            `json:"url,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	StatusCode int               `json:"status_code,omitempty"`
	LatencyMs  int64             `json:"latency_ms,omitempty"`
}

// WebSocketHub manages active WebSocket connections per session
type WebSocketHub struct {
	mu          sync.RWMutex
	connections map[string]map[*websocket.Conn]bool // sessionID -> set of connections
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewWebSocketHub creates a new WebSocket hub bound to the given context.
// When ctx is cancelled, all connections are drained via Close.
func NewWebSocketHub() *WebSocketHub {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketHub{
		connections: make(map[string]map[*websocket.Conn]bool),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// NewWebSocketHubWithContext creates a new WebSocket hub that shuts down when
// the provided context is cancelled.
func NewWebSocketHubWithContext(ctx context.Context) *WebSocketHub {
	childCtx, cancel := context.WithCancel(ctx)
	hub := &WebSocketHub{
		connections: make(map[string]map[*websocket.Conn]bool),
		ctx:         childCtx,
		cancel:      cancel,
	}
	go hub.watchShutdown()
	return hub
}

// watchShutdown waits for the context to be cancelled, then drains connections.
func (hub *WebSocketHub) watchShutdown() {
	<-hub.ctx.Done()
	hub.Close()
}

// Close sends a close message to all connections and removes them.
func (hub *WebSocketHub) Close() {
	hub.cancel()
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for sessionID, conns := range hub.connections {
		for conn := range conns {
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"))
			conn.Close()
		}
		delete(hub.connections, sessionID)
	}
}

// Register adds a connection to a session
func (hub *WebSocketHub) Register(sessionID string, conn *websocket.Conn) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.connections[sessionID] == nil {
		hub.connections[sessionID] = make(map[*websocket.Conn]bool)
	}
	hub.connections[sessionID][conn] = true
}

// Unregister removes a connection from a session
func (hub *WebSocketHub) Unregister(sessionID string, conn *websocket.Conn) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if conns, ok := hub.connections[sessionID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(hub.connections, sessionID)
		}
	}
}

// Broadcast sends a message to all connections in a session
func (hub *WebSocketHub) Broadcast(sessionID string, msg WSMessage) {
	hub.mu.RLock()
	conns := hub.connections[sessionID]
	hub.mu.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			hub.Unregister(sessionID, conn)
			conn.Close()
		}
	}
}

// ConnectionCount returns the number of active connections for a session
func (hub *WebSocketHub) ConnectionCount(sessionID string) int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.connections[sessionID])
}

// RegisterWSRoutes registers WebSocket routes for real-time playground communication
func (h *Handler) RegisterWSRoutes(r *gin.RouterGroup) {
	pg := r.Group("/playground")
	{
		pg.GET("/ws/:session_id", h.HandleWebSocket)
		pg.POST("/ws/:session_id/send", h.SendWebhookEvent)
		pg.POST("/ws/:session_id/simulate", h.SimulateFailure)
		pg.POST("/ws/:session_id/transform", h.PreviewTransform)
	}
}

// HandleWebSocket upgrades HTTP to WebSocket for real-time event streaming
// @Summary Connect to playground WebSocket
// @Tags Playground
// @Param session_id path string true "Session ID"
// @Router /playground/ws/{session_id} [get]
func (h *Handler) HandleWebSocket(c *gin.Context) {
	sessionID := c.Param("session_id")

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("websocket upgrade failed: %v", err)})
		return
	}
	defer conn.Close()

	h.hub.Register(sessionID, conn)
	defer h.hub.Unregister(sessionID, conn)

	// Send connected message
	connectMsg := WSMessage{
		Type:      "connected",
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"message": "Connected to playground session"},
	}
	data, _ := json.Marshal(connectMsg)
	conn.WriteMessage(websocket.TextMessage, data)

	// Read loop — process incoming commands
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var incoming WSMessage
		if err := json.Unmarshal(msgBytes, &incoming); err != nil {
			errMsg := WSMessage{
				Type:      "error",
				SessionID: sessionID,
				Timestamp: time.Now(),
				Data:      map[string]string{"error": "invalid message format"},
			}
			errData, _ := json.Marshal(errMsg)
			conn.WriteMessage(websocket.TextMessage, errData)
			continue
		}

		// Handle ping/pong
		if incoming.Type == "ping" {
			pong := WSMessage{
				Type:      "pong",
				SessionID: sessionID,
				Timestamp: time.Now(),
			}
			pongData, _ := json.Marshal(pong)
			conn.WriteMessage(websocket.TextMessage, pongData)
		}
	}
}

// SendWebhookEvent sends a webhook through the playground and broadcasts to WebSocket
// @Summary Send a webhook event through the playground
// @Tags Playground
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200
// @Router /playground/ws/{session_id}/send [post]
func (h *Handler) SendWebhookEvent(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req struct {
		EventType string            `json:"event_type"`
		Headers   map[string]string `json:"headers,omitempty"`
		Payload   string            `json:"payload" binding:"required"`
		URL       string            `json:"url,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Broadcast outbound event to all connected clients
	outbound := WSMessage{
		Type:      "event",
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: WSEventPayload{
			Direction: "outbound",
			EventType: req.EventType,
			Method:    "POST",
			URL:       req.URL,
			Headers:   req.Headers,
			Body:      req.Payload,
		},
	}
	h.hub.Broadcast(sessionID, outbound)

	// Simulate response after a short delay
	inbound := WSMessage{
		Type:      "event",
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: WSEventPayload{
			Direction:  "inbound",
			StatusCode: 200,
			LatencyMs:  42,
			Body:       `{"status":"ok"}`,
		},
	}
	h.hub.Broadcast(sessionID, inbound)

	c.JSON(http.StatusOK, gin.H{
		"message":     "webhook sent",
		"session_id":  sessionID,
		"connections": h.hub.ConnectionCount(sessionID),
	})
}

// SimulateFailure simulates a delivery failure and broadcasts the result
// @Summary Simulate a webhook delivery failure
// @Tags Playground
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200
// @Router /playground/ws/{session_id}/simulate [post]
func (h *Handler) SimulateFailure(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req struct {
		FailureType string `json:"failure_type" binding:"required"` // timeout, server_error, rate_limit, network_error, slow_response
		Payload     string `json:"payload,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	simulator := NewFailureSimulator()
	sid := uuid.New() // Ephemeral simulation
	simulator.SetSimulation(sid, &FailureSimulation{
		Type:    req.FailureType,
		Enabled: true,
	})
	statusCode, body, latencyMs, _ := simulator.SimulateResponse(sid)

	// Broadcast failure event
	event := WSMessage{
		Type:      "event",
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: WSEventPayload{
			Direction:  "inbound",
			StatusCode: statusCode,
			LatencyMs:  latencyMs,
			Body:       body,
		},
	}
	h.hub.Broadcast(sessionID, event)

	c.JSON(http.StatusOK, gin.H{
		"simulation": map[string]interface{}{
			"failure_type": req.FailureType,
			"status_code":  statusCode,
			"body":         body,
			"latency_ms":   latencyMs,
		},
		"broadcast": true,
	})
}

// PreviewTransform previews a transformation and broadcasts the result
// @Summary Preview a webhook transformation
// @Tags Playground
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200
// @Router /playground/ws/{session_id}/transform [post]
func (h *Handler) PreviewTransform(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req struct {
		Input     map[string]interface{} `json:"input" binding:"required"`
		Template  string                 `json:"template" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	previewer := NewTransformPreviewer()
	inputJSON, _ := json.Marshal(req.Input)
	sid := uuid.New()
	transformFn := func(code string, input json.RawMessage) (json.RawMessage, error) {
		// Simple pass-through transform for preview
		return input, nil
	}
	result := previewer.Preview(sid, req.Template, json.RawMessage(inputJSON), transformFn)

	// Broadcast transform preview
	event := WSMessage{
		Type:      "event",
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"type":   "transform_preview",
			"input":  req.Input,
			"output": result.OutputPayload,
			"error":  result.ErrorMessage,
		},
	}
	h.hub.Broadcast(sessionID, event)

	c.JSON(http.StatusOK, gin.H{
		"output":       result.OutputPayload,
		"success":      result.Success,
		"error":        result.ErrorMessage,
		"execution_ms": result.ExecutionMs,
	})
}
