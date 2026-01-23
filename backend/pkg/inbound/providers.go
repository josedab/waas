package inbound

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"
)

// SignatureVerifier validates inbound webhook signatures for a provider
type SignatureVerifier interface {
	Verify(payload []byte, headers map[string][]string, secret string) (bool, error)
	ProviderName() string
}

// GetVerifier returns the appropriate signature verifier for a provider
func GetVerifier(provider string) SignatureVerifier {
	switch provider {
	case ProviderStripe:
		return &StripeVerifier{}
	case ProviderGitHub:
		return &GitHubVerifier{}
	case ProviderTwilio:
		return &TwilioVerifier{}
	case ProviderShopify:
		return &ShopifyVerifier{}
	case ProviderSlack:
		return &SlackVerifier{}
	case ProviderSendGrid:
		return &SendGridVerifier{}
	case ProviderCustom:
		return &CustomVerifier{}
	default:
		return &CustomVerifier{}
	}
}

// StripeVerifier validates Stripe webhook signatures (v1 with timestamp)
type StripeVerifier struct{}

func (v *StripeVerifier) ProviderName() string { return ProviderStripe }

func (v *StripeVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "Stripe-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing Stripe-Signature header")
	}

	// Parse the signature header: t=timestamp,v1=signature
	parts := strings.Split(sigHeader, ",")
	var timestamp, signature string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signature = kv[1]
		}
	}

	if timestamp == "" || signature == "" {
		return false, fmt.Errorf("invalid Stripe-Signature format")
	}

	// Validate timestamp is within tolerance (5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid timestamp in Stripe-Signature")
	}
	if time.Now().Unix()-ts > 300 {
		return false, fmt.Errorf("stripe signature timestamp too old")
	}

	// Compute expected signature: HMAC-SHA256 of "timestamp.payload"
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

// GitHubVerifier validates GitHub webhook signatures (X-Hub-Signature-256)
type GitHubVerifier struct{}

func (v *GitHubVerifier) ProviderName() string { return ProviderGitHub }

func (v *GitHubVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Hub-Signature-256")
	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Hub-Signature-256 header")
	}

	if !strings.HasPrefix(sigHeader, "sha256=") {
		return false, fmt.Errorf("invalid X-Hub-Signature-256 format")
	}
	signature := strings.TrimPrefix(sigHeader, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

// TwilioVerifier validates Twilio webhook signatures (X-Twilio-Signature)
type TwilioVerifier struct{}

func (v *TwilioVerifier) ProviderName() string { return ProviderTwilio }

func (v *TwilioVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Twilio-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Twilio-Signature header")
	}

	// Twilio uses HMAC-SHA1, base64-encoded
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// ShopifyVerifier validates Shopify webhook signatures (X-Shopify-Hmac-Sha256)
type ShopifyVerifier struct{}

func (v *ShopifyVerifier) ProviderName() string { return ProviderShopify }

func (v *ShopifyVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Shopify-Hmac-Sha256")
	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Shopify-Hmac-Sha256 header")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// SlackVerifier validates Slack webhook signatures (X-Slack-Signature with timestamp)
type SlackVerifier struct{}

func (v *SlackVerifier) ProviderName() string { return ProviderSlack }

func (v *SlackVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Slack-Signature")
	timestamp := getHeader(headers, "X-Slack-Request-Timestamp")

	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Slack-Signature header")
	}
	if timestamp == "" {
		return false, fmt.Errorf("missing X-Slack-Request-Timestamp header")
	}

	// Validate timestamp is within tolerance (5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid X-Slack-Request-Timestamp")
	}
	if time.Now().Unix()-ts > 300 {
		return false, fmt.Errorf("slack signature timestamp too old")
	}

	if !strings.HasPrefix(sigHeader, "v0=") {
		return false, fmt.Errorf("invalid X-Slack-Signature format")
	}
	signature := strings.TrimPrefix(sigHeader, "v0=")

	// sig_basestring = "v0:timestamp:body"
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(baseString))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

// SendGridVerifier validates SendGrid webhooks (basic auth verification)
type SendGridVerifier struct{}

func (v *SendGridVerifier) ProviderName() string { return ProviderSendGrid }

func (v *SendGridVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	// SendGrid uses a verification key for Event Webhook signature
	sigHeader := getHeader(headers, "X-Twilio-Email-Event-Webhook-Signature")
	if sigHeader == "" {
		// Fallback: check basic auth
		authHeader := getHeader(headers, "Authorization")
		if authHeader == "" {
			return false, fmt.Errorf("missing signature or authorization header")
		}
		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte(secret))
		return hmac.Equal([]byte(expected), []byte(authHeader)), nil
	}

	// HMAC-SHA256 verification
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// CustomVerifier validates webhooks using configurable HMAC header
type CustomVerifier struct {
	HeaderName string
	Algorithm  string
}

func (v *CustomVerifier) ProviderName() string { return ProviderCustom }

func (v *CustomVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	headerName := v.HeaderName
	if headerName == "" {
		headerName = "X-Webhook-Signature"
	}

	sigHeader := getHeader(headers, headerName)
	if sigHeader == "" {
		return false, fmt.Errorf("missing %s header", headerName)
	}

	algorithm := v.Algorithm
	if algorithm == "" {
		algorithm = "hmac-sha256"
	}

	var hashFunc func() hash.Hash
	switch algorithm {
	case "hmac-sha256":
		hashFunc = sha256.New
	case "hmac-sha1":
		hashFunc = sha1.New
	default:
		hashFunc = sha256.New
	}

	mac := hmac.New(hashFunc, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	// Try hex comparison first
	if hmac.Equal([]byte(expected), []byte(sigHeader)) {
		return true, nil
	}

	// Try base64 comparison
	expectedB64 := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedB64), []byte(sigHeader)), nil
}

// getHeader retrieves a header value case-insensitively
func getHeader(headers map[string][]string, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}
