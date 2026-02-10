package multiregion

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ActiveActiveHandler provides HTTP handlers for active-active and failover
type ActiveActiveHandler struct {
	service *ActiveActiveService
}

// NewActiveActiveHandler creates a new handler
func NewActiveActiveHandler(service *ActiveActiveService) *ActiveActiveHandler {
	return &ActiveActiveHandler{service: service}
}

// RegisterRoutes registers active-active routes
func (h *ActiveActiveHandler) RegisterRoutes(r *gin.RouterGroup) {
	mr := r.Group("/multiregion")
	{
		aa := mr.Group("/active-active")
		{
			aa.POST("/config", h.CreateActiveActiveConfig)
			aa.GET("/config", h.GetActiveActiveConfig)
			aa.POST("/select-region", h.SelectRegion)
			aa.POST("/sync-event", h.SyncEvent)
			aa.GET("/events", h.ListCrossRegionEvents)
		}

		fo := mr.Group("/failover")
		{
			fo.POST("/config", h.CreateFailoverConfig)
			fo.GET("/config", h.GetFailoverConfig)
		}
	}
}

// CreateActiveActiveConfig creates active-active configuration
// @Summary Create active-active config
// @Tags multiregion
// @Accept json
// @Produce json
// @Success 201 {object} ActiveActiveConfig
// @Router /multiregion/active-active/config [post]
func (h *ActiveActiveHandler) CreateActiveActiveConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateActiveActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetActiveActiveConfig retrieves active-active configuration
// @Summary Get active-active config
// @Tags multiregion
// @Produce json
// @Success 200 {object} ActiveActiveConfig
// @Router /multiregion/active-active/config [get]
func (h *ActiveActiveHandler) GetActiveActiveConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	config, err := h.service.GetActiveActiveConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// SelectRegion selects the best region for delivery
// @Summary Select delivery region
// @Tags multiregion
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /multiregion/active-active/select-region [post]
func (h *ActiveActiveHandler) SelectRegion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	region, err := h.service.SelectRegion(c.Request.Context(), tenantID, req.Latitude, req.Longitude)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"selected_region": region})
}

// SyncEvent synchronizes an event across regions
// @Summary Sync cross-region event
// @Tags multiregion
// @Accept json
// @Produce json
// @Success 201 {object} CrossRegionEvent
// @Router /multiregion/active-active/sync-event [post]
func (h *ActiveActiveHandler) SyncEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		SourceRegion string      `json:"source_region" binding:"required"`
		EventType    string      `json:"event_type" binding:"required"`
		Payload      interface{} `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payloadBytes, _ := json.Marshal(req.Payload)

	event, err := h.service.SyncEvent(c.Request.Context(), tenantID, req.SourceRegion, req.EventType, payloadBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// ListCrossRegionEvents lists cross-region sync events
// @Summary List cross-region events
// @Tags multiregion
// @Produce json
// @Success 200 {array} CrossRegionEvent
// @Router /multiregion/active-active/events [get]
func (h *ActiveActiveHandler) ListCrossRegionEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	events, err := h.service.ListCrossRegionEvents(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// CreateFailoverConfig creates failover configuration
// @Summary Create failover config
// @Tags multiregion
// @Accept json
// @Produce json
// @Success 201 {object} FailoverConfig
// @Router /multiregion/failover/config [post]
func (h *ActiveActiveHandler) CreateFailoverConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateFailoverConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateFailoverConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetFailoverConfig retrieves failover configuration
// @Summary Get failover config
// @Tags multiregion
// @Produce json
// @Success 200 {object} FailoverConfig
// @Router /multiregion/failover/config [get]
func (h *ActiveActiveHandler) GetFailoverConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	config, err := h.service.GetFailoverConfig(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}
