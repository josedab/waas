package onboarding

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for onboarding.
type Handler struct {
	service *Service
}

// NewHandler creates a new onboarding handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers onboarding routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/onboarding")
	{
		g.POST("/start", h.StartOnboarding)
		g.GET("/progress", h.GetProgress)
		g.POST("/step/complete", h.CompleteStep)
		g.POST("/step/skip", h.SkipStep)
		g.POST("/snippets", h.GetSnippets)
		g.GET("/analytics", h.GetAnalytics)
	}
}

func (h *Handler) StartOnboarding(c *gin.Context) {
	var req StartOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.service.StartOnboarding(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, progress)
}

func (h *Handler) GetProgress(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	progress, err := h.service.GetProgress(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

func (h *Handler) CompleteStep(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CompleteStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.service.CompleteStep(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

func (h *Handler) SkipStep(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CompleteStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	progress, err := h.service.SkipStep(c.Request.Context(), tenantID, req.StepID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}

func (h *Handler) GetSnippets(c *gin.Context) {
	var req GetSnippetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	snippets, err := h.service.GetSnippets(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snippets": snippets})
}

func (h *Handler) GetAnalytics(c *gin.Context) {
	analytics, err := h.service.GetAnalytics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}
