package marketplacetpl

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for the marketplace
type Handler struct {
	service *Service
}

// NewHandler creates a new marketplace handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers marketplace routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	mp := router.Group("/marketplace")
	{
		mp.GET("/templates", h.ListTemplates)
		mp.GET("/templates/builtin", h.GetBuiltinTemplates)
		mp.GET("/templates/search", h.SearchTemplates)
		mp.GET("/templates/:id", h.GetTemplate)
		mp.POST("/templates", h.CreateTemplate)
		mp.GET("/stats", h.GetStats)

		// Installations
		mp.POST("/templates/:id/install", h.InstallTemplate)
		mp.GET("/installations", h.ListInstallations)
		mp.DELETE("/installations/:id", h.UninstallTemplate)

		// Reviews
		mp.POST("/templates/:id/reviews", h.SubmitReview)
		mp.GET("/templates/:id/reviews", h.ListReviews)
	}
}

// @Summary List marketplace templates
// @Tags Marketplace
// @Produce json
// @Param category query string false "Filter by category"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /marketplace/templates [get]
func (h *Handler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	templates, total, err := h.service.ListTemplates(c.Request.Context(), category, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates, "total": total})
}

// @Summary Get built-in templates
// @Tags Marketplace
// @Produce json
// @Success 200 {object} map[string][]Template
// @Router /marketplace/templates/builtin [get]
func (h *Handler) GetBuiltinTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"templates": h.service.GetBuiltinTemplates()})
}

// @Summary Search templates
// @Tags Marketplace
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /marketplace/templates/search [get]
func (h *Handler) SearchTemplates(c *gin.Context) {
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	templates, total, err := h.service.SearchTemplates(c.Request.Context(), query, limit, offset)
	if err != nil {
		httputil.InternalError(c, "SEARCH_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates, "total": total})
}

// @Summary Get a template
// @Tags Marketplace
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} Template
// @Router /marketplace/templates/{id} [get]
func (h *Handler) GetTemplate(c *gin.Context) {
	templateID := c.Param("id")

	template, err := h.service.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, template)
}

// @Summary Submit a template
// @Tags Marketplace
// @Accept json
// @Produce json
// @Param body body CreateTemplateRequest true "Template definition"
// @Success 201 {object} Template
// @Router /marketplace/templates [post]
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	template, err := h.service.CreateTemplate(c.Request.Context(), &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, template)
}

// @Summary Install a template
// @Tags Marketplace
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param body body InstallTemplateRequest true "Installation configuration"
// @Success 201 {object} Installation
// @Router /marketplace/templates/{id}/install [post]
func (h *Handler) InstallTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	templateID := c.Param("id")

	var req InstallTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	install, err := h.service.InstallTemplate(c.Request.Context(), tenantID, templateID, &req)
	if err != nil {
		httputil.InternalError(c, "INSTALL_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, install)
}

// @Summary List installations
// @Tags Marketplace
// @Produce json
// @Success 200 {object} map[string][]Installation
// @Router /marketplace/installations [get]
func (h *Handler) ListInstallations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	installs, err := h.service.ListInstallations(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"installations": installs})
}

// @Summary Uninstall a template
// @Tags Marketplace
// @Param id path string true "Installation ID"
// @Success 204
// @Router /marketplace/installations/{id} [delete]
func (h *Handler) UninstallTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	installID := c.Param("id")

	if err := h.service.UninstallTemplate(c.Request.Context(), tenantID, installID); err != nil {
		httputil.InternalError(c, "UNINSTALL_FAILED", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Submit a review
// @Tags Marketplace
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param body body SubmitReviewRequest true "Review"
// @Success 201 {object} Review
// @Router /marketplace/templates/{id}/reviews [post]
func (h *Handler) SubmitReview(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	templateID := c.Param("id")

	var req SubmitReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	review, err := h.service.SubmitReview(c.Request.Context(), tenantID, templateID, &req)
	if err != nil {
		httputil.InternalError(c, "REVIEW_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, review)
}

// @Summary List reviews for a template
// @Tags Marketplace
// @Produce json
// @Param id path string true "Template ID"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string][]Review
// @Router /marketplace/templates/{id}/reviews [get]
func (h *Handler) ListReviews(c *gin.Context) {
	templateID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	reviews, err := h.service.ListReviews(c.Request.Context(), templateID, limit, offset)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"reviews": reviews})
}

// @Summary Get marketplace statistics
// @Tags Marketplace
// @Produce json
// @Success 200 {object} MarketplaceStats
// @Router /marketplace/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		httputil.InternalError(c, "STATS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
