package pushbridge

import (
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handlers provides HTTP handlers for push bridge operations
type Handlers struct {
	service *Service
}

// NewHandlers creates new push bridge handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers push bridge routes
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	push := r.Group("/push")
	{
		// Device management
		push.POST("/devices", h.RegisterDevice)
		push.GET("/devices", h.ListDevices)
		push.GET("/devices/:id", h.GetDevice)
		push.PUT("/devices/:id", h.UpdateDevice)
		push.DELETE("/devices/:id", h.DeleteDevice)
		push.POST("/devices/:id/unregister", h.UnregisterDevice)

		// Mapping management
		push.POST("/mappings", h.CreateMapping)
		push.GET("/mappings", h.ListMappings)
		push.GET("/mappings/:id", h.GetMapping)
		push.DELETE("/mappings/:id", h.DeleteMapping)

		// Sending notifications
		push.POST("/send", h.SendPush)
		push.POST("/webhook/:webhookId", h.ProcessWebhook)

		// Notification tracking
		push.GET("/notifications", h.ListNotifications)
		push.GET("/notifications/:id", h.GetNotification)
		push.POST("/notifications/:id/opened", h.RecordOpen)

		// Stats and info
		push.GET("/stats", h.GetStats)
		push.GET("/platforms", h.GetSupportedPlatforms)
		push.GET("/providers", h.ListProviders)
	}
}

// RegisterDevice registers a mobile device
// @Summary Register device
// @Tags Push
// @Accept json
// @Produce json
// @Param request body RegisterDeviceRequest true "Device registration"
// @Success 201 {object} PushDevice
// @Router /push/devices [post]
func (h *Handlers) RegisterDevice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := h.service.RegisterDevice(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, device)
}

// ListDevices lists registered devices
// @Summary List devices
// @Tags Push
// @Produce json
// @Param platform query string false "Filter by platform"
// @Param status query string false "Filter by status"
// @Param user_id query string false "Filter by user ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} PushDevice
// @Router /push/devices [get]
func (h *Handlers) ListDevices(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	filter := &DeviceFilter{}
	if platform := c.Query("platform"); platform != "" {
		p := Platform(platform)
		filter.Platform = &p
	}
	if status := c.Query("status"); status != "" {
		s := DeviceStatus(status)
		filter.Status = &s
	}
	if userID := c.Query("user_id"); userID != "" {
		filter.UserID = &userID
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(c.Query("offset")); err == nil {
		filter.Offset = offset
	}

	devices, err := h.service.ListDevices(c.Request.Context(), tenantID, filter)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, devices)
}

// GetDevice gets a device by ID
// @Summary Get device
// @Tags Push
// @Produce json
// @Param id path string true "Device ID"
// @Success 200 {object} PushDevice
// @Router /push/devices/{id} [get]
func (h *Handlers) GetDevice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deviceID := c.Param("id")

	device, err := h.service.GetDevice(c.Request.Context(), tenantID, deviceID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, device)
}

// UpdateDevice updates a device
// @Summary Update device
// @Tags Push
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Param request body UpdateDeviceRequest true "Update data"
// @Success 200 {object} PushDevice
// @Router /push/devices/{id} [put]
func (h *Handlers) UpdateDevice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deviceID := c.Param("id")

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := h.service.UpdateDevice(c.Request.Context(), tenantID, deviceID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, device)
}

// DeleteDevice deletes a device
// @Summary Delete device
// @Tags Push
// @Param id path string true "Device ID"
// @Success 204
// @Router /push/devices/{id} [delete]
func (h *Handlers) DeleteDevice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deviceID := c.Param("id")

	if err := h.service.DeleteDevice(c.Request.Context(), tenantID, deviceID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// UnregisterDevice unregisters a device
// @Summary Unregister device
// @Tags Push
// @Param id path string true "Device ID"
// @Success 200
// @Router /push/devices/{id}/unregister [post]
func (h *Handlers) UnregisterDevice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	deviceID := c.Param("id")

	if err := h.service.UnregisterDevice(c.Request.Context(), tenantID, deviceID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "unregistered"})
}

// CreateMapping creates a push mapping
// @Summary Create push mapping
// @Tags Push
// @Accept json
// @Produce json
// @Param request body CreateMappingRequest true "Mapping configuration"
// @Success 201 {object} PushMapping
// @Router /push/mappings [post]
func (h *Handlers) CreateMapping(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req CreateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mapping, err := h.service.CreateMapping(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusCreated, mapping)
}

// ListMappings lists push mappings
// @Summary List push mappings
// @Tags Push
// @Produce json
// @Success 200 {array} PushMapping
// @Router /push/mappings [get]
func (h *Handlers) ListMappings(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	mappings, err := h.service.ListMappings(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, mappings)
}

// GetMapping gets a push mapping
// @Summary Get push mapping
// @Tags Push
// @Produce json
// @Param id path string true "Mapping ID"
// @Success 200 {object} PushMapping
// @Router /push/mappings/{id} [get]
func (h *Handlers) GetMapping(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	mappingID := c.Param("id")

	mapping, err := h.service.GetMapping(c.Request.Context(), tenantID, mappingID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, mapping)
}

// DeleteMapping deletes a push mapping
// @Summary Delete push mapping
// @Tags Push
// @Param id path string true "Mapping ID"
// @Success 204
// @Router /push/mappings/{id} [delete]
func (h *Handlers) DeleteMapping(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	mappingID := c.Param("id")

	if err := h.service.DeleteMapping(c.Request.Context(), tenantID, mappingID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// SendPush sends a push notification
// @Summary Send push notification
// @Tags Push
// @Accept json
// @Produce json
// @Param request body SendPushRequest true "Push notification"
// @Success 200 {array} PushNotification
// @Router /push/send [post]
func (h *Handlers) SendPush(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req SendPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notifications, err := h.service.SendPush(c.Request.Context(), tenantID, &req)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sent":          len(notifications),
		"notifications": notifications,
	})
}

// ProcessWebhook processes a webhook and sends push notifications
// @Summary Process webhook for push
// @Tags Push
// @Accept json
// @Produce json
// @Param webhookId path string true "Webhook ID"
// @Success 200 {array} PushNotification
// @Router /push/webhook/{webhookId} [post]
func (h *Handlers) ProcessWebhook(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	webhookID := c.Param("webhookId")

	payload, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
		return
	}

	notifications, err := h.service.ProcessWebhook(c.Request.Context(), tenantID, webhookID, payload)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sent":          len(notifications),
		"notifications": notifications,
	})
}

// ListNotifications lists push notifications
// @Summary List push notifications
// @Tags Push
// @Produce json
// @Param device_id query string false "Filter by device ID"
// @Param mapping_id query string false "Filter by mapping ID"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} PushNotification
// @Router /push/notifications [get]
func (h *Handlers) ListNotifications(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	filter := &NotificationFilter{}
	if deviceID := c.Query("device_id"); deviceID != "" {
		filter.DeviceID = &deviceID
	}
	if mappingID := c.Query("mapping_id"); mappingID != "" {
		filter.MappingID = &mappingID
	}
	if status := c.Query("status"); status != "" {
		s := DeliveryStatus(status)
		filter.Status = &s
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(c.Query("offset")); err == nil {
		filter.Offset = offset
	}

	notifications, err := h.service.ListNotifications(c.Request.Context(), tenantID, filter)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// GetNotification gets a notification by ID
// @Summary Get notification
// @Tags Push
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} PushNotification
// @Router /push/notifications/{id} [get]
func (h *Handlers) GetNotification(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	notifID := c.Param("id")

	notification, err := h.service.GetNotification(c.Request.Context(), tenantID, notifID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, notification)
}

// RecordOpen records notification open event
// @Summary Record notification opened
// @Tags Push
// @Param id path string true "Notification ID"
// @Success 200
// @Router /push/notifications/{id}/opened [post]
func (h *Handlers) RecordOpen(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	notifID := c.Param("id")

	if err := h.service.RecordOpen(c.Request.Context(), tenantID, notifID); err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

// GetStats gets push notification statistics
// @Summary Get push statistics
// @Tags Push
// @Produce json
// @Param from query string false "From date (RFC3339)"
// @Param to query string false "To date (RFC3339)"
// @Success 200 {object} PushStats
// @Router /push/stats [get]
func (h *Handlers) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	from := time.Now().AddDate(0, 0, -7) // Default last 7 days
	to := time.Now()

	if fromStr := c.Query("from"); fromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = parsed
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = parsed
		}
	}

	stats, err := h.service.GetStats(c.Request.Context(), tenantID, from, to)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetSupportedPlatforms returns supported platforms
// @Summary Get supported platforms
// @Tags Push
// @Produce json
// @Success 200 {array} PlatformInfo
// @Router /push/platforms [get]
func (h *Handlers) GetSupportedPlatforms(c *gin.Context) {
	platforms := h.service.GetSupportedPlatforms()
	c.JSON(http.StatusOK, platforms)
}

// ListProviders lists configured push providers
// @Summary List push providers
// @Tags Push
// @Produce json
// @Success 200 {array} PushProviderConfig
// @Router /push/providers [get]
func (h *Handlers) ListProviders(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	providers, err := h.service.ListProviders(c.Request.Context(), tenantID)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, providers)
}
