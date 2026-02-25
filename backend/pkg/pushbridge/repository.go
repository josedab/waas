package pushbridge

import (
	"context"
	"database/sql"
	"time"
)

// Repository defines push bridge data access
type Repository interface {
	// Devices
	SaveDevice(ctx context.Context, device *PushDevice) error
	GetDevice(ctx context.Context, tenantID, deviceID string) (*PushDevice, error)
	GetDeviceByToken(ctx context.Context, tenantID, pushToken string) (*PushDevice, error)
	ListDevices(ctx context.Context, tenantID string, filter *DeviceFilter) ([]PushDevice, error)
	DeleteDevice(ctx context.Context, tenantID, deviceID string) error
	UpdateDeviceStatus(ctx context.Context, deviceID string, status DeviceStatus) error
	
	// Mappings
	SaveMapping(ctx context.Context, mapping *PushMapping) error
	GetMapping(ctx context.Context, tenantID, mappingID string) (*PushMapping, error)
	ListMappings(ctx context.Context, tenantID string) ([]PushMapping, error)
	GetMappingByWebhook(ctx context.Context, tenantID, webhookID string) ([]PushMapping, error)
	DeleteMapping(ctx context.Context, tenantID, mappingID string) error
	
	// Notifications
	SaveNotification(ctx context.Context, notif *PushNotification) error
	GetNotification(ctx context.Context, notifID string) (*PushNotification, error)
	ListNotifications(ctx context.Context, tenantID string, filter *NotificationFilter) ([]PushNotification, error)
	UpdateNotificationStatus(ctx context.Context, notifID string, status DeliveryStatus, response *ProviderResponse) error
	
	// Offline Queue
	QueueNotification(ctx context.Context, queue *OfflineQueue) error
	GetQueuedNotifications(ctx context.Context, deviceID string, limit int) ([]OfflineQueue, error)
	DeleteQueuedNotification(ctx context.Context, queueID string) error
	CleanExpiredQueue(ctx context.Context) error
	
	// Providers
	SaveProvider(ctx context.Context, provider *PushProviderConfig) error
	GetProvider(ctx context.Context, tenantID string, providerType ProviderType) (*PushProviderConfig, error)
	ListProviders(ctx context.Context, tenantID string) ([]PushProviderConfig, error)
	DeleteProvider(ctx context.Context, tenantID string, providerType ProviderType) error
	
	// Credentials
	SaveCredentials(ctx context.Context, creds *ProviderCredentials) error
	GetCredentials(ctx context.Context, tenantID, credID string) (*ProviderCredentials, error)
	ListCredentials(ctx context.Context, tenantID string, provider *Platform) ([]*ProviderCredentials, error)
	UpdateCredentials(ctx context.Context, creds *ProviderCredentials) error
	DeleteCredentials(ctx context.Context, tenantID, credID string) error
	
	// Stats
	GetStats(ctx context.Context, tenantID string, from, to time.Time) (*PushStats, error)
	IncrementStats(ctx context.Context, tenantID string, platform Platform, status DeliveryStatus) error
	
	// Segments
	SaveSegment(ctx context.Context, segment *DeviceSegment) error
	GetSegment(ctx context.Context, tenantID, segmentID string) (*DeviceSegment, error)
	ListSegments(ctx context.Context, tenantID string) ([]DeviceSegment, error)
	DeleteSegment(ctx context.Context, tenantID, segmentID string) error
	GetDevicesInSegment(ctx context.Context, tenantID, segmentID string) ([]PushDevice, error)
}

// DeviceFilter defines device list filters
type DeviceFilter struct {
	Platform  *Platform
	Status    *DeviceStatus
	UserID    *string
	Tags      []string
	Limit     int
	Offset    int
}

// NotificationFilter defines notification list filters
type NotificationFilter struct {
	DeviceID  *string
	MappingID *string
	Status    *DeliveryStatus
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}
