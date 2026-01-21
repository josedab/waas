package pushbridge

import (
	"encoding/json"
	"fmt"
	"time"
)

// PushDevice represents a registered mobile device
type PushDevice struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`
	UserID        string            `json:"user_id,omitempty"`
	Platform      Platform          `json:"platform"`
	PushToken     string            `json:"push_token"`
	DeviceInfo    DeviceInfo        `json:"device_info"`
	Status        DeviceStatus      `json:"status"`
	Preferences   PushPreferences   `json:"preferences"`
	Tags          []string          `json:"tags,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	LastActiveAt  *time.Time        `json:"last_active_at,omitempty"`
	RegisteredAt  time.Time         `json:"registered_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Platform defines supported mobile platforms
type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformWeb     Platform = "web"      // Web push
	PlatformHuawei  Platform = "huawei"   // Huawei Push Kit
)

// DeviceStatus defines device states
type DeviceStatus string

const (
	DeviceActive    DeviceStatus = "active"
	DeviceInactive  DeviceStatus = "inactive"
	DeviceUnregistered DeviceStatus = "unregistered"
	DeviceSuspended DeviceStatus = "suspended"
)

// DeviceInfo holds device metadata
type DeviceInfo struct {
	Model        string `json:"model,omitempty"`
	OS           string `json:"os,omitempty"`
	OSVersion    string `json:"os_version,omitempty"`
	AppVersion   string `json:"app_version,omitempty"`
	SDKVersion   string `json:"sdk_version,omitempty"`
	Locale       string `json:"locale,omitempty"`
	Timezone     string `json:"timezone,omitempty"`
	BundleID     string `json:"bundle_id,omitempty"`     // iOS
	PackageName  string `json:"package_name,omitempty"` // Android
}

// PushPreferences defines user notification preferences
type PushPreferences struct {
	Enabled         bool     `json:"enabled"`
	QuietHoursStart string   `json:"quiet_hours_start,omitempty"` // HH:MM
	QuietHoursEnd   string   `json:"quiet_hours_end,omitempty"`
	AllowedTypes    []string `json:"allowed_types,omitempty"`
	BlockedTypes    []string `json:"blocked_types,omitempty"`
	MaxPerHour      int      `json:"max_per_hour,omitempty"`
	MaxPerDay       int      `json:"max_per_day,omitempty"`
	Sound           bool     `json:"sound"`
	Badge           bool     `json:"badge"`
	Alert           bool     `json:"alert"`
	Priority        string   `json:"priority,omitempty"` // high, normal, low
}

// PushMapping defines webhook to push notification mapping
type PushMapping struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	WebhookID   string          `json:"webhook_id,omitempty"`
	EventType   string          `json:"event_type,omitempty"`
	Enabled     bool            `json:"enabled"`
	Config      MappingConfig   `json:"config"`
	Template    PushTemplate    `json:"template"`
	Targeting   TargetingRules  `json:"targeting"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// MappingConfig defines mapping configuration
type MappingConfig struct {
	Platform         []Platform `json:"platforms,omitempty"`          // If empty, all platforms
	RespectQuietHours bool      `json:"respect_quiet_hours"`
	Collapsible      bool       `json:"collapsible"`                 // Collapse similar notifications
	CollapseKey      string     `json:"collapse_key,omitempty"`
	TTLSeconds       int        `json:"ttl_seconds"`                 // Time to live
	Priority         string     `json:"priority"`                    // high, normal
	DryRun           bool       `json:"dry_run,omitempty"`
}

// PushTemplate defines notification template
type PushTemplate struct {
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	TitleLocKey  string            `json:"title_loc_key,omitempty"`
	TitleLocArgs []string          `json:"title_loc_args,omitempty"`
	BodyLocKey   string            `json:"body_loc_key,omitempty"`
	BodyLocArgs  []string          `json:"body_loc_args,omitempty"`
	Image        string            `json:"image,omitempty"`
	Icon         string            `json:"icon,omitempty"`
	Sound        string            `json:"sound,omitempty"`
	Badge        *int              `json:"badge,omitempty"`
	ClickAction  string            `json:"click_action,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	
	// iOS specific
	IOSCategory     string `json:"ios_category,omitempty"`
	IOSThreadID     string `json:"ios_thread_id,omitempty"`
	IOSInterruptionLevel string `json:"ios_interruption_level,omitempty"` // passive, active, time-sensitive, critical
	
	// Android specific
	AndroidChannelID string `json:"android_channel_id,omitempty"`
	AndroidTag       string `json:"android_tag,omitempty"`
	AndroidColor     string `json:"android_color,omitempty"`
}

// TargetingRules defines who receives the push
type TargetingRules struct {
	Type       TargetingType `json:"type"`
	UserIDs    []string      `json:"user_ids,omitempty"`
	DeviceIDs  []string      `json:"device_ids,omitempty"`
	Tags       []string      `json:"tags,omitempty"`
	Segments   []string      `json:"segments,omitempty"`
	Expression string        `json:"expression,omitempty"` // Complex targeting expression
}

// TargetingType defines targeting modes
type TargetingType string

const (
	TargetAll       TargetingType = "all"
	TargetUsers     TargetingType = "users"
	TargetDevices   TargetingType = "devices"
	TargetTags      TargetingType = "tags"
	TargetSegments  TargetingType = "segments"
	TargetExpression TargetingType = "expression"
)

// PushNotification represents a push notification
type PushNotification struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	MappingID    string          `json:"mapping_id,omitempty"`
	WebhookID    string          `json:"webhook_id,omitempty"`
	Platform     Platform        `json:"platform"`
	DeviceID     string          `json:"device_id"`
	PushToken    string          `json:"push_token"`
	Status       DeliveryStatus  `json:"status"`
	Payload      json.RawMessage `json:"payload"`
	Response     *ProviderResponse `json:"response,omitempty"`
	Attempts     int             `json:"attempts"`
	LastAttempt  *time.Time      `json:"last_attempt,omitempty"`
	DeliveredAt  *time.Time      `json:"delivered_at,omitempty"`
	OpenedAt     *time.Time      `json:"opened_at,omitempty"`
	Error        string          `json:"error,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// DeliveryStatus defines delivery states
type DeliveryStatus string

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliveryQueued    DeliveryStatus = "queued"
	DeliverySent      DeliveryStatus = "sent"
	DeliveryDelivered DeliveryStatus = "delivered"
	DeliveryOpened    DeliveryStatus = "opened"
	DeliveryFailed    DeliveryStatus = "failed"
	DeliveryDropped   DeliveryStatus = "dropped"
)

// ProviderResponse holds push provider response
type ProviderResponse struct {
	Provider   string `json:"provider"`
	MessageID  string `json:"message_id,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`
}

// PushProviderConfig defines push provider configuration
type PushProviderConfig struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Provider    ProviderType    `json:"provider"`
	Name        string          `json:"name"`
	Enabled     bool            `json:"enabled"`
	Config      json.RawMessage `json:"config"`
	Credentials json.RawMessage `json:"-"` // Never expose
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ProviderType defines supported push providers
type ProviderType string

const (
	ProviderFCM    ProviderType = "fcm"     // Firebase Cloud Messaging
	ProviderAPNS   ProviderType = "apns"    // Apple Push Notification Service
	ProviderHuawei ProviderType = "huawei"  // Huawei Push Kit
	ProviderWebPush ProviderType = "webpush" // Web Push
)

// FCMConfig holds FCM configuration
type FCMConfig struct {
	ProjectID      string `json:"project_id"`
	ServiceAccount string `json:"-"` // JSON service account key (stored encrypted)
	DryRun         bool   `json:"dry_run,omitempty"`
}

// APNSConfig holds APNs configuration
type APNSConfig struct {
	TeamID      string `json:"team_id"`
	KeyID       string `json:"key_id"`
	BundleID    string `json:"bundle_id"`
	PrivateKey  string `json:"-"` // .p8 key contents (stored encrypted)
	Production  bool   `json:"production"`
}

// OfflineQueue represents queued notifications for offline devices
type OfflineQueue struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	DeviceID     string          `json:"device_id"`
	Notification json.RawMessage `json:"notification"`
	Priority     int             `json:"priority"`
	ExpiresAt    time.Time       `json:"expires_at"`
	CreatedAt    time.Time       `json:"created_at"`
}

// SendPushRequest represents a push notification request
type SendPushRequest struct {
	MappingID    string            `json:"mapping_id,omitempty"`
	Title        string            `json:"title,omitempty"`
	Body         string            `json:"body,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Image        string            `json:"image,omitempty"`
	Targeting    *TargetingRules   `json:"targeting,omitempty"`
	Config       *MappingConfig    `json:"config,omitempty"`
	ScheduleAt   *time.Time        `json:"schedule_at,omitempty"`
}

// RegisterDeviceRequest represents device registration
type RegisterDeviceRequest struct {
	Platform    Platform          `json:"platform" binding:"required"`
	PushToken   string            `json:"push_token" binding:"required"`
	UserID      string            `json:"user_id,omitempty"`
	DeviceInfo  *DeviceInfo       `json:"device_info,omitempty"`
	Preferences *PushPreferences  `json:"preferences,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UpdateDeviceRequest represents device update
type UpdateDeviceRequest struct {
	PushToken   *string           `json:"push_token,omitempty"`
	UserID      *string           `json:"user_id,omitempty"`
	DeviceInfo  *DeviceInfo       `json:"device_info,omitempty"`
	Preferences *PushPreferences  `json:"preferences,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CreateMappingRequest represents mapping creation
type CreateMappingRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description,omitempty"`
	WebhookID   string          `json:"webhook_id,omitempty"`
	EventType   string          `json:"event_type,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	Config      *MappingConfig  `json:"config,omitempty"`
	Template    *PushTemplate   `json:"template,omitempty"`
	Targeting   *TargetingRules `json:"targeting,omitempty"`
}

// PushStats represents push statistics
type PushStats struct {
	TenantID     string    `json:"tenant_id"`
	Period       string    `json:"period"`
	TotalSent    int64     `json:"total_sent"`
	TotalDelivered int64   `json:"total_delivered"`
	TotalOpened  int64     `json:"total_opened"`
	TotalFailed  int64     `json:"total_failed"`
	TotalDropped int64     `json:"total_dropped"`
	DeliveryRate float64   `json:"delivery_rate"`
	OpenRate     float64   `json:"open_rate"`
	ByPlatform   map[Platform]PlatformStats `json:"by_platform"`
}

// PlatformStats holds platform-specific stats
type PlatformStats struct {
	Sent      int64   `json:"sent"`
	Delivered int64   `json:"delivered"`
	Opened    int64   `json:"opened"`
	Failed    int64   `json:"failed"`
	Rate      float64 `json:"delivery_rate"`
}

// DeviceSegment represents a user segment
type DeviceSegment struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Query       string    `json:"query"` // Segment query expression
	DeviceCount int       `json:"device_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GetDefaultPreferences returns default push preferences
func GetDefaultPreferences() PushPreferences {
	return PushPreferences{
		Enabled:    true,
		Sound:      true,
		Badge:      true,
		Alert:      true,
		Priority:   "normal",
		MaxPerHour: 10,
		MaxPerDay:  50,
	}
}

// GetDefaultMappingConfig returns default mapping config
func GetDefaultMappingConfig() MappingConfig {
	return MappingConfig{
		RespectQuietHours: true,
		Collapsible:       false,
		TTLSeconds:        86400, // 24 hours
		Priority:          "normal",
	}
}

// GetSupportedPlatforms returns supported platforms
func GetSupportedPlatforms() []PlatformInfo {
	return []PlatformInfo{
		{Platform: PlatformIOS, Name: "iOS", Provider: ProviderAPNS, Description: "Apple iOS devices via APNs"},
		{Platform: PlatformAndroid, Name: "Android", Provider: ProviderFCM, Description: "Android devices via FCM"},
		{Platform: PlatformWeb, Name: "Web", Provider: ProviderWebPush, Description: "Web browsers via Web Push"},
		{Platform: PlatformHuawei, Name: "Huawei", Provider: ProviderHuawei, Description: "Huawei devices via Push Kit"},
	}
}

// PlatformInfo describes a platform
type PlatformInfo struct {
	Platform    Platform     `json:"platform"`
	Name        string       `json:"name"`
	Provider    ProviderType `json:"provider"`
	Description string       `json:"description"`
}

// Type aliases for test compatibility
type PushProviderType = ProviderType

const (
	ProviderFirebase PushProviderType = ProviderFCM
)

// DeviceRegistration alias for PushDevice
type DeviceRegistration = PushDevice

// TargetingRule represents a targeting rule for push notifications
type TargetingRule struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Priority   int             `json:"priority"`
	Conditions []RuleCondition `json:"conditions"`
	Enabled    bool            `json:"enabled"`
}

// RuleCondition defines a targeting condition
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// QuietHours defines notification quiet hours
type QuietHours struct {
	Enabled  bool   `json:"enabled"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Timezone string `json:"timezone"`
}

// Validate validates quiet hours configuration
func (qh *QuietHours) Validate() error {
	if !qh.Enabled {
		return nil
	}
	
	// Validate time format HH:MM
	if !isValidTimeFormat(qh.Start) {
		return fmt.Errorf("invalid start time format: %s", qh.Start)
	}
	if !isValidTimeFormat(qh.End) {
		return fmt.Errorf("invalid end time format: %s", qh.End)
	}
	
	// Validate timezone
	_, err := time.LoadLocation(qh.Timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone: %s", qh.Timezone)
	}
	
	return nil
}

func isValidTimeFormat(t string) bool {
	if len(t) != 5 || t[2] != ':' {
		return false
	}
	hour := t[0:2]
	min := t[3:5]
	
	h := (hour[0]-'0')*10 + (hour[1] - '0')
	m := (min[0]-'0')*10 + (min[1] - '0')
	
	return h <= 23 && m <= 59
}

// ProviderConfig defines provider configuration
type ProviderConfig struct {
	Provider ProviderType           `json:"provider"`
	Enabled  bool                   `json:"enabled"`
	Config   map[string]interface{} `json:"config"`
}

// DeliveryAttempt represents a push delivery attempt
type DeliveryAttempt struct {
	ID             string         `json:"id"`
	NotificationID string         `json:"notification_id"`
	DeviceID       string         `json:"device_id"`
	Provider       ProviderType   `json:"provider"`
	Status         DeliveryStatus `json:"status"`
	ProviderMsgID  string         `json:"provider_msg_id"`
	AttemptedAt    time.Time      `json:"attempted_at"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
}
