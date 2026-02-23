package mobileinspector

import (
	"fmt"
	"time"
)

// Repository defines the data access interface for mobile inspector.
type Repository interface {
	CreateSession(s *MobileSession) error
	GetSession(id string) (*MobileSession, error)
	GetSessionByToken(token string) (*MobileSession, error)
	DeleteSession(id string) error

	CreatePushRegistration(r *PushRegistration) error
	GetPushRegistration(deviceID string) (*PushRegistration, error)
	UpdatePushRegistration(r *PushRegistration) error

	GetEventFeed(tenantID string, since time.Time, limit int) ([]EventFeedItem, error)
	GetEndpointOverviews(tenantID string) ([]EndpointOverview, error)

	SaveAlertConfig(cfg *AlertConfig) error
	GetAlertConfig(userID string) (*AlertConfig, error)

	CreateAlert(alert *AlertNotification) error
	ListAlerts(userID string, limit int) ([]*AlertNotification, error)
	AcknowledgeAlert(id string) error
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	sessions map[string]*MobileSession
	push     map[string]*PushRegistration
	events   []EventFeedItem
	alerts   []*AlertNotification
	configs  map[string]*AlertConfig
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		sessions: make(map[string]*MobileSession),
		push:     make(map[string]*PushRegistration),
		events:   make([]EventFeedItem, 0),
		alerts:   make([]*AlertNotification, 0),
		configs:  make(map[string]*AlertConfig),
	}
}

func (r *MemoryRepository) CreateSession(s *MobileSession) error {
	r.sessions[s.ID] = s
	return nil
}

func (r *MemoryRepository) GetSession(id string) (*MobileSession, error) {
	if s, ok := r.sessions[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found: %s", id)
}

func (r *MemoryRepository) GetSessionByToken(token string) (*MobileSession, error) {
	for _, s := range r.sessions {
		if s.Token == token {
			return s, nil
		}
	}
	return nil, fmt.Errorf("session not found for token")
}

func (r *MemoryRepository) DeleteSession(id string) error {
	delete(r.sessions, id)
	return nil
}

func (r *MemoryRepository) CreatePushRegistration(reg *PushRegistration) error {
	r.push[reg.DeviceID] = reg
	return nil
}

func (r *MemoryRepository) GetPushRegistration(deviceID string) (*PushRegistration, error) {
	if p, ok := r.push[deviceID]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("push registration not found: %s", deviceID)
}

func (r *MemoryRepository) UpdatePushRegistration(reg *PushRegistration) error {
	r.push[reg.DeviceID] = reg
	return nil
}

func (r *MemoryRepository) GetEventFeed(tenantID string, since time.Time, limit int) ([]EventFeedItem, error) {
	var result []EventFeedItem
	for i := len(r.events) - 1; i >= 0 && len(result) < limit; i-- {
		if r.events[i].Timestamp.After(since) {
			result = append(result, r.events[i])
		}
	}
	return result, nil
}

func (r *MemoryRepository) GetEndpointOverviews(tenantID string) ([]EndpointOverview, error) {
	return []EndpointOverview{
		{ID: "ep-1", Name: "Orders API", HealthScore: 98.5, SuccessRate: 99.2, TotalDeliveries: 5000, AvgLatencyMs: 120},
		{ID: "ep-2", Name: "Payments Webhook", HealthScore: 95.0, SuccessRate: 97.8, TotalDeliveries: 3200, AvgLatencyMs: 180},
	}, nil
}

func (r *MemoryRepository) SaveAlertConfig(cfg *AlertConfig) error {
	r.configs[cfg.UserID] = cfg
	return nil
}

func (r *MemoryRepository) GetAlertConfig(userID string) (*AlertConfig, error) {
	if c, ok := r.configs[userID]; ok {
		return c, nil
	}
	return &AlertConfig{
		UserID:             userID,
		FailureThreshold:   5,
		LatencyThresholdMs: 5000,
		SuccessRateMin:     95.0,
		NotifyOnFailure:    true,
		NotifyOnRecovery:   true,
	}, nil
}

func (r *MemoryRepository) CreateAlert(alert *AlertNotification) error {
	r.alerts = append(r.alerts, alert)
	return nil
}

func (r *MemoryRepository) ListAlerts(userID string, limit int) ([]*AlertNotification, error) {
	var result []*AlertNotification
	for i := len(r.alerts) - 1; i >= 0 && len(result) < limit; i-- {
		if r.alerts[i].UserID == userID {
			result = append(result, r.alerts[i])
		}
	}
	return result, nil
}

func (r *MemoryRepository) AcknowledgeAlert(id string) error {
	for _, a := range r.alerts {
		if a.ID == id {
			a.Acknowledged = true
			return nil
		}
	}
	return fmt.Errorf("alert not found: %s", id)
}
