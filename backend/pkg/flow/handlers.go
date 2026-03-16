package flow

import (
	"fmt"
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// errorResponse represents a structured error returned by flow handlers.
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Handler provides HTTP handlers for flow management
type Handler struct {
	service *Service
}

// NewHandler creates a new flow handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers flow routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	flows := router.Group("/flows")
	{
		flows.POST("", h.CreateFlow)
		flows.GET("", h.ListFlows)
		flows.GET("/templates", h.GetTemplates)
		flows.GET("/:id", h.GetFlow)
		flows.PUT("/:id", h.UpdateFlow)
		flows.DELETE("/:id", h.DeleteFlow)
		flows.POST("/:id/execute", h.ExecuteFlow)
		flows.GET("/:id/executions", h.ListExecutions)
		flows.POST("/validate", h.ValidateFlow)
	}

	// Execution details
	executions := router.Group("/executions")
	{
		executions.GET("/:id", h.GetExecution)
	}

	// Endpoint flow assignments
	endpoints := router.Group("/endpoints")
	{
		endpoints.POST("/:id/flows", h.AssignFlow)
		endpoints.GET("/:id/flows", h.GetEndpointFlows)
		endpoints.DELETE("/:id/flows/:flowId", h.RemoveFlow)
	}
}

// CreateFlow godoc
//
//	@Summary		Create a new flow
//	@Description	Create a new visual workflow for webhook processing
//	@Tags			flows
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateFlowRequest	true	"Flow creation request"
//	@Success		201		{object}	Flow
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows [post]
func (h *Handler) CreateFlow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	var req CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	flow, err := h.service.CreateFlow(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, flow)
}

// ListFlows godoc
//
//	@Summary		List flows
//	@Description	Get a list of flows for the tenant
//	@Tags			flows
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"		default(20)
//	@Param			offset	query		int	false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows [get]
func (h *Handler) ListFlows(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
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

	flows, total, err := h.service.ListFlows(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"flows":  flows,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetFlow godoc
//
//	@Summary		Get flow details
//	@Description	Get details of a specific flow
//	@Tags			flows
//	@Produce		json
//	@Param			id	path		string	true	"Flow ID"
//	@Success		200	{object}	Flow
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows/{id} [get]
func (h *Handler) GetFlow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	flowID := c.Param("id")
	flow, err := h.service.GetFlow(c.Request.Context(), tenantID, flowID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if flow == nil {
		c.JSON(http.StatusNotFound, errorResponse{Code: "FLOW_NOT_FOUND", Message: "flow not found"})
		return
	}

	c.JSON(http.StatusOK, flow)
}

// UpdateFlow godoc
//
//	@Summary		Update flow
//	@Description	Update an existing flow
//	@Tags			flows
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Flow ID"
//	@Param			request	body		UpdateFlowRequest	true	"Flow update request"
//	@Success		200		{object}	Flow
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows/{id} [put]
func (h *Handler) UpdateFlow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	flowID := c.Param("id")
	var req UpdateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	flow, err := h.service.UpdateFlow(c.Request.Context(), tenantID, flowID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flow)
}

// DeleteFlow godoc
//
//	@Summary		Delete flow
//	@Description	Delete a flow
//	@Tags			flows
//	@Produce		json
//	@Param			id	path	string	true	"Flow ID"
//	@Success		204	"No content"
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows/{id} [delete]
func (h *Handler) DeleteFlow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	flowID := c.Param("id")
	if err := h.service.DeleteFlow(c.Request.Context(), tenantID, flowID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ExecuteFlow godoc
//
//	@Summary		Execute flow
//	@Description	Execute a flow with the given input
//	@Tags			flows
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Flow ID"
//	@Param			request	body		ExecuteFlowRequest	true	"Execution request"
//	@Success		200		{object}	FlowExecution
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows/{id}/execute [post]
func (h *Handler) ExecuteFlow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	flowID := c.Param("id")
	var req ExecuteFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	execution, err := h.service.ExecuteFlow(c.Request.Context(), tenantID, flowID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// ListExecutions godoc
//
//	@Summary		List flow executions
//	@Description	Get execution history for a flow
//	@Tags			flows
//	@Produce		json
//	@Param			id		path		string	true	"Flow ID"
//	@Param			limit	query		int		false	"Limit"		default(20)
//	@Param			offset	query		int		false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows/{id}/executions [get]
func (h *Handler) ListExecutions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	flowID := c.Param("id")
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	executions, total, err := h.service.ListExecutions(c.Request.Context(), tenantID, flowID, limit, offset)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// GetExecution godoc
//
//	@Summary		Get execution details
//	@Description	Get details of a specific flow execution
//	@Tags			flows
//	@Produce		json
//	@Param			id	path		string	true	"Execution ID"
//	@Success		200	{object}	FlowExecution
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/executions/{id} [get]
func (h *Handler) GetExecution(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	executionID := c.Param("id")
	execution, err := h.service.GetExecution(c.Request.Context(), tenantID, executionID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	if execution == nil {
		c.JSON(http.StatusNotFound, errorResponse{Code: "EXECUTION_NOT_FOUND", Message: "execution not found"})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// GetTemplates godoc
//
//	@Summary		Get flow templates
//	@Description	Get available flow templates
//	@Tags			flows
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows/templates [get]
func (h *Handler) GetTemplates(c *gin.Context) {
	templates := h.service.GetTemplates()
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// ValidateFlow godoc
//
//	@Summary		Validate flow
//	@Description	Validate a flow structure without creating it
//	@Tags			flows
//	@Accept			json
//	@Produce		json
//	@Param			request	body		Flow	true	"Flow to validate"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/flows/validate [post]
func (h *Handler) ValidateFlow(c *gin.Context) {
	var flow Flow
	if err := c.ShouldBindJSON(&flow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.service.ValidateFlow(&flow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// AssignFlowRequest represents a request to assign a flow
type AssignFlowRequest struct {
	FlowID   string `json:"flow_id" binding:"required"`
	Priority int    `json:"priority"`
}

// AssignFlow godoc
//
//	@Summary		Assign flow to endpoint
//	@Description	Assign a flow to process webhooks for an endpoint
//	@Tags			flows
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Endpoint ID"
//	@Param			request	body		AssignFlowRequest	true	"Assignment request"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/endpoints/{id}/flows [post]
func (h *Handler) AssignFlow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	var req AssignFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.service.AssignFlowToEndpoint(c.Request.Context(), tenantID, endpointID, req.FlowID, req.Priority); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "flow assigned successfully"})
}

// GetEndpointFlows godoc
//
//	@Summary		Get endpoint flows
//	@Description	Get flows assigned to an endpoint
//	@Tags			flows
//	@Produce		json
//	@Param			id	path		string	true	"Endpoint ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/endpoints/{id}/flows [get]
func (h *Handler) GetEndpointFlows(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	assignments, err := h.service.GetEndpointFlows(c.Request.Context(), endpointID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"flows": assignments})
}

// RemoveFlow godoc
//
//	@Summary		Remove flow from endpoint
//	@Description	Remove a flow assignment from an endpoint
//	@Tags			flows
//	@Produce		json
//	@Param			id		path	string	true	"Endpoint ID"
//	@Param			flowId	path	string	true	"Flow ID"
//	@Success		204		"No content"
//	@Failure		401		{object}	map[string]interface{}
//	@Security		ApiKeyAuth
//	@Router			/endpoints/{id}/flows/{flowId} [delete]
func (h *Handler) RemoveFlow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: "unauthorized"})
		return
	}

	endpointID := c.Param("id")
	flowID := c.Param("flowId")

	if err := h.service.RemoveFlowFromEndpoint(c.Request.Context(), endpointID, flowID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
