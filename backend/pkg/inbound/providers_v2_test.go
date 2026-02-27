package inbound

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAllProviders(t *testing.T) {
	providers := AllProviders()
	assert.GreaterOrEqual(t, len(providers), 19, "should have at least 19 providers")
	assert.Contains(t, providers, ProviderPayPal)
	assert.Contains(t, providers, ProviderClerk)
	assert.Contains(t, providers, ProviderVercel)
}

func TestGetVerifierV2_AllProviders(t *testing.T) {
	for _, provider := range AllProviders() {
		v := GetVerifierV2(provider)
		assert.NotNil(t, v, "verifier should not be nil for %s", provider)
		assert.NotEmpty(t, v.ProviderName())
	}
}

func TestLinearVerifier(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"action":"create","type":"Issue"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	v := &LinearVerifier{}
	headers := map[string][]string{
		"Linear-Signature": {sig},
	}

	valid, err := v.Verify(payload, headers, secret)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestSentryVerifier(t *testing.T) {
	secret := "sentry-secret"
	payload := []byte(`{"event":"issue.created"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	v := &SentryVerifier{}
	headers := map[string][]string{
		"Sentry-Hook-Signature": {sig},
	}

	valid, err := v.Verify(payload, headers, secret)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestClerkVerifier(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("clerk-test-secret"))
	payload := []byte(`{"type":"user.created"}`)
	msgID := "msg_test123"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	toSign := fmt.Sprintf("%s.%s.%s", msgID, timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte("clerk-test-secret"))
	mac.Write([]byte(toSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	v := &ClerkVerifier{}
	headers := map[string][]string{
		"Svix-Id":        {msgID},
		"Svix-Timestamp": {timestamp},
		"Svix-Signature": {"v1," + sig},
	}

	valid, err := v.Verify(payload, headers, secret)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestZendeskVerifier(t *testing.T) {
	secret := "zendesk-secret"
	payload := []byte(`{"ticket_id":"123"}`)
	timestamp := "2026-01-01T00:00:00Z"

	signedPayload := timestamp + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	v := &ZendeskVerifier{}
	headers := map[string][]string{
		"X-Zendesk-Webhook-Signature":           {sig},
		"X-Zendesk-Webhook-Signature-Timestamp": {timestamp},
	}

	valid, err := v.Verify(payload, headers, secret)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifierV2_MissingHeaders(t *testing.T) {
	providers := []struct {
		name     string
		verifier SignatureVerifier
	}{
		{"paypal", &PayPalVerifier{}},
		{"square", &SquareVerifier{}},
		{"intercom", &IntercomVerifier{}},
		{"zendesk", &ZendeskVerifier{}},
		{"hubspot", &HubSpotVerifier{}},
		{"jira", &JiraVerifier{}},
		{"linear", &LinearVerifier{}},
		{"pagerduty", &PagerDutyVerifier{}},
		{"datadog", &DatadogVerifier{}},
		{"sentry", &SentryVerifier{}},
		{"vercel", &VercelVerifier{}},
		{"clerk", &ClerkVerifier{}},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			_, err := p.verifier.Verify([]byte("test"), map[string][]string{}, "secret")
			assert.Error(t, err, "should error on missing headers for %s", p.name)
		})
	}
}
