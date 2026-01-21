package otel

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for OTEL configuration
type Handler struct {
	service *Service
}

// NewHandler creates a new OTEL handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers OTEL routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	otel := router.Group("/otel")
	{
		otel.POST("/configs", h.CreateConfig)
		otel.GET("/configs", h.ListConfigs)
		otel.GET("/configs/:id", h.GetConfig)
		otel.PUT("/configs/:id", h.UpdateConfig)
		otel.DELETE("/configs/:id", h.DeleteConfig)
		otel.POST("/configs/:id/test", h.TestConnection)
		otel.POST("/configs/:id/enable", h.EnableConfig)
		otel.POST("/configs/:id/disable", h.DisableConfig)
	}
}

// CreateConfig godoc
//
//	@Summary		Create OTEL config
//	@Description	Create a new OpenTelemetry configuration
//	@Tags			otel
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateConfigRequest	true	"Config request"
//	@Success		201		{object}	Config
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/otel/configs [post]
func (h *Handler) CreateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	config, err := h.service.CreateConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// ListConfigs godoc
//
//	@Summary		List OTEL configs
//	@Description	Get a list of OpenTelemetry configurations
//	@Tags			otel
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"		default(20)
//	@Param			offset	query		int	false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/otel/configs [get]
func (h *Handler) ListConfigs(c *gin.Context) {
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

	configs, total, err := h.service.ListConfigs(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetConfig godoc
//
//	@Summary		Get OTEL config
//	@Description	Get OpenTelemetry configuration details
//	@Tags			otel
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	Config
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/otel/configs/{id} [get]
func (h *Handler) GetConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	config, err := h.service.GetConfig(c.Request.Context(), tenantID, configID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfig godoc
//
//	@Summary		Update OTEL config
//	@Description	Update an OpenTelemetry configuration
//	@Tags			otel
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Config ID"
//	@Param			request	body		UpdateConfigRequest	true	"Update request"
//	@Success		200		{object}	Config
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/otel/configs/{id} [put]
func (h *Handler) UpdateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	config, err := h.service.UpdateConfig(c.Request.Context(), tenantID, configID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeleteConfig godoc
//
//	@Summary		Delete OTEL config
//	@Description	Delete an OpenTelemetry configuration
//	@Tags			otel
//	@Produce		json
//	@Param			id	path	string	true	"Config ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/otel/configs/{id} [delete]
func (h *Handler) DeleteConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	if err := h.service.DeleteConfig(c.Request.Context(), tenantID, configID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestConnection godoc
//
//	@Summary		Test OTEL connection
//	@Description	Test connection to the configured OTEL endpoint
//	@Tags			otel
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/otel/configs/{id}/test [post]
func (h *Handler) TestConnection(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	config, err := h.service.GetConfig(c.Request.Context(), tenantID, configID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if err := h.service.TestConnection(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection test successful",
	})
}

// EnableConfig godoc
//
//	@Summary		Enable OTEL config
//	@Description	Enable an OpenTelemetry configuration
//	@Tags			otel
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/otel/configs/{id}/enable [post]
func (h *Handler) EnableConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	enabled := true
	_, err := h.service.UpdateConfig(c.Request.Context(), tenantID, configID, &UpdateConfigRequest{
		Enabled: &enabled,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration enabled",
	})
}

// DisableConfig godoc
//
//	@Summary		Disable OTEL config
//	@Description	Disable an OpenTelemetry configuration
//	@Tags			otel
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/otel/configs/{id}/disable [post]
func (h *Handler) DisableConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("id")
	enabled := false
	_, err := h.service.UpdateConfig(c.Request.Context(), tenantID, configID, &UpdateConfigRequest{
		Enabled: &enabled,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration disabled",
	})
}

// RegisterPrometheusHandler registers a Prometheus metrics endpoint
func (h *Handler) RegisterPrometheusHandler(router *gin.RouterGroup, exporter *PrometheusExporter) {
	router.GET("/metrics", func(c *gin.Context) {
		exporter.ServeHTTP(c.Writer, c.Request)
	})
}
