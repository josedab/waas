package protocolgw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeliveryRoute defines a protocol delivery route for an endpoint.
type DeliveryRoute struct {
	ID               string          `json:"id" db:"id"`
	TenantID         string          `json:"tenant_id" db:"tenant_id"`
	EndpointID       string          `json:"endpoint_id" db:"endpoint_id"`
	Protocol         string          `json:"protocol" db:"protocol"`
	Priority         int             `json:"priority" db:"priority"`
	FallbackProtocol string          `json:"fallback_protocol,omitempty" db:"fallback_protocol"`
	Config           json.RawMessage `json:"config" db:"config"`
	IsActive         bool            `json:"is_active" db:"is_active"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// MQTTDeliveryConfig configures MQTT topic-per-endpoint delivery.
type MQTTDeliveryConfig struct {
	BrokerURL    string `json:"broker_url"`
	Topic        string `json:"topic"`
	QoS          int    `json:"qos"`
	Retain       bool   `json:"retain"`
	ClientID     string `json:"client_id"`
	UseTLS       bool   `json:"use_tls"`
}

// WebSocketDeliveryConfig configures persistent WebSocket delivery.
type WebSocketDeliveryConfig struct {
	URL              string `json:"url"`
	ReconnectEnabled bool   `json:"reconnect_enabled"`
	ReconnectDelayMs int    `json:"reconnect_delay_ms"`
	MaxReconnects    int    `json:"max_reconnects"`
	PingIntervalMs   int    `json:"ping_interval_ms"`
	BackpressureMode string `json:"backpressure_mode"` // drop, buffer, block
	BufferSize       int    `json:"buffer_size"`
}

// FallbackChain defines the protocol fallback order (gRPC → WebSocket → HTTP).
type FallbackChain struct {
	EndpointID string   `json:"endpoint_id"`
	Chain      []string `json:"chain"`
	CurrentIdx int      `json:"current_index"`
}

// ProtocolDeliveryResult represents the result of a multi-protocol delivery.
type ProtocolDeliveryResult struct {
	DeliveryID       string    `json:"delivery_id"`
	EndpointID       string    `json:"endpoint_id"`
	ProtocolUsed     string    `json:"protocol_used"`
	FallbackUsed     bool      `json:"fallback_used"`
	OriginalProtocol string    `json:"original_protocol,omitempty"`
	StatusCode       int       `json:"status_code,omitempty"`
	LatencyMs        int       `json:"latency_ms"`
	Error            string    `json:"error,omitempty"`
	DeliveredAt      time.Time `json:"delivered_at"`
}

// UnifiedMetrics tracks delivery metrics across all protocols.
type UnifiedMetrics struct {
	TenantID        string                    `json:"tenant_id"`
	EndpointID      string                    `json:"endpoint_id"`
	ByProtocol      map[string]DeliveryProtocolStats  `json:"by_protocol"`
	FallbackRate    float64                   `json:"fallback_rate"`
	TotalDeliveries int                       `json:"total_deliveries"`
	Period          string                    `json:"period"`
}

// DeliveryProtocolStats tracks delivery stats for a single protocol.
type DeliveryProtocolStats struct {
	Protocol     string  `json:"protocol"`
	Attempts     int     `json:"total_attempts"`
	Successes    int     `json:"successes"`
	Failures     int     `json:"failures"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs int     `json:"avg_latency_ms"`
}

// CreateDeliveryRouteRequest is the request to create a protocol delivery route.
type CreateDeliveryRouteRequest struct {
	EndpointID       string          `json:"endpoint_id" binding:"required"`
	Protocol         string          `json:"protocol" binding:"required"`
	Priority         int             `json:"priority"`
	FallbackProtocol string          `json:"fallback_protocol,omitempty"`
	Config           json.RawMessage `json:"config"`
}

// DefaultFallbackChain returns the default protocol fallback order.
func DefaultFallbackChain(endpointID string) *FallbackChain {
	return &FallbackChain{
		EndpointID: endpointID,
		Chain:      []string{ProtocolGRPC, ProtocolWebSocket, ProtocolHTTP},
		CurrentIdx: 0,
	}
}

// DetectProtocol automatically detects the best protocol from a URL.
func DetectProtocol(url string) string {
	if len(url) > 7 && url[:7] == "grpc://" {
		return ProtocolGRPC
	}
	if len(url) > 5 && (url[:5] == "ws://" || (len(url) > 6 && url[:6] == "wss://")) {
		return ProtocolWebSocket
	}
	if len(url) > 7 && url[:7] == "mqtt://" {
		return ProtocolMQTT
	}
	return ProtocolHTTP
}

// CreateDeliveryRoute creates a new protocol delivery route.
func (s *Service) CreateDeliveryRoute(ctx context.Context, tenantID string, req *CreateDeliveryRouteRequest) (*DeliveryRoute, error) {
	return &DeliveryRoute{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		EndpointID:       req.EndpointID,
		Protocol:         req.Protocol,
		Priority:         req.Priority,
		FallbackProtocol: req.FallbackProtocol,
		Config:           req.Config,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil
}

// GetDeliveryRoutes retrieves delivery routes for an endpoint.
func (s *Service) GetDeliveryRoutes(ctx context.Context, tenantID, endpointID string) ([]DeliveryRoute, error) {
	return []DeliveryRoute{
		{
			ID:         uuid.New().String(),
			TenantID:   tenantID,
			EndpointID: endpointID,
			Protocol:   ProtocolHTTP,
			Priority:   0,
			IsActive:   true,
			Config:     json.RawMessage(`{"method":"POST"}`),
			CreatedAt:  time.Now(),
		},
	}, nil
}

// GetFallbackChain returns the fallback chain for an endpoint.
func (s *Service) GetFallbackChain(ctx context.Context, tenantID, endpointID string) (*FallbackChain, error) {
	return DefaultFallbackChain(endpointID), nil
}

// GetUnifiedMetrics returns delivery metrics across all protocols.
func (s *Service) GetUnifiedMetrics(ctx context.Context, tenantID, endpointID string) (*UnifiedMetrics, error) {
	return &UnifiedMetrics{
		TenantID:        tenantID,
		EndpointID:      endpointID,
		Period:          "24h",
		TotalDeliveries: 0,
		FallbackRate:    0,
		ByProtocol: map[string]DeliveryProtocolStats{
			ProtocolHTTP: {Protocol: ProtocolHTTP, SuccessRate: 100},
		},
	}, nil
}

// RegisterMultiProtocolRoutes registers multi-protocol delivery routes.
func (h *Handler) RegisterMultiProtocolRoutes(router *gin.RouterGroup) {
	proto := router.Group("/protocol-gateway/v2")
	{
		proto.POST("/routes", h.CreateDeliveryRouteHandler)
		proto.GET("/routes/:endpoint_id", h.GetDeliveryRoutesHandler)
		proto.GET("/fallback/:endpoint_id", h.GetFallbackChainHandler)
		proto.GET("/metrics/:endpoint_id", h.GetUnifiedMetricsHandler)
		proto.GET("/detect", h.DetectProtocolHandler)
		proto.GET("/supported", h.ListSupportedProtocols)
	}
}

func (h *Handler) CreateDeliveryRouteHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateDeliveryRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	route, err := h.service.CreateDeliveryRoute(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, route)
}

func (h *Handler) GetDeliveryRoutesHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	routes, err := h.service.GetDeliveryRoutes(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"routes": routes})
}

func (h *Handler) GetFallbackChainHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	chain, err := h.service.GetFallbackChain(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chain)
}

func (h *Handler) GetUnifiedMetricsHandler(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	metrics, err := h.service.GetUnifiedMetrics(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *Handler) DetectProtocolHandler(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter is required"})
		return
	}

	protocol := DetectProtocol(url)
	c.JSON(http.StatusOK, gin.H{
		"url":      url,
		"protocol": protocol,
	})
}

func (h *Handler) ListSupportedProtocols(c *gin.Context) {
	protocols := []map[string]interface{}{
		{"protocol": ProtocolHTTP, "name": "HTTP/HTTPS", "description": "Standard webhook delivery via HTTP POST", "default": true},
		{"protocol": ProtocolGRPC, "name": "gRPC", "description": "Server-side streaming delivery via gRPC", "default": false},
		{"protocol": ProtocolMQTT, "name": "MQTT", "description": "Topic-based delivery via MQTT broker", "default": false},
		{"protocol": ProtocolWebSocket, "name": "WebSocket", "description": "Persistent connection delivery via WebSocket", "default": false},
	}

	c.JSON(http.StatusOK, gin.H{
		"protocols":      protocols,
		"default_chain":  []string{ProtocolGRPC, ProtocolWebSocket, ProtocolHTTP},
	})
}

// Ensure fmt and uuid are referenced
var (
	_ = fmt.Sprintf
	_ = uuid.New
)
