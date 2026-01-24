package flowbuilder

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pkgerrors "webhook-platform/pkg/errors"
)

// Handler implements HTTP handlers for the visual workflow builder
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	wf := r.Group("/flow-builder")
	{
		wf.POST("/workflows", h.CreateWorkflow)
		wf.GET("/workflows", h.ListWorkflows)
		wf.GET("/workflows/:id", h.GetWorkflow)
		wf.PUT("/workflows/:id", h.UpdateWorkflow)
		wf.DELETE("/workflows/:id", h.DeleteWorkflow)
		wf.POST("/workflows/:id/activate", h.ActivateWorkflow)
		wf.POST("/workflows/:id/pause", h.PauseWorkflow)

		wf.POST("/workflows/:id/execute", h.ExecuteWorkflow)
		wf.GET("/workflows/:id/executions", h.ListExecutions)
		wf.GET("/executions/:execId", h.GetExecution)

		wf.POST("/workflows/validate", h.ValidateWorkflow)
		wf.GET("/workflows/:id/analytics", h.GetAnalytics)

		wf.GET("/templates", h.ListTemplates)
		wf.GET("/templates/:id", h.GetTemplate)
	}
}

func (h *Handler) CreateWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	workflow, err := h.service.CreateWorkflow(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusCreated, workflow)
}

func (h *Handler) GetWorkflow(c *gin.Context) {
	id := c.Param("id")
	workflow, err := h.service.GetWorkflow(c.Request.Context(), id)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "workflow")
		return
	}
	c.JSON(http.StatusOK, workflow)
}

func (h *Handler) UpdateWorkflow(c *gin.Context) {
	id := c.Param("id")
	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	workflow, err := h.service.UpdateWorkflow(c.Request.Context(), id, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, workflow)
}

func (h *Handler) ListWorkflows(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := WorkflowStatus(c.Query("status"))

	workflows, total, err := h.service.ListWorkflows(c.Request.Context(), tenantID, status, page, pageSize)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"workflows": workflows, "total": total})
}

func (h *Handler) DeleteWorkflow(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteWorkflow(c.Request.Context(), id); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "workflow archived"})
}

func (h *Handler) ActivateWorkflow(c *gin.Context) {
	id := c.Param("id")
	workflow, err := h.service.ActivateWorkflow(c.Request.Context(), id)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, workflow)
}

func (h *Handler) PauseWorkflow(c *gin.Context) {
	id := c.Param("id")
	workflow, err := h.service.PauseWorkflow(c.Request.Context(), id)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, workflow)
}

func (h *Handler) ExecuteWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workflowID := c.Param("id")

	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	exec, err := h.service.ExecuteWorkflow(c.Request.Context(), tenantID, workflowID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, exec)
}

func (h *Handler) GetExecution(c *gin.Context) {
	id := c.Param("execId")
	exec, err := h.service.GetExecution(c.Request.Context(), id)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "execution")
		return
	}
	c.JSON(http.StatusOK, exec)
}

func (h *Handler) ListExecutions(c *gin.Context) {
	workflowID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	execs, total, err := h.service.ListExecutions(c.Request.Context(), workflowID, page, pageSize)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"executions": execs, "total": total})
}

func (h *Handler) ValidateWorkflow(c *gin.Context) {
	var req ValidateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	result := h.service.ValidateDAG(req.Nodes, req.Edges)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetAnalytics(c *gin.Context) {
	workflowID := c.Param("id")
	analytics, err := h.service.GetAnalytics(c.Request.Context(), workflowID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, analytics)
}

func (h *Handler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	templates, total, err := h.service.ListTemplates(c.Request.Context(), category, page, pageSize)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": templates, "total": total})
}

func (h *Handler) GetTemplate(c *gin.Context) {
	id := c.Param("id")
	template, err := h.service.GetTemplate(c.Request.Context(), id)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "template")
		return
	}
	c.JSON(http.StatusOK, template)
}
