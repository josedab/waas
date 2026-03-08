package livemigration

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for live migration management
type Handler struct {
	service      *Service
	svixCompat   *SvixCompatLayer
	convoyCompat *ConvoyCompatLayer
}

// NewHandler creates a new live migration handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service:      service,
		svixCompat:   NewSvixCompatLayer(service),
		convoyCompat: NewConvoyCompatLayer(service),
	}
}

// RegisterRoutes registers live migration routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	migrations := router.Group("/migrations")
	{
		migrations.POST("", h.CreateMigration)
		migrations.GET("", h.ListMigrations)
		migrations.GET("/:id", h.GetMigration)
		migrations.POST("/:id/discover", h.DiscoverEndpoints)
		migrations.POST("/:id/import", h.ImportEndpoints)
		migrations.POST("/:id/validate", h.ValidateEndpoints)
		migrations.POST("/:id/parallel", h.StartParallelDelivery)
		migrations.GET("/:id/cutover-plan", h.GetCutoverPlan)
		migrations.POST("/:id/cutover", h.ExecuteCutover)
		migrations.POST("/:id/rollback", h.RollbackMigration)
		migrations.GET("/:id/stats", h.GetMigrationStats)
		migrations.POST("/:id/dry-run", h.DryRunMigration)
		migrations.POST("/:id/import/svix", h.ImportFromSvix)
		migrations.POST("/:id/import/convoy", h.ImportFromConvoy)
		migrations.POST("/:id/import/csv", h.ImportFromCSV)
		migrations.GET("/:id/checkpoint", h.GetCheckpoint)
	}

	compat := router.Group("/livemigration/compat")
	{
		compat.POST("/svix/endpoints", h.SvixCreateEndpoint)
		compat.GET("/svix/endpoints", h.SvixListEndpoints)
		compat.POST("/convoy/endpoints", h.ConvoyCreateEndpoint)
		compat.GET("/convoy/endpoints", h.ConvoyListEndpoints)
	}
}

// @Summary Create a migration job
// @Tags LiveMigration
// @Accept json
// @Produce json
// @Param body body CreateMigrationRequest true "Migration job configuration"
// @Success 201 {object} MigrationJob
// @Router /migrations [post]
func (h *Handler) CreateMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	job, err := h.service.CreateMigration(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "CREATE_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, job)
}

// @Summary List migration jobs
// @Tags LiveMigration
// @Produce json
// @Success 200 {object} map[string][]MigrationJob
// @Router /migrations [get]
func (h *Handler) ListMigrations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	jobs, err := h.service.ListMigrations(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalError(c, "LIST_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"migrations": jobs})
}

// @Summary Get a migration job
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} MigrationJob
// @Router /migrations/{id} [get]
func (h *Handler) GetMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	job, err := h.service.GetMigration(c.Request.Context(), tenantID, jobID)
	if err != nil {
		httputil.InternalError(c, "NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Discover endpoints from source platform
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} map[string][]MigrationEndpoint
// @Router /migrations/{id}/discover [post]
func (h *Handler) DiscoverEndpoints(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	endpoints, err := h.service.DiscoverEndpoints(c.Request.Context(), tenantID, jobID)
	if err != nil {
		httputil.InternalError(c, "DISCOVER_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"endpoints": endpoints, "count": len(endpoints)})
}

// @Summary Import discovered endpoints
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} map[string][]MigrationEndpoint
// @Router /migrations/{id}/import [post]
func (h *Handler) ImportEndpoints(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	endpoints, err := h.service.ImportEndpoints(c.Request.Context(), tenantID, jobID)
	if err != nil {
		httputil.InternalError(c, "IMPORT_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"endpoints": endpoints, "count": len(endpoints)})
}

// @Summary Validate imported endpoints
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} map[string][]MigrationEndpoint
// @Router /migrations/{id}/validate [post]
func (h *Handler) ValidateEndpoints(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	endpoints, err := h.service.ValidateEndpoints(c.Request.Context(), tenantID, jobID)
	if err != nil {
		httputil.InternalError(c, "VALIDATE_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"endpoints": endpoints, "count": len(endpoints)})
}

// @Summary Start parallel delivery comparison
// @Tags LiveMigration
// @Accept json
// @Produce json
// @Param id path string true "Migration Job ID"
// @Param body body StartParallelRequest true "Parallel delivery configuration"
// @Success 200 {object} map[string][]ParallelDeliveryResult
// @Router /migrations/{id}/parallel [post]
func (h *Handler) StartParallelDelivery(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	var req StartParallelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	req.JobID = jobID

	results, err := h.service.StartParallelDelivery(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalError(c, "PARALLEL_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

// @Summary Get cutover plan
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} CutoverPlan
// @Router /migrations/{id}/cutover-plan [get]
func (h *Handler) GetCutoverPlan(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	plan, err := h.service.GetCutoverPlan(c.Request.Context(), tenantID, jobID)
	if err != nil {
		httputil.InternalError(c, "CUTOVER_PLAN_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, plan)
}

// @Summary Execute cutover to destination platform
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} MigrationJob
// @Router /migrations/{id}/cutover [post]
func (h *Handler) ExecuteCutover(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	job, err := h.service.ExecuteCutover(c.Request.Context(), tenantID, jobID)
	if err != nil {
		httputil.InternalError(c, "CUTOVER_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Rollback migration to source platform
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} MigrationJob
// @Router /migrations/{id}/rollback [post]
func (h *Handler) RollbackMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	job, err := h.service.RollbackMigration(c.Request.Context(), tenantID, jobID)
	if err != nil {
		httputil.InternalError(c, "ROLLBACK_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, job)
}

// @Summary Get migration statistics
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} MigrationStats
// @Router /migrations/{id}/stats [get]
func (h *Handler) GetMigrationStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	jobID := c.Param("id")

	stats, err := h.service.GetMigrationStats(c.Request.Context(), tenantID, jobID)
	if err != nil {
		httputil.InternalError(c, "STATS_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary Dry-run migration analysis
// @Tags LiveMigration
// @Accept json
// @Produce json
// @Param body body ImporterConfig true "Importer configuration"
// @Success 200 {object} DryRunResult
// @Router /migrations/{id}/dry-run [post]
func (h *Handler) DryRunMigration(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var config ImporterConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	result, err := h.service.DryRunMigration(c.Request.Context(), tenantID, &config)
	if err != nil {
		httputil.InternalError(c, "DRY_RUN_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Import endpoints from Svix
// @Tags LiveMigration
// @Accept json
// @Produce json
// @Param body body ImporterConfig true "Svix importer configuration"
// @Success 201 {object} MigrationJob
// @Router /migrations/{id}/import/svix [post]
func (h *Handler) ImportFromSvix(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var config ImporterConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	job, err := h.service.ImportFromSvix(c.Request.Context(), tenantID, &config)
	if err != nil {
		httputil.InternalError(c, "SVIX_IMPORT_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, job)
}

// @Summary Import endpoints from Convoy
// @Tags LiveMigration
// @Accept json
// @Produce json
// @Param body body ImporterConfig true "Convoy importer configuration"
// @Success 201 {object} MigrationJob
// @Router /migrations/{id}/import/convoy [post]
func (h *Handler) ImportFromConvoy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var config ImporterConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	job, err := h.service.ImportFromConvoy(c.Request.Context(), tenantID, &config)
	if err != nil {
		httputil.InternalError(c, "CONVOY_IMPORT_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, job)
}

// @Summary Import endpoints from CSV/JSON file
// @Tags LiveMigration
// @Accept json
// @Produce json
// @Param body body ImporterConfig true "CSV/JSON importer configuration"
// @Success 201 {object} MigrationJob
// @Router /migrations/{id}/import/csv [post]
func (h *Handler) ImportFromCSV(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var config ImporterConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	job, err := h.service.ImportFromCSV(c.Request.Context(), tenantID, &config)
	if err != nil {
		httputil.InternalError(c, "CSV_IMPORT_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, job)
}

// @Summary Get migration checkpoint
// @Tags LiveMigration
// @Produce json
// @Param id path string true "Migration Job ID"
// @Success 200 {object} MigrationCheckpoint
// @Router /migrations/{id}/checkpoint [get]
func (h *Handler) GetCheckpoint(c *gin.Context) {
	jobID := c.Param("id")

	checkpoint, err := h.service.GetCheckpoint(c.Request.Context(), jobID)
	if err != nil {
		httputil.InternalError(c, "CHECKPOINT_NOT_FOUND", err)
		return
	}

	c.JSON(http.StatusOK, checkpoint)
}

// @Summary Create endpoint via Svix-compatible API
// @Tags LiveMigration Compat
// @Accept json
// @Produce json
// @Param body body SvixEndpointIn true "Svix endpoint"
// @Success 201 {object} SvixEndpointOut
// @Router /livemigration/compat/svix/endpoints [post]
func (h *Handler) SvixCreateEndpoint(c *gin.Context) {
	appUID := c.Query("app_uid")

	var endpoint SvixEndpointIn
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	out, err := h.svixCompat.CreateEndpoint(appUID, endpoint)
	if err != nil {
		httputil.InternalError(c, "SVIX_COMPAT_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, out)
}

// @Summary List endpoints via Svix-compatible API
// @Tags LiveMigration Compat
// @Produce json
// @Success 200 {array} SvixEndpointOut
// @Router /livemigration/compat/svix/endpoints [get]
func (h *Handler) SvixListEndpoints(c *gin.Context) {
	appUID := c.Query("app_uid")

	endpoints, err := h.svixCompat.ListEndpoints(appUID)
	if err != nil {
		httputil.InternalError(c, "SVIX_COMPAT_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": endpoints})
}

// @Summary Create endpoint via Convoy-compatible API
// @Tags LiveMigration Compat
// @Accept json
// @Produce json
// @Param body body ConvoyEndpointIn true "Convoy endpoint"
// @Success 201 {object} ConvoyEndpointOut
// @Router /livemigration/compat/convoy/endpoints [post]
func (h *Handler) ConvoyCreateEndpoint(c *gin.Context) {
	projectID := c.Query("project_id")

	var endpoint ConvoyEndpointIn
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	out, err := h.convoyCompat.CreateEndpoint(projectID, endpoint)
	if err != nil {
		httputil.InternalError(c, "CONVOY_COMPAT_FAILED", err)
		return
	}

	c.JSON(http.StatusCreated, out)
}

// @Summary List endpoints via Convoy-compatible API
// @Tags LiveMigration Compat
// @Produce json
// @Success 200 {array} ConvoyEndpointOut
// @Router /livemigration/compat/convoy/endpoints [get]
func (h *Handler) ConvoyListEndpoints(c *gin.Context) {
	projectID := c.Query("project_id")

	endpoints, err := h.convoyCompat.ListEndpoints(projectID)
	if err != nil {
		httputil.InternalError(c, "CONVOY_COMPAT_FAILED", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": endpoints})
}
