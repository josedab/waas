package connectors

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// --- Stripe Connector ---

type StripeConnector struct{}

func NewStripeConnector() *StripeConnector  { return &StripeConnector{} }
func (c *StripeConnector) ID() string       { return "connector-stripe" }
func (c *StripeConnector) Name() string     { return "Stripe" }
func (c *StripeConnector) Provider() string { return "stripe" }
func (c *StripeConnector) Version() string  { return "1.0.0" }
func (c *StripeConnector) EventTypes() []string {
	return []string{
		"payment_intent.succeeded", "payment_intent.failed", "charge.succeeded",
		"charge.refunded", "customer.created", "customer.updated",
		"invoice.paid", "invoice.payment_failed", "checkout.session.completed",
		"subscription.created", "subscription.updated", "subscription.deleted",
	}
}
func (c *StripeConnector) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"webhook_secret":{"type":"string","description":"Stripe webhook signing secret (whsec_...)"}}}`)
}

func (c *StripeConnector) VerifySignature(payload []byte, headers map[string]string, secret string) (bool, error) {
	sig := headers["Stripe-Signature"]
	if sig == "" {
		sig = headers["stripe-signature"]
	}
	if sig == "" {
		return false, fmt.Errorf("missing Stripe-Signature header")
	}
	parts := strings.Split(sig, ",")
	var timestamp, signature string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			switch kv[0] {
			case "t":
				timestamp = kv[1]
			case "v1":
				signature = kv[1]
			}
		}
	}
	if timestamp == "" || signature == "" {
		return false, fmt.Errorf("invalid Stripe-Signature format")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Now().Unix()-ts > 300 {
		return false, fmt.Errorf("stripe signature timestamp invalid or expired")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%s.%s", timestamp, string(payload))))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

func (c *StripeConnector) NormalizePayload(payload []byte, headers map[string]string) (*NormalizedEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	eventType, _ := raw["type"].(string)
	externalID, _ := raw["id"].(string)
	return &NormalizedEvent{
		Provider:   "stripe",
		EventType:  eventType,
		ExternalID: externalID,
		Timestamp:  time.Now(),
		Payload:    raw,
		RawPayload: payload,
	}, nil
}

func (c *StripeConnector) DetectProvider(headers map[string]string, payload []byte) (float64, error) {
	if _, ok := headers["Stripe-Signature"]; ok {
		return 0.95, nil
	}
	if _, ok := headers["stripe-signature"]; ok {
		return 0.95, nil
	}
	var p map[string]interface{}
	if json.Unmarshal(payload, &p) == nil {
		if _, ok := p["livemode"]; ok {
			if apiVersion, ok := p["api_version"]; ok {
				if strings.Contains(fmt.Sprint(apiVersion), "20") {
					return 0.7, nil
				}
			}
		}
	}
	return 0, nil
}

// --- GitHub Connector ---

type GitHubConnector struct{}

func NewGitHubConnector() *GitHubConnector  { return &GitHubConnector{} }
func (c *GitHubConnector) ID() string       { return "connector-github" }
func (c *GitHubConnector) Name() string     { return "GitHub" }
func (c *GitHubConnector) Provider() string { return "github" }
func (c *GitHubConnector) Version() string  { return "1.0.0" }
func (c *GitHubConnector) EventTypes() []string {
	return []string{
		"push", "pull_request", "issues", "issue_comment", "create", "delete",
		"release", "workflow_run", "check_run", "deployment", "star", "fork",
	}
}
func (c *GitHubConnector) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"webhook_secret":{"type":"string","description":"GitHub webhook secret"}}}`)
}

func (c *GitHubConnector) VerifySignature(payload []byte, headers map[string]string, secret string) (bool, error) {
	sig := headers["X-Hub-Signature-256"]
	if sig == "" {
		sig = headers["x-hub-signature-256"]
	}
	if sig == "" {
		return false, fmt.Errorf("missing X-Hub-Signature-256 header")
	}
	if !strings.HasPrefix(sig, "sha256=") {
		return false, fmt.Errorf("invalid signature format")
	}
	signature := strings.TrimPrefix(sig, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

func (c *GitHubConnector) NormalizePayload(payload []byte, headers map[string]string) (*NormalizedEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	eventType := headers["X-GitHub-Event"]
	if eventType == "" {
		eventType = headers["x-github-event"]
	}
	action, _ := raw["action"].(string)
	if action != "" {
		eventType = eventType + "." + action
	}
	return &NormalizedEvent{
		Provider:  "github",
		EventType: eventType,
		Timestamp: time.Now(),
		Payload:   raw,
		Metadata: map[string]string{
			"delivery_id": headers["X-GitHub-Delivery"],
		},
		RawPayload: payload,
	}, nil
}

func (c *GitHubConnector) DetectProvider(headers map[string]string, payload []byte) (float64, error) {
	if _, ok := headers["X-GitHub-Event"]; ok {
		return 0.95, nil
	}
	if _, ok := headers["x-github-event"]; ok {
		return 0.95, nil
	}
	if _, ok := headers["X-Hub-Signature-256"]; ok {
		return 0.8, nil
	}
	return 0, nil
}

// --- Shopify Connector ---

type ShopifyConnector struct{}

func NewShopifyConnector() *ShopifyConnector { return &ShopifyConnector{} }
func (c *ShopifyConnector) ID() string       { return "connector-shopify" }
func (c *ShopifyConnector) Name() string     { return "Shopify" }
func (c *ShopifyConnector) Provider() string { return "shopify" }
func (c *ShopifyConnector) Version() string  { return "1.0.0" }
func (c *ShopifyConnector) EventTypes() []string {
	return []string{
		"orders/create", "orders/updated", "orders/paid", "orders/cancelled",
		"products/create", "products/update", "products/delete",
		"customers/create", "customers/update", "carts/create", "carts/update",
		"checkouts/create", "refunds/create", "fulfillments/create",
	}
}
func (c *ShopifyConnector) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"api_secret":{"type":"string","description":"Shopify API secret key"}}}`)
}

func (c *ShopifyConnector) VerifySignature(payload []byte, headers map[string]string, secret string) (bool, error) {
	sig := headers["X-Shopify-Hmac-Sha256"]
	if sig == "" {
		sig = headers["x-shopify-hmac-sha256"]
	}
	if sig == "" {
		return false, fmt.Errorf("missing X-Shopify-Hmac-Sha256 header")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig)), nil
}

func (c *ShopifyConnector) NormalizePayload(payload []byte, headers map[string]string) (*NormalizedEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	topic := headers["X-Shopify-Topic"]
	if topic == "" {
		topic = headers["x-shopify-topic"]
	}
	return &NormalizedEvent{
		Provider:  "shopify",
		EventType: topic,
		Timestamp: time.Now(),
		Payload:   raw,
		Metadata: map[string]string{
			"shop_domain": headers["X-Shopify-Shop-Domain"],
			"api_version": headers["X-Shopify-API-Version"],
		},
		RawPayload: payload,
	}, nil
}

func (c *ShopifyConnector) DetectProvider(headers map[string]string, payload []byte) (float64, error) {
	if _, ok := headers["X-Shopify-Topic"]; ok {
		return 0.95, nil
	}
	if _, ok := headers["x-shopify-topic"]; ok {
		return 0.95, nil
	}
	if _, ok := headers["X-Shopify-Hmac-Sha256"]; ok {
		return 0.85, nil
	}
	return 0, nil
}

// --- Twilio Connector ---

type TwilioConnector struct{}

func NewTwilioConnector() *TwilioConnector  { return &TwilioConnector{} }
func (c *TwilioConnector) ID() string       { return "connector-twilio" }
func (c *TwilioConnector) Name() string     { return "Twilio" }
func (c *TwilioConnector) Provider() string { return "twilio" }
func (c *TwilioConnector) Version() string  { return "1.0.0" }
func (c *TwilioConnector) EventTypes() []string {
	return []string{"sms.received", "sms.sent", "sms.delivered", "sms.failed", "call.initiated", "call.completed", "call.failed"}
}
func (c *TwilioConnector) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"auth_token":{"type":"string","description":"Twilio auth token"}}}`)
}

func (c *TwilioConnector) VerifySignature(payload []byte, headers map[string]string, secret string) (bool, error) {
	sig := headers["X-Twilio-Signature"]
	if sig == "" {
		sig = headers["x-twilio-signature"]
	}
	if sig == "" {
		return false, fmt.Errorf("missing X-Twilio-Signature header")
	}
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig)), nil
}

func (c *TwilioConnector) NormalizePayload(payload []byte, headers map[string]string) (*NormalizedEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	eventType := "sms.received"
	if status, ok := raw["MessageStatus"].(string); ok {
		eventType = "sms." + strings.ToLower(status)
	}
	return &NormalizedEvent{
		Provider:   "twilio",
		EventType:  eventType,
		ExternalID: fmt.Sprintf("%v", raw["MessageSid"]),
		Timestamp:  time.Now(),
		Payload:    raw,
		RawPayload: payload,
	}, nil
}

func (c *TwilioConnector) DetectProvider(headers map[string]string, payload []byte) (float64, error) {
	if _, ok := headers["X-Twilio-Signature"]; ok {
		return 0.95, nil
	}
	if _, ok := headers["x-twilio-signature"]; ok {
		return 0.95, nil
	}
	return 0, nil
}

// --- Slack Connector ---

type SlackConnector struct{}

func NewSlackConnector() *SlackConnector   { return &SlackConnector{} }
func (c *SlackConnector) ID() string       { return "connector-slack" }
func (c *SlackConnector) Name() string     { return "Slack" }
func (c *SlackConnector) Provider() string { return "slack" }
func (c *SlackConnector) Version() string  { return "1.0.0" }
func (c *SlackConnector) EventTypes() []string {
	return []string{
		"message", "app_mention", "reaction_added", "reaction_removed",
		"channel_created", "member_joined_channel", "team_join",
		"url_verification", "event_callback",
	}
}
func (c *SlackConnector) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"signing_secret":{"type":"string","description":"Slack app signing secret"}}}`)
}

func (c *SlackConnector) VerifySignature(payload []byte, headers map[string]string, secret string) (bool, error) {
	sig := headers["X-Slack-Signature"]
	if sig == "" {
		sig = headers["x-slack-signature"]
	}
	timestamp := headers["X-Slack-Request-Timestamp"]
	if timestamp == "" {
		timestamp = headers["x-slack-request-timestamp"]
	}
	if sig == "" || timestamp == "" {
		return false, fmt.Errorf("missing Slack signature headers")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Now().Unix()-ts > 300 {
		return false, fmt.Errorf("slack timestamp invalid or expired")
	}
	if !strings.HasPrefix(sig, "v0=") {
		return false, fmt.Errorf("invalid Slack signature format")
	}
	signature := strings.TrimPrefix(sig, "v0=")
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(baseString))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

func (c *SlackConnector) NormalizePayload(payload []byte, headers map[string]string) (*NormalizedEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	eventType, _ := raw["type"].(string)
	if event, ok := raw["event"].(map[string]interface{}); ok {
		if t, ok := event["type"].(string); ok {
			eventType = t
		}
	}
	return &NormalizedEvent{
		Provider:   "slack",
		EventType:  eventType,
		ExternalID: fmt.Sprintf("%v", raw["event_id"]),
		Timestamp:  time.Now(),
		Payload:    raw,
		RawPayload: payload,
	}, nil
}

func (c *SlackConnector) DetectProvider(headers map[string]string, payload []byte) (float64, error) {
	if _, ok := headers["X-Slack-Signature"]; ok {
		return 0.95, nil
	}
	if _, ok := headers["x-slack-signature"]; ok {
		return 0.95, nil
	}
	return 0, nil
}

// --- Stub connectors for remaining 15 providers ---
// These follow the same pattern with provider-specific detection

type stubConnector struct {
	id, name, provider, version string
	events                      []string
	detectHeader                string
}

func (c *stubConnector) ID() string           { return c.id }
func (c *stubConnector) Name() string         { return c.name }
func (c *stubConnector) Provider() string     { return c.provider }
func (c *stubConnector) Version() string      { return c.version }
func (c *stubConnector) EventTypes() []string { return c.events }
func (c *stubConnector) ConfigSchema() json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"type":"object","properties":{"webhook_secret":{"type":"string","description":"%s webhook secret"}}}`, c.name))
}

func (c *stubConnector) VerifySignature(payload []byte, headers map[string]string, secret string) (bool, error) {
	// Generic HMAC-SHA256 verification
	sig := headers[c.detectHeader]
	if sig == "" {
		return false, fmt.Errorf("missing %s header", c.detectHeader)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if hmac.Equal([]byte(expected), []byte(sig)) {
		return true, nil
	}
	expectedB64 := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedB64), []byte(sig)), nil
}

func (c *stubConnector) NormalizePayload(payload []byte, headers map[string]string) (*NormalizedEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	eventType := ""
	for _, key := range []string{"type", "event_type", "event", "action", "topic"} {
		if v, ok := raw[key].(string); ok {
			eventType = v
			break
		}
	}
	return &NormalizedEvent{
		Provider:   c.provider,
		EventType:  eventType,
		Timestamp:  time.Now(),
		Payload:    raw,
		RawPayload: payload,
	}, nil
}

func (c *stubConnector) DetectProvider(headers map[string]string, payload []byte) (float64, error) {
	if _, ok := headers[c.detectHeader]; ok {
		return 0.9, nil
	}
	// Case-insensitive fallback
	lowerHeader := strings.ToLower(c.detectHeader)
	for k := range headers {
		if strings.ToLower(k) == lowerHeader {
			return 0.9, nil
		}
	}
	return 0, nil
}

func NewSendGridConnector() ConnectorInterface {
	return &stubConnector{"connector-sendgrid", "SendGrid", "sendgrid", "1.0.0",
		[]string{"email.delivered", "email.opened", "email.clicked", "email.bounced", "email.dropped"},
		"X-Twilio-Email-Event-Webhook-Signature"}
}

func NewLinearConnector() ConnectorInterface {
	return &stubConnector{"connector-linear", "Linear", "linear", "1.0.0",
		[]string{"issue.created", "issue.updated", "comment.created", "project.updated"},
		"Linear-Signature"}
}

func NewIntercomConnector() ConnectorInterface {
	return &stubConnector{"connector-intercom", "Intercom", "intercom", "1.0.0",
		[]string{"conversation.created", "conversation.closed", "user.created", "contact.created"},
		"X-Hub-Signature"}
}

func NewHubSpotConnector() ConnectorInterface {
	return &stubConnector{"connector-hubspot", "HubSpot", "hubspot", "1.0.0",
		[]string{"contact.creation", "contact.propertyChange", "deal.creation", "deal.propertyChange"},
		"X-HubSpot-Signature-v3"}
}

func NewZoomConnector() ConnectorInterface {
	return &stubConnector{"connector-zoom", "Zoom", "zoom", "1.0.0",
		[]string{"meeting.started", "meeting.ended", "participant.joined", "recording.completed"},
		"X-Zm-Signature"}
}

func NewAsanaConnector() ConnectorInterface {
	return &stubConnector{"connector-asana", "Asana", "asana", "1.0.0",
		[]string{"task.created", "task.completed", "project.changed", "story.created"},
		"X-Hook-Secret"}
}

func NewJiraConnector() ConnectorInterface {
	return &stubConnector{"connector-jira", "Jira", "jira", "1.0.0",
		[]string{"jira:issue_created", "jira:issue_updated", "comment_created", "sprint_started"},
		"X-Atlassian-Webhook-Identifier"}
}

func NewPaddleConnector() ConnectorInterface {
	return &stubConnector{"connector-paddle", "Paddle", "paddle", "1.0.0",
		[]string{"subscription.created", "subscription.updated", "transaction.completed", "transaction.payment_failed"},
		"Paddle-Signature"}
}

func NewSquareConnector() ConnectorInterface {
	return &stubConnector{"connector-square", "Square", "square", "1.0.0",
		[]string{"payment.completed", "order.created", "order.updated", "refund.created"},
		"X-Square-Hmacsha256-Signature"}
}

func NewBrexConnector() ConnectorInterface {
	return &stubConnector{"connector-brex", "Brex", "brex", "1.0.0",
		[]string{"transaction.created", "expense.updated", "card.activated"},
		"X-Brex-Webhook-Signature"}
}

func NewPostmarkConnector() ConnectorInterface {
	return &stubConnector{"connector-postmark", "Postmark", "postmark", "1.0.0",
		[]string{"email.delivered", "email.bounced", "email.opened", "email.clicked"},
		"X-Postmark-Signature"}
}

func NewSvixConnector() ConnectorInterface {
	return &stubConnector{"connector-svix", "Svix", "svix", "1.0.0",
		[]string{"message.attempt.exhausted", "message.attempt.failing"},
		"Svix-Signature"}
}

func NewClerkConnector() ConnectorInterface {
	return &stubConnector{"connector-clerk", "Clerk", "clerk", "1.0.0",
		[]string{"user.created", "user.updated", "session.created", "organization.created"},
		"Svix-Signature"}
}

func NewResendConnector() ConnectorInterface {
	return &stubConnector{"connector-resend", "Resend", "resend", "1.0.0",
		[]string{"email.sent", "email.delivered", "email.bounced", "email.opened"},
		"Svix-Signature"}
}

func NewVercelConnector() ConnectorInterface {
	return &stubConnector{"connector-vercel", "Vercel", "vercel", "1.0.0",
		[]string{"deployment.created", "deployment.succeeded", "deployment.error"},
		"X-Vercel-Signature"}
}
