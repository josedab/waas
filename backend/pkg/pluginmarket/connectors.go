package pluginmarket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ConnectorTemplate defines a pre-built integration connector
type ConnectorTemplate struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Provider    string           `json:"provider"`
	Category    string           `json:"category"`
	Description string           `json:"description"`
	Version     string           `json:"version"`
	Events      []ConnectorEvent `json:"events"`
	Config      ConnectorConfig  `json:"config"`
	Icon        string           `json:"icon"`
	DocsURL     string           `json:"docs_url"`
	SetupGuide  string           `json:"setup_guide"`
}

// ConnectorEvent represents an event type supported by a connector
type ConnectorEvent struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Schema      string `json:"schema,omitempty"`
}

// ConnectorConfig describes configuration fields for a connector
type ConnectorConfig struct {
	Fields []ConfigField `json:"fields"`
}

// ConfigField represents a single configuration field
type ConfigField struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // string, secret, url, number, boolean, select
	Required    bool     `json:"required"`
	Description string   `json:"description"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"` // for select type
}

// ConnectorInstance represents an installed connector
type ConnectorInstance struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	TemplateID  string                 `json:"template_id"`
	Name        string                 `json:"name"`
	Config      map[string]interface{} `json:"config"`
	Status      string                 `json:"status"` // active, paused, error
	EventsRecvd int64                  `json:"events_received"`
	LastEventAt *time.Time             `json:"last_event_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// BuiltinConnectors returns all pre-built connector templates
func BuiltinConnectors() []ConnectorTemplate {
	return []ConnectorTemplate{
		{
			ID:          "stripe",
			Name:        "Stripe Webhooks",
			Provider:    "stripe",
			Category:    "payments",
			Description: "Receive payment, subscription, and invoice events from Stripe",
			Version:     "1.0.0",
			Icon:        "💳",
			DocsURL:     "https://stripe.com/docs/webhooks",
			Events: []ConnectorEvent{
				{Type: "payment_intent.succeeded", Description: "Payment completed successfully"},
				{Type: "payment_intent.failed", Description: "Payment attempt failed"},
				{Type: "customer.subscription.created", Description: "New subscription created"},
				{Type: "customer.subscription.deleted", Description: "Subscription cancelled"},
				{Type: "invoice.paid", Description: "Invoice payment received"},
				{Type: "invoice.payment_failed", Description: "Invoice payment failed"},
			},
			Config: ConnectorConfig{
				Fields: []ConfigField{
					{Name: "webhook_secret", Type: "secret", Required: true, Description: "Stripe webhook signing secret (whsec_...)"},
					{Name: "api_version", Type: "string", Required: false, Description: "Stripe API version", Default: "2023-10-16"},
				},
			},
			SetupGuide: "1. Go to Stripe Dashboard → Developers → Webhooks\n2. Add endpoint URL\n3. Select events to listen for\n4. Copy the signing secret",
		},
		{
			ID:          "github",
			Name:        "GitHub Webhooks",
			Provider:    "github",
			Category:    "developer_tools",
			Description: "Receive repository, PR, issue, and deployment events from GitHub",
			Version:     "1.0.0",
			Icon:        "🐙",
			DocsURL:     "https://docs.github.com/en/webhooks",
			Events: []ConnectorEvent{
				{Type: "push", Description: "Push to repository"},
				{Type: "pull_request", Description: "Pull request opened, closed, or merged"},
				{Type: "issues", Description: "Issue opened, closed, or commented"},
				{Type: "release", Description: "Release published"},
				{Type: "deployment", Description: "Deployment created"},
				{Type: "workflow_run", Description: "GitHub Actions workflow completed"},
			},
			Config: ConnectorConfig{
				Fields: []ConfigField{
					{Name: "webhook_secret", Type: "secret", Required: true, Description: "GitHub webhook secret"},
					{Name: "content_type", Type: "select", Required: false, Description: "Payload format", Default: "json", Options: []string{"json", "form"}},
				},
			},
			SetupGuide: "1. Go to Repository → Settings → Webhooks → Add webhook\n2. Set payload URL to your WaaS endpoint\n3. Set content type to application/json\n4. Enter a secret and save it here",
		},
		{
			ID:          "slack",
			Name:        "Slack Events",
			Provider:    "slack",
			Category:    "communication",
			Description: "Receive message, channel, and app events from Slack",
			Version:     "1.0.0",
			Icon:        "💬",
			DocsURL:     "https://api.slack.com/events",
			Events: []ConnectorEvent{
				{Type: "message", Description: "New message in a channel"},
				{Type: "app_mention", Description: "Your app was mentioned"},
				{Type: "channel_created", Description: "New channel created"},
				{Type: "member_joined_channel", Description: "User joined a channel"},
				{Type: "reaction_added", Description: "Reaction added to a message"},
			},
			Config: ConnectorConfig{
				Fields: []ConfigField{
					{Name: "signing_secret", Type: "secret", Required: true, Description: "Slack app signing secret"},
					{Name: "bot_token", Type: "secret", Required: false, Description: "Slack bot token (xoxb-...)"},
				},
			},
			SetupGuide: "1. Go to api.slack.com/apps → Your App → Event Subscriptions\n2. Enable events and set the request URL\n3. Subscribe to bot events\n4. Copy the signing secret from Basic Information",
		},
		{
			ID:          "shopify",
			Name:        "Shopify Webhooks",
			Provider:    "shopify",
			Category:    "ecommerce",
			Description: "Receive order, product, and customer events from Shopify",
			Version:     "1.0.0",
			Icon:        "🛒",
			DocsURL:     "https://shopify.dev/docs/apps/webhooks",
			Events: []ConnectorEvent{
				{Type: "orders/create", Description: "New order placed"},
				{Type: "orders/fulfilled", Description: "Order fulfilled"},
				{Type: "orders/cancelled", Description: "Order cancelled"},
				{Type: "products/create", Description: "New product created"},
				{Type: "products/update", Description: "Product updated"},
				{Type: "customers/create", Description: "New customer registered"},
			},
			Config: ConnectorConfig{
				Fields: []ConfigField{
					{Name: "api_secret", Type: "secret", Required: true, Description: "Shopify API secret key"},
					{Name: "shop_domain", Type: "string", Required: true, Description: "Shop domain (e.g., mystore.myshopify.com)"},
					{Name: "api_version", Type: "string", Required: false, Description: "API version", Default: "2024-01"},
				},
			},
			SetupGuide: "1. Go to Shopify Admin → Settings → Notifications → Webhooks\n2. Create webhook with your WaaS endpoint URL\n3. Select the event type and format (JSON)\n4. Copy the API secret from your app settings",
		},
		{
			ID:          "twilio",
			Name:        "Twilio Webhooks",
			Provider:    "twilio",
			Category:    "communication",
			Description: "Receive SMS, voice, and messaging events from Twilio",
			Version:     "1.0.0",
			Icon:        "📱",
			DocsURL:     "https://www.twilio.com/docs/usage/webhooks",
			Events: []ConnectorEvent{
				{Type: "sms.received", Description: "Incoming SMS message"},
				{Type: "sms.delivered", Description: "SMS delivery confirmation"},
				{Type: "call.initiated", Description: "Voice call started"},
				{Type: "call.completed", Description: "Voice call ended"},
			},
			Config: ConnectorConfig{
				Fields: []ConfigField{
					{Name: "auth_token", Type: "secret", Required: true, Description: "Twilio Auth Token"},
					{Name: "account_sid", Type: "string", Required: true, Description: "Twilio Account SID"},
				},
			},
			SetupGuide: "1. Go to Twilio Console → Phone Numbers → Active Numbers\n2. Select your number and configure webhook URLs\n3. Copy your Account SID and Auth Token from the Console",
		},
		{
			ID:          "sendgrid",
			Name:        "SendGrid Event Webhooks",
			Provider:    "sendgrid",
			Category:    "communication",
			Description: "Receive email delivery, open, and click events from SendGrid",
			Version:     "1.0.0",
			Icon:        "📧",
			DocsURL:     "https://docs.sendgrid.com/for-developers/tracking-events/event",
			Events: []ConnectorEvent{
				{Type: "delivered", Description: "Email delivered to recipient"},
				{Type: "open", Description: "Email opened by recipient"},
				{Type: "click", Description: "Link clicked in email"},
				{Type: "bounce", Description: "Email bounced"},
				{Type: "dropped", Description: "Email dropped by SendGrid"},
				{Type: "unsubscribe", Description: "Recipient unsubscribed"},
			},
			Config: ConnectorConfig{
				Fields: []ConfigField{
					{Name: "verification_key", Type: "secret", Required: true, Description: "SendGrid Event Webhook verification key"},
				},
			},
			SetupGuide: "1. Go to SendGrid → Settings → Mail Settings → Event Webhooks\n2. Set the HTTP POST URL to your WaaS endpoint\n3. Select which events to receive\n4. Enable and copy the verification key",
		},
	}
}

// GetConnector returns a specific connector template by ID
func GetConnector(id string) (*ConnectorTemplate, error) {
	for _, c := range BuiltinConnectors() {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("connector not found: %s", id)
}

// GetConnectorsByCategory returns connectors filtered by category
func GetConnectorsByCategory(category string) []ConnectorTemplate {
	var result []ConnectorTemplate
	for _, c := range BuiltinConnectors() {
		if c.Category == category {
			result = append(result, c)
		}
	}
	return result
}

// CreateConnectorPlugin creates a PluginSDK instance from a connector template
func CreateConnectorPlugin(template *ConnectorTemplate) *PluginSDK {
	sdk := NewPluginSDK(
		template.Name,
		template.Version,
		template.Provider,
		template.Description,
	)

	// Register default hooks based on provider
	sdk.RegisterSDKHook(SDKHookOnReceive, func(ctx context.Context, hookCtx *SDKHookContext) (*SDKHookResult, error) {
		// Default receive hook: validate and normalize the payload
		var payload map[string]interface{}
		if err := json.Unmarshal(hookCtx.Payload, &payload); err != nil {
			return &SDKHookResult{Error: "invalid JSON payload"}, nil
		}

		normalized := map[string]interface{}{
			"provider":    template.Provider,
			"received_at": time.Now().UTC().Format(time.RFC3339),
			"raw":         payload,
		}

		result, _ := json.Marshal(normalized)
		return &SDKHookResult{
			Modified: true,
			Payload:  result,
		}, nil
	})

	sdk.RegisterSDKHook(SDKHookOnValidate, func(ctx context.Context, hookCtx *SDKHookContext) (*SDKHookResult, error) {
		// Validate event type is in the connector's supported events
		for _, event := range template.Events {
			if event.Type == hookCtx.EventType {
				return &SDKHookResult{Modified: false}, nil
			}
		}
		return &SDKHookResult{
			Error: fmt.Sprintf("unsupported event type '%s' for connector '%s'", hookCtx.EventType, template.Name),
		}, nil
	})

	return sdk
}
