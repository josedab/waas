package ai

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for AI debugging
type Handler struct {
	service *Service
}

// NewHandler creates a new AI debugging handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers AI debugging routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	ai := router.Group("/ai")
	{
		ai.POST("/analyze", h.AnalyzeDelivery)
		ai.POST("/analyze/batch", h.AnalyzeBatch)
		ai.GET("/analyses", h.ListAnalyses)
		ai.GET("/analyses/:id", h.GetAnalysis)
		ai.POST("/transform/generate", h.GenerateTransform)
		ai.GET("/patterns", h.GetPatterns)
	}
}

// AnalyzeDelivery godoc
// @Summary Analyze webhook delivery failure
// @Description Use AI to analyze a failed webhook delivery and get debugging suggestions
// @Tags ai-debugging
// @Accept json
// @Produce json
// @Param request body AnalysisRequest true "Analysis request"
// @Success 200 {object} DebugAnalysis
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/analyze [post]
func (h *Handler) AnalyzeDelivery(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	analysis, err := h.service.AnalyzeDelivery(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// AnalyzeBatch godoc
// @Summary Analyze multiple webhook delivery failures
// @Description Analyze multiple failed deliveries and get aggregated insights
// @Tags ai-debugging
// @Accept json
// @Produce json
// @Param request body BatchAnalysisRequest true "Batch analysis request"
// @Success 200 {object} BatchAnalysisResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/analyze/batch [post]
func (h *Handler) AnalyzeBatch(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req BatchAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	response, err := h.service.AnalyzeBatch(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListAnalyses godoc
// @Summary List AI analyses
// @Description Get a list of previous AI analyses for the tenant
// @Tags ai-debugging
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/analyses [get]
func (h *Handler) ListAnalyses(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	analyses, total, err := h.service.ListAnalyses(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analyses": analyses,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetAnalysis godoc
// @Summary Get AI analysis details
// @Description Get details of a specific AI analysis
// @Tags ai-debugging
// @Produce json
// @Param id path string true "Analysis ID"
// @Success 200 {object} DebugAnalysis
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/analyses/{id} [get]
func (h *Handler) GetAnalysis(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	analysisID := c.Param("id")
	analysis, err := h.service.GetAnalysis(c.Request.Context(), tenantID, analysisID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if analysis == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// GenerateTransform godoc
// @Summary Generate transformation script
// @Description Use AI to generate a transformation script from input/output examples
// @Tags ai-debugging
// @Accept json
// @Produce json
// @Param request body TransformGenerateRequest true "Transform generation request"
// @Success 200 {object} TransformGenerateResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/transform/generate [post]
func (h *Handler) GenerateTransform(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req TransformGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	response, err := h.service.GenerateTransformation(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPatterns godoc
// @Summary Get learned error patterns
// @Description Get error patterns learned from previous failures
// @Tags ai-debugging
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/patterns [get]
func (h *Handler) GetPatterns(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	patterns, err := h.service.GetPatterns(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"patterns": patterns})
}
