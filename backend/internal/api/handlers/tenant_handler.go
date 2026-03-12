package handlers

import (
	"github.com/josedab/waas/pkg/auth"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantHandler handles tenant CRUD, subscription management, and API key regeneration.
type TenantHandler struct {
	tenantRepo repository.TenantRepository
	logger     *utils.Logger
}

// CreateTenantRequest is the request payload for creating a new tenant account.
type CreateTenantRequest struct {
	Name               string `json:"name" binding:"required,min=1,max=255"`
	SubscriptionTier   string `json:"subscription_tier" binding:"required,oneof=free basic premium enterprise"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute,omitempty"`
	MonthlyQuota       int    `json:"monthly_quota,omitempty"`
}

// CreateTenantResponse is returned after successful tenant creation with the API key.
type CreateTenantResponse struct {
	Tenant *models.Tenant `json:"tenant"`
	APIKey string         `json:"api_key"`
}

// UpdateTenantRequest is the request payload for self-service tenant updates.
type UpdateTenantRequest struct {
	Name         string `json:"name,omitempty"`
	MonthlyQuota int    `json:"monthly_quota,omitempty"`
}

// validSubscriptionTiers are the tiers allowed for tenant accounts.
var validSubscriptionTiers = map[string]bool{
	"free": true, "basic": true, "premium": true, "enterprise": true,
}

// AdminUpdateTenantRequest is the request payload for admin-level tenant updates.
type AdminUpdateTenantRequest struct {
	Name               string `json:"name,omitempty"`
	SubscriptionTier   string `json:"subscription_tier,omitempty"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute,omitempty"`
	MonthlyQuota       int    `json:"monthly_quota,omitempty"`
}

func NewTenantHandler(tenantRepo repository.TenantRepository, logger *utils.Logger) *TenantHandler {
	return &TenantHandler{
		tenantRepo: tenantRepo,
		logger:     logger,
	}
}

// CreateTenant creates a new tenant account
// @Summary Create tenant
// @Description Register a new tenant account and receive an API key for authentication
// @Tags tenants
// @Accept json
// @Produce json
// @Param request body CreateTenantRequest true "Tenant creation request"
// @Success 201 {object} CreateTenantResponse "Tenant created successfully with API key"
// @Failure 400 {object} map[string]interface{} "Invalid request format or validation error"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /tenants [post]
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: "Invalid request format", Details: err.Error()})
		return
	}

	// Set default values based on subscription tier
	rateLimitPerMinute := req.RateLimitPerMinute
	monthlyQuota := req.MonthlyQuota

	if rateLimitPerMinute == 0 {
		rateLimitPerMinute = getDefaultRateLimit(req.SubscriptionTier)
	}
	if monthlyQuota == 0 {
		monthlyQuota = getDefaultMonthlyQuota(req.SubscriptionTier)
	}

	// Generate API key and hash
	apiKey, apiKeyHash, err := auth.GenerateAPIKeyHash()
	if err != nil {
		h.logger.Error("Failed to generate API key", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "API_KEY_GENERATION_FAILED", Message: "Failed to generate API key"})
		return
	}

	// Create tenant
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               req.Name,
		APIKeyHash:         apiKeyHash,
		APIKeyLookupHash:   auth.LookupHash(apiKey),
		SubscriptionTier:   req.SubscriptionTier,
		RateLimitPerMinute: rateLimitPerMinute,
		MonthlyQuota:       monthlyQuota,
	}

	if err := h.tenantRepo.Create(c.Request.Context(), tenant); err != nil {
		h.logger.Error("Failed to create tenant", map[string]interface{}{
			"error":       err.Error(),
			"tenant_name": req.Name,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "TENANT_CREATION_FAILED", Message: "Failed to create tenant"})
		return
	}

	h.logger.Info("Tenant created successfully", map[string]interface{}{
		"tenant_id":   tenant.ID.String(),
		"tenant_name": tenant.Name,
		"tier":        tenant.SubscriptionTier,
	})

	c.JSON(http.StatusCreated, CreateTenantResponse{
		Tenant: tenant,
		APIKey: apiKey,
	})
}

// GetTenant retrieves current tenant information
// @Summary Get tenant info
// @Description Get information about the currently authenticated tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Tenant information"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /tenant [get]
func (h *TenantHandler) GetTenant(c *gin.Context) {
	tenant, exists := auth.GetTenantFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "MISSING_TENANT_CONTEXT", Message: "Tenant context not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant": tenant,
	})
}

// UpdateTenant updates tenant information
// @Summary Update tenant
// @Description Update information for the currently authenticated tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param request body UpdateTenantRequest true "Tenant update request"
// @Success 200 {object} map[string]interface{} "Tenant updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format or validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /tenant [put]
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	tenant, exists := auth.GetTenantFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "MISSING_TENANT_CONTEXT", Message: "Tenant context not found"})
		return
	}

	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: "Invalid request format", Details: err.Error()})
		return
	}

	// Update fields if provided — only Name and MonthlyQuota are self-service.
	// SubscriptionTier and RateLimitPerMinute are admin-only fields.
	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.MonthlyQuota > 0 {
		tenant.MonthlyQuota = req.MonthlyQuota
	}

	if err := h.tenantRepo.Update(c.Request.Context(), tenant); err != nil {
		h.logger.Error("Failed to update tenant", map[string]interface{}{
			"error":     err.Error(),
			"tenant_id": tenant.ID.String(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "TENANT_UPDATE_FAILED", Message: "Failed to update tenant"})
		return
	}

	h.logger.Info("Tenant updated successfully", map[string]interface{}{
		"tenant_id": tenant.ID.String(),
	})

	c.JSON(http.StatusOK, gin.H{
		"tenant": tenant,
	})
}

// RegenerateAPIKey generates a new API key for the tenant
// @Summary Regenerate API key
// @Description Generate a new API key for the currently authenticated tenant (invalidates the old key)
// @Tags tenants
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "New API key generated successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security ApiKeyAuth
// @Router /tenant/regenerate-key [post]
func (h *TenantHandler) RegenerateAPIKey(c *gin.Context) {
	tenant, exists := auth.GetTenantFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "MISSING_TENANT_CONTEXT", Message: "Tenant context not found"})
		return
	}

	// Generate new API key and hash
	apiKey, apiKeyHash, err := auth.GenerateAPIKeyHash()
	if err != nil {
		h.logger.Error("Failed to generate new API key", map[string]interface{}{
			"error":     err.Error(),
			"tenant_id": tenant.ID.String(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "API_KEY_GENERATION_FAILED", Message: "Failed to generate new API key"})
		return
	}

	// Update tenant with new API key hash
	tenant.APIKeyHash = apiKeyHash
	tenant.APIKeyLookupHash = auth.LookupHash(apiKey)
	if err := h.tenantRepo.Update(c.Request.Context(), tenant); err != nil {
		h.logger.Error("Failed to update tenant with new API key", map[string]interface{}{
			"error":     err.Error(),
			"tenant_id": tenant.ID.String(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "API_KEY_UPDATE_FAILED", Message: "Failed to update API key"})
		return
	}

	h.logger.Info("API key regenerated successfully", map[string]interface{}{
		"tenant_id": tenant.ID.String(),
	})

	c.JSON(http.StatusOK, gin.H{
		"api_key": apiKey,
		"message": "API key regenerated successfully",
	})
}

// ListTenants lists all tenants (admin endpoint)
func (h *TenantHandler) ListTenants(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	tenants, err := h.tenantRepo.List(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("Failed to list tenants", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "TENANT_LIST_FAILED", Message: "Failed to retrieve tenants"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
		"limit":   limit,
		"offset":  offset,
	})
}

// AdminUpdateTenant updates any tenant field including subscription tier (admin only)
func (h *TenantHandler) AdminUpdateTenant(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "MISSING_TENANT_ID", Message: "Tenant ID is required"})
		return
	}

	tid, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_TENANT_ID", Message: "Invalid tenant ID format"})
		return
	}

	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tid)
	if err != nil || tenant == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Code: "TENANT_NOT_FOUND", Message: "Tenant not found"})
		return
	}

	var req AdminUpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_REQUEST", Message: "Invalid request format", Details: err.Error()})
		return
	}

	if req.SubscriptionTier != "" {
		if !validSubscriptionTiers[req.SubscriptionTier] {
			c.JSON(http.StatusBadRequest, ErrorResponse{Code: "INVALID_SUBSCRIPTION_TIER", Message: "Subscription tier must be one of: free, basic, premium, enterprise"})
			return
		}
		tenant.SubscriptionTier = req.SubscriptionTier
		if req.RateLimitPerMinute == 0 {
			tenant.RateLimitPerMinute = getDefaultRateLimit(req.SubscriptionTier)
		}
		if req.MonthlyQuota == 0 {
			tenant.MonthlyQuota = getDefaultMonthlyQuota(req.SubscriptionTier)
		}
	}
	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.RateLimitPerMinute > 0 {
		tenant.RateLimitPerMinute = req.RateLimitPerMinute
	}
	if req.MonthlyQuota > 0 {
		tenant.MonthlyQuota = req.MonthlyQuota
	}

	if err := h.tenantRepo.Update(c.Request.Context(), tenant); err != nil {
		h.logger.Error("Failed to admin-update tenant", map[string]interface{}{
			"error":     err.Error(),
			"tenant_id": tenant.ID.String(),
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "TENANT_UPDATE_FAILED", Message: "Failed to update tenant"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant": tenant,
	})
}

// Helper functions for default values based on subscription tier
func getDefaultRateLimit(tier string) int {
	switch tier {
	case "free":
		return 100
	case "basic":
		return 1000
	case "premium":
		return 5000
	case "enterprise":
		return 10000
	default:
		return 100
	}
}

func getDefaultMonthlyQuota(tier string) int {
	switch tier {
	case "free":
		return 10000
	case "basic":
		return 100000
	case "premium":
		return 1000000
	case "enterprise":
		return 10000000
	default:
		return 10000
	}
}
