package livemigration

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP handlers for live migration management
type Handler struct {
	service *Service
}

// NewHandler creates a new live migration handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "LIST_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DISCOVER_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "IMPORT_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "VALIDATE_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "PARALLEL_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CUTOVER_PLAN_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CUTOVER_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "ROLLBACK_FAILED", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "STATS_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, stats)
}
