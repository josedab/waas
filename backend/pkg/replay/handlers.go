package replay

import (
	"github.com/josedab/waas/pkg/httputil"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for replay operations
type Handler struct {
	service *Service
}

// NewHandler creates a new replay handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers replay routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	replay := router.Group("/replay")
	{
		replay.POST("/single", h.ReplaySingle)
		replay.POST("/bulk", h.ReplayBulk)
		replay.POST("/what-if", h.WhatIf)

		// Snapshots
		replay.POST("/snapshots", h.CreateSnapshot)
		replay.GET("/snapshots", h.ListSnapshots)
		replay.GET("/snapshots/:id", h.GetSnapshot)
		replay.POST("/snapshots/:id/replay", h.ReplayFromSnapshot)
		replay.DELETE("/snapshots/:id", h.DeleteSnapshot)

		// Archives
		replay.GET("/deliveries/:id", h.GetDeliveryArchive)
	}
}

// ReplaySingle replays a single delivery
func (h *Handler) ReplaySingle(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req ReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	result, err := h.service.ReplaySingle(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ReplayBulk replays multiple deliveries
func (h *Handler) ReplayBulk(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req BulkReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	result, err := h.service.ReplayBulk(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// WhatIf runs a what-if scenario — simulates delivery without actually sending
func (h *Handler) WhatIf(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req WhatIfRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	result, err := h.service.RunWhatIf(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateSnapshot creates a point-in-time snapshot
func (h *Handler) CreateSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	snapshot, err := h.service.CreateSnapshot(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, snapshot)
}

// ListSnapshots lists all snapshots for a tenant
func (h *Handler) ListSnapshots(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	snapshots, _, err := h.service.ListSnapshots(c.Request.Context(), tenantID, 100, 0)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

// GetSnapshot retrieves a snapshot by ID
func (h *Handler) GetSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	snapshot, err := h.service.GetSnapshot(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// ReplayFromSnapshot replays deliveries captured in a snapshot
func (h *Handler) ReplayFromSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	result, err := h.service.ReplayFromSnapshot(c.Request.Context(), tenantID, &ReplayFromSnapshotRequest{
		SnapshotID: c.Param("id"),
		Limit:      limit,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteSnapshot deletes a snapshot
func (h *Handler) DeleteSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.service.DeleteSnapshot(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDeliveryArchive retrieves the full archived delivery with payload
func (h *Handler) GetDeliveryArchive(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	archive, err := h.service.GetDeliveryForReplay(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, archive)
}
