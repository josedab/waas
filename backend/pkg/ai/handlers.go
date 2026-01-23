package ai

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for AI debugging
type Handler struct {
	service    *Service
	recommender *Recommender
	analyzer   *DeliveryAnalyzer
}

// NewHandler creates a new AI debugging handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// SetRecommender sets the recommender for the handler
func (h *Handler) SetRecommender(r *Recommender) {
	h.recommender = r
}

// SetAnalyzer sets the analyzer for the handler
func (h *Handler) SetAnalyzer(a *DeliveryAnalyzer) {
	h.analyzer = a
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
		ai.GET("/health-report/:endpointId", h.GetHealthReport)
		ai.GET("/recommendations/:endpointId", h.GetRecommendations)
		ai.GET("/anomalies", h.GetAnomalies)
		ai.GET("/insights", h.GetInsights)
		ai.GET("/failing-endpoints", h.GetFailingEndpoints)
		ai.POST("/classify", h.ClassifyError)
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

// GetHealthReport returns a comprehensive health report for an endpoint
// @Summary Get endpoint health report
// @Tags ai-debugging
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Param range query string false "Time range" default(24h)
// @Success 200 {object} EndpointHealthReport
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/health-report/{endpointId} [get]
func (h *Handler) GetHealthReport(c *gin.Context) {
	if h.recommender == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": map[string]interface{}{"code": "SERVICE_UNAVAILABLE", "message": "recommender not configured"}})
		return
	}

	endpointID := c.Param("endpointId")
	if endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]interface{}{"code": "INVALID_REQUEST", "message": "endpoint ID is required"}})
		return
	}

	timeRange := 24 * time.Hour
	if r := c.Query("range"); r != "" {
		if parsed, err := time.ParseDuration(r); err == nil {
			timeRange = parsed
		}
	}

	report, err := h.recommender.GenerateHealthReport(c.Request.Context(), endpointID, timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]interface{}{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetRecommendations returns retry recommendations for an endpoint
// @Summary Get retry recommendations
// @Tags ai-debugging
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Param error query string false "Last error message"
// @Success 200 {object} RetryRecommendation
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/recommendations/{endpointId} [get]
func (h *Handler) GetRecommendations(c *gin.Context) {
	if h.recommender == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": map[string]interface{}{"code": "SERVICE_UNAVAILABLE", "message": "recommender not configured"}})
		return
	}

	endpointID := c.Param("endpointId")
	if endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]interface{}{"code": "INVALID_REQUEST", "message": "endpoint ID is required"}})
		return
	}

	lastError := c.Query("error")
	if lastError == "" {
		lastError = "unknown error"
	}

	rec, err := h.recommender.RecommendRetryStrategy(c.Request.Context(), endpointID, lastError)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]interface{}{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, rec)
}

// GetAnomalies returns detected anomalies for the tenant
// @Summary List detected anomalies
// @Tags ai-debugging
// @Produce json
// @Param range query string false "Time range" default(24h)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/anomalies [get]
func (h *Handler) GetAnomalies(c *gin.Context) {
	if h.recommender == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": map[string]interface{}{"code": "SERVICE_UNAVAILABLE", "message": "recommender not configured"}})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	timeRange := 24 * time.Hour
	if r := c.Query("range"); r != "" {
		if parsed, err := time.ParseDuration(r); err == nil {
			timeRange = parsed
		}
	}

	anomalies, err := h.recommender.DetectAnomalies(c.Request.Context(), tenantID, timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]interface{}{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"anomalies": anomalies})
}

// GetInsights returns AI-generated insights for the tenant
// @Summary Get AI-generated insights
// @Tags ai-debugging
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/insights [get]
func (h *Handler) GetInsights(c *gin.Context) {
	if h.analyzer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": map[string]interface{}{"code": "SERVICE_UNAVAILABLE", "message": "analyzer not configured"}})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	insights, err := h.analyzer.GenerateInsights(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]interface{}{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"insights": insights})
}

// GetFailingEndpoints returns top failing endpoints
// @Summary Top failing endpoints
// @Tags ai-debugging
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/failing-endpoints [get]
func (h *Handler) GetFailingEndpoints(c *gin.Context) {
	if h.recommender == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": map[string]interface{}{"code": "SERVICE_UNAVAILABLE", "message": "recommender not configured"}})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	endpoints, err := h.recommender.GetTopFailingEndpoints(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]interface{}{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
}

// ClassifyError manually classifies an error
// @Summary Manually classify an error
// @Tags ai-debugging
// @Accept json
// @Produce json
// @Param request body ClassifyRequest true "Classify request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /ai/classify [post]
func (h *Handler) ClassifyError(c *gin.Context) {
	var req ClassifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]interface{}{"code": "INVALID_REQUEST", "message": "invalid request: " + err.Error()}})
		return
	}

	classifier := h.service.classifier

	var classification ErrorClassification
	if req.Headers != nil || req.LatencyMs > 0 {
		latency := time.Duration(req.LatencyMs) * time.Millisecond
		statusCode := 0
		if req.StatusCode != nil {
			statusCode = *req.StatusCode
		}
		classification = classifier.ClassifyFromHTTPResponse(statusCode, req.ResponseBody, req.Headers, latency)
	} else {
		classification = classifier.Classify(req.ErrorMessage, req.StatusCode, req.ResponseBody)
	}

	suggestions := classifier.GetSuggestions(classification, &DeliveryContext{
		ErrorMessage: req.ErrorMessage,
		HTTPStatus:   req.StatusCode,
		ResponseBody: req.ResponseBody,
	})

	c.JSON(http.StatusOK, gin.H{
		"classification": classification,
		"suggestions":    suggestions,
	})
}
