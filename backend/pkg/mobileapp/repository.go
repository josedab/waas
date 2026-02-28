package mobileapp

import (
	"fmt"
	"time"
)

// Repository defines the data access interface for the mobile app.
type Repository interface {
	CreateDevice(d *DeviceRegistration) error
	GetDevice(id string) (*DeviceRegistration, error)
	ListDevices(tenantID string) ([]*DeviceRegistration, error)
	UpdateDevice(d *DeviceRegistration) error
	DeleteDevice(id string) error

	CreateNotification(n *PushNotification) error
	ListNotifications(tenantID string, limit int) ([]*PushNotification, error)
	MarkNotificationRead(id string) error

	GetDashboard(tenantID string) (*MobileDashboard, error)
	GetLivePayloads(tenantID string, limit int) ([]LivePayloadEvent, error)
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	devices       map[string]*DeviceRegistration
	notifications []*PushNotification
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		devices:       make(map[string]*DeviceRegistration),
		notifications: make([]*PushNotification, 0),
	}
}

func (r *MemoryRepository) CreateDevice(d *DeviceRegistration) error {
	r.devices[d.ID] = d
	return nil
}

func (r *MemoryRepository) GetDevice(id string) (*DeviceRegistration, error) {
	if d, ok := r.devices[id]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("device not found: %s", id)
}

func (r *MemoryRepository) ListDevices(tenantID string) ([]*DeviceRegistration, error) {
	var result []*DeviceRegistration
	for _, d := range r.devices {
		if d.TenantID == tenantID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *MemoryRepository) UpdateDevice(d *DeviceRegistration) error {
	if _, ok := r.devices[d.ID]; !ok {
		return fmt.Errorf("device not found: %s", d.ID)
	}
	r.devices[d.ID] = d
	return nil
}

func (r *MemoryRepository) DeleteDevice(id string) error {
	delete(r.devices, id)
	return nil
}

func (r *MemoryRepository) CreateNotification(n *PushNotification) error {
	r.notifications = append(r.notifications, n)
	return nil
}

func (r *MemoryRepository) ListNotifications(tenantID string, limit int) ([]*PushNotification, error) {
	var result []*PushNotification
	for i := len(r.notifications) - 1; i >= 0 && len(result) < limit; i-- {
		if r.notifications[i].TenantID == tenantID {
			result = append(result, r.notifications[i])
		}
	}
	return result, nil
}

func (r *MemoryRepository) MarkNotificationRead(id string) error {
	for _, n := range r.notifications {
		if n.ID == id {
			now := time.Now()
			n.ReadAt = &now
			return nil
		}
	}
	return fmt.Errorf("notification not found: %s", id)
}

func (r *MemoryRepository) GetDashboard(tenantID string) (*MobileDashboard, error) {
	return &MobileDashboard{
		ActiveEndpoints:  3,
		RecentDeliveries: 150,
		FailureRate:      2.5,
		AlertCount:       1,
		TopEndpoints: []EndpointSummary{
			{ID: "ep-1", URL: "https://api.example.com/webhooks", Status: "active", SuccessRate: 99.2},
			{ID: "ep-2", URL: "https://payments.example.com/hooks", Status: "active", SuccessRate: 97.8},
		},
	}, nil
}

func (r *MemoryRepository) GetLivePayloads(tenantID string, limit int) ([]LivePayloadEvent, error) {
	return []LivePayloadEvent{
		{
			DeliveryID: "del-1",
			EndpointID: "ep-1",
			EventType:  "order.created",
			StatusCode: 200,
			LatencyMs:  120,
			Payload:    `{"event":"order.created","id":"ord-123"}`,
			Timestamp:  time.Now().Add(-1 * time.Minute),
		},
	}, nil
}
