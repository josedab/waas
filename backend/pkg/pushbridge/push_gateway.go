package pushbridge

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PushPlatformType represents a push notification platform for gateway
type PushPlatformType string

const (
	PushPlatformAPNs    PushPlatformType = "apns"
	PushPlatformFCM     PushPlatformType = "fcm"
	PushPlatformWebPush PushPlatformType = "web_push"
	PushPlatformHMS     PushPlatformType = "hms"
)

// PushGatewayStatus represents gateway operational status
type PushGatewayStatus string

const (
	GatewayStatusActive   PushGatewayStatus = "active"
	GatewayStatusDegraded PushGatewayStatus = "degraded"
	GatewayStatusDown     PushGatewayStatus = "down"
)

// PushGateway represents a push notification gateway configuration
type PushGateway struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	Platform        PushPlatformType  `json:"platform"`
	Name            string            `json:"name"`
	Enabled         bool              `json:"enabled"`
	Credentials     *PushCredentials  `json:"credentials,omitempty"`
	RateLimitPerSec int               `json:"rate_limit_per_sec"`
	Status          PushGatewayStatus `json:"status"`
	Stats           *PushGatewayStats `json:"stats,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// PushCredentials stores platform-specific credentials
type PushCredentials struct {
	APNsKeyID       string `json:"apns_key_id,omitempty"`
	APNsTeamID      string `json:"apns_team_id,omitempty"`
	APNsBundleID    string `json:"apns_bundle_id,omitempty"`
	APNsKeyPath     string `json:"apns_key_path,omitempty"`
	APNsSandbox     bool   `json:"apns_sandbox,omitempty"`
	FCMProjectID    string `json:"fcm_project_id,omitempty"`
	FCMCredPath     string `json:"fcm_cred_path,omitempty"`
	VAPIDPublicKey  string `json:"vapid_public_key,omitempty"`
	VAPIDPrivateKey string `json:"-"` // Never serialize
}

// PushGatewayStats tracks gateway metrics
type PushGatewayStats struct {
	TotalSent      int64      `json:"total_sent"`
	TotalDelivered int64      `json:"total_delivered"`
	TotalFailed    int64      `json:"total_failed"`
	AvgLatencyMs   float64    `json:"avg_latency_ms"`
	SuccessRate    float64    `json:"success_rate"`
	LastSentAt     *time.Time `json:"last_sent_at,omitempty"`
}

// OfflineQueueEntry represents a queued event for offline devices
type OfflineQueueEntry struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenant_id"`
	DeviceID    string        `json:"device_id"`
	WebhookID   string        `json:"webhook_id"`
	EventType   string        `json:"event_type"`
	Payload     string        `json:"payload"`
	Priority    int           `json:"priority"` // 0-10
	TTL         time.Duration `json:"ttl"`
	DeliveredAt *time.Time    `json:"delivered_at,omitempty"`
	ExpiredAt   *time.Time    `json:"expired_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
}

// DeliveryReceipt represents confirmation of push delivery
type DeliveryReceipt struct {
	ID             string           `json:"id"`
	TenantID       string           `json:"tenant_id"`
	DeviceID       string           `json:"device_id"`
	NotificationID string           `json:"notification_id"`
	Platform       PushPlatformType `json:"platform"`
	Status         string           `json:"status"` // delivered, read, dismissed, failed
	ReceivedAt     time.Time        `json:"received_at"`
	ReadAt         *time.Time       `json:"read_at,omitempty"`
	ErrorMessage   string           `json:"error_message,omitempty"`
}

// MobileDashboard provides mobile push analytics
type MobileDashboard struct {
	TenantID          string                     `json:"tenant_id"`
	TotalDevices      int64                      `json:"total_devices"`
	ActiveDevices     int64                      `json:"active_devices"`
	PlatformBreakdown map[PushPlatformType]int64 `json:"platform_breakdown"`
	DeliveryStats     *PushDeliveryStats         `json:"delivery_stats"`
	OfflineQueueSize  int64                      `json:"offline_queue_size"`
	Gateways          []PushGateway              `json:"gateways"`
	RecentReceipts    []DeliveryReceipt          `json:"recent_receipts"`
	GeneratedAt       time.Time                  `json:"generated_at"`
}

// PushDeliveryStats provides delivery statistics
type PushDeliveryStats struct {
	TotalSent      int64   `json:"total_sent"`
	TotalDelivered int64   `json:"total_delivered"`
	TotalRead      int64   `json:"total_read"`
	TotalFailed    int64   `json:"total_failed"`
	DeliveryRate   float64 `json:"delivery_rate"`
	ReadRate       float64 `json:"read_rate"`
	AvgDeliveryMs  float64 `json:"avg_delivery_ms"`
}

// CreatePushGatewayRequest represents a request to create a push gateway
type CreatePushGatewayRequest struct {
	Platform        PushPlatformType `json:"platform" binding:"required"`
	Name            string           `json:"name" binding:"required"`
	Credentials     *PushCredentials `json:"credentials" binding:"required"`
	RateLimitPerSec int              `json:"rate_limit_per_sec,omitempty"`
}

// SendPushRequest represents a request to send a push notification
type GatewaySendRequest struct {
	DeviceIDs   []string          `json:"device_ids" binding:"required"`
	Title       string            `json:"title" binding:"required"`
	Body        string            `json:"body" binding:"required"`
	Data        map[string]string `json:"data,omitempty"`
	Priority    int               `json:"priority,omitempty"`
	TTLSeconds  int               `json:"ttl_seconds,omitempty"`
	CollapseKey string            `json:"collapse_key,omitempty"`
	Badge       *int              `json:"badge,omitempty"`
	Sound       string            `json:"sound,omitempty"`
	ImageURL    string            `json:"image_url,omitempty"`
}

// PushGatewayRepository defines storage for push gateway
type PushGatewayRepository interface {
	CreateGateway(ctx context.Context, gateway *PushGateway) error
	GetGateway(ctx context.Context, tenantID, gatewayID string) (*PushGateway, error)
	ListGateways(ctx context.Context, tenantID string) ([]PushGateway, error)
	UpdateGateway(ctx context.Context, gateway *PushGateway) error
	DeleteGateway(ctx context.Context, tenantID, gatewayID string) error
	EnqueueOffline(ctx context.Context, entry *OfflineQueueEntry) error
	DequeueOffline(ctx context.Context, deviceID string, limit int) ([]OfflineQueueEntry, error)
	GetOfflineQueueSize(ctx context.Context, tenantID string) (int64, error)
	SaveReceipt(ctx context.Context, receipt *DeliveryReceipt) error
	ListReceipts(ctx context.Context, tenantID string, limit int) ([]DeliveryReceipt, error)
	GetDeliveryStats(ctx context.Context, tenantID string) (*PushDeliveryStats, error)
	GetDeviceCount(ctx context.Context, tenantID string) (total, active int64, err error)
	GetPlatformBreakdown(ctx context.Context, tenantID string) (map[PushPlatformType]int64, error)
}

// PushGatewayService manages push notification gateways and delivery
type PushGatewayService struct {
	repo PushGatewayRepository
}

// NewPushGatewayService creates a new push gateway service
func NewPushGatewayService(repo PushGatewayRepository) *PushGatewayService {
	return &PushGatewayService{repo: repo}
}

// CreateGateway creates a push notification gateway
func (s *PushGatewayService) CreateGateway(ctx context.Context, tenantID string, req *CreatePushGatewayRequest) (*PushGateway, error) {
	validPlatforms := map[PushPlatformType]bool{
		PushPlatformAPNs: true, PushPlatformFCM: true, PushPlatformWebPush: true, PushPlatformHMS: true,
	}
	if !validPlatforms[req.Platform] {
		return nil, fmt.Errorf("unsupported platform: %s", req.Platform)
	}

	rateLimit := req.RateLimitPerSec
	if rateLimit == 0 {
		rateLimit = 1000
	}

	now := time.Now()
	gateway := &PushGateway{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Platform:        req.Platform,
		Name:            req.Name,
		Enabled:         true,
		Credentials:     req.Credentials,
		RateLimitPerSec: rateLimit,
		Status:          GatewayStatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateGateway(ctx, gateway); err != nil {
		return nil, fmt.Errorf("failed to create gateway: %w", err)
	}

	// Clear sensitive data from response
	gateway.Credentials.VAPIDPrivateKey = ""

	return gateway, nil
}

// SendPush sends a push notification to specified devices
func (s *PushGatewayService) SendPush(ctx context.Context, tenantID string, req *GatewaySendRequest) (int, int, error) {
	if len(req.DeviceIDs) == 0 {
		return 0, 0, fmt.Errorf("at least one device ID is required")
	}

	// In production, this would dispatch to APNs/FCM/etc.
	// Here we record the send attempt and return success count
	sent := len(req.DeviceIDs)
	failed := 0

	for _, deviceID := range req.DeviceIDs {
		receipt := &DeliveryReceipt{
			ID:             uuid.New().String(),
			TenantID:       tenantID,
			DeviceID:       deviceID,
			NotificationID: uuid.New().String(),
			Status:         "delivered",
			ReceivedAt:     time.Now(),
		}
		if err := s.repo.SaveReceipt(ctx, receipt); err != nil {
			failed++
			receipt.Status = "failed"
			receipt.ErrorMessage = err.Error()
		}
	}

	return sent - failed, failed, nil
}

// QueueForOffline queues an event for an offline device
func (s *PushGatewayService) QueueForOffline(ctx context.Context, tenantID, deviceID, webhookID, eventType, payload string, priority int) error {
	ttl := 72 * time.Hour
	entry := &OfflineQueueEntry{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		DeviceID:  deviceID,
		WebhookID: webhookID,
		EventType: eventType,
		Payload:   payload,
		Priority:  priority,
		TTL:       ttl,
		CreatedAt: time.Now(),
	}

	return s.repo.EnqueueOffline(ctx, entry)
}

// DrainOfflineQueue delivers queued events when device comes online
func (s *PushGatewayService) DrainOfflineQueue(ctx context.Context, deviceID string, limit int) ([]OfflineQueueEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.DequeueOffline(ctx, deviceID, limit)
}

// GetDashboard generates the mobile push dashboard
func (s *PushGatewayService) GetDashboard(ctx context.Context, tenantID string) (*MobileDashboard, error) {
	totalDevices, activeDevices, _ := s.repo.GetDeviceCount(ctx, tenantID)

	platformBreakdown, _ := s.repo.GetPlatformBreakdown(ctx, tenantID)
	if platformBreakdown == nil {
		platformBreakdown = make(map[PushPlatformType]int64)
	}

	deliveryStats, _ := s.repo.GetDeliveryStats(ctx, tenantID)
	if deliveryStats == nil {
		deliveryStats = &PushDeliveryStats{}
	}

	queueSize, _ := s.repo.GetOfflineQueueSize(ctx, tenantID)

	gateways, _ := s.repo.ListGateways(ctx, tenantID)
	if gateways == nil {
		gateways = []PushGateway{}
	}

	receipts, _ := s.repo.ListReceipts(ctx, tenantID, 20)
	if receipts == nil {
		receipts = []DeliveryReceipt{}
	}

	return &MobileDashboard{
		TenantID:          tenantID,
		TotalDevices:      totalDevices,
		ActiveDevices:     activeDevices,
		PlatformBreakdown: platformBreakdown,
		DeliveryStats:     deliveryStats,
		OfflineQueueSize:  queueSize,
		Gateways:          gateways,
		RecentReceipts:    receipts,
		GeneratedAt:       time.Now(),
	}, nil
}

// GetGateway retrieves a gateway
func (s *PushGatewayService) GetGateway(ctx context.Context, tenantID, gatewayID string) (*PushGateway, error) {
	return s.repo.GetGateway(ctx, tenantID, gatewayID)
}

// ListGateways lists gateways
func (s *PushGatewayService) ListGateways(ctx context.Context, tenantID string) ([]PushGateway, error) {
	return s.repo.ListGateways(ctx, tenantID)
}

// DeleteGateway deletes a gateway
func (s *PushGatewayService) DeleteGateway(ctx context.Context, tenantID, gatewayID string) error {
	return s.repo.DeleteGateway(ctx, tenantID, gatewayID)
}
