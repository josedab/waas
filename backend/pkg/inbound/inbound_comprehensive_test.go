package inbound

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Enhanced Mock Repository with DLQ and configurable errors
// ============================================================

type enhancedMockRepository struct {
	sources        map[string]*InboundSource
	events         map[string]*InboundEvent
	rules          map[string][]RoutingRule
	dlqEntries     map[string]*InboundDLQEntry
	contentRoutes  map[string]*ContentRoute
	transformRules map[string]*TransformRule
	providerHealth map[string]*ProviderHealth
	rateLimits     map[string]*RateLimitConfig
	stats          map[string]*InboundStats

	// Error injection
	createSourceErr       error
	getSourceErr          error
	createEventErr        error
	updateEventStatusErr  error
	getRoutingRulesErr    error
	createContentRouteErr error
	createTransformErr    error
}

func newEnhancedMockRepository() *enhancedMockRepository {
	return &enhancedMockRepository{
		sources:        make(map[string]*InboundSource),
		events:         make(map[string]*InboundEvent),
		rules:          make(map[string][]RoutingRule),
		dlqEntries:     make(map[string]*InboundDLQEntry),
		contentRoutes:  make(map[string]*ContentRoute),
		transformRules: make(map[string]*TransformRule),
		providerHealth: make(map[string]*ProviderHealth),
		rateLimits:     make(map[string]*RateLimitConfig),
		stats:          make(map[string]*InboundStats),
	}
}

func (m *enhancedMockRepository) CreateSource(_ context.Context, source *InboundSource) error {
	if m.createSourceErr != nil {
		return m.createSourceErr
	}
	m.sources[source.ID] = source
	return nil
}

func (m *enhancedMockRepository) GetSource(_ context.Context, sourceID string) (*InboundSource, error) {
	if m.getSourceErr != nil {
		return nil, m.getSourceErr
	}
	s, ok := m.sources[sourceID]
	if !ok {
		return nil, fmt.Errorf("source not found")
	}
	return s, nil
}

func (m *enhancedMockRepository) GetSourceByTenant(_ context.Context, tenantID, sourceID string) (*InboundSource, error) {
	s, ok := m.sources[sourceID]
	if !ok || s.TenantID != tenantID {
		return nil, fmt.Errorf("source not found")
	}
	return s, nil
}

func (m *enhancedMockRepository) ListSources(_ context.Context, tenantID string, limit, offset int) ([]InboundSource, int, error) {
	var result []InboundSource
	for _, s := range m.sources {
		if s.TenantID == tenantID {
			result = append(result, *s)
		}
	}
	total := len(result)
	if offset >= len(result) {
		return []InboundSource{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *enhancedMockRepository) UpdateSource(_ context.Context, source *InboundSource) error {
	m.sources[source.ID] = source
	return nil
}

func (m *enhancedMockRepository) DeleteSource(_ context.Context, tenantID, sourceID string) error {
	s, ok := m.sources[sourceID]
	if !ok || s.TenantID != tenantID {
		return fmt.Errorf("source not found")
	}
	delete(m.sources, sourceID)
	return nil
}

func (m *enhancedMockRepository) CreateRoutingRule(_ context.Context, rule *RoutingRule) error {
	m.rules[rule.SourceID] = append(m.rules[rule.SourceID], *rule)
	return nil
}

func (m *enhancedMockRepository) GetRoutingRules(_ context.Context, sourceID string) ([]RoutingRule, error) {
	if m.getRoutingRulesErr != nil {
		return nil, m.getRoutingRulesErr
	}
	return m.rules[sourceID], nil
}

func (m *enhancedMockRepository) UpdateRoutingRule(_ context.Context, rule *RoutingRule) error {
	return nil
}

func (m *enhancedMockRepository) DeleteRoutingRule(_ context.Context, ruleID string) error {
	return nil
}

func (m *enhancedMockRepository) CreateEvent(_ context.Context, event *InboundEvent) error {
	if m.createEventErr != nil {
		return m.createEventErr
	}
	m.events[event.ID] = event
	return nil
}

func (m *enhancedMockRepository) GetEvent(_ context.Context, eventID string) (*InboundEvent, error) {
	e, ok := m.events[eventID]
	if !ok {
		return nil, fmt.Errorf("event not found")
	}
	return e, nil
}

func (m *enhancedMockRepository) GetEventByTenant(_ context.Context, tenantID, eventID string) (*InboundEvent, error) {
	e, ok := m.events[eventID]
	if !ok || e.TenantID != tenantID {
		return nil, fmt.Errorf("event not found")
	}
	return e, nil
}

func (m *enhancedMockRepository) ListEventsBySource(_ context.Context, sourceID, status string, limit, offset int) ([]InboundEvent, int, error) {
	var result []InboundEvent
	for _, e := range m.events {
		if e.SourceID == sourceID {
			if status == "" || e.Status == status {
				result = append(result, *e)
			}
		}
	}
	total := len(result)
	if offset >= len(result) {
		return []InboundEvent{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *enhancedMockRepository) UpdateEventStatus(_ context.Context, eventID, status, errorMsg string) error {
	if m.updateEventStatusErr != nil {
		return m.updateEventStatusErr
	}
	if e, ok := m.events[eventID]; ok {
		e.Status = status
		e.ErrorMessage = errorMsg
	}
	return nil
}

func (m *enhancedMockRepository) GetDLQEntries(_ context.Context, tenantID string, limit, offset int) ([]InboundDLQEntry, error) {
	var result []InboundDLQEntry
	for _, e := range m.dlqEntries {
		if e.TenantID == tenantID && !e.Replayed {
			result = append(result, *e)
		}
	}
	if offset >= len(result) {
		return []InboundDLQEntry{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *enhancedMockRepository) GetDLQEntry(_ context.Context, tenantID, entryID string) (*InboundDLQEntry, error) {
	e, ok := m.dlqEntries[entryID]
	if !ok || e.TenantID != tenantID {
		return nil, fmt.Errorf("DLQ entry not found")
	}
	return e, nil
}

func (m *enhancedMockRepository) MarkDLQEntryReplayed(_ context.Context, entryID string) error {
	if e, ok := m.dlqEntries[entryID]; ok {
		e.Replayed = true
	}
	return nil
}

func (m *enhancedMockRepository) GetProviderHealth(_ context.Context, sourceID string) (*ProviderHealth, error) {
	h, ok := m.providerHealth[sourceID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return h, nil
}

func (m *enhancedMockRepository) GetRateLimitConfig(_ context.Context, sourceID string) (*RateLimitConfig, error) {
	c, ok := m.rateLimits[sourceID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (m *enhancedMockRepository) GetInboundStats(_ context.Context, sourceID string) (*InboundStats, error) {
	s, ok := m.stats[sourceID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return s, nil
}

func (m *enhancedMockRepository) CreateTransformRule(_ context.Context, rule *TransformRule) error {
	if m.createTransformErr != nil {
		return m.createTransformErr
	}
	m.transformRules[rule.ID] = rule
	return nil
}

func (m *enhancedMockRepository) ListTransformRules(_ context.Context, sourceID string) ([]TransformRule, error) {
	var result []TransformRule
	for _, r := range m.transformRules {
		if r.SourceID == sourceID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *enhancedMockRepository) DeleteTransformRule(_ context.Context, ruleID string) error {
	delete(m.transformRules, ruleID)
	return nil
}

func (m *enhancedMockRepository) CreateContentRoute(_ context.Context, route *ContentRoute) error {
	if m.createContentRouteErr != nil {
		return m.createContentRouteErr
	}
	m.contentRoutes[route.ID] = route
	return nil
}

func (m *enhancedMockRepository) ListContentRoutes(_ context.Context, sourceID string) ([]ContentRoute, error) {
	var result []ContentRoute
	for _, r := range m.contentRoutes {
		if r.SourceID == sourceID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *enhancedMockRepository) DeleteContentRoute(_ context.Context, routeID string) error {
	delete(m.contentRoutes, routeID)
	return nil
}

// ============================================================
// Filter Evaluation Tests (evaluateFilter / navigateJSONPath)
// ============================================================

func TestEvaluateFilter_EmptyExpression(t *testing.T) {
	matched, err := evaluateFilter("", `{"type":"test"}`)
	require.NoError(t, err)
	assert.True(t, matched, "empty expression should match everything")
}

func TestEvaluateFilter_FieldExists(t *testing.T) {
	payload := `{"event":{"type":"push"}}`

	matched, err := evaluateFilter("$.event.type", payload)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = evaluateFilter("$.event.missing", payload)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestEvaluateFilter_Equality(t *testing.T) {
	payload := `{"type":"payment_intent.succeeded","status":"active"}`

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"match equal", `$.type == payment_intent.succeeded`, true},
		{"no match equal", `$.type == payment_intent.failed`, false},
		{"match status", `$.status == active`, true},
		{"no match status", `$.status == inactive`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := evaluateFilter(tt.expression, payload)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, matched)
		})
	}
}

func TestEvaluateFilter_Inequality(t *testing.T) {
	payload := `{"type":"push","action":"created"}`

	matched, err := evaluateFilter(`$.type != pull_request`, payload)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = evaluateFilter(`$.type != push`, payload)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestEvaluateFilter_InequalityMissingField(t *testing.T) {
	payload := `{"type":"push"}`

	matched, err := evaluateFilter(`$.missing_field != something`, payload)
	require.NoError(t, err)
	assert.True(t, matched, "inequality on missing field should return true")
}

func TestEvaluateFilter_Contains(t *testing.T) {
	payload := `{"message":"Hello World from Stripe","type":"notification"}`

	matched, err := evaluateFilter(`$.message contains Stripe`, payload)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = evaluateFilter(`$.message contains GitHub`, payload)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestEvaluateFilter_NestedPath(t *testing.T) {
	payload := `{"data":{"object":{"status":"succeeded","amount":2000}}}`

	matched, err := evaluateFilter(`$.data.object.status == succeeded`, payload)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = evaluateFilter(`$.data.object.amount == 2000`, payload)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestEvaluateFilter_InvalidJSON(t *testing.T) {
	_, err := evaluateFilter(`$.type == test`, `not valid json`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse payload")
}

func TestNavigateJSONPath_RootPrefix(t *testing.T) {
	data := map[string]interface{}{
		"type": "test",
		"nested": map[string]interface{}{
			"key": "value",
		},
	}

	val, found := navigateJSONPath("$.type", data)
	assert.True(t, found)
	assert.Equal(t, "test", val)

	val, found = navigateJSONPath("$type", data)
	assert.True(t, found)
	assert.Equal(t, "test", val)

	val, found = navigateJSONPath("type", data)
	assert.True(t, found)
	assert.Equal(t, "test", val)
}

func TestNavigateJSONPath_DeepNesting(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{
					"d": "deep_value",
				},
			},
		},
	}

	val, found := navigateJSONPath("$.a.b.c.d", data)
	assert.True(t, found)
	assert.Equal(t, "deep_value", val)
}

func TestNavigateJSONPath_NonMapIntermediate(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
	}

	_, found := navigateJSONPath("$.name.sub", data)
	assert.False(t, found, "should fail when intermediate value is not a map")
}

func TestNavigateJSONPath_EmptyParts(t *testing.T) {
	data := map[string]interface{}{
		"type": "test",
	}

	val, found := navigateJSONPath("$..type", data)
	assert.True(t, found)
	assert.Equal(t, "test", val)
}

// ============================================================
// Payload Normalization Tests
// ============================================================

func TestNormalizePayload_Stripe(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	raw := `{"id":"evt_123","type":"payment_intent.succeeded","data":{"amount":2000}}`
	result := svc.normalizePayload(raw, ProviderStripe)

	var normalized map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &normalized))

	assert.Equal(t, ProviderStripe, normalized["provider"])
	assert.Equal(t, "payment_intent.succeeded", normalized["event_type"])
	assert.Equal(t, "evt_123", normalized["external_id"])
	assert.NotNil(t, normalized["received_at"])
	assert.NotNil(t, normalized["payload"])
}

func TestNormalizePayload_GitHub(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	raw := `{"action":"opened","pull_request":{"title":"Fix bug"}}`
	result := svc.normalizePayload(raw, ProviderGitHub)

	var normalized map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &normalized))

	assert.Equal(t, ProviderGitHub, normalized["provider"])
	assert.Equal(t, "opened", normalized["event_type"])
}

func TestNormalizePayload_Slack(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	raw := `{"type":"event_callback","event":{"type":"message"}}`
	result := svc.normalizePayload(raw, ProviderSlack)

	var normalized map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &normalized))

	assert.Equal(t, ProviderSlack, normalized["provider"])
	assert.Equal(t, "event_callback", normalized["event_type"])
}

func TestNormalizePayload_DefaultProvider(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	// With "event_type" field
	raw := `{"event_type":"order.created","data":{"id":1}}`
	result := svc.normalizePayload(raw, ProviderCustom)

	var normalized map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &normalized))
	assert.Equal(t, "order.created", normalized["event_type"])

	// With "type" field fallback
	raw = `{"type":"notification","data":{"id":1}}`
	result = svc.normalizePayload(raw, ProviderCustom)
	require.NoError(t, json.Unmarshal([]byte(result), &normalized))
	assert.Equal(t, "notification", normalized["event_type"])
}

func TestNormalizePayload_InvalidJSON(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	raw := `not valid json`
	result := svc.normalizePayload(raw, ProviderStripe)
	assert.Equal(t, raw, result, "should return raw payload for invalid JSON")
}

func TestNormalizePayload_EmptyPayload(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	result := svc.normalizePayload("", ProviderStripe)
	assert.Equal(t, "", result)
}

func TestNormalizePayload_NoEventType(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	raw := `{"data":{"amount":100}}`
	result := svc.normalizePayload(raw, ProviderCustom)

	var normalized map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result), &normalized))
	assert.Equal(t, ProviderCustom, normalized["provider"])
	_, hasEventType := normalized["event_type"]
	assert.False(t, hasEventType, "should not have event_type when source has none")
}

// ============================================================
// Routing Tests
// ============================================================

func TestRouteEvent_InactiveRulesSkipped(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	rules := []RoutingRule{
		{ID: "r1", SourceID: "src-1", Active: false, DestinationType: DestinationHTTP, DestinationConfig: `{"url":"http://example.com"}`},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.NoError(t, err, "inactive rules should be skipped without error")
}

func TestRouteEvent_InternalDestination(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	rules := []RoutingRule{
		{ID: "r1", SourceID: "src-1", Active: true, DestinationType: DestinationInternal, DestinationConfig: `{}`},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.NoError(t, err, "internal routing should succeed")
}

func TestRouteEvent_QueueDestination_Valid(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	rules := []RoutingRule{
		{ID: "r1", SourceID: "src-1", Active: true, DestinationType: DestinationQueue, DestinationConfig: `{"queue":"events-queue"}`},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.NoError(t, err)
}

func TestRouteEvent_QueueDestination_MissingQueueName(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	rules := []RoutingRule{
		{ID: "r1", SourceID: "src-1", Active: true, DestinationType: DestinationQueue, DestinationConfig: `{"url":"http://example.com"}`},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue name is required")
}

func TestRouteEvent_QueueDestination_InvalidConfig(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	rules := []RoutingRule{
		{ID: "r1", SourceID: "src-1", Active: true, DestinationType: DestinationQueue, DestinationConfig: `invalid-json`},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid destination config")
}

func TestRouteEvent_HTTPDestination_InvalidConfig(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	rules := []RoutingRule{
		{ID: "r1", SourceID: "src-1", Active: true, DestinationType: DestinationHTTP, DestinationConfig: `invalid-json`},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid destination config")
}

func TestRouteEvent_HTTPDestination_MissingURL(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	rules := []RoutingRule{
		{ID: "r1", SourceID: "src-1", Active: true, DestinationType: DestinationHTTP, DestinationConfig: `{"method":"POST"}`},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination URL is required")
}

func TestRouteEvent_FilterExpression_Match(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"payment_intent.succeeded","status":"active"}`,
	}

	rules := []RoutingRule{
		{
			ID:                "r1",
			SourceID:          "src-1",
			Active:            true,
			FilterExpression:  `$.type == payment_intent.succeeded`,
			DestinationType:   DestinationInternal,
			DestinationConfig: `{}`,
		},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.NoError(t, err)
}

func TestRouteEvent_FilterExpression_NoMatch(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"payment_intent.failed"}`,
	}

	// Rule that only routes succeeded events — should skip this event
	rules := []RoutingRule{
		{
			ID:                "r1",
			SourceID:          "src-1",
			Active:            true,
			FilterExpression:  `$.type == payment_intent.succeeded`,
			DestinationType:   DestinationQueue,
			DestinationConfig: `{"queue":"payments"}`,
		},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.NoError(t, err, "non-matching filter should be skipped, not error")
}

func TestRouteEvent_MultipleRules(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	rules := []RoutingRule{
		{ID: "r1", SourceID: "src-1", Active: true, DestinationType: DestinationInternal, DestinationConfig: `{}`},
		{ID: "r2", SourceID: "src-1", Active: true, DestinationType: DestinationQueue, DestinationConfig: `{"queue":"q1"}`},
		{ID: "r3", SourceID: "src-1", Active: false, DestinationType: DestinationQueue, DestinationConfig: `{"queue":"q2"}`},
	}

	err := svc.RouteEvent(context.Background(), event, rules)
	assert.NoError(t, err)
}

func TestRouteEvent_EmptyRules(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	event := &InboundEvent{
		ID:         "evt-1",
		SourceID:   "src-1",
		RawPayload: `{"type":"test"}`,
	}

	err := svc.RouteEvent(context.Background(), event, []RoutingRule{})
	assert.NoError(t, err)
}

// ============================================================
// DLQ Tests
// ============================================================

func TestService_GetDLQEntries_WithEntries(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	repo.dlqEntries["dlq-1"] = &InboundDLQEntry{
		ID:           "dlq-1",
		EventID:      "evt-1",
		SourceID:     "src-1",
		TenantID:     "tenant-1",
		RawPayload:   `{"type":"test"}`,
		ErrorMessage: "signature failed",
		AttemptCount: 3,
		CreatedAt:    time.Now(),
		Replayed:     false,
	}
	repo.dlqEntries["dlq-2"] = &InboundDLQEntry{
		ID:           "dlq-2",
		EventID:      "evt-2",
		SourceID:     "src-1",
		TenantID:     "tenant-1",
		RawPayload:   `{"type":"other"}`,
		ErrorMessage: "timeout",
		AttemptCount: 1,
		CreatedAt:    time.Now(),
		Replayed:     false,
	}

	entries, err := svc.GetDLQEntries(context.Background(), "tenant-1", 10, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestService_GetDLQEntries_ExcludesReplayed(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	repo.dlqEntries["dlq-1"] = &InboundDLQEntry{
		ID: "dlq-1", TenantID: "tenant-1", Replayed: false,
	}
	repo.dlqEntries["dlq-2"] = &InboundDLQEntry{
		ID: "dlq-2", TenantID: "tenant-1", Replayed: true,
	}

	entries, err := svc.GetDLQEntries(context.Background(), "tenant-1", 10, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "dlq-1", entries[0].ID)
}

func TestService_GetDLQEntries_LimitClamping(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	entries, err := svc.GetDLQEntries(context.Background(), "tenant-1", -1, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)

	entries, err = svc.GetDLQEntries(context.Background(), "tenant-1", 200, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestService_GetDLQEntries_NilRepo(t *testing.T) {
	svc := &Service{repo: nil}
	entries, err := svc.GetDLQEntries(context.Background(), "tenant-1", 10, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestService_ReplayDLQEntry_Success(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	// Create a source that the DLQ entry references
	source := &InboundSource{
		ID:       "src-1",
		TenantID: "tenant-1",
		Name:     "Test Source",
		Provider: ProviderCustom,
		Status:   SourceStatusActive,
	}
	repo.sources["src-1"] = source

	repo.dlqEntries["dlq-1"] = &InboundDLQEntry{
		ID:         "dlq-1",
		EventID:    "evt-old",
		SourceID:   "src-1",
		TenantID:   "tenant-1",
		RawPayload: `{"type":"retry_me"}`,
		Replayed:   false,
	}

	event, err := svc.ReplayDLQEntry(context.Background(), "tenant-1", "dlq-1")
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, EventStatusRouted, event.Status)

	// Verify DLQ entry is marked as replayed
	assert.True(t, repo.dlqEntries["dlq-1"].Replayed)
}

func TestService_ReplayDLQEntry_NotFound(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	_, err := svc.ReplayDLQEntry(context.Background(), "tenant-1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DLQ entry not found")
}

func TestService_ReplayDLQEntry_NilRepo(t *testing.T) {
	svc := &Service{repo: nil}
	_, err := svc.ReplayDLQEntry(context.Background(), "tenant-1", "any")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DLQ entry not found")
}

// ============================================================
// Health Monitoring Tests
// ============================================================

func TestService_GetProviderHealth_FromRepo(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	now := time.Now()
	repo.providerHealth["src-1"] = &ProviderHealth{
		SourceID:      "src-1",
		Provider:      ProviderStripe,
		Status:        "degraded",
		SuccessRate:   85.0,
		AvgLatencyMs:  120,
		EventsLast24h: 500,
		ErrorsLast24h: 75,
		LastEventAt:   &now,
	}

	health, err := svc.GetProviderHealth(context.Background(), "tenant-1", "src-1")
	require.NoError(t, err)
	assert.Equal(t, "degraded", health.Status)
	assert.Equal(t, 85.0, health.SuccessRate)
	assert.Equal(t, int64(500), health.EventsLast24h)
}

func TestService_GetProviderHealth_Fallback(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderGitHub, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	health, err := svc.GetProviderHealth(context.Background(), "tenant-1", "src-1")
	require.NoError(t, err)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, ProviderGitHub, health.Provider)
	assert.Equal(t, 99.5, health.SuccessRate)
	assert.NotNil(t, health.LastEventAt)
}

func TestService_GetProviderHealth_SourceNotFound(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	_, err := svc.GetProviderHealth(context.Background(), "tenant-1", "nonexistent")
	assert.Error(t, err)
}

func TestService_GetRateLimitConfig_FromRepo(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	repo.rateLimits["src-1"] = &RateLimitConfig{
		SourceID:       "src-1",
		RequestsPerMin: 500,
		BurstSize:      50,
		Enabled:        true,
	}

	config, err := svc.GetRateLimitConfig(context.Background(), "tenant-1", "src-1")
	require.NoError(t, err)
	assert.Equal(t, 500, config.RequestsPerMin)
	assert.Equal(t, 50, config.BurstSize)
	assert.True(t, config.Enabled)
}

func TestService_GetRateLimitConfig_Fallback(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	config, err := svc.GetRateLimitConfig(context.Background(), "tenant-1", "src-1")
	require.NoError(t, err)
	assert.Equal(t, 1000, config.RequestsPerMin)
	assert.Equal(t, 100, config.BurstSize)
	assert.True(t, config.Enabled)
}

func TestService_GetRateLimitConfig_SourceNotFound(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	_, err := svc.GetRateLimitConfig(context.Background(), "tenant-1", "nonexistent")
	assert.Error(t, err)
}

func TestService_GetInboundStats_FromRepo(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	repo.stats["src-1"] = &InboundStats{
		TotalEvents:    1000,
		ValidatedCount: 950,
		RoutedCount:    900,
		FailedCount:    50,
		DLQCount:       10,
		SuccessRate:    90.0,
	}

	stats, err := svc.GetInboundStats(context.Background(), "tenant-1", "src-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1000), stats.TotalEvents)
	assert.Equal(t, int64(900), stats.RoutedCount)
	assert.Equal(t, int64(50), stats.FailedCount)
}

func TestService_GetInboundStats_Fallback(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	stats, err := svc.GetInboundStats(context.Background(), "tenant-1", "src-1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEvents)
	assert.Equal(t, float64(0), stats.SuccessRate)
}

func TestService_GetInboundStats_SourceNotFound(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	_, err := svc.GetInboundStats(context.Background(), "tenant-1", "nonexistent")
	assert.Error(t, err)
}

// ============================================================
// Content Route and Transform Rule Tests
// ============================================================

func TestService_CreateContentRoute_Success(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	route, err := svc.CreateContentRoute(context.Background(), "tenant-1", "src-1", &CreateContentRouteRequest{
		Name:            "Payment Route",
		MatchExpression: "$.type",
		MatchValue:      "payment_intent.succeeded",
		DestinationType: DestinationHTTP,
		DestinationURL:  "https://payments.example.com/webhook",
		FanOut:          true,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, route.ID)
	assert.Equal(t, "src-1", route.SourceID)
	assert.Equal(t, "Payment Route", route.Name)
	assert.Equal(t, "$.type", route.MatchExpression)
	assert.Equal(t, "payment_intent.succeeded", route.MatchValue)
	assert.True(t, route.FanOut)
	assert.True(t, route.Active)
}

func TestService_CreateContentRoute_SourceNotFound(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateContentRoute(context.Background(), "tenant-1", "nonexistent", &CreateContentRouteRequest{
		Name:            "Test",
		MatchExpression: "$.type",
		DestinationType: DestinationHTTP,
		DestinationURL:  "https://example.com",
	})
	assert.Error(t, err)
}

func TestService_CreateContentRoute_RepoError(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source
	repo.createContentRouteErr = fmt.Errorf("database connection failed")

	_, err := svc.CreateContentRoute(context.Background(), "tenant-1", "src-1", &CreateContentRouteRequest{
		Name:            "Test",
		MatchExpression: "$.type",
		DestinationType: DestinationHTTP,
		DestinationURL:  "https://example.com",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
}

func TestService_CreateTransformRule_Success(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	rule, err := svc.CreateTransformRule(context.Background(), "tenant-1", "src-1", &CreateTransformRuleRequest{
		Name:        "Extract Amount",
		Expression:  "$.data.object.amount",
		TargetField: "amount",
		Priority:    1,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID)
	assert.Equal(t, "src-1", rule.SourceID)
	assert.Equal(t, "Extract Amount", rule.Name)
	assert.Equal(t, "$.data.object.amount", rule.Expression)
	assert.Equal(t, "amount", rule.TargetField)
	assert.True(t, rule.Active)
	assert.Equal(t, 1, rule.Priority)
}

func TestService_CreateTransformRule_SourceNotFound(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateTransformRule(context.Background(), "tenant-1", "nonexistent", &CreateTransformRuleRequest{
		Name:        "Test",
		Expression:  "$.type",
		TargetField: "event_type",
	})
	assert.Error(t, err)
}

func TestService_CreateTransformRule_RepoError(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderStripe, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source
	repo.createTransformErr = fmt.Errorf("storage error")

	_, err := svc.CreateTransformRule(context.Background(), "tenant-1", "src-1", &CreateTransformRuleRequest{
		Name:        "Test",
		Expression:  "$.type",
		TargetField: "event_type",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// ============================================================
// Source CRUD Edge Cases
// ============================================================

func TestService_CreateSource_DefaultAlgorithm(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, err := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Test Source",
		Provider: ProviderGitHub,
	})

	require.NoError(t, err)
	assert.Equal(t, "hmac-sha256", source.VerificationAlgorithm)
}

func TestService_CreateSource_CustomAlgorithm(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, err := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:                  "Test Source",
		Provider:              ProviderCustom,
		VerificationAlgorithm: "hmac-sha1",
	})

	require.NoError(t, err)
	assert.Equal(t, "hmac-sha1", source.VerificationAlgorithm)
}

func TestService_CreateSource_RepoError(t *testing.T) {
	repo := newEnhancedMockRepository()
	repo.createSourceErr = fmt.Errorf("database error")
	svc := NewService(repo)

	_, err := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Test",
		Provider: ProviderStripe,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestService_UpdateSource_PartialUpdate(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:               "Original",
		Provider:           ProviderStripe,
		VerificationSecret: "old_secret",
	})

	// Update only the name — other fields should stay unchanged
	updated, err := svc.UpdateSource(context.Background(), "tenant-1", source.ID, &UpdateSourceRequest{
		Name: "Updated Name",
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "old_secret", updated.VerificationSecret)
	assert.Equal(t, SourceStatusActive, updated.Status)
}

func TestService_UpdateSource_AllStatuses(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Test",
		Provider: ProviderStripe,
	})

	for _, status := range []string{SourceStatusActive, SourceStatusPaused, SourceStatusDisabled} {
		updated, err := svc.UpdateSource(context.Background(), "tenant-1", source.ID, &UpdateSourceRequest{
			Status: status,
		})
		require.NoError(t, err)
		assert.Equal(t, status, updated.Status)
	}
}

func TestService_UpdateSource_NotFound(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateSource(context.Background(), "tenant-1", "nonexistent", &UpdateSourceRequest{
		Name: "Test",
	})
	assert.Error(t, err)
}

func TestService_DeleteSource_WrongTenant(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Test",
		Provider: ProviderStripe,
	})

	err := svc.DeleteSource(context.Background(), "tenant-2", source.ID)
	assert.Error(t, err)
}

func TestService_GetSource_WrongTenant(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Test",
		Provider: ProviderStripe,
	})

	_, err := svc.GetSource(context.Background(), "tenant-2", source.ID)
	assert.Error(t, err)
}

func TestService_ListSources_LimitClamping(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	_, _ = svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "S1", Provider: ProviderStripe,
	})

	// Negative limit → clamped to 20
	sources, _, err := svc.ListSources(context.Background(), "tenant-1", -1, 0)
	require.NoError(t, err)
	assert.Len(t, sources, 1)

	// Zero limit → clamped to 20
	sources, _, err = svc.ListSources(context.Background(), "tenant-1", 0, 0)
	require.NoError(t, err)
	assert.Len(t, sources, 1)

	// Over-limit → clamped to 100
	sources, _, err = svc.ListSources(context.Background(), "tenant-1", 200, 0)
	require.NoError(t, err)
	assert.Len(t, sources, 1)
}

func TestService_ListSources_EmptyTenant(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	sources, total, err := svc.ListSources(context.Background(), "empty-tenant", 10, 0)
	require.NoError(t, err)
	assert.Empty(t, sources)
	assert.Equal(t, 0, total)
}

// ============================================================
// GetSourceEvents Tests
// ============================================================

func TestService_GetSourceEvents_Success(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	// Create some events
	_, _ = svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"n":1}`), map[string][]string{})
	_, _ = svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"n":2}`), map[string][]string{})

	events, total, err := svc.GetSourceEvents(context.Background(), "tenant-1", source.ID, "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, events, 2)
}

func TestService_GetSourceEvents_FilterByStatus(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	_, _ = svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"n":1}`), map[string][]string{})

	events, _, err := svc.GetSourceEvents(context.Background(), "tenant-1", source.ID, EventStatusRouted, 10, 0)
	require.NoError(t, err)
	assert.Len(t, events, 1)

	events, _, err = svc.GetSourceEvents(context.Background(), "tenant-1", source.ID, EventStatusFailed, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestService_GetSourceEvents_WrongTenant(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	_, _, err := svc.GetSourceEvents(context.Background(), "tenant-2", source.ID, "", 10, 0)
	assert.Error(t, err)
}

func TestService_GetSourceEvents_LimitClamping(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	_, _ = svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"n":1}`), map[string][]string{})

	// Negative limit → clamped to 20
	events, _, err := svc.GetSourceEvents(context.Background(), "tenant-1", source.ID, "", -5, 0)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

// ============================================================
// Webhook Processing Edge Cases
// ============================================================

func TestService_ProcessInboundWebhook_EmptyPayload(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(""), map[string][]string{})
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, EventStatusRouted, event.Status)
}

func TestService_ProcessInboundWebhook_LargePayload(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	// 1MB payload
	largeData := strings.Repeat("x", 1024*1024)
	payload := fmt.Sprintf(`{"data":"%s"}`, largeData)

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(payload), map[string][]string{})
	require.NoError(t, err)
	assert.NotNil(t, event)
}

func TestService_ProcessInboundWebhook_MalformedJSON(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{malformed`), map[string][]string{})
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, EventStatusRouted, event.Status)
}

func TestService_ProcessInboundWebhook_DisabledSource(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderStripe,
	})

	_, _ = svc.UpdateSource(context.Background(), "tenant-1", source.ID, &UpdateSourceRequest{
		Status: SourceStatusDisabled,
	})

	_, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte("test"), map[string][]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestService_ProcessInboundWebhook_WithRoutingRules(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	// Add a routing rule
	repo.rules[source.ID] = []RoutingRule{
		{
			ID:                "rule-1",
			SourceID:          source.ID,
			Active:            true,
			DestinationType:   DestinationQueue,
			DestinationConfig: `{"queue":"events"}`,
		},
	}

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"type":"test"}`), map[string][]string{})
	require.NoError(t, err)
	assert.Equal(t, EventStatusRouted, event.Status)
}

func TestService_ProcessInboundWebhook_RoutingFailure(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	// Add a routing rule with invalid config to cause a routing error
	repo.rules[source.ID] = []RoutingRule{
		{
			ID:                "rule-1",
			SourceID:          source.ID,
			Active:            true,
			DestinationType:   DestinationQueue,
			DestinationConfig: `{"url":"http://example.com"}`, // missing queue name
		},
	}

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"type":"test"}`), map[string][]string{})
	// Routing failure returns nil error but sets event status to failed
	assert.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, EventStatusFailed, event.Status)
}

func TestService_ProcessInboundWebhook_CreateEventError(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderCustom, Status: SourceStatusActive,
	}
	repo.sources["src-1"] = source

	repo.createEventErr = fmt.Errorf("storage full")

	_, err := svc.ProcessInboundWebhook(context.Background(), "src-1", []byte(`{"type":"test"}`), map[string][]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store event")
}

func TestService_ProcessInboundWebhook_CustomVerifierConfig(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	secret := "custom_secret"
	payload := []byte(`{"event":"test"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:                  "Custom Source",
		Provider:              ProviderCustom,
		VerificationSecret:    secret,
		VerificationHeader:    "X-My-Sig",
		VerificationAlgorithm: "hmac-sha256",
	})

	headers := map[string][]string{
		"X-My-Sig": {sig},
	}

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, payload, headers)
	require.NoError(t, err)
	assert.True(t, event.SignatureValid)
}

func TestService_ProcessInboundWebhook_StoresHeaders(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	headers := map[string][]string{
		"Content-Type": {"application/json"},
		"X-Custom":     {"value1", "value2"},
	}

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{}`), headers)
	require.NoError(t, err)

	var storedHeaders map[string][]string
	require.NoError(t, json.Unmarshal(event.Headers, &storedHeaders))
	assert.Equal(t, "application/json", storedHeaders["Content-Type"][0])
	assert.Contains(t, storedHeaders["X-Custom"], "value1")
}

// ============================================================
// Replay Tests
// ============================================================

func TestService_ReplayInboundEvent_WithRoutingRules(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	repo.rules[source.ID] = []RoutingRule{
		{ID: "r1", SourceID: source.ID, Active: true, DestinationType: DestinationInternal, DestinationConfig: `{}`},
	}

	event, _ := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"test":true}`), map[string][]string{})

	replayed, err := svc.ReplayInboundEvent(context.Background(), "tenant-1", event.ID)
	require.NoError(t, err)
	assert.Equal(t, EventStatusRouted, replayed.Status)
	assert.NotNil(t, replayed.ProcessedAt)
}

func TestService_ReplayInboundEvent_WrongTenant(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name: "Test", Provider: ProviderCustom,
	})

	event, _ := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"test":true}`), map[string][]string{})

	_, err := svc.ReplayInboundEvent(context.Background(), "tenant-2", event.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event not found")
}

// ============================================================
// Gateway V2 Tests
// ============================================================

func TestGetProviderRegistry_Complete(t *testing.T) {
	registry := GetProviderRegistry()

	assert.GreaterOrEqual(t, len(registry), 19)

	// Verify all entries have required fields
	for _, entry := range registry {
		assert.NotEmpty(t, entry.Provider, "provider should not be empty")
		assert.NotEmpty(t, entry.DisplayName, "display name should not be empty")
		assert.NotEmpty(t, entry.SignatureMethod, "signature method should not be empty")
		assert.NotEmpty(t, entry.SignatureHeader, "signature header should not be empty")
		assert.True(t, entry.IsActive, "all providers should be active")
	}
}

func TestGetProviderRegistry_HasDocURLs(t *testing.T) {
	registry := GetProviderRegistry()

	for _, entry := range registry {
		if entry.Provider != ProviderCustom {
			assert.NotEmpty(t, entry.DocURL, "provider %s should have a doc URL", entry.Provider)
			assert.True(t, strings.HasPrefix(entry.DocURL, "https://"), "doc URL should be HTTPS for %s", entry.Provider)
		}
	}
}

func TestGetProviderRegistry_UniqueProviders(t *testing.T) {
	registry := GetProviderRegistry()
	seen := make(map[string]bool)
	for _, entry := range registry {
		assert.False(t, seen[entry.Provider], "duplicate provider: %s", entry.Provider)
		seen[entry.Provider] = true
	}
}

func TestGenerateIngestURL(t *testing.T) {
	url := GenerateIngestURL("https://api.waas.cloud", "stripe", "src-123")

	assert.Equal(t, "stripe", url.Provider)
	assert.Equal(t, "src-123", url.SourceID)
	assert.Equal(t, "https://api.waas.cloud/inbound/stripe/src-123", url.IngestURL)
}

func TestGenerateIngestURL_DifferentProviders(t *testing.T) {
	tests := []struct {
		provider string
		sourceID string
	}{
		{ProviderStripe, "src-1"},
		{ProviderGitHub, "src-2"},
		{ProviderCustom, "src-3"},
		{ProviderPayPal, "src-4"},
	}

	for _, tt := range tests {
		url := GenerateIngestURL("https://api.example.com", tt.provider, tt.sourceID)
		assert.Contains(t, url.IngestURL, tt.provider)
		assert.Contains(t, url.IngestURL, tt.sourceID)
	}
}

func TestService_GetProviderHealthDashboard(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	dashboard, err := svc.GetProviderHealthDashboard(context.Background(), "tenant-1")
	require.NoError(t, err)

	assert.Equal(t, "tenant-1", dashboard.TenantID)
	assert.NotEmpty(t, dashboard.Providers)
	assert.NotZero(t, dashboard.UpdatedAt)

	// Custom provider should be excluded
	for _, p := range dashboard.Providers {
		assert.NotEqual(t, ProviderCustom, p.Provider)
		assert.Equal(t, "healthy", p.Status)
		assert.Equal(t, 100.0, p.SuccessRate)
	}
}

func TestService_RotateSourceSecret(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	result, err := svc.RotateSourceSecret(context.Background(), "tenant-1", "src-1", "admin@example.com")
	require.NoError(t, err)

	assert.Equal(t, "src-1", result.SourceID)
	assert.Equal(t, "admin@example.com", result.RotatedBy)
	assert.NotZero(t, result.RotatedAt)
	assert.NotEmpty(t, result.NewPrefix)
	assert.True(t, strings.HasSuffix(result.NewPrefix, "..."))
}

// ============================================================
// Additional Signature Verifier Tests
// ============================================================

func TestSendGridVerifier_HMACSignature(t *testing.T) {
	verifier := &SendGridVerifier{}

	secret := "sendgrid_key"
	payload := []byte(`[{"email":"test@example.com","event":"delivered"}]`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Twilio-Email-Event-Webhook-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestSendGridVerifier_InvalidBasicAuth(t *testing.T) {
	verifier := &SendGridVerifier{}

	headers := map[string][]string{
		"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("wrong:creds"))},
	}

	valid, err := verifier.Verify([]byte("test"), headers, "correct:creds")
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestSendGridVerifier_MissingHeaders(t *testing.T) {
	verifier := &SendGridVerifier{}
	_, err := verifier.Verify([]byte("test"), map[string][]string{}, "secret")
	assert.Error(t, err)
}

func TestCustomVerifier_SHA1Algorithm(t *testing.T) {
	verifier := &CustomVerifier{
		HeaderName: "X-Sig",
		Algorithm:  "hmac-sha1",
	}

	secret := "sha1_secret"
	payload := []byte(`test payload`)

	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Sig": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestCustomVerifier_Base64Signature(t *testing.T) {
	verifier := &CustomVerifier{
		HeaderName: "X-Sig",
		Algorithm:  "hmac-sha256",
	}

	secret := "b64_secret"
	payload := []byte(`test payload`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Sig": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestCustomVerifier_MissingHeader(t *testing.T) {
	verifier := &CustomVerifier{HeaderName: "X-Custom-Sig"}

	_, err := verifier.Verify([]byte("test"), map[string][]string{}, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "X-Custom-Sig")
}

func TestCustomVerifier_UnknownAlgorithm(t *testing.T) {
	verifier := &CustomVerifier{
		HeaderName: "X-Sig",
		Algorithm:  "unknown-algo",
	}

	// Unknown algo falls back to SHA256
	secret := "test_secret"
	payload := []byte(`test`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Sig": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestPayPalVerifier_ValidSignature(t *testing.T) {
	verifier := &PayPalVerifier{}
	secret := "paypal_webhook_id"
	payload := []byte(`{"event_type":"PAYMENT.SALE.COMPLETED"}`)
	transmissionID := "txn-123"
	transmissionTime := "2024-01-01T00:00:00Z"

	message := fmt.Sprintf("%s|%s|%s|%s", transmissionID, transmissionTime, secret, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"Paypal-Transmission-Sig":  {sig},
		"Paypal-Transmission-Id":   {transmissionID},
		"Paypal-Transmission-Time": {transmissionTime},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestPayPalVerifier_MissingTransmissionHeaders(t *testing.T) {
	verifier := &PayPalVerifier{}

	headers := map[string][]string{
		"Paypal-Transmission-Sig": {"sig"},
	}
	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing PayPal transmission headers")
}

func TestSquareVerifier_ValidSignature(t *testing.T) {
	verifier := &SquareVerifier{}
	secret := "square_secret"
	notifURL := "https://example.com/webhooks/square"
	payload := []byte(`{"type":"payment.completed"}`)

	signedPayload := notifURL + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Square-Hmacsha256-Signature": {sig},
		"X-Square-Notification-Url":     {notifURL},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestSquareVerifier_WithoutNotificationURL(t *testing.T) {
	verifier := &SquareVerifier{}
	secret := "square_secret"
	payload := []byte(`{"type":"payment.completed"}`)

	// Without notification URL, signedPayload is just the body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Square-Hmacsha256-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestIntercomVerifier_ValidSignature(t *testing.T) {
	verifier := &IntercomVerifier{}
	secret := "intercom_secret"
	payload := []byte(`{"type":"notification_event"}`)

	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	sig := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Hub-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestIntercomVerifier_InvalidFormat(t *testing.T) {
	verifier := &IntercomVerifier{}

	headers := map[string][]string{
		"X-Hub-Signature": {"md5=invalid"},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid X-Hub-Signature format")
}

func TestHubSpotVerifier_V3Signature(t *testing.T) {
	verifier := &HubSpotVerifier{}
	secret := "hubspot_secret"
	payload := []byte(`{"subscriptionType":"contact.creation"}`)
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	message := "POST" + "" + string(payload) + timestamp
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-HubSpot-Signature-V3":      {sig},
		"X-HubSpot-Request-Timestamp": {timestamp},
		"X-HubSpot-Request-Method":    {"POST"},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestHubSpotVerifier_V2Fallback(t *testing.T) {
	verifier := &HubSpotVerifier{}
	secret := "hubspot_secret"
	payload := []byte(`{"subscriptionType":"contact.creation"}`)

	h := sha256.New()
	h.Write([]byte(secret))
	h.Write(payload)
	sig := hex.EncodeToString(h.Sum(nil))

	headers := map[string][]string{
		"X-HubSpot-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestHubSpotVerifier_MissingAllHeaders(t *testing.T) {
	verifier := &HubSpotVerifier{}

	_, err := verifier.Verify([]byte("test"), map[string][]string{}, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing HubSpot signature header")
}

func TestHubSpotVerifier_V3MissingTimestamp(t *testing.T) {
	verifier := &HubSpotVerifier{}

	headers := map[string][]string{
		"X-HubSpot-Signature-V3": {"somesig"},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing X-HubSpot-Request-Timestamp")
}

func TestJiraVerifier_WithPrefix(t *testing.T) {
	verifier := &JiraVerifier{}
	secret := "jira_secret"
	payload := []byte(`{"webhookEvent":"jira:issue_created"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Hub-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestJiraVerifier_WithoutPrefix(t *testing.T) {
	verifier := &JiraVerifier{}
	secret := "jira_secret"
	payload := []byte(`{"webhookEvent":"jira:issue_created"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Hub-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestPagerDutyVerifier_ValidSignature(t *testing.T) {
	verifier := &PagerDutyVerifier{}
	secret := "pagerduty_secret"
	payload := []byte(`{"event":{"event_type":"incident.trigger"}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "v1=" + hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-PagerDuty-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestPagerDutyVerifier_WithoutPrefix(t *testing.T) {
	verifier := &PagerDutyVerifier{}
	secret := "pagerduty_secret"
	payload := []byte(`{"event":"test"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-PagerDuty-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestDatadogVerifier_ValidSignature(t *testing.T) {
	verifier := &DatadogVerifier{}
	secret := "dd_secret"
	payload := []byte(`{"alert_type":"metric"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"DD-Webhook-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVercelVerifier_ValidSignature(t *testing.T) {
	verifier := &VercelVerifier{}
	secret := "vercel_secret"
	payload := []byte(`{"type":"deployment.succeeded"}`)

	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Vercel-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestClerkVerifier_WithWhsecPrefix(t *testing.T) {
	rawSecret := []byte("clerk-test-secret-key")
	b64Secret := base64.StdEncoding.EncodeToString(rawSecret)
	secret := "whsec_" + b64Secret

	payload := []byte(`{"type":"user.created"}`)
	msgID := "msg_abc"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	toSign := fmt.Sprintf("%s.%s.%s", msgID, timestamp, string(payload))
	mac := hmac.New(sha256.New, rawSecret)
	mac.Write([]byte(toSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	verifier := &ClerkVerifier{}
	headers := map[string][]string{
		"Svix-Id":        {msgID},
		"Svix-Timestamp": {timestamp},
		"Svix-Signature": {"v1," + sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestClerkVerifier_MultipleSignatures(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("clerk-secret"))
	payload := []byte(`{"data":"test"}`)
	msgID := "msg_multi"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	toSign := fmt.Sprintf("%s.%s.%s", msgID, timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte("clerk-secret"))
	mac.Write([]byte(toSign))
	validSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	verifier := &ClerkVerifier{}
	headers := map[string][]string{
		"Svix-Id":        {msgID},
		"Svix-Timestamp": {timestamp},
		"Svix-Signature": {"v1,invalidsig v1," + validSig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestClerkVerifier_ExpiredTimestamp(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("clerk-secret"))
	payload := []byte(`{"data":"test"}`)
	msgID := "msg_expired"
	// 10 minutes ago
	timestamp := strconv.FormatInt(time.Now().Unix()-600, 10)

	verifier := &ClerkVerifier{}
	headers := map[string][]string{
		"Svix-Id":        {msgID},
		"Svix-Timestamp": {timestamp},
		"Svix-Signature": {"v1,somesig"},
	}

	_, err := verifier.Verify(payload, headers, secret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of tolerance")
}

func TestClerkVerifier_InvalidTimestamp(t *testing.T) {
	verifier := &ClerkVerifier{}
	headers := map[string][]string{
		"Svix-Id":        {"msg-1"},
		"Svix-Timestamp": {"not-a-number"},
		"Svix-Signature": {"v1,sig"},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Svix-Timestamp")
}

func TestStripeVerifier_ExpiredTimestamp(t *testing.T) {
	verifier := &StripeVerifier{}
	// 10 minutes ago
	timestamp := strconv.FormatInt(time.Now().Unix()-600, 10)

	headers := map[string][]string{
		"Stripe-Signature": {fmt.Sprintf("t=%s,v1=somesig", timestamp)},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp too old")
}

func TestStripeVerifier_InvalidTimestamp(t *testing.T) {
	verifier := &StripeVerifier{}

	headers := map[string][]string{
		"Stripe-Signature": {"t=notanumber,v1=somesig"},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestStripeVerifier_MissingTimestampInHeader(t *testing.T) {
	verifier := &StripeVerifier{}

	headers := map[string][]string{
		"Stripe-Signature": {"v1=somesig"},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Stripe-Signature format")
}

func TestSlackVerifier_ExpiredTimestamp(t *testing.T) {
	verifier := &SlackVerifier{}
	timestamp := strconv.FormatInt(time.Now().Unix()-600, 10)

	headers := map[string][]string{
		"X-Slack-Signature":         {"v0=somesig"},
		"X-Slack-Request-Timestamp": {timestamp},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp too old")
}

func TestSlackVerifier_InvalidSignatureFormat(t *testing.T) {
	verifier := &SlackVerifier{}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	headers := map[string][]string{
		"X-Slack-Signature":         {"invalid_prefix=somesig"},
		"X-Slack-Request-Timestamp": {timestamp},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid X-Slack-Signature format")
}

func TestSlackVerifier_MissingSignature(t *testing.T) {
	verifier := &SlackVerifier{}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	headers := map[string][]string{
		"X-Slack-Request-Timestamp": {timestamp},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing X-Slack-Signature")
}

func TestSlackVerifier_InvalidTimestamp(t *testing.T) {
	verifier := &SlackVerifier{}
	headers := map[string][]string{
		"X-Slack-Signature":         {"v0=sig"},
		"X-Slack-Request-Timestamp": {"not-a-number"},
	}

	_, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid X-Slack-Request-Timestamp")
}

// ============================================================
// V2 Provider Verifier Edge Cases
// ============================================================

func TestGetVerifierV2_UnknownProvider(t *testing.T) {
	v := GetVerifierV2("totally_unknown")
	assert.NotNil(t, v)
	assert.Equal(t, ProviderCustom, v.ProviderName())
}

func TestGetVerifierV2_OriginalProviders(t *testing.T) {
	originals := []string{ProviderStripe, ProviderGitHub, ProviderTwilio, ProviderShopify, ProviderSlack, ProviderSendGrid, ProviderCustom}

	for _, p := range originals {
		v := GetVerifierV2(p)
		assert.Equal(t, p, v.ProviderName(), "v2 should delegate to v1 for %s", p)
	}
}

func TestAbs64(t *testing.T) {
	assert.Equal(t, int64(5), abs64(5))
	assert.Equal(t, int64(5), abs64(-5))
	assert.Equal(t, int64(0), abs64(0))
}

// ============================================================
// Model Serialization Tests
// ============================================================

func TestInboundSource_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	source := InboundSource{
		ID:                    "src-123",
		TenantID:              "tenant-1",
		Name:                  "Test Source",
		Provider:              ProviderStripe,
		VerificationSecret:    "secret",
		VerificationAlgorithm: "hmac-sha256",
		Status:                SourceStatusActive,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	data, err := json.Marshal(source)
	require.NoError(t, err)

	var decoded InboundSource
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, source.ID, decoded.ID)
	assert.Equal(t, source.Name, decoded.Name)
	assert.Equal(t, source.Provider, decoded.Provider)
	assert.Equal(t, source.Status, decoded.Status)
}

func TestInboundEvent_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	event := InboundEvent{
		ID:             "evt-123",
		SourceID:       "src-1",
		TenantID:       "tenant-1",
		Provider:       ProviderGitHub,
		RawPayload:     `{"action":"push"}`,
		Headers:        json.RawMessage(`{"Content-Type":["application/json"]}`),
		SignatureValid: true,
		Status:         EventStatusRouted,
		ProcessedAt:    &now,
		CreatedAt:      now,
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded InboundEvent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, event.ID, decoded.ID)
	assert.Equal(t, event.Provider, decoded.Provider)
	assert.Equal(t, event.SignatureValid, decoded.SignatureValid)
}

func TestInboundDLQEntry_JSONSerialization(t *testing.T) {
	entry := InboundDLQEntry{
		ID:           "dlq-1",
		EventID:      "evt-1",
		SourceID:     "src-1",
		TenantID:     "tenant-1",
		RawPayload:   `{"type":"failed"}`,
		ErrorMessage: "signature mismatch",
		AttemptCount: 3,
		Replayed:     false,
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded InboundDLQEntry
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, entry.ID, decoded.ID)
	assert.Equal(t, entry.ErrorMessage, decoded.ErrorMessage)
	assert.Equal(t, entry.AttemptCount, decoded.AttemptCount)
	assert.False(t, decoded.Replayed)
}

func TestProviderHealth_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	health := ProviderHealth{
		SourceID:          "src-1",
		Provider:          ProviderStripe,
		Status:            "healthy",
		SuccessRate:       99.5,
		AvgLatencyMs:      45,
		EventsLast24h:     1000,
		ErrorsLast24h:     5,
		LastEventAt:       &now,
		ConsecutiveErrors: 0,
	}

	data, err := json.Marshal(health)
	require.NoError(t, err)

	var decoded ProviderHealth
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, health.SuccessRate, decoded.SuccessRate)
	assert.Equal(t, health.EventsLast24h, decoded.EventsLast24h)
}

func TestContentRoute_JSONSerialization(t *testing.T) {
	route := ContentRoute{
		ID:              "cr-1",
		SourceID:        "src-1",
		Name:            "Payment Route",
		MatchExpression: "$.type",
		MatchValue:      "payment",
		DestinationType: DestinationHTTP,
		DestinationURL:  "https://example.com",
		Headers:         map[string]string{"Authorization": "Bearer token"},
		FanOut:          true,
		Active:          true,
		Priority:        1,
	}

	data, err := json.Marshal(route)
	require.NoError(t, err)

	var decoded ContentRoute
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, route.Name, decoded.Name)
	assert.Equal(t, route.FanOut, decoded.FanOut)
	assert.Equal(t, "Bearer token", decoded.Headers["Authorization"])
}

// ============================================================
// Constants Verification
// ============================================================

func TestConstants_SourceStatuses(t *testing.T) {
	assert.Equal(t, "active", SourceStatusActive)
	assert.Equal(t, "paused", SourceStatusPaused)
	assert.Equal(t, "disabled", SourceStatusDisabled)
}

func TestConstants_EventStatuses(t *testing.T) {
	assert.Equal(t, "received", EventStatusReceived)
	assert.Equal(t, "validated", EventStatusValidated)
	assert.Equal(t, "routed", EventStatusRouted)
	assert.Equal(t, "failed", EventStatusFailed)
}

func TestConstants_DestinationTypes(t *testing.T) {
	assert.Equal(t, "http", DestinationHTTP)
	assert.Equal(t, "queue", DestinationQueue)
	assert.Equal(t, "internal", DestinationInternal)
}

func TestConstants_ProviderNames(t *testing.T) {
	assert.Equal(t, "stripe", ProviderStripe)
	assert.Equal(t, "github", ProviderGitHub)
	assert.Equal(t, "twilio", ProviderTwilio)
	assert.Equal(t, "shopify", ProviderShopify)
	assert.Equal(t, "slack", ProviderSlack)
	assert.Equal(t, "sendgrid", ProviderSendGrid)
	assert.Equal(t, "custom", ProviderCustom)
	assert.Equal(t, "paypal", ProviderPayPal)
	assert.Equal(t, "square", ProviderSquare)
	assert.Equal(t, "intercom", ProviderIntercom)
	assert.Equal(t, "zendesk", ProviderZendesk)
	assert.Equal(t, "hubspot", ProviderHubSpot)
	assert.Equal(t, "jira", ProviderJira)
	assert.Equal(t, "linear", ProviderLinear)
	assert.Equal(t, "pagerduty", ProviderPagerDuty)
	assert.Equal(t, "datadog", ProviderDatadog)
	assert.Equal(t, "sentry", ProviderSentry)
	assert.Equal(t, "vercel", ProviderVercel)
	assert.Equal(t, "clerk", ProviderClerk)
}

// ============================================================
// Header-Based Routing Tests
// ============================================================

func TestGetHeader_MultipleValues(t *testing.T) {
	headers := map[string][]string{
		"X-Custom": {"first", "second"},
	}

	assert.Equal(t, "first", getHeader(headers, "X-Custom"))
}

func TestGetHeader_EmptyValues(t *testing.T) {
	headers := map[string][]string{
		"X-Custom": {},
	}

	assert.Equal(t, "", getHeader(headers, "X-Custom"))
}

func TestGetHeader_NilHeaders(t *testing.T) {
	assert.Equal(t, "", getHeader(nil, "X-Custom"))
}

// ============================================================
// Integration-like Tests (full workflow)
// ============================================================

func TestFullWorkflow_CreateSourceProcessAndReplay(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	// 1. Create source
	source, err := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Integration Test Source",
		Provider: ProviderCustom,
	})
	require.NoError(t, err)

	// 2. Add routing rule
	repo.rules[source.ID] = []RoutingRule{
		{ID: "r1", SourceID: source.ID, Active: true, DestinationType: DestinationInternal, DestinationConfig: `{}`},
	}

	// 3. Process webhook
	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"action":"test"}`), map[string][]string{
		"Content-Type": {"application/json"},
	})
	require.NoError(t, err)
	assert.Equal(t, EventStatusRouted, event.Status)
	assert.True(t, event.SignatureValid)

	// 4. Verify event is retrievable
	events, total, err := svc.GetSourceEvents(context.Background(), "tenant-1", source.ID, "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, events, 1)

	// 5. Replay the event
	replayed, err := svc.ReplayInboundEvent(context.Background(), "tenant-1", event.ID)
	require.NoError(t, err)
	assert.Equal(t, EventStatusRouted, replayed.Status)
}

func TestFullWorkflow_VerifiedWebhookWithRouting(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	secret := "github_secret_key"
	source, err := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:               "GitHub Hooks",
		Provider:           ProviderGitHub,
		VerificationSecret: secret,
	})
	require.NoError(t, err)

	// Add routing rules
	repo.rules[source.ID] = []RoutingRule{
		{
			ID:                "r1",
			SourceID:          source.ID,
			Active:            true,
			FilterExpression:  `$.action == opened`,
			DestinationType:   DestinationQueue,
			DestinationConfig: `{"queue":"pr-events"}`,
		},
		{
			ID:                "r2",
			SourceID:          source.ID,
			Active:            true,
			DestinationType:   DestinationInternal,
			DestinationConfig: `{}`,
		},
	}

	// Compute valid signature
	payload := []byte(`{"action":"opened","pull_request":{"title":"New feature"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Hub-Signature-256": {sig},
		"X-GitHub-Event":      {"pull_request"},
	}

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, payload, headers)
	require.NoError(t, err)
	assert.True(t, event.SignatureValid)
	assert.Equal(t, EventStatusRouted, event.Status)
	assert.Equal(t, ProviderGitHub, event.Provider)
}

func TestFullWorkflow_DLQReplaySuccess(t *testing.T) {
	repo := newEnhancedMockRepository()
	svc := NewService(repo)

	source := &InboundSource{
		ID: "src-1", TenantID: "tenant-1", Provider: ProviderCustom,
		Status: SourceStatusActive, Name: "DLQ Test",
	}
	repo.sources["src-1"] = source

	// Seed a DLQ entry
	repo.dlqEntries["dlq-1"] = &InboundDLQEntry{
		ID:           "dlq-1",
		EventID:      "evt-old",
		SourceID:     "src-1",
		TenantID:     "tenant-1",
		RawPayload:   `{"type":"retry_event"}`,
		ErrorMessage: "timeout on first attempt",
		AttemptCount: 1,
		Replayed:     false,
	}

	// Replay the DLQ entry
	event, err := svc.ReplayDLQEntry(context.Background(), "tenant-1", "dlq-1")
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, EventStatusRouted, event.Status)
	assert.True(t, repo.dlqEntries["dlq-1"].Replayed)
}
