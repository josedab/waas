package pipeline

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles pipeline HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new pipeline handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers pipeline routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	pipelines := r.Group("/pipelines")
	{
		pipelines.GET("", h.ListPipelines)
		pipelines.POST("", h.CreatePipeline)
		pipelines.GET("/templates", h.GetTemplates)
		pipelines.GET("/:pipelineId", h.GetPipeline)
		pipelines.PUT("/:pipelineId", h.UpdatePipeline)
		pipelines.DELETE("/:pipelineId", h.DeletePipeline)
		pipelines.POST("/:pipelineId/execute", h.ExecutePipeline)
		pipelines.GET("/:pipelineId/executions", h.ListExecutions)
		pipelines.GET("/:pipelineId/executions/:executionId", h.GetExecution)
	}
}

// CreatePipeline godoc
// @Summary Create a delivery pipeline
// @Description Create a declarative multi-step delivery pipeline
// @Tags pipelines
// @Accept json
// @Produce json
// @Param request body CreatePipelineRequest true "Pipeline definition"
// @Success 201 {object} Pipeline
// @Router /pipelines [post]
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

// GetPipeline godoc
// @Summary Get a pipeline
// @Description Retrieve a pipeline by ID
// @Tags pipelines
// @Produce json
// @Param pipelineId path string true "Pipeline ID"
// @Success 200 {object} Pipeline
// @Router /pipelines/{pipelineId} [get]
func (h *Handler) GetPipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("pipelineId")

	pipeline, err := h.service.GetPipeline(c.Request.Context(), tenantID, pipelineID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}

	c.JSON(http.StatusOK, pipeline)
}

// ListPipelines godoc
// @Summary List pipelines
// @Description List all pipelines for a tenant
// @Tags pipelines
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} Pipeline
// @Router /pipelines [get]
func (h *Handler) ListPipelines(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	pipelines, total, err := h.service.ListPipelines(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pipelines": pipelines,
		"total":     total,
	})
}

// UpdatePipeline godoc
// @Summary Update a pipeline
// @Description Update an existing pipeline
// @Tags pipelines
// @Accept json
// @Produce json
// @Param pipelineId path string true "Pipeline ID"
// @Param request body UpdatePipelineRequest true "Update request"
// @Success 200 {object} Pipeline
// @Router /pipelines/{pipelineId} [put]
func (h *Handler) UpdatePipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("pipelineId")

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

// DeletePipeline godoc
// @Summary Delete a pipeline
// @Description Delete a pipeline
// @Tags pipelines
// @Param pipelineId path string true "Pipeline ID"
// @Success 204
// @Router /pipelines/{pipelineId} [delete]
func (h *Handler) DeletePipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("pipelineId")

	if err := h.service.DeletePipeline(c.Request.Context(), tenantID, pipelineID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ExecutePipeline godoc
// @Summary Execute a pipeline
// @Description Execute a delivery pipeline with a test payload
// @Tags pipelines
// @Accept json
// @Produce json
// @Param pipelineId path string true "Pipeline ID"
// @Param request body map[string]interface{} true "Payload"
// @Success 200 {object} PipelineExecution
// @Router /pipelines/{pipelineId}/execute [post]
func (h *Handler) ExecutePipeline(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("pipelineId")

	var body struct {
		DeliveryID string          `json:"delivery_id"`
		Payload    json.RawMessage `json:"payload"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	execution, err := h.service.ExecutePipeline(c.Request.Context(), tenantID, pipelineID, body.DeliveryID, body.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// ListExecutions godoc
// @Summary List pipeline executions
// @Description List executions for a pipeline
// @Tags pipelines
// @Produce json
// @Param pipelineId path string true "Pipeline ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Router /pipelines/{pipelineId}/executions [get]
func (h *Handler) ListExecutions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	pipelineID := c.Param("pipelineId")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	executions, total, err := h.service.ListExecutions(c.Request.Context(), tenantID, pipelineID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"total":      total,
	})
}

// GetExecution godoc
// @Summary Get a pipeline execution
// @Description Retrieve a specific execution result
// @Tags pipelines
// @Produce json
// @Param pipelineId path string true "Pipeline ID"
// @Param executionId path string true "Execution ID"
// @Success 200 {object} PipelineExecution
// @Router /pipelines/{pipelineId}/executions/{executionId} [get]
func (h *Handler) GetExecution(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	executionID := c.Param("executionId")

	execution, err := h.service.GetExecution(c.Request.Context(), tenantID, executionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// GetTemplates godoc
// @Summary Get pipeline templates
// @Description Get pre-built pipeline templates
// @Tags pipelines
// @Produce json
// @Success 200 {array} Pipeline
// @Router /pipelines/templates [get]
func (h *Handler) GetTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetTemplates())
}
