package openapigen

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/josedab/waas/pkg/httputil"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

// OpenAPISpec represents a parsed OpenAPI 3.x spec
type OpenAPISpec struct {
	OpenAPI   string                 `json:"openapi"`
	Info      SpecInfo               `json:"info"`
	Webhooks  map[string]WebhookDef  `json:"webhooks,omitempty"`
	Paths     map[string]PathItem    `json:"paths,omitempty"`
	Callbacks map[string]interface{} `json:"callbacks,omitempty"`
}

// SpecInfo holds basic spec metadata.
type SpecInfo struct {
	Title       string `json:"title" yaml:"title"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
}

// WebhookDef describes a webhook entry in the spec.
type WebhookDef struct {
	Post *OperationDef `json:"post,omitempty" yaml:"post,omitempty"`
}

// OperationDef describes an operation (e.g. POST).
type OperationDef struct {
	Summary     string                 `json:"summary" yaml:"summary"`
	Description string                 `json:"description" yaml:"description"`
	OperationID string                 `json:"operationId" yaml:"operationId"`
	RequestBody *RequestBodyDef        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Tags        []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	Callbacks   map[string]interface{} `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
}

// RequestBodyDef describes a request body.
type RequestBodyDef struct {
	Content map[string]MediaTypeDef `json:"content" yaml:"content"`
}

// MediaTypeDef describes a media type entry.
type MediaTypeDef struct {
	Schema  json.RawMessage `json:"schema" yaml:"schema"`
	Example json.RawMessage `json:"example,omitempty" yaml:"example,omitempty"`
}

// PathItem describes a single path entry.
type PathItem struct {
	Post      *OperationDef          `json:"post,omitempty" yaml:"post,omitempty"`
	Callbacks map[string]interface{} `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
}

// GeneratedConfig represents generated WaaS configuration from an OpenAPI spec.
type GeneratedConfig struct {
	EventTypes   []GeneratedEventType   `json:"event_types"`
	Endpoints    []GeneratedEndpoint    `json:"endpoints"`
	Schemas      []GeneratedSchema      `json:"schemas"`
	Transforms   []GeneratedTransform   `json:"transforms,omitempty"`
	TestFixtures []GeneratedTestFixture `json:"test_fixtures,omitempty"`
}

// GeneratedEventType represents a single event type extracted from the spec.
type GeneratedEventType struct {
	Name           string          `json:"name"`
	Slug           string          `json:"slug"`
	Description    string          `json:"description"`
	Category       string          `json:"category"`
	Schema         json.RawMessage `json:"schema,omitempty"`
	ExamplePayload json.RawMessage `json:"example_payload,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
}

// GeneratedEndpoint represents a generated webhook endpoint.
type GeneratedEndpoint struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	EventTypes  []string `json:"event_types"`
}

// GeneratedSchema represents a generated JSON schema.
type GeneratedSchema struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Schema  json.RawMessage `json:"schema"`
}

// GeneratedTransform represents a payload transformation rule.
type GeneratedTransform struct {
	Name       string `json:"name"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	Expression string `json:"expression"`
}

// GeneratedTestFixture represents a test fixture for an event type.
type GeneratedTestFixture struct {
	EventType string            `json:"event_type"`
	Payload   json.RawMessage   `json:"payload"`
	Headers   map[string]string `json:"headers"`
}

// GenerateRequest is the HTTP request payload for generating config.
type GenerateRequest struct {
	SpecContent string          `json:"spec_content" binding:"required"`
	SpecURL     string          `json:"spec_url,omitempty"`
	Options     GenerateOptions `json:"options"`
}

// GenerateOptions controls what the generator produces.
type GenerateOptions struct {
	IncludeTests      bool   `json:"include_tests"`
	IncludeTransforms bool   `json:"include_transforms"`
	TargetLanguage    string `json:"target_language,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
}

// GeneratedSDKClient represents a generated SDK client.
type GeneratedSDKClient struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Package  string `json:"package_name"`
}

// ContractTestSuite for CI/CD webhook contract testing.
type ContractTestSuite struct {
	Name       string         `json:"name"`
	EventTypes []string       `json:"event_types"`
	Tests      []ContractTest `json:"tests"`
}

// ContractTest represents a single contract test case.
type ContractTest struct {
	Name         string          `json:"name"`
	EventType    string          `json:"event_type"`
	Payload      json.RawMessage `json:"payload"`
	ExpectValid  bool            `json:"expect_valid"`
	ExpectFields []string        `json:"expect_fields,omitempty"`
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service provides OpenAPI-to-Webhook generation logic.
type Service struct{}

// NewService creates a new openapigen service.
func NewService() *Service {
	return &Service{}
}

// ParseSpec parses a JSON or YAML OpenAPI 3.x spec.
func (s *Service) ParseSpec(content []byte) (*OpenAPISpec, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("empty spec content")
	}

	// Normalise to JSON via YAML (YAML is a superset of JSON).
	var raw map[string]interface{}
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to convert spec to JSON: %w", err)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(jsonBytes, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	if spec.OpenAPI == "" {
		return nil, fmt.Errorf("missing openapi version field")
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		return nil, fmt.Errorf("unsupported openapi version: %s (only 3.x is supported)", spec.OpenAPI)
	}

	return &spec, nil
}

// GenerateFromSpec generates WaaS configuration from a parsed OpenAPI spec.
func (s *Service) GenerateFromSpec(ctx context.Context, spec *OpenAPISpec, opts GenerateOptions) (*GeneratedConfig, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec is nil")
	}

	eventTypes := s.extractWebhooks(spec)
	eventTypes = append(eventTypes, s.extractCallbacks(spec)...)

	if opts.Namespace != "" {
		for i := range eventTypes {
			eventTypes[i].Slug = opts.Namespace + "." + eventTypes[i].Slug
		}
	}

	// Build schemas from event types that carry a schema.
	var schemas []GeneratedSchema
	for _, et := range eventTypes {
		if len(et.Schema) > 0 {
			schemas = append(schemas, GeneratedSchema{
				Name:    et.Slug,
				Version: "1.0.0",
				Schema:  et.Schema,
			})
		}
	}

	// Build a single default endpoint subscribing to all event types.
	slugs := make([]string, 0, len(eventTypes))
	for _, et := range eventTypes {
		slugs = append(slugs, et.Slug)
	}
	var endpoints []GeneratedEndpoint
	if len(slugs) > 0 {
		endpoints = append(endpoints, GeneratedEndpoint{
			Name:        spec.Info.Title + " Webhook Endpoint",
			URL:         "https://example.com/webhooks",
			Description: "Auto-generated endpoint for " + spec.Info.Title,
			EventTypes:  slugs,
		})
	}

	config := &GeneratedConfig{
		EventTypes: eventTypes,
		Endpoints:  endpoints,
		Schemas:    schemas,
	}

	if opts.IncludeTransforms {
		config.Transforms = s.GenerateTransformTemplates(config)
	}

	if opts.IncludeTests {
		config.TestFixtures = s.generateTestFixtures(eventTypes)
	}

	return config, nil
}

// GenerateSDKClient generates a minimal SDK client for the given language.
func (s *Service) GenerateSDKClient(ctx context.Context, config *GeneratedConfig, language string) (*GeneratedSDKClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	switch strings.ToLower(language) {
	case "go":
		return s.generateGoSDK(config)
	case "python":
		return s.generatePythonSDK(config)
	case "typescript":
		return s.generateTypeScriptSDK(config)
	default:
		return nil, fmt.Errorf("unsupported language: %s (supported: go, python, typescript)", language)
	}
}

// GenerateContractTests generates a contract test suite from generated config.
func (s *Service) GenerateContractTests(ctx context.Context, config *GeneratedConfig) (*ContractTestSuite, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	slugs := make([]string, 0, len(config.EventTypes))
	var tests []ContractTest
	for _, et := range config.EventTypes {
		slugs = append(slugs, et.Slug)

		payload := et.ExamplePayload
		if len(payload) == 0 {
			payload = json.RawMessage(`{"event":"` + et.Slug + `","data":{}}`)
		}

		// Extract expected field names from schema properties
		var expectFields []string
		if len(et.Schema) > 0 {
			var schemaMap map[string]interface{}
			if err := json.Unmarshal(et.Schema, &schemaMap); err == nil {
				if props, ok := schemaMap["properties"].(map[string]interface{}); ok {
					for fieldName := range props {
						expectFields = append(expectFields, fieldName)
					}
				}
			}
		}

		tests = append(tests, ContractTest{
			Name:         "valid_" + et.Slug,
			EventType:    et.Slug,
			Payload:      payload,
			ExpectValid:  true,
			ExpectFields: expectFields,
		})

		// Negative test with empty payload.
		tests = append(tests, ContractTest{
			Name:        "invalid_empty_" + et.Slug,
			EventType:   et.Slug,
			Payload:     json.RawMessage(`{}`),
			ExpectValid: false,
		})
	}

	return &ContractTestSuite{
		Name:       "Contract tests",
		EventTypes: slugs,
		Tests:      tests,
	}, nil
}

// GenerateTransformTemplates generates common transformation templates for each event type.
func (s *Service) GenerateTransformTemplates(config *GeneratedConfig) []GeneratedTransform {
	var transforms []GeneratedTransform
	for _, et := range config.EventTypes {
		transforms = append(transforms, GeneratedTransform{
			Name:       "flatten_" + et.Slug,
			Source:     et.Slug,
			Target:     et.Slug + ".flat",
			Expression: "Object.keys(payload).reduce((acc, k) => ({...acc, [k]: payload[k]}), {})",
		})
	}
	return transforms
}

// ---------------------------------------------------------------------------
// Internal extraction helpers
// ---------------------------------------------------------------------------

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func toSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonAlphaNum.ReplaceAllString(s, ".")
	return strings.Trim(s, ".")
}

func (s *Service) extractWebhooks(spec *OpenAPISpec) []GeneratedEventType {
	var out []GeneratedEventType
	for name, wh := range spec.Webhooks {
		if wh.Post == nil {
			continue
		}
		et := GeneratedEventType{
			Name:        name,
			Slug:        toSlug(name),
			Description: wh.Post.Description,
			Category:    "webhook",
			Tags:        wh.Post.Tags,
		}
		if wh.Post.Summary != "" && et.Description == "" {
			et.Description = wh.Post.Summary
		}
		if wh.Post.RequestBody != nil {
			if mt, ok := wh.Post.RequestBody.Content["application/json"]; ok {
				et.Schema = mt.Schema
				et.ExamplePayload = mt.Example
			}
		}
		out = append(out, et)
	}
	return out
}

func (s *Service) extractCallbacks(spec *OpenAPISpec) []GeneratedEventType {
	var out []GeneratedEventType
	for path, item := range spec.Paths {
		// Collect callbacks from both path-item level and operation level.
		callbacks := item.Callbacks
		if item.Post != nil && len(item.Post.Callbacks) > 0 {
			if callbacks == nil {
				callbacks = make(map[string]interface{})
			}
			for k, v := range item.Post.Callbacks {
				callbacks[k] = v
			}
		}
		if len(callbacks) == 0 {
			continue
		}
		for cbName := range callbacks {
			slug := toSlug(cbName)
			if slug == "" {
				slug = toSlug(path + "." + cbName)
			}
			et := GeneratedEventType{
				Name:     cbName,
				Slug:     slug,
				Category: "callback",
			}
			if item.Post != nil {
				et.Description = item.Post.Description
				et.Tags = item.Post.Tags
			}
			out = append(out, et)
		}
	}
	return out
}

func (s *Service) generateTestFixtures(eventTypes []GeneratedEventType) []GeneratedTestFixture {
	var fixtures []GeneratedTestFixture
	for _, et := range eventTypes {
		payload := et.ExamplePayload
		if len(payload) == 0 {
			payload = json.RawMessage(`{"event":"` + et.Slug + `","data":{}}`)
		}
		fixtures = append(fixtures, GeneratedTestFixture{
			EventType: et.Slug,
			Payload:   payload,
			Headers: map[string]string{
				"Content-Type":    "application/json",
				"X-Webhook-Event": et.Slug,
			},
		})
	}
	return fixtures
}

// ---------------------------------------------------------------------------
// SDK generation helpers
// ---------------------------------------------------------------------------

func (s *Service) generateGoSDK(config *GeneratedConfig) (*GeneratedSDKClient, error) {
	var b strings.Builder
	b.WriteString("package webhooks\n\n")
	b.WriteString("// Auto-generated webhook types\n\n")
	for _, et := range config.EventTypes {
		typeName := goTypeName(et.Name)
		b.WriteString(fmt.Sprintf("// %s represents the %s event.\n", typeName, et.Name))
		b.WriteString(fmt.Sprintf("type %s struct {\n\tEvent string `json:\"event\"`\n\tData  map[string]interface{} `json:\"data\"`\n}\n\n", typeName))
	}
	return &GeneratedSDKClient{Language: "go", Code: b.String(), Package: "webhooks"}, nil
}

func (s *Service) generatePythonSDK(config *GeneratedConfig) (*GeneratedSDKClient, error) {
	var b strings.Builder
	b.WriteString("\"\"\"Auto-generated webhook types.\"\"\"\nfrom dataclasses import dataclass, field\nfrom typing import Any, Dict\n\n")
	for _, et := range config.EventTypes {
		className := pythonClassName(et.Name)
		b.WriteString(fmt.Sprintf("@dataclass\nclass %s:\n    \"\"\"Represents the %s event.\"\"\"\n    event: str = \"%s\"\n    data: Dict[str, Any] = field(default_factory=dict)\n\n", className, et.Name, et.Slug))
	}
	return &GeneratedSDKClient{Language: "python", Code: b.String(), Package: "webhooks"}, nil
}

func (s *Service) generateTypeScriptSDK(config *GeneratedConfig) (*GeneratedSDKClient, error) {
	var b strings.Builder
	b.WriteString("// Auto-generated webhook types\n\n")
	for _, et := range config.EventTypes {
		ifName := tsInterfaceName(et.Name)
		b.WriteString(fmt.Sprintf("export interface %s {\n  event: string;\n  data: Record<string, unknown>;\n}\n\n", ifName))
	}
	return &GeneratedSDKClient{Language: "typescript", Code: b.String(), Package: "webhooks"}, nil
}

func goTypeName(name string) string {
	parts := nonAlphaNum.Split(strings.ToLower(name), -1)
	var out string
	for _, p := range parts {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	if out == "" {
		return "WebhookEvent"
	}
	return out
}

func pythonClassName(name string) string {
	return goTypeName(name) // Same PascalCase logic works for Python.
}

func tsInterfaceName(name string) string {
	return goTypeName(name)
}

// ---------------------------------------------------------------------------
// Handler (HTTP)
// ---------------------------------------------------------------------------

// Handler provides HTTP handlers for OpenAPI generation.
type Handler struct {
	service *Service
}

// NewHandler creates a new openapigen handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers openapigen routes on the given router group.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	og := router.Group("/openapi")
	{
		og.POST("/parse", h.ParseSpec)
		og.POST("/generate", h.GenerateConfig)
		og.POST("/generate-sdk", h.GenerateSDK)
		og.POST("/generate-tests", h.GenerateTests)
	}
}

// @Summary Parse and validate OpenAPI spec
// @Tags OpenAPIGen
// @Accept json
// @Produce json
// @Success 200 {object} OpenAPISpec
// @Router /openapi/parse [post]
func (h *Handler) ParseSpec(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	spec, err := h.service.ParseSpec([]byte(req.SpecContent))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, spec)
}

// @Summary Generate WaaS config from OpenAPI spec
// @Tags OpenAPIGen
// @Accept json
// @Produce json
// @Success 200 {object} GeneratedConfig
// @Router /openapi/generate [post]
func (h *Handler) GenerateConfig(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	spec, err := h.service.ParseSpec([]byte(req.SpecContent))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.GenerateFromSpec(c.Request.Context(), spec, req.Options)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary Generate SDK client from OpenAPI spec
// @Tags OpenAPIGen
// @Accept json
// @Produce json
// @Success 200 {object} GeneratedSDKClient
// @Router /openapi/generate-sdk [post]
func (h *Handler) GenerateSDK(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	spec, err := h.service.ParseSpec([]byte(req.SpecContent))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.GenerateFromSpec(c.Request.Context(), spec, req.Options)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	lang := req.Options.TargetLanguage
	if lang == "" {
		lang = "go"
	}

	sdk, err := h.service.GenerateSDKClient(c.Request.Context(), config, lang)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, sdk)
}

// @Summary Generate contract tests from OpenAPI spec
// @Tags OpenAPIGen
// @Accept json
// @Produce json
// @Success 200 {object} ContractTestSuite
// @Router /openapi/generate-tests [post]
func (h *Handler) GenerateTests(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	spec, err := h.service.ParseSpec([]byte(req.SpecContent))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.GenerateFromSpec(c.Request.Context(), spec, req.Options)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	suite, err := h.service.GenerateContractTests(c.Request.Context(), config)
	if err != nil {
		httputil.InternalErrorGeneric(c, err)
		return
	}

	c.JSON(http.StatusOK, suite)
}
