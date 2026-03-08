package chaos

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles chaos HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new chaos handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers chaos routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	experiments := r.Group("/chaos")
	{
		experiments.GET("/templates", h.GetTemplates)
		experiments.GET("/experiments", h.ListExperiments)
		experiments.POST("/experiments", h.CreateExperiment)
		experiments.GET("/experiments/:id", h.GetExperiment)
		experiments.DELETE("/experiments/:id", h.DeleteExperiment)
		experiments.POST("/experiments/:id/start", h.StartExperiment)
		experiments.POST("/experiments/:id/stop", h.StopExperiment)
		experiments.GET("/experiments/:id/events", h.GetEvents)
		experiments.GET("/resilience-report", h.GetResilienceReport)
	}
}

// GetTemplates godoc
// @Summary Get experiment templates
// @Description Get predefined chaos experiment templates
// @Tags chaos
// @Produce json
// @Success 200 {array} ExperimentTemplate
// @Router /chaos/templates [get]
func (h *Handler) GetTemplates(c *gin.Context) {
	templates := h.service.GetTemplates()
	c.JSON(http.StatusOK, templates)
}

// ListExperiments godoc
// @Summary List chaos experiments
// @Description List all chaos experiments with optional filtering
// @Tags chaos
// @Produce json
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /chaos/experiments [get]
func (h *Handler) ListExperiments(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var status *ExperimentStatus
	if s := c.Query("status"); s != "" {
		st := ExperimentStatus(s)
		status = &st
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if _, err := parseInt(l); err == nil {
			limit, _ = parseInt(l)
		}
	}
	if o := c.Query("offset"); o != "" {
		if _, err := parseInt(o); err == nil {
			offset, _ = parseInt(o)
		}
	}

	experiments, total, err := h.service.ListExperiments(c.Request.Context(), tenantID, status, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"experiments": experiments,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// CreateExperiment godoc
// @Summary Create chaos experiment
// @Description Create a new chaos experiment
// @Tags chaos
// @Accept json
// @Produce json
// @Param request body CreateExperimentRequest true "Experiment request"
// @Success 201 {object} ChaosExperiment
// @Router /chaos/experiments [post]
func (h *Handler) CreateExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	var req CreateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exp, err := h.service.CreateExperiment(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, exp)
}

// GetExperiment godoc
// @Summary Get chaos experiment
// @Description Get a chaos experiment by ID
// @Tags chaos
// @Produce json
// @Param id path string true "Experiment ID"
// @Success 200 {object} ChaosExperiment
// @Router /chaos/experiments/{id} [get]
func (h *Handler) GetExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	exp, err := h.service.GetExperiment(c.Request.Context(), tenantID, expID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Experiment not found"})
		return
	}

	c.JSON(http.StatusOK, exp)
}

// DeleteExperiment godoc
// @Summary Delete chaos experiment
// @Description Delete a chaos experiment
// @Tags chaos
// @Produce json
// @Param id path string true "Experiment ID"
// @Success 204
// @Router /chaos/experiments/{id} [delete]
func (h *Handler) DeleteExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	if err := h.service.DeleteExperiment(c.Request.Context(), tenantID, expID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// StartExperiment godoc
// @Summary Start chaos experiment
// @Description Start a chaos experiment
// @Tags chaos
// @Produce json
// @Param id path string true "Experiment ID"
// @Success 200 {object} ChaosExperiment
// @Router /chaos/experiments/{id}/start [post]
func (h *Handler) StartExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	exp, err := h.service.StartExperiment(c.Request.Context(), tenantID, expID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, exp)
}

// StopExperiment godoc
// @Summary Stop chaos experiment
// @Description Stop a running chaos experiment
// @Tags chaos
// @Produce json
// @Param id path string true "Experiment ID"
// @Success 200 {object} ChaosExperiment
// @Router /chaos/experiments/{id}/stop [post]
func (h *Handler) StopExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	exp, err := h.service.StopExperiment(c.Request.Context(), tenantID, expID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, exp)
}

// GetEvents godoc
// @Summary Get chaos events
// @Description Get events for a chaos experiment
// @Tags chaos
// @Produce json
// @Param id path string true "Experiment ID"
// @Param limit query int false "Limit results"
// @Success 200 {array} ChaosEvent
// @Router /chaos/experiments/{id}/events [get]
func (h *Handler) GetEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil {
			limit = parsed
		}
	}

	events, err := h.service.GetEvents(c.Request.Context(), tenantID, expID, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetResilienceReport godoc
// @Summary Get resilience report
// @Description Get a resilience assessment report
// @Tags chaos
// @Produce json
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Success 200 {object} ResilienceReport
// @Router /chaos/resilience-report [get]
func (h *Handler) GetResilienceReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()

	if s := c.Query("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = t
		}
	}
	if e := c.Query("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			end = t
		}
	}

	report, err := h.service.GetResilienceReport(c.Request.Context(), tenantID, start, end)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, report)
}

func parseInt(s string) (int, error) {
	var val int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		val = val*10 + int(c-'0')
	}
	return val, nil
}
