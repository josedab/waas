package gateway

import (
	"crypto/hmac"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GitLabVerifier verifies GitLab webhook signatures
type GitLabVerifier struct{}

func (v *GitLabVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	token := headers["X-Gitlab-Token"]
	if token == "" {
		return false, fmt.Errorf("missing X-Gitlab-Token header")
	}
	return hmac.Equal([]byte(token), []byte(config.SecretKey)), nil
}

// BitbucketVerifier verifies Bitbucket webhook signatures
type BitbucketVerifier struct{}

func (v *BitbucketVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Hub-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Hub-Signature header")
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false, fmt.Errorf("invalid signature format")
	}
	sig := strings.TrimPrefix(signature, "sha256=")
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(sig), []byte(expected)), nil
}

// ZoomVerifier verifies Zoom webhook signatures
type ZoomVerifier struct{}

func (v *ZoomVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Zm-Signature"]
	timestamp := headers["X-Zm-Request-Timestamp"]
	if signature == "" || timestamp == "" {
		return false, fmt.Errorf("missing Zoom signature headers")
	}

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

	msg := fmt.Sprintf("v0:%s:%s", timestamp, string(payload))
	expected := "v0=" + computeHMACSHA256([]byte(msg), []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// SquareVerifier verifies Square webhook signatures
type SquareVerifier struct{}

func (v *SquareVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Square-Hmacsha256-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Square-Hmacsha256-Signature header")
	}
	notificationURL := headers["X-Square-Notification-Url"]
	signedPayload := notificationURL + string(payload)
	expected := computeHMACSHA256Base64([]byte(signedPayload), []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// HubSpotVerifier verifies HubSpot webhook signatures (v3)
type HubSpotVerifier struct{}

func (v *HubSpotVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Hubspot-Signature-V3"]
	timestamp := headers["X-Hubspot-Request-Timestamp"]
	if signature == "" {
		// Fallback to v2
		signature = headers["X-Hubspot-Signature"]
		if signature == "" {
			return false, fmt.Errorf("missing HubSpot signature header")
		}
		expected := computeHMACSHA256(payload, []byte(config.SecretKey))
		return hmac.Equal([]byte(signature), []byte(expected)), nil
	}

	if timestamp != "" {
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err == nil {
			tolerance := config.ToleranceSeconds
			if tolerance == 0 {
				tolerance = 300
			}
			if abs(time.Now().UnixMilli()-ts) > int64(tolerance)*1000 {
				return false, fmt.Errorf("timestamp too old")
			}
		}
	}

	// v3: HMAC-SHA256 of method + URL + body + timestamp
	msg := string(payload) + timestamp
	expected := computeHMACSHA256Base64([]byte(msg), []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// MailgunVerifier verifies Mailgun webhook signatures
type MailgunVerifier struct{}

func (v *MailgunVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Mailgun-Signature"]
	timestamp := headers["X-Mailgun-Timestamp"]
	token := headers["X-Mailgun-Token"]

	if signature == "" || timestamp == "" || token == "" {
		return false, fmt.Errorf("missing Mailgun signature headers")
	}

	msg := timestamp + token
	expected := computeHMACSHA256([]byte(msg), []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// DocuSignVerifier verifies DocuSign webhook signatures
type DocuSignVerifier struct{}

func (v *DocuSignVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Docusign-Signature-1"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Docusign-Signature-1 header")
	}
	expected := computeHMACSHA256Base64(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// TypeformVerifier verifies Typeform webhook signatures
type TypeformVerifier struct{}

func (v *TypeformVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["Typeform-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing Typeform-Signature header")
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false, fmt.Errorf("invalid signature format")
	}
	sig := strings.TrimPrefix(signature, "sha256=")
	expected := computeHMACSHA256Base64(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(sig), []byte(expected)), nil
}

// JiraVerifier verifies Jira/Atlassian webhook signatures
type JiraVerifier struct{}

func (v *JiraVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Hub-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Hub-Signature header")
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false, fmt.Errorf("invalid signature format")
	}
	sig := strings.TrimPrefix(signature, "sha256=")
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(sig), []byte(expected)), nil
}

// PagerDutyVerifier verifies PagerDuty webhook signatures (v3)
type PagerDutyVerifier struct{}

func (v *PagerDutyVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Pagerduty-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Pagerduty-Signature header")
	}

	signatures := strings.Split(signature, ",")
	expected := "v1=" + computeHMACSHA256(payload, []byte(config.SecretKey))

	for _, sig := range signatures {
		if hmac.Equal([]byte(strings.TrimSpace(sig)), []byte(expected)) {
			return true, nil
		}
	}
	return false, nil
}

// ZendeskVerifier verifies Zendesk webhook signatures
type ZendeskVerifier struct{}

func (v *ZendeskVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Zendesk-Webhook-Signature"]
	timestamp := headers["X-Zendesk-Webhook-Signature-Timestamp"]
	if signature == "" || timestamp == "" {
		return false, fmt.Errorf("missing Zendesk signature headers")
	}

	msg := timestamp + string(payload)
	expected := computeHMACSHA256Base64([]byte(msg), []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// AsanaVerifier verifies Asana webhook signatures
type AsanaVerifier struct{}

func (v *AsanaVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Hook-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Hook-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// CloudflareVerifier verifies Cloudflare webhook signatures
type CloudflareVerifier struct{}

func (v *CloudflareVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["Cf-Webhook-Auth"]
	if signature == "" {
		return false, fmt.Errorf("missing Cf-Webhook-Auth header")
	}
	return hmac.Equal([]byte(signature), []byte(config.SecretKey)), nil
}

// FigmaVerifier verifies Figma webhook signatures
type FigmaVerifier struct{}

func (v *FigmaVerifier) Verify(payload []byte, headers map[string]string, config *SignatureConfig) (bool, error) {
	signature := headers["X-Figma-Signature"]
	if signature == "" {
		return false, fmt.Errorf("missing X-Figma-Signature header")
	}
	expected := computeHMACSHA256(payload, []byte(config.SecretKey))
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}
