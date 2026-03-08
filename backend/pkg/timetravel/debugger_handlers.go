package timetravel

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/httputil"
)

// CreateDebugSession handles creation of a new debug session.
func (h *Handler) CreateDebugSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateDebugSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.service.CreateDebugSession(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// InspectEvent handles detailed event inspection.
func (h *Handler) InspectEvent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	eventID := c.Param("id")

	inspection, err := h.service.InspectEvent(c.Request.Context(), tenantID, eventID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, inspection)
}

// ReplayWithModification handles replaying an event with modifications.
func (h *Handler) ReplayWithModification(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req ReplayWithModificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	step, err := h.service.ReplayWithModification(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, step)
}

// CompareEvents handles comparison of two events.
func (h *Handler) CompareEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	eventID1 := c.Query("event_id_1")
	eventID2 := c.Query("event_id_2")

	if eventID1 == "" || eventID2 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_id_1 and event_id_2 query params required"})
		return
	}

	diff, err := h.service.CompareEvents(c.Request.Context(), tenantID, eventID1, eventID2)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, diff)
}

// AddBreakpointHandler handles adding a breakpoint to a debug session.
func (h *Handler) AddBreakpointHandler(c *gin.Context) {
	sessionID := c.Param("id")
	var req AddBreakpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bp, err := h.service.AddBreakpoint(c.Request.Context(), sessionID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bp)
}
