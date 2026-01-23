package debugger

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for the webhook debugger
type Handler struct {
	service *Service
}

// NewHandler creates a new debugger handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers debugger routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	dbg := router.Group("/debugger")
	{
		// Traces
		dbg.GET("/traces", h.ListTraces)
		dbg.GET("/traces/:delivery_id", h.GetTrace)
		dbg.GET("/traces/:delivery_id/diff", h.DiffPayloads)

		// Replay
		dbg.POST("/replay", h.ReplayWithMods)
		dbg.POST("/replay/bulk", h.BulkReplay)

		// Debug sessions
		dbg.POST("/sessions", h.CreateDebugSession)
		dbg.GET("/sessions", h.ListDebugSessions)
		dbg.GET("/sessions/:id", h.GetDebugSession)
		dbg.POST("/sessions/:id/step", h.StepDebugSession)
	}
}

// @Summary List delivery traces
// @Tags Debugger
// @Produce json
// @Param endpoint_id query string false "Filter by endpoint"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /debugger/traces [get]
func (h *Handler) ListTraces(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Query("endpoint_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	traces, total, err := h.service.ListTraces(c.Request.Context(), tenantID, endpointID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"traces": traces, "total": total})
}

// @Summary Get delivery trace (step-through view)
// @Tags Debugger
// @Produce json
// @Param delivery_id path string true "Delivery ID"
// @Success 200 {object} DeliveryTrace
// @Router /debugger/traces/{delivery_id} [get]
func (h *Handler) GetTrace(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deliveryID := c.Param("delivery_id")

	trace, err := h.service.GetTrace(c.Request.Context(), tenantID, deliveryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, trace)
}

// @Summary Diff delivery payloads at different stages
// @Tags Debugger
// @Produce json
// @Param delivery_id path string true "Delivery ID"
// @Success 200 {object} PayloadDiff
// @Router /debugger/traces/{delivery_id}/diff [get]
func (h *Handler) DiffPayloads(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deliveryID := c.Param("delivery_id")

	diff, err := h.service.DiffPayloads(c.Request.Context(), tenantID, deliveryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DIFF_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, diff)
}

// @Summary Replay a delivery with modifications
// @Tags Debugger
// @Accept json
// @Produce json
// @Param body body ReplayWithModRequest true "Replay configuration"
// @Success 200 {object} DeliveryTrace
// @Router /debugger/replay [post]
func (h *Handler) ReplayWithMods(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req ReplayWithModRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	trace, err := h.service.ReplayWithModifications(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "REPLAY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, trace)
}

// @Summary Bulk replay deliveries
// @Tags Debugger
// @Accept json
// @Produce json
// @Param body body BulkReplayRequest true "Bulk replay configuration"
// @Success 200 {object} BulkReplayResult
// @Router /debugger/replay/bulk [post]
func (h *Handler) BulkReplay(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req BulkReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	result, err := h.service.BulkReplay(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BULK_REPLAY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Create a debug session
// @Tags Debugger
// @Accept json
// @Produce json
// @Param body body CreateDebugSessionRequest true "Debug session config"
// @Success 201 {object} DebugSession
// @Router /debugger/sessions [post]
func (h *Handler) CreateDebugSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateDebugSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	session, err := h.service.CreateDebugSession(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// @Summary List debug sessions
// @Tags Debugger
// @Produce json
// @Success 200 {object} map[string][]DebugSession
// @Router /debugger/sessions [get]
func (h *Handler) ListDebugSessions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	sessions, err := h.service.ListDebugSessions(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// @Summary Get a debug session
// @Tags Debugger
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} DebugSession
// @Router /debugger/sessions/{id} [get]
func (h *Handler) GetDebugSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sessionID := c.Param("id")

	session, err := h.service.GetDebugSession(c.Request.Context(), tenantID, sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, session)
}

// @Summary Step through a debug session
// @Tags Debugger
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Router /debugger/sessions/{id}/step [post]
func (h *Handler) StepDebugSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	sessionID := c.Param("id")

	session, stage, err := h.service.StepDebugSession(c.Request.Context(), tenantID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "STEP_FAILED", "message": err.Error()}})
		return
	}

	response := gin.H{"session": session}
	if stage != nil {
		response["current_stage"] = stage
		response["has_more"] = session.Status == DebugStatusActive
	} else {
		response["has_more"] = false
		response["message"] = "No more stages"
	}

	c.JSON(http.StatusOK, response)
}
