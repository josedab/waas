package loadtest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the load testing suite.
type Handler struct {
	service *Service
}

// NewHandler creates a new load test handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all load test routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/loadtest")
	{
		group.POST("/runs", h.CreateTestRun)
		group.GET("/runs", h.ListTestRuns)
		group.GET("/runs/:id", h.GetTestRun)
		group.POST("/runs/:id/cancel", h.CancelTestRun)
		group.GET("/scenarios", h.GetScenarios)
	}
}

// CreateTestRun starts a new load test.
// @Summary Create load test run
// @Tags loadtest
// @Accept json
// @Produce json
// @Success 201 {object} TestRun
// @Router /loadtest/runs [post]
func (h *Handler) CreateTestRun(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var cfg TestConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	run, err := h.service.CreateTestRun(tenantID, &cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, run)
}

// ListTestRuns returns all test runs.
// @Summary List load test runs
// @Tags loadtest
// @Produce json
// @Success 200 {array} TestRun
// @Router /loadtest/runs [get]
func (h *Handler) ListTestRuns(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	runs, err := h.service.ListTestRuns(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, runs)
}

// GetTestRun retrieves a specific test run.
// @Summary Get load test run
// @Tags loadtest
// @Param id path string true "Test Run ID"
// @Produce json
// @Success 200 {object} TestRun
// @Router /loadtest/runs/{id} [get]
func (h *Handler) GetTestRun(c *gin.Context) {
	run, err := h.service.GetTestRun(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, run)
}

// CancelTestRun cancels a running test.
// @Summary Cancel load test run
// @Tags loadtest
// @Param id path string true "Test Run ID"
// @Produce json
// @Router /loadtest/runs/{id}/cancel [post]
func (h *Handler) CancelTestRun(c *gin.Context) {
	if err := h.service.CancelTestRun(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

// GetScenarios returns pre-built test scenarios.
// @Summary Get pre-built scenarios
// @Tags loadtest
// @Produce json
// @Success 200 {array} Scenario
// @Router /loadtest/scenarios [get]
func (h *Handler) GetScenarios(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetScenarios())
}
