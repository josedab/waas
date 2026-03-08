package pluginmarket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/httputil"
)

// RegisterMarketplaceRoutes registers marketplace, SDK, and monetization routes
func (h *Handler) RegisterMarketplaceRoutes(r *gin.RouterGroup) {
	mp := r.Group("/marketplace")
	{
		// Marketplace browsing
		mp.GET("/listings", h.BrowseListings)
		mp.GET("/listings/:id", h.GetListing)
		mp.GET("/categories", h.GetCategories)
		mp.POST("/listings/:id/install", h.InstallFromMarketplace)
		mp.DELETE("/listings/:id/uninstall", h.UninstallFromMarketplace)
		mp.GET("/installed", h.GetInstalledPlugins)
		mp.POST("/listings/:id/review", h.SubmitMarketplaceReview)

		// SDK v2 sandbox
		mp.POST("/sandbox/execute", h.ExecuteSandbox)
		mp.POST("/sandbox/test", h.TestPlugin)

		// Monetization
		mp.POST("/developers/register", h.RegisterDeveloper)
		mp.GET("/developers/:id/earnings", h.GetDeveloperEarnings)
		mp.POST("/developers/:id/payout", h.ProcessPayout)
		mp.POST("/purchase", h.RecordPurchase)
	}
}

func (h *Handler) BrowseListings(c *gin.Context) {
	svc := NewMarketplaceService()
	category := c.Query("category")
	sortBy := c.DefaultQuery("sort", "downloads")

	params := &MarketplaceSearchParams{
		Category: category,
		SortBy:   sortBy,
		Limit:    50,
	}

	listings, total := svc.Browse(c.Request.Context(), params)
	c.JSON(http.StatusOK, gin.H{"listings": listings, "total": total})
}

func (h *Handler) GetListing(c *gin.Context) {
	svc := NewMarketplaceService()
	id := c.Param("id")

	listing, err := svc.GetListing(c.Request.Context(), id)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, listing)
}

func (h *Handler) GetCategories(c *gin.Context) {
	categories := []string{
		"transform", "filter", "routing", "security",
		"monitoring", "integration", "analytics", "utility",
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (h *Handler) InstallFromMarketplace(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	listingID := c.Param("id")

	svc := NewMarketplaceService()
	installation, err := svc.Install(c.Request.Context(), tenantID, listingID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusCreated, installation)
}

func (h *Handler) UninstallFromMarketplace(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	listingID := c.Param("id")

	svc := NewMarketplaceService()
	if err := svc.Uninstall(c.Request.Context(), tenantID, listingID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetInstalledPlugins(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	svc := NewMarketplaceService()

	installed := svc.GetInstalled(c.Request.Context(), tenantID)
	c.JSON(http.StatusOK, gin.H{"installed": installed})
}

func (h *Handler) SubmitMarketplaceReview(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	listingID := c.Param("id")

	var req struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := NewMarketplaceService()
	review, err := svc.SubmitReview(c.Request.Context(), tenantID, listingID, req.Rating, req.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, review)
}

func (h *Handler) ExecuteSandbox(c *gin.Context) {
	var req struct {
		Code    string                 `json:"code" binding:"required"`
		Input   map[string]interface{} `json:"input"`
		Timeout int                    `json:"timeout_ms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 5000
	}

	sandbox := NewPluginSandbox(SandboxConfig{
		TimeoutMs:   timeout,
		MaxMemoryKB: 10 * 1024,
	})
	result, err := sandbox.Execute(c.Request.Context(), req.Code, req.Input)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result, "success": true})
}

func (h *Handler) TestPlugin(c *gin.Context) {
	var req struct {
		Manifest PluginManifest `json:"manifest" binding:"required"`
		Hook     string         `json:"hook" binding:"required"`
		Source   string         `json:"source" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	runner := NewPluginTestRunner(&req.Manifest, SandboxConfig{
		TimeoutMs:   5000,
		MaxMemoryKB: 10 * 1024,
	})
	event := &MockWebhookEvent{
		EventType: "test.event",
		Payload:   map[string]interface{}{"test": true},
	}
	result, err := runner.RunHook(c.Request.Context(), PluginHookTypeV2(req.Hook), req.Source, event)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result, "success": true})
}

func (h *Handler) RegisterDeveloper(c *gin.Context) {
	var req struct {
		DeveloperID     string `json:"developer_id" binding:"required"`
		Email           string `json:"email" binding:"required"`
		DisplayName     string `json:"display_name" binding:"required"`
		StripeConnectID string `json:"stripe_connect_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := NewMonetizationService()
	account, err := svc.RegisterDeveloper(nil, req.DeveloperID, req.Email, req.DisplayName, req.StripeConnectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, account)
}

func (h *Handler) GetDeveloperEarnings(c *gin.Context) {
	devID := c.Param("id")

	svc := NewMonetizationService()
	earnings, err := svc.GetEarnings(nil, devID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, earnings)
}

func (h *Handler) ProcessPayout(c *gin.Context) {
	devID := c.Param("id")

	svc := NewMonetizationService()
	payout, err := svc.ProcessPayout(nil, devID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, payout)
}

func (h *Handler) RecordPurchase(c *gin.Context) {
	var req struct {
		ListingID     string  `json:"listing_id" binding:"required"`
		BuyerTenantID string  `json:"buyer_tenant_id" binding:"required"`
		DeveloperID   string  `json:"developer_id" binding:"required"`
		Amount        float64 `json:"amount" binding:"required"`
		Currency      string  `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "usd"
	}

	svc := NewMonetizationService()
	tx, err := svc.RecordPurchase(nil, req.ListingID, req.BuyerTenantID, req.DeveloperID, req.Amount, currency)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusCreated, tx)
}
