package tfprovider

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for the Terraform provider API
type Handler struct {
	service *Service
}

// NewHandler creates a new Terraform provider handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers Terraform provider routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	tf := router.Group("/terraform")
	{
		tf.GET("/schemas", h.GetSchemas)
		tf.POST("/import", h.ImportResource)
		tf.GET("/resources", h.ListResources)
		tf.POST("/resources", h.RegisterResource)
		tf.DELETE("/resources/:type/:id", h.DeregisterResource)
	}
}

// @Summary Get resource schemas
// @Tags Terraform
// @Produce json
// @Success 200 {object} map[string][]ResourceSchema
// @Router /terraform/schemas [get]
func (h *Handler) GetSchemas(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"schemas": h.service.GetResourceSchemas()})
}

// @Summary Import an existing resource
// @Tags Terraform
// @Accept json
// @Produce json
// @Param body body StateImportRequest true "Resource to import"
// @Success 200 {object} StateExport
// @Router /terraform/import [post]
func (h *Handler) ImportResource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req StateImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	export, err := h.service.ImportResource(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "IMPORT_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, export)
}

// @Summary List managed resources
// @Tags Terraform
// @Produce json
// @Param type query string false "Resource type filter"
// @Success 200 {object} map[string][]ManagedResource
// @Router /terraform/resources [get]
func (h *Handler) ListResources(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	resourceType := ResourceType(c.Query("type"))

	resources, err := h.service.ListManagedResources(c.Request.Context(), tenantID, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"resources": resources})
}

// @Summary Register a resource as managed
// @Tags Terraform
// @Accept json
// @Produce json
// @Success 201 {object} ManagedResource
// @Router /terraform/resources [post]
func (h *Handler) RegisterResource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		ResourceType ResourceType      `json:"resource_type" binding:"required"`
		ResourceID   string            `json:"resource_id" binding:"required"`
		Attributes   map[string]string `json:"attributes"`
		ManagedBy    string            `json:"managed_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	if req.ManagedBy == "" {
		req.ManagedBy = "terraform"
	}

	resource, err := h.service.RegisterResource(c.Request.Context(), tenantID, req.ResourceType, req.ResourceID, req.Attributes, req.ManagedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "REGISTER_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, resource)
}

// @Summary Deregister a resource from IaC management
// @Tags Terraform
// @Param type path string true "Resource type"
// @Param id path string true "Resource ID"
// @Success 204
// @Router /terraform/resources/{type}/{id} [delete]
func (h *Handler) DeregisterResource(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	resourceType := ResourceType(c.Param("type"))
	resourceID := c.Param("id")

	if err := h.service.DeregisterResource(c.Request.Context(), tenantID, resourceType, resourceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DEREGISTER_FAILED", "message": err.Error()}})
		return
	}

	c.Status(http.StatusNoContent)
}
