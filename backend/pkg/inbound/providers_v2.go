package inbound

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Extended provider constants for v2 gateway
const (
	ProviderPayPal    = "paypal"
	ProviderSquare    = "square"
	ProviderIntercom  = "intercom"
	ProviderZendesk   = "zendesk"
	ProviderHubSpot   = "hubspot"
	ProviderJira      = "jira"
	ProviderLinear    = "linear"
	ProviderPagerDuty = "pagerduty"
	ProviderDatadog   = "datadog"
	ProviderSentry    = "sentry"
	ProviderVercel    = "vercel"
	ProviderClerk     = "clerk"
)

// AllProviders returns all supported provider names
func AllProviders() []string {
	return []string{
		ProviderStripe, ProviderGitHub, ProviderTwilio, ProviderShopify,
		ProviderSlack, ProviderSendGrid, ProviderCustom,
		ProviderPayPal, ProviderSquare, ProviderIntercom, ProviderZendesk,
		ProviderHubSpot, ProviderJira, ProviderLinear, ProviderPagerDuty,
		ProviderDatadog, ProviderSentry, ProviderVercel, ProviderClerk,
	}
}

// GetVerifierV2 returns the appropriate verifier for v2 providers
func GetVerifierV2(provider string) SignatureVerifier {
	switch provider {
	// Original providers handled by GetVerifier
	case ProviderStripe, ProviderGitHub, ProviderTwilio, ProviderShopify,
		ProviderSlack, ProviderSendGrid, ProviderCustom:
		return GetVerifier(provider)
	// New v2 providers
	case ProviderPayPal:
		return &PayPalVerifier{}
	case ProviderSquare:
		return &SquareVerifier{}
	case ProviderIntercom:
		return &IntercomVerifier{}
	case ProviderZendesk:
		return &ZendeskVerifier{}
	case ProviderHubSpot:
		return &HubSpotVerifier{}
	case ProviderJira:
		return &JiraVerifier{}
	case ProviderLinear:
		return &LinearVerifier{}
	case ProviderPagerDuty:
		return &PagerDutyVerifier{}
	case ProviderDatadog:
		return &DatadogVerifier{}
	case ProviderSentry:
		return &SentryVerifier{}
	case ProviderVercel:
		return &VercelVerifier{}
	case ProviderClerk:
		return &ClerkVerifier{}
	default:
		return &CustomVerifier{}
	}
}

// PayPalVerifier validates PayPal webhook signatures
// PayPal uses HMAC-SHA256 with transmission-id + timestamp + webhook-id + crc32
type PayPalVerifier struct{}

func (v *PayPalVerifier) ProviderName() string { return ProviderPayPal }

func (v *PayPalVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "Paypal-Transmission-Sig")
	if sigHeader == "" {
		return false, fmt.Errorf("missing Paypal-Transmission-Sig header")
	}

	transmissionID := getHeader(headers, "Paypal-Transmission-Id")
	transmissionTime := getHeader(headers, "Paypal-Transmission-Time")

	if transmissionID == "" || transmissionTime == "" {
		return false, fmt.Errorf("missing PayPal transmission headers")
	}

	// Simplified HMAC verification (full verification requires PayPal cert)
	message := fmt.Sprintf("%s|%s|%s|%s", transmissionID, transmissionTime, secret, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// SquareVerifier validates Square webhook signatures (X-Square-Hmacsha256-Signature)
type SquareVerifier struct{}

func (v *SquareVerifier) ProviderName() string { return ProviderSquare }

func (v *SquareVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Square-Hmacsha256-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Square-Hmacsha256-Signature header")
	}

	// Square uses HMAC-SHA256, base64 encoded: HMAC(notification_url + body)
	notifURL := getHeader(headers, "X-Square-Notification-Url")
	signedPayload := notifURL + string(payload)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// IntercomVerifier validates Intercom webhook signatures (X-Hub-Signature)
type IntercomVerifier struct{}

func (v *IntercomVerifier) ProviderName() string { return ProviderIntercom }

func (v *IntercomVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Hub-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Hub-Signature header")
	}

	if !strings.HasPrefix(sigHeader, "sha1=") {
		return false, fmt.Errorf("invalid X-Hub-Signature format")
	}
	signature := strings.TrimPrefix(sigHeader, "sha1=")

	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

// ZendeskVerifier validates Zendesk webhook signatures (X-Zendesk-Webhook-Signature)
type ZendeskVerifier struct{}

func (v *ZendeskVerifier) ProviderName() string { return ProviderZendesk }

func (v *ZendeskVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Zendesk-Webhook-Signature")
	timestamp := getHeader(headers, "X-Zendesk-Webhook-Signature-Timestamp")

	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Zendesk-Webhook-Signature header")
	}
	if timestamp == "" {
		return false, fmt.Errorf("missing X-Zendesk-Webhook-Signature-Timestamp header")
	}

	// Zendesk: HMAC-SHA256 of timestamp + body
	signedPayload := timestamp + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// HubSpotVerifier validates HubSpot webhook signatures (X-HubSpot-Signature-v3)
type HubSpotVerifier struct{}

func (v *HubSpotVerifier) ProviderName() string { return ProviderHubSpot }

func (v *HubSpotVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-HubSpot-Signature-V3")
	timestamp := getHeader(headers, "X-HubSpot-Request-Timestamp")

	if sigHeader == "" {
		// Fallback to v2
		return v.verifyV2(payload, headers, secret)
	}

	if timestamp == "" {
		return false, fmt.Errorf("missing X-HubSpot-Request-Timestamp header")
	}

	// v3: HMAC-SHA256 of method + url + body + timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid timestamp")
	}
	if time.Now().UnixMilli()-ts > 300000 { // 5 min tolerance
		return false, fmt.Errorf("hubspot signature timestamp too old")
	}

	requestMethod := getHeader(headers, "X-HubSpot-Request-Method")
	if requestMethod == "" {
		requestMethod = "POST"
	}
	requestURI := getHeader(headers, "X-HubSpot-Request-Uri")

	message := requestMethod + requestURI + string(payload) + timestamp
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

func (v *HubSpotVerifier) verifyV2(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-HubSpot-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing HubSpot signature header")
	}

	// v2: SHA-256(secret + body)
	h := sha256.New()
	h.Write([]byte(secret))
	h.Write(payload)
	expected := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// JiraVerifier validates Jira/Atlassian webhook signatures
type JiraVerifier struct{}

func (v *JiraVerifier) ProviderName() string { return ProviderJira }

func (v *JiraVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Hub-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Hub-Signature header")
	}

	// Atlassian uses sha256= prefix with HMAC-SHA256
	var signature string
	if strings.HasPrefix(sigHeader, "sha256=") {
		signature = strings.TrimPrefix(sigHeader, "sha256=")
	} else {
		signature = sigHeader
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

// LinearVerifier validates Linear webhook signatures (Linear-Signature)
type LinearVerifier struct{}

func (v *LinearVerifier) ProviderName() string { return ProviderLinear }

func (v *LinearVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "Linear-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing Linear-Signature header")
	}

	// Linear uses HMAC-SHA256, hex encoded
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// PagerDutyVerifier validates PagerDuty webhook signatures (X-PagerDuty-Signature)
type PagerDutyVerifier struct{}

func (v *PagerDutyVerifier) ProviderName() string { return ProviderPagerDuty }

func (v *PagerDutyVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-PagerDuty-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing X-PagerDuty-Signature header")
	}

	// PagerDuty v3: HMAC-SHA256 with "v1=" prefix
	var signature string
	for _, part := range strings.Split(sigHeader, ",") {
		if strings.HasPrefix(part, "v1=") {
			signature = strings.TrimPrefix(part, "v1=")
			break
		}
	}
	if signature == "" {
		signature = sigHeader
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

// DatadogVerifier validates Datadog webhook signatures (DD-WEBHOOK-SIGNATURE)
type DatadogVerifier struct{}

func (v *DatadogVerifier) ProviderName() string { return ProviderDatadog }

func (v *DatadogVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "DD-Webhook-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing DD-Webhook-Signature header")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// SentryVerifier validates Sentry webhook signatures (Sentry-Hook-Signature)
type SentryVerifier struct{}

func (v *SentryVerifier) ProviderName() string { return ProviderSentry }

func (v *SentryVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "Sentry-Hook-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing Sentry-Hook-Signature header")
	}

	// Sentry uses HMAC-SHA256, hex encoded
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// VercelVerifier validates Vercel webhook signatures (x-vercel-signature)
type VercelVerifier struct{}

func (v *VercelVerifier) ProviderName() string { return ProviderVercel }

func (v *VercelVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "X-Vercel-Signature")
	if sigHeader == "" {
		return false, fmt.Errorf("missing X-Vercel-Signature header")
	}

	// Vercel uses HMAC-SHA1, hex encoded
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sigHeader)), nil
}

// ClerkVerifier validates Clerk webhook signatures (svix-signature)
type ClerkVerifier struct{}

func (v *ClerkVerifier) ProviderName() string { return ProviderClerk }

func (v *ClerkVerifier) Verify(payload []byte, headers map[string][]string, secret string) (bool, error) {
	sigHeader := getHeader(headers, "Svix-Signature")
	msgID := getHeader(headers, "Svix-Id")
	timestamp := getHeader(headers, "Svix-Timestamp")

	if sigHeader == "" || msgID == "" || timestamp == "" {
		return false, fmt.Errorf("missing Svix signature headers (Svix-Id, Svix-Timestamp, Svix-Signature)")
	}

	// Validate timestamp (5 min tolerance)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid Svix-Timestamp")
	}
	if abs64(time.Now().Unix()-ts) > 300 {
		return false, fmt.Errorf("svix signature timestamp out of tolerance")
	}

	// Svix/Clerk: HMAC-SHA256 of "msg_id.timestamp.body", base64 encoded
	// Secret is base64-encoded, strip "whsec_" prefix
	secretKey := secret
	if strings.HasPrefix(secretKey, "whsec_") {
		secretKey = strings.TrimPrefix(secretKey, "whsec_")
	}
	decodedSecret, err := base64.StdEncoding.DecodeString(secretKey)
	if err != nil {
		decodedSecret = []byte(secretKey)
	}

	toSign := fmt.Sprintf("%s.%s.%s", msgID, timestamp, string(payload))
	mac := hmac.New(sha256.New, decodedSecret)
	mac.Write([]byte(toSign))
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Svix-Signature may contain multiple signatures separated by space
	for _, sig := range strings.Split(sigHeader, " ") {
		cleanSig := sig
		if strings.HasPrefix(cleanSig, "v1,") {
			cleanSig = strings.TrimPrefix(cleanSig, "v1,")
		}
		if hmac.Equal([]byte(expectedSig), []byte(cleanSig)) {
			return true, nil
		}
	}

	return false, nil
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
