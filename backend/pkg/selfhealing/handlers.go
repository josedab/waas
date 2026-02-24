package selfhealing

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for self-healing.
type Handler struct {
	service *Service
}

// NewHandler creates a new self-healing handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all self-healing routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/self-healing")
	{
		group.POST("/record-failure", h.RecordFailure)
		group.POST("/record-success", h.RecordSuccess)
		group.POST("/discover", h.DiscoverNewURL)
		group.POST("/validate/:id", h.ValidateAndApply)
		group.GET("/discoveries/:endpoint_id", h.GetDiscoveries)
		group.GET("/failures/:endpoint_id", h.GetFailureStatus)
		group.GET("/events", h.GetMigrationEvents)
		group.POST("/well-known", h.GenerateWellKnownSpec)
	}
}

// RecordFailure records a delivery failure.
// @Summary Record delivery failure
// @Tags self-healing
// @Accept json
// @Produce json
// @Router /self-healing/record-failure [post]
func (h *Handler) RecordFailure(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
		CurrentURL string `json:"current_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	discovery, err := h.service.RecordFailure(tenantID, req.EndpointID, req.CurrentURL)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	if discovery != nil {
		c.JSON(http.StatusOK, gin.H{"healing_triggered": true, "discovery": discovery})
	} else {
		c.JSON(http.StatusOK, gin.H{"healing_triggered": false})
	}
}

// RecordSuccess records a successful delivery.
// @Summary Record delivery success
// @Tags self-healing
// @Accept json
// @Produce json
// @Router /self-healing/record-success [post]
func (h *Handler) RecordSuccess(c *gin.Context) {
	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RecordSuccess(req.EndpointID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reset"})
}

// DiscoverNewURL attempts to discover a new URL.
// @Summary Discover new endpoint URL
// @Tags self-healing
// @Accept json
// @Produce json
// @Router /self-healing/discover [post]
func (h *Handler) DiscoverNewURL(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
		CurrentURL string `json:"current_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	discovery, err := h.service.DiscoverNewURL(tenantID, req.EndpointID, req.CurrentURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, discovery)
}

// ValidateAndApply validates and applies a discovery.
// @Summary Validate and apply discovered URL
// @Tags self-healing
// @Param id path string true "Discovery ID"
// @Produce json
// @Router /self-healing/validate/{id} [post]
func (h *Handler) ValidateAndApply(c *gin.Context) {
	discovery, err := h.service.ValidateAndApply(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, discovery)
}

// GetDiscoveries returns discoveries for an endpoint.
// @Summary List discoveries for endpoint
// @Tags self-healing
// @Param endpoint_id path string true "Endpoint ID"
// @Produce json
// @Router /self-healing/discoveries/{endpoint_id} [get]
func (h *Handler) GetDiscoveries(c *gin.Context) {
	discoveries, err := h.service.GetDiscoveries(c.Param("endpoint_id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, discoveries)
}

// GetFailureStatus returns failure tracking status.
// @Summary Get failure status
// @Tags self-healing
// @Param endpoint_id path string true "Endpoint ID"
// @Produce json
// @Router /self-healing/failures/{endpoint_id} [get]
func (h *Handler) GetFailureStatus(c *gin.Context) {
	ft, err := h.service.GetFailureStatus(c.Param("endpoint_id"))
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, ft)
}

// GetMigrationEvents returns recent migration events.
// @Summary List migration events
// @Tags self-healing
// @Param limit query int false "Max results"
// @Produce json
// @Router /self-healing/events [get]
func (h *Handler) GetMigrationEvents(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	events, err := h.service.GetMigrationEvents(tenantID, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, events)
}

// GenerateWellKnownSpec generates a .well-known spec.
// @Summary Generate .well-known/waas-webhooks spec
// @Tags self-healing
// @Accept json
// @Produce json
// @Router /self-healing/well-known [post]
func (h *Handler) GenerateWellKnownSpec(c *gin.Context) {
	var req struct {
		Endpoints []WellKnownEndpoint `json:"endpoints" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	spec := h.service.GenerateWellKnownSpec(req.Endpoints)
	c.JSON(http.StatusOK, spec)
}
