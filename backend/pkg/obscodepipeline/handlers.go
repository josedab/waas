package obscodepipeline

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for observability pipeline management.
type Handler struct {
	service *Service
}

// NewHandler creates a new observability pipeline handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers observability pipeline routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/obs-pipelines")
	{
		g.POST("", h.CreatePipeline)
		g.GET("", h.ListPipelines)
		g.GET("/:id", h.GetPipeline)
		g.PUT("/:id", h.UpdatePipeline)
		g.DELETE("/:id", h.DeletePipeline)
		g.POST("/:id/activate", h.ActivatePipeline)
		g.POST("/:id/pause", h.PausePipeline)
		g.POST("/validate", h.ValidateSpec)
		g.GET("/:id/stats", h.GetPipelineStats)
		g.GET("/:id/executions", h.ListExecutions)
		g.GET("/alerts", h.ListAlertEvents)
		g.GET("/alerts/active", h.GetActiveAlerts)
	}
}

func (h *Handler) CreatePipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pipeline, err := h.service.CreatePipeline(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pipeline)
}

func (h *Handler) GetPipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("id")

	pipeline, err := h.service.GetPipeline(c.Request.Context(), tenantID, pipelineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

func (h *Handler) ListPipelines(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	pipelines, err := h.service.ListPipelines(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pipelines": pipelines})
}

func (h *Handler) UpdatePipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("id")

	var req UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pipeline, err := h.service.UpdatePipeline(c.Request.Context(), tenantID, pipelineID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

func (h *Handler) DeletePipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("id")

	if err := h.service.DeletePipeline(c.Request.Context(), tenantID, pipelineID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ActivatePipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("id")

	pipeline, err := h.service.ActivatePipeline(c.Request.Context(), tenantID, pipelineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

func (h *Handler) PausePipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("id")

	pipeline, err := h.service.PausePipeline(c.Request.Context(), tenantID, pipelineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

func (h *Handler) ValidateSpec(c *gin.Context) {
	var body struct {
		Spec json.RawMessage `json:"spec" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	errors, err := h.service.ValidateSpec(c.Request.Context(), body.Spec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "errors": errors})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (h *Handler) GetPipelineStats(c *gin.Context) {
	pipelineID := c.Param("id")

	stats, err := h.service.GetPipelineStats(c.Request.Context(), pipelineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) ListExecutions(c *gin.Context) {
	pipelineID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	executions, err := h.service.ListExecutions(c.Request.Context(), pipelineID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"executions": executions})
}

func (h *Handler) ListAlertEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	alerts, err := h.service.ListAlertEvents(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (h *Handler) GetActiveAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	alerts, err := h.service.GetActiveAlerts(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}
