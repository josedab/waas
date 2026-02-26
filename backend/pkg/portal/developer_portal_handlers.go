package portal

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterDeveloperPortalRoutes registers the unified developer portal routes
func (h *Handler) RegisterDeveloperPortalRoutes(router *gin.RouterGroup) {
	dp := router.Group("/developer-portal")
	{
		// Unified dashboard
		dp.GET("/dashboard", h.GetDashboard)

		// Onboarding wizard
		dp.POST("/onboarding/start", h.StartOnboarding)
		dp.POST("/onboarding/complete-step", h.CompleteOnboardingStep)
		dp.GET("/onboarding/progress", h.GetOnboardingProgress)

		// Interactive API explorer
		dp.GET("/explorer", h.GetAPIExplorer)
		dp.POST("/explorer/try", h.TryAPIEndpoint)

		// SDK code generation (all 12 languages)
		dp.POST("/sdk/generate", h.GenerateSDKCode)
		dp.GET("/sdk/languages", h.ListSDKLanguages)
	}
}

// @Summary Get unified developer portal dashboard
// @Tags DeveloperPortal
// @Produce json
// @Success 200 {object} UnifiedPortalView
// @Router /developer-portal/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	apiURL := c.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.waas.dev"
	}

	view, err := h.service.GetUnifiedPortalView(c.Request.Context(), tenantID, apiURL)
	if err != nil {
		httputil.InternalError(c, "DASHBOARD_ERROR", err)
		return
	}

	c.JSON(http.StatusOK, view)
}

// @Summary Start onboarding wizard
// @Tags DeveloperPortal
// @Accept json
// @Produce json
// @Param body body StartOnboardingRequest true "Onboarding config"
// @Success 201 {object} OnboardingWizard
// @Router /developer-portal/onboarding/start [post]
func (h *Handler) StartOnboarding(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req StartOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	wizard, err := h.service.StartOnboarding(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "ONBOARDING_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, wizard)
}

// @Summary Complete an onboarding step
// @Tags DeveloperPortal
// @Accept json
// @Produce json
// @Param body body CompleteStepRequest true "Step completion"
// @Success 200 {object} OnboardingWizard
// @Router /developer-portal/onboarding/complete-step [post]
func (h *Handler) CompleteOnboardingStep(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CompleteStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	// Start a new wizard for demo purposes if none exists
	wizard, err := h.service.StartOnboarding(c.Request.Context(), tenantID, &StartOnboardingRequest{TenantName: "default"})
	if err != nil {
		httputil.InternalError(c, "ONBOARDING_ERROR", err)
		return
	}

	updated, err := h.service.CompleteOnboardingStep(c.Request.Context(), tenantID, wizard, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "STEP_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Get onboarding progress
// @Tags DeveloperPortal
// @Produce json
// @Success 200
// @Router /developer-portal/onboarding/progress [get]
func (h *Handler) GetOnboardingProgress(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	wizard, _ := h.service.StartOnboarding(c.Request.Context(), tenantID, &StartOnboardingRequest{TenantName: "default"})
	progress := h.service.GetOnboardingProgress(wizard)
	c.JSON(http.StatusOK, progress)
}

// @Summary Get interactive API explorer
// @Tags DeveloperPortal
// @Produce json
// @Success 200 {object} APIExplorerConfig
// @Router /developer-portal/explorer [get]
func (h *Handler) GetAPIExplorer(c *gin.Context) {
	apiURL := c.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.waas.dev"
	}

	explorer := h.service.GetAPIExplorer(c.Request.Context(), apiURL)
	c.JSON(http.StatusOK, explorer)
}

// @Summary Try an API endpoint from the explorer
// @Tags DeveloperPortal
// @Accept json
// @Produce json
// @Param body body TryEndpointRequest true "API request"
// @Success 200 {object} TryEndpointResponse
// @Router /developer-portal/explorer/try [post]
func (h *Handler) TryAPIEndpoint(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	apiKey := c.GetString("api_key")
	apiURL := c.GetString("api_url")
	if apiURL == "" {
		apiURL = "https://api.waas.dev"
	}

	var req TryEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	resp, err := h.service.TryAPIEndpoint(c.Request.Context(), apiURL, tenantID, apiKey, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Generate SDK code
// @Tags DeveloperPortal
// @Accept json
// @Produce json
// @Param body body SDKCodeGenRequest true "SDK generation request"
// @Success 200 {object} SDKCodeGenResponse
// @Router /developer-portal/sdk/generate [post]
func (h *Handler) GenerateSDKCode(c *gin.Context) {
	var req SDKCodeGenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	resp, err := h.service.GenerateSDKCode(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "GENERATION_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary List available SDK languages
// @Tags DeveloperPortal
// @Produce json
// @Success 200
// @Router /developer-portal/sdk/languages [get]
func (h *Handler) ListSDKLanguages(c *gin.Context) {
	languages := []map[string]string{
		{"id": "go", "name": "Go", "framework": "net/http"},
		{"id": "python", "name": "Python", "framework": "Flask"},
		{"id": "nodejs", "name": "Node.js", "framework": "Express"},
		{"id": "typescript", "name": "TypeScript", "framework": "Express"},
		{"id": "java", "name": "Java", "framework": "Spring Boot"},
		{"id": "ruby", "name": "Ruby", "framework": "Sinatra"},
		{"id": "php", "name": "PHP", "framework": "PHP"},
		{"id": "csharp", "name": "C#", "framework": "ASP.NET Core"},
		{"id": "rust", "name": "Rust", "framework": "Actix-web"},
		{"id": "kotlin", "name": "Kotlin", "framework": "Ktor"},
		{"id": "swift", "name": "Swift", "framework": "Vapor"},
		{"id": "elixir", "name": "Elixir", "framework": "Phoenix"},
	}
	c.JSON(http.StatusOK, gin.H{"languages": languages, "total": len(languages)})
}
