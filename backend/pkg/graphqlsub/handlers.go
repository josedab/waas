package graphqlsub

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Handler provides HTTP and WebSocket handlers for GraphQL subscriptions
type Handler struct {
	service  *Service
	upgrader websocket.Upgrader
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			CheckOrigin:       utils.CheckWebSocketOrigin(),
			EnableCompression: true,
		},
	}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	graphql := r.Group("/graphql")
	{
		// WebSocket endpoint for subscriptions
		graphql.GET("/subscriptions", h.HandleWebSocket)

		// HTTP endpoints
		graphql.GET("/schema", h.GetSchema)
		graphql.GET("/stats", h.GetStats)
		graphql.POST("/publish", h.PublishEvent)
	}
}

// HandleWebSocket handles WebSocket connections for GraphQL subscriptions
// @Summary GraphQL subscription WebSocket endpoint
// @Tags graphql
// @Router /graphql/subscriptions [get]
func (h *Handler) HandleWebSocket(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = c.Query("tenant_id")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	// Determine protocol
	protocol := c.GetHeader("Sec-WebSocket-Protocol")
	if protocol == "" {
		protocol = "graphql-transport-ws"
	}

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, http.Header{
		"Sec-WebSocket-Protocol": {protocol},
	})
	if err != nil {
		return
	}
	defer conn.Close()

	// Register client
	client, err := h.service.RegisterClient(tenantID, protocol)
	if err != nil {
		conn.WriteJSON(&GraphQLMessage{Type: MessageTypeError, Payload: mustMarshal(map[string]string{"message": err.Error()})})
		return
	}
	defer h.service.UnregisterClient(client.ID)

	// Start goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine
	go func() {
		defer wg.Done()
		h.writeLoop(conn, client)
	}()

	// Reader goroutine
	go func() {
		defer wg.Done()
		h.readLoop(c.Request.Context(), conn, client)
	}()

	wg.Wait()
}

// readLoop handles incoming WebSocket messages
func (h *Handler) readLoop(ctx context.Context, conn *websocket.Conn, client *Client) {
	config := h.service.config
	conn.SetReadLimit(config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(config.ReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(config.ReadTimeout))
		return nil
	})

	for {
		select {
		case <-client.CloseCh:
			return
		default:
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg GraphQLMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.sendError(client, "", "Invalid message format")
			continue
		}

		h.handleMessage(ctx, client, &msg)
	}
}

// writeLoop handles outgoing WebSocket messages
func (h *Handler) writeLoop(conn *websocket.Conn, client *Client) {
	config := h.service.config
	ticker := time.NewTicker(config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-client.CloseCh:
			return
		case msg := <-client.SendCh:
			conn.SetWriteDeadline(time.Now().Add(config.WriteTimeout))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(config.WriteTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes a GraphQL WebSocket message
func (h *Handler) handleMessage(ctx context.Context, client *Client, msg *GraphQLMessage) {
	switch msg.Type {
	case MessageTypeConnectionInit:
		h.handleConnectionInit(ctx, client, msg)
	case MessageTypePing:
		h.handlePing(client)
	case MessageTypeSubscribe:
		h.handleSubscribe(ctx, client, msg)
	case MessageTypeComplete:
		h.handleComplete(ctx, client, msg)
	case MessageTypeConnectionTerminate:
		close(client.CloseCh)
	}
}

// handleConnectionInit handles connection initialization
func (h *Handler) handleConnectionInit(ctx context.Context, client *Client, msg *GraphQLMessage) {
	// Authenticate if needed
	if h.service.auth != nil && msg.Payload != nil {
		var initPayload map[string]interface{}
		if err := json.Unmarshal(msg.Payload, &initPayload); err == nil {
			if token, ok := initPayload["Authorization"].(string); ok {
				tenantID, err := h.service.auth.Authenticate(ctx, token)
				if err != nil {
					h.sendError(client, "", "Unauthorized")
					close(client.CloseCh)
					return
				}
				client.TenantID = tenantID
			}
		}
	}

	// Send acknowledgment
	ackMsg := &GraphQLMessage{Type: MessageTypeConnectionAck}
	data, _ := json.Marshal(ackMsg)
	client.SendCh <- data
}

// handlePing handles ping messages
func (h *Handler) handlePing(client *Client) {
	now := time.Now()
	client.LastPingAt = &now
	pongMsg := &GraphQLMessage{Type: MessageTypePong}
	data, _ := json.Marshal(pongMsg)
	client.SendCh <- data
}

// handleSubscribe handles subscription requests
func (h *Handler) handleSubscribe(ctx context.Context, client *Client, msg *GraphQLMessage) {
	var payload SubscriptionPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.sendError(client, msg.ID, "Invalid subscription payload")
		return
	}

	if err := h.service.Subscribe(ctx, client.ID, msg.ID, &payload); err != nil {
		h.sendError(client, msg.ID, err.Error())
		return
	}
}

// handleComplete handles subscription completion
func (h *Handler) handleComplete(ctx context.Context, client *Client, msg *GraphQLMessage) {
	_ = h.service.Unsubscribe(ctx, client.ID, msg.ID)
}

// sendError sends an error message to the client
func (h *Handler) sendError(client *Client, id, message string) {
	errMsg := &GraphQLMessage{
		ID:   id,
		Type: MessageTypeError,
		Payload: mustMarshal([]GQLError{{
			Message: message,
		}}),
	}
	data, _ := json.Marshal(errMsg)
	select {
	case client.SendCh <- data:
	default:
	}
}

// GetSchema returns the GraphQL subscription schema
// @Summary Get GraphQL subscription schema
// @Tags graphql
// @Produce json
// @Success 200 {object} Schema
// @Router /graphql/schema [get]
func (h *Handler) GetSchema(c *gin.Context) {
	schema := h.service.GetSchema()
	c.JSON(http.StatusOK, schema)
}

// GetStats returns subscription statistics
// @Summary Get subscription statistics
// @Tags graphql
// @Produce json
// @Success 200 {object} ClientStats
// @Router /graphql/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// PublishEvent publishes an event to subscriptions (for internal use)
// @Summary Publish event to subscriptions
// @Tags graphql
// @Accept json
// @Produce json
// @Param request body Event true "Event to publish"
// @Success 200
// @Router /graphql/publish [post]
func (h *Handler) PublishEvent(c *gin.Context) {
	var event Event
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if event.TenantID == "" {
		event.TenantID = c.GetString("tenant_id")
		if event.TenantID == "" {
			event.TenantID = "default"
		}
	}

	event.Timestamp = time.Now()

	if err := h.service.PublishEvent(c.Request.Context(), &event); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "published"})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
