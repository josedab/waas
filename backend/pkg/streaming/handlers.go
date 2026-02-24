package streaming

import (
	"github.com/josedab/waas/pkg/httputil"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for streaming bridges
type Handler struct {
	service *Service
}

// NewHandler creates a new streaming handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers streaming routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	streaming := r.Group("/streaming")
	{
		streaming.POST("/bridges", h.CreateBridge)
		streaming.GET("/bridges", h.ListBridges)
		streaming.GET("/bridges/:id", h.GetBridge)
		streaming.PUT("/bridges/:id", h.UpdateBridge)
		streaming.DELETE("/bridges/:id", h.DeleteBridge)
		streaming.POST("/bridges/:id/send", h.SendToStream)
		streaming.GET("/bridges/:id/metrics", h.GetBridgeMetrics)
		streaming.POST("/bridges/test-connection", h.TestConnection)
	}
}

// CreateBridge creates a new streaming bridge
// @Summary Create streaming bridge
// @Description Create a new streaming bridge to connect webhooks with Kafka/Kinesis/Pulsar
// @Tags streaming
// @Accept json
// @Produce json
// @Param request body CreateBridgeRequest true "Bridge configuration"
// @Success 201 {object} StreamingBridge
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /streaming/bridges [post]
func (h *Handler) CreateBridge(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bridge, err := h.service.CreateBridge(c.Request.Context(), tenantID, &req)
	if err != nil {
		if err == ErrBridgeAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if err == ErrInvalidConfig {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, bridge)
}

// GetBridge retrieves a streaming bridge
// @Summary Get streaming bridge
// @Description Get details of a streaming bridge
// @Tags streaming
// @Produce json
// @Param id path string true "Bridge ID"
// @Success 200 {object} StreamingBridge
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /streaming/bridges/{id} [get]
func (h *Handler) GetBridge(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Param("id")
	bridge, err := h.service.GetBridge(c.Request.Context(), tenantID, bridgeID)
	if err != nil {
		if err == ErrBridgeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "bridge not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, bridge)
}

// UpdateBridge updates a streaming bridge
// @Summary Update streaming bridge
// @Description Update a streaming bridge configuration
// @Tags streaming
// @Accept json
// @Produce json
// @Param id path string true "Bridge ID"
// @Param request body UpdateBridgeRequest true "Update request"
// @Success 200 {object} StreamingBridge
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /streaming/bridges/{id} [put]
func (h *Handler) UpdateBridge(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Param("id")

	var req UpdateBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bridge, err := h.service.UpdateBridge(c.Request.Context(), tenantID, bridgeID, &req)
	if err != nil {
		if err == ErrBridgeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "bridge not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, bridge)
}

// DeleteBridge deletes a streaming bridge
// @Summary Delete streaming bridge
// @Description Delete a streaming bridge
// @Tags streaming
// @Param id path string true "Bridge ID"
// @Success 204 "No content"
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /streaming/bridges/{id} [delete]
func (h *Handler) DeleteBridge(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Param("id")
	err := h.service.DeleteBridge(c.Request.Context(), tenantID, bridgeID)
	if err != nil {
		if err == ErrBridgeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "bridge not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListBridges lists streaming bridges
// @Summary List streaming bridges
// @Description List all streaming bridges for the tenant
// @Tags streaming
// @Produce json
// @Param stream_type query string false "Filter by stream type"
// @Param direction query string false "Filter by direction"
// @Param status query string false "Filter by status"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListBridgesResponse
// @Security ApiKeyAuth
// @Router /streaming/bridges [get]
func (h *Handler) ListBridges(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	filters := &BridgeFilters{}

	if st := c.Query("stream_type"); st != "" {
		streamType := StreamType(st)
		filters.StreamType = &streamType
	}
	if dir := c.Query("direction"); dir != "" {
		direction := Direction(dir)
		filters.Direction = &direction
	}
	if status := c.Query("status"); status != "" {
		bridgeStatus := BridgeStatus(status)
		filters.Status = &bridgeStatus
	}
	if search := c.Query("search"); search != "" {
		filters.Search = search
	}
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		filters.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil && pageSize > 0 {
		filters.PageSize = pageSize
	}

	response, err := h.service.ListBridges(c.Request.Context(), tenantID, filters)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// SendToStream sends an event to a streaming platform
// @Summary Send to stream
// @Description Send a webhook event to the configured streaming platform
// @Tags streaming
// @Accept json
// @Produce json
// @Param id path string true "Bridge ID"
// @Param request body map[string]interface{} true "Event payload"
// @Success 202 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /streaming/bridges/{id}/send [post]
func (h *Handler) SendToStream(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Param("id")

	var req struct {
		Payload interface{}       `json:"payload"`
		Headers map[string]string `json:"headers,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payloadBytes, _ := json.Marshal(req.Payload)

	err := h.service.SendToStream(c.Request.Context(), tenantID, bridgeID, payloadBytes, req.Headers)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
}

// GetBridgeMetrics retrieves metrics for a bridge
// @Summary Get bridge metrics
// @Description Get operational metrics for a streaming bridge
// @Tags streaming
// @Produce json
// @Param id path string true "Bridge ID"
// @Success 200 {object} BridgeMetrics
// @Security ApiKeyAuth
// @Router /streaming/bridges/{id}/metrics [get]
func (h *Handler) GetBridgeMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bridgeID := c.Param("id")
	metrics, err := h.service.GetBridgeMetrics(c.Request.Context(), tenantID, bridgeID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// TestConnection tests connectivity to a streaming platform
// @Summary Test connection
// @Description Test connectivity to a streaming platform before creating a bridge
// @Tags streaming
// @Accept json
// @Produce json
// @Param request body CreateBridgeRequest true "Connection configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /streaming/bridges/test-connection [post]
func (h *Handler) TestConnection(c *gin.Context) {
	var req CreateBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.TestConnection(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection successful",
	})
}
