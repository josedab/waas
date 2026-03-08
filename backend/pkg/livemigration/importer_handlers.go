package livemigration

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterImporterRoutes registers platform-specific importer and cutover routes
func (h *Handler) RegisterImporterRoutes(router *gin.RouterGroup) {
	mig := router.Group("/migration")
	{
		// Platform-specific importers
		mig.POST("/import/svix", h.ImportSvixV2)
		mig.POST("/import/convoy", h.ImportConvoyV2)
		mig.POST("/import/hookdeck", h.ImportHookdeckV2)
		mig.POST("/import/csv", h.ImportCSVV2)
		mig.POST("/import/json", h.ImportJSONV2)

		// Dual-write cutover
		mig.POST("/cutover/start", h.StartCutoverV2)
		mig.GET("/cutover/:job_id/status", h.GetCutoverStatusV2)
		mig.POST("/cutover/:job_id/adjust", h.AdjustTrafficV2)
		mig.POST("/cutover/:job_id/rollback", h.RollbackCutoverV2)
	}
}

func runImport(c *gin.Context, platform string) {
	var req struct {
		BaseURL      string            `json:"base_url"`
		APIKey       string            `json:"api_key"`
		Data         string            `json:"data"`
		FieldMapping map[string]string `json:"field_mapping"`
		DryRun       bool              `json:"dry_run"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	importer, err := NewImporter(platform)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := ImportConfig{
		SourceType:   platform,
		APIKey:       req.APIKey,
		BaseURL:      req.BaseURL,
		RawData:      req.Data,
		FieldMapping: req.FieldMapping,
		DryRun:       req.DryRun,
	}

	result, err := importer.Import(c.Request.Context(), &config)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ImportSvixV2(c *gin.Context)     { runImport(c, "svix") }
func (h *Handler) ImportConvoyV2(c *gin.Context)   { runImport(c, "convoy") }
func (h *Handler) ImportHookdeckV2(c *gin.Context) { runImport(c, "hookdeck") }
func (h *Handler) ImportCSVV2(c *gin.Context)      { runImport(c, "csv") }
func (h *Handler) ImportJSONV2(c *gin.Context)     { runImport(c, "json") }

func (h *Handler) StartCutoverV2(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req struct {
		JobID  string `json:"job_id" binding:"required"`
		DryRun bool   `json:"dry_run"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := NewCutoverService(h.service.repo)
	plan, err := svc.StartCutover(c.Request.Context(), tenantID, req.JobID, req.DryRun)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusCreated, plan)
}

func (h *Handler) GetCutoverStatusV2(c *gin.Context) {
	jobID := c.Param("job_id")
	svc := NewCutoverService(h.service.repo)
	status, err := svc.GetStatus(c.Request.Context(), jobID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) AdjustTrafficV2(c *gin.Context) {
	jobID := c.Param("job_id")
	var req struct {
		Percentage int `json:"percentage" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := NewCutoverService(h.service.repo)
	plan, err := svc.AdjustTrafficSplit(c.Request.Context(), jobID, req.Percentage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) RollbackCutoverV2(c *gin.Context) {
	jobID := c.Param("job_id")
	svc := NewCutoverService(h.service.repo)
	plan, err := svc.Rollback(c.Request.Context(), jobID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}
