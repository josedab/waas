package pushbridge

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PushGatewayHandler provides HTTP handlers for push gateway
type PushGatewayHandler struct {
	service *PushGatewayService
}

// NewPushGatewayHandler creates a new handler
func NewPushGatewayHandler(service *PushGatewayService) *PushGatewayHandler {
	return &PushGatewayHandler{service: service}
}

// RegisterPushRoutes registers push gateway routes
func (h *PushGatewayHandler) RegisterPushRoutes(r *gin.RouterGroup) {
	push := r.Group("/push")
	{
		push.GET("/dashboard", h.GetDashboard)
		push.POST("/send", h.SendPush)

		gw := push.Group("/gateways")
		{
			gw.POST("", h.CreateGateway)
			gw.GET("", h.ListGateways)
			gw.GET("/:gatewayId", h.GetGateway)
			gw.DELETE("/:gatewayId", h.DeleteGateway)
		}

		offline := push.Group("/offline")
		{
			offline.POST("/queue", h.QueueForOffline)
			offline.POST("/drain/:deviceId", h.DrainOfflineQueue)
		}
	}
}

// GetDashboard retrieves the mobile push dashboard
// @Summary Get push dashboard
// @Tags push
// @Produce json
// @Success 200 {object} MobileDashboard
// @Router /push/dashboard [get]
func (h *PushGatewayHandler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// SendPush sends a push notification
// @Summary Send push notification
// @Tags push
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /push/send [post]
func (h *PushGatewayHandler) SendPush(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req GatewaySendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sent, failed, err := h.service.SendPush(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sent":   sent,
		"failed": failed,
		"total":  len(req.DeviceIDs),
	})
}

// CreateGateway creates a push gateway
// @Summary Create push gateway
// @Tags push
// @Accept json
// @Produce json
// @Success 201 {object} PushGateway
// @Router /push/gateways [post]
func (h *PushGatewayHandler) CreateGateway(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreatePushGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gateway, err := h.service.CreateGateway(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gateway)
}

// ListGateways lists push gateways
// @Summary List push gateways
// @Tags push
// @Produce json
// @Success 200 {array} PushGateway
// @Router /push/gateways [get]
func (h *PushGatewayHandler) ListGateways(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	gateways, err := h.service.ListGateways(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gateways)
}

// GetGateway retrieves a push gateway
// @Summary Get push gateway
// @Tags push
// @Produce json
// @Param gatewayId path string true "Gateway ID"
// @Success 200 {object} PushGateway
// @Router /push/gateways/{gatewayId} [get]
func (h *PushGatewayHandler) GetGateway(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	gatewayID := c.Param("gatewayId")
	gateway, err := h.service.GetGateway(c.Request.Context(), tenantID, gatewayID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "gateway not found"})
		return
	}

	c.JSON(http.StatusOK, gateway)
}

// DeleteGateway deletes a push gateway
// @Summary Delete push gateway
// @Tags push
// @Param gatewayId path string true "Gateway ID"
// @Success 204 "No content"
// @Router /push/gateways/{gatewayId} [delete]
func (h *PushGatewayHandler) DeleteGateway(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	gatewayID := c.Param("gatewayId")
	if err := h.service.DeleteGateway(c.Request.Context(), tenantID, gatewayID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// QueueForOffline queues an event for an offline device
// @Summary Queue for offline delivery
// @Tags push
// @Accept json
// @Success 202 {object} map[string]interface{}
// @Router /push/offline/queue [post]
func (h *PushGatewayHandler) QueueForOffline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		DeviceID  string `json:"device_id" binding:"required"`
		WebhookID string `json:"webhook_id" binding:"required"`
		EventType string `json:"event_type" binding:"required"`
		Payload   string `json:"payload" binding:"required"`
		Priority  int    `json:"priority,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.QueueForOffline(c.Request.Context(), tenantID, req.DeviceID, req.WebhookID, req.EventType, req.Payload, req.Priority); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
}

// DrainOfflineQueue delivers queued events for a device
// @Summary Drain offline queue
// @Tags push
// @Produce json
// @Param deviceId path string true "Device ID"
// @Param limit query int false "Limit"
// @Success 200 {array} OfflineQueueEntry
// @Router /push/offline/drain/{deviceId} [post]
func (h *PushGatewayHandler) DrainOfflineQueue(c *gin.Context) {
	deviceID := c.Param("deviceId")
	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	entries, err := h.service.DrainOfflineQueue(c.Request.Context(), deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entries)
}
