package edge

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// WasmHandler provides HTTP handlers for Wasm edge functions
type WasmHandler struct {
	runtime *WasmRuntime
}

// NewWasmHandler creates a new handler
func NewWasmHandler(runtime *WasmRuntime) *WasmHandler {
	return &WasmHandler{runtime: runtime}
}

// RegisterWasmRoutes registers Wasm runtime routes
func (h *WasmHandler) RegisterWasmRoutes(r *gin.RouterGroup) {
	edge := r.Group("/edge/functions")
	{
		edge.POST("", h.CreateFunction)
		edge.GET("", h.ListFunctions)
		edge.GET("/:functionId", h.GetFunction)
		edge.DELETE("/:functionId", h.DeleteFunction)
		edge.POST("/:functionId/invoke", h.InvokeFunction)
		edge.GET("/:functionId/invocations", h.ListInvocations)
		edge.GET("/templates", h.ListTemplates)
	}
}

// CreateFunction creates an edge function
// @Summary Create edge function
// @Tags edge
// @Accept json
// @Produce json
// @Success 201 {object} WasmFunction
// @Router /edge/functions [post]
func (h *WasmHandler) CreateFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateWasmFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fn, err := h.runtime.CreateFunction(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, fn)
}

// ListFunctions lists edge functions
// @Summary List edge functions
// @Tags edge
// @Produce json
// @Success 200 {array} WasmFunction
// @Router /edge/functions [get]
func (h *WasmHandler) ListFunctions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	functions, err := h.runtime.ListFunctions(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, functions)
}

// GetFunction retrieves an edge function
// @Summary Get edge function
// @Tags edge
// @Produce json
// @Param functionId path string true "Function ID"
// @Success 200 {object} WasmFunction
// @Router /edge/functions/{functionId} [get]
func (h *WasmHandler) GetFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	functionID := c.Param("functionId")
	fn, err := h.runtime.GetFunction(c.Request.Context(), tenantID, functionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "function not found"})
		return
	}

	c.JSON(http.StatusOK, fn)
}

// DeleteFunction deletes an edge function
// @Summary Delete edge function
// @Tags edge
// @Param functionId path string true "Function ID"
// @Success 204 "No content"
// @Router /edge/functions/{functionId} [delete]
func (h *WasmHandler) DeleteFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	functionID := c.Param("functionId")
	if err := h.runtime.DeleteFunction(c.Request.Context(), tenantID, functionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// InvokeFunction invokes an edge function
// @Summary Invoke edge function
// @Tags edge
// @Accept json
// @Produce json
// @Param functionId path string true "Function ID"
// @Success 200 {object} FunctionInvocation
// @Router /edge/functions/{functionId}/invoke [post]
func (h *WasmHandler) InvokeFunction(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	functionID := c.Param("functionId")
	var req WasmInvokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invocation, err := h.runtime.InvokeFunction(c.Request.Context(), tenantID, functionID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invocation)
}

// ListInvocations lists function invocations
// @Summary List function invocations
// @Tags edge
// @Produce json
// @Param functionId path string true "Function ID"
// @Param limit query int false "Limit"
// @Success 200 {array} FunctionInvocation
// @Router /edge/functions/{functionId}/invocations [get]
func (h *WasmHandler) ListInvocations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	functionID := c.Param("functionId")
	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	invocations, err := h.runtime.ListInvocations(c.Request.Context(), tenantID, functionID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invocations)
}

// ListTemplates lists available function templates
// @Summary List function templates
// @Tags edge
// @Produce json
// @Success 200 {array} FunctionTemplate
// @Router /edge/functions/templates [get]
func (h *WasmHandler) ListTemplates(c *gin.Context) {
	templates, err := h.runtime.ListTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, templates)
}
