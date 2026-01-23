// Package mobilesdk provides mobile SDK configuration and device management for iOS and Android
package mobilesdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrDeviceNotFound     = errors.New("device not found")
	ErrInvalidPlatform    = errors.New("invalid platform: must be ios or android")
	ErrInvalidToken       = errors.New("invalid push token")
	ErrSDKConfigNotFound  = errors.New("SDK config not found")
	ErrAppAlreadyExists   = errors.New("app already registered")
)

// Platform represents the mobile platform
type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

// MobileApp represents a registered mobile application
type MobileApp struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	Name           string            `json:"name"`
	BundleID       string            `json:"bundle_id"`       // iOS bundle ID or Android package name
	Platform       Platform          `json:"platform"`
	APNsKeyID      string            `json:"apns_key_id,omitempty"`      // iOS APNs key
	APNsTeamID     string            `json:"apns_team_id,omitempty"`     // iOS APNs team
	FCMProjectID   string            `json:"fcm_project_id,omitempty"`   // Android FCM project
	PushEnabled    bool              `json:"push_enabled"`
	WebhookTopics  []string          `json:"webhook_topics"`             // Which event types to push
	SDKVersion     string            `json:"sdk_version,omitempty"`
	Environment    string            `json:"environment"`                // sandbox or production
	Config         map[string]string `json:"config,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// DeviceRegistration represents a registered mobile device
type DeviceRegistration struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	AppID        string    `json:"app_id"`
	Platform     Platform  `json:"platform"`
	PushToken    string    `json:"push_token"`
	DeviceModel  string    `json:"device_model,omitempty"`
	OSVersion    string    `json:"os_version,omitempty"`
	SDKVersion   string    `json:"sdk_version,omitempty"`
	Locale       string    `json:"locale,omitempty"`
	Timezone     string    `json:"timezone,omitempty"`
	Topics       []string  `json:"topics"` // Subscribed webhook event topics
	IsActive     bool      `json:"is_active"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// PushDeliveryLog records a push notification delivery attempt
type PushDeliveryLog struct {
	ID           string    `json:"id"`
	DeviceID     string    `json:"device_id"`
	WebhookID    string    `json:"webhook_id"`
	EventType    string    `json:"event_type"`
	Status       string    `json:"status"` // sent, delivered, failed
	ErrorMessage string    `json:"error_message,omitempty"`
	PayloadSize  int       `json:"payload_size"`
	Timestamp    time.Time `json:"timestamp"`
}

// SDKConfig represents the configuration payload sent to mobile SDKs on init
type SDKConfig struct {
	AppID          string            `json:"app_id"`
	APIEndpoint    string            `json:"api_endpoint"`
	WebSocketURL   string            `json:"websocket_url"`
	PushEnabled    bool              `json:"push_enabled"`
	Topics         []string          `json:"topics"`
	RetryPolicy    RetryPolicy       `json:"retry_policy"`
	OfflineQueue   OfflineQueueConfig `json:"offline_queue"`
	BatteryOptimize bool             `json:"battery_optimize"`
}

// RetryPolicy configures retry behavior for the mobile SDK
type RetryPolicy struct {
	MaxRetries       int `json:"max_retries"`
	InitialDelayMs   int `json:"initial_delay_ms"`
	MaxDelayMs       int `json:"max_delay_ms"`
	BackoffMultiplier float64 `json:"backoff_multiplier"`
}

// OfflineQueueConfig configures offline queuing behavior
type OfflineQueueConfig struct {
	Enabled       bool `json:"enabled"`
	MaxItems      int  `json:"max_items"`
	MaxAgeSeconds int  `json:"max_age_seconds"`
	SyncOnConnect bool `json:"sync_on_connect"`
}

// Service manages mobile SDK operations
type Service struct {
	apps    map[string]*MobileApp           // appID -> app
	devices map[string]*DeviceRegistration   // deviceID -> device
	counter int64
}

// NewService creates a new mobile SDK service
func NewService() *Service {
	return &Service{
		apps:    make(map[string]*MobileApp),
		devices: make(map[string]*DeviceRegistration),
	}
}

// RegisterApp registers a new mobile application
func (s *Service) RegisterApp(_ context.Context, tenantID string, app *MobileApp) (*MobileApp, error) {
	if app.Platform != PlatformIOS && app.Platform != PlatformAndroid {
		return nil, ErrInvalidPlatform
	}
	if app.Name == "" || app.BundleID == "" {
		return nil, errors.New("name and bundle_id are required")
	}

	// Check for duplicate
	for _, existing := range s.apps {
		if existing.TenantID == tenantID && existing.BundleID == app.BundleID && existing.Platform == app.Platform {
			return nil, ErrAppAlreadyExists
		}
	}

	s.counter++
	app.ID = fmt.Sprintf("app-%d", s.counter)
	app.TenantID = tenantID
	app.PushEnabled = true
	if app.Environment == "" {
		app.Environment = "sandbox"
	}
	app.CreatedAt = time.Now()

	s.apps[app.ID] = app
	return app, nil
}

// GetApp retrieves an app by ID
func (s *Service) GetApp(_ context.Context, appID string) (*MobileApp, error) {
	app, ok := s.apps[appID]
	if !ok {
		return nil, ErrSDKConfigNotFound
	}
	return app, nil
}

// ListApps lists all apps for a tenant
func (s *Service) ListApps(_ context.Context, tenantID string) []*MobileApp {
	var result []*MobileApp
	for _, app := range s.apps {
		if app.TenantID == tenantID {
			result = append(result, app)
		}
	}
	return result
}

// RegisterDevice registers a mobile device for push notifications
func (s *Service) RegisterDevice(_ context.Context, tenantID string, reg *DeviceRegistration) (*DeviceRegistration, error) {
	if reg.Platform != PlatformIOS && reg.Platform != PlatformAndroid {
		return nil, ErrInvalidPlatform
	}
	if reg.PushToken == "" {
		return nil, ErrInvalidToken
	}

	// Check for existing device with same push token and update it
	for id, existing := range s.devices {
		if existing.TenantID == tenantID && existing.PushToken == reg.PushToken {
			existing.LastSeenAt = time.Now()
			existing.IsActive = true
			existing.OSVersion = reg.OSVersion
			existing.SDKVersion = reg.SDKVersion
			if len(reg.Topics) > 0 {
				existing.Topics = reg.Topics
			}
			return s.devices[id], nil
		}
	}

	s.counter++
	reg.ID = fmt.Sprintf("dev-%d", s.counter)
	reg.TenantID = tenantID
	reg.IsActive = true
	reg.LastSeenAt = time.Now()
	reg.CreatedAt = time.Now()

	s.devices[reg.ID] = reg
	return reg, nil
}

// UnregisterDevice deactivates a device
func (s *Service) UnregisterDevice(_ context.Context, deviceID string) error {
	dev, ok := s.devices[deviceID]
	if !ok {
		return ErrDeviceNotFound
	}
	dev.IsActive = false
	return nil
}

// GetSDKConfig returns the SDK configuration for a mobile app
func (s *Service) GetSDKConfig(_ context.Context, appID, apiEndpoint string) (*SDKConfig, error) {
	app, ok := s.apps[appID]
	if !ok {
		return nil, ErrSDKConfigNotFound
	}

	wsURL := strings.Replace(apiEndpoint, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL += "/ws"

	return &SDKConfig{
		AppID:       app.ID,
		APIEndpoint: apiEndpoint,
		WebSocketURL: wsURL,
		PushEnabled: app.PushEnabled,
		Topics:      app.WebhookTopics,
		RetryPolicy: RetryPolicy{
			MaxRetries:       5,
			InitialDelayMs:   1000,
			MaxDelayMs:       60000,
			BackoffMultiplier: 2.0,
		},
		OfflineQueue: OfflineQueueConfig{
			Enabled:       true,
			MaxItems:      100,
			MaxAgeSeconds: 86400,
			SyncOnConnect: true,
		},
		BatteryOptimize: true,
	}, nil
}

// ListActiveDevices returns all active devices for a tenant
func (s *Service) ListActiveDevices(_ context.Context, tenantID string) []*DeviceRegistration {
	var result []*DeviceRegistration
	for _, dev := range s.devices {
		if dev.TenantID == tenantID && dev.IsActive {
			result = append(result, dev)
		}
	}
	return result
}

// Handler provides HTTP endpoints for mobile SDK management
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers mobile SDK HTTP routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	mobile := rg.Group("/mobile")
	{
		// App management
		mobile.POST("/apps", h.RegisterApp)
		mobile.GET("/apps", h.ListApps)
		mobile.GET("/apps/:id", h.GetApp)
		mobile.GET("/apps/:id/config", h.GetSDKConfig)

		// Device management
		mobile.POST("/devices", h.RegisterDevice)
		mobile.GET("/devices", h.ListDevices)
		mobile.DELETE("/devices/:id", h.UnregisterDevice)
	}
}

// RegisterApp handles app registration
func (h *Handler) RegisterApp(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var app MobileApp
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.RegisterApp(c.Request.Context(), tenantID, &app)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == ErrInvalidPlatform || err == ErrAppAlreadyExists {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// ListApps handles listing apps
func (h *Handler) ListApps(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	apps := h.service.ListApps(c.Request.Context(), tenantID)
	c.JSON(http.StatusOK, gin.H{"apps": apps})
}

// GetApp handles getting a single app
func (h *Handler) GetApp(c *gin.Context) {
	app, err := h.service.GetApp(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, app)
}

// GetSDKConfig returns SDK configuration
func (h *Handler) GetSDKConfig(c *gin.Context) {
	config, err := h.service.GetSDKConfig(c.Request.Context(), c.Param("id"), c.Request.Host)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// RegisterDevice handles device registration
func (h *Handler) RegisterDevice(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var reg DeviceRegistration
	if err := c.ShouldBindJSON(&reg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.RegisterDevice(c.Request.Context(), tenantID, &reg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// ListDevices handles listing active devices
func (h *Handler) ListDevices(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	devices := h.service.ListActiveDevices(c.Request.Context(), tenantID)
	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// UnregisterDevice handles device unregistration
func (h *Handler) UnregisterDevice(c *gin.Context) {
	if err := h.service.UnregisterDevice(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
