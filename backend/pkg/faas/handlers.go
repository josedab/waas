package faas

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the FaaS runtime.
type Handler struct {
	service *Service
}

// NewHandler creates a new FaaS handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers FaaS routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/faas")
	{
		g.POST("/functions", h.CreateFunction)
		g.GET("/functions", h.ListFunctions)
		g.GET("/functions/:id", h.GetFunction)
		g.PUT("/functions/:id", h.UpdateFunction)
		g.DELETE("/functions/:id", h.DeleteFunction)
		g.POST("/functions/:id/invoke", h.InvokeFunction)
		g.GET("/functions/:id/metrics", h.GetMetrics)
		g.GET("/functions/:id/executions", h.ListExecutions)
		g.GET("/templates", h.ListTemplates)
	}
}

func (h *Handler) CreateFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fn, err := h.service.CreateFunction(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, fn)
}

func (h *Handler) ListFunctions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	fns, err := h.service.ListFunctions(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"functions": fns})
}

func (h *Handler) GetFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	fn, err := h.service.GetFunction(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fn)
}

func (h *Handler) UpdateFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fn, err := h.service.UpdateFunction(c.Request.Context(), tenantID, c.Param("id"), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fn)
}

func (h *Handler) DeleteFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if err := h.service.DeleteFunction(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) InvokeFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req InvokeFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.InvokeFunction(c.Request.Context(), tenantID, c.Param("id"), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	metrics, err := h.service.GetMetrics(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *Handler) ListExecutions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	// Verify tenant owns function
	if _, err := h.service.GetFunction(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	execs, _ := h.service.repo.ListExecutions(c.Request.Context(), c.Param("id"), limit)
	c.JSON(http.StatusOK, gin.H{"executions": execs})
}

func (h *Handler) ListTemplates(c *gin.Context) {
	templates := h.service.ListTemplates()
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}
