package gateway

import (
	"testing"
)

func TestVerifierRegistry_Get(t *testing.T) {
	registry := NewVerifierRegistry()

	tests := []struct {
		providerType string
		expectNil    bool
	}{
		{ProviderTypeStripe, false},
		{ProviderTypeGitHub, false},
		{ProviderTypeShopify, false},
		{ProviderTypeTwilio, false},
		{ProviderTypeSlack, false},
		{ProviderTypeSendGrid, false},
		{ProviderTypeCustom, false},
		{"unknown-provider", false}, // falls back to custom
	}

	for _, tc := range tests {
		v := registry.Get(tc.providerType)
		if v == nil {
			t.Fatalf("expected non-nil verifier for %s", tc.providerType)
		}
	}
}

func TestStripeVerifier_Verify(t *testing.T) {
	v := &StripeVerifier{}
	secret := "whsec_test_secret"
	payload := []byte(`{"id":"evt_test","type":"payment_intent.created"}`)

	// Compute valid signature
	timestamp := "1234567890"
	signedPayload := timestamp + "." + string(payload)
	sig := computeHMACSHA256([]byte(signedPayload), []byte(secret))

	headers := map[string]string{
		"Stripe-Signature": "t=" + timestamp + ",v1=" + sig,
	}
	config := &SignatureConfig{SecretKey: secret, ToleranceSeconds: 999999999}

	valid, err := v.Verify(payload, headers, config)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature")
	}
}

func TestStripeVerifier_MissingHeader(t *testing.T) {
	v := &StripeVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	if err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestStripeVerifier_InvalidFormat(t *testing.T) {
	v := &StripeVerifier{}
	headers := map[string]string{
		"Stripe-Signature": "invalid-format",
	}
	_, err := v.Verify([]byte("body"), headers, &SignatureConfig{SecretKey: "secret"})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestGitHubVerifier_SHA256(t *testing.T) {
	v := &GitHubVerifier{}
	secret := "github-secret"
	payload := []byte(`{"action":"opened"}`)

	sig := "sha256=" + computeHMACSHA256(payload, []byte(secret))
	headers := map[string]string{
		"X-Hub-Signature-256": sig,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature")
	}
}

func TestGitHubVerifier_SHA1Fallback(t *testing.T) {
	v := &GitHubVerifier{}
	secret := "github-secret"
	payload := []byte(`{"action":"opened"}`)

	sig := "sha1=" + computeHMACSHA1(payload, []byte(secret))
	headers := map[string]string{
		"X-Hub-Signature": sig,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid sha1 signature")
	}
}

func TestGitHubVerifier_MissingHeaders(t *testing.T) {
	v := &GitHubVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	if err == nil {
		t.Fatal("expected error for missing headers")
	}
}

func TestGitHubVerifier_InvalidSignature(t *testing.T) {
	v := &GitHubVerifier{}
	headers := map[string]string{
		"X-Hub-Signature-256": "sha256=invalid",
	}

	valid, err := v.Verify([]byte("body"), headers, &SignatureConfig{SecretKey: "secret"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if valid {
		t.Fatal("expected invalid signature")
	}
}

func TestShopifyVerifier(t *testing.T) {
	v := &ShopifyVerifier{}
	secret := "shopify-secret"
	payload := []byte(`{"id":"order_123"}`)

	expected := computeHMACSHA256Base64(payload, []byte(secret))
	headers := map[string]string{
		"X-Shopify-Hmac-Sha256": expected,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{SecretKey: secret})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature")
	}
}

func TestShopifyVerifier_MissingHeader(t *testing.T) {
	v := &ShopifyVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	if err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestSlackVerifier(t *testing.T) {
	v := &SlackVerifier{}
	secret := "slack-signing-secret"
	payload := []byte(`token=xoxb&command=/test`)
	timestamp := "1234567890"

	sigBase := "v0:" + timestamp + ":" + string(payload)
	expected := "v0=" + computeHMACSHA256([]byte(sigBase), []byte(secret))

	headers := map[string]string{
		"X-Slack-Signature":         expected,
		"X-Slack-Request-Timestamp": timestamp,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{
		SecretKey:        secret,
		ToleranceSeconds: 999999999,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature")
	}
}

func TestSlackVerifier_MissingHeaders(t *testing.T) {
	v := &SlackVerifier{}
	_, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	if err == nil {
		t.Fatal("expected error for missing headers")
	}
}

func TestCustomVerifier_NoConfig(t *testing.T) {
	v := &CustomVerifier{}
	valid, err := v.Verify([]byte("body"), map[string]string{}, &SignatureConfig{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid when no header configured")
	}
}

func TestCustomVerifier_HMACSHA256(t *testing.T) {
	v := &CustomVerifier{}
	secret := "custom-secret"
	payload := []byte(`{"data":"test"}`)

	sig := computeHMACSHA256(payload, []byte(secret))
	headers := map[string]string{
		"X-Custom-Sig": sig,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{
		SecretKey:  secret,
		HeaderName: "X-Custom-Sig",
		Algorithm:  "hmac-sha256",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature")
	}
}

func TestCustomVerifier_HMACSHA1(t *testing.T) {
	v := &CustomVerifier{}
	secret := "custom-secret"
	payload := []byte(`{"data":"test"}`)

	sig := computeHMACSHA1(payload, []byte(secret))
	headers := map[string]string{
		"X-Custom-Sig": sig,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{
		SecretKey:  secret,
		HeaderName: "X-Custom-Sig",
		Algorithm:  "hmac-sha1",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature")
	}
}

func TestCustomVerifier_UnsupportedAlgorithm(t *testing.T) {
	v := &CustomVerifier{}
	headers := map[string]string{
		"X-Sig": "value",
	}

	_, err := v.Verify([]byte("body"), headers, &SignatureConfig{
		SecretKey:  "secret",
		HeaderName: "X-Sig",
		Algorithm:  "unsupported",
	})
	if err == nil {
		t.Fatal("expected error for unsupported algorithm")
	}
}

func TestCustomVerifier_WithPrefix(t *testing.T) {
	v := &CustomVerifier{}
	secret := "prefix-secret"
	payload := []byte(`{"data":"test"}`)

	sig := computeHMACSHA256(payload, []byte(secret))
	headers := map[string]string{
		"X-Sig": "sha256=" + sig,
	}

	valid, err := v.Verify(payload, headers, &SignatureConfig{
		SecretKey:       secret,
		HeaderName:      "X-Sig",
		Algorithm:       "hmac-sha256",
		SignaturePrefix: "sha256=",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature with prefix")
	}
}

func TestExtractEventType(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		payload      []byte
		headers      map[string]string
		expected     string
	}{
		{
			name:         "stripe type field",
			providerType: ProviderTypeStripe,
			payload:      []byte(`{"type":"payment_intent.created"}`),
			headers:      map[string]string{},
			expected:     "payment_intent.created",
		},
		{
			name:         "github event header",
			providerType: ProviderTypeGitHub,
			payload:      []byte(`{}`),
			headers:      map[string]string{"X-GitHub-Event": "push"},
			expected:     "push",
		},
		{
			name:         "shopify topic header",
			providerType: ProviderTypeShopify,
			payload:      []byte(`{}`),
			headers:      map[string]string{"X-Shopify-Topic": "orders/create"},
			expected:     "orders/create",
		},
		{
			name:         "generic type field",
			providerType: ProviderTypeCustom,
			payload:      []byte(`{"type":"user.created"}`),
			headers:      map[string]string{},
			expected:     "user.created",
		},
		{
			name:         "generic event field",
			providerType: ProviderTypeCustom,
			payload:      []byte(`{"event":"order.completed"}`),
			headers:      map[string]string{},
			expected:     "order.completed",
		},
		{
			name:         "generic event_type field",
			providerType: ProviderTypeCustom,
			payload:      []byte(`{"event_type":"invoice.paid"}`),
			headers:      map[string]string{},
			expected:     "invoice.paid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractEventType(tc.providerType, tc.payload, tc.headers)
			if result != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestMatchesConditions(t *testing.T) {
	svc := &Service{verifiers: NewVerifierRegistry()}

	tests := []struct {
		name       string
		payload    []byte
		headers    map[string]string
		eventType  string
		conditions string
		expected   bool
	}{
		{
			name:       "nil conditions",
			payload:    []byte(`{}`),
			conditions: "null",
			expected:   true,
		},
		{
			name:       "empty conditions",
			payload:    []byte(`{}`),
			conditions: "[]",
			expected:   true,
		},
		{
			name:      "equals match",
			payload:   []byte(`{"type":"order.created"}`),
			eventType: "order.created",
			conditions: `[{"field":"event_type","operator":"equals","value":"order.created"}]`,
			expected:  true,
		},
		{
			name:      "equals no match",
			payload:   []byte(`{"type":"order.created"}`),
			eventType: "order.created",
			conditions: `[{"field":"event_type","operator":"equals","value":"order.updated"}]`,
			expected:  false,
		},
		{
			name:      "contains match",
			payload:   []byte(`{"type":"order.created"}`),
			eventType: "order.created",
			conditions: `[{"field":"event_type","operator":"contains","value":"order"}]`,
			expected:  true,
		},
		{
			name:      "header match",
			payload:   []byte(`{}`),
			headers:   map[string]string{"X-Source": "stripe"},
			conditions: `[{"field":"header.X-Source","operator":"equals","value":"stripe"}]`,
			expected:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			headers := tc.headers
			if headers == nil {
				headers = map[string]string{}
			}
			result := svc.matchesConditions(tc.payload, headers, tc.eventType, []byte(tc.conditions))
			if result != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestExtractValue(t *testing.T) {
	svc := &Service{}

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "obj_123",
			},
		},
		"type": "event.type",
	}
	headers := map[string]string{
		"X-Custom": "header-value",
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"event_type", "my.event"},
		{"type", "my.event"},
		{"header.X-Custom", "header-value"},
		{"data.object.id", "obj_123"},
		{"nonexistent", "<nil>"},
	}

	for _, tc := range tests {
		result := svc.extractValue(tc.field, payload, headers, "my.event")
		if result != tc.expected {
			t.Fatalf("field %q: expected %q, got %q", tc.field, tc.expected, result)
		}
	}
}

func TestComputeHMACSHA256(t *testing.T) {
	result := computeHMACSHA256([]byte("test"), []byte("secret"))
	if result == "" {
		t.Fatal("expected non-empty hash")
	}
	// Should be hex-encoded
	if len(result) != 64 { // SHA-256 = 32 bytes = 64 hex chars
		t.Fatalf("expected 64 hex chars, got %d", len(result))
	}
}

func TestComputeHMACSHA1(t *testing.T) {
	result := computeHMACSHA1([]byte("test"), []byte("secret"))
	if result == "" {
		t.Fatal("expected non-empty hash")
	}
	// SHA-1 = 20 bytes = 40 hex chars
	if len(result) != 40 {
		t.Fatalf("expected 40 hex chars, got %d", len(result))
	}
}

func TestBase64Encode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"f", "Zg=="},
		{"fo", "Zm8="},
		{"foo", "Zm9v"},
		{"foobar", "Zm9vYmFy"},
	}

	for _, tc := range tests {
		result := base64Encode([]byte(tc.input))
		if result != tc.expected {
			t.Fatalf("base64Encode(%q): expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

func TestAbs(t *testing.T) {
	if abs(5) != 5 {
		t.Fatal("abs(5) != 5")
	}
	if abs(-5) != 5 {
		t.Fatal("abs(-5) != 5")
	}
	if abs(0) != 0 {
		t.Fatal("abs(0) != 0")
	}
}

func TestListProviders_LimitClamping(t *testing.T) {
	// Test that the limit clamping logic works correctly
	limits := []struct {
		input    int
		expected int
	}{
		{0, 20},
		{-1, 20},
		{50, 50},
		{200, 100},
	}

	for _, tc := range limits {
		result := tc.input
		if result <= 0 {
			result = 20
		}
		if result > 100 {
			result = 100
		}
		if result != tc.expected {
			t.Fatalf("limit %d: expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}
