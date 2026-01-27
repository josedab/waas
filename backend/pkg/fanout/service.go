package fanout

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
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

// FanOutDelivery takes an event and delivers to all matching subscriptions in parallel.
// Each delivery has its own retry policy and error handling.
func (s *Service) FanOutDelivery(ctx context.Context, tenantID, topicID uuid.UUID, payload json.RawMessage, headers map[string]string) (*FanOutDeliveryResult, error) {
	topic, err := s.repo.GetTopic(ctx, tenantID, topicID)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %w", err)
	}
	if topic.Status != TopicStatusActive {
		return nil, fmt.Errorf("topic %q is not active (status: %s)", topic.Name, topic.Status)
	}

	subs, err := s.repo.GetActiveSubscriptions(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	eventID := uuid.New()
	result := &FanOutDeliveryResult{
		EventID:       eventID,
		TopicID:       topicID,
		TotalTargets:  len(subs),
		TargetResults: make([]TargetDeliveryResult, len(subs)),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, sub := range subs {
		wg.Add(1)
		go func(idx int, sub Subscription) {
			defer wg.Done()
			start := time.Now()
			tr := TargetDeliveryResult{
				SubscriptionID: sub.ID,
				EndpointURL:    sub.EndpointID.String(),
			}

			// Evaluate filter
			if sub.FilterExpr != "" {
				match, err := s.filter.Evaluate(sub.FilterExpr, payload)
				if err != nil {
					tr.Status = DeliveryStatusFailed
					tr.ErrorMessage = fmt.Sprintf("filter evaluation error: %v", err)
					tr.DurationMs = int(time.Since(start).Milliseconds())
					mu.Lock()
					result.TargetResults[idx] = tr
					result.Failed++
					mu.Unlock()
					return
				}
				if !match {
					tr.Status = DeliveryStatusFiltered
					tr.DurationMs = int(time.Since(start).Milliseconds())
					mu.Lock()
					result.TargetResults[idx] = tr
					mu.Unlock()
					return
				}
			}

			// Deliver via publisher or mark as delivered
			if s.publisher != nil {
				msg := &queue.DeliveryMessage{
					DeliveryID:    uuid.New(),
					EndpointID:    sub.EndpointID,
					TenantID:      tenantID,
					Payload:       payload,
					AttemptNumber: 1,
					ScheduledAt:   time.Now(),
					MaxAttempts:   3,
				}
				if err := s.publisher.PublishDelivery(ctx, msg); err != nil {
					tr.Status = DeliveryStatusFailed
					tr.ErrorMessage = fmt.Sprintf("delivery error: %v", err)
					tr.DurationMs = int(time.Since(start).Milliseconds())
					mu.Lock()
					result.TargetResults[idx] = tr
					result.Failed++
					mu.Unlock()
					return
				}
			}

			tr.Status = DeliveryStatusDelivered
			tr.DurationMs = int(time.Since(start).Milliseconds())
			mu.Lock()
			result.TargetResults[idx] = tr
			result.Succeeded++
			mu.Unlock()
		}(i, sub)
	}

	wg.Wait()
	return result, nil
}

// CreateRoutingRule creates a new routing rule for a topic
func (s *Service) CreateRoutingRule(ctx context.Context, tenantID, topicID uuid.UUID, req *CreateRuleRequest) (*RoutingRule, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("rule name is required")
	}
	if len(req.Conditions) == 0 {
		return nil, fmt.Errorf("at least one condition is required")
	}
	if len(req.Actions) == 0 {
		return nil, fmt.Errorf("at least one action is required")
	}

	now := time.Now()
	rule := &RoutingRule{
		ID:          uuid.New(),
		TenantID:    tenantID,
		TopicID:     topicID,
		Name:        req.Name,
		Description: req.Description,
		Version:     1,
		Conditions:  req.Conditions,
		Actions:     req.Actions,
		Priority:    req.Priority,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateRoutingRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create routing rule: %w", err)
	}

	// Save initial version
	ver := &RuleVersion{
		ID:         uuid.New(),
		RuleID:     rule.ID,
		Version:    1,
		Conditions: rule.Conditions,
		Actions:    rule.Actions,
		CreatedAt:  now,
	}
	if err := s.repo.CreateRuleVersion(ctx, ver); err != nil {
		return nil, fmt.Errorf("failed to save rule version: %w", err)
	}

	return rule, nil
}

// ListRoutingRules lists routing rules for a topic
func (s *Service) ListRoutingRules(ctx context.Context, tenantID, topicID uuid.UUID) ([]RoutingRule, error) {
	return s.repo.ListRoutingRules(ctx, tenantID, topicID)
}

// GetRoutingRule gets a routing rule by ID
func (s *Service) GetRoutingRule(ctx context.Context, tenantID, ruleID uuid.UUID) (*RoutingRule, error) {
	return s.repo.GetRoutingRule(ctx, tenantID, ruleID)
}

// UpdateRoutingRule updates a routing rule, creating a new version
func (s *Service) UpdateRoutingRule(ctx context.Context, tenantID, ruleID uuid.UUID, req *CreateRuleRequest) (*RoutingRule, error) {
	rule, err := s.repo.GetRoutingRule(ctx, tenantID, ruleID)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	now := time.Now()

	// Save previous version
	ver := &RuleVersion{
		ID:         uuid.New(),
		RuleID:     rule.ID,
		Version:    rule.Version,
		Conditions: rule.Conditions,
		Actions:    rule.Actions,
		CreatedAt:  now,
	}
	if err := s.repo.CreateRuleVersion(ctx, ver); err != nil {
		return nil, fmt.Errorf("failed to save rule version: %w", err)
	}

	// Update the rule
	rule.Version++
	rule.Name = req.Name
	rule.Description = req.Description
	rule.Conditions = req.Conditions
	rule.Actions = req.Actions
	rule.Priority = req.Priority
	rule.UpdatedAt = now

	if err := s.repo.UpdateRoutingRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update routing rule: %w", err)
	}

	return rule, nil
}

// DeleteRoutingRule deletes a routing rule
func (s *Service) DeleteRoutingRule(ctx context.Context, tenantID, ruleID uuid.UUID) error {
	return s.repo.DeleteRoutingRule(ctx, tenantID, ruleID)
}

// TestRule evaluates a rule against a sample payload and returns detailed results
func (s *Service) TestRule(ctx context.Context, tenantID, ruleID uuid.UUID, req *RuleTestRequest) (*RuleTestResult, error) {
	rule, err := s.repo.GetRoutingRule(ctx, tenantID, ruleID)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}

	result := &RuleTestResult{
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		Matched:    true,
		Conditions: make([]ConditionResult, 0, len(rule.Conditions)),
	}

	for _, cond := range rule.Conditions {
		matched, actual := s.evaluateCondition(cond, payload, req.Headers, req.EventType)
		cr := ConditionResult{
			Condition:   cond,
			Matched:     matched,
			ActualValue: actual,
		}
		result.Conditions = append(result.Conditions, cr)
		if !matched {
			result.Matched = false
		}
	}

	if result.Matched {
		result.Actions = rule.Actions
	}

	return result, nil
}

// RollbackRule rolls back a routing rule to a specific version
func (s *Service) RollbackRule(ctx context.Context, tenantID, ruleID uuid.UUID, version int) (*RoutingRule, error) {
	rule, err := s.repo.GetRoutingRule(ctx, tenantID, ruleID)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	ver, err := s.repo.GetRuleVersion(ctx, ruleID, version)
	if err != nil {
		return nil, fmt.Errorf("version %d not found: %w", version, err)
	}

	now := time.Now()

	// Save current state as a version before rollback
	currentVer := &RuleVersion{
		ID:         uuid.New(),
		RuleID:     rule.ID,
		Version:    rule.Version,
		Conditions: rule.Conditions,
		Actions:    rule.Actions,
		CreatedAt:  now,
	}
	if err := s.repo.CreateRuleVersion(ctx, currentVer); err != nil {
		return nil, fmt.Errorf("failed to save current version: %w", err)
	}

	// Apply rollback
	rule.Version++
	rule.Conditions = ver.Conditions
	rule.Actions = ver.Actions
	rule.UpdatedAt = now

	if err := s.repo.UpdateRoutingRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to rollback rule: %w", err)
	}

	return rule, nil
}

// GetRuleVersions returns version history for a rule
func (s *Service) GetRuleVersions(ctx context.Context, ruleID uuid.UUID) ([]RuleVersion, error) {
	return s.repo.GetRuleVersions(ctx, ruleID)
}

// evaluateCondition evaluates a single rule condition against event data
func (s *Service) evaluateCondition(condition RuleCondition, payload map[string]interface{}, headers map[string]string, eventType string) (bool, string) {
	switch condition.Type {
	case "jsonpath":
		return s.evaluateJSONPath(condition, payload)
	case "header":
		return s.evaluateHeader(condition, headers)
	case "regex":
		return s.evaluateRegex(condition, payload)
	case "event_type":
		return s.evaluateEventType(condition, eventType)
	default:
		return false, ""
	}
}

func (s *Service) evaluateJSONPath(condition RuleCondition, payload map[string]interface{}) (bool, string) {
	value, err := s.resolvePayloadPath(condition.Expression, payload)
	if err != nil {
		if condition.Operator == "exists" {
			return condition.Value == "false", ""
		}
		return false, ""
	}

	actual := fmt.Sprintf("%v", value)

	switch condition.Operator {
	case "equals":
		return actual == condition.Value, actual
	case "contains":
		return strings.Contains(actual, condition.Value), actual
	case "gt":
		return s.numericCompare(value, condition.Value, func(a, b float64) bool { return a > b }), actual
	case "lt":
		return s.numericCompare(value, condition.Value, func(a, b float64) bool { return a < b }), actual
	case "exists":
		return condition.Value != "false", actual
	case "matches":
		re, err := regexp.Compile(condition.Value)
		if err != nil {
			return false, actual
		}
		return re.MatchString(actual), actual
	default:
		return false, actual
	}
}

func (s *Service) evaluateHeader(condition RuleCondition, headers map[string]string) (bool, string) {
	actual, exists := headers[condition.Expression]
	if !exists {
		if condition.Operator == "exists" {
			return condition.Value == "false", ""
		}
		return false, ""
	}

	switch condition.Operator {
	case "equals":
		return actual == condition.Value, actual
	case "contains":
		return strings.Contains(actual, condition.Value), actual
	case "exists":
		return condition.Value != "false", actual
	case "matches":
		re, err := regexp.Compile(condition.Value)
		if err != nil {
			return false, actual
		}
		return re.MatchString(actual), actual
	default:
		return false, actual
	}
}

func (s *Service) evaluateRegex(condition RuleCondition, payload map[string]interface{}) (bool, string) {
	value, err := s.resolvePayloadPath(condition.Expression, payload)
	if err != nil {
		return false, ""
	}

	actual := fmt.Sprintf("%v", value)
	re, err := regexp.Compile(condition.Value)
	if err != nil {
		return false, actual
	}
	return re.MatchString(actual), actual
}

func (s *Service) evaluateEventType(condition RuleCondition, eventType string) (bool, string) {
	switch condition.Operator {
	case "equals":
		return eventType == condition.Value, eventType
	case "contains":
		return strings.Contains(eventType, condition.Value), eventType
	case "matches":
		re, err := regexp.Compile(condition.Value)
		if err != nil {
			return false, eventType
		}
		return re.MatchString(eventType), eventType
	default:
		return eventType == condition.Value, eventType
	}
}

// resolvePayloadPath navigates a map using dot-notation (e.g., "data.user.name")
func (s *Service) resolvePayloadPath(path string, payload map[string]interface{}) (interface{}, error) {
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	if path == "" {
		return payload, nil
	}

	parts := strings.Split(path, ".")
	var current interface{} = payload

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot traverse into non-object at %q", part)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("field %q not found", part)
		}
		current = val
	}

	return current, nil
}

func (s *Service) numericCompare(a interface{}, bStr string, cmp func(float64, float64) bool) bool {
	aF, ok := toFloat(a)
	if !ok {
		return false
	}
	bF, ok := toFloat(bStr)
	if !ok {
		return false
	}
	return cmp(aF, bF)
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		var f float64
		_, err := fmt.Sscanf(n, "%f", &f)
		return f, err == nil
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
