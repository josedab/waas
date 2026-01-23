package fanout

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"webhook-platform/pkg/queue"
	"webhook-platform/pkg/utils"
)

// ServiceConfig holds configuration for the fanout service
type ServiceConfig struct {
	MaxSubscribersPerTopic int
	MaxFanOutConcurrency   int
	DefaultRetentionDays   int
}

// DefaultServiceConfig returns sensible defaults
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxSubscribersPerTopic: 100,
		MaxFanOutConcurrency:   10,
		DefaultRetentionDays:   30,
	}
}

// Service provides fan-out business logic
type Service struct {
	repo      Repository
	publisher queue.PublisherInterface
	filter    *FilterEngine
	logger    *utils.Logger
	config    ServiceConfig
}

// NewService creates a new fanout service
func NewService(repo Repository) *Service {
	return &Service{
		repo:   repo,
		filter: NewFilterEngine(),
		logger: utils.NewLogger("fanout-service"),
		config: DefaultServiceConfig(),
	}
}

// NewServiceWithDeps creates a new fanout service with all dependencies
func NewServiceWithDeps(repo Repository, publisher queue.PublisherInterface, logger *utils.Logger, config ServiceConfig) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		filter:    NewFilterEngine(),
		logger:    logger,
		config:    config,
	}
}

// CreateTopic creates a new topic for the tenant
func (s *Service) CreateTopic(ctx context.Context, tenantID uuid.UUID, req *CreateTopicRequest) (*Topic, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("topic name is required")
	}

	now := time.Now()
	maxSubs := req.MaxSubscribers
	if maxSubs <= 0 {
		maxSubs = s.config.MaxSubscribersPerTopic
	}
	retDays := req.RetentionDays
	if retDays <= 0 {
		retDays = s.config.DefaultRetentionDays
	}

	topic := &Topic{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		Status:         TopicStatusActive,
		MaxSubscribers: maxSubs,
		RetentionDays:  retDays,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateTopic(ctx, topic); err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	return topic, nil
}

// GetTopic retrieves a topic by ID
func (s *Service) GetTopic(ctx context.Context, tenantID, topicID uuid.UUID) (*Topic, error) {
	return s.repo.GetTopic(ctx, tenantID, topicID)
}

// ListTopics lists all topics for a tenant
func (s *Service) ListTopics(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Topic, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListTopics(ctx, tenantID, limit, offset)
}

// UpdateTopic updates an existing topic
func (s *Service) UpdateTopic(ctx context.Context, tenantID, topicID uuid.UUID, req *UpdateTopicRequest) (*Topic, error) {
	topic, err := s.repo.GetTopic(ctx, tenantID, topicID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		topic.Name = req.Name
	}
	if req.Description != "" {
		topic.Description = req.Description
	}
	if req.Status != "" {
		if req.Status != TopicStatusActive && req.Status != TopicStatusPaused && req.Status != TopicStatusArchived {
			return nil, fmt.Errorf("invalid topic status: %s", req.Status)
		}
		topic.Status = req.Status
	}
	if req.MaxSubscribers > 0 {
		topic.MaxSubscribers = req.MaxSubscribers
	}
	if req.RetentionDays > 0 {
		topic.RetentionDays = req.RetentionDays
	}
	topic.UpdatedAt = time.Now()

	if err := s.repo.UpdateTopic(ctx, topic); err != nil {
		return nil, fmt.Errorf("failed to update topic: %w", err)
	}

	return topic, nil
}

// DeleteTopic deletes a topic
func (s *Service) DeleteTopic(ctx context.Context, tenantID, topicID uuid.UUID) error {
	return s.repo.DeleteTopic(ctx, tenantID, topicID)
}

// Subscribe subscribes an endpoint to a topic with an optional filter expression
func (s *Service) Subscribe(ctx context.Context, tenantID, topicID, endpointID uuid.UUID, filterExpr string) (*Subscription, error) {
	// Validate the topic exists
	topic, err := s.repo.GetTopic(ctx, tenantID, topicID)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	// Validate filter expression if provided
	if filterExpr != "" {
		if err := s.filter.Validate(filterExpr); err != nil {
			return nil, fmt.Errorf("invalid filter expression: %w", err)
		}
	}

	// Check subscriber limit
	count, err := s.repo.CountSubscriptions(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("failed to count subscriptions: %w", err)
	}
	if count >= topic.MaxSubscribers {
		return nil, fmt.Errorf("topic has reached maximum subscriber limit of %d", topic.MaxSubscribers)
	}

	sub := &Subscription{
		ID:         uuid.New(),
		TopicID:    topicID,
		TenantID:   tenantID,
		EndpointID: endpointID,
		FilterExpr: filterExpr,
		Active:     true,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return sub, nil
}

// Unsubscribe removes a subscription
func (s *Service) Unsubscribe(ctx context.Context, subscriptionID uuid.UUID) error {
	return s.repo.DeleteSubscription(ctx, subscriptionID)
}

// GetTopicSubscribers lists subscriptions for a topic
func (s *Service) GetTopicSubscribers(ctx context.Context, topicID uuid.UUID, limit, offset int) ([]Subscription, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListSubscriptions(ctx, topicID, limit, offset)
}

// GetTopicEvents lists events for a topic
func (s *Service) GetTopicEvents(ctx context.Context, topicID uuid.UUID, limit, offset int) ([]TopicEvent, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListEvents(ctx, topicID, limit, offset)
}

// Publish publishes an event to a topic and triggers fan-out to all matching subscribers
func (s *Service) Publish(ctx context.Context, tenantID uuid.UUID, topicName, eventType string, payload, metadata json.RawMessage) (*FanOutResult, error) {
	topic, err := s.repo.GetTopicByName(ctx, tenantID, topicName)
	if err != nil {
		return nil, fmt.Errorf("topic %q not found: %w", topicName, err)
	}

	if topic.Status != TopicStatusActive {
		return nil, fmt.Errorf("topic %q is not active (status: %s)", topicName, topic.Status)
	}

	event := &TopicEvent{
		ID:          uuid.New(),
		TopicID:     topic.ID,
		TenantID:    tenantID,
		EventType:   eventType,
		Payload:     payload,
		Metadata:    metadata,
		FanOutCount: 0,
		Status:      EventStatusPublished,
		PublishedAt: time.Now(),
	}

	if err := s.repo.CreateEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	subs, err := s.repo.GetActiveSubscriptions(ctx, topic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	result := s.FanOut(ctx, event, subs)

	// Update event status based on results
	status := EventStatusFanOutComplete
	if result.Failed > 0 {
		status = EventStatusPartialFailure
	}
	_ = s.repo.UpdateEventStatus(ctx, event.ID, status, result.TotalTargets)

	return result, nil
}

// FanOut executes fan-out delivery to all matching subscribers
func (s *Service) FanOut(ctx context.Context, event *TopicEvent, subscriptions []Subscription) *FanOutResult {
	result := &FanOutResult{
		EventID:      event.ID,
		TotalTargets: len(subscriptions),
		Results:      make([]DeliveryTarget, 0, len(subscriptions)),
	}

	for _, sub := range subscriptions {
		target := DeliveryTarget{
			SubscriptionID: sub.ID,
			EndpointID:     sub.EndpointID,
		}

		// Evaluate filter
		if sub.FilterExpr != "" {
			match, err := s.filter.Evaluate(sub.FilterExpr, event.Payload)
			if err != nil {
				target.Status = TargetStatusFailed
				target.Error = fmt.Sprintf("filter evaluation error: %v", err)
				result.Failed++
				result.Results = append(result.Results, target)
				continue
			}
			if !match {
				target.Status = TargetStatusFiltered
				result.Filtered++
				result.Results = append(result.Results, target)
				continue
			}
		}

		// Publish to delivery queue
		if s.publisher != nil {
			msg := &queue.DeliveryMessage{
				DeliveryID:    uuid.New(),
				EndpointID:    sub.EndpointID,
				TenantID:      event.TenantID,
				Payload:       event.Payload,
				AttemptNumber: 1,
				ScheduledAt:   time.Now(),
				MaxAttempts:   3,
			}

			if err := s.publisher.PublishDelivery(ctx, msg); err != nil {
				target.Status = TargetStatusFailed
				target.Error = fmt.Sprintf("queue publish error: %v", err)
				result.Failed++
				result.Results = append(result.Results, target)
				continue
			}
			target.Status = TargetStatusQueued
		} else {
			target.Status = TargetStatusDelivered
		}

		result.Delivered++
		result.Results = append(result.Results, target)
	}

	return result
}
