package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PublishTimeValidator validates event payloads at publish time against
// registered schemas. This ensures only conformant events enter the system.
type PublishTimeValidator struct {
	schemas map[string]*EventSchema // key: event type slug
	configs map[string]*SchemaValidationConfig
}

// NewPublishTimeValidator creates a new publish-time validator
func NewPublishTimeValidator() *PublishTimeValidator {
	return &PublishTimeValidator{
		schemas: make(map[string]*EventSchema),
		configs: make(map[string]*SchemaValidationConfig),
	}
}

// RegisterSchema registers a schema for validation
func (v *PublishTimeValidator) RegisterSchema(eventTypeSlug string, schema *EventSchema) {
	v.schemas[eventTypeSlug] = schema
}

// SetConfig sets validation config for a tenant
func (v *PublishTimeValidator) SetConfig(tenantID uuid.UUID, config *SchemaValidationConfig) {
	v.configs[tenantID.String()] = config
}

// Validate validates a payload against the schema for its event type
func (v *PublishTimeValidator) Validate(tenantID uuid.UUID, eventTypeSlug string, payload json.RawMessage) *ValidationResult {
	result := &ValidationResult{
		EventType: eventTypeSlug,
	}

	// Check if validation is configured
	config, hasConfig := v.configs[tenantID.String()]
	if hasConfig && config.Mode == ValidationModeNone {
		result.Valid = true
		result.Mode = string(ValidationModeNone)
		return result
	}

	schema, exists := v.schemas[eventTypeSlug]
	if !exists {
		result.Valid = true
		result.Message = "no schema registered for event type"
		return result
	}

	mode := ValidationModeWarn
	if hasConfig {
		mode = config.Mode
	}
	result.Mode = string(mode)

	// Validate payload size
	if hasConfig && config.MaxPayloadBytes > 0 && len(payload) > config.MaxPayloadBytes {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"payload size %d exceeds maximum %d bytes", len(payload), config.MaxPayloadBytes))
	}

	// Validate against JSON schema
	if schema.Schema != nil {
		issues := validatePayloadAgainstSchema(payload, schema)
		result.Issues = issues

		if len(issues) > 0 {
			if mode == ValidationModeStrict {
				result.Errors = append(result.Errors, issues...)
			} else {
				result.Warnings = append(result.Warnings, issues...)
			}
		}
	}

	// Check required properties
	if schema.Properties != nil {
		propIssues := validateRequiredProperties(payload, schema.Properties)
		if len(propIssues) > 0 {
			if mode == ValidationModeStrict {
				result.Errors = append(result.Errors, propIssues...)
			} else {
				result.Warnings = append(result.Warnings, propIssues...)
			}
		}
	}

	result.Valid = len(result.Errors) == 0

	if result.Valid && len(result.Warnings) > 0 {
		result.Message = fmt.Sprintf("valid with %d warnings", len(result.Warnings))
	} else if result.Valid {
		result.Message = "valid"
	} else {
		result.Message = fmt.Sprintf("validation failed with %d errors", len(result.Errors))
	}

	return result
}

func validatePayloadAgainstSchema(payload json.RawMessage, schema *EventSchema) []string {
	var issues []string
	var schemaMap map[string]interface{}

	if err := json.Unmarshal(schema.Schema, &schemaMap); err != nil {
		return []string{"invalid schema definition"}
	}

	var data interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return []string{"payload is not valid JSON"}
	}

	// Type check
	expectedType, _ := schemaMap["type"].(string)
	switch expectedType {
	case "object":
		if _, ok := data.(map[string]interface{}); !ok {
			issues = append(issues, "expected object type")
		}
	case "array":
		if _, ok := data.([]interface{}); !ok {
			issues = append(issues, "expected array type")
		}
	}

	// Required fields check
	if required, ok := schemaMap["required"].([]interface{}); ok {
		dataMap, isMap := data.(map[string]interface{})
		if isMap {
			for _, req := range required {
				fieldName, _ := req.(string)
				if fieldName != "" {
					if _, exists := dataMap[fieldName]; !exists {
						issues = append(issues, fmt.Sprintf("missing required field: %s", fieldName))
					}
				}
			}
		}
	}

	return issues
}

func validateRequiredProperties(payload json.RawMessage, properties []SchemaProperty) []string {
	var issues []string
	var data map[string]interface{}

	if err := json.Unmarshal(payload, &data); err != nil {
		return nil
	}

	for _, prop := range properties {
		if prop.Required {
			if _, exists := data[prop.Name]; !exists {
				issues = append(issues, fmt.Sprintf("missing required property: %s", prop.Name))
			}
		}
	}

	return issues
}

// AutoDocGenerator generates documentation from event schemas automatically
type AutoDocGenerator struct{}

// NewAutoDocGenerator creates a new documentation generator
func NewAutoDocGenerator() *AutoDocGenerator {
	return &AutoDocGenerator{}
}

// GenerateMarkdown generates Markdown documentation for an event type
func (g *AutoDocGenerator) GenerateMarkdown(eventType *EventType) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", eventType.Name))

	if eventType.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", eventType.Description))
	}

	sb.WriteString("## Details\n\n")
	sb.WriteString(fmt.Sprintf("| Field | Value |\n"))
	sb.WriteString(fmt.Sprintf("|-------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| Slug | `%s` |\n", eventType.Slug))
	sb.WriteString(fmt.Sprintf("| Version | `%s` |\n", eventType.Version))
	sb.WriteString(fmt.Sprintf("| Status | %s |\n", eventType.Status))
	if eventType.Category != "" {
		sb.WriteString(fmt.Sprintf("| Category | %s |\n", eventType.Category))
	}
	if len(eventType.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("| Tags | %s |\n", strings.Join(eventType.Tags, ", ")))
	}

	if eventType.Status == StatusDeprecated && eventType.DeprecationMessage != "" {
		sb.WriteString(fmt.Sprintf("\n> ⚠️ **Deprecated**: %s\n\n", eventType.DeprecationMessage))
	}

	// Schema properties
	if eventType.Schema != nil && len(eventType.Schema.Properties) > 0 {
		sb.WriteString("\n## Payload Schema\n\n")
		sb.WriteString("| Property | Type | Required | Description |\n")
		sb.WriteString("|----------|------|----------|-------------|\n")
		g.writeProperties(&sb, eventType.Schema.Properties, "")
	}

	// Example payload
	if eventType.ExamplePayload != nil {
		sb.WriteString("\n## Example Payload\n\n")
		sb.WriteString("```json\n")
		var pretty json.RawMessage
		if err := json.Unmarshal(eventType.ExamplePayload, &pretty); err == nil {
			formatted, _ := json.MarshalIndent(pretty, "", "  ")
			sb.Write(formatted)
		} else {
			sb.Write(eventType.ExamplePayload)
		}
		sb.WriteString("\n```\n")
	}

	// Versions
	if len(eventType.Versions) > 0 {
		sb.WriteString("\n## Version History\n\n")
		sb.WriteString("| Version | Date | Breaking | Changelog |\n")
		sb.WriteString("|---------|------|----------|----------|\n")
		for _, v := range eventType.Versions {
			breaking := ""
			if v.IsBreakingChange {
				breaking = "⚠️ Yes"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				v.Version,
				v.PublishedAt.Format("2006-01-02"),
				breaking,
				v.Changelog,
			))
		}
	}

	return sb.String()
}

func (g *AutoDocGenerator) writeProperties(sb *strings.Builder, props []SchemaProperty, prefix string) {
	for _, prop := range props {
		name := prefix + prop.Name
		required := ""
		if prop.Required {
			required = "✅"
		}
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n",
			name, prop.Type, required, prop.Description))

		if len(prop.Properties) > 0 {
			g.writeProperties(sb, prop.Properties, name+".")
		}
	}
}

// GenerateOpenAPISpec generates an OpenAPI-compatible schema for the event
func (g *AutoDocGenerator) GenerateOpenAPISpec(eventType *EventType) map[string]interface{} {
	spec := map[string]interface{}{
		"type":        "object",
		"title":       eventType.Name,
		"description": eventType.Description,
	}

	if eventType.Schema != nil && eventType.Schema.Schema != nil {
		var schemaData interface{}
		if err := json.Unmarshal(eventType.Schema.Schema, &schemaData); err == nil {
			return schemaData.(map[string]interface{})
		}
	}

	return spec
}

// GenerateCatalogSite generates a full catalog documentation site
func (g *AutoDocGenerator) GenerateCatalogSite(eventTypes []*EventType, categories []*EventCategory) *CatalogSite {
	site := &CatalogSite{
		Title:       "Event Catalog",
		GeneratedAt: time.Now(),
		Pages:       make([]CatalogPage, 0),
		Navigation:  make([]NavItem, 0),
	}

	// Group events by category
	catMap := make(map[string][]*EventType)
	for _, et := range eventTypes {
		cat := et.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		catMap[cat] = append(catMap[cat], et)
	}

	// Sort categories
	catNames := make([]string, 0, len(catMap))
	for name := range catMap {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	for _, catName := range catNames {
		events := catMap[catName]
		navItem := NavItem{
			Title: catName,
			Items: make([]NavItem, 0, len(events)),
		}

		for _, et := range events {
			page := CatalogPage{
				Slug:    et.Slug,
				Title:   et.Name,
				Content: g.GenerateMarkdown(et),
			}
			site.Pages = append(site.Pages, page)
			navItem.Items = append(navItem.Items, NavItem{
				Title: et.Name,
				Slug:  et.Slug,
			})
		}

		site.Navigation = append(site.Navigation, navItem)
	}

	return site
}

// CatalogSite represents a generated documentation site
type CatalogSite struct {
	Title       string        `json:"title"`
	GeneratedAt time.Time     `json:"generated_at"`
	Pages       []CatalogPage `json:"pages"`
	Navigation  []NavItem     `json:"navigation"`
}

// CatalogPage represents a single documentation page
type CatalogPage struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// NavItem represents a navigation item
type NavItem struct {
	Title string    `json:"title"`
	Slug  string    `json:"slug,omitempty"`
	Items []NavItem `json:"items,omitempty"`
}
