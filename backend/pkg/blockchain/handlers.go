package blockchain

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/josedab/waas/pkg/httputil"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for blockchain monitoring
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	blockchain := r.Group("/blockchain")
	{
		// Monitors
		blockchain.POST("/monitors", h.CreateMonitor)
		blockchain.GET("/monitors", h.ListMonitors)
		blockchain.GET("/monitors/:id", h.GetMonitor)
		blockchain.PUT("/monitors/:id", h.UpdateMonitor)
		blockchain.DELETE("/monitors/:id", h.DeleteMonitor)
		blockchain.POST("/monitors/:id/pause", h.PauseMonitor)
		blockchain.POST("/monitors/:id/resume", h.ResumeMonitor)

		// Events
		blockchain.GET("/monitors/:id/events", h.GetMonitorEvents)
		blockchain.GET("/events", h.GetEvents)

		// Stats
		blockchain.GET("/monitors/:id/stats", h.GetMonitorStats)

		// Chains
		blockchain.GET("/chains", h.ListChains)

		// RPC Providers
		blockchain.POST("/providers", h.AddProvider)
		blockchain.GET("/providers", h.ListProviders)
	}
}

// CreateMonitor creates a new contract monitor
// @Summary Create contract monitor
// @Tags blockchain
// @Accept json
// @Produce json
// @Param request body CreateMonitorRequest true "Monitor request"
// @Success 201 {object} ContractMonitor
// @Failure 400 {object} ErrorResponse
// @Router /blockchain/monitors [post]
func (h *Handler) CreateMonitor(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req CreateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	monitor, err := h.service.CreateMonitor(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, monitor)
}

// GetMonitor retrieves a contract monitor
// @Summary Get contract monitor
// @Tags blockchain
// @Produce json
// @Param id path string true "Monitor ID"
// @Success 200 {object} ContractMonitor
// @Failure 404 {object} ErrorResponse
// @Router /blockchain/monitors/{id} [get]
func (h *Handler) GetMonitor(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	monitorID := c.Param("id")

	monitor, err := h.service.GetMonitor(c.Request.Context(), tenantID, monitorID)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "monitor not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, monitor)
}

// UpdateMonitor updates a contract monitor
// @Summary Update contract monitor
// @Tags blockchain
// @Accept json
// @Produce json
// @Param id path string true "Monitor ID"
// @Param request body UpdateMonitorRequest true "Update request"
// @Success 200 {object} ContractMonitor
// @Failure 400 {object} ErrorResponse
// @Router /blockchain/monitors/{id} [put]
func (h *Handler) UpdateMonitor(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	monitorID := c.Param("id")

	var req UpdateMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	monitor, err := h.service.UpdateMonitor(c.Request.Context(), tenantID, monitorID, &req)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "monitor not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, monitor)
}

// DeleteMonitor deletes a contract monitor
// @Summary Delete contract monitor
// @Tags blockchain
// @Param id path string true "Monitor ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /blockchain/monitors/{id} [delete]
func (h *Handler) DeleteMonitor(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	monitorID := c.Param("id")

	err := h.service.DeleteMonitor(c.Request.Context(), tenantID, monitorID)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "monitor not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListMonitors lists contract monitors
// @Summary List contract monitors
// @Tags blockchain
// @Produce json
// @Param chain query string false "Filter by chain"
// @Param network query string false "Filter by network"
// @Param status query string false "Filter by status"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListMonitorsResponse
// @Router /blockchain/monitors [get]
func (h *Handler) ListMonitors(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	filters := &MonitorFilters{
		Page:     1,
		PageSize: 20,
	}

	if chain := c.Query("chain"); chain != "" {
		ch := ChainType(chain)
		filters.Chain = &ch
	}
	if network := c.Query("network"); network != "" {
		n := NetworkType(network)
		filters.Network = &n
	}
	if status := c.Query("status"); status != "" {
		s := ContractMonitorStatus(status)
		filters.Status = &s
	}
	if page, _ := strconv.Atoi(c.Query("page")); page > 0 {
		filters.Page = page
	}
	if pageSize, _ := strconv.Atoi(c.Query("page_size")); pageSize > 0 && pageSize <= 100 {
		filters.PageSize = pageSize
	}

	response, err := h.service.ListMonitors(c.Request.Context(), tenantID, filters)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// PauseMonitor pauses a contract monitor
// @Summary Pause contract monitor
// @Tags blockchain
// @Param id path string true "Monitor ID"
// @Success 200 {object} ContractMonitor
// @Router /blockchain/monitors/{id}/pause [post]
func (h *Handler) PauseMonitor(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	monitorID := c.Param("id")

	status := MonitorStatusPaused
	monitor, err := h.service.UpdateMonitor(c.Request.Context(), tenantID, monitorID, &UpdateMonitorRequest{
		Status: &status,
	})
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "monitor not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, monitor)
}

// ResumeMonitor resumes a contract monitor
// @Summary Resume contract monitor
// @Tags blockchain
// @Param id path string true "Monitor ID"
// @Success 200 {object} ContractMonitor
// @Router /blockchain/monitors/{id}/resume [post]
func (h *Handler) ResumeMonitor(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	monitorID := c.Param("id")

	status := MonitorStatusActive
	monitor, err := h.service.UpdateMonitor(c.Request.Context(), tenantID, monitorID, &UpdateMonitorRequest{
		Status: &status,
	})
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "monitor not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, monitor)
}

// GetMonitorEvents retrieves events for a specific monitor
// @Summary Get monitor events
// @Tags blockchain
// @Produce json
// @Param id path string true "Monitor ID"
// @Param event_name query string false "Filter by event name"
// @Param confirmed query bool false "Filter by confirmed status"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListEventsResponse
// @Router /blockchain/monitors/{id}/events [get]
func (h *Handler) GetMonitorEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	monitorID := c.Param("id")

	filters := &EventFilters{
		MonitorID: monitorID,
		Page:      1,
		PageSize:  50,
	}

	if eventName := c.Query("event_name"); eventName != "" {
		filters.EventName = eventName
	}
	if confirmed := c.Query("confirmed"); confirmed != "" {
		b := confirmed == "true"
		filters.Confirmed = &b
	}
	if page, _ := strconv.Atoi(c.Query("page")); page > 0 {
		filters.Page = page
	}
	if pageSize, _ := strconv.Atoi(c.Query("page_size")); pageSize > 0 && pageSize <= 100 {
		filters.PageSize = pageSize
	}

	response, err := h.service.GetEvents(c.Request.Context(), tenantID, filters)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetEvents retrieves all events
// @Summary Get events
// @Tags blockchain
// @Produce json
// @Param monitor_id query string false "Filter by monitor ID"
// @Param event_name query string false "Filter by event name"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} ListEventsResponse
// @Router /blockchain/events [get]
func (h *Handler) GetEvents(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	filters := &EventFilters{
		Page:     1,
		PageSize: 50,
	}

	if monitorID := c.Query("monitor_id"); monitorID != "" {
		filters.MonitorID = monitorID
	}
	if eventName := c.Query("event_name"); eventName != "" {
		filters.EventName = eventName
	}
	if page, _ := strconv.Atoi(c.Query("page")); page > 0 {
		filters.Page = page
	}
	if pageSize, _ := strconv.Atoi(c.Query("page_size")); pageSize > 0 && pageSize <= 100 {
		filters.PageSize = pageSize
	}

	response, err := h.service.GetEvents(c.Request.Context(), tenantID, filters)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetMonitorStats retrieves monitor statistics
// @Summary Get monitor statistics
// @Tags blockchain
// @Produce json
// @Param id path string true "Monitor ID"
// @Success 200 {object} MonitorStats
// @Router /blockchain/monitors/{id}/stats [get]
func (h *Handler) GetMonitorStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	monitorID := c.Param("id")

	stats, err := h.service.GetMonitorStats(c.Request.Context(), tenantID, monitorID)
	if err != nil {
		if errors.Is(err, ErrMonitorNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "monitor not found"})
			return
		}
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ListChains lists supported blockchain networks
// @Summary List supported chains
// @Tags blockchain
// @Produce json
// @Success 200 {array} ChainConfig
// @Router /blockchain/chains [get]
func (h *Handler) ListChains(c *gin.Context) {
	chains := h.service.ListChains()
	c.JSON(http.StatusOK, chains)
}

// AddProvider adds an RPC provider
// @Summary Add RPC provider
// @Tags blockchain
// @Accept json
// @Produce json
// @Param request body RPCProvider true "Provider request"
// @Success 201 {object} RPCProvider
// @Failure 400 {object} ErrorResponse
// @Router /blockchain/providers [post]
func (h *Handler) AddProvider(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var provider RPCProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	err := h.service.AddRPCProvider(c.Request.Context(), tenantID, &provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// ListProviders lists RPC providers
// @Summary List RPC providers
// @Tags blockchain
// @Produce json
// @Param chain query string false "Filter by chain"
// @Param network query string false "Filter by network"
// @Success 200 {array} RPCProvider
// @Router /blockchain/providers [get]
func (h *Handler) ListProviders(c *gin.Context) {
	chain := ChainType(c.Query("chain"))
	network := NetworkType(c.Query("network"))

	providers, err := h.service.repo.ListProviders(c.Request.Context(), chain, network)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, providers)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
