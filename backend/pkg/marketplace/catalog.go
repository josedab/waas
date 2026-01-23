package marketplace

import "time"

// BuiltinCategory represents a category of integrations
type BuiltinCategory struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	IconEmoji   string            `json:"icon_emoji"`
	Templates   []BuiltinTemplate `json:"templates"`
}

// BuiltinTemplate represents a pre-built integration template
type BuiltinTemplate struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Slug              string            `json:"slug"`
	Description       string            `json:"description"`
	Provider          string            `json:"provider"`
	Category          string            `json:"category"`
	EventTypes        []string          `json:"event_types"`
	SetupGuideURL     string            `json:"setup_guide_url,omitempty"`
	TransformTemplate string            `json:"transform_template,omitempty"`
	SamplePayload     string            `json:"sample_payload,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
	SchemaVersion     string            `json:"schema_version"`
	Verified          bool              `json:"verified"`
	CreatedAt         time.Time         `json:"created_at"`
}

// GetBuiltinCatalog returns the full built-in integration catalog
func GetBuiltinCatalog() []BuiltinCategory {
	return []BuiltinCategory{
		{
			ID: "payments", Name: "Payments", Description: "Payment processing integrations",
			IconEmoji: "💳",
			Templates: []BuiltinTemplate{
				{
					ID: "stripe-webhooks", Name: "Stripe", Slug: "stripe",
					Provider: "Stripe", Category: "payments",
					Description:   "Receive Stripe payment events including charges, subscriptions, and disputes",
					EventTypes:    []string{"charge.succeeded", "charge.failed", "invoice.paid", "customer.subscription.created", "customer.subscription.deleted", "payment_intent.succeeded"},
					SamplePayload: `{"type":"charge.succeeded","data":{"object":{"id":"ch_1","amount":2000,"currency":"usd"}}}`,
					Verified:      true,
				},
				{
					ID: "paypal-webhooks", Name: "PayPal", Slug: "paypal",
					Provider: "PayPal", Category: "payments",
					Description: "Receive PayPal payment and dispute notifications",
					EventTypes:  []string{"PAYMENT.CAPTURE.COMPLETED", "PAYMENT.CAPTURE.DENIED", "CUSTOMER.DISPUTE.CREATED"},
					Verified:    true,
				},
				{
					ID: "square-webhooks", Name: "Square", Slug: "square",
					Provider: "Square", Category: "payments",
					Description:   "Receive Square payment, order, and catalog events",
					EventTypes:    []string{"payment.completed", "payment.updated", "order.created", "order.updated"},
					Verified:      true,
				},
				{
					ID: "paddle-webhooks", Name: "Paddle", Slug: "paddle",
					Provider: "Paddle", Category: "payments",
					Description: "Receive Paddle subscription and payment events",
					EventTypes:  []string{"subscription.created", "subscription.updated", "subscription.cancelled", "transaction.completed"},
					Verified:    true,
				},
			},
		},
		{
			ID: "communication", Name: "Communication", Description: "Messaging and email integrations",
			IconEmoji: "📧",
			Templates: []BuiltinTemplate{
				{
					ID: "twilio-webhooks", Name: "Twilio", Slug: "twilio",
					Provider: "Twilio", Category: "communication",
					Description: "Receive Twilio SMS, voice, and messaging events",
					EventTypes:  []string{"message.sent", "message.delivered", "message.failed", "call.completed"},
					Verified:    true,
				},
				{
					ID: "sendgrid-webhooks", Name: "SendGrid", Slug: "sendgrid",
					Provider: "SendGrid", Category: "communication",
					Description: "Receive SendGrid email delivery and engagement events",
					EventTypes:  []string{"delivered", "open", "click", "bounce", "dropped", "spam_report"},
					Verified:    true,
				},
				{
					ID: "slack-webhooks", Name: "Slack", Slug: "slack",
					Provider: "Slack", Category: "communication",
					Description: "Receive Slack workspace events and interactions",
					EventTypes:  []string{"message", "app_mention", "member_joined_channel", "reaction_added"},
					Verified:    true,
				},
				{
					ID: "discord-webhooks", Name: "Discord", Slug: "discord",
					Provider: "Discord", Category: "communication",
					Description: "Send webhook notifications to Discord channels",
					EventTypes:  []string{"message.create", "interaction.create"},
					Verified:    true,
				},
			},
		},
		{
			ID: "devtools", Name: "Developer Tools", Description: "Source control and project management",
			IconEmoji: "🔧",
			Templates: []BuiltinTemplate{
				{
					ID: "github-webhooks", Name: "GitHub", Slug: "github",
					Provider: "GitHub", Category: "devtools",
					Description: "Receive GitHub repository, pull request, and issue events",
					EventTypes:  []string{"push", "pull_request", "issues", "release", "deployment", "workflow_run"},
					Verified:    true,
				},
				{
					ID: "gitlab-webhooks", Name: "GitLab", Slug: "gitlab",
					Provider: "GitLab", Category: "devtools",
					Description: "Receive GitLab merge request, pipeline, and push events",
					EventTypes:  []string{"push", "merge_request", "pipeline", "tag_push", "note"},
					Verified:    true,
				},
				{
					ID: "jira-webhooks", Name: "Jira", Slug: "jira",
					Provider: "Atlassian", Category: "devtools",
					Description: "Receive Jira issue creation, update, and transition events",
					EventTypes:  []string{"jira:issue_created", "jira:issue_updated", "jira:issue_deleted"},
					Verified:    true,
				},
				{
					ID: "linear-webhooks", Name: "Linear", Slug: "linear",
					Provider: "Linear", Category: "devtools",
					Description: "Receive Linear issue and project events",
					EventTypes:  []string{"Issue", "Comment", "Project", "Cycle"},
					Verified:    true,
				},
			},
		},
		{
			ID: "ecommerce", Name: "E-Commerce", Description: "Online store integrations",
			IconEmoji: "🛒",
			Templates: []BuiltinTemplate{
				{
					ID: "shopify-webhooks", Name: "Shopify", Slug: "shopify",
					Provider: "Shopify", Category: "ecommerce",
					Description: "Receive Shopify order, product, and customer events",
					EventTypes:  []string{"orders/create", "orders/updated", "products/create", "customers/create", "checkouts/create"},
					Verified:    true,
				},
				{
					ID: "woocommerce-webhooks", Name: "WooCommerce", Slug: "woocommerce",
					Provider: "WooCommerce", Category: "ecommerce",
					Description: "Receive WooCommerce order and product events",
					EventTypes:  []string{"order.created", "order.updated", "product.created", "customer.created"},
					Verified:    true,
				},
			},
		},
		{
			ID: "crm", Name: "CRM", Description: "Customer relationship management",
			IconEmoji: "👥",
			Templates: []BuiltinTemplate{
				{
					ID: "salesforce-webhooks", Name: "Salesforce", Slug: "salesforce",
					Provider: "Salesforce", Category: "crm",
					Description: "Receive Salesforce object change notifications via Platform Events",
					EventTypes:  []string{"AccountChangeEvent", "ContactChangeEvent", "OpportunityChangeEvent", "LeadChangeEvent"},
					Verified:    true,
				},
				{
					ID: "hubspot-webhooks", Name: "HubSpot", Slug: "hubspot",
					Provider: "HubSpot", Category: "crm",
					Description: "Receive HubSpot contact, deal, and company events",
					EventTypes:  []string{"contact.creation", "contact.propertyChange", "deal.creation", "company.creation"},
					Verified:    true,
				},
			},
		},
		{
			ID: "monitoring", Name: "Monitoring & Alerting", Description: "Infrastructure and application monitoring",
			IconEmoji: "📊",
			Templates: []BuiltinTemplate{
				{
					ID: "datadog-webhooks", Name: "Datadog", Slug: "datadog",
					Provider: "Datadog", Category: "monitoring",
					Description: "Receive Datadog alert and monitor notifications",
					EventTypes:  []string{"alert", "monitor.triggered", "monitor.recovered"},
					Verified:    true,
				},
				{
					ID: "pagerduty-webhooks", Name: "PagerDuty", Slug: "pagerduty",
					Provider: "PagerDuty", Category: "monitoring",
					Description: "Receive PagerDuty incident lifecycle events",
					EventTypes:  []string{"incident.triggered", "incident.acknowledged", "incident.resolved"},
					Verified:    true,
				},
			},
		},
	}
}

// SearchCatalog searches the built-in catalog by query string
func SearchCatalog(query string) []BuiltinTemplate {
	if query == "" {
		return allTemplates()
	}

	var results []BuiltinTemplate
	q := toLower(query)
	for _, cat := range GetBuiltinCatalog() {
		for _, tmpl := range cat.Templates {
			if matchesQuery(tmpl, q) {
				results = append(results, tmpl)
			}
		}
	}
	return results
}

// GetTemplateBySlug finds a template by its slug
func GetTemplateBySlug(slug string) *BuiltinTemplate {
	for _, cat := range GetBuiltinCatalog() {
		for i := range cat.Templates {
			if cat.Templates[i].Slug == slug {
				return &cat.Templates[i]
			}
		}
	}
	return nil
}

// GetTemplatesByCategory returns all templates in a category
func GetTemplatesByCategory(categoryID string) []BuiltinTemplate {
	for _, cat := range GetBuiltinCatalog() {
		if cat.ID == categoryID {
			return cat.Templates
		}
	}
	return nil
}

func allTemplates() []BuiltinTemplate {
	var all []BuiltinTemplate
	for _, cat := range GetBuiltinCatalog() {
		all = append(all, cat.Templates...)
	}
	return all
}

func matchesQuery(tmpl BuiltinTemplate, query string) bool {
	searchable := toLower(tmpl.Name + " " + tmpl.Provider + " " + tmpl.Description + " " + tmpl.Category)
	return containsStr(searchable, query)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func containsStr(s, substr string) bool {
	return len(substr) <= len(s) && findStr(s, substr)
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
