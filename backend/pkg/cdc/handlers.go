package cdc

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for CDC operations
type Handlers struct {
	service *Service
}

// NewHandlers creates new CDC handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers CDC routes
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	cdc := r.Group("/cdc")
	{
		cdc.GET("/connectors", h.ListConnectors)
		cdc.POST("/connectors", h.CreateConnector)
		cdc.GET("/connectors/:id", h.GetConnector)
		cdc.PUT("/connectors/:id", h.UpdateConnector)
		cdc.DELETE("/connectors/:id", h.DeleteConnector)

		cdc.POST("/connectors/:id/start", h.StartConnector)
		cdc.POST("/connectors/:id/stop", h.StopConnector)
		cdc.POST("/connectors/:id/pause", h.PauseConnector)
		cdc.POST("/connectors/:id/resume", h.ResumeConnector)

		cdc.POST("/test-connection", h.TestConnection)
		cdc.GET("/connectors/:id/schema", h.GetSchema)
		cdc.GET("/connectors/:id/metrics", h.GetMetrics)
		cdc.GET("/connectors/:id/health", h.GetHealth)
		cdc.GET("/connectors/:id/events", h.GetEventHistory)

		cdc.GET("/supported-types", h.GetSupportedTypes)
	}
}

// ListConnectors lists all CDC connectors for a tenant
// @Summary List CDC connectors
// @Tags CDC
// @Produce json
// @Param status query string false "Filter by status"
// @Success 200 {array} CDCConnector
// @Router /cdc/connectors [get]
func (h *Handlers) ListConnectors(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var status *ConnectorStatus
	if s := c.Query("status"); s != "" {
		st := ConnectorStatus(s)
		status = &st
	}

	connectors, err := h.service.ListConnectors(c.Request.Context(), tenantID, status)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, connectors)
}

// CreateConnector creates a new CDC connector
// @Summary Create CDC connector
// @Tags CDC
// @Accept json
// @Produce json
// @Param request body CreateConnectorRequest true "Connector configuration"
// @Success 201 {object} CDCConnector
// @Router /cdc/connectors [post]
func (h *Handlers) CreateConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	connector, err := h.service.CreateConnector(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, connector)
}

// GetConnector gets a CDC connector by ID
// @Summary Get CDC connector
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Success 200 {object} CDCConnector
// @Router /cdc/connectors/{id} [get]
func (h *Handlers) GetConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	connector, err := h.service.GetConnector(c.Request.Context(), tenantID, connID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, connector)
}

// UpdateConnector updates a CDC connector
// @Summary Update CDC connector
// @Tags CDC
// @Accept json
// @Produce json
// @Param id path string true "Connector ID"
// @Param request body UpdateConnectorRequest true "Update configuration"
// @Success 200 {object} CDCConnector
// @Router /cdc/connectors/{id} [put]
func (h *Handlers) UpdateConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	var req UpdateConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	connector, err := h.service.UpdateConnector(c.Request.Context(), tenantID, connID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, connector)
}

// DeleteConnector deletes a CDC connector
// @Summary Delete CDC connector
// @Tags CDC
// @Param id path string true "Connector ID"
// @Success 204
// @Router /cdc/connectors/{id} [delete]
func (h *Handlers) DeleteConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	if err := h.service.DeleteConnector(c.Request.Context(), tenantID, connID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// StartConnector starts a CDC connector
// @Summary Start CDC connector
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Success 200 {object} CDCConnector
// @Router /cdc/connectors/{id}/start [post]
func (h *Handlers) StartConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	connector, err := h.service.StartConnector(c.Request.Context(), tenantID, connID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, connector)
}

// StopConnector stops a CDC connector
// @Summary Stop CDC connector
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Success 200 {object} CDCConnector
// @Router /cdc/connectors/{id}/stop [post]
func (h *Handlers) StopConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	connector, err := h.service.StopConnector(c.Request.Context(), tenantID, connID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, connector)
}

// PauseConnector pauses a CDC connector
// @Summary Pause CDC connector
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Success 200 {object} CDCConnector
// @Router /cdc/connectors/{id}/pause [post]
func (h *Handlers) PauseConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	connector, err := h.service.PauseConnector(c.Request.Context(), tenantID, connID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, connector)
}

// ResumeConnector resumes a paused CDC connector
// @Summary Resume CDC connector
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Success 200 {object} CDCConnector
// @Router /cdc/connectors/{id}/resume [post]
func (h *Handlers) ResumeConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	// Resume uses Start internally
	connector, err := h.service.StartConnector(c.Request.Context(), tenantID, connID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, connector)
}

// TestConnection tests database connectivity
// @Summary Test database connection
// @Tags CDC
// @Accept json
// @Produce json
// @Param request body TestConnectionRequest true "Connection configuration"
// @Success 200 {object} TestConnectionResult
// @Router /cdc/test-connection [post]
func (h *Handlers) TestConnection(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.TestConnection(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSchema retrieves table schema
// @Summary Get table schema
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Param schema query string true "Schema name"
// @Param table query string true "Table name"
// @Success 200 {object} SchemaInfo
// @Router /cdc/connectors/{id}/schema [get]
func (h *Handlers) GetSchema(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")
	schema := c.Query("schema")
	table := c.Query("table")

	if schema == "" || table == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "schema and table parameters required"})
		return
	}

	schemaInfo, err := h.service.GetSchema(c.Request.Context(), tenantID, connID, schema, table)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, schemaInfo)
}

// GetMetrics retrieves connector metrics
// @Summary Get connector metrics
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Success 200 {object} CDCMetrics
// @Router /cdc/connectors/{id}/metrics [get]
func (h *Handlers) GetMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	metrics, err := h.service.GetMetrics(c.Request.Context(), tenantID, connID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetHealth retrieves connector health
// @Summary Get connector health
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Success 200 {object} ConnectorHealth
// @Router /cdc/connectors/{id}/health [get]
func (h *Handlers) GetHealth(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	health, err := h.service.GetHealth(c.Request.Context(), tenantID, connID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetEventHistory retrieves event history
// @Summary Get event history
// @Tags CDC
// @Produce json
// @Param id path string true "Connector ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} EventHistory
// @Router /cdc/connectors/{id}/events [get]
func (h *Handlers) GetEventHistory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	connID := c.Param("id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	events, err := h.service.GetEventHistory(c.Request.Context(), tenantID, connID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, events)
}

// GetSupportedTypes returns supported connector types
// @Summary Get supported connector types
// @Tags CDC
// @Produce json
// @Success 200 {array} ConnectorTypeInfo
// @Router /cdc/supported-types [get]
func (h *Handlers) GetSupportedTypes(c *gin.Context) {
	types := h.service.GetSupportedConnectors()
	c.JSON(http.StatusOK, types)
}
