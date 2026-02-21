package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifierRegistry_Has50PlusProviders(t *testing.T) {
	registry := NewVerifierRegistry()
	// Count registered verifiers
	count := 0
	allProviders := []string{
		ProviderTypeStripe, ProviderTypeGitHub, ProviderTypeShopify, ProviderTypeTwilio,
		ProviderTypeSlack, ProviderTypeSendGrid, ProviderTypePaddle, ProviderTypeLinear,
		ProviderTypeIntercom, ProviderTypeDiscord, ProviderTypeCustom, ProviderTypeGitLab,
		ProviderTypeBitbucket, ProviderTypeZoom, ProviderTypeSquare, ProviderTypeHubSpot,
		ProviderTypeMailgun, ProviderTypeDocuSign, ProviderTypeTypeform, ProviderTypeJira,
		ProviderTypePagerDuty, ProviderTypeZendesk, ProviderTypeAsana, ProviderTypeCloudflare,
		ProviderTypeFigma, ProviderTypeAWSEventBridge, ProviderTypeAzureEventGrid,
		ProviderTypeGooglePubSub, ProviderTypeSalesforce, ProviderTypeWorkday,
		ProviderTypeDatadog, ProviderTypeLaunchDarkly, ProviderTypePlaid, ProviderTypeCircleCI,
		ProviderTypeVercel, ProviderTypeNetlify, ProviderTypeSentry, ProviderTypeNewRelic,
		ProviderTypeMongoDB, ProviderTypeSupabase, ProviderTypeAuth0, ProviderTypeOkta,
		ProviderTypeTrello, ProviderTypeClickUp, ProviderTypeNotion, ProviderTypeAirtable,
		ProviderTypeSegment, ProviderTypeBraze, ProviderTypeContentful, ProviderTypeSanity,
		ProviderTypeCalendly, ProviderTypeGong,
	}
	for _, p := range allProviders {
		v := registry.Get(p)
		if v != nil {
			count++
		}
	}
	assert.True(t, count >= 50, "expected 50+ providers, got %d", count)
}

func TestAutoDetectProvider(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{"Stripe", map[string]string{"Stripe-Signature": "t=123,v1=abc"}, ProviderTypeStripe},
		{"GitHub", map[string]string{"X-Hub-Signature-256": "sha256=abc"}, ProviderTypeGitHub},
		{"Shopify", map[string]string{"X-Shopify-Hmac-Sha256": "abc"}, ProviderTypeShopify},
		{"Slack", map[string]string{"X-Slack-Signature": "v0=abc"}, ProviderTypeSlack},
		{"AWS EventBridge", map[string]string{"X-Amz-Ce-Signature": "abc"}, ProviderTypeAWSEventBridge},
		{"Datadog", map[string]string{"Dd-Webhook-Signature": "abc"}, ProviderTypeDatadog},
		{"Vercel", map[string]string{"X-Vercel-Signature": "abc"}, ProviderTypeVercel},
		{"Sentry", map[string]string{"Sentry-Hook-Signature": "abc"}, ProviderTypeSentry},
		{"Unknown", map[string]string{"X-Random": "abc"}, ""},
		{"GitHub UA", map[string]string{"User-Agent": "GitHub-Hookshot/abc"}, ProviderTypeGitHub},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AutoDetectProvider(tt.headers)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommunityProviderRegistry(t *testing.T) {
	registry := NewCommunityProviderRegistry()

	err := registry.Register(&ProviderDefinition{
		Name:            "Custom SaaS",
		Type:            "custom_saas",
		SignatureHeader: "X-Custom-Sig",
		Algorithm:       "hmac-sha256",
	})
	require.NoError(t, err)

	def, ok := registry.Get("custom_saas")
	assert.True(t, ok)
	assert.Equal(t, "Custom SaaS", def.Name)

	all := registry.List()
	assert.Equal(t, 1, len(all))

	verifier := registry.ToVerifier(def)
	assert.NotNil(t, verifier)
}

func TestCommunityProviderRegistry_InvalidRegistration(t *testing.T) {
	registry := NewCommunityProviderRegistry()
	err := registry.Register(&ProviderDefinition{})
	assert.Error(t, err)
}

func TestPayloadNormalizer(t *testing.T) {
	normalizer := NewPayloadNormalizer()

	payload := map[string]interface{}{
		"id":   "evt_123",
		"type": "payment.completed",
		"data": map[string]interface{}{
			"amount": 2500,
		},
		"created_at": "2024-01-01T00:00:00Z",
	}

	result := normalizer.Normalize("stripe", payload)
	assert.Equal(t, "evt_123", result.EventID)
	assert.Equal(t, "payment.completed", result.EventType)
	assert.Equal(t, "stripe", result.Source)
	assert.Equal(t, "2024-01-01T00:00:00Z", result.Timestamp)
	assert.NotNil(t, result.Data)
}

func TestPayloadNormalizer_EventTypeVariants(t *testing.T) {
	normalizer := NewPayloadNormalizer()

	tests := []struct {
		name    string
		payload map[string]interface{}
		expected string
	}{
		{"type field", map[string]interface{}{"type": "invoice.paid"}, "invoice.paid"},
		{"event field", map[string]interface{}{"event": "user.created"}, "user.created"},
		{"event_type field", map[string]interface{}{"event_type": "order.shipped"}, "order.shipped"},
		{"action field", map[string]interface{}{"action": "push"}, "push"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize("test", tt.payload)
			assert.Equal(t, tt.expected, result.EventType)
		})
	}
}

func TestExtendedProviderVerifiers(t *testing.T) {
	registry := NewVerifierRegistry()

	// Test that all extended providers have verifiers
	extendedProviders := []string{
		ProviderTypeAWSEventBridge, ProviderTypeAzureEventGrid, ProviderTypeSalesforce,
		ProviderTypeWorkday, ProviderTypeDatadog, ProviderTypeLaunchDarkly,
		ProviderTypeCircleCI, ProviderTypeVercel, ProviderTypeNetlify,
		ProviderTypeSentry, ProviderTypeNewRelic, ProviderTypeMongoDB,
		ProviderTypeSupabase, ProviderTypeAuth0, ProviderTypeOkta,
	}

	for _, p := range extendedProviders {
		v := registry.Get(p)
		assert.NotNil(t, v, "verifier for %s should not be nil", p)
	}
}
