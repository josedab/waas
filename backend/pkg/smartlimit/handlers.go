package smartlimit

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles smart limit HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new smart limit handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers smart limit routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	configs := r.Group("/rate-limits")
	{
		configs.GET("", h.ListConfigs)
		configs.POST("", h.CreateConfig)
		configs.GET("/:endpointId", h.GetConfig)
		configs.PUT("/:endpointId", h.UpdateConfig)
		configs.DELETE("/:endpointId", h.DeleteConfig)
		configs.GET("/:endpointId/prediction", h.GetPrediction)
		configs.POST("/:endpointId/train", h.TrainModel)
		configs.GET("/:endpointId/pattern", h.GetPattern)
		configs.GET("/:endpointId/recommendation", h.GetRecommendation)
	}

	r.GET("/rate-limits/stats", h.GetStats)
}

// ListConfigs godoc
// @Summary List adaptive rate configs
// @Description List all adaptive rate limit configurations
// @Tags rate-limits
// @Accept json
// @Produce json
// @Success 200 {array} AdaptiveRateConfig
// @Router /rate-limits [get]
func (h *Handler) ListConfigs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	configs, err := h.service.ListConfigs(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// CreateConfig godoc
// @Summary Create adaptive rate config
// @Description Create a new adaptive rate limit configuration
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param request body CreateAdaptiveConfigRequest true "Config request"
// @Success 201 {object} AdaptiveRateConfig
// @Router /rate-limits [post]
func (h *Handler) CreateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateAdaptiveConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetConfig godoc
// @Summary Get adaptive rate config
// @Description Get adaptive rate limit configuration for an endpoint
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} AdaptiveRateConfig
// @Router /rate-limits/{endpointId} [get]
func (h *Handler) GetConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	config, err := h.service.GetConfig(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfig godoc
// @Summary Update adaptive rate config
// @Description Update adaptive rate limit configuration
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Param request body UpdateAdaptiveConfigRequest true "Update request"
// @Success 200 {object} AdaptiveRateConfig
// @Router /rate-limits/{endpointId} [put]
func (h *Handler) UpdateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	var req UpdateAdaptiveConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.UpdateConfig(c.Request.Context(), tenantID, endpointID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeleteConfig godoc
// @Summary Delete adaptive rate config
// @Description Delete adaptive rate limit configuration
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 204
// @Router /rate-limits/{endpointId} [delete]
func (h *Handler) DeleteConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	if err := h.service.DeleteConfig(c.Request.Context(), tenantID, endpointID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetPrediction godoc
// @Summary Get rate limit prediction
// @Description Get ML-based rate limit prediction for an endpoint
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} RateLimitPrediction
// @Router /rate-limits/{endpointId}/prediction [get]
func (h *Handler) GetPrediction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	prediction, err := h.service.GetPrediction(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prediction not available"})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

// TrainModel godoc
// @Summary Train prediction model
// @Description Train or update the ML prediction model for an endpoint
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} PredictionModel
// @Router /rate-limits/{endpointId}/train [post]
func (h *Handler) TrainModel(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	model, err := h.service.TrainModel(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if model == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient data for training"})
		return
	}

	c.JSON(http.StatusOK, model)
}

// GetStats godoc
// @Summary Get smart limit stats
// @Description Get statistics for smart rate limiting
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Success 200 {object} SmartLimitStats
// @Router /rate-limits/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	start := time.Now().Add(-24 * time.Hour)
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

	stats, err := h.service.GetStats(c.Request.Context(), tenantID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetPattern godoc
// @Summary Get endpoint throughput pattern
// @Description Get the learned throughput pattern for an endpoint
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} EndpointPattern
// @Router /rate-limits/{endpointId}/pattern [get]
func (h *Handler) GetPattern(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	pattern, err := h.service.GetEndpointPattern(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if pattern == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "insufficient data for pattern analysis"})
		return
	}

	c.JSON(http.StatusOK, pattern)
}

// GetRecommendation godoc
// @Summary Get intelligent rate recommendation
// @Description Get a pattern-based rate recommendation for an endpoint
// @Tags rate-limits
// @Accept json
// @Produce json
// @Param endpointId path string true "Endpoint ID"
// @Success 200 {object} RateRecommendation
// @Router /rate-limits/{endpointId}/recommendation [get]
func (h *Handler) GetRecommendation(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	endpointID := c.Param("endpointId")

	recommendation, err := h.service.GetRateRecommendation(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if recommendation == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pattern analysis not available"})
		return
	}

	c.JSON(http.StatusOK, recommendation)
}
