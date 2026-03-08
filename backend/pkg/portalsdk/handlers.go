package portalsdk

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the portal SDK.
type Handler struct {
	service *Service
}

// NewHandler creates a new portal SDK handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers portal SDK routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/portal-sdk")
	{
		// Portal configs
		g.POST("/configs", h.CreateConfig)
		g.GET("/configs", h.ListConfigs)
		g.GET("/configs/:id", h.GetConfig)
		g.PUT("/configs/:id", h.UpdateConfig)
		g.DELETE("/configs/:id", h.DeleteConfig)

		// Sessions
		g.POST("/sessions", h.CreateSession)
		g.POST("/sessions/validate", h.ValidateSession)
		g.DELETE("/sessions/:id", h.RevokeSession)

		// SDK generation
		g.POST("/generate-snippet", h.GenerateSDKSnippet)

		// Stats
		g.GET("/configs/:id/stats", h.GetUsageStats)
	}
}

func (h *Handler) CreateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreatePortalConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.CreateConfig(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

func (h *Handler) GetConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	configID := c.Param("id")

	config, err := h.service.GetConfig(c.Request.Context(), tenantID, configID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *Handler) ListConfigs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	configs, err := h.service.ListConfigs(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

func (h *Handler) UpdateConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	configID := c.Param("id")
	var req UpdatePortalConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.UpdateConfig(c.Request.Context(), tenantID, configID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *Handler) DeleteConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	configID := c.Param("id")

	if err := h.service.DeleteConfig(c.Request.Context(), tenantID, configID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) CreateSession(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.service.CreateSession(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, session)
}

func (h *Handler) ValidateSession(c *gin.Context) {
	var body struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.service.ValidateSession(c.Request.Context(), body.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *Handler) RevokeSession(c *gin.Context) {
	sessionID := c.Param("id")

	if err := h.service.RevokeSession(c.Request.Context(), sessionID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) GenerateSDKSnippet(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req GenerateSDKRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	snippet, err := h.service.GenerateSDKSnippet(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snippet": snippet, "framework": req.Framework})
}

func (h *Handler) GetUsageStats(c *gin.Context) {
	configID := c.Param("id")

	stats, err := h.service.GetUsageStats(c.Request.Context(), configID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}
