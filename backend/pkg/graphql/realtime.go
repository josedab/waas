package graphql

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TenantIsolationMode defines multi-tenant isolation levels
type TenantIsolationMode string

const (
	IsolationShared    TenantIsolationMode = "shared"
	IsolationDedicated TenantIsolationMode = "dedicated"
	IsolationFederated TenantIsolationMode = "federated"
)

// TenantGraphQLConfig holds per-tenant GraphQL configuration
type TenantGraphQLConfig struct {
	ID                      string              `json:"id"`
	TenantID                string              `json:"tenant_id"`
	IsolationMode           TenantIsolationMode `json:"isolation_mode"`
	MaxSubscriptions        int                 `json:"max_subscriptions"`
	MaxConnectionsPerClient int                 `json:"max_connections_per_client"`
	RateLimitPerSecond      int                 `json:"rate_limit_per_second"`
	AllowedEventTypes       []string            `json:"allowed_event_types"`
	CreatedAt               time.Time           `json:"created_at"`
	UpdatedAt               time.Time           `json:"updated_at"`
}

// SubscriptionInfo provides information about an active subscription
type SubscriptionInfo struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Query       string     `json:"query"`
	Active      bool       `json:"active"`
	EventCount  int64      `json:"event_count"`
	CreatedAt   time.Time  `json:"created_at"`
	LastEventAt *time.Time `json:"last_event_at,omitempty"`
}

// SubscriptionStats provides statistics about subscriptions
type SubscriptionStats struct {
	ActiveSubscriptions int     `json:"active_subscriptions"`
	TotalEvents         int64   `json:"total_events"`
	EventsPerMinute     float64 `json:"events_per_minute"`
	ActiveConnections   int     `json:"active_connections"`
	AvgLatencyMs        float64 `json:"avg_latency_ms"`
}

// DeliveryEvent represents a real-time delivery event
type DeliveryEvent struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	EndpointID string                 `json:"endpoint_id"`
	EventType  string                 `json:"event_type"`
	DeliveryID string                 `json:"delivery_id,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// EndpointHealthEvent represents an endpoint health change event
type EndpointHealthEvent struct {
	EndpointID     string    `json:"endpoint_id"`
	URL            string    `json:"url"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	CurrentStatus  string    `json:"current_status"`
	HealthScore    float64   `json:"health_score"`
	LatencyMs      float64   `json:"latency_ms"`
	ErrorRate      float64   `json:"error_rate"`
	Timestamp      time.Time `json:"timestamp"`
}

// RealTimeSubscriptionEngine manages real-time GraphQL subscriptions
type RealTimeSubscriptionEngine struct {
	subscribers map[string]map[string]chan interface{} // tenantID -> subID -> channel
	configs     map[string]*TenantGraphQLConfig
	stats       map[string]*SubscriptionStats
	repo        RealTimeRepository
	mu          sync.RWMutex
}

// RealTimeRepository defines storage for real-time subscription data
type RealTimeRepository interface {
	SaveTenantConfig(ctx context.Context, config *TenantGraphQLConfig) error
	GetTenantConfig(ctx context.Context, tenantID string) (*TenantGraphQLConfig, error)
	UpdateTenantConfig(ctx context.Context, config *TenantGraphQLConfig) error
	SaveSubscription(ctx context.Context, info *SubscriptionInfo, tenantID string) error
	ListSubscriptions(ctx context.Context, tenantID string) ([]SubscriptionInfo, error)
	GetSubscription(ctx context.Context, tenantID, subID string) (*SubscriptionInfo, error)
	DeleteSubscription(ctx context.Context, tenantID, subID string) error
	IncrementEventCount(ctx context.Context, subID string) error
	GetStats(ctx context.Context, tenantID string) (*SubscriptionStats, error)
}

// NewRealTimeSubscriptionEngine creates a new subscription engine
func NewRealTimeSubscriptionEngine(repo RealTimeRepository) *RealTimeSubscriptionEngine {
	return &RealTimeSubscriptionEngine{
		subscribers: make(map[string]map[string]chan interface{}),
		configs:     make(map[string]*TenantGraphQLConfig),
		stats:       make(map[string]*SubscriptionStats),
		repo:        repo,
	}
}

// ConfigureTenant sets up tenant-specific GraphQL configuration
func (e *RealTimeSubscriptionEngine) ConfigureTenant(ctx context.Context, tenantID string, config *TenantGraphQLConfig) (*TenantGraphQLConfig, error) {
	if config.MaxSubscriptions == 0 {
		config.MaxSubscriptions = 100
	}
	if config.MaxConnectionsPerClient == 0 {
		config.MaxConnectionsPerClient = 5
	}
	if config.RateLimitPerSecond == 0 {
		config.RateLimitPerSecond = 100
	}
	if config.IsolationMode == "" {
		config.IsolationMode = IsolationShared
	}

	config.ID = uuid.New().String()
	config.TenantID = tenantID
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	if err := e.repo.SaveTenantConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to save tenant config: %w", err)
	}

	e.mu.Lock()
	e.configs[tenantID] = config
	e.mu.Unlock()

	return config, nil
}

// Subscribe creates a new subscription
func (e *RealTimeSubscriptionEngine) Subscribe(ctx context.Context, tenantID, subType, query string) (*SubscriptionInfo, <-chan interface{}, error) {
	// Check tenant limits
	e.mu.RLock()
	config, hasConfig := e.configs[tenantID]
	tenantSubs := e.subscribers[tenantID]
	e.mu.RUnlock()

	if hasConfig && tenantSubs != nil && len(tenantSubs) >= config.MaxSubscriptions {
		return nil, nil, fmt.Errorf("subscription limit reached: %d", config.MaxSubscriptions)
	}

	subID := uuid.New().String()
	now := time.Now()
	info := &SubscriptionInfo{
		ID:        subID,
		Type:      subType,
		Query:     query,
		Active:    true,
		CreatedAt: now,
	}

	ch := make(chan interface{}, 100)

	e.mu.Lock()
	if e.subscribers[tenantID] == nil {
		e.subscribers[tenantID] = make(map[string]chan interface{})
	}
	e.subscribers[tenantID][subID] = ch
	e.mu.Unlock()

	_ = e.repo.SaveSubscription(ctx, info, tenantID)

	return info, ch, nil
}

// Unsubscribe removes a subscription
func (e *RealTimeSubscriptionEngine) Unsubscribe(ctx context.Context, tenantID, subID string) error {
	e.mu.Lock()
	if subs, ok := e.subscribers[tenantID]; ok {
		if ch, ok := subs[subID]; ok {
			close(ch)
			delete(subs, subID)
		}
	}
	e.mu.Unlock()

	return e.repo.DeleteSubscription(ctx, tenantID, subID)
}

// Publish sends an event to all matching subscribers
func (e *RealTimeSubscriptionEngine) Publish(ctx context.Context, tenantID string, event interface{}) {
	e.mu.RLock()
	subs, ok := e.subscribers[tenantID]
	e.mu.RUnlock()

	if !ok {
		return
	}

	for subID, ch := range subs {
		select {
		case ch <- event:
			_ = e.repo.IncrementEventCount(ctx, subID)
		default:
			// Channel full, skip
		}
	}
}

// ListSubscriptions lists active subscriptions for a tenant
func (e *RealTimeSubscriptionEngine) ListSubscriptions(ctx context.Context, tenantID string) ([]SubscriptionInfo, error) {
	return e.repo.ListSubscriptions(ctx, tenantID)
}

// GetStats returns subscription statistics
func (e *RealTimeSubscriptionEngine) GetStats(ctx context.Context, tenantID string) (*SubscriptionStats, error) {
	stats, err := e.repo.GetStats(ctx, tenantID)
	if err != nil {
		// Fallback to in-memory counts
		e.mu.RLock()
		count := len(e.subscribers[tenantID])
		e.mu.RUnlock()
		return &SubscriptionStats{
			ActiveSubscriptions: count,
		}, nil
	}
	return stats, nil
}

// GetTenantConfig retrieves tenant configuration
func (e *RealTimeSubscriptionEngine) GetTenantConfig(ctx context.Context, tenantID string) (*TenantGraphQLConfig, error) {
	e.mu.RLock()
	config, ok := e.configs[tenantID]
	e.mu.RUnlock()

	if ok {
		return config, nil
	}

	return e.repo.GetTenantConfig(ctx, tenantID)
}
