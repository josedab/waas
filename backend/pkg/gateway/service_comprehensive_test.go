package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock Repository
// ---------------------------------------------------------------------------

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) CreateProvider(ctx context.Context, provider *Provider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *mockRepository) GetProvider(ctx context.Context, tenantID, providerID string) (*Provider, error) {
	args := m.Called(ctx, tenantID, providerID)
	if p := args.Get(0); p != nil {
		return p.(*Provider), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) ListProviders(ctx context.Context, tenantID string, limit, offset int) ([]Provider, int, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]Provider), args.Int(1), args.Error(2)
}

func (m *mockRepository) UpdateProvider(ctx context.Context, provider *Provider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *mockRepository) DeleteProvider(ctx context.Context, tenantID, providerID string) error {
	args := m.Called(ctx, tenantID, providerID)
	return args.Error(0)
}

func (m *mockRepository) CreateRoutingRule(ctx context.Context, rule *RoutingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *mockRepository) GetRoutingRule(ctx context.Context, tenantID, ruleID string) (*RoutingRule, error) {
	args := m.Called(ctx, tenantID, ruleID)
	if r := args.Get(0); r != nil {
		return r.(*RoutingRule), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) ListRoutingRules(ctx context.Context, tenantID, providerID string) ([]RoutingRule, error) {
	args := m.Called(ctx, tenantID, providerID)
	return args.Get(0).([]RoutingRule), args.Error(1)
}

func (m *mockRepository) UpdateRoutingRule(ctx context.Context, rule *RoutingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *mockRepository) DeleteRoutingRule(ctx context.Context, tenantID, ruleID string) error {
	args := m.Called(ctx, tenantID, ruleID)
	return args.Error(0)
}

func (m *mockRepository) SaveInboundWebhook(ctx context.Context, webhook *InboundWebhook) error {
	args := m.Called(ctx, webhook)
	return args.Error(0)
}

func (m *mockRepository) GetInboundWebhook(ctx context.Context, tenantID, webhookID string) (*InboundWebhook, error) {
	args := m.Called(ctx, tenantID, webhookID)
	if w := args.Get(0); w != nil {
		return w.(*InboundWebhook), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) ListInboundWebhooks(ctx context.Context, tenantID, providerID string, limit, offset int) ([]InboundWebhook, int, error) {
	args := m.Called(ctx, tenantID, providerID, limit, offset)
	return args.Get(0).([]InboundWebhook), args.Int(1), args.Error(2)
}

// ---------------------------------------------------------------------------
// Mock DeliveryPublisher
// ---------------------------------------------------------------------------

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, tenantID, endpointID string, payload []byte, headers map[string]string) (string, error) {
	args := m.Called(ctx, tenantID, endpointID, payload, headers)
	return args.String(0), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock SignatureVerifier
// ---------------------------------------------------------------------------

type mockVerifier struct {
	mock.Mock
}

func (m *mockVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	args := m.Called(payload, headers, config)
	return args.Bool(0), args.Error(1)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestService(repo *mockRepository, pub *mockPublisher) *Service {
	return NewService(repo, pub)
}

func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// =========================================================================
// Service construction tests
// =========================================================================

func TestNewService(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)

	svc := NewService(repo, pub)

	require.NotNil(t, svc)
	assert.NotNil(t, svc.repo)
	assert.NotNil(t, svc.publisher)
	assert.NotNil(t, svc.verifiers)
}

func TestNewService_NilPublisher(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, nil)

	require.NotNil(t, svc)
	assert.Nil(t, svc.publisher)
	assert.NotNil(t, svc.verifiers)
}

func TestNewService_VerifierRegistryPopulated(t *testing.T) {
	svc := NewService(new(mockRepository), new(mockPublisher))

	for _, pt := range []string{
		ProviderTypeStripe, ProviderTypeGitHub, ProviderTypeShopify,
		ProviderTypeSlack, ProviderTypeCustom,
	} {
		v := svc.verifiers.Get(pt)
		assert.NotNil(t, v, "verifier for %s", pt)
	}
}

// =========================================================================
// Provider CRUD tests
// =========================================================================

func TestCreateProvider_Success(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	repo.On("CreateProvider", mock.Anything, mock.AnythingOfType("*gateway.Provider")).
		Return(nil)

	req := &CreateProviderRequest{
		Name:        "My Stripe",
		Type:        ProviderTypeStripe,
		Description: "Stripe webhook provider",
	}

	provider, err := svc.CreateProvider(context.Background(), "tenant-1", req)

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "tenant-1", provider.TenantID)
	assert.Equal(t, "My Stripe", provider.Name)
	assert.Equal(t, ProviderTypeStripe, provider.Type)
	assert.True(t, provider.IsActive)
	repo.AssertExpectations(t)
}

func TestCreateProvider_WithSignatureConfig(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	repo.On("CreateProvider", mock.Anything, mock.AnythingOfType("*gateway.Provider")).
		Return(nil)

	sigConfig := &SignatureConfig{
		SecretKey:        "whsec_test",
		HeaderName:       "Stripe-Signature",
		Algorithm:        "hmac-sha256",
		ToleranceSeconds: 300,
	}
	req := &CreateProviderRequest{
		Name:            "Stripe",
		Type:            ProviderTypeStripe,
		SignatureConfig: sigConfig,
	}

	provider, err := svc.CreateProvider(context.Background(), "t1", req)

	require.NoError(t, err)
	assert.NotNil(t, provider.SignatureConfig)

	var parsed SignatureConfig
	require.NoError(t, json.Unmarshal(provider.SignatureConfig, &parsed))
	assert.Equal(t, "whsec_test", parsed.SecretKey)
	assert.Equal(t, 300, parsed.ToleranceSeconds)
}

func TestCreateProvider_RepoError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("CreateProvider", mock.Anything, mock.Anything).
		Return(errors.New("db connection lost"))

	_, err := svc.CreateProvider(context.Background(), "t1", &CreateProviderRequest{
		Name: "P", Type: "custom",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create provider")
}

func TestGetProvider_Found(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	expected := &Provider{ID: "p-1", TenantID: "t1", Name: "GitHub"}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(expected, nil)

	provider, err := svc.GetProvider(context.Background(), "t1", "p-1")

	require.NoError(t, err)
	assert.Equal(t, "GitHub", provider.Name)
}

func TestGetProvider_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetProvider", mock.Anything, "t1", "missing").Return(nil, nil)

	provider, err := svc.GetProvider(context.Background(), "t1", "missing")

	require.NoError(t, err)
	assert.Nil(t, provider)
}

func TestGetProvider_RepoError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetProvider", mock.Anything, "t1", "p-1").
		Return(nil, errors.New("timeout"))

	_, err := svc.GetProvider(context.Background(), "t1", "p-1")
	require.Error(t, err)
}

func TestListProviders_DefaultLimit(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	providers := []Provider{{ID: "p1"}, {ID: "p2"}}
	repo.On("ListProviders", mock.Anything, "t1", 20, 0).Return(providers, 2, nil)

	result, total, err := svc.ListProviders(context.Background(), "t1", 0, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, result, 2)
}

func TestListProviders_NegativeLimit(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("ListProviders", mock.Anything, "t1", 20, 5).Return([]Provider{}, 0, nil)

	_, _, err := svc.ListProviders(context.Background(), "t1", -10, 5)
	require.NoError(t, err)
}

func TestListProviders_ExcessiveLimit(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("ListProviders", mock.Anything, "t1", 100, 0).Return([]Provider{}, 0, nil)

	_, _, err := svc.ListProviders(context.Background(), "t1", 500, 0)
	require.NoError(t, err)
}

func TestDeleteProvider_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("DeleteProvider", mock.Anything, "t1", "p-1").Return(nil)

	err := svc.DeleteProvider(context.Background(), "t1", "p-1")
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteProvider_Error(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("DeleteProvider", mock.Anything, "t1", "p-1").
		Return(errors.New("foreign key violation"))

	err := svc.DeleteProvider(context.Background(), "t1", "p-1")
	require.Error(t, err)
}

// =========================================================================
// Routing Rule CRUD tests
// =========================================================================

func TestCreateRoutingRule_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	provider := &Provider{ID: "p-1", TenantID: "t1", IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("CreateRoutingRule", mock.Anything, mock.AnythingOfType("*gateway.RoutingRule")).Return(nil)

	req := &CreateRoutingRuleRequest{
		ProviderID: "p-1",
		Name:       "Route orders",
		Priority:   10,
		Conditions: []RoutingCondition{
			{Field: "event_type", Operator: "equals", Value: "order.created"},
		},
		Destinations: []RoutingDestination{
			{Type: "endpoint", EndpointID: "ep-1"},
		},
	}

	rule, err := svc.CreateRoutingRule(context.Background(), "t1", req)

	require.NoError(t, err)
	require.NotNil(t, rule)
	assert.Equal(t, "Route orders", rule.Name)
	assert.Equal(t, 10, rule.Priority)
	assert.True(t, rule.IsActive)
	repo.AssertExpectations(t)
}

func TestCreateRoutingRule_ProviderNotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetProvider", mock.Anything, "t1", "missing").Return(nil, nil)

	_, err := svc.CreateRoutingRule(context.Background(), "t1", &CreateRoutingRuleRequest{
		ProviderID:   "missing",
		Name:         "Rule",
		Destinations: []RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider not found")
}

func TestCreateRoutingRule_GetProviderError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetProvider", mock.Anything, "t1", "p-1").
		Return(nil, errors.New("db error"))

	_, err := svc.CreateRoutingRule(context.Background(), "t1", &CreateRoutingRuleRequest{
		ProviderID:   "p-1",
		Name:         "Rule",
		Destinations: []RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get provider")
}

func TestCreateRoutingRule_RepoSaveError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	provider := &Provider{ID: "p-1", TenantID: "t1"}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("CreateRoutingRule", mock.Anything, mock.Anything).
		Return(errors.New("duplicate name"))

	_, err := svc.CreateRoutingRule(context.Background(), "t1", &CreateRoutingRuleRequest{
		ProviderID:   "p-1",
		Name:         "Rule",
		Destinations: []RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create routing rule")
}

func TestCreateRoutingRule_WithNoConditions(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	provider := &Provider{ID: "p-1", TenantID: "t1"}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("CreateRoutingRule", mock.Anything, mock.Anything).Return(nil)

	rule, err := svc.CreateRoutingRule(context.Background(), "t1", &CreateRoutingRuleRequest{
		ProviderID:   "p-1",
		Name:         "Catch-all",
		Destinations: []RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}},
	})

	require.NoError(t, err)
	// Nil conditions should marshal to "null"
	assert.NotNil(t, rule)
}

func TestGetRoutingRule_Found(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	expected := &RoutingRule{ID: "r-1", Name: "My rule"}
	repo.On("GetRoutingRule", mock.Anything, "t1", "r-1").Return(expected, nil)

	rule, err := svc.GetRoutingRule(context.Background(), "t1", "r-1")

	require.NoError(t, err)
	assert.Equal(t, "My rule", rule.Name)
}

func TestGetRoutingRule_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetRoutingRule", mock.Anything, "t1", "missing").Return(nil, nil)

	rule, err := svc.GetRoutingRule(context.Background(), "t1", "missing")

	require.NoError(t, err)
	assert.Nil(t, rule)
}

func TestListRoutingRules_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	rules := []RoutingRule{{ID: "r-1"}, {ID: "r-2"}}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)

	result, err := svc.ListRoutingRules(context.Background(), "t1", "p-1")

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestUpdateRoutingRule_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	existing := &RoutingRule{
		ID: "r-1", TenantID: "t1", Name: "Old name", Priority: 1, IsActive: true,
		Conditions:   mustJSON([]RoutingCondition{}),
		Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}}),
	}
	repo.On("GetRoutingRule", mock.Anything, "t1", "r-1").Return(existing, nil)
	repo.On("UpdateRoutingRule", mock.Anything, mock.AnythingOfType("*gateway.RoutingRule")).Return(nil)

	updated, err := svc.UpdateRoutingRule(context.Background(), "t1", "r-1", &UpdateRoutingRuleRequest{
		Name:     "New name",
		Priority: 5,
		IsActive: false,
	})

	require.NoError(t, err)
	assert.Equal(t, "New name", updated.Name)
	assert.Equal(t, 5, updated.Priority)
	assert.False(t, updated.IsActive)
	repo.AssertExpectations(t)
}

func TestUpdateRoutingRule_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetRoutingRule", mock.Anything, "t1", "missing").Return(nil, nil)

	_, err := svc.UpdateRoutingRule(context.Background(), "t1", "missing", &UpdateRoutingRuleRequest{
		Name: "Updated",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "routing rule not found")
}

func TestUpdateRoutingRule_GetError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetRoutingRule", mock.Anything, "t1", "r-1").
		Return(nil, errors.New("db error"))

	_, err := svc.UpdateRoutingRule(context.Background(), "t1", "r-1", &UpdateRoutingRuleRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rule")
}

func TestUpdateRoutingRule_SaveError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	existing := &RoutingRule{ID: "r-1", TenantID: "t1", Name: "Old"}
	repo.On("GetRoutingRule", mock.Anything, "t1", "r-1").Return(existing, nil)
	repo.On("UpdateRoutingRule", mock.Anything, mock.Anything).
		Return(errors.New("constraint violation"))

	_, err := svc.UpdateRoutingRule(context.Background(), "t1", "r-1", &UpdateRoutingRuleRequest{
		Name: "New",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update rule")
}

func TestUpdateRoutingRule_PartialUpdate(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	existing := &RoutingRule{
		ID: "r-1", TenantID: "t1", Name: "Original", Description: "Orig desc",
		Priority: 3, IsActive: true,
	}
	repo.On("GetRoutingRule", mock.Anything, "t1", "r-1").Return(existing, nil)
	repo.On("UpdateRoutingRule", mock.Anything, mock.Anything).Return(nil)

	// Only update description; name and priority stay the same
	updated, err := svc.UpdateRoutingRule(context.Background(), "t1", "r-1", &UpdateRoutingRuleRequest{
		Description: "New desc",
		IsActive:    true,
	})

	require.NoError(t, err)
	assert.Equal(t, "Original", updated.Name) // unchanged
	assert.Equal(t, "New desc", updated.Description)
	assert.Equal(t, 3, updated.Priority) // unchanged (0 is not applied)
}

func TestUpdateRoutingRule_WithConditionsAndDestinations(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	existing := &RoutingRule{ID: "r-1", TenantID: "t1", Name: "Rule"}
	repo.On("GetRoutingRule", mock.Anything, "t1", "r-1").Return(existing, nil)
	repo.On("UpdateRoutingRule", mock.Anything, mock.Anything).Return(nil)

	newConds := []RoutingCondition{{Field: "event_type", Operator: "equals", Value: "push"}}
	newDests := []RoutingDestination{{Type: "endpoint", EndpointID: "ep-new"}}

	updated, err := svc.UpdateRoutingRule(context.Background(), "t1", "r-1", &UpdateRoutingRuleRequest{
		Conditions:   newConds,
		Destinations: newDests,
		IsActive:     true,
	})

	require.NoError(t, err)
	var conds []RoutingCondition
	require.NoError(t, json.Unmarshal(updated.Conditions, &conds))
	assert.Len(t, conds, 1)
	assert.Equal(t, "push", conds[0].Value)
}

func TestDeleteRoutingRule_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("DeleteRoutingRule", mock.Anything, "t1", "r-1").Return(nil)

	err := svc.DeleteRoutingRule(context.Background(), "t1", "r-1")
	require.NoError(t, err)
}

func TestDeleteRoutingRule_Error(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("DeleteRoutingRule", mock.Anything, "t1", "r-1").
		Return(errors.New("not found"))

	err := svc.DeleteRoutingRule(context.Background(), "t1", "r-1")
	require.Error(t, err)
}

// =========================================================================
// ProcessInboundWebhook tests (routing + fanout)
// =========================================================================

func TestProcessInboundWebhook_Success(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{
		ID:       "p-1",
		TenantID: "t1",
		Type:     ProviderTypeCustom,
		IsActive: true,
	}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)

	rules := []RoutingRule{
		{
			ID: "r-1", Name: "All events", IsActive: true,
			Conditions:   mustJSON([]RoutingCondition{}),
			Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}}),
		},
	}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)
	pub.On("Publish", mock.Anything, "t1", "ep-1", mock.Anything, mock.Anything).
		Return("del-123", nil)

	payload := []byte(`{"type":"user.created"}`)
	headers := map[string]string{}

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1", payload, headers)

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalRouted)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Len(t, result.Destinations, 1)
	assert.Equal(t, "queued", result.Destinations[0].Status)
	assert.Equal(t, "del-123", result.Destinations[0].DeliveryID)
}

func TestProcessInboundWebhook_ProviderNotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetProvider", mock.Anything, "t1", "missing").Return(nil, nil)

	_, err := svc.ProcessInboundWebhook(context.Background(), "t1", "missing", []byte(`{}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider not found")
}

func TestProcessInboundWebhook_ProviderInactive(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: "custom", IsActive: false}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)

	_, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1", []byte(`{}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider is inactive")
}

func TestProcessInboundWebhook_SignatureVerificationFailure(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	// Use Stripe provider which requires a Stripe-Signature header
	provider := &Provider{
		ID: "p-1", TenantID: "t1", Type: ProviderTypeStripe, IsActive: true,
		SignatureConfig: mustJSON(&SignatureConfig{SecretKey: "secret"}),
	}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)

	// No Stripe-Signature header → verification fails
	_, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"test"}`), map[string]string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
}

func TestProcessInboundWebhook_NoMatchingRules(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)

	// Rule that won't match
	rules := []RoutingRule{
		{
			ID: "r-1", IsActive: true,
			Conditions:   mustJSON([]RoutingCondition{{Field: "event_type", Operator: "equals", Value: "payment.failed"}}),
			Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}}),
		},
	}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"user.created"}`), map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalRouted)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Empty(t, result.Destinations)
}

func TestProcessInboundWebhook_InactiveRulesSkipped(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)

	rules := []RoutingRule{
		{
			ID: "r-inactive", IsActive: false,
			Conditions:   mustJSON([]RoutingCondition{}),
			Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}}),
		},
	}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"test"}`), map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalRouted)
	pub.AssertNotCalled(t, "Publish")
}

func TestProcessInboundWebhook_MultipleRulesAndDestinations(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)

	rules := []RoutingRule{
		{
			ID: "r-1", Name: "Rule A", IsActive: true,
			Conditions:   mustJSON([]RoutingCondition{}),
			Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}, {Type: "endpoint", EndpointID: "ep-2"}}),
		},
		{
			ID: "r-2", Name: "Rule B", IsActive: true,
			Conditions:   mustJSON([]RoutingCondition{}),
			Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-3"}}),
		},
	}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)
	pub.On("Publish", mock.Anything, "t1", mock.Anything, mock.Anything, mock.Anything).
		Return("del-x", nil)

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"event"}`), map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalRouted)
	assert.Len(t, result.Destinations, 3)
}

func TestProcessInboundWebhook_PublishFailure(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)

	rules := []RoutingRule{
		{
			ID: "r-1", IsActive: true,
			Conditions:   mustJSON([]RoutingCondition{}),
			Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-1"}}),
		},
	}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)
	pub.On("Publish", mock.Anything, "t1", "ep-1", mock.Anything, mock.Anything).
		Return("", errors.New("endpoint unreachable"))

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"test"}`), map[string]string{})

	require.NoError(t, err) // whole process doesn't error; individual destinations do
	assert.Equal(t, 0, result.TotalRouted)
	assert.Equal(t, 1, result.TotalFailed)
	assert.Equal(t, "failed", result.Destinations[0].Status)
	assert.Contains(t, result.Destinations[0].Error, "endpoint unreachable")
}

func TestProcessInboundWebhook_UnsupportedDestinationType(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)

	rules := []RoutingRule{
		{
			ID: "r-1", IsActive: true,
			Conditions:   mustJSON([]RoutingCondition{}),
			Destinations: mustJSON([]RoutingDestination{{Type: "unknown_type", URL: "https://example.com"}}),
		},
	}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"test"}`), map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalFailed)
	assert.Contains(t, result.Destinations[0].Error, "unsupported destination type")
}

func TestProcessInboundWebhook_GetProviderError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetProvider", mock.Anything, "t1", "p-1").
		Return(nil, errors.New("db timeout"))

	_, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1", []byte(`{}`), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get provider")
}

func TestProcessInboundWebhook_ListRulesError(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").
		Return([]RoutingRule{}, errors.New("db error"))

	_, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"test"}`), map[string]string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get routing rules")
}

func TestProcessInboundWebhook_SaveWebhookErrorDoesNotBlock(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	// SaveInboundWebhook fails, but processing continues
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).
		Return(errors.New("save failed"))
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").
		Return([]RoutingRule{}, nil)

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"test"}`), map[string]string{})

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestProcessInboundWebhook_MixedDestinationResults(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)

	rules := []RoutingRule{
		{
			ID: "r-1", IsActive: true,
			Conditions: mustJSON([]RoutingCondition{}),
			Destinations: mustJSON([]RoutingDestination{
				{Type: "endpoint", EndpointID: "ep-ok"},
				{Type: "endpoint", EndpointID: "ep-fail"},
				{Type: "queue", QueueName: "unsupported-dest"},
			}),
		},
	}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)
	pub.On("Publish", mock.Anything, "t1", "ep-ok", mock.Anything, mock.Anything).
		Return("del-ok", nil)
	pub.On("Publish", mock.Anything, "t1", "ep-fail", mock.Anything, mock.Anything).
		Return("", errors.New("timeout"))

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"test"}`), map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalRouted)
	assert.Equal(t, 2, result.TotalFailed) // ep-fail + queue
	assert.Len(t, result.Destinations, 3)
}

// =========================================================================
// matchesConditions tests (detailed)
// =========================================================================

func TestMatchesConditions_NotEqualsOperator(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}

	conds := mustJSON([]RoutingCondition{
		{Field: "event_type", Operator: "not_equals", Value: "payment.failed"},
	})

	assert.True(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "payment.success", conds))
	assert.False(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "payment.failed", conds))
}

func TestMatchesConditions_ContainsOperator(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}

	conds := mustJSON([]RoutingCondition{
		{Field: "event_type", Operator: "contains", Value: "order"},
	})

	assert.True(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "order.created", conds))
	assert.True(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "new_order.shipped", conds))
	assert.False(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "payment.success", conds))
}

func TestMatchesConditions_MatchesRegexOperator(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}

	conds := mustJSON([]RoutingCondition{
		{Field: "event_type", Operator: "matches", Value: `^order\..+`},
	})

	assert.True(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "order.created", conds))
	assert.True(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "order.updated", conds))
	assert.False(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "payment.created", conds))
}

func TestMatchesConditions_ExistsOperator(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}

	conds := mustJSON([]RoutingCondition{
		{Field: "header.X-Custom", Operator: "exists", Value: ""},
	})

	assert.True(t, svc.matchesConditions([]byte(`{}`), map[string]string{"X-Custom": "val"}, "", conds))
	assert.False(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "", conds))
}

func TestMatchesConditions_MultipleConditionsAllMustMatch(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}

	conds := mustJSON([]RoutingCondition{
		{Field: "event_type", Operator: "contains", Value: "order"},
		{Field: "header.X-Source", Operator: "equals", Value: "shopify"},
	})

	// Both match
	assert.True(t, svc.matchesConditions(
		[]byte(`{}`), map[string]string{"X-Source": "shopify"}, "order.created", conds))

	// Only one matches
	assert.False(t, svc.matchesConditions(
		[]byte(`{}`), map[string]string{"X-Source": "stripe"}, "order.created", conds))

	// Neither matches
	assert.False(t, svc.matchesConditions(
		[]byte(`{}`), map[string]string{"X-Source": "stripe"}, "payment.done", conds))
}

func TestMatchesConditions_InvalidJSON(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}
	// Invalid JSON conditions → should return true (fail open)
	assert.True(t, svc.matchesConditions([]byte(`{}`), map[string]string{}, "", []byte(`{not json}`)))
}

func TestMatchesConditions_PayloadFieldExtraction(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}

	payload := []byte(`{"data":{"customer":{"id":"cust_123"}}}`)
	conds := mustJSON([]RoutingCondition{
		{Field: "data.customer.id", Operator: "equals", Value: "cust_123"},
	})

	assert.True(t, svc.matchesConditions(payload, map[string]string{}, "", conds))
}

// =========================================================================
// extractValue tests (detailed)
// =========================================================================

func TestExtractValue_EventTypeAlias(t *testing.T) {
	svc := &Service{}
	payload := map[string]interface{}{"type": "some.type"}
	// Both "event_type" and "type" should return the eventType argument
	assert.Equal(t, "my.event", svc.extractValue("event_type", payload, nil, "my.event"))
	assert.Equal(t, "my.event", svc.extractValue("type", payload, nil, "my.event"))
}

func TestExtractValue_HeaderPrefix(t *testing.T) {
	svc := &Service{}
	headers := map[string]string{"Content-Type": "application/json", "X-Custom": "val"}

	assert.Equal(t, "application/json", svc.extractValue("header.Content-Type", nil, headers, ""))
	assert.Equal(t, "val", svc.extractValue("header.X-Custom", nil, headers, ""))
	assert.Equal(t, "", svc.extractValue("header.Missing", nil, headers, ""))
}

func TestExtractValue_NestedPayload(t *testing.T) {
	svc := &Service{}
	payload := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "deep-value",
			},
		},
	}

	assert.Equal(t, "deep-value", svc.extractValue("a.b.c", payload, nil, ""))
}

func TestExtractValue_NonStringPayloadValue(t *testing.T) {
	svc := &Service{}
	payload := map[string]interface{}{
		"count":  float64(42),
		"active": true,
	}

	assert.Equal(t, "42", svc.extractValue("count", payload, nil, ""))
	assert.Equal(t, "true", svc.extractValue("active", payload, nil, ""))
}

func TestExtractValue_MissingPath(t *testing.T) {
	svc := &Service{}
	payload := map[string]interface{}{"a": "b"}

	assert.Equal(t, "", svc.extractValue("x.y.z", payload, nil, ""))
}

// =========================================================================
// extractEventType tests (additional providers)
// =========================================================================

func TestExtractEventType_Slack(t *testing.T) {
	payload := []byte(`{"type":"event_callback"}`)
	result := extractEventType(ProviderTypeSlack, payload, map[string]string{})
	assert.Equal(t, "event_callback", result)
}

func TestExtractEventType_Paddle(t *testing.T) {
	payload := []byte(`{"event_type":"subscription.created"}`)
	result := extractEventType(ProviderTypePaddle, payload, map[string]string{})
	assert.Equal(t, "subscription.created", result)
}

func TestExtractEventType_Linear_TypeField(t *testing.T) {
	payload := []byte(`{"type":"Issue"}`)
	result := extractEventType(ProviderTypeLinear, payload, map[string]string{})
	assert.Equal(t, "Issue", result)
}

func TestExtractEventType_Linear_ActionFallback(t *testing.T) {
	payload := []byte(`{"action":"create"}`)
	result := extractEventType(ProviderTypeLinear, payload, map[string]string{})
	assert.Equal(t, "create", result)
}

func TestExtractEventType_Intercom(t *testing.T) {
	payload := []byte(`{"topic":"conversation.user.created"}`)
	result := extractEventType(ProviderTypeIntercom, payload, map[string]string{})
	assert.Equal(t, "conversation.user.created", result)
}

func TestExtractEventType_Discord(t *testing.T) {
	payload := []byte(`{"t":"MESSAGE_CREATE"}`)
	result := extractEventType(ProviderTypeDiscord, payload, map[string]string{})
	assert.Equal(t, "MESSAGE_CREATE", result)
}

func TestExtractEventType_UnknownProviderFallsBackToGeneric(t *testing.T) {
	// Falls through provider switch to generic extraction
	payload := []byte(`{"event":"some.event"}`)
	result := extractEventType("unknown_provider", payload, map[string]string{})
	assert.Equal(t, "some.event", result)
}

func TestExtractEventType_EmptyPayload(t *testing.T) {
	result := extractEventType(ProviderTypeStripe, []byte(`{}`), map[string]string{})
	assert.Equal(t, "", result)
}

func TestExtractEventType_InvalidJSON(t *testing.T) {
	result := extractEventType(ProviderTypeStripe, []byte(`not json`), map[string]string{})
	assert.Equal(t, "", result)
}

// =========================================================================
// GetProviderEndpointURL tests
// =========================================================================

func TestGetProviderEndpointURL(t *testing.T) {
	svc := NewService(new(mockRepository), new(mockPublisher))

	tests := []struct {
		baseURL    string
		tenantID   string
		providerID string
		expected   string
	}{
		{"https://api.example.com", "t1", "p1", "https://api.example.com/gateway/t1/p1"},
		{"http://localhost:8080", "tenant-abc", "prov-xyz", "http://localhost:8080/gateway/tenant-abc/prov-xyz"},
		{"", "t1", "p1", "/gateway/t1/p1"},
	}

	for _, tc := range tests {
		result := svc.GetProviderEndpointURL(tc.baseURL, tc.tenantID, tc.providerID)
		assert.Equal(t, tc.expected, result)
	}
}

// =========================================================================
// Inbound webhook retrieval tests
// =========================================================================

func TestGetInboundWebhook_Found(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	expected := &InboundWebhook{ID: "wh-1", TenantID: "t1", EventType: "push"}
	repo.On("GetInboundWebhook", mock.Anything, "t1", "wh-1").Return(expected, nil)

	webhook, err := svc.GetInboundWebhook(context.Background(), "t1", "wh-1")
	require.NoError(t, err)
	assert.Equal(t, "push", webhook.EventType)
}

func TestGetInboundWebhook_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("GetInboundWebhook", mock.Anything, "t1", "missing").Return(nil, nil)

	webhook, err := svc.GetInboundWebhook(context.Background(), "t1", "missing")
	require.NoError(t, err)
	assert.Nil(t, webhook)
}

func TestListInboundWebhooks_DefaultLimit(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("ListInboundWebhooks", mock.Anything, "t1", "p-1", 20, 0).
		Return([]InboundWebhook{{ID: "wh-1"}}, 1, nil)

	webhooks, total, err := svc.ListInboundWebhooks(context.Background(), "t1", "p-1", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, webhooks, 1)
}

func TestListInboundWebhooks_ExcessiveLimit(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockPublisher))

	repo.On("ListInboundWebhooks", mock.Anything, "t1", "", 100, 0).
		Return([]InboundWebhook{}, 0, nil)

	_, _, err := svc.ListInboundWebhooks(context.Background(), "t1", "", 200, 0)
	require.NoError(t, err)
}

// =========================================================================
// VerifierRegistry tests (extended)
// =========================================================================

func TestVerifierRegistry_Register(t *testing.T) {
	registry := NewVerifierRegistry()
	custom := &mockVerifier{}

	registry.Register("my-custom", custom)

	v := registry.Get("my-custom")
	assert.Equal(t, custom, v)
}

func TestVerifierRegistry_UnknownFallsBackToCustom(t *testing.T) {
	registry := NewVerifierRegistry()

	v1 := registry.Get("totally-unknown")
	v2 := registry.Get(ProviderTypeCustom)

	// Both should be the CustomVerifier
	assert.IsType(t, &CustomVerifier{}, v1)
	assert.IsType(t, &CustomVerifier{}, v2)
}

func TestVerifierRegistry_OverrideExisting(t *testing.T) {
	registry := NewVerifierRegistry()
	custom := &mockVerifier{}

	registry.Register(ProviderTypeStripe, custom)

	v := registry.Get(ProviderTypeStripe)
	assert.Equal(t, custom, v) // should be the mock, not StripeVerifier
}

// =========================================================================
// Verifier provider tests (additional verifiers)
// =========================================================================

func TestLinearVerifier_Valid(t *testing.T) {
	v := &LinearVerifier{}
	secret := "linear-secret"
	payload := []byte(`{"type":"Issue","action":"create"}`)

	sig := computeHMACSHA256(payload, []byte(secret))
	headers := map[string]string{"Linear-Signature": sig}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestLinearVerifier_MissingHeader(t *testing.T) {
	v := &LinearVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing Linear-Signature")
}

func TestPaddleVerifier_Valid(t *testing.T) {
	v := &PaddleVerifier{}
	secret := "paddle-secret"
	payload := []byte(`{"event_type":"subscription.created"}`)
	ts := "1700000000"

	signedPayload := ts + ":" + string(payload)
	sig := computeHMACSHA256([]byte(signedPayload), []byte(secret))

	headers := map[string]string{
		"Paddle-Signature": "ts=" + ts + ";h1=" + sig,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestPaddleVerifier_MissingHeader(t *testing.T) {
	v := &PaddleVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	require.Error(t, err)
}

func TestPaddleVerifier_InvalidFormat(t *testing.T) {
	v := &PaddleVerifier{}
	headers := map[string]string{"Paddle-Signature": "invalid"}
	_, err := v.Verify([]byte("body"), headers, &SignatureConfig{SecretKey: "s"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Paddle-Signature format")
}

func TestIntercomVerifier_Valid(t *testing.T) {
	v := &IntercomVerifier{}
	secret := "intercom-secret"
	payload := []byte(`{"topic":"conversation.user.created"}`)

	sig := "sha1=" + computeHMACSHA1(payload, []byte(secret))
	headers := map[string]string{"X-Hub-Signature": sig}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestIntercomVerifier_InvalidFormat(t *testing.T) {
	v := &IntercomVerifier{}
	headers := map[string]string{"X-Hub-Signature": "md5=abc"}
	_, err := v.Verify([]byte("body"), headers, &SignatureConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature format")
}

func TestDiscordVerifier_Valid(t *testing.T) {
	v := &DiscordVerifier{}
	secret := "discord-secret"
	payload := []byte(`{"t":"MESSAGE_CREATE"}`)
	ts := "1700000000"

	signedPayload := ts + string(payload)
	sig := computeHMACSHA256([]byte(signedPayload), []byte(secret))

	headers := map[string]string{
		"X-Signature-Ed25519":   sig,
		"X-Signature-Timestamp": ts,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestDiscordVerifier_MissingHeaders(t *testing.T) {
	v := &DiscordVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	require.Error(t, err)
}

func TestGitLabVerifier_Valid(t *testing.T) {
	v := &GitLabVerifier{}
	secret := "my-gitlab-token"

	headers := map[string]string{"X-Gitlab-Token": secret}
	valid, err := v.Verify([]byte("body"), headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestGitLabVerifier_InvalidToken(t *testing.T) {
	v := &GitLabVerifier{}
	headers := map[string]string{"X-Gitlab-Token": "wrong"}
	valid, err := v.Verify([]byte("body"), headers, &SignatureConfig{SecretKey: "correct"})
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestGitLabVerifier_MissingHeader(t *testing.T) {
	v := &GitLabVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	require.Error(t, err)
}

func TestBitbucketVerifier_Valid(t *testing.T) {
	v := &BitbucketVerifier{}
	secret := "bb-secret"
	payload := []byte(`{"push":{"changes":[]}}`)

	sig := "sha256=" + computeHMACSHA256(payload, []byte(secret))
	headers := map[string]string{"X-Hub-Signature": sig}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestBitbucketVerifier_InvalidFormat(t *testing.T) {
	v := &BitbucketVerifier{}
	headers := map[string]string{"X-Hub-Signature": "sha1=abc"}
	_, err := v.Verify([]byte("body"), headers, &SignatureConfig{})
	require.Error(t, err)
}

func TestCloudflareVerifier_Valid(t *testing.T) {
	v := &CloudflareVerifier{}
	secret := "cf-webhook-secret"

	headers := map[string]string{"Cf-Webhook-Auth": secret}
	valid, err := v.Verify([]byte("body"), headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestCloudflareVerifier_Invalid(t *testing.T) {
	v := &CloudflareVerifier{}
	headers := map[string]string{"Cf-Webhook-Auth": "wrong"}
	valid, err := v.Verify([]byte("body"), headers, &SignatureConfig{SecretKey: "correct"})
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestSendGridVerifier_Valid(t *testing.T) {
	v := &SendGridVerifier{}
	secret := "sg-secret"
	payload := []byte(`[{"email":"test@example.com","event":"delivered"}]`)
	ts := "1700000000"

	signedPayload := ts + string(payload)
	sig := computeHMACSHA256([]byte(signedPayload), []byte(secret))

	headers := map[string]string{
		"X-Twilio-Email-Event-Webhook-Signature": sig,
		"X-Twilio-Email-Event-Webhook-Timestamp": ts,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestSendGridVerifier_MissingHeader(t *testing.T) {
	v := &SendGridVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	require.Error(t, err)
}

func TestTwilioVerifier_Valid(t *testing.T) {
	v := &TwilioVerifier{}
	secret := "twilio-auth-token"
	payload := []byte(`AccountSid=AC123&Body=Hello`)

	sig := computeHMACSHA1Base64(payload, []byte(secret))
	headers := map[string]string{"X-Twilio-Signature": sig}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestTwilioVerifier_MissingHeader(t *testing.T) {
	v := &TwilioVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	require.Error(t, err)
}

// =========================================================================
// GenericHMACVerifier tests
// =========================================================================

func TestGenericHMACVerifier_Valid(t *testing.T) {
	v := &GenericHMACVerifier{HeaderName: "X-Webhook-Sig"}
	secret := "generic-secret"
	payload := []byte(`{"event":"test"}`)

	sig := computeHMACSHA256(payload, []byte(secret))
	headers := map[string]string{"X-Webhook-Sig": sig}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestGenericHMACVerifier_WithPrefix(t *testing.T) {
	v := &GenericHMACVerifier{HeaderName: "X-Sig", SignaturePrefix: "sha1="}
	secret := "generic-secret"
	payload := []byte(`data`)

	sig := computeHMACSHA256(payload, []byte(secret))
	headers := map[string]string{"X-Sig": "sha1=" + sig}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestGenericHMACVerifier_MissingHeader(t *testing.T) {
	v := &GenericHMACVerifier{HeaderName: "X-Sig"}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing X-Sig header")
}

// =========================================================================
// AutoDetectProvider tests (additional)
// =========================================================================

func TestAutoDetectProvider_UserAgentFallback(t *testing.T) {
	tests := []struct {
		ua       string
		expected string
	}{
		{"GitHub-Hookshot/abc123", ProviderTypeGitHub},
		{"Shopify Webhook/1.0", ProviderTypeShopify},
		{"Stripe/1.0 (+https://stripe.com)", ProviderTypeStripe},
		{"Bitbucket-Webhooks/2.0", ProviderTypeBitbucket},
		{"Zendesk Webhook", ProviderTypeZendesk},
		{"Mozilla/5.0", ""},
	}

	for _, tc := range tests {
		t.Run(tc.ua, func(t *testing.T) {
			result := AutoDetectProvider(map[string]string{"User-Agent": tc.ua})
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAutoDetectProvider_HeaderTakesPrecedenceOverUA(t *testing.T) {
	headers := map[string]string{
		"Stripe-Signature": "t=1,v1=abc",
		"User-Agent":       "GitHub-Hookshot/abc",
	}
	result := AutoDetectProvider(headers)
	assert.Equal(t, ProviderTypeStripe, result)
}

func TestAutoDetectProvider_AllKnownHeaders(t *testing.T) {
	knownHeaders := map[string]string{
		"Stripe-Signature":              ProviderTypeStripe,
		"X-Hub-Signature-256":           ProviderTypeGitHub,
		"X-Shopify-Hmac-Sha256":         ProviderTypeShopify,
		"X-Twilio-Signature":            ProviderTypeTwilio,
		"X-Slack-Signature":             ProviderTypeSlack,
		"Paddle-Signature":              ProviderTypePaddle,
		"Linear-Signature":              ProviderTypeLinear,
		"X-Signature-Ed25519":           ProviderTypeDiscord,
		"X-Gitlab-Token":                ProviderTypeGitLab,
		"X-Zm-Signature":                ProviderTypeZoom,
		"X-Square-Hmacsha256-Signature": ProviderTypeSquare,
		"X-Docusign-Signature-1":        ProviderTypeDocuSign,
		"Typeform-Signature":            ProviderTypeTypeform,
		"X-Pagerduty-Signature":         ProviderTypePagerDuty,
		"X-Zendesk-Webhook-Signature":   ProviderTypeZendesk,
		"X-Hook-Signature":              ProviderTypeAsana,
		"Cf-Webhook-Auth":               ProviderTypeCloudflare,
		"X-Figma-Signature":             ProviderTypeFigma,
		"X-Amz-Ce-Signature":            ProviderTypeAWSEventBridge,
		"X-Salesforce-Signature":        ProviderTypeSalesforce,
		"X-Workday-Signature":           ProviderTypeWorkday,
		"Dd-Webhook-Signature":          ProviderTypeDatadog,
		"X-Ld-Signature":                ProviderTypeLaunchDarkly,
		"Plaid-Verification":            ProviderTypePlaid,
		"Circleci-Signature":            ProviderTypeCircleCI,
		"X-Vercel-Signature":            ProviderTypeVercel,
		"Sentry-Hook-Signature":         ProviderTypeSentry,
		"X-Newrelic-Webhook-Signature":  ProviderTypeNewRelic,
		"X-Mongodb-Signature":           ProviderTypeMongoDB,
		"X-Supabase-Signature":          ProviderTypeSupabase,
		"Auth0-Signature":               ProviderTypeAuth0,
		"X-Notion-Signature":            ProviderTypeNotion,
		"Calendly-Webhook-Signature":    ProviderTypeCalendly,
		"X-Gong-Signature":              ProviderTypeGong,
	}

	for header, expectedProvider := range knownHeaders {
		t.Run(header, func(t *testing.T) {
			result := AutoDetectProvider(map[string]string{header: "some-value"})
			assert.Equal(t, expectedProvider, result)
		})
	}
}

// =========================================================================
// PayloadNormalizer tests (additional)
// =========================================================================

func TestPayloadNormalizer_MinimalPayload(t *testing.T) {
	normalizer := NewPayloadNormalizer()
	payload := map[string]interface{}{}

	result := normalizer.Normalize("test", payload)

	assert.Equal(t, "", result.EventID)
	assert.Equal(t, "", result.EventType)
	assert.Equal(t, "test", result.Source)
}

func TestPayloadNormalizer_WithDataField(t *testing.T) {
	normalizer := NewPayloadNormalizer()
	payload := map[string]interface{}{
		"id":   "evt-1",
		"type": "order.created",
		"data": map[string]interface{}{
			"order_id": "ord-123",
			"amount":   99.99,
		},
	}

	result := normalizer.Normalize("shopify", payload)

	assert.Equal(t, "evt-1", result.EventID)
	assert.Equal(t, "order.created", result.EventType)
	assert.Equal(t, "shopify", result.Source)
	assert.Equal(t, "ord-123", result.Data["order_id"])
}

func TestPayloadNormalizer_TimestampExtraction(t *testing.T) {
	normalizer := NewPayloadNormalizer()

	tests := []struct {
		name     string
		payload  map[string]interface{}
		expected string
	}{
		{"timestamp field", map[string]interface{}{"timestamp": "2024-01-01T00:00:00Z"}, "2024-01-01T00:00:00Z"},
		{"created_at field", map[string]interface{}{"created_at": "2024-06-15"}, "2024-06-15"},
		{"created field", map[string]interface{}{"created": "1700000000"}, "1700000000"},
		{"time field", map[string]interface{}{"time": "now"}, "now"},
		{"occurred_at field", map[string]interface{}{"occurred_at": "2024-12-25"}, "2024-12-25"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizer.Normalize("test", tc.payload)
			assert.Equal(t, tc.expected, result.Timestamp)
		})
	}
}

func TestPayloadNormalizer_NonMapDataField(t *testing.T) {
	normalizer := NewPayloadNormalizer()
	payload := map[string]interface{}{
		"type": "test",
		"data": "not-a-map", // string, not map
	}

	result := normalizer.Normalize("test", payload)
	// When "data" key exists but isn't a map, Data stays as the initialized empty map
	assert.Equal(t, map[string]interface{}{}, result.Data)
}

func TestPayloadNormalizer_NoDataField(t *testing.T) {
	normalizer := NewPayloadNormalizer()
	payload := map[string]interface{}{
		"type":  "test",
		"extra": "value",
	}

	result := normalizer.Normalize("test", payload)
	// When no "data" key at all, the entire payload becomes Data
	assert.Equal(t, payload, result.Data)
}

func TestPayloadNormalizer_RawFieldPreserved(t *testing.T) {
	normalizer := NewPayloadNormalizer()
	payload := map[string]interface{}{
		"id":     "evt-1",
		"type":   "test",
		"extra":  "field",
		"nested": map[string]interface{}{"key": "val"},
	}

	result := normalizer.Normalize("provider", payload)
	assert.Equal(t, payload, result.Raw)
}

// =========================================================================
// CommunityProviderRegistry tests (additional)
// =========================================================================

func TestCommunityProviderRegistry_DefaultAlgorithm(t *testing.T) {
	registry := NewCommunityProviderRegistry()

	err := registry.Register(&ProviderDefinition{
		Name: "MyProvider",
		Type: "my_provider",
	})
	require.NoError(t, err)

	def, ok := registry.Get("my_provider")
	require.True(t, ok)
	assert.Equal(t, "hmac-sha256", def.Algorithm)
}

func TestCommunityProviderRegistry_GetNonExistent(t *testing.T) {
	registry := NewCommunityProviderRegistry()

	_, ok := registry.Get("nonexistent")
	assert.False(t, ok)
}

func TestCommunityProviderRegistry_OverrideExisting(t *testing.T) {
	registry := NewCommunityProviderRegistry()

	require.NoError(t, registry.Register(&ProviderDefinition{
		Name: "V1", Type: "my_type", SignatureHeader: "X-Sig-V1",
	}))
	require.NoError(t, registry.Register(&ProviderDefinition{
		Name: "V2", Type: "my_type", SignatureHeader: "X-Sig-V2",
	}))

	def, ok := registry.Get("my_type")
	require.True(t, ok)
	assert.Equal(t, "V2", def.Name)
	assert.Equal(t, "X-Sig-V2", def.SignatureHeader)
}

func TestCommunityProviderRegistry_ListMultiple(t *testing.T) {
	registry := NewCommunityProviderRegistry()

	for i := 0; i < 5; i++ {
		require.NoError(t, registry.Register(&ProviderDefinition{
			Name: fmt.Sprintf("Provider %d", i),
			Type: fmt.Sprintf("type_%d", i),
		}))
	}

	list := registry.List()
	assert.Len(t, list, 5)
}

func TestCommunityProviderRegistry_ToVerifier(t *testing.T) {
	registry := NewCommunityProviderRegistry()

	def := &ProviderDefinition{
		Name:            "Custom",
		Type:            "custom_saas",
		SignatureHeader: "X-Custom-Sig",
		SignaturePrefix: "v1=",
	}

	verifier := registry.ToVerifier(def)
	require.NotNil(t, verifier)

	hmacV, ok := verifier.(*GenericHMACVerifier)
	require.True(t, ok)
	assert.Equal(t, "X-Custom-Sig", hmacV.HeaderName)
	assert.Equal(t, "v1=", hmacV.SignaturePrefix)
}

func TestCommunityProviderRegistry_RequiresNameAndType(t *testing.T) {
	registry := NewCommunityProviderRegistry()

	tests := []struct {
		name string
		def  *ProviderDefinition
		err  bool
	}{
		{"missing both", &ProviderDefinition{}, true},
		{"missing type", &ProviderDefinition{Name: "Test"}, true},
		{"missing name", &ProviderDefinition{Type: "test"}, true},
		{"both present", &ProviderDefinition{Name: "Test", Type: "test"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := registry.Register(tc.def)
			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =========================================================================
// routeToDestination tests
// =========================================================================

func TestRouteToDestination_EndpointSuccess(t *testing.T) {
	pub := new(mockPublisher)
	svc := &Service{publisher: pub}

	pub.On("Publish", mock.Anything, "t1", "ep-1", mock.Anything, mock.Anything).
		Return("del-1", nil)

	rule := &RoutingRule{ID: "r-1", Name: "Test rule"}
	dest := &RoutingDestination{Type: "endpoint", EndpointID: "ep-1"}

	result := svc.routeToDestination(context.Background(), "t1", []byte(`{}`), map[string]string{}, rule, dest)

	assert.Equal(t, "queued", result.Status)
	assert.Equal(t, "del-1", result.DeliveryID)
	assert.Equal(t, "ep-1", result.Target)
	assert.Equal(t, "r-1", result.RuleID)
	assert.Equal(t, "Test rule", result.RuleName)
}

func TestRouteToDestination_EndpointError(t *testing.T) {
	pub := new(mockPublisher)
	svc := &Service{publisher: pub}

	pub.On("Publish", mock.Anything, "t1", "ep-1", mock.Anything, mock.Anything).
		Return("", errors.New("connection refused"))

	rule := &RoutingRule{ID: "r-1", Name: "Rule"}
	dest := &RoutingDestination{Type: "endpoint", EndpointID: "ep-1"}

	result := svc.routeToDestination(context.Background(), "t1", []byte(`{}`), nil, rule, dest)

	assert.Equal(t, "failed", result.Status)
	assert.Contains(t, result.Error, "connection refused")
}

func TestRouteToDestination_UnsupportedType(t *testing.T) {
	svc := &Service{}

	rule := &RoutingRule{ID: "r-1", Name: "Rule"}
	dest := &RoutingDestination{Type: "sns", QueueName: "my-topic"}

	result := svc.routeToDestination(context.Background(), "t1", []byte(`{}`), nil, rule, dest)

	assert.Equal(t, "failed", result.Status)
	assert.Contains(t, result.Error, "unsupported destination type: sns")
}

// =========================================================================
// Edge case: overlapping conditions in multiple rules
// =========================================================================

func TestProcessInboundWebhook_OverlappingRules(t *testing.T) {
	repo := new(mockRepository)
	pub := new(mockPublisher)
	svc := newTestService(repo, pub)

	provider := &Provider{ID: "p-1", TenantID: "t1", Type: ProviderTypeCustom, IsActive: true}
	repo.On("GetProvider", mock.Anything, "t1", "p-1").Return(provider, nil)
	repo.On("SaveInboundWebhook", mock.Anything, mock.Anything).Return(nil)

	// Two rules that both match "order.created"
	rules := []RoutingRule{
		{
			ID: "r-1", Name: "Catch-all", IsActive: true, Priority: 100,
			Conditions:   mustJSON([]RoutingCondition{}), // matches everything
			Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-all"}}),
		},
		{
			ID: "r-2", Name: "Orders only", IsActive: true, Priority: 1,
			Conditions:   mustJSON([]RoutingCondition{{Field: "event_type", Operator: "contains", Value: "order"}}),
			Destinations: mustJSON([]RoutingDestination{{Type: "endpoint", EndpointID: "ep-orders"}}),
		},
	}
	repo.On("ListRoutingRules", mock.Anything, "t1", "p-1").Return(rules, nil)
	pub.On("Publish", mock.Anything, "t1", mock.Anything, mock.Anything, mock.Anything).
		Return("del-x", nil)

	result, err := svc.ProcessInboundWebhook(context.Background(), "t1", "p-1",
		[]byte(`{"type":"order.created"}`), map[string]string{})

	require.NoError(t, err)
	// Both rules should fire (fanout)
	assert.Equal(t, 2, result.TotalRouted)
	assert.Len(t, result.Destinations, 2)
}

// =========================================================================
// Edge case: regex conditions
// =========================================================================

func TestMatchesConditions_RegexComplexPattern(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}

	tests := []struct {
		name    string
		pattern string
		value   string
		match   bool
	}{
		{"dot-separated event", `^[a-z]+\.[a-z]+$`, "order.created", true},
		{"no match uppercase", `^[a-z]+\.[a-z]+$`, "Order.Created", false},
		{"wildcard", `.*\.updated$`, "customer.updated", true},
		{"alternation", `^(order|payment)\..+`, "payment.success", true},
		{"alternation no match", `^(order|payment)\..+`, "user.created", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conds := mustJSON([]RoutingCondition{
				{Field: "event_type", Operator: "matches", Value: tc.pattern},
			})
			result := svc.matchesConditions([]byte(`{}`), map[string]string{}, tc.value, conds)
			assert.Equal(t, tc.match, result)
		})
	}
}

// =========================================================================
// Handler construction test
// =========================================================================

func TestNewHandler(t *testing.T) {
	svc := NewService(new(mockRepository), new(mockPublisher))
	handler := NewHandler(svc, "https://api.example.com")

	require.NotNil(t, handler)
	assert.Equal(t, "https://api.example.com", handler.baseURL)
}

// =========================================================================
// NormalizationRule struct test
// =========================================================================

func TestNewPayloadNormalizer_DefaultRules(t *testing.T) {
	normalizer := NewPayloadNormalizer()
	assert.NotNil(t, normalizer)
	assert.True(t, len(normalizer.rules) > 0, "should have default normalization rules")
}
