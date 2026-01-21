package connectors

import (
	"encoding/json"
)

// Registry holds all available connectors
type Registry struct {
	connectors map[string]*Connector
}

// NewRegistry creates a new connector registry with built-in connectors
func NewRegistry() *Registry {
	r := &Registry{
		connectors: make(map[string]*Connector),
	}
	r.registerBuiltInConnectors()
	return r
}

// Get returns a connector by ID
func (r *Registry) Get(id string) *Connector {
	return r.connectors[id]
}

// List returns all connectors matching the filters
func (r *Registry) List(filters *MarketplaceListRequest) []*Connector {
	var result []*Connector

	for _, c := range r.connectors {
		if filters.Category != "" && c.Category != filters.Category {
			continue
		}
		if filters.Source != "" && c.Source != filters.Source {
			continue
		}
		if filters.Destination != "" && c.Destination != filters.Destination {
			continue
		}
		if filters.Official != nil && c.IsOfficial != *filters.Official {
			continue
		}
		result = append(result, c)
	}

	return result
}

// Register adds a connector to the registry
func (r *Registry) Register(c *Connector) {
	r.connectors[c.ID] = c
}

func (r *Registry) registerBuiltInConnectors() {
	// Stripe to Slack
	r.Register(&Connector{
		ID:          "stripe-to-slack",
		Name:        "Stripe to Slack",
		Description: "Send Stripe payment notifications to Slack channels",
		Category:    CategoryNotification,
		Source:      "stripe",
		Destination: "slack",
		Version:     "1.0.0",
		Author:      "WAAS Team",
		IsOfficial:  true,
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["slack_webhook_url"],
			"properties": {
				"slack_webhook_url": {"type": "string", "description": "Slack Incoming Webhook URL"},
				"slack_channel": {"type": "string", "description": "Override channel (optional)"}
			}
		}`),
		Transform: stripeToSlackTransform,
	})

	// GitHub to Slack
	r.Register(&Connector{
		ID:          "github-to-slack",
		Name:        "GitHub to Slack",
		Description: "Send GitHub repository events to Slack channels",
		Category:    CategoryNotification,
		Source:      "github",
		Destination: "slack",
		Version:     "1.0.0",
		Author:      "WAAS Team",
		IsOfficial:  true,
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["slack_webhook_url"],
			"properties": {
				"slack_webhook_url": {"type": "string"},
				"events_filter": {"type": "array", "items": {"type": "string"}}
			}
		}`),
		Transform: githubToSlackTransform,
	})

	// GitHub to Discord
	r.Register(&Connector{
		ID:          "github-to-discord",
		Name:        "GitHub to Discord",
		Description: "Send GitHub repository events to Discord channels",
		Category:    CategoryNotification,
		Source:      "github",
		Destination: "discord",
		Version:     "1.0.0",
		Author:      "WAAS Team",
		IsOfficial:  true,
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["discord_webhook_url"],
			"properties": {
				"discord_webhook_url": {"type": "string"}
			}
		}`),
		Transform: githubToDiscordTransform,
	})

	// Shopify to Slack
	r.Register(&Connector{
		ID:          "shopify-to-slack",
		Name:        "Shopify to Slack",
		Description: "Send Shopify order notifications to Slack",
		Category:    CategoryNotification,
		Source:      "shopify",
		Destination: "slack",
		Version:     "1.0.0",
		Author:      "WAAS Team",
		IsOfficial:  true,
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["slack_webhook_url"],
			"properties": {
				"slack_webhook_url": {"type": "string"}
			}
		}`),
		Transform: shopifyToSlackTransform,
	})

	// Stripe to PagerDuty
	r.Register(&Connector{
		ID:          "stripe-to-pagerduty",
		Name:        "Stripe to PagerDuty",
		Description: "Create PagerDuty alerts for critical Stripe events",
		Category:    CategoryNotification,
		Source:      "stripe",
		Destination: "pagerduty",
		Version:     "1.0.0",
		Author:      "WAAS Team",
		IsOfficial:  true,
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["pagerduty_key"],
			"properties": {
				"pagerduty_key": {"type": "string", "description": "PagerDuty Integration Key"},
				"severity": {"type": "string", "enum": ["critical", "error", "warning", "info"]}
			}
		}`),
		Transform: stripeToPagerDutyTransform,
	})

	// GitHub to Jira
	r.Register(&Connector{
		ID:          "github-to-jira",
		Name:        "GitHub to Jira",
		Description: "Create Jira issues from GitHub issues and PRs",
		Category:    CategoryTicketing,
		Source:      "github",
		Destination: "jira",
		Version:     "1.0.0",
		Author:      "WAAS Team",
		IsOfficial:  true,
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["jira_url", "jira_project", "jira_token"],
			"properties": {
				"jira_url": {"type": "string"},
				"jira_project": {"type": "string"},
				"jira_token": {"type": "string"}
			}
		}`),
		Transform: githubToJiraTransform,
	})

	// Generic Webhook Forwarder
	r.Register(&Connector{
		ID:          "webhook-forwarder",
		Name:        "Webhook Forwarder",
		Description: "Forward webhooks to any HTTP endpoint with custom headers",
		Category:    CategoryCustom,
		Source:      "any",
		Destination: "http",
		Version:     "1.0.0",
		Author:      "WAAS Team",
		IsOfficial:  true,
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["webhook_url"],
			"properties": {
				"webhook_url": {"type": "string"},
				"headers": {"type": "object", "additionalProperties": {"type": "string"}}
			}
		}`),
		Transform: webhookForwarderTransform,
	})
}

// JavaScript transforms for each connector
const stripeToSlackTransform = `
function transform(event, config) {
	var emoji = "💳";
	var color = "#6772e5";
	
	switch(event.type) {
		case "payment_intent.succeeded":
			emoji = "✅";
			color = "#36a64f";
			break;
		case "payment_intent.payment_failed":
			emoji = "❌";
			color = "#dc3545";
			break;
		case "customer.subscription.created":
			emoji = "🎉";
			break;
		case "invoice.paid":
			emoji = "💰";
			color = "#36a64f";
			break;
	}
	
	return {
		text: emoji + " Stripe: " + event.type,
		attachments: [{
			color: color,
			fields: [
				{title: "Event", value: event.type, short: true},
				{title: "ID", value: event.id, short: true}
			],
			ts: event.created
		}]
	};
}
`

const githubToSlackTransform = `
function transform(event, config) {
	var emoji = "📦";
	var text = "";
	
	if (event.action === "opened" && event.pull_request) {
		emoji = "🔀";
		text = "New PR: " + event.pull_request.title;
	} else if (event.action === "opened" && event.issue) {
		emoji = "🐛";
		text = "New Issue: " + event.issue.title;
	} else if (event.action === "closed" && event.pull_request && event.pull_request.merged) {
		emoji = "✅";
		text = "PR Merged: " + event.pull_request.title;
	} else if (event.ref && event.pusher) {
		emoji = "📝";
		text = "Push to " + event.ref + " by " + event.pusher.name;
	} else {
		text = event.action + " on " + event.repository.full_name;
	}
	
	return {
		text: emoji + " " + text,
		attachments: [{
			color: "#24292e",
			author_name: event.sender ? event.sender.login : "GitHub",
			author_icon: event.sender ? event.sender.avatar_url : null,
			footer: event.repository ? event.repository.full_name : "GitHub"
		}]
	};
}
`

const githubToDiscordTransform = `
function transform(event, config) {
	var title = "GitHub Event";
	var description = "";
	var color = 2303786; // GitHub color
	
	if (event.pull_request) {
		title = "Pull Request: " + event.pull_request.title;
		description = event.action + " by " + event.sender.login;
	} else if (event.issue) {
		title = "Issue: " + event.issue.title;
		description = event.action + " by " + event.sender.login;
	}
	
	return {
		embeds: [{
			title: title,
			description: description,
			color: color,
			author: {
				name: event.sender ? event.sender.login : "GitHub",
				icon_url: event.sender ? event.sender.avatar_url : null
			}
		}]
	};
}
`

const shopifyToSlackTransform = `
function transform(event, config) {
	var order = event;
	var total = order.total_price || "0.00";
	var currency = order.currency || "USD";
	
	return {
		text: "🛒 New Shopify Order #" + order.order_number,
		attachments: [{
			color: "#96bf48",
			fields: [
				{title: "Customer", value: order.customer ? order.customer.email : "Guest", short: true},
				{title: "Total", value: currency + " " + total, short: true},
				{title: "Items", value: order.line_items ? order.line_items.length + " items" : "N/A", short: true}
			]
		}]
	};
}
`

const stripeToPagerDutyTransform = `
function transform(event, config) {
	var severity = config.severity || "warning";
	
	// Only alert on important events
	var criticalEvents = ["charge.failed", "payment_intent.payment_failed", "invoice.payment_failed"];
	if (criticalEvents.indexOf(event.type) >= 0) {
		severity = "critical";
	}
	
	return {
		routing_key: config.pagerduty_key,
		event_action: "trigger",
		payload: {
			summary: "Stripe: " + event.type,
			severity: severity,
			source: "waas-stripe-connector",
			custom_details: {
				event_id: event.id,
				event_type: event.type,
				created: event.created
			}
		}
	};
}
`

const githubToJiraTransform = `
function transform(event, config) {
	if (!event.issue && !event.pull_request) {
		return null; // Skip non-issue/PR events
	}
	
	var item = event.issue || event.pull_request;
	var issueType = event.issue ? "Bug" : "Task";
	
	return {
		fields: {
			project: {key: config.jira_project},
			summary: "[GitHub] " + item.title,
			description: item.body + "\\n\\nGitHub URL: " + item.html_url,
			issuetype: {name: issueType}
		}
	};
}
`

const webhookForwarderTransform = `
function transform(event, config) {
	// Pass through without modification
	return event;
}
`
