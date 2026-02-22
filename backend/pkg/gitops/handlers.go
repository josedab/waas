package gitops

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for GitOps configuration management
type Handler struct {
	service *Service
}

// NewHandler creates a new GitOps handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers GitOps routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	gitops := router.Group("/gitops")
	{
		// Manifests
		gitops.POST("/manifests/validate", h.ValidateManifest)
		gitops.POST("/manifests", h.UploadManifest)
		gitops.GET("/manifests", h.ListManifests)
		gitops.GET("/manifests/:id", h.GetManifest)
		gitops.POST("/manifests/:id/plan", h.PlanApply)
		gitops.POST("/manifests/:id/apply", h.ApplyManifest)
		gitops.POST("/manifests/:id/rollback", h.RollbackManifest)

		// Drift
		gitops.POST("/drift/detect", h.DetectDrift)
		gitops.GET("/drift", h.ListDriftReports)
		gitops.GET("/drift/:id", h.GetDriftReport)

		// Declarative Config (GitOps-driven)
		gitops.POST("/declarative/validate", h.ValidateDeclarativeConfig)
		gitops.POST("/declarative/apply", h.ApplyDeclarativeConfig)
		gitops.GET("/declarative/:manifest_id/sync-state", h.GetSyncState)
	}
}

// @Summary Validate a configuration manifest
// @Tags GitOps
// @Accept json
// @Produce json
// @Param body body ValidateManifestRequest true "Manifest content to validate"
// @Success 200 {object} map[string]interface{}
// @Router /gitops/manifests/validate [post]
func (h *Handler) ValidateManifest(c *gin.Context) {
	var req ValidateManifestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	errors, err := h.service.ValidateManifest(c.Request.Context(), req.Content)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"valid": false, "errors": errors})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "errors": []string{}})
}

// @Summary Upload a configuration manifest
// @Tags GitOps
// @Accept json
// @Produce json
// @Param body body ValidateManifestRequest true "Manifest content"
// @Success 201 {object} ConfigManifest
// @Router /gitops/manifests [post]
func (h *Handler) UploadManifest(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req struct {
		Name    string `json:"name" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	manifest, err := h.service.UploadManifest(c.Request.Context(), tenantID, req.Name, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "UPLOAD_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, manifest)
}

// @Summary List configuration manifests
// @Tags GitOps
// @Produce json
// @Success 200 {object} map[string][]ConfigManifest
// @Router /gitops/manifests [get]
func (h *Handler) ListManifests(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	manifests, err := h.service.ListManifests(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"manifests": manifests})
}

// @Summary Get a configuration manifest
// @Tags GitOps
// @Produce json
// @Param id path string true "Manifest ID"
// @Success 200 {object} ConfigManifest
// @Router /gitops/manifests/{id} [get]
func (h *Handler) GetManifest(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	manifestID := c.Param("id")

	manifest, err := h.service.GetManifest(c.Request.Context(), tenantID, manifestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, manifest)
}

// @Summary Plan manifest application
// @Tags GitOps
// @Produce json
// @Param id path string true "Manifest ID"
// @Success 200 {object} ApplyPlan
// @Router /gitops/manifests/{id}/plan [post]
func (h *Handler) PlanApply(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	manifestID := c.Param("id")

	plan, err := h.service.PlanApply(c.Request.Context(), tenantID, manifestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "PLAN_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, plan)
}

// @Summary Apply a configuration manifest
// @Tags GitOps
// @Accept json
// @Produce json
// @Param id path string true "Manifest ID"
// @Param body body ApplyManifestRequest true "Apply options"
// @Success 200 {object} ApplyResult
// @Router /gitops/manifests/{id}/apply [post]
func (h *Handler) ApplyManifest(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	manifestID := c.Param("id")

	var req struct {
		DryRun bool `json:"dry_run"`
		Force  bool `json:"force"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body, default to no dry_run and no force
		req.DryRun = false
		req.Force = false
	}

	if req.DryRun {
		plan, err := h.service.PlanApply(c.Request.Context(), tenantID, manifestID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "PLAN_FAILED", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"dry_run": true, "plan": plan})
		return
	}

	result, err := h.service.ApplyManifest(c.Request.Context(), tenantID, manifestID, req.Force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "APPLY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Rollback a configuration manifest
// @Tags GitOps
// @Produce json
// @Param id path string true "Manifest ID"
// @Success 200 {object} ApplyResult
// @Router /gitops/manifests/{id}/rollback [post]
func (h *Handler) RollbackManifest(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	manifestID := c.Param("id")

	result, err := h.service.RollbackManifest(c.Request.Context(), tenantID, manifestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "ROLLBACK_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Detect configuration drift
// @Tags GitOps
// @Produce json
// @Success 200 {object} DriftReport
// @Router /gitops/drift/detect [post]
func (h *Handler) DetectDrift(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	report, err := h.service.DetectDrift(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DRIFT_DETECTION_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, report)
}

// @Summary List drift reports
// @Tags GitOps
// @Produce json
// @Success 200 {object} map[string][]DriftReport
// @Router /gitops/drift [get]
func (h *Handler) ListDriftReports(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	reports, err := h.service.ListDriftReports(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"drift_reports": reports})
}

// @Summary Get a drift report
// @Tags GitOps
// @Produce json
// @Param id path string true "Drift Report ID"
// @Success 200 {object} DriftReport
// @Router /gitops/drift/{id} [get]
func (h *Handler) GetDriftReport(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	reportID := c.Param("id")

	report, err := h.service.GetDriftReport(c.Request.Context(), tenantID, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ValidateDeclarativeConfig validates a declarative YAML config.
func (h *Handler) ValidateDeclarativeConfig(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	errors, err := h.service.ValidateDeclarativeConfig(c.Request.Context(), req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "errors": errors})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// ApplyDeclarativeConfig applies a declarative YAML configuration.
func (h *Handler) ApplyDeclarativeConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		Content string `json:"content" binding:"required"`
		DryRun  bool   `json:"dry_run"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	result, err := h.service.ApplyDeclarativeConfig(c.Request.Context(), tenantID, req.Content, req.DryRun)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "APPLY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSyncState returns the sync state for a manifest.
func (h *Handler) GetSyncState(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	manifestID := c.Param("manifest_id")

	state, err := h.service.GetSyncState(c.Request.Context(), tenantID, manifestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SYNC_STATE_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, state)
}
