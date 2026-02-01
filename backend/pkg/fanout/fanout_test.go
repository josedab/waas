package fanout

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/josedab/waas/pkg/queue"
)

// ---- Filter Tests ----

func TestFilterEngine_EmptyExpression(t *testing.T) {
	f := NewFilterEngine()
	match, err := f.Evaluate("", json.RawMessage(`{"key":"value"}`))
	require.NoError(t, err)
	assert.True(t, match)
}

func TestFilterEngine_ExactMatch(t *testing.T) {
	f := NewFilterEngine()

	payload := json.RawMessage(`{"event_type":"order.created","status":"active"}`)

	match, err := f.Evaluate(`$.event_type == "order.created"`, payload)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = f.Evaluate(`$.event_type == "order.updated"`, payload)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestFilterEngine_NotEqual(t *testing.T) {
	f := NewFilterEngine()

	payload := json.RawMessage(`{"status":"active"}`)

	match, err := f.Evaluate(`$.status != "inactive"`, payload)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = f.Evaluate(`$.status != "active"`, payload)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestFilterEngine_NumericComparison(t *testing.T) {
	f := NewFilterEngine()
	payload := json.RawMessage(`{"amount":150,"count":5}`)

	tests := []struct {
		expr     string
		expected bool
	}{
		{`$.amount > 100`, true},
		{`$.amount > 200`, false},
		{`$.amount >= 150`, true},
		{`$.amount < 200`, true},
		{`$.amount < 100`, false},
		{`$.amount <= 150`, true},
		{`$.count == 5`, true},
	}

	for _, tt := range tests {
		match, err := f.Evaluate(tt.expr, payload)
		require.NoError(t, err, "expression: %s", tt.expr)
		assert.Equal(t, tt.expected, match, "expression: %s", tt.expr)
	}
}

func TestFilterEngine_InOperator(t *testing.T) {
	f := NewFilterEngine()
	payload := json.RawMessage(`{"status":"active","region":"us-east"}`)

	match, err := f.Evaluate(`$.status in ["active", "pending"]`, payload)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = f.Evaluate(`$.status in ["inactive", "deleted"]`, payload)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestFilterEngine_NestedFieldAccess(t *testing.T) {
	f := NewFilterEngine()
	payload := json.RawMessage(`{"metadata":{"region":"us-east","tier":"premium"}}`)

	match, err := f.Evaluate(`$.metadata.region == "us-east"`, payload)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = f.Evaluate(`$.metadata.region == "eu-west"`, payload)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestFilterEngine_MissingField(t *testing.T) {
	f := NewFilterEngine()
	payload := json.RawMessage(`{"status":"active"}`)

	match, err := f.Evaluate(`$.nonexistent == "value"`, payload)
	require.NoError(t, err)
	assert.False(t, match) // missing field → no match
}

func TestFilterEngine_InvalidExpression(t *testing.T) {
	f := NewFilterEngine()
	_, err := f.Evaluate(`this is not valid`, json.RawMessage(`{}`))
	assert.Error(t, err)
}

func TestFilterEngine_Validate(t *testing.T) {
	f := NewFilterEngine()

	assert.NoError(t, f.Validate(""))
	assert.NoError(t, f.Validate(`$.status == "active"`))
	assert.NoError(t, f.Validate(`$.amount > 100`))
	assert.Error(t, f.Validate(`no operator here`))
}

func TestFilterEngine_InvalidPayload(t *testing.T) {
	f := NewFilterEngine()
	_, err := f.Evaluate(`$.key == "val"`, json.RawMessage(`not json`))
	assert.Error(t, err)
}

// ---- In-Memory Repository for Service Tests ----

type memoryRepository struct {
	topics        map[uuid.UUID]*Topic
	subscriptions map[uuid.UUID]*Subscription
	events        map[uuid.UUID]*TopicEvent
	rules         map[uuid.UUID]*RoutingRule
	ruleVersions  map[uuid.UUID][]RuleVersion
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		topics:        make(map[uuid.UUID]*Topic),
		subscriptions: make(map[uuid.UUID]*Subscription),
		events:        make(map[uuid.UUID]*TopicEvent),
		rules:         make(map[uuid.UUID]*RoutingRule),
		ruleVersions:  make(map[uuid.UUID][]RuleVersion),
	}
}

func (r *memoryRepository) CreateTopic(_ context.Context, t *Topic) error {
	r.topics[t.ID] = t
	return nil
}

func (r *memoryRepository) GetTopic(_ context.Context, tenantID, topicID uuid.UUID) (*Topic, error) {
	t, ok := r.topics[topicID]
	if !ok || t.TenantID != tenantID {
		return nil, assert.AnError
	}
	return t, nil
}

func (r *memoryRepository) GetTopicByName(_ context.Context, tenantID uuid.UUID, name string) (*Topic, error) {
	for _, t := range r.topics {
		if t.TenantID == tenantID && t.Name == name {
			return t, nil
		}
	}
	return nil, assert.AnError
}

func (r *memoryRepository) ListTopics(_ context.Context, tenantID uuid.UUID, limit, offset int) ([]Topic, int, error) {
	var result []Topic
	for _, t := range r.topics {
		if t.TenantID == tenantID {
			result = append(result, *t)
		}
	}
	total := len(result)
	if offset >= len(result) {
		return []Topic{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (r *memoryRepository) UpdateTopic(_ context.Context, t *Topic) error {
	r.topics[t.ID] = t
	return nil
}

func (r *memoryRepository) DeleteTopic(_ context.Context, tenantID, topicID uuid.UUID) error {
	delete(r.topics, topicID)
	return nil
}

func (r *memoryRepository) CreateSubscription(_ context.Context, s *Subscription) error {
	r.subscriptions[s.ID] = s
	return nil
}

func (r *memoryRepository) GetSubscription(_ context.Context, subID uuid.UUID) (*Subscription, error) {
	s, ok := r.subscriptions[subID]
	if !ok {
		return nil, assert.AnError
	}
	return s, nil
}

func (r *memoryRepository) ListSubscriptions(_ context.Context, topicID uuid.UUID, limit, offset int) ([]Subscription, int, error) {
	var result []Subscription
	for _, s := range r.subscriptions {
		if s.TopicID == topicID {
			result = append(result, *s)
		}
	}
	total := len(result)
	if offset >= len(result) {
		return []Subscription{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (r *memoryRepository) GetActiveSubscriptions(_ context.Context, topicID uuid.UUID) ([]Subscription, error) {
	var result []Subscription
	for _, s := range r.subscriptions {
		if s.TopicID == topicID && s.Active {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (r *memoryRepository) DeleteSubscription(_ context.Context, subID uuid.UUID) error {
	delete(r.subscriptions, subID)
	return nil
}

func (r *memoryRepository) CountSubscriptions(_ context.Context, topicID uuid.UUID) (int, error) {
	count := 0
	for _, s := range r.subscriptions {
		if s.TopicID == topicID && s.Active {
			count++
		}
	}
	return count, nil
}

func (r *memoryRepository) CreateEvent(_ context.Context, e *TopicEvent) error {
	r.events[e.ID] = e
	return nil
}

func (r *memoryRepository) GetEvent(_ context.Context, eventID uuid.UUID) (*TopicEvent, error) {
	e, ok := r.events[eventID]
	if !ok {
		return nil, assert.AnError
	}
	return e, nil
}

func (r *memoryRepository) ListEvents(_ context.Context, topicID uuid.UUID, limit, offset int) ([]TopicEvent, int, error) {
	var result []TopicEvent
	for _, e := range r.events {
		if e.TopicID == topicID {
			result = append(result, *e)
		}
	}
	total := len(result)
	if offset >= len(result) {
		return []TopicEvent{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (r *memoryRepository) UpdateEventStatus(_ context.Context, eventID uuid.UUID, status string, fanOutCount int) error {
	e, ok := r.events[eventID]
	if !ok {
		return assert.AnError
	}
	e.Status = status
	e.FanOutCount = fanOutCount
	return nil
}

func (r *memoryRepository) CreateRoutingRule(_ context.Context, rule *RoutingRule) error {
	r.rules[rule.ID] = rule
	return nil
}

func (r *memoryRepository) GetRoutingRule(_ context.Context, tenantID, ruleID uuid.UUID) (*RoutingRule, error) {
	rule, ok := r.rules[ruleID]
	if !ok || rule.TenantID != tenantID {
		return nil, assert.AnError
	}
	return rule, nil
}

func (r *memoryRepository) ListRoutingRules(_ context.Context, tenantID, topicID uuid.UUID) ([]RoutingRule, error) {
	var result []RoutingRule
	for _, rule := range r.rules {
		if rule.TenantID == tenantID && rule.TopicID == topicID {
			result = append(result, *rule)
		}
	}
	return result, nil
}

func (r *memoryRepository) UpdateRoutingRule(_ context.Context, rule *RoutingRule) error {
	r.rules[rule.ID] = rule
	return nil
}

func (r *memoryRepository) DeleteRoutingRule(_ context.Context, tenantID, ruleID uuid.UUID) error {
	delete(r.rules, ruleID)
	return nil
}

func (r *memoryRepository) CreateRuleVersion(_ context.Context, v *RuleVersion) error {
	r.ruleVersions[v.RuleID] = append(r.ruleVersions[v.RuleID], *v)
	return nil
}

func (r *memoryRepository) GetRuleVersions(_ context.Context, ruleID uuid.UUID) ([]RuleVersion, error) {
	return r.ruleVersions[ruleID], nil
}

func (r *memoryRepository) GetRuleVersion(_ context.Context, ruleID uuid.UUID, version int) (*RuleVersion, error) {
	for _, v := range r.ruleVersions[ruleID] {
		if v.Version == version {
			return &v, nil
		}
	}
	return nil, assert.AnError
}

// ---- Topic CRUD Tests ----

func TestService_CreateTopic(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, err := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{
		Name:        "orders",
		Description: "Order events",
	})

	require.NoError(t, err)
	assert.Equal(t, "orders", topic.Name)
	assert.Equal(t, TopicStatusActive, topic.Status)
	assert.Equal(t, tenantID, topic.TenantID)
	assert.NotEqual(t, uuid.Nil, topic.ID)
}

func TestService_CreateTopic_EmptyName(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	_, err := svc.CreateTopic(context.Background(), uuid.New(), &CreateTopicRequest{})
	assert.Error(t, err)
}

func TestService_UpdateTopic(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	updated, err := svc.UpdateTopic(context.Background(), tenantID, topic.ID, &UpdateTopicRequest{
		Status: TopicStatusPaused,
	})

	require.NoError(t, err)
	assert.Equal(t, TopicStatusPaused, updated.Status)
}

func TestService_UpdateTopic_InvalidStatus(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	_, err := svc.UpdateTopic(context.Background(), tenantID, topic.ID, &UpdateTopicRequest{
		Status: "invalid_status",
	})
	assert.Error(t, err)
}

func TestService_DeleteTopic(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	err := svc.DeleteTopic(context.Background(), tenantID, topic.ID)
	require.NoError(t, err)

	_, err = svc.GetTopic(context.Background(), tenantID, topic.ID)
	assert.Error(t, err)
}

// ---- Subscription Tests ----

func TestService_Subscribe(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	endpointID := uuid.New()
	sub, err := svc.Subscribe(context.Background(), tenantID, topic.ID, endpointID, `$.event_type == "order.created"`)

	require.NoError(t, err)
	assert.Equal(t, topic.ID, sub.TopicID)
	assert.Equal(t, endpointID, sub.EndpointID)
	assert.True(t, sub.Active)
}

func TestService_Subscribe_InvalidFilter(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	_, err := svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), "bad filter")
	assert.Error(t, err)
}

func TestService_Subscribe_MaxLimit(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)
	svc.config.MaxSubscribersPerTopic = 2

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{
		Name:           "orders",
		MaxSubscribers: 2,
	})

	_, err := svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), "")
	require.NoError(t, err)
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), "")
	require.NoError(t, err)

	// Third should fail
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), "")
	assert.Error(t, err)
}

func TestService_Unsubscribe(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	sub, _ := svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), "")
	err := svc.Unsubscribe(context.Background(), sub.ID)
	require.NoError(t, err)
}

// ---- Fan-Out Tests ----

func TestService_FanOut_AllMatch(t *testing.T) {
	svc := NewService(newMemoryRepository())

	event := &TopicEvent{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Payload:  json.RawMessage(`{"event_type":"order.created","amount":100}`),
	}

	subs := []Subscription{
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true},
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true},
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true},
	}

	result := svc.FanOut(context.Background(), event, subs)

	assert.Equal(t, 3, result.TotalTargets)
	assert.Equal(t, 3, result.Delivered)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 0, result.Filtered)
}

func TestService_FanOut_WithFilters(t *testing.T) {
	svc := NewService(newMemoryRepository())

	event := &TopicEvent{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Payload:  json.RawMessage(`{"event_type":"order.created","amount":150}`),
	}

	subs := []Subscription{
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true, FilterExpr: `$.event_type == "order.created"`},
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true, FilterExpr: `$.event_type == "order.updated"`},
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true, FilterExpr: `$.amount > 100`},
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true}, // no filter - matches all
	}

	result := svc.FanOut(context.Background(), event, subs)

	assert.Equal(t, 4, result.TotalTargets)
	assert.Equal(t, 3, result.Delivered) // first, third, and fourth match
	assert.Equal(t, 1, result.Filtered)  // second doesn't match
	assert.Equal(t, 0, result.Failed)
}

func TestService_FanOut_WithPublisher(t *testing.T) {
	repo := newMemoryRepository()
	pub := queue.NewTestPublisher()
	svc := NewService(repo)
	svc.publisher = pub

	event := &TopicEvent{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Payload:  json.RawMessage(`{"key":"value"}`),
	}

	subs := []Subscription{
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true},
		{ID: uuid.New(), EndpointID: uuid.New(), Active: true},
	}

	result := svc.FanOut(context.Background(), event, subs)

	assert.Equal(t, 2, result.Delivered)
	assert.Len(t, pub.GetDeliveryMessages(), 2)

	for _, target := range result.Results {
		assert.Equal(t, TargetStatusQueued, target.Status)
	}
}

func TestService_Publish_EndToEnd(t *testing.T) {
	repo := newMemoryRepository()
	pub := queue.NewTestPublisher()
	svc := NewService(repo)
	svc.publisher = pub

	tenantID := uuid.New()
	topic, err := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})
	require.NoError(t, err)

	// Subscribe two endpoints
	ep1 := uuid.New()
	ep2 := uuid.New()
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, ep1, `$.event_type == "order.created"`)
	require.NoError(t, err)
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, ep2, "")
	require.NoError(t, err)

	// Publish event
	payload := json.RawMessage(`{"event_type":"order.created","order_id":"123"}`)
	result, err := svc.Publish(context.Background(), tenantID, "orders", "order.created", payload, nil)
	require.NoError(t, err)

	assert.Equal(t, 2, result.TotalTargets)
	assert.Equal(t, 2, result.Delivered) // both should match
	assert.Len(t, pub.GetDeliveryMessages(), 2)
}

func TestService_Publish_PausedTopic(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	// Pause the topic
	svc.UpdateTopic(context.Background(), tenantID, topic.ID, &UpdateTopicRequest{Status: TopicStatusPaused})

	_, err := svc.Publish(context.Background(), tenantID, "orders", "test", json.RawMessage(`{}`), nil)
	assert.Error(t, err)
}

func TestService_Publish_NonexistentTopic(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	_, err := svc.Publish(context.Background(), uuid.New(), "nonexistent", "test", json.RawMessage(`{}`), nil)
	assert.Error(t, err)
}

// ---- Model Validation Tests ----

func TestTopicDefaults(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, err := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{
		Name: "test-topic",
	})

	require.NoError(t, err)
	assert.Equal(t, 100, topic.MaxSubscribers) // default from config
	assert.Equal(t, 30, topic.RetentionDays)   // default from config
}

func TestSubscription_CreatedAtIsSet(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "test"})
	sub, err := svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), "")

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), sub.CreatedAt, time.Second)
}

// ---- Routing Rule Tests ----

func TestService_CreateRoutingRule(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	rule, err := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name: "route-high-value",
		Conditions: []RuleCondition{
			{Type: "jsonpath", Expression: "$.amount", Operator: "gt", Value: "100"},
		},
		Actions: []RuleAction{
			{Type: "route", DestinationID: uuid.New().String()},
		},
		Priority: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, "route-high-value", rule.Name)
	assert.Equal(t, 1, rule.Version)
	assert.True(t, rule.Enabled)
	assert.Equal(t, topic.ID, rule.TopicID)
}

func TestService_CreateRoutingRule_Validation(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topicID := uuid.New()

	_, err := svc.CreateRoutingRule(context.Background(), tenantID, topicID, &CreateRuleRequest{})
	assert.Error(t, err)

	_, err = svc.CreateRoutingRule(context.Background(), tenantID, topicID, &CreateRuleRequest{
		Name:       "test",
		Conditions: []RuleCondition{},
		Actions:    []RuleAction{{Type: "route"}},
	})
	assert.Error(t, err)

	_, err = svc.CreateRoutingRule(context.Background(), tenantID, topicID, &CreateRuleRequest{
		Name:       "test",
		Conditions: []RuleCondition{{Type: "jsonpath", Expression: "$.x", Operator: "equals", Value: "1"}},
		Actions:    []RuleAction{},
	})
	assert.Error(t, err)
}

func TestService_UpdateRoutingRule_Versioning(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	rule, _ := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name:       "rule-v1",
		Conditions: []RuleCondition{{Type: "jsonpath", Expression: "$.status", Operator: "equals", Value: "active"}},
		Actions:    []RuleAction{{Type: "route", DestinationID: "dest-1"}},
	})

	updated, err := svc.UpdateRoutingRule(context.Background(), tenantID, rule.ID, &CreateRuleRequest{
		Name:       "rule-v2",
		Conditions: []RuleCondition{{Type: "jsonpath", Expression: "$.status", Operator: "equals", Value: "pending"}},
		Actions:    []RuleAction{{Type: "route", DestinationID: "dest-2"}},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, updated.Version)
	assert.Equal(t, "rule-v2", updated.Name)

	// Verify version history was saved
	versions, err := svc.GetRuleVersions(context.Background(), rule.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 2) // initial v1 + saved v1 before update
}

func TestService_RollbackRule(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	rule, _ := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name:       "rule",
		Conditions: []RuleCondition{{Type: "jsonpath", Expression: "$.status", Operator: "equals", Value: "v1"}},
		Actions:    []RuleAction{{Type: "route", DestinationID: "dest-1"}},
	})

	// Update to v2
	svc.UpdateRoutingRule(context.Background(), tenantID, rule.ID, &CreateRuleRequest{
		Name:       "rule",
		Conditions: []RuleCondition{{Type: "jsonpath", Expression: "$.status", Operator: "equals", Value: "v2"}},
		Actions:    []RuleAction{{Type: "route", DestinationID: "dest-2"}},
	})

	// Rollback to v1
	rolled, err := svc.RollbackRule(context.Background(), tenantID, rule.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, rolled.Version)
	assert.Equal(t, "v1", rolled.Conditions[0].Value)
}

func TestService_TestRule_JSONPath(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	rule, _ := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name: "high-value",
		Conditions: []RuleCondition{
			{Type: "jsonpath", Expression: "$.amount", Operator: "gt", Value: "100"},
			{Type: "jsonpath", Expression: "$.status", Operator: "equals", Value: "active"},
		},
		Actions: []RuleAction{{Type: "route", DestinationID: "dest-1"}},
	})

	// Test matching payload
	result, err := svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload: json.RawMessage(`{"amount": 200, "status": "active"}`),
	})
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.Len(t, result.Conditions, 2)
	assert.True(t, result.Conditions[0].Matched)
	assert.True(t, result.Conditions[1].Matched)
	assert.Len(t, result.Actions, 1)

	// Test non-matching payload
	result, err = svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload: json.RawMessage(`{"amount": 50, "status": "active"}`),
	})
	require.NoError(t, err)
	assert.False(t, result.Matched)
	assert.False(t, result.Conditions[0].Matched)
	assert.True(t, result.Conditions[1].Matched)
	assert.Nil(t, result.Actions)
}

func TestService_TestRule_HeaderMatching(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "events"})

	rule, _ := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name: "content-type-check",
		Conditions: []RuleCondition{
			{Type: "header", Expression: "Content-Type", Operator: "equals", Value: "application/json"},
		},
		Actions: []RuleAction{{Type: "route", DestinationID: "dest-1"}},
	})

	result, err := svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload: json.RawMessage(`{}`),
		Headers: map[string]string{"Content-Type": "application/json"},
	})
	require.NoError(t, err)
	assert.True(t, result.Matched)

	result, err = svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload: json.RawMessage(`{}`),
		Headers: map[string]string{"Content-Type": "text/plain"},
	})
	require.NoError(t, err)
	assert.False(t, result.Matched)
}

func TestService_TestRule_Regex(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "events"})

	rule, _ := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name: "email-check",
		Conditions: []RuleCondition{
			{Type: "regex", Expression: "$.email", Value: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`},
		},
		Actions: []RuleAction{{Type: "filter"}},
	})

	result, err := svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload: json.RawMessage(`{"email": "test@example.com"}`),
	})
	require.NoError(t, err)
	assert.True(t, result.Matched)

	result, err = svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload: json.RawMessage(`{"email": "not-an-email"}`),
	})
	require.NoError(t, err)
	assert.False(t, result.Matched)
}

func TestService_TestRule_EventType(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "events"})

	rule, _ := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name: "order-events",
		Conditions: []RuleCondition{
			{Type: "event_type", Operator: "contains", Value: "order"},
		},
		Actions: []RuleAction{{Type: "route", DestinationID: "dest-1"}},
	})

	result, err := svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload:   json.RawMessage(`{}`),
		EventType: "order.created",
	})
	require.NoError(t, err)
	assert.True(t, result.Matched)

	result, err = svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload:   json.RawMessage(`{}`),
		EventType: "user.updated",
	})
	require.NoError(t, err)
	assert.False(t, result.Matched)
}

func TestService_DeleteRoutingRule(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	rule, _ := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name:       "rule",
		Conditions: []RuleCondition{{Type: "jsonpath", Expression: "$.x", Operator: "equals", Value: "1"}},
		Actions:    []RuleAction{{Type: "route"}},
	})

	err := svc.DeleteRoutingRule(context.Background(), tenantID, rule.ID)
	require.NoError(t, err)

	_, err = svc.GetRoutingRule(context.Background(), tenantID, rule.ID)
	assert.Error(t, err)
}

func TestService_ListRoutingRules(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name:       "rule-1",
		Conditions: []RuleCondition{{Type: "jsonpath", Expression: "$.x", Operator: "equals", Value: "1"}},
		Actions:    []RuleAction{{Type: "route"}},
	})
	svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name:       "rule-2",
		Conditions: []RuleCondition{{Type: "jsonpath", Expression: "$.y", Operator: "equals", Value: "2"}},
		Actions:    []RuleAction{{Type: "filter"}},
	})

	rules, err := svc.ListRoutingRules(context.Background(), tenantID, topic.ID)
	require.NoError(t, err)
	assert.Len(t, rules, 2)
}

func TestService_TestRule_RetryPolicy(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "events"})

	rule, _ := svc.CreateRoutingRule(context.Background(), tenantID, topic.ID, &CreateRuleRequest{
		Name: "with-retry",
		Conditions: []RuleCondition{
			{Type: "jsonpath", Expression: "$.status", Operator: "equals", Value: "active"},
		},
		Actions: []RuleAction{{
			Type:          "route",
			DestinationID: "dest-1",
			RetryPolicy: &RetryPolicy{
				MaxRetries:        5,
				InitialDelayMs:    1000,
				MaxDelayMs:        30000,
				BackoffMultiplier: 2,
			},
		}},
	})

	result, err := svc.TestRule(context.Background(), tenantID, rule.ID, &RuleTestRequest{
		Payload: json.RawMessage(`{"status": "active"}`),
	})
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.NotNil(t, result.Actions[0].RetryPolicy)
	assert.Equal(t, 5, result.Actions[0].RetryPolicy.MaxRetries)
}

// ---- FanOutDelivery Tests ----

func TestService_FanOutDelivery_AllSucceed(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, err := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})
	require.NoError(t, err)

	// Subscribe three endpoints with no filters
	for i := 0; i < 3; i++ {
		_, err := svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), "")
		require.NoError(t, err)
	}

	payload := json.RawMessage(`{"event_type":"order.created","amount":100}`)
	result, err := svc.FanOutDelivery(context.Background(), tenantID, topic.ID, payload, nil)

	require.NoError(t, err)
	assert.Equal(t, topic.ID, result.TopicID)
	assert.Equal(t, 3, result.TotalTargets)
	assert.Equal(t, 3, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 0, result.Pending)
	assert.Len(t, result.TargetResults, 3)
	for _, tr := range result.TargetResults {
		assert.Equal(t, DeliveryStatusDelivered, tr.Status)
	}
}

func TestService_FanOutDelivery_WithFilterExclusion(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, err := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})
	require.NoError(t, err)

	// Sub 1: matches
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), `$.event_type == "order.created"`)
	require.NoError(t, err)
	// Sub 2: does NOT match
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), `$.event_type == "order.updated"`)
	require.NoError(t, err)
	// Sub 3: no filter, matches all
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), "")
	require.NoError(t, err)

	payload := json.RawMessage(`{"event_type":"order.created","amount":50}`)
	result, err := svc.FanOutDelivery(context.Background(), tenantID, topic.ID, payload, nil)

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalTargets)
	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 0, result.Failed)

	filteredCount := 0
	for _, tr := range result.TargetResults {
		if tr.Status == DeliveryStatusFiltered {
			filteredCount++
		}
	}
	assert.Equal(t, 1, filteredCount)
}

func TestService_FanOutDelivery_PausedTopic(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, _ := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})

	// Pause the topic
	svc.UpdateTopic(context.Background(), tenantID, topic.ID, &UpdateTopicRequest{Status: TopicStatusPaused})

	_, err := svc.FanOutDelivery(context.Background(), tenantID, topic.ID, json.RawMessage(`{}`), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestService_FanOutDelivery_NoMatchingSubscriptions(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	topic, err := svc.CreateTopic(context.Background(), tenantID, &CreateTopicRequest{Name: "orders"})
	require.NoError(t, err)

	// All subscriptions have filters that won't match
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), `$.event_type == "user.deleted"`)
	require.NoError(t, err)
	_, err = svc.Subscribe(context.Background(), tenantID, topic.ID, uuid.New(), `$.event_type == "user.updated"`)
	require.NoError(t, err)

	payload := json.RawMessage(`{"event_type":"order.created"}`)
	result, err := svc.FanOutDelivery(context.Background(), tenantID, topic.ID, payload, nil)

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalTargets)
	assert.Equal(t, 0, result.Succeeded)
	assert.Equal(t, 0, result.Failed)

	for _, tr := range result.TargetResults {
		assert.Equal(t, DeliveryStatusFiltered, tr.Status)
	}
}
