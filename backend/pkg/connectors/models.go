package connectors

import (
	"encoding/json"
	"time"
)

// Connector represents a pre-built integration connector
type Connector struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`    // notification, crm, ticketing, etc.
	Source      string          `json:"source"`      // stripe, github, shopify, etc.
	Destination string          `json:"destination"` // slack, discord, email, etc.
	Version     string          `json:"version"`
	Author      string          `json:"author"`
	IconURL     string          `json:"icon_url,omitempty"`
	DocsURL     string          `json:"docs_url,omitempty"`
	ConfigSchema json.RawMessage `json:"config_schema"` // JSON Schema for configuration
	Transform   string          `json:"transform"`     // JavaScript transformation
	IsOfficial  bool            `json:"is_official"`
	InstallCount int64          `json:"install_count"`
	Rating      float64         `json:"rating"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// InstalledConnector represents a connector installed by a tenant
type InstalledConnector struct {
	ID           string          `json:"id" db:"id"`
	TenantID     string          `json:"tenant_id" db:"tenant_id"`
	ConnectorID  string          `json:"connector_id" db:"connector_id"`
	Name         string          `json:"name" db:"name"`
	Config       json.RawMessage `json:"config" db:"config"`
	IsActive     bool            `json:"is_active" db:"is_active"`
	ProviderID   string          `json:"provider_id,omitempty" db:"provider_id"`   // For inbound
	EndpointID   string          `json:"endpoint_id,omitempty" db:"endpoint_id"`   // For outbound
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// ConnectorConfig represents configuration for a connector
type ConnectorConfig struct {
	// Slack config
	SlackWebhookURL string `json:"slack_webhook_url,omitempty"`
	SlackChannel    string `json:"slack_channel,omitempty"`
	
	// Discord config
	DiscordWebhookURL string `json:"discord_webhook_url,omitempty"`
	
	// Email config
	EmailTo      string `json:"email_to,omitempty"`
	EmailFrom    string `json:"email_from,omitempty"`
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUser     string `json:"smtp_user,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	
	// PagerDuty config
	PagerDutyKey string `json:"pagerduty_key,omitempty"`
	
	// Jira config
	JiraURL     string `json:"jira_url,omitempty"`
	JiraProject string `json:"jira_project,omitempty"`
	JiraToken   string `json:"jira_token,omitempty"`
	
	// Generic webhook config
	WebhookURL  string            `json:"webhook_url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// Category constants
const (
	CategoryNotification = "notification"
	CategoryTicketing    = "ticketing"
	CategoryCRM          = "crm"
	CategoryAnalytics    = "analytics"
	CategoryCustom       = "custom"
)

// ConnectorEvent represents an event type handled by a connector
type ConnectorEvent struct {
	ConnectorID string `json:"connector_id"`
	EventType   string `json:"event_type"`
	Description string `json:"description"`
}

// InstallConnectorRequest represents a request to install a connector
type InstallConnectorRequest struct {
	ConnectorID string          `json:"connector_id" binding:"required"`
	Name        string          `json:"name" binding:"required,min=1,max=255"`
	Config      json.RawMessage `json:"config" binding:"required"`
}

// UpdateConnectorRequest represents a request to update an installed connector
type UpdateConnectorRequest struct {
	Name     string          `json:"name,omitempty"`
	Config   json.RawMessage `json:"config,omitempty"`
	IsActive bool            `json:"is_active"`
}

// ConnectorExecution represents a connector execution result
type ConnectorExecution struct {
	ID                   string    `json:"id" db:"id"`
	InstalledConnectorID string    `json:"installed_connector_id" db:"installed_connector_id"`
	EventType            string    `json:"event_type" db:"event_type"`
	InputPayload         []byte    `json:"-" db:"input_payload"`
	OutputPayload        []byte    `json:"-" db:"output_payload"`
	Status               string    `json:"status" db:"status"`
	Error                string    `json:"error,omitempty" db:"error"`
	Duration             int64     `json:"duration_ms" db:"duration_ms"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

// MarketplaceListRequest represents filters for listing connectors
type MarketplaceListRequest struct {
	Category    string `json:"category,omitempty" form:"category"`
	Source      string `json:"source,omitempty" form:"source"`
	Destination string `json:"destination,omitempty" form:"destination"`
	Search      string `json:"search,omitempty" form:"search"`
	Official    *bool  `json:"official,omitempty" form:"official"`
	Limit       int    `json:"limit,omitempty" form:"limit"`
	Offset      int    `json:"offset,omitempty" form:"offset"`
}
