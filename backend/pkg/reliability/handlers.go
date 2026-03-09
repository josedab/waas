package reliability

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for reliability scoring.
type Handler struct {
	service *Service
}

// NewHandler creates a new reliability handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers reliability routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	reliability := router.Group("/reliability")
	{
		reliability.GET("/endpoints/:endpoint_id", h.GetReliability)
		reliability.GET("/endpoints/:endpoint_id/score", h.GetScore)
		reliability.GET("/endpoints/:endpoint_id/trend", h.GetTrend)
		reliability.POST("/endpoints/:endpoint_id/snapshot", h.TakeSnapshot)
		reliability.PUT("/endpoints/:endpoint_id/sla", h.SetSLA)
		reliability.GET("/endpoints/:endpoint_id/sla", h.GetSLA)
		reliability.PUT("/endpoints/:endpoint_id/alerts", h.SetAlertThreshold)
		reliability.POST("/endpoints/:endpoint_id/alerts/check", h.CheckAlerts)
	}
}

// GetReliability godoc
//
//	@Summary		Get endpoint reliability report
//	@Description	Returns the full reliability report including score, trend, and SLA status
//	@Tags			reliability
//	@Produce		json
//	@Param			endpoint_id	path		string	true	"Endpoint ID"
//	@Success		200			{object}	ReliabilityReport
//	@Failure		401			{object}	map[string]interface{}
//	@Failure		500			{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/reliability/endpoints/{endpoint_id} [get]
func (h *Handler) GetReliability(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	report, err := h.service.GetReliability(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetScore godoc
//
//	@Summary		Get current reliability score
//	@Description	Computes and returns the current reliability score for an endpoint
//	@Tags			reliability
//	@Produce		json
//	@Param			endpoint_id	path		string	true	"Endpoint ID"
//	@Param			window_hours	query	int		false	"Window hours"	default(24)
//	@Success		200			{object}	ReliabilityScore
//	@Security		ApiKeyAuth
//	@Router			/reliability/endpoints/{endpoint_id}/score [get]
func (h *Handler) GetScore(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	windowHours := 24
	if wh := c.Query("window_hours"); wh != "" {
		if parsed, err := strconv.Atoi(wh); err == nil && parsed > 0 {
			windowHours = parsed
		}
	}

	score, err := h.service.ComputeScore(c.Request.Context(), tenantID, endpointID, windowHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, score)
}

// GetTrend godoc
//
//	@Summary		Get reliability trend data
//	@Description	Returns time-series reliability snapshots for charting
//	@Tags			reliability
//	@Produce		json
//	@Param			endpoint_id	path	string	true	"Endpoint ID"
//	@Param			limit		query	int		false	"Number of data points"	default(168)
//	@Success		200			{object}	ReliabilityTrend
//	@Security		ApiKeyAuth
//	@Router			/reliability/endpoints/{endpoint_id}/trend [get]
func (h *Handler) GetTrend(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	limit := 168
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	snapshots, err := h.service.repo.ListSnapshots(c.Request.Context(), tenantID, endpointID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, &ReliabilityTrend{
		EndpointID: endpointID,
		Period:     "custom",
		DataPoints: snapshots,
	})
}

// TakeSnapshot godoc
//
//	@Summary		Take a reliability snapshot
//	@Description	Records an hourly reliability score snapshot
//	@Tags			reliability
//	@Produce		json
//	@Param			endpoint_id	path		string	true	"Endpoint ID"
//	@Success		201			{object}	ScoreSnapshot
//	@Security		ApiKeyAuth
//	@Router			/reliability/endpoints/{endpoint_id}/snapshot [post]
func (h *Handler) TakeSnapshot(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	snapshot, err := h.service.TakeSnapshot(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, snapshot)
}

// SetSLA godoc
//
//	@Summary		Set SLA target
//	@Description	Create or update SLA targets for an endpoint
//	@Tags			reliability
//	@Accept			json
//	@Produce		json
//	@Param			endpoint_id	path		string				true	"Endpoint ID"
//	@Param			request		body		CreateSLARequest	true	"SLA request"
//	@Success		200			{object}	SLATarget
//	@Security		ApiKeyAuth
//	@Router			/reliability/endpoints/{endpoint_id}/sla [put]
func (h *Handler) SetSLA(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	var req CreateSLARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	sla, err := h.service.SetSLA(c.Request.Context(), tenantID, endpointID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sla)
}

// GetSLA returns the current SLA target for an endpoint.
func (h *Handler) GetSLA(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	sla, err := h.service.repo.GetSLA(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sla == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no SLA target configured"})
		return
	}

	c.JSON(http.StatusOK, sla)
}

// SetAlertThreshold creates or updates alert thresholds.
func (h *Handler) SetAlertThreshold(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	var req CreateAlertThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	threshold, err := h.service.SetAlertThreshold(c.Request.Context(), tenantID, endpointID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, threshold)
}

// CheckAlerts evaluates alert thresholds against current scores.
func (h *Handler) CheckAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	endpointID := c.Param("endpoint_id")
	violated, err := h.service.CheckAlerts(c.Request.Context(), tenantID, endpointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"endpoint_id": endpointID,
		"violated":    violated,
	})
}
