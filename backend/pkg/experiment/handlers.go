package experiment

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for experiments.
type Handler struct {
	service *Service
}

// NewHandler creates a new experiment handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers experiment routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/experiments")
	{
		g.POST("", h.CreateExperiment)
		g.GET("", h.ListExperiments)
		g.GET("/:id", h.GetExperiment)
		g.POST("/:id/start", h.StartExperiment)
		g.POST("/:id/stop", h.StopExperiment)
		g.GET("/:id/results", h.GetResults)
		g.DELETE("/:id", h.DeleteExperiment)
	}
}

func (h *Handler) CreateExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exp, err := h.service.CreateExperiment(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, exp)
}

func (h *Handler) GetExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	exp, err := h.service.GetExperiment(c.Request.Context(), tenantID, expID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, exp)
}

func (h *Handler) ListExperiments(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	experiments, err := h.service.ListExperiments(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"experiments": experiments})
}

func (h *Handler) StartExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	exp, err := h.service.StartExperiment(c.Request.Context(), tenantID, expID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, exp)
}

func (h *Handler) StopExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	exp, err := h.service.StopExperiment(c.Request.Context(), tenantID, expID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, exp)
}

func (h *Handler) GetResults(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	results, err := h.service.GetResults(c.Request.Context(), tenantID, expID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

func (h *Handler) DeleteExperiment(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	expID := c.Param("id")

	if err := h.service.DeleteExperiment(c.Request.Context(), tenantID, expID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
