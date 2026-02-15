package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/josedab/waas/pkg/connectors"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
)

// ConnectorsHandler handles connector marketplace HTTP requests
type ConnectorsHandler struct {
	service *connectors.Service
	logger  *utils.Logger
}

// NewConnectorsHandler creates a new connectors handler
func NewConnectorsHandler(service *connectors.Service, logger *utils.Logger) *ConnectorsHandler {
	return &ConnectorsHandler{
		service: service,
		logger:  logger,
	}
}

// InstallConnectorRequest represents connector installation request
type InstallConnectorRequest struct {
	ConnectorID string          `json:"connector_id" binding:"required"`
	Name        string          `json:"name" binding:"required"`
	Config      json.RawMessage `json:"config" binding:"required"`
}

// UpdateConnectorRequest represents connector update request
type UpdateConnectorRequest struct {
	Name     string          `json:"name"`
	Config   json.RawMessage `json:"config"`
	IsActive bool            `json:"is_active"`
}

// CreateCustomConnectorRequest represents custom connector creation
type CreateCustomConnectorRequest struct {
	Name           string                 `json:"name" binding:"required"`
	Description    string                 `json:"description"`
	SourceProvider string                 `json:"source_provider" binding:"required"`
	TargetProvider string                 `json:"target_provider" binding:"required"`
	EventTypes     []string               `json:"event_types"`
	Transform      string                 `json:"transform" binding:"required"`
	ConfigSchema   map[string]interface{} `json:"config_schema"`
}

// ListAvailableConnectors lists all available connectors in the marketplace
// @Summary List available connectors
// @Tags connectors
// @Produce json
// @Param category query string false "Filter by category"
// @Success 200 {array} connectors.Connector
// @Router /connectors/marketplace [get]
func (h *ConnectorsHandler) ListAvailableConnectors(c *gin.Context) {
	category := c.Query("category")

	filters := &connectors.MarketplaceListRequest{
		Category: category,
	}
	available, err := h.service.ListMarketplace(c.Request.Context(), filters)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, available)
}

// GetConnectorDetails gets details of a connector
// @Summary Get connector details
// @Tags connectors
// @Produce json
// @Param id path string true "Connector ID"
// @Success 200 {object} connectors.Connector
// @Router /connectors/marketplace/{id} [get]
func (h *ConnectorsHandler) GetConnectorDetails(c *gin.Context) {
	id := c.Param("id")

	connector, err := h.service.GetConnector(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connector not found"})
		return
	}

	c.JSON(http.StatusOK, connector)
}

// InstallConnector installs a connector for a tenant
// @Summary Install connector
// @Tags connectors
// @Accept json
// @Produce json
// @Param request body InstallConnectorRequest true "Install request"
// @Success 201 {object} connectors.InstalledConnector
// @Router /connectors/installed [post]
func (h *ConnectorsHandler) InstallConnector(c *gin.Context) {
	var req InstallConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	installReq := &connectors.InstallConnectorRequest{
		ConnectorID: req.ConnectorID,
		Name:        req.Name,
		Config:      req.Config,
	}

	installed, err := h.service.InstallConnector(c.Request.Context(), tenantID, installReq)
	if err != nil {
		h.logger.Error("Failed to install connector", map[string]interface{}{"error": err.Error()})
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, installed)
}

// ListInstalledConnectors lists installed connectors for a tenant
// @Summary List installed connectors
// @Tags connectors
// @Produce json
// @Success 200 {array} connectors.InstalledConnector
// @Router /connectors/installed [get]
func (h *ConnectorsHandler) ListInstalledConnectors(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	installed, _, err := h.service.ListInstalledConnectors(c.Request.Context(), tenantID, 100, 0)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, installed)
}

// GetInstalledConnector gets an installed connector
// @Summary Get installed connector
// @Tags connectors
// @Produce json
// @Param id path string true "Installed connector ID"
// @Success 200 {object} connectors.InstalledConnector
// @Router /connectors/installed/{id} [get]
func (h *ConnectorsHandler) GetInstalledConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	installed, err := h.service.GetInstalledConnector(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connector not found"})
		return
	}

	c.JSON(http.StatusOK, installed)
}

// UpdateInstalledConnector updates an installed connector
// @Summary Update installed connector
// @Tags connectors
// @Accept json
// @Produce json
// @Param id path string true "Installed connector ID"
// @Param request body UpdateConnectorRequest true "Update request"
// @Success 200 {object} connectors.InstalledConnector
// @Router /connectors/installed/{id} [patch]
func (h *ConnectorsHandler) UpdateInstalledConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req UpdateConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateReq := &connectors.UpdateConnectorRequest{
		Name:     req.Name,
		Config:   req.Config,
		IsActive: req.IsActive,
	}

	installed, err := h.service.UpdateInstalledConnector(c.Request.Context(), tenantID, id, updateReq)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, installed)
}

// UninstallConnector uninstalls a connector
// @Summary Uninstall connector
// @Tags connectors
// @Param id path string true "Installed connector ID"
// @Success 204
// @Router /connectors/installed/{id} [delete]
func (h *ConnectorsHandler) UninstallConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.UninstallConnector(c.Request.Context(), tenantID, id); err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateCustomConnector creates a custom connector
// @Summary Create custom connector
// @Tags connectors
// @Accept json
// @Produce json
// @Param request body CreateCustomConnectorRequest true "Connector request"
// @Success 201 {object} connectors.Connector
// @Router /connectors/custom [post]
func (h *ConnectorsHandler) CreateCustomConnector(c *gin.Context) {
	// Custom connectors not yet implemented
	c.JSON(http.StatusNotImplemented, gin.H{"error": "custom connectors coming soon"})
}

// ListCustomConnectors lists custom connectors for a tenant
// @Summary List custom connectors
// @Tags connectors
// @Produce json
// @Success 200 {array} connectors.Connector
// @Router /connectors/custom [get]
func (h *ConnectorsHandler) ListCustomConnectors(c *gin.Context) {
	// Custom connectors not yet implemented
	c.JSON(http.StatusOK, []interface{}{})
}

// DeleteCustomConnector deletes a custom connector
// @Summary Delete custom connector
// @Tags connectors
// @Param id path string true "Connector ID"
// @Success 204
// @Router /connectors/custom/{id} [delete]
func (h *ConnectorsHandler) DeleteCustomConnector(c *gin.Context) {
	// Custom connectors not yet implemented
	c.JSON(http.StatusNotImplemented, gin.H{"error": "custom connectors coming soon"})
}

// TestConnector tests a connector transformation
// @Summary Test connector
// @Tags connectors
// @Accept json
// @Produce json
// @Param id path string true "Connector ID"
// @Param payload body map[string]interface{} true "Test payload"
// @Success 200 {object} map[string]interface{}
// @Router /connectors/{id}/test [post]
func (h *ConnectorsHandler) TestConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payloadBytes, _ := json.Marshal(payload)
	result, err := h.service.ExecuteConnector(c.Request.Context(), tenantID, id, "test", payloadBytes)
	if err != nil {
		InternalErrorGeneric(c, err)
		return
	}

	var resultMap map[string]interface{}
	json.Unmarshal(result, &resultMap)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  resultMap,
	})
}

// GetConnectorStats gets statistics for an installed connector
// @Summary Get connector stats
// @Tags connectors
// @Produce json
// @Param id path string true "Installed connector ID"
// @Success 200 {object} map[string]interface{}
// @Router /connectors/installed/{id}/stats [get]
func (h *ConnectorsHandler) GetConnectorStats(c *gin.Context) {
	id := c.Param("id")
	_ = c.DefaultQuery("period", "7d")

	// In production, fetch from analytics
	stats := gin.H{
		"connector_id":      id,
		"total_executions":  1250,
		"successful":        1200,
		"failed":            50,
		"success_rate":      96.0,
		"avg_latency_ms":    45,
		"last_execution_at": "2024-01-15T10:30:00Z",
	}

	c.JSON(http.StatusOK, stats)
}

// RegisterConnectorsRoutes registers connector routes
func RegisterConnectorsRoutes(r *gin.RouterGroup, h *ConnectorsHandler) {
	conn := r.Group("/connectors")
	{
		// Marketplace
		conn.GET("/marketplace", h.ListAvailableConnectors)
		conn.GET("/marketplace/:id", h.GetConnectorDetails)

		// Installed connectors
		conn.POST("/installed", h.InstallConnector)
		conn.GET("/installed", h.ListInstalledConnectors)
		conn.GET("/installed/:id", h.GetInstalledConnector)
		conn.PATCH("/installed/:id", h.UpdateInstalledConnector)
		conn.DELETE("/installed/:id", h.UninstallConnector)
		conn.GET("/installed/:id/stats", h.GetConnectorStats)

		// Custom connectors
		conn.POST("/custom", h.CreateCustomConnector)
		conn.GET("/custom", h.ListCustomConnectors)
		conn.DELETE("/custom/:id", h.DeleteCustomConnector)

		// Testing
		conn.POST("/:id/test", h.TestConnector)
	}
}

// Suppress unused variable warning
var _ = strconv.Atoi
