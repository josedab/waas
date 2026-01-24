package whitelabel

import (
	"net/http"

	"github.com/gin-gonic/gin"
	pkgerrors "webhook-platform/pkg/errors"
)

// Handler handles whitelabel HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new whitelabel handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers whitelabel routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	wl := r.Group("/whitelabel")
	{
		// Config
		wl.POST("/config", h.CreateConfig)
		wl.GET("/config", h.GetConfig)
		wl.PUT("/config", h.UpdateConfig)
		wl.DELETE("/config", h.DeleteConfig)

		// Domain
		wl.POST("/domain/setup", h.SetupDomain)
		wl.POST("/domain/verify", h.VerifyDomain)
		wl.GET("/domain/dns", h.GetDNSRecords)

		// Sub-tenants
		wl.POST("/sub-tenants", h.CreateSubTenant)
		wl.GET("/sub-tenants", h.ListSubTenants)
		wl.GET("/sub-tenants/:id", h.GetSubTenant)
		wl.POST("/sub-tenants/:id/suspend", h.SuspendSubTenant)
		wl.POST("/sub-tenants/:id/reactivate", h.ReactivateSubTenant)

		// Partners
		wl.POST("/partners", h.RegisterPartner)
		wl.GET("/partners", h.ListPartners)
		wl.GET("/partners/:id", h.GetPartner)

		// Partner revenue
		wl.GET("/partners/:id/revenue", h.GetPartnerRevenue)

		// Branding
		wl.PUT("/branding", h.UpdateBranding)
		wl.POST("/branding/preview", h.GeneratePreview)

		// Analytics
		wl.GET("/analytics", h.GetAnalytics)
	}
}

// CreateConfig godoc
// @Summary Create whitelabel config
// @Description Create a new whitelabel configuration for the tenant
// @Tags Whitelabel
// @Accept json
// @Produce json
// @Param request body CreateWhitelabelRequest true "Whitelabel config"
// @Success 201 {object} WhitelabelConfig
// @Router /whitelabel/config [post]
func (h *Handler) CreateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateWhitelabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	config, err := h.service.CreateWhitelabelConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetConfig godoc
// @Summary Get whitelabel config
// @Description Get the whitelabel configuration for the tenant
// @Tags Whitelabel
// @Produce json
// @Success 200 {object} WhitelabelConfig
// @Router /whitelabel/config [get]
func (h *Handler) GetConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, err := h.service.GetConfig(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "config")
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfig godoc
// @Summary Update whitelabel config
// @Description Update the whitelabel configuration for the tenant
// @Tags Whitelabel
// @Accept json
// @Produce json
// @Param request body CreateWhitelabelRequest true "Whitelabel config"
// @Success 200 {object} WhitelabelConfig
// @Router /whitelabel/config [put]
func (h *Handler) UpdateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateWhitelabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	config, err := h.service.UpdateConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeleteConfig godoc
// @Summary Delete whitelabel config
// @Description Delete the whitelabel configuration for the tenant
// @Tags Whitelabel
// @Success 204
// @Router /whitelabel/config [delete]
func (h *Handler) DeleteConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	if err := h.service.DeleteConfig(c.Request.Context(), tenantID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

// SetupDomain godoc
// @Summary Setup custom domain
// @Description Setup DNS verification records for a custom domain
// @Tags Whitelabel
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /whitelabel/domain/setup [post]
func (h *Handler) SetupDomain(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, records, err := h.service.SetupCustomDomain(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"config":      config,
		"dns_records": records,
	})
}

// VerifyDomain godoc
// @Summary Verify domain
// @Description Verify DNS records for the custom domain
// @Tags Whitelabel
// @Produce json
// @Success 200 {object} WhitelabelConfig
// @Router /whitelabel/domain/verify [post]
func (h *Handler) VerifyDomain(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	config, err := h.service.VerifyDomain(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetDNSRecords godoc
// @Summary Get DNS records
// @Description Get DNS verification records for the custom domain
// @Tags Whitelabel
// @Produce json
// @Success 200 {array} DNSVerification
// @Router /whitelabel/domain/dns [get]
func (h *Handler) GetDNSRecords(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	records, err := h.service.GetDNSRecords(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"dns_records": records})
}

// CreateSubTenant godoc
// @Summary Create sub-tenant
// @Description Create a new sub-tenant
// @Tags Whitelabel
// @Accept json
// @Produce json
// @Param request body CreateSubTenantRequest true "Sub-tenant details"
// @Success 201 {object} SubTenant
// @Router /whitelabel/sub-tenants [post]
func (h *Handler) CreateSubTenant(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateSubTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	subTenant, err := h.service.CreateSubTenant(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, subTenant)
}

// ListSubTenants godoc
// @Summary List sub-tenants
// @Description List all sub-tenants for the parent tenant
// @Tags Whitelabel
// @Produce json
// @Success 200 {array} SubTenant
// @Router /whitelabel/sub-tenants [get]
func (h *Handler) ListSubTenants(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	subTenants, err := h.service.ListSubTenants(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"sub_tenants": subTenants})
}

// GetSubTenant godoc
// @Summary Get sub-tenant
// @Description Get a sub-tenant by ID
// @Tags Whitelabel
// @Produce json
// @Param id path string true "Sub-tenant ID"
// @Success 200 {object} SubTenant
// @Router /whitelabel/sub-tenants/{id} [get]
func (h *Handler) GetSubTenant(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	subTenantID := c.Param("id")

	subTenant, err := h.service.GetSubTenant(c.Request.Context(), tenantID, subTenantID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "sub-tenant")
		return
	}

	c.JSON(http.StatusOK, subTenant)
}

// SuspendSubTenant godoc
// @Summary Suspend sub-tenant
// @Description Suspend a sub-tenant
// @Tags Whitelabel
// @Param id path string true "Sub-tenant ID"
// @Success 200 {object} map[string]interface{}
// @Router /whitelabel/sub-tenants/{id}/suspend [post]
func (h *Handler) SuspendSubTenant(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	subTenantID := c.Param("id")

	if err := h.service.SuspendSubTenant(c.Request.Context(), tenantID, subTenantID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "suspended"})
}

// ReactivateSubTenant godoc
// @Summary Reactivate sub-tenant
// @Description Reactivate a suspended sub-tenant
// @Tags Whitelabel
// @Param id path string true "Sub-tenant ID"
// @Success 200 {object} map[string]interface{}
// @Router /whitelabel/sub-tenants/{id}/reactivate [post]
func (h *Handler) ReactivateSubTenant(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	subTenantID := c.Param("id")

	if err := h.service.ReactivateSubTenant(c.Request.Context(), tenantID, subTenantID); err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "reactivated"})
}

// RegisterPartner godoc
// @Summary Register partner
// @Description Register a new partner
// @Tags Whitelabel
// @Accept json
// @Produce json
// @Param request body CreatePartnerRequest true "Partner details"
// @Success 201 {object} Partner
// @Router /whitelabel/partners [post]
func (h *Handler) RegisterPartner(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreatePartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	partner, err := h.service.RegisterPartner(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusCreated, partner)
}

// ListPartners godoc
// @Summary List partners
// @Description List all partners for the tenant
// @Tags Whitelabel
// @Produce json
// @Success 200 {array} Partner
// @Router /whitelabel/partners [get]
func (h *Handler) ListPartners(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	partners, err := h.service.ListPartners(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"partners": partners})
}

// GetPartner godoc
// @Summary Get partner
// @Description Get a partner by ID
// @Tags Whitelabel
// @Produce json
// @Param id path string true "Partner ID"
// @Success 200 {object} Partner
// @Router /whitelabel/partners/{id} [get]
func (h *Handler) GetPartner(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	partnerID := c.Param("id")

	partner, err := h.service.GetPartner(c.Request.Context(), tenantID, partnerID)
	if err != nil {
		pkgerrors.AbortWithNotFound(c, "partner")
		return
	}

	c.JSON(http.StatusOK, partner)
}

// GetPartnerRevenue godoc
// @Summary Get partner revenue
// @Description Get revenue records for a partner
// @Tags Whitelabel
// @Produce json
// @Param id path string true "Partner ID"
// @Success 200 {array} PartnerRevenue
// @Router /whitelabel/partners/{id}/revenue [get]
func (h *Handler) GetPartnerRevenue(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	partnerID := c.Param("id")

	revenues, err := h.service.GetPartnerRevenue(c.Request.Context(), tenantID, partnerID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"revenue": revenues})
}

// UpdateBranding godoc
// @Summary Update branding
// @Description Update branding for the whitelabel config
// @Tags Whitelabel
// @Accept json
// @Produce json
// @Param request body UpdateBrandingRequest true "Branding updates"
// @Success 200 {object} WhitelabelConfig
// @Router /whitelabel/branding [put]
func (h *Handler) UpdateBranding(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req UpdateBrandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkgerrors.HandleBindError(c, err)
		return
	}

	config, err := h.service.UpdateBranding(c.Request.Context(), tenantID, &req)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, config)
}

// GeneratePreview godoc
// @Summary Generate branding preview
// @Description Generate a preview URL for the current branding
// @Tags Whitelabel
// @Produce json
// @Success 200 {object} BrandingPreview
// @Router /whitelabel/branding/preview [post]
func (h *Handler) GeneratePreview(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	preview, err := h.service.GeneratePreview(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, preview)
}

// GetAnalytics godoc
// @Summary Get whitelabel analytics
// @Description Get analytics for the whitelabel configuration
// @Tags Whitelabel
// @Produce json
// @Success 200 {object} WhitelabelAnalytics
// @Router /whitelabel/analytics [get]
func (h *Handler) GetAnalytics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	analytics, err := h.service.GetAnalytics(c.Request.Context(), tenantID)
	if err != nil {
		pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
		return
	}

	c.JSON(http.StatusOK, analytics)
}