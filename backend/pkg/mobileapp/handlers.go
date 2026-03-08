package mobileapp

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/httputil"
)

// Handler provides HTTP endpoints for the mobile app.
type Handler struct {
	service *Service
}

// NewHandler creates a new mobile app handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all mobile app routes.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/mobile")
	{
		group.POST("/devices", h.RegisterDevice)
		group.GET("/devices", h.ListDevices)
		group.DELETE("/devices/:id", h.UnregisterDevice)
		group.PUT("/devices/:id/preferences", h.UpdatePreferences)
		group.GET("/dashboard", h.GetDashboard)
		group.GET("/live-payloads", h.GetLivePayloads)
		group.GET("/notifications", h.ListNotifications)
		group.POST("/notifications/:id/read", h.MarkNotificationRead)
		group.POST("/notifications/test", h.SendTestNotification)
	}
}

// RegisterDevice registers a mobile device.
// @Summary Register mobile device
// @Tags mobile
// @Accept json
// @Produce json
// @Success 201 {object} DeviceRegistration
// @Router /mobile/devices [post]
func (h *Handler) RegisterDevice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := h.service.RegisterDevice(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, device)
}

// ListDevices returns all devices for the tenant.
// @Summary List devices
// @Tags mobile
// @Produce json
// @Success 200 {array} DeviceRegistration
// @Router /mobile/devices [get]
func (h *Handler) ListDevices(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	devices, err := h.service.ListDevices(tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, devices)
}

// UnregisterDevice removes a device registration.
// @Summary Unregister device
// @Tags mobile
// @Param id path string true "Device ID"
// @Produce json
// @Router /mobile/devices/{id} [delete]
func (h *Handler) UnregisterDevice(c *gin.Context) {
	if err := h.service.UnregisterDevice(c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// UpdatePreferences updates notification preferences for a device.
// @Summary Update notification preferences
// @Tags mobile
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Success 200 {object} DeviceRegistration
// @Router /mobile/devices/{id}/preferences [put]
func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req NotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := h.service.UpdatePreferences(c.Param("id"), &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, device)
}

// GetDashboard returns the mobile dashboard overview.
// @Summary Get mobile dashboard
// @Tags mobile
// @Produce json
// @Success 200 {object} MobileDashboard
// @Router /mobile/dashboard [get]
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	dashboard, err := h.service.GetDashboard(tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, dashboard)
}

// GetLivePayloads returns recent webhook payloads.
// @Summary Get live payloads
// @Tags mobile
// @Produce json
// @Param limit query int false "Max payloads"
// @Success 200 {array} LivePayloadEvent
// @Router /mobile/live-payloads [get]
func (h *Handler) GetLivePayloads(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	payloads, err := h.service.GetLivePayloads(tenantID, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, payloads)
}

// ListNotifications returns recent notifications.
// @Summary List notifications
// @Tags mobile
// @Produce json
// @Param limit query int false "Max notifications"
// @Success 200 {array} PushNotification
// @Router /mobile/notifications [get]
func (h *Handler) ListNotifications(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	notifications, err := h.service.ListNotifications(tenantID, limit)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, notifications)
}

// MarkNotificationRead marks a notification as read.
// @Summary Mark notification read
// @Tags mobile
// @Param id path string true "Notification ID"
// @Produce json
// @Router /mobile/notifications/{id}/read [post]
func (h *Handler) MarkNotificationRead(c *gin.Context) {
	if err := h.service.MarkNotificationRead(c.Param("id")); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "read"})
}

// SendTestNotification sends a test push notification.
// @Summary Send test notification
// @Tags mobile
// @Accept json
// @Produce json
// @Success 201 {object} PushNotification
// @Router /mobile/notifications/test [post]
func (h *Handler) SendTestNotification(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification, err := h.service.SendTestNotification(tenantID, req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, notification)
}
