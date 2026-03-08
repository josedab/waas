package workflow

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for workflow operations
type Handlers struct {
	service *Service
}

// NewHandlers creates new workflow handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers workflow routes
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	workflows := r.Group("/workflows")
	{
		workflows.GET("", h.ListWorkflows)
		workflows.POST("", h.CreateWorkflow)
		workflows.GET("/:id", h.GetWorkflow)
		workflows.PUT("/:id", h.UpdateWorkflow)
		workflows.DELETE("/:id", h.DeleteWorkflow)

		workflows.POST("/:id/publish", h.PublishWorkflow)
		workflows.POST("/:id/unpublish", h.UnpublishWorkflow)
		workflows.POST("/:id/clone", h.CloneWorkflow)
		workflows.POST("/:id/validate", h.ValidateWorkflow)

		workflows.GET("/:id/versions", h.ListWorkflowVersions)
		workflows.GET("/:id/versions/:version", h.GetWorkflowVersion)

		workflows.POST("/:id/execute", h.ExecuteWorkflow)
		workflows.GET("/:id/executions", h.ListExecutions)
		workflows.GET("/:id/stats", h.GetWorkflowStats)
	}

	executions := r.Group("/executions")
	{
		executions.GET("/:id", h.GetExecution)
		executions.POST("/:id/cancel", h.CancelExecution)
	}

	r.GET("/workflow-catalog", h.GetNodeCatalog)
	r.GET("/workflow-templates", h.ListTemplates)
	r.GET("/workflow-templates/:id", h.GetTemplate)
}

// ListWorkflows lists workflows for a tenant
// @Summary List workflows
// @Tags Workflows
// @Produce json
// @Param status query string false "Filter by status"
// @Param search query string false "Search by name"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} Workflow
// @Router /workflows [get]
func (h *Handlers) ListWorkflows(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	filter := &WorkflowFilter{
		Search: c.Query("search"),
	}

	if status := c.Query("status"); status != "" {
		s := WorkflowStatus(status)
		filter.Status = &s
	}

	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(c.Query("offset")); err == nil {
		filter.Offset = offset
	}

	workflows, err := h.service.ListWorkflows(c.Request.Context(), tenantID, filter)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, workflows)
}

// CreateWorkflow creates a new workflow
// @Summary Create workflow
// @Tags Workflows
// @Accept json
// @Produce json
// @Param request body CreateWorkflowRequest true "Workflow configuration"
// @Success 201 {object} Workflow
// @Router /workflows [post]
func (h *Handlers) CreateWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, err := h.service.CreateWorkflow(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, workflow)
}

// GetWorkflow gets a workflow by ID
// @Summary Get workflow
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} Workflow
// @Router /workflows/{id} [get]
func (h *Handlers) GetWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	workflow, err := h.service.GetWorkflow(c.Request.Context(), tenantID, workflowID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// UpdateWorkflow updates a workflow
// @Summary Update workflow
// @Tags Workflows
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param request body UpdateWorkflowRequest true "Update configuration"
// @Success 200 {object} Workflow
// @Router /workflows/{id} [put]
func (h *Handlers) UpdateWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	var req UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, err := h.service.UpdateWorkflow(c.Request.Context(), tenantID, workflowID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// DeleteWorkflow deletes a workflow
// @Summary Delete workflow
// @Tags Workflows
// @Param id path string true "Workflow ID"
// @Success 204
// @Router /workflows/{id} [delete]
func (h *Handlers) DeleteWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	if err := h.service.DeleteWorkflow(c.Request.Context(), tenantID, workflowID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// PublishWorkflow publishes a workflow
// @Summary Publish workflow
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} Workflow
// @Router /workflows/{id}/publish [post]
func (h *Handlers) PublishWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	workflow, err := h.service.PublishWorkflow(c.Request.Context(), tenantID, workflowID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// UnpublishWorkflow unpublishes a workflow
// @Summary Unpublish workflow
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} Workflow
// @Router /workflows/{id}/unpublish [post]
func (h *Handlers) UnpublishWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	workflow, err := h.service.UnpublishWorkflow(c.Request.Context(), tenantID, workflowID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// CloneWorkflow creates a copy of a workflow
// @Summary Clone workflow
// @Tags Workflows
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param request body object{name=string} true "New workflow name"
// @Success 201 {object} Workflow
// @Router /workflows/{id}/clone [post]
func (h *Handlers) CloneWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, err := h.service.CloneWorkflow(c.Request.Context(), tenantID, workflowID, req.Name)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, workflow)
}

// ValidateWorkflow validates a workflow
// @Summary Validate workflow
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} ValidationResult
// @Router /workflows/{id}/validate [post]
func (h *Handlers) ValidateWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	workflow, err := h.service.GetWorkflow(c.Request.Context(), tenantID, workflowID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	result := h.service.ValidateWorkflow(c.Request.Context(), workflow)
	c.JSON(http.StatusOK, result)
}

// ListWorkflowVersions lists workflow versions
// @Summary List workflow versions
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {array} WorkflowVersionInfo
// @Router /workflows/{id}/versions [get]
func (h *Handlers) ListWorkflowVersions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	versions, err := h.service.ListWorkflowVersions(c.Request.Context(), tenantID, workflowID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

// GetWorkflowVersion gets a specific workflow version
// @Summary Get workflow version
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param version path int true "Version number"
// @Success 200 {object} Workflow
// @Router /workflows/{id}/versions/{version} [get]
func (h *Handlers) GetWorkflowVersion(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")
	version, _ := strconv.Atoi(c.Param("version"))

	workflow, err := h.service.GetWorkflowVersion(c.Request.Context(), tenantID, workflowID, version)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// ExecuteWorkflow executes a workflow
// @Summary Execute workflow
// @Tags Workflows
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param request body ExecuteWorkflowRequest true "Execution input"
// @Success 201 {object} WorkflowExecution
// @Router /workflows/{id}/execute [post]
func (h *Handlers) ExecuteWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is OK
		req = ExecuteWorkflowRequest{}
	}

	execution, err := h.service.ExecuteWorkflow(c.Request.Context(), tenantID, workflowID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, execution)
}

// ListExecutions lists workflow executions
// @Summary List workflow executions
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} WorkflowExecution
// @Router /workflows/{id}/executions [get]
func (h *Handlers) ListExecutions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	filter := &ExecutionFilter{}
	if status := c.Query("status"); status != "" {
		s := ExecutionStatus(status)
		filter.Status = &s
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(c.Query("offset")); err == nil {
		filter.Offset = offset
	}

	executions, err := h.service.ListExecutions(c.Request.Context(), tenantID, workflowID, filter)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetExecution gets an execution by ID
// @Summary Get execution
// @Tags Executions
// @Produce json
// @Param id path string true "Execution ID"
// @Success 200 {object} WorkflowExecution
// @Router /executions/{id} [get]
func (h *Handlers) GetExecution(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	execID := c.Param("id")

	execution, err := h.service.GetExecution(c.Request.Context(), tenantID, execID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, execution)
}

// CancelExecution cancels an execution
// @Summary Cancel execution
// @Tags Executions
// @Produce json
// @Param id path string true "Execution ID"
// @Success 200 {object} WorkflowExecution
// @Router /executions/{id}/cancel [post]
func (h *Handlers) CancelExecution(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	execID := c.Param("id")

	execution, err := h.service.CancelExecution(c.Request.Context(), tenantID, execID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, execution)
}

// GetWorkflowStats gets workflow statistics
// @Summary Get workflow stats
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} WorkflowStats
// @Router /workflows/{id}/stats [get]
func (h *Handlers) GetWorkflowStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	stats, err := h.service.GetWorkflowStats(c.Request.Context(), tenantID, workflowID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetNodeCatalog returns the node catalog
// @Summary Get node catalog
// @Tags Workflows
// @Produce json
// @Success 200 {object} NodeCatalog
// @Router /workflow-catalog [get]
func (h *Handlers) GetNodeCatalog(c *gin.Context) {
	catalog := h.service.GetNodeCatalog()
	c.JSON(http.StatusOK, catalog)
}

// ListTemplates lists workflow templates
// @Summary List workflow templates
// @Tags Workflows
// @Produce json
// @Param category query string false "Filter by category"
// @Success 200 {array} WorkflowTemplate
// @Router /workflow-templates [get]
func (h *Handlers) ListTemplates(c *gin.Context) {
	category := c.Query("category")

	templates, err := h.service.ListTemplates(c.Request.Context(), category)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, templates)
}

// GetTemplate gets a workflow template
// @Summary Get workflow template
// @Tags Workflows
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} WorkflowTemplate
// @Router /workflow-templates/{id} [get]
func (h *Handlers) GetTemplate(c *gin.Context) {
	templateID := c.Param("id")

	template, err := h.service.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, template)
}
