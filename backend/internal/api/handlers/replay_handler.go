package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/josedab/waas/pkg/replay"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
)

// ReplayHandler handles webhook replay HTTP requests
type ReplayHandler struct {
	service *replay.Service
	logger  *utils.Logger
}

// NewReplayHandler creates a new replay handler
func NewReplayHandler(service *replay.Service, logger *utils.Logger) *ReplayHandler {
	return &ReplayHandler{
		service: service,
		logger:  logger,
	}
}

// SingleReplayRequest represents a single replay request
type SingleReplayRequest struct {
	DeliveryID    string `json:"delivery_id" binding:"required"`
	EndpointID    string `json:"endpoint_id,omitempty"`
	ModifyPayload bool   `json:"modify_payload,omitempty"`
}

// BulkReplayRequest represents a bulk replay request
type BulkReplayRequest struct {
	DeliveryIDs []string `json:"delivery_ids,omitempty"`
	EndpointID  string   `json:"endpoint_id,omitempty"`
	Status      string   `json:"status,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	DryRun      bool     `json:"dry_run,omitempty"`
}

// TimeRangeReplayRequest represents a time-range replay request
type TimeRangeReplayRequest struct {
	EndpointID string `json:"endpoint_id"`
	StartTime  string `json:"start_time" binding:"required"`
	EndTime    string `json:"end_time" binding:"required"`
	Status     string `json:"status"` // filter by original status
}

// CreateSnapshotRequest represents a snapshot creation request
type CreateSnapshotRequest struct {
	EndpointID  string `json:"endpoint_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	StartTime   string `json:"start_time" binding:"required"`
	EndTime     string `json:"end_time" binding:"required"`
}

// ReplayFromSnapshotRequest represents replaying from a snapshot
type ReplayFromSnapshotRequest struct {
	TargetEndpointID string `json:"target_endpoint_id"`
}

// ReplayDelivery replays a single delivery
// @Summary Replay single delivery
// @Tags replay
// @Accept json
// @Produce json
// @Param request body SingleReplayRequest true "Replay request"
// @Success 201 {object} replay.Replay
// @Router /replay [post]
func (h *ReplayHandler) ReplayDelivery(c *gin.Context) {
	var req SingleReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	replayReq := &replay.ReplayRequest{
		DeliveryID:    req.DeliveryID,
		EndpointID:    req.EndpointID,
		ModifyPayload: req.ModifyPayload,
	}

	r, err := h.service.ReplaySingle(c.Request.Context(), tenantID, replayReq)
	if err != nil {
		h.logger.Error("Failed to replay delivery", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, r)
}

// BulkReplay replays multiple deliveries
// @Summary Bulk replay deliveries
// @Tags replay
// @Accept json
// @Produce json
// @Param request body BulkReplayRequest true "Bulk replay request"
// @Success 201 {object} map[string]interface{}
// @Router /replay/bulk [post]
func (h *ReplayHandler) BulkReplay(c *gin.Context) {
	var req BulkReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bulkReq := &replay.BulkReplayRequest{
		DeliveryIDs: req.DeliveryIDs,
		EndpointID:  req.EndpointID,
		Status:      req.Status,
		Limit:       req.Limit,
		DryRun:      req.DryRun,
	}

	result, err := h.service.ReplayBulk(c.Request.Context(), tenantID, bulkReq)
	if err != nil {
		h.logger.Error("Failed to bulk replay", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ReplayTimeRange replays deliveries within a time range
// @Summary Replay by time range
// @Tags replay
// @Accept json
// @Produce json
// @Param request body TimeRangeReplayRequest true "Time range request"
// @Success 201 {object} map[string]interface{}
// @Router /replay/time-range [post]
func (h *ReplayHandler) ReplayTimeRange(c *gin.Context) {
	var req TimeRangeReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format"})
		return
	}

	bulkReq := &replay.BulkReplayRequest{
		EndpointID: req.EndpointID,
		Status:     req.Status,
		StartTime:  startTime,
		EndTime:    endTime,
	}

	result, err := h.service.ReplayBulk(c.Request.Context(), tenantID, bulkReq)
	if err != nil {
		h.logger.Error("Failed to replay time range", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetReplay gets replay details - returns not implemented for now
// @Summary Get replay details
// @Tags replay
// @Produce json
// @Param id path string true "Replay ID"
// @Success 200 {object} map[string]interface{}
// @Router /replay/{id} [get]
func (h *ReplayHandler) GetReplay(c *gin.Context) {
	// Replay results are returned immediately, no persistent replay records yet
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// ListReplays lists replays for a tenant - returns not implemented for now
// @Summary List replays
// @Tags replay
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} map[string]interface{}
// @Router /replay [get]
func (h *ReplayHandler) ListReplays(c *gin.Context) {
	// Replay results are returned immediately, no persistent replay records yet
	c.JSON(http.StatusOK, []interface{}{})
}

// CreateSnapshot creates a point-in-time snapshot
// @Summary Create snapshot
// @Tags replay
// @Accept json
// @Produce json
// @Param request body CreateSnapshotRequest true "Snapshot request"
// @Success 201 {object} replay.Snapshot
// @Router /replay/snapshots [post]
func (h *ReplayHandler) CreateSnapshot(c *gin.Context) {
	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format"})
		return
	}

	createReq := &replay.CreateSnapshotRequest{
		Name:        req.Name,
		Description: req.Description,
		Filters: replay.SnapshotFilters{
			EndpointID: req.EndpointID,
			StartTime:  startTime,
			EndTime:    endTime,
		},
	}

	snapshot, err := h.service.CreateSnapshot(c.Request.Context(), tenantID, createReq)
	if err != nil {
		h.logger.Error("Failed to create snapshot", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, snapshot)
}

// ListSnapshots lists snapshots for a tenant
// @Summary List snapshots
// @Tags replay
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} replay.Snapshot
// @Router /replay/snapshots [get]
func (h *ReplayHandler) ListSnapshots(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	snapshots, _, err := h.service.ListSnapshots(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshots)
}

// GetSnapshot gets a snapshot
// @Summary Get snapshot
// @Tags replay
// @Produce json
// @Param id path string true "Snapshot ID"
// @Success 200 {object} replay.Snapshot
// @Router /replay/snapshots/{id} [get]
func (h *ReplayHandler) GetSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	snapshot, err := h.service.GetSnapshot(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// ReplayFromSnapshot replays deliveries from a snapshot
// @Summary Replay from snapshot
// @Tags replay
// @Accept json
// @Produce json
// @Param id path string true "Snapshot ID"
// @Param request body ReplayFromSnapshotRequest false "Replay options"
// @Success 201 {object} map[string]interface{}
// @Router /replay/snapshots/{id}/replay [post]
func (h *ReplayHandler) ReplayFromSnapshot(c *gin.Context) {
	id := c.Param("id")

	var req ReplayFromSnapshotRequest
	c.ShouldBindJSON(&req) // Optional body

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	replayReq := &replay.ReplayFromSnapshotRequest{
		SnapshotID: id,
	}

	result, err := h.service.ReplayFromSnapshot(c.Request.Context(), tenantID, replayReq)
	if err != nil {
		h.logger.Error("Failed to replay from snapshot", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// DeleteSnapshot deletes a snapshot
// @Summary Delete snapshot
// @Tags replay
// @Param id path string true "Snapshot ID"
// @Success 204
// @Router /replay/snapshots/{id} [delete]
func (h *ReplayHandler) DeleteSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.DeleteSnapshot(c.Request.Context(), tenantID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterReplayRoutes registers replay routes
func RegisterReplayRoutes(r *gin.RouterGroup, h *ReplayHandler) {
	rp := r.Group("/replay")
	{
		// Single and bulk replay
		rp.POST("", h.ReplayDelivery)
		rp.POST("/bulk", h.BulkReplay)
		rp.POST("/time-range", h.ReplayTimeRange)
		rp.GET("", h.ListReplays)
		rp.GET("/:id", h.GetReplay)

		// Snapshots
		rp.POST("/snapshots", h.CreateSnapshot)
		rp.GET("/snapshots", h.ListSnapshots)
		rp.GET("/snapshots/:id", h.GetSnapshot)
		rp.POST("/snapshots/:id/replay", h.ReplayFromSnapshot)
		rp.DELETE("/snapshots/:id", h.DeleteSnapshot)
	}
}
