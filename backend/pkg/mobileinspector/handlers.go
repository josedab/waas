package mobileinspector

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler provides HTTP endpoints for the mobile inspector.
type Handler struct {
	service *Service
}

// NewHandler creates a new mobile inspector handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all mobile inspector routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/mobile")
	{
		group.POST("/register", h.RegisterDevice)
		group.POST("/logout", h.Logout)

		// Authenticated mobile routes
		auth := group.Group("")
		auth.Use(h.mobileAuth())
		{
			auth.GET("/feed", h.GetEventFeed)
			auth.GET("/endpoints", h.GetEndpointOverviews)
			auth.GET("/alerts", h.GetAlerts)
			auth.POST("/alerts/:id/acknowledge", h.AcknowledgeAlert)
			auth.POST("/alerts/snooze", h.SnoozeAlerts)
			auth.GET("/alert-config", h.GetAlertConfig)
			auth.PUT("/alert-config", h.UpdateAlertConfig)
		}
	}
}

func (h *Handler) mobileAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Mobile-Token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Mobile-Token header required"})
			c.Abort()
			return
		}
		session, err := h.service.ValidateSession(token)
		if err != nil {
			httputil.InternalErrorGeneric(c, err)
			c.Abort()
			return
		}
		c.Set("mobile_session", session)
		c.Next()
	}
}

func getMobileSession(c *gin.Context) *MobileSession {
	val, _ := c.Get("mobile_session")
	if s, ok := val.(*MobileSession); ok {
		return s
	}
	return nil
}

// RegisterDevice registers a mobile device.
// @Summary Register mobile device
// @Tags mobile
// @Accept json
// @Produce json
// @Success 201 {object} MobileSession
// @Router /mobile/register [post]
func (h *Handler) RegisterDevice(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "default"
	}

	var reg DeviceRegistration
	if err := c.ShouldBindJSON(&reg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.service.RegisterDevice(tenantID, userID, &reg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, session)
}

// Logout invalidates a session.
// @Summary Mobile logout
// @Tags mobile
// @Produce json
// @Router /mobile/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	session := getMobileSession(c)
	if session == nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	_ = h.service.Logout(session.ID)
	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

// GetEventFeed returns the event feed with sync support.
// @Summary Get event feed
// @Tags mobile
// @Produce json
// @Param last_sync_at query string false "Last sync timestamp (RFC3339)"
// @Param limit query int false "Max events"
// @Success 200 {object} SyncResponse
// @Router /mobile/feed [get]
func (h *Handler) GetEventFeed(c *gin.Context) {
	session := getMobileSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req SyncRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.GetEventFeed(session.TenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetEndpointOverviews returns endpoint summaries.
// @Summary Get endpoint overviews
// @Tags mobile
// @Produce json
// @Success 200 {array} EndpointOverview
// @Router /mobile/endpoints [get]
func (h *Handler) GetEndpointOverviews(c *gin.Context) {
	session := getMobileSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	overviews, err := h.service.GetEndpointOverviews(session.TenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, overviews)
}

// GetAlerts returns recent alerts.
// @Summary Get alerts
// @Tags mobile
// @Produce json
// @Param limit query int false "Max alerts"
// @Success 200 {array} AlertNotification
// @Router /mobile/alerts [get]
func (h *Handler) GetAlerts(c *gin.Context) {
	session := getMobileSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	alerts, err := h.service.GetAlerts(session.UserID, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, alerts)
}

// AcknowledgeAlert acknowledges an alert.
// @Summary Acknowledge alert
// @Tags mobile
// @Param id path string true "Alert ID"
// @Produce json
// @Router /mobile/alerts/{id}/acknowledge [post]
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	if err := h.service.AcknowledgeAlert(c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
}

// SnoozeAlerts snoozes alerts for a duration.
// @Summary Snooze alerts
// @Tags mobile
// @Accept json
// @Produce json
// @Router /mobile/alerts/snooze [post]
func (h *Handler) SnoozeAlerts(c *gin.Context) {
	session := getMobileSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Duration string `json:"duration" binding:"required"` // e.g. "1h", "4h"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid duration"})
		return
	}

	cfg, err := h.service.SnoozeAlerts(session.UserID, duration)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// GetAlertConfig returns alert configuration.
// @Summary Get alert config
// @Tags mobile
// @Produce json
// @Router /mobile/alert-config [get]
func (h *Handler) GetAlertConfig(c *gin.Context) {
	session := getMobileSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	cfg, err := h.service.GetAlertConfig(session.UserID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// UpdateAlertConfig updates alert thresholds.
// @Summary Update alert config
// @Tags mobile
// @Accept json
// @Produce json
// @Router /mobile/alert-config [put]
func (h *Handler) UpdateAlertConfig(c *gin.Context) {
	session := getMobileSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var cfg AlertConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.UpdateAlertConfig(session.UserID, &cfg)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
