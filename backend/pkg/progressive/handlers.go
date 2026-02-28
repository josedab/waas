package progressive

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for progressive delivery.
type Handler struct {
	service *Service
}

// NewHandler creates a new progressive delivery handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers progressive delivery routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/progressive")
	{
		g.POST("/rollouts", h.CreateRollout)
		g.GET("/rollouts", h.ListRollouts)
		g.GET("/rollouts/:id", h.GetRollout)
		g.POST("/rollouts/:id/start", h.StartRollout)
		g.POST("/rollouts/:id/pause", h.PauseRollout)
		g.POST("/rollouts/:id/resume", h.ResumeRollout)
		g.PUT("/rollouts/:id/traffic", h.UpdateTraffic)
		g.POST("/rollouts/:id/complete", h.CompleteRollout)
		g.POST("/rollouts/:id/rollback", h.RollbackRollout)
		g.GET("/rollouts/:id/evaluate", h.EvaluateRollout)
		g.DELETE("/rollouts/:id", h.DeleteRollout)
	}
}

func (h *Handler) CreateRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateRolloutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rollout, err := h.service.CreateRollout(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rollout)
}

func (h *Handler) ListRollouts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	rollouts, err := h.service.ListRollouts(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rollouts": rollouts})
}

func (h *Handler) GetRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	rollout, err := h.service.GetRollout(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rollout)
}

func (h *Handler) StartRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	rollout, err := h.service.StartRollout(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rollout)
}

func (h *Handler) PauseRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	rollout, err := h.service.PauseRollout(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rollout)
}

func (h *Handler) ResumeRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	rollout, err := h.service.ResumeRollout(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rollout)
}

func (h *Handler) UpdateTraffic(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req UpdateTrafficRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rollout, err := h.service.UpdateTraffic(c.Request.Context(), tenantID, c.Param("id"), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rollout)
}

func (h *Handler) CompleteRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	rollout, err := h.service.CompleteRollout(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rollout)
}

func (h *Handler) RollbackRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	rollout, err := h.service.RollbackRollout(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rollout)
}

func (h *Handler) EvaluateRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	result, err := h.service.EvaluateRollout(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) DeleteRollout(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if err := h.service.DeleteRollout(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
