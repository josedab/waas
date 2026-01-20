package metaevents

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service manages meta-event subscriptions
type Service struct {
	repo    Repository
	emitter *Emitter
}

// NewService creates a new meta-events service
func NewService(repo Repository, emitter *Emitter) *Service {
	return &Service{
		repo:    repo,
		emitter: emitter,
	}
}

// CreateSubscription creates a new subscription
func (s *Service) CreateSubscription(ctx context.Context, tenantID string, req *CreateSubscriptionRequest) (*Subscription, error) {
	// Validate event types
	for _, et := range req.EventTypes {
		if !isValidEventType(et) {
			return nil, fmt.Errorf("invalid event type: %s", et)
		}
	}

	// Generate webhook secret
	secret, err := generateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	retryPolicy := req.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}

	sub := &Subscription{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		URL:         req.URL,
		Secret:      secret,
		EventTypes:  req.EventTypes,
		Filters:     req.Filters,
		IsActive:    true,
		Headers:     req.Headers,
		RetryPolicy: retryPolicy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// GetSubscription retrieves a subscription by ID
func (s *Service) GetSubscription(ctx context.Context, tenantID, subID string) (*Subscription, error) {
	return s.repo.GetSubscription(ctx, tenantID, subID)
}

// ListSubscriptions lists subscriptions for a tenant
func (s *Service) ListSubscriptions(ctx context.Context, tenantID string, limit, offset int) ([]Subscription, int, error) {
	return s.repo.ListSubscriptions(ctx, tenantID, limit, offset)
}

// UpdateSubscription updates a subscription
func (s *Service) UpdateSubscription(ctx context.Context, tenantID, subID string, req *UpdateSubscriptionRequest) (*Subscription, error) {
	sub, err := s.repo.GetSubscription(ctx, tenantID, subID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}

	if req.Name != nil {
		sub.Name = *req.Name
	}
	if req.URL != nil {
		sub.URL = *req.URL
	}
	if len(req.EventTypes) > 0 {
		for _, et := range req.EventTypes {
			if !isValidEventType(et) {
				return nil, fmt.Errorf("invalid event type: %s", et)
			}
		}
		sub.EventTypes = req.EventTypes
	}
	if req.Filters != nil {
		sub.Filters = req.Filters
	}
	if req.Headers != nil {
		sub.Headers = req.Headers
	}
	if req.IsActive != nil {
		sub.IsActive = *req.IsActive
	}
	if req.RetryPolicy != nil {
		sub.RetryPolicy = req.RetryPolicy
	}

	sub.UpdatedAt = time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// DeleteSubscription deletes a subscription
func (s *Service) DeleteSubscription(ctx context.Context, tenantID, subID string) error {
	return s.repo.DeleteSubscription(ctx, tenantID, subID)
}

// RotateSecret generates a new secret for a subscription
func (s *Service) RotateSecret(ctx context.Context, tenantID, subID string) (*Subscription, string, error) {
	sub, err := s.repo.GetSubscription(ctx, tenantID, subID)
	if err != nil {
		return nil, "", err
	}
	if sub == nil {
		return nil, "", fmt.Errorf("subscription not found")
	}

	newSecret, err := generateSecret()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate secret: %w", err)
	}

	sub.Secret = newSecret
	sub.UpdatedAt = time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return nil, "", err
	}

	return sub, newSecret, nil
}

// GetSecret retrieves the secret for a subscription (used for display once)
func (s *Service) GetSecret(ctx context.Context, tenantID, subID string) (string, error) {
	sub, err := s.repo.GetSubscription(ctx, tenantID, subID)
	if err != nil {
		return "", err
	}
	if sub == nil {
		return "", fmt.Errorf("subscription not found")
	}
	return sub.Secret, nil
}

// ListEvents lists meta-events for a tenant
func (s *Service) ListEvents(ctx context.Context, tenantID string, eventType *EventType, limit, offset int) ([]MetaEvent, int, error) {
	return s.repo.ListEvents(ctx, tenantID, eventType, limit, offset)
}

// GetEvent retrieves an event by ID
func (s *Service) GetEvent(ctx context.Context, tenantID, eventID string) (*MetaEvent, error) {
	return s.repo.GetEvent(ctx, tenantID, eventID)
}

// ListDeliveries lists deliveries for a subscription
func (s *Service) ListDeliveries(ctx context.Context, tenantID, subID string, limit, offset int) ([]Delivery, int, error) {
	return s.repo.ListDeliveries(ctx, tenantID, subID, limit, offset)
}

// TestSubscription sends a test event to a subscription
func (s *Service) TestSubscription(ctx context.Context, tenantID, subID string) error {
	sub, err := s.repo.GetSubscription(ctx, tenantID, subID)
	if err != nil {
		return err
	}
	if sub == nil {
		return fmt.Errorf("subscription not found")
	}

	// Emit a test event
	return s.emitter.Emit(ctx, tenantID, "test.ping", "test", subID, map[string]interface{}{
		"message":         "This is a test event",
		"subscription_id": subID,
		"timestamp":       time.Now(),
	})
}

// GetAvailableEventTypes returns all supported event types
func (s *Service) GetAvailableEventTypes() []EventTypeInfo {
	return []EventTypeInfo{
		{Type: EventDeliveryAttempted, Category: "delivery", Description: "Webhook delivery was attempted"},
		{Type: EventDeliverySucceeded, Category: "delivery", Description: "Webhook was successfully delivered"},
		{Type: EventDeliveryFailed, Category: "delivery", Description: "Webhook delivery failed"},
		{Type: EventDeliveryRetrying, Category: "delivery", Description: "Webhook delivery is being retried"},
		{Type: EventDeliveryExhausted, Category: "delivery", Description: "All retry attempts exhausted"},
		{Type: EventEndpointCreated, Category: "endpoint", Description: "New endpoint was created"},
		{Type: EventEndpointUpdated, Category: "endpoint", Description: "Endpoint was updated"},
		{Type: EventEndpointDeleted, Category: "endpoint", Description: "Endpoint was deleted"},
		{Type: EventEndpointDisabled, Category: "endpoint", Description: "Endpoint was disabled"},
		{Type: EventEndpointEnabled, Category: "endpoint", Description: "Endpoint was enabled"},
		{Type: EventEndpointHealthy, Category: "health", Description: "Endpoint is healthy"},
		{Type: EventEndpointUnhealthy, Category: "health", Description: "Endpoint became unhealthy"},
		{Type: EventEndpointRecovered, Category: "health", Description: "Endpoint recovered from unhealthy state"},
		{Type: EventThresholdError, Category: "threshold", Description: "Error rate threshold exceeded"},
		{Type: EventThresholdLatency, Category: "threshold", Description: "Latency threshold exceeded"},
		{Type: EventThresholdVolume, Category: "threshold", Description: "Volume threshold exceeded"},
		{Type: EventAnomalyDetected, Category: "anomaly", Description: "Anomalous behavior detected"},
		{Type: EventSecurityViolation, Category: "security", Description: "Security violation detected"},
		{Type: EventAPIKeyRotated, Category: "security", Description: "API key was rotated"},
	}
}

// EventTypeInfo provides information about an event type
type EventTypeInfo struct {
	Type        EventType `json:"type"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
}

func isValidEventType(et EventType) bool {
	validTypes := []EventType{
		EventDeliveryAttempted, EventDeliverySucceeded, EventDeliveryFailed,
		EventDeliveryRetrying, EventDeliveryExhausted,
		EventEndpointCreated, EventEndpointUpdated, EventEndpointDeleted,
		EventEndpointDisabled, EventEndpointEnabled,
		EventEndpointHealthy, EventEndpointUnhealthy, EventEndpointRecovered,
		EventThresholdError, EventThresholdLatency, EventThresholdVolume,
		EventAnomalyDetected, EventSecurityViolation, EventAPIKeyRotated,
	}

	for _, vt := range validTypes {
		if vt == et {
			return true
		}
	}
	return false
}

func generateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(bytes), nil
}
