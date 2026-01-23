package canary

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles canary deployment HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new canary handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers canary routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	canary := r.Group("/canary")
	{
		canary.POST("", h.CreateDeployment)
		canary.GET("", h.ListDeployments)
		canary.GET("/:id", h.GetDeployment)
		canary.PUT("/:id/traffic", h.UpdateTraffic)
		canary.POST("/:id/promote", h.PromoteCanary)
		canary.POST("/:id/rollback", h.RollbackCanary)
		canary.POST("/:id/pause", h.PauseCanary)
		canary.POST("/:id/resume", h.ResumeCanary)
		canary.GET("/:id/compare", h.EvaluateCanary)
		canary.POST("/:id/metrics", h.RecordMetrics)
	}
}

// CreateDeployment godoc
// @Summary Create canary deployment
// @Description Create a new canary deployment for webhook changes
// @Tags canary
// @Accept json
// @Produce json
// @Param request body CreateCanaryRequest true "Canary deployment request"
// @Success 201 {object} CanaryDeployment
// @Failure 400 {object} map[string]interface{}
// @Router /canary [post]
func (h *Handler) CreateDeployment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateCanaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	deployment, err := h.service.CreateDeployment(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, deployment)
}

// ListDeployments godoc
// @Summary List canary deployments
// @Description List canary deployments with optional filtering
// @Tags canary
// @Produce json
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} map[string]interface{}
// @Router /canary [get]
func (h *Handler) ListDeployments(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	status := c.Query("status")

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

	deployments, total, err := h.service.ListDeployments(c.Request.Context(), tenantID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deployments": deployments,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetDeployment godoc
// @Summary Get canary deployment
// @Description Get a canary deployment by ID
// @Tags canary
// @Produce json
// @Param id path string true "Deployment ID"
// @Success 200 {object} CanaryDeployment
// @Failure 404 {object} map[string]interface{}
// @Router /canary/{id} [get]
func (h *Handler) GetDeployment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deploymentID := c.Param("id")

	deployment, err := h.service.GetDeployment(c.Request.Context(), tenantID, deploymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// UpdateTraffic godoc
// @Summary Update canary traffic
// @Description Update the traffic percentage for a canary deployment
// @Tags canary
// @Accept json
// @Produce json
// @Param id path string true "Deployment ID"
// @Param request body UpdateTrafficRequest true "Traffic update request"
// @Success 200 {object} CanaryDeployment
// @Failure 400 {object} map[string]interface{}
// @Router /canary/{id}/traffic [put]
func (h *Handler) UpdateTraffic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deploymentID := c.Param("id")

	var req UpdateTrafficRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	deployment, err := h.service.UpdateTraffic(c.Request.Context(), tenantID, deploymentID, req.TrafficPct)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// PromoteCanary godoc
// @Summary Promote canary
// @Description Promote the canary deployment to full traffic
// @Tags canary
// @Produce json
// @Param id path string true "Deployment ID"
// @Success 200 {object} CanaryDeployment
// @Failure 400 {object} map[string]interface{}
// @Router /canary/{id}/promote [post]
func (h *Handler) PromoteCanary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deploymentID := c.Param("id")

	deployment, err := h.service.PromoteCanary(c.Request.Context(), tenantID, deploymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "PROMOTE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// RollbackCanary godoc
// @Summary Rollback canary
// @Description Rollback the canary deployment
// @Tags canary
// @Produce json
// @Param id path string true "Deployment ID"
// @Success 200 {object} CanaryDeployment
// @Failure 400 {object} map[string]interface{}
// @Router /canary/{id}/rollback [post]
func (h *Handler) RollbackCanary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deploymentID := c.Param("id")

	deployment, err := h.service.RollbackCanary(c.Request.Context(), tenantID, deploymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "ROLLBACK_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// PauseCanary godoc
// @Summary Pause canary
// @Description Pause an active canary deployment
// @Tags canary
// @Produce json
// @Param id path string true "Deployment ID"
// @Success 200 {object} CanaryDeployment
// @Failure 400 {object} map[string]interface{}
// @Router /canary/{id}/pause [post]
func (h *Handler) PauseCanary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deploymentID := c.Param("id")

	deployment, err := h.service.PauseCanary(c.Request.Context(), tenantID, deploymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "PAUSE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// ResumeCanary godoc
// @Summary Resume canary
// @Description Resume a paused canary deployment
// @Tags canary
// @Produce json
// @Param id path string true "Deployment ID"
// @Success 200 {object} CanaryDeployment
// @Failure 400 {object} map[string]interface{}
// @Router /canary/{id}/resume [post]
func (h *Handler) ResumeCanary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deploymentID := c.Param("id")

	deployment, err := h.service.ResumeCanary(c.Request.Context(), tenantID, deploymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "RESUME_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// EvaluateCanary godoc
// @Summary Evaluate canary
// @Description Compare canary vs baseline metrics and get health assessment
// @Tags canary
// @Produce json
// @Param id path string true "Deployment ID"
// @Success 200 {object} CanaryComparison
// @Failure 404 {object} map[string]interface{}
// @Router /canary/{id}/compare [get]
func (h *Handler) EvaluateCanary(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deploymentID := c.Param("id")

	comparison, err := h.service.EvaluateCanary(c.Request.Context(), tenantID, deploymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "EVALUATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, comparison)
}

// RecordMetrics godoc
// @Summary Record canary metrics
// @Description Record a metrics snapshot for a canary deployment
// @Tags canary
// @Accept json
// @Produce json
// @Param id path string true "Deployment ID"
// @Param request body CanaryMetrics true "Metrics snapshot"
// @Success 201 {object} CanaryMetrics
// @Failure 400 {object} map[string]interface{}
// @Router /canary/{id}/metrics [post]
func (h *Handler) RecordMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deploymentID := c.Param("id")

	var metrics CanaryMetrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	metrics.DeploymentID = deploymentID

	if err := h.service.RecordMetrics(c.Request.Context(), tenantID, &metrics); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "METRICS_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, metrics)
}
