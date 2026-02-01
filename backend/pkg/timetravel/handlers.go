package timetravel

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/josedab/waas/pkg/errors"
)

// Handler handles time-travel HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new time-travel handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers time-travel routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	tt := r.Group("/timetravel")
	{
		tt.POST("/events", h.RecordEvent)
		tt.GET("/events", h.GetEventTimeline)

		tt.POST("/replay", h.CreateReplayJob)
		tt.GET("/replay", h.ListReplayJobs)
		tt.GET("/replay/:id", h.GetReplayJob)
		tt.POST("/replay/:id/cancel", h.CancelReplay)

		tt.POST("/snapshots", h.CreateSnapshot)
		tt.GET("/snapshots", h.ListSnapshots)
		tt.GET("/snapshots/:id", h.GetSnapshot)
		tt.DELETE("/snapshots/:id", h.DeleteSnapshot)

		tt.POST("/blast-radius", h.AnalyzeBlastRadius)
		tt.POST("/what-if", h.RunWhatIfScenario)
		tt.GET("/what-if", h.ListWhatIfScenarios)
	}
}

// RecordEvent godoc
// @Summary Record event
// @Description Record a webhook event for time-travel replay
// @Tags timetravel
// @Accept json
// @Produce json
// @Param request body EventRecord true "Event record"
// @Success 201 {object} EventRecord
// @Failure 400 {object} map[string]interface{}
// @Router /timetravel/events [post]
func (h *Handler) RecordEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var event EventRecord
	if err := c.ShouldBindJSON(&event); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	result, err := h.service.RecordEvent(c.Request.Context(), tenantID, &event)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetEventTimeline godoc
// @Summary Get event timeline
// @Description Get events with optional filters and pagination
// @Tags timetravel
// @Produce json
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param endpoint_id query string false "Filter by endpoint ID"
// @Param event_type query string false "Filter by event type"
// @Success 200 {object} map[string]interface{}
// @Router /timetravel/events [get]
func (h *Handler) GetEventTimeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil {
			pageSize = parsed
		}
	}

	var filters ReplayFilter
	if eid := c.Query("endpoint_id"); eid != "" {
		filters.EndpointIDs = []string{eid}
	}
	if et := c.Query("event_type"); et != "" {
		filters.EventTypes = []string{et}
	}

	events, total, err := h.service.GetEventTimeline(c.Request.Context(), tenantID, &filters, page, pageSize)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":    events,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CreateReplayJob godoc
// @Summary Create replay job
// @Description Create a new webhook replay job
// @Tags timetravel
// @Accept json
// @Produce json
// @Param request body CreateReplayJobRequest true "Replay job request"
// @Success 201 {object} ReplayJob
// @Failure 400 {object} map[string]interface{}
// @Router /timetravel/replay [post]
func (h *Handler) CreateReplayJob(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateReplayJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	job, err := h.service.CreateReplayJob(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, job)
}

// ListReplayJobs godoc
// @Summary List replay jobs
// @Description List replay jobs for the tenant
// @Tags timetravel
// @Produce json
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /timetravel/replay [get]
func (h *Handler) ListReplayJobs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	jobs, total, err := h.service.repo.ListReplayJobs(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetReplayJob godoc
// @Summary Get replay job
// @Description Get a replay job by ID
// @Tags timetravel
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} ReplayJob
// @Failure 404 {object} map[string]interface{}
// @Router /timetravel/replay/{id} [get]
func (h *Handler) GetReplayJob(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	job, err := h.service.repo.GetReplayJob(c.Request.Context(), tenantID, jobID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	if job == nil {
		pkgerrors.AbortWithNotFound(c, "replay job")
		return
	}

	c.JSON(http.StatusOK, job)
}

// CancelReplay godoc
// @Summary Cancel replay job
// @Description Cancel a pending or running replay job
// @Tags timetravel
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} ReplayJob
// @Failure 400 {object} map[string]interface{}
// @Router /timetravel/replay/{id}/cancel [post]
func (h *Handler) CancelReplay(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	job, err := h.service.CancelReplay(c.Request.Context(), tenantID, jobID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, job)
}

// CreateSnapshot godoc
// @Summary Create snapshot
// @Description Create a point-in-time snapshot
// @Tags timetravel
// @Accept json
// @Produce json
// @Param request body CreateSnapshotRequest true "Snapshot request"
// @Success 201 {object} PointInTimeSnapshot
// @Failure 400 {object} map[string]interface{}
// @Router /timetravel/snapshots [post]
func (h *Handler) CreateSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	snapshot, err := h.service.CreateSnapshot(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, snapshot)
}

// ListSnapshots godoc
// @Summary List snapshots
// @Description List point-in-time snapshots for the tenant
// @Tags timetravel
// @Produce json
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /timetravel/snapshots [get]
func (h *Handler) ListSnapshots(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	snapshots, total, err := h.service.ListSnapshots(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"snapshots": snapshots,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetSnapshot godoc
// @Summary Get snapshot
// @Description Get a point-in-time snapshot by ID
// @Tags timetravel
// @Produce json
// @Param id path string true "Snapshot ID"
// @Success 200 {object} PointInTimeSnapshot
// @Failure 404 {object} map[string]interface{}
// @Router /timetravel/snapshots/{id} [get]
func (h *Handler) GetSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	snapshotID := c.Param("id")

	snapshot, err := h.service.GetSnapshot(c.Request.Context(), tenantID, snapshotID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "snapshot")
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// DeleteSnapshot godoc
// @Summary Delete snapshot
// @Description Delete a point-in-time snapshot
// @Tags timetravel
// @Produce json
// @Param id path string true "Snapshot ID"
// @Success 204
// @Failure 500 {object} map[string]interface{}
// @Router /timetravel/snapshots/{id} [delete]
func (h *Handler) DeleteSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	snapshotID := c.Param("id")

	if err := h.service.repo.DeleteSnapshot(c.Request.Context(), tenantID, snapshotID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

// AnalyzeBlastRadius godoc
// @Summary Analyze blast radius
// @Description Calculate the impact of an endpoint going down
// @Tags timetravel
// @Accept json
// @Produce json
// @Param request body object true "Blast radius request with endpoint_id"
// @Success 200 {object} BlastRadiusAnalysis
// @Failure 400 {object} map[string]interface{}
// @Router /timetravel/blast-radius [post]
func (h *Handler) AnalyzeBlastRadius(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req struct {
		EndpointID string `json:"endpoint_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	analysis, err := h.service.AnalyzeBlastRadius(c.Request.Context(), tenantID, req.EndpointID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// RunWhatIfScenario godoc
// @Summary Run what-if scenario
// @Description Compare original vs modified payload and analyze impact
// @Tags timetravel
// @Accept json
// @Produce json
// @Param request body WhatIfRequest true "What-if request"
// @Success 201 {object} WhatIfScenario
// @Failure 400 {object} map[string]interface{}
// @Router /timetravel/what-if [post]
func (h *Handler) RunWhatIfScenario(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req WhatIfRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	scenario, err := h.service.RunWhatIfScenario(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, scenario)
}

// ListWhatIfScenarios godoc
// @Summary List what-if scenarios
// @Description List what-if scenarios for the tenant
// @Tags timetravel
// @Produce json
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /timetravel/what-if [get]
func (h *Handler) ListWhatIfScenarios(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	scenarios, total, err := h.service.repo.GetWhatIfScenarios(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scenarios": scenarios,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}
