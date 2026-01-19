package gateway

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SignatureVerifier verifies webhook signatures
type SignatureVerifier interface {
	Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error)
}

// VerifierRegistry holds all provider verifiers
type VerifierRegistry struct {
	verifiers map[string]SignatureVerifier
}

// NewVerifierRegistry creates a new verifier registry with all built-in verifiers
func NewVerifierRegistry() *VerifierRegistry {
	registry := &VerifierRegistry{
		verifiers: make(map[string]SignatureVerifier),
	}

	// Register built-in verifiers
	registry.Register(ProviderTypeStripe, &StripeVerifier{})
	registry.Register(ProviderTypeGitHub, &GitHubVerifier{})
	registry.Register(ProviderTypeShopify, &ShopifyVerifier{})
	registry.Register(ProviderTypeTwilio, &TwilioVerifier{})
	registry.Register(ProviderTypeSlack, &SlackVerifier{})
	registry.Register(ProviderTypeSendGrid, &SendGridVerifier{})
	registry.Register(ProviderTypeCustom, &CustomVerifier{})

	return registry
}

// Register registers a verifier for a provider type
func (r *VerifierRegistry) Register(providerType string, verifier SignatureVerifier) {
	r.verifiers[providerType] = verifier
}

// Get returns the verifier for a provider type
func (r *VerifierRegistry) Get(providerType string) SignatureVerifier {
	if v, ok := r.verifiers[providerType]; ok {
		return v
	}
	return r.verifiers[ProviderTypeCustom]
}

// StripeVerifier verifies Stripe webhook signatures
type StripeVerifier struct{}

func (v *StripeVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["Stripe-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing Stripe-Signature header")
	}

	// Parse the signature header
	parts := strings.Split(signature, ",")
	var timestamp string
	var sig string

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sig = kv[1]
		}
	}

	if timestamp == "" || sig == "" {
		return false, fmt.Errorf("invalid Stripe-Signature format")
	}

	// Check timestamp tolerance
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid timestamp")
	}

	tolerance := config.ToleranceSeconds
	if tolerance == 0 {
		tolerance = 300 // 5 minutes default
	}

	if time.Now().Unix()-ts > int64(tolerance) {
		return false, fmt.Errorf("timestamp too old")
	}

	// Compute expected signature
	signedPayload := timestamp + "." + string(payload)
	expected := computeHMACSHA256([]byte(signedPayload), []byte(config.SecretKey))

	return hmac.Equal([]byte(sig), []byte(expected)), nil
}

// GitHubVerifier verifies GitHub webhook signatures
type GitHubVerifier struct{}

func (v *GitHubVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Hub-Signature-256"]
	if signature == "" {
		// Fallback to SHA-1
		signature = headers["X-Hub-Signature"]
		if signature == "" {
			return false, fmt.Errorf("missing signature header")
		}
		return v.verifySHA1(payload, signature, config.SecretKey)
	}

	return v.verifySHA256(payload, signature, config.SecretKey)
}

func (v *GitHubVerifier) verifySHA256(payload []byte, signature, secret string) (bool, error) {
	if !strings.HasPrefix(signature, "sha256=") {
		return false, fmt.Errorf("invalid signature format")
	}

	sig := strings.TrimPrefix(signature, "sha256=")
	expected := computeHMACSHA256(payload, []byte(secret))

	return hmac.Equal([]byte(sig), []byte(expected)), nil
}

func (v *GitHubVerifier) verifySHA1(payload []byte, signature, secret string) (bool, error) {
	if !strings.HasPrefix(signature, "sha1=") {
		return false, fmt.Errorf("invalid signature format")
	}

	sig := strings.TrimPrefix(signature, "sha1=")
	expected := computeHMACSHA1(payload, []byte(secret))

	return hmac.Equal([]byte(sig), []byte(expected)), nil
}

// ShopifyVerifier verifies Shopify webhook signatures
type ShopifyVerifier struct{}

func (v *ShopifyVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Shopify-Hmac-Sha256"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Shopify-Hmac-Sha256 header")
	}

	expected := computeHMACSHA256Base64(payload, []byte(config.SecretKey))

	return signature == expected, nil
}

// TwilioVerifier verifies Twilio webhook signatures
type TwilioVerifier struct{}

func (v *TwilioVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Twilio-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Twilio-Signature header")
	}

	// Twilio uses a different signature scheme with URL + sorted params
	// For simplicity, using basic HMAC-SHA1 verification
	expected := computeHMACSHA1Base64(payload, []byte(config.SecretKey))

	return signature == expected, nil
}

// SlackVerifier verifies Slack webhook signatures
type SlackVerifier struct{}

func (v *SlackVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Slack-Signature"]
	timestamp := headers["X-Slack-Request-Timestamp"]

	if signature == "" || timestamp == "" {
		return false, fmt.Errorf("missing required Slack headers")
	}

	// Check timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid timestamp")
	}

	tolerance := config.ToleranceSeconds
	if tolerance == 0 {
		tolerance = 300
	}

	if abs(time.Now().Unix()-ts) > int64(tolerance) {
		return false, fmt.Errorf("timestamp too old")
	}

	// Compute signature
	sigBaseString := fmt.Sprintf("v0:%s:%s", timestamp, string(payload))
	expected := "v0=" + computeHMACSHA256([]byte(sigBaseString), []byte(config.SecretKey))

	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// SendGridVerifier verifies SendGrid webhook signatures
type SendGridVerifier struct{}

func (v *SendGridVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Twilio-Email-Event-Webhook-Signature"]
	timestamp := headers["X-Twilio-Email-Event-Webhook-Timestamp"]

	if signature == "" {
		return false, fmt.Errorf("missing signature header")
	}

	// SendGrid uses ECDSA signatures, simplified to HMAC for this implementation
	signedPayload := timestamp + string(payload)
	expected := computeHMACSHA256([]byte(signedPayload), []byte(config.SecretKey))

	return signature == expected, nil
}

// CustomVerifier verifies custom webhook signatures
type CustomVerifier struct{}

func (v *CustomVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	if config.HeaderName == "" {
		return true, nil // No signature verification configured
	}

	signature := headers[config.HeaderName]
	if signature == "" {
		return false, fmt.Errorf("missing signature header: %s", config.HeaderName)
	}

	// Strip prefix if configured
	if config.SignaturePrefix != "" {
		signature = strings.TrimPrefix(signature, config.SignaturePrefix)
	}

	// Compute expected signature based on algorithm
	var expected string
	switch config.Algorithm {
	case "hmac-sha256", "":
		expected = computeHMACSHA256(payload, []byte(config.SecretKey))
	case "hmac-sha1":
		expected = computeHMACSHA1(payload, []byte(config.SecretKey))
	default:
		return false, fmt.Errorf("unsupported algorithm: %s", config.Algorithm)
	}

	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// Helper functions

func computeHMACSHA256(payload, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func computeHMACSHA1(payload, secret []byte) string {
	mac := hmac.New(sha1.New, secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func computeHMACSHA256Base64(payload, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return base64Encode(mac.Sum(nil))
}

func computeHMACSHA1Base64(payload, secret []byte) string {
	mac := hmac.New(sha1.New, secret)
	mac.Write(payload)
	return base64Encode(mac.Sum(nil))
}

func base64Encode(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, 0, (len(data)+2)/3*4)

	for i := 0; i < len(data); i += 3 {
		var b uint32
		n := 0
		for j := 0; j < 3 && i+j < len(data); j++ {
			b = (b << 8) | uint32(data[i+j])
			n++
		}
		b <<= (3 - n) * 8

		result = append(result, base64Chars[(b>>18)&0x3F])
		result = append(result, base64Chars[(b>>12)&0x3F])
		if n > 1 {
			result = append(result, base64Chars[(b>>6)&0x3F])
		} else {
			result = append(result, '=')
		}
		if n > 2 {
			result = append(result, base64Chars[b&0x3F])
		} else {
			result = append(result, '=')
		}
	}

	return string(result)
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
