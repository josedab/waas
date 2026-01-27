package inbound

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Provider Signature Verifier Tests
// ============================================================

func TestStripeVerifier_Verify(t *testing.T) {
	verifier := &StripeVerifier{}
	assert.Equal(t, ProviderStripe, verifier.ProviderName())

	secret := "whsec_test_secret"
	payload := []byte(`{"type":"payment_intent.succeeded"}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Compute valid signature
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"Stripe-Signature": {fmt.Sprintf("t=%s,v1=%s", timestamp, sig)},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestStripeVerifier_MissingHeader(t *testing.T) {
	verifier := &StripeVerifier{}
	valid, err := verifier.Verify([]byte("test"), map[string][]string{}, "secret")
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestStripeVerifier_InvalidSignature(t *testing.T) {
	verifier := &StripeVerifier{}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	headers := map[string][]string{
		"Stripe-Signature": {fmt.Sprintf("t=%s,v1=invalidsig", timestamp)},
	}

	valid, err := verifier.Verify([]byte("test"), headers, "secret")
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestGitHubVerifier_Verify(t *testing.T) {
	verifier := &GitHubVerifier{}
	assert.Equal(t, ProviderGitHub, verifier.ProviderName())

	secret := "github_secret"
	payload := []byte(`{"action":"push"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Hub-Signature-256": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestGitHubVerifier_MissingHeader(t *testing.T) {
	verifier := &GitHubVerifier{}
	valid, err := verifier.Verify([]byte("test"), map[string][]string{}, "secret")
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestGitHubVerifier_InvalidFormat(t *testing.T) {
	verifier := &GitHubVerifier{}
	headers := map[string][]string{
		"X-Hub-Signature-256": {"invalid_format"},
	}
	valid, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestTwilioVerifier_Verify(t *testing.T) {
	verifier := &TwilioVerifier{}
	assert.Equal(t, ProviderTwilio, verifier.ProviderName())

	secret := "twilio_auth_token"
	payload := []byte("https://example.com/twilioBodyParam1=value1")

	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Twilio-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestShopifyVerifier_Verify(t *testing.T) {
	verifier := &ShopifyVerifier{}
	assert.Equal(t, ProviderShopify, verifier.ProviderName())

	secret := "shopify_secret"
	payload := []byte(`{"id":1234}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Shopify-Hmac-Sha256": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestSlackVerifier_Verify(t *testing.T) {
	verifier := &SlackVerifier{}
	assert.Equal(t, ProviderSlack, verifier.ProviderName())

	secret := "slack_signing_secret"
	payload := []byte(`{"type":"event_callback"}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(baseString))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Slack-Signature":         {sig},
		"X-Slack-Request-Timestamp": {timestamp},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestSlackVerifier_MissingTimestamp(t *testing.T) {
	verifier := &SlackVerifier{}
	headers := map[string][]string{
		"X-Slack-Signature": {"v0=abc123"},
	}
	valid, err := verifier.Verify([]byte("test"), headers, "secret")
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestSendGridVerifier_BasicAuth(t *testing.T) {
	verifier := &SendGridVerifier{}
	assert.Equal(t, ProviderSendGrid, verifier.ProviderName())

	secret := "user:password"
	authValue := "Basic " + base64.StdEncoding.EncodeToString([]byte(secret))

	headers := map[string][]string{
		"Authorization": {authValue},
	}

	valid, err := verifier.Verify([]byte("test"), headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestCustomVerifier_Verify(t *testing.T) {
	verifier := &CustomVerifier{}
	assert.Equal(t, ProviderCustom, verifier.ProviderName())

	secret := "custom_secret"
	payload := []byte(`{"event":"test"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Webhook-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestCustomVerifier_CustomHeader(t *testing.T) {
	verifier := &CustomVerifier{
		HeaderName: "X-My-Signature",
		Algorithm:  "hmac-sha256",
	}

	secret := "my_secret"
	payload := []byte(`test`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-My-Signature": {sig},
	}

	valid, err := verifier.Verify(payload, headers, secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestGetVerifier(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{ProviderStripe, ProviderStripe},
		{ProviderGitHub, ProviderGitHub},
		{ProviderTwilio, ProviderTwilio},
		{ProviderShopify, ProviderShopify},
		{ProviderSlack, ProviderSlack},
		{ProviderSendGrid, ProviderSendGrid},
		{ProviderCustom, ProviderCustom},
		{"unknown", ProviderCustom},
	}

	for _, tt := range tests {
		v := GetVerifier(tt.provider)
		assert.Equal(t, tt.expected, v.ProviderName())
	}
}

func TestGetHeader_CaseInsensitive(t *testing.T) {
	headers := map[string][]string{
		"X-Hub-Signature-256": {"test_value"},
	}

	assert.Equal(t, "test_value", getHeader(headers, "x-hub-signature-256"))
	assert.Equal(t, "test_value", getHeader(headers, "X-Hub-Signature-256"))
	assert.Equal(t, "", getHeader(headers, "X-Missing"))
}

// ============================================================
// Service Tests (with mock repository)
// ============================================================

type mockRepository struct {
	sources map[string]*InboundSource
	events  map[string]*InboundEvent
	rules   map[string][]RoutingRule
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		sources: make(map[string]*InboundSource),
		events:  make(map[string]*InboundEvent),
		rules:   make(map[string][]RoutingRule),
	}
}

func (m *mockRepository) CreateSource(_ context.Context, source *InboundSource) error {
	m.sources[source.ID] = source
	return nil
}

func (m *mockRepository) GetSource(_ context.Context, sourceID string) (*InboundSource, error) {
	s, ok := m.sources[sourceID]
	if !ok {
		return nil, fmt.Errorf("source not found")
	}
	return s, nil
}

func (m *mockRepository) GetSourceByTenant(_ context.Context, tenantID, sourceID string) (*InboundSource, error) {
	s, ok := m.sources[sourceID]
	if !ok || s.TenantID != tenantID {
		return nil, fmt.Errorf("source not found")
	}
	return s, nil
}

func (m *mockRepository) ListSources(_ context.Context, tenantID string, limit, offset int) ([]InboundSource, int, error) {
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

func (m *mockRepository) UpdateSource(_ context.Context, source *InboundSource) error {
	m.sources[source.ID] = source
	return nil
}

func (m *mockRepository) DeleteSource(_ context.Context, tenantID, sourceID string) error {
	s, ok := m.sources[sourceID]
	if !ok || s.TenantID != tenantID {
		return fmt.Errorf("source not found")
	}
	delete(m.sources, sourceID)
	return nil
}

func (m *mockRepository) CreateRoutingRule(_ context.Context, rule *RoutingRule) error {
	m.rules[rule.SourceID] = append(m.rules[rule.SourceID], *rule)
	return nil
}

func (m *mockRepository) GetRoutingRules(_ context.Context, sourceID string) ([]RoutingRule, error) {
	return m.rules[sourceID], nil
}

func (m *mockRepository) UpdateRoutingRule(_ context.Context, rule *RoutingRule) error {
	return nil
}

func (m *mockRepository) DeleteRoutingRule(_ context.Context, ruleID string) error {
	return nil
}

func (m *mockRepository) CreateEvent(_ context.Context, event *InboundEvent) error {
	m.events[event.ID] = event
	return nil
}

func (m *mockRepository) GetEvent(_ context.Context, eventID string) (*InboundEvent, error) {
	e, ok := m.events[eventID]
	if !ok {
		return nil, fmt.Errorf("event not found")
	}
	return e, nil
}

func (m *mockRepository) GetEventByTenant(_ context.Context, tenantID, eventID string) (*InboundEvent, error) {
	e, ok := m.events[eventID]
	if !ok || e.TenantID != tenantID {
		return nil, fmt.Errorf("event not found")
	}
	return e, nil
}

func (m *mockRepository) ListEventsBySource(_ context.Context, sourceID, status string, limit, offset int) ([]InboundEvent, int, error) {
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

func (m *mockRepository) UpdateEventStatus(_ context.Context, eventID, status, errorMsg string) error {
	if e, ok := m.events[eventID]; ok {
		e.Status = status
		e.ErrorMessage = errorMsg
	}
	return nil
}

func (m *mockRepository) GetDLQEntries(_ context.Context, tenantID string, limit, offset int) ([]InboundDLQEntry, error) {
	return []InboundDLQEntry{}, nil
}

func (m *mockRepository) GetDLQEntry(_ context.Context, tenantID, entryID string) (*InboundDLQEntry, error) {
	return nil, fmt.Errorf("DLQ entry not found")
}

func (m *mockRepository) MarkDLQEntryReplayed(_ context.Context, entryID string) error {
	return nil
}

func (m *mockRepository) GetProviderHealth(_ context.Context, sourceID string) (*ProviderHealth, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockRepository) GetRateLimitConfig(_ context.Context, sourceID string) (*RateLimitConfig, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockRepository) GetInboundStats(_ context.Context, sourceID string) (*InboundStats, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestService_CreateSource(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	source, err := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:               "Stripe Webhooks",
		Provider:           ProviderStripe,
		VerificationSecret: "whsec_test",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, source.ID)
	assert.Equal(t, "tenant-1", source.TenantID)
	assert.Equal(t, "Stripe Webhooks", source.Name)
	assert.Equal(t, ProviderStripe, source.Provider)
	assert.Equal(t, SourceStatusActive, source.Status)
	assert.Equal(t, "hmac-sha256", source.VerificationAlgorithm)
}

func TestService_CreateSource_EmptyName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Provider: ProviderGitHub,
	})
	assert.Error(t, err)
}

func TestService_ListSources(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, _ = svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Source 1",
		Provider: ProviderStripe,
	})
	_, _ = svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Source 2",
		Provider: ProviderGitHub,
	})

	sources, total, err := svc.ListSources(context.Background(), "tenant-1", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, sources, 2)
}

func TestService_UpdateSource(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Original",
		Provider: ProviderStripe,
	})

	updated, err := svc.UpdateSource(context.Background(), "tenant-1", source.ID, &UpdateSourceRequest{
		Name:   "Updated Name",
		Status: SourceStatusPaused,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, SourceStatusPaused, updated.Status)
}

func TestService_UpdateSource_InvalidStatus(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Test",
		Provider: ProviderStripe,
	})

	_, err := svc.UpdateSource(context.Background(), "tenant-1", source.ID, &UpdateSourceRequest{
		Status: "invalid_status",
	})
	assert.Error(t, err)
}

func TestService_DeleteSource(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "To Delete",
		Provider: ProviderStripe,
	})

	err := svc.DeleteSource(context.Background(), "tenant-1", source.ID)
	require.NoError(t, err)

	_, err = svc.GetSource(context.Background(), "tenant-1", source.ID)
	assert.Error(t, err)
}

func TestService_ProcessInboundWebhook_NoSecret(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "No Secret Source",
		Provider: ProviderCustom,
	})

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"test":true}`), map[string][]string{})
	require.NoError(t, err)
	assert.Equal(t, EventStatusRouted, event.Status)
	assert.True(t, event.SignatureValid)
}

func TestService_ProcessInboundWebhook_ValidSignature(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	secret := "github_secret"
	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:               "GitHub Source",
		Provider:           ProviderGitHub,
		VerificationSecret: secret,
	})

	payload := []byte(`{"action":"push"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	headers := map[string][]string{
		"X-Hub-Signature-256": {sig},
	}

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, payload, headers)
	require.NoError(t, err)
	assert.True(t, event.SignatureValid)
	assert.Equal(t, EventStatusRouted, event.Status)
}

func TestService_ProcessInboundWebhook_InvalidSignature(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:               "GitHub Source",
		Provider:           ProviderGitHub,
		VerificationSecret: "correct_secret",
	})

	headers := map[string][]string{
		"X-Hub-Signature-256": {"sha256=invalid"},
	}

	event, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte("test"), headers)
	assert.Error(t, err)
	assert.NotNil(t, event)
	assert.False(t, event.SignatureValid)
	assert.Equal(t, EventStatusFailed, event.Status)
}

func TestService_ProcessInboundWebhook_InactiveSource(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Paused Source",
		Provider: ProviderStripe,
	})

	// Pause the source
	_, _ = svc.UpdateSource(context.Background(), "tenant-1", source.ID, &UpdateSourceRequest{
		Status: SourceStatusPaused,
	})

	_, err := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte("test"), map[string][]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestService_ProcessInboundWebhook_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.ProcessInboundWebhook(context.Background(), "nonexistent", []byte("test"), map[string][]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source not found")
}

func TestService_ReplayInboundEvent(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	source, _ := svc.CreateSource(context.Background(), "tenant-1", &CreateSourceRequest{
		Name:     "Replay Source",
		Provider: ProviderCustom,
	})

	event, _ := svc.ProcessInboundWebhook(context.Background(), source.ID, []byte(`{"test":true}`), map[string][]string{})

	replayed, err := svc.ReplayInboundEvent(context.Background(), "tenant-1", event.ID)
	require.NoError(t, err)
	assert.Equal(t, EventStatusRouted, replayed.Status)
}

func TestService_ReplayInboundEvent_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.ReplayInboundEvent(context.Background(), "tenant-1", "nonexistent")
	assert.Error(t, err)
}
