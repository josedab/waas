package workflow

import (
	"encoding/json"
	"time"
)

// Template represents a reusable workflow template
type Template struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	Description     string            `json:"description"`
	Category        TemplateCategory  `json:"category"`
	Tags            []string          `json:"tags,omitempty"`
	Author          string            `json:"author"`
	AuthorVerified  bool              `json:"author_verified"`
	Version         string            `json:"version"`
	MinPlatformVer  string            `json:"min_platform_version,omitempty"`
	Workflow        *WorkflowDef      `json:"workflow"`
	Variables       []TemplateVar     `json:"variables,omitempty"`
	Documentation   string            `json:"documentation,omitempty"`
	Preview         *TemplatePreview  `json:"preview,omitempty"`
	UsageCount      int               `json:"usage_count"`
	Rating          float64           `json:"rating"`
	RatingCount     int               `json:"rating_count"`
	Featured        bool              `json:"featured"`
	Official        bool              `json:"official"`
	Public          bool              `json:"public"`
	TenantID        string            `json:"tenant_id,omitempty"` // Empty for public templates
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// TemplateCategory represents template categories
type TemplateCategory string

const (
	CategoryIntegration   TemplateCategory = "integration"
	CategoryNotification  TemplateCategory = "notification"
	CategoryDataPipeline  TemplateCategory = "data_pipeline"
	CategoryETL           TemplateCategory = "etl"
	CategoryMonitoring    TemplateCategory = "monitoring"
	CategorySecurity      TemplateCategory = "security"
	CategoryCustom        TemplateCategory = "custom"
)

// WorkflowDef is the template workflow definition (without tenant-specific data)
type WorkflowDef struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Trigger     TriggerConfig    `json:"trigger"`
	Nodes       []NodeDef        `json:"nodes"`
	Edges       []Edge           `json:"edges"`
	Variables   []Variable       `json:"variables,omitempty"`
	Settings    WorkflowSettings `json:"settings"`
	Canvas      CanvasState      `json:"canvas"`
}

// NodeDef is a node definition in a template
type NodeDef struct {
	ID          string          `json:"id"`
	Type        NodeType        `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Position    Position        `json:"position"`
	Config      json.RawMessage `json:"config"`
	Inputs      []Port          `json:"inputs,omitempty"`
	Outputs     []Port          `json:"outputs,omitempty"`
	Metadata    NodeMetadata    `json:"metadata,omitempty"`
	Configurable []ConfigField  `json:"configurable,omitempty"`
}

// ConfigField defines a configurable field in a node
type ConfigField struct {
	Key          string      `json:"key"`
	Label        string      `json:"label"`
	Type         string      `json:"type"` // string, number, boolean, select, json, expression
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Description  string      `json:"description,omitempty"`
	Placeholder  string      `json:"placeholder,omitempty"`
	Options      []Option    `json:"options,omitempty"` // For select type
	Validation   *Validation `json:"validation,omitempty"`
}

// Option represents a select option
type Option struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// Validation defines field validation rules
type Validation struct {
	Pattern   string `json:"pattern,omitempty"`
	Min       *int   `json:"min,omitempty"`
	Max       *int   `json:"max,omitempty"`
	MinLength *int   `json:"min_length,omitempty"`
	MaxLength *int   `json:"max_length,omitempty"`
}

// TemplateVar represents a template variable that users must configure
type TemplateVar struct {
	Name         string      `json:"name"`
	Label        string      `json:"label"`
	Type         string      `json:"type"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Description  string      `json:"description,omitempty"`
	Secret       bool        `json:"secret,omitempty"`
	Options      []Option    `json:"options,omitempty"`
}

// TemplatePreview contains preview/thumbnail data
type TemplatePreview struct {
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Description  string `json:"description,omitempty"`
	Complexity   string `json:"complexity,omitempty"` // simple, medium, complex
	EstSteps     int    `json:"estimated_steps,omitempty"`
	EstTimeMin   int    `json:"estimated_time_min,omitempty"`
}

// NodeTypeDefinition describes a node type for the visual builder
type NodeTypeDefinition struct {
	Type        NodeType      `json:"type"`
	Category    string        `json:"category"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Icon        string        `json:"icon"`
	Color       string        `json:"color"`
	Inputs      []PortDef     `json:"inputs"`
	Outputs     []PortDef     `json:"outputs"`
	Config      []ConfigField `json:"config"`
	Examples    []Example     `json:"examples,omitempty"`
}

// PortDef defines a port for a node type
type PortDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Multiple    bool   `json:"multiple,omitempty"`
	Description string `json:"description,omitempty"`
}

// Example provides usage examples for a node type
type Example struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
}

// ExportFormat represents workflow export formats
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatYAML ExportFormat = "yaml"
	ExportFormatCode ExportFormat = "code"
)

// WorkflowExport represents an exported workflow
type WorkflowExport struct {
	Version    string          `json:"version"`
	Format     ExportFormat    `json:"format"`
	ExportedAt time.Time       `json:"exported_at"`
	Workflow   json.RawMessage `json:"workflow"`
	Metadata   ExportMetadata  `json:"metadata"`
}

// ExportMetadata contains export metadata
type ExportMetadata struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	Author         string   `json:"author,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	OriginalID     string   `json:"original_id,omitempty"`
	PlatformVersion string  `json:"platform_version"`
}

// CreateTemplateRequest represents a request to create a template
type CreateTemplateRequest struct {
	Name          string           `json:"name" binding:"required"`
	Description   string           `json:"description"`
	Category      TemplateCategory `json:"category" binding:"required"`
	Tags          []string         `json:"tags,omitempty"`
	Workflow      *WorkflowDef     `json:"workflow" binding:"required"`
	Variables     []TemplateVar    `json:"variables,omitempty"`
	Documentation string           `json:"documentation,omitempty"`
	Public        bool             `json:"public"`
}

// InstantiateTemplateRequest represents a request to create a workflow from a template
type InstantiateTemplateRequest struct {
	TemplateID    string                 `json:"template_id" binding:"required"`
	Name          string                 `json:"name" binding:"required"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	Customizations map[string]interface{} `json:"customizations,omitempty"`
}

// TemplateFilters for listing templates
type TemplateFilters struct {
	Category   *TemplateCategory `json:"category,omitempty"`
	Search     string            `json:"search,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Author     string            `json:"author,omitempty"`
	Official   *bool             `json:"official,omitempty"`
	Featured   *bool             `json:"featured,omitempty"`
	MinRating  *float64          `json:"min_rating,omitempty"`
	Page       int               `json:"page,omitempty"`
	PageSize   int               `json:"page_size,omitempty"`
	SortBy     string            `json:"sort_by,omitempty"` // usage_count, rating, created_at
	SortOrder  string            `json:"sort_order,omitempty"`
}

// ListTemplatesResponse represents paginated template list
type ListTemplatesResponse struct {
	Templates  []Template `json:"templates"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// NodePalette represents the node palette for the visual builder
type NodePalette struct {
	Categories []PaletteCategory `json:"categories"`
}

// PaletteCategory groups node types in the palette
type PaletteCategory struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Icon        string               `json:"icon,omitempty"`
	NodeDefs    []NodeTypeDefinition `json:"nodes"`
}

// GetDefaultNodePalette returns the default node palette
func GetDefaultNodePalette() *NodePalette {
	return &NodePalette{
		Categories: []PaletteCategory{
			{
				ID:          "control",
				Name:        "Control Flow",
				Description: "Control workflow execution flow",
				Icon:        "git-branch",
				NodeDefs: []NodeTypeDefinition{
					{
						Type:        NodeCondition,
						Category:    "control",
						Name:        "Condition",
						Description: "Branch based on a condition",
						Icon:        "git-branch",
						Color:       "#f59e0b",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "true", Name: "True", Type: "any"}, {ID: "false", Name: "False", Type: "any"}},
						Config: []ConfigField{
							{Key: "expression", Label: "Condition", Type: "expression", Required: true, Description: "JavaScript expression that evaluates to true/false"},
						},
					},
					{
						Type:        NodeSwitch,
						Category:    "control",
						Name:        "Switch",
						Description: "Route to multiple branches based on value",
						Icon:        "share-2",
						Color:       "#f59e0b",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "default", Name: "Default", Type: "any"}},
						Config: []ConfigField{
							{Key: "field", Label: "Switch Field", Type: "string", Required: true},
							{Key: "cases", Label: "Cases", Type: "json", Required: true},
						},
					},
					{
						Type:        NodeLoop,
						Category:    "control",
						Name:        "Loop",
						Description: "Iterate over an array",
						Icon:        "repeat",
						Color:       "#f59e0b",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "array", Required: true}},
						Outputs:     []PortDef{{ID: "item", Name: "Item", Type: "any"}, {ID: "done", Name: "Done", Type: "any"}},
						Config: []ConfigField{
							{Key: "concurrency", Label: "Concurrency", Type: "number", DefaultValue: 1},
						},
					},
					{
						Type:        NodeParallel,
						Category:    "control",
						Name:        "Parallel",
						Description: "Execute branches in parallel",
						Icon:        "layers",
						Color:       "#f59e0b",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "branch1", Name: "Branch 1", Type: "any"}, {ID: "branch2", Name: "Branch 2", Type: "any"}},
						Config:      []ConfigField{},
					},
					{
						Type:        NodeDelay,
						Category:    "control",
						Name:        "Delay",
						Description: "Wait for a specified duration",
						Icon:        "clock",
						Color:       "#f59e0b",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "output", Name: "Output", Type: "any"}},
						Config: []ConfigField{
							{Key: "duration_ms", Label: "Duration (ms)", Type: "number", Required: true, DefaultValue: 1000},
						},
					},
				},
			},
			{
				ID:          "action",
				Name:        "Actions",
				Description: "Perform actions and transformations",
				Icon:        "zap",
				NodeDefs: []NodeTypeDefinition{
					{
						Type:        NodeHTTP,
						Category:    "action",
						Name:        "HTTP Request",
						Description: "Make an HTTP request",
						Icon:        "globe",
						Color:       "#3b82f6",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any"}},
						Outputs:     []PortDef{{ID: "success", Name: "Success", Type: "any"}, {ID: "error", Name: "Error", Type: "any"}},
						Config: []ConfigField{
							{Key: "method", Label: "Method", Type: "select", Required: true, Options: []Option{{Label: "GET", Value: "GET"}, {Label: "POST", Value: "POST"}, {Label: "PUT", Value: "PUT"}, {Label: "DELETE", Value: "DELETE"}}},
							{Key: "url", Label: "URL", Type: "string", Required: true},
							{Key: "headers", Label: "Headers", Type: "json"},
							{Key: "body", Label: "Body", Type: "json"},
							{Key: "timeout_ms", Label: "Timeout (ms)", Type: "number", DefaultValue: 30000},
						},
					},
					{
						Type:        NodeWebhook,
						Category:    "action",
						Name:        "Send Webhook",
						Description: "Send to a webhook endpoint",
						Icon:        "send",
						Color:       "#3b82f6",
						Inputs:      []PortDef{{ID: "input", Name: "Payload", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "success", Name: "Success", Type: "any"}, {ID: "error", Name: "Error", Type: "any"}},
						Config: []ConfigField{
							{Key: "endpoint_id", Label: "Endpoint", Type: "string", Required: true},
							{Key: "headers", Label: "Additional Headers", Type: "json"},
						},
					},
					{
						Type:        NodeTransform,
						Category:    "action",
						Name:        "Transform",
						Description: "Transform data with JavaScript",
						Icon:        "code",
						Color:       "#3b82f6",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "output", Name: "Output", Type: "any"}},
						Config: []ConfigField{
							{Key: "script", Label: "Transform Script", Type: "expression", Required: true, Description: "JavaScript code to transform the input"},
						},
					},
					{
						Type:        NodeFilter,
						Category:    "action",
						Name:        "Filter",
						Description: "Filter items based on condition",
						Icon:        "filter",
						Color:       "#3b82f6",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "array", Required: true}},
						Outputs:     []PortDef{{ID: "output", Name: "Output", Type: "array"}},
						Config: []ConfigField{
							{Key: "expression", Label: "Filter Expression", Type: "expression", Required: true},
						},
					},
				},
			},
			{
				ID:          "integration",
				Name:        "Integrations",
				Description: "Connect to external services",
				Icon:        "plug",
				NodeDefs: []NodeTypeDefinition{
					{
						Type:        NodeSlack,
						Category:    "integration",
						Name:        "Slack",
						Description: "Send message to Slack",
						Icon:        "slack",
						Color:       "#10b981",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any"}},
						Outputs:     []PortDef{{ID: "output", Name: "Output", Type: "any"}},
						Config: []ConfigField{
							{Key: "webhook_url", Label: "Webhook URL", Type: "string", Required: true},
							{Key: "channel", Label: "Channel", Type: "string"},
							{Key: "message", Label: "Message", Type: "expression", Required: true},
						},
					},
					{
						Type:        NodeEmail,
						Category:    "integration",
						Name:        "Email",
						Description: "Send an email",
						Icon:        "mail",
						Color:       "#10b981",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any"}},
						Outputs:     []PortDef{{ID: "output", Name: "Output", Type: "any"}},
						Config: []ConfigField{
							{Key: "to", Label: "To", Type: "string", Required: true},
							{Key: "subject", Label: "Subject", Type: "expression", Required: true},
							{Key: "body", Label: "Body", Type: "expression", Required: true},
							{Key: "html", Label: "HTML Email", Type: "boolean", DefaultValue: false},
						},
					},
					{
						Type:        NodeDatabase,
						Category:    "integration",
						Name:        "Database",
						Description: "Query a database",
						Icon:        "database",
						Color:       "#10b981",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any"}},
						Outputs:     []PortDef{{ID: "output", Name: "Output", Type: "any"}, {ID: "error", Name: "Error", Type: "any"}},
						Config: []ConfigField{
							{Key: "connection_id", Label: "Connection", Type: "string", Required: true},
							{Key: "operation", Label: "Operation", Type: "select", Required: true, Options: []Option{{Label: "Query", Value: "query"}, {Label: "Insert", Value: "insert"}, {Label: "Update", Value: "update"}, {Label: "Delete", Value: "delete"}}},
							{Key: "query", Label: "Query/Table", Type: "expression", Required: true},
							{Key: "parameters", Label: "Parameters", Type: "json"},
						},
					},
					{
						Type:        NodeQueue,
						Category:    "integration",
						Name:        "Queue",
						Description: "Send to a message queue",
						Icon:        "inbox",
						Color:       "#10b981",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "output", Name: "Output", Type: "any"}},
						Config: []ConfigField{
							{Key: "queue_name", Label: "Queue Name", Type: "string", Required: true},
							{Key: "message", Label: "Message", Type: "expression", Required: true},
						},
					},
				},
			},
			{
				ID:          "error",
				Name:        "Error Handling",
				Description: "Handle errors gracefully",
				Icon:        "alert-triangle",
				NodeDefs: []NodeTypeDefinition{
					{
						Type:        NodeTryCatch,
						Category:    "error",
						Name:        "Try/Catch",
						Description: "Catch and handle errors",
						Icon:        "shield",
						Color:       "#ef4444",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "try", Name: "Try", Type: "any"}, {ID: "catch", Name: "Catch", Type: "any"}},
						Config:      []ConfigField{},
					},
					{
						Type:        NodeRetry,
						Category:    "error",
						Name:        "Retry",
						Description: "Retry on failure",
						Icon:        "refresh-cw",
						Color:       "#ef4444",
						Inputs:      []PortDef{{ID: "input", Name: "Input", Type: "any", Required: true}},
						Outputs:     []PortDef{{ID: "success", Name: "Success", Type: "any"}, {ID: "exhausted", Name: "Exhausted", Type: "any"}},
						Config: []ConfigField{
							{Key: "max_retries", Label: "Max Retries", Type: "number", Required: true, DefaultValue: 3},
							{Key: "delay_ms", Label: "Retry Delay (ms)", Type: "number", DefaultValue: 1000},
							{Key: "backoff", Label: "Exponential Backoff", Type: "boolean", DefaultValue: true},
						},
					},
				},
			},
		},
	}
}

// GetBuiltInTemplates returns built-in workflow templates
func GetBuiltInTemplates() []Template {
	return []Template{
		{
			ID:          "slack-webhook-notification",
			Name:        "Slack Webhook Notification",
			Slug:        "slack-webhook-notification",
			Description: "Send webhook events to Slack with formatted messages",
			Category:    CategoryNotification,
			Tags:        []string{"slack", "notification", "alerts"},
			Author:      "WaaS Platform",
			AuthorVerified: true,
			Version:     "1.0.0",
			Official:    true,
			Public:      true,
			Featured:    true,
			Workflow: &WorkflowDef{
				Name:        "Slack Notification",
				Description: "Forward webhook events to Slack",
				Trigger: TriggerConfig{
					Type: TriggerWebhook,
				},
				Nodes: []NodeDef{
					{
						ID:       "start",
						Type:     NodeStart,
						Name:     "Webhook Received",
						Position: Position{X: 100, Y: 200},
					},
					{
						ID:       "transform",
						Type:     NodeTransform,
						Name:     "Format Message",
						Position: Position{X: 300, Y: 200},
						Config:   json.RawMessage(`{"script": "return { text: '*New Event*\\n' + JSON.stringify(input, null, 2) }"}`),
					},
					{
						ID:       "slack",
						Type:     NodeSlack,
						Name:     "Send to Slack",
						Position: Position{X: 500, Y: 200},
						Config:   json.RawMessage(`{"webhook_url": "{{SLACK_WEBHOOK_URL}}", "message": "{{input.text}}"}`),
					},
					{
						ID:       "end",
						Type:     NodeEnd,
						Name:     "End",
						Position: Position{X: 700, Y: 200},
					},
				},
				Edges: []Edge{
					{ID: "e1", Source: "start", SourcePort: "output", Target: "transform", TargetPort: "input"},
					{ID: "e2", Source: "transform", SourcePort: "output", Target: "slack", TargetPort: "input"},
					{ID: "e3", Source: "slack", SourcePort: "output", Target: "end", TargetPort: "input"},
				},
				Settings: WorkflowSettings{
					TimeoutSeconds: 30,
					MaxRetries:     3,
				},
			},
			Variables: []TemplateVar{
				{Name: "SLACK_WEBHOOK_URL", Label: "Slack Webhook URL", Type: "string", Required: true, Secret: true},
			},
		},
		{
			ID:          "conditional-routing",
			Name:        "Conditional Routing",
			Slug:        "conditional-routing",
			Description: "Route webhooks to different endpoints based on event type",
			Category:    CategoryDataPipeline,
			Tags:        []string{"routing", "conditional", "fan-out"},
			Author:      "WaaS Platform",
			AuthorVerified: true,
			Version:     "1.0.0",
			Official:    true,
			Public:      true,
			Workflow: &WorkflowDef{
				Name:        "Conditional Router",
				Description: "Route events based on type",
				Trigger: TriggerConfig{
					Type: TriggerWebhook,
				},
				Nodes: []NodeDef{
					{
						ID:       "start",
						Type:     NodeStart,
						Name:     "Start",
						Position: Position{X: 100, Y: 200},
					},
					{
						ID:       "condition",
						Type:     NodeCondition,
						Name:     "Check Event Type",
						Position: Position{X: 300, Y: 200},
						Config:   json.RawMessage(`{"expression": "input.event_type === 'order'"}`),
					},
					{
						ID:       "order_webhook",
						Type:     NodeWebhook,
						Name:     "Order Webhook",
						Position: Position{X: 500, Y: 100},
						Config:   json.RawMessage(`{"endpoint_id": "{{ORDER_ENDPOINT}}"}`),
					},
					{
						ID:       "other_webhook",
						Type:     NodeWebhook,
						Name:     "Other Webhook",
						Position: Position{X: 500, Y: 300},
						Config:   json.RawMessage(`{"endpoint_id": "{{OTHER_ENDPOINT}}"}`),
					},
					{
						ID:       "end",
						Type:     NodeEnd,
						Name:     "End",
						Position: Position{X: 700, Y: 200},
					},
				},
				Edges: []Edge{
					{ID: "e1", Source: "start", SourcePort: "output", Target: "condition", TargetPort: "input"},
					{ID: "e2", Source: "condition", SourcePort: "true", Target: "order_webhook", TargetPort: "input"},
					{ID: "e3", Source: "condition", SourcePort: "false", Target: "other_webhook", TargetPort: "input"},
					{ID: "e4", Source: "order_webhook", SourcePort: "success", Target: "end", TargetPort: "input"},
					{ID: "e5", Source: "other_webhook", SourcePort: "success", Target: "end", TargetPort: "input"},
				},
			},
			Variables: []TemplateVar{
				{Name: "ORDER_ENDPOINT", Label: "Order Endpoint ID", Type: "string", Required: true},
				{Name: "OTHER_ENDPOINT", Label: "Other Endpoint ID", Type: "string", Required: true},
			},
		},
		{
			ID:          "retry-with-fallback",
			Name:        "Retry with Fallback",
			Slug:        "retry-with-fallback",
			Description: "Attempt delivery with automatic retry and fallback to secondary endpoint",
			Category:    CategoryIntegration,
			Tags:        []string{"retry", "fallback", "reliability"},
			Author:      "WaaS Platform",
			AuthorVerified: true,
			Version:     "1.0.0",
			Official:    true,
			Public:      true,
			Workflow: &WorkflowDef{
				Name:        "Retry with Fallback",
				Description: "Reliable delivery with fallback",
				Trigger: TriggerConfig{
					Type: TriggerWebhook,
				},
				Nodes: []NodeDef{
					{
						ID:       "start",
						Type:     NodeStart,
						Name:     "Start",
						Position: Position{X: 100, Y: 200},
					},
					{
						ID:       "retry",
						Type:     NodeRetry,
						Name:     "Retry Primary",
						Position: Position{X: 300, Y: 200},
						Config:   json.RawMessage(`{"max_retries": 3, "delay_ms": 1000, "backoff": true}`),
					},
					{
						ID:       "primary",
						Type:     NodeWebhook,
						Name:     "Primary Endpoint",
						Position: Position{X: 500, Y: 100},
						Config:   json.RawMessage(`{"endpoint_id": "{{PRIMARY_ENDPOINT}}"}`),
					},
					{
						ID:       "fallback",
						Type:     NodeWebhook,
						Name:     "Fallback Endpoint",
						Position: Position{X: 500, Y: 300},
						Config:   json.RawMessage(`{"endpoint_id": "{{FALLBACK_ENDPOINT}}"}`),
					},
					{
						ID:       "end",
						Type:     NodeEnd,
						Name:     "End",
						Position: Position{X: 700, Y: 200},
					},
				},
				Edges: []Edge{
					{ID: "e1", Source: "start", SourcePort: "output", Target: "retry", TargetPort: "input"},
					{ID: "e2", Source: "retry", SourcePort: "success", Target: "primary", TargetPort: "input"},
					{ID: "e3", Source: "retry", SourcePort: "exhausted", Target: "fallback", TargetPort: "input"},
					{ID: "e4", Source: "primary", SourcePort: "success", Target: "end", TargetPort: "input"},
					{ID: "e5", Source: "fallback", SourcePort: "success", Target: "end", TargetPort: "input"},
				},
			},
			Variables: []TemplateVar{
				{Name: "PRIMARY_ENDPOINT", Label: "Primary Endpoint ID", Type: "string", Required: true},
				{Name: "FALLBACK_ENDPOINT", Label: "Fallback Endpoint ID", Type: "string", Required: true},
			},
		},
	}
}
