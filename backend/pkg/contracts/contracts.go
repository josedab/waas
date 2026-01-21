// Package contracts provides consumer-driven contract testing
package contracts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

var (
	ErrContractNotFound      = errors.New("contract not found")
	ErrValidationFailed      = errors.New("contract validation failed")
	ErrBreakingChangeDetected = errors.New("breaking change detected")
	ErrInvalidSchema         = errors.New("invalid schema format")
)

// Contract represents a webhook contract definition
type Contract struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	Version         string          `json:"version"`
	ProviderName    string          `json:"provider_name"`
	ConsumerName    string          `json:"consumer_name,omitempty"`
	EventType       string          `json:"event_type"`
	SchemaFormat    string          `json:"schema_format"`
	RequestSchema   json.RawMessage `json:"request_schema"`
	ResponseSchema  json.RawMessage `json:"response_schema,omitempty"`
	RequiredHeaders []string        `json:"required_headers,omitempty"`
	OptionalHeaders []string        `json:"optional_headers,omitempty"`
	Status          string          `json:"status"`
	PublishedAt     *time.Time      `json:"published_at,omitempty"`
	DeprecatedAt    *time.Time      `json:"deprecated_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ContractVersion represents an immutable version snapshot
type ContractVersion struct {
	ID              string          `json:"id"`
	ContractID      string          `json:"contract_id"`
	Version         string          `json:"version"`
	SchemaSnapshot  json.RawMessage `json:"schema_snapshot"`
	HeadersSnapshot json.RawMessage `json:"headers_snapshot,omitempty"`
	ChangeSummary   string          `json:"change_summary,omitempty"`
	BreakingChanges []string        `json:"breaking_changes,omitempty"`
	PublishedBy     string          `json:"published_by,omitempty"`
	PublishedAt     time.Time       `json:"published_at"`
}

// ValidationResult represents the result of contract validation
type ValidationResult struct {
	ID              string                   `json:"id"`
	ContractID      string                   `json:"contract_id"`
	ContractVersion string                   `json:"contract_version"`
	ValidationType  string                   `json:"validation_type"`
	Source          string                   `json:"source,omitempty"`
	Payload         json.RawMessage          `json:"payload"`
	Headers         map[string]string        `json:"headers,omitempty"`
	IsValid         bool                     `json:"is_valid"`
	Errors          []ValidationError        `json:"errors,omitempty"`
	Warnings        []ValidationWarning      `json:"warnings,omitempty"`
	ValidationTimeMs int                     `json:"validation_time_ms"`
	ValidatedAt     time.Time                `json:"validated_at"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// BreakingChange represents a detected breaking change
type BreakingChange struct {
	ID               string    `json:"id"`
	ContractID       string    `json:"contract_id"`
	OldVersion       string    `json:"old_version"`
	NewVersion       string    `json:"new_version"`
	HasBreakingChanges bool    `json:"has_breaking_changes"`
	ChangeType       string    `json:"change_type,omitempty"`
	Changes          []Change  `json:"changes"`
	AffectedFields   []string  `json:"affected_fields,omitempty"`
	ImpactAssessment string    `json:"impact_assessment,omitempty"`
	MigrationSteps   []string  `json:"migration_steps,omitempty"`
	DetectedAt       time.Time `json:"detected_at"`
}

// Change represents a single change between versions
type Change struct {
	Type        string      `json:"type"`
	Path        string      `json:"path"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	IsBreaking  bool        `json:"is_breaking"`
	Description string      `json:"description"`
}

// Subscription represents a consumer subscription to a contract
type Subscription struct {
	ID                   string     `json:"id"`
	ContractID           string     `json:"contract_id"`
	ConsumerID           string     `json:"consumer_id"`
	ConsumerName         string     `json:"consumer_name,omitempty"`
	EndpointID           *string    `json:"endpoint_id,omitempty"`
	SubscribedVersion    string     `json:"subscribed_version"`
	NotificationEmail    string     `json:"notification_email,omitempty"`
	NotifyOnBreaking     bool       `json:"notify_on_breaking_changes"`
	NotifyOnDeprecation  bool       `json:"notify_on_deprecation"`
	Status               string     `json:"status"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// TestSuite represents a contract test suite
type TestSuite struct {
	ID             string      `json:"id"`
	ContractID     string      `json:"contract_id"`
	Name           string      `json:"name"`
	Description    string      `json:"description,omitempty"`
	TestCases      []TestCase  `json:"test_cases"`
	SetupScript    string      `json:"setup_script,omitempty"`
	TeardownScript string      `json:"teardown_script,omitempty"`
	TimeoutSeconds int         `json:"timeout_seconds"`
	RetryCount     int         `json:"retry_count"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// TestCase represents a single test case
type TestCase struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Input          json.RawMessage        `json:"input"`
	ExpectedOutput json.RawMessage        `json:"expected_output,omitempty"`
	Assertions     []Assertion            `json:"assertions,omitempty"`
	Skip           bool                   `json:"skip,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
}

// Assertion represents a test assertion
type Assertion struct {
	Type     string      `json:"type"` // equals, contains, matches, type_is
	Path     string      `json:"path"`
	Expected interface{} `json:"expected"`
}

// TestResult represents test execution results
type TestResult struct {
	ID           string           `json:"id"`
	TestSuiteID  string           `json:"test_suite_id"`
	RunID        string           `json:"run_id"`
	Environment  string           `json:"environment,omitempty"`
	TriggeredBy  string           `json:"triggered_by,omitempty"`
	TotalTests   int              `json:"total_tests"`
	Passed       int              `json:"passed"`
	Failed       int              `json:"failed"`
	Skipped      int              `json:"skipped"`
	Results      []CaseResult     `json:"results"`
	DurationMs   int              `json:"duration_ms"`
	Status       string           `json:"status"`
	ErrorMessage string           `json:"error_message,omitempty"`
	StartedAt    time.Time        `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

// CaseResult represents results for a single test case
type CaseResult struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"` // passed, failed, skipped, error
	DurationMs int               `json:"duration_ms"`
	Error      string            `json:"error,omitempty"`
	Assertions []AssertionResult `json:"assertions,omitempty"`
}

// AssertionResult represents result of a single assertion
type AssertionResult struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	Passed   bool   `json:"passed"`
	Expected string `json:"expected"`
	Actual   string `json:"actual,omitempty"`
	Message  string `json:"message,omitempty"`
}

// Validator provides contract validation
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidatePayload validates a payload against a contract
func (v *Validator) ValidatePayload(contract *Contract, payload json.RawMessage, headers map[string]string) *ValidationResult {
	start := time.Now()
	result := &ValidationResult{
		ContractID:      contract.ID,
		ContractVersion: contract.Version,
		ValidationType:  "schema",
		Payload:         payload,
		Headers:         headers,
		IsValid:         true,
		ValidatedAt:     time.Now(),
	}

	// Validate required headers
	for _, h := range contract.RequiredHeaders {
		if _, ok := headers[h]; !ok {
			result.Errors = append(result.Errors, ValidationError{
				Path:    fmt.Sprintf("headers.%s", h),
				Message: fmt.Sprintf("required header '%s' is missing", h),
				Code:    "MISSING_HEADER",
			})
		}
	}

	// Parse and validate schema
	var schema map[string]interface{}
	if err := json.Unmarshal(contract.RequestSchema, &schema); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:    "$",
			Message: "invalid schema format",
			Code:    "INVALID_SCHEMA",
		})
		result.IsValid = false
		result.ValidationTimeMs = int(time.Since(start).Milliseconds())
		return result
	}

	// Validate payload against schema
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Path:    "$",
			Message: "invalid JSON payload",
			Code:    "INVALID_JSON",
		})
		result.IsValid = false
		result.ValidationTimeMs = int(time.Since(start).Milliseconds())
		return result
	}

	// Validate required properties
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		if required, ok := schema["required"].([]interface{}); ok {
			for _, r := range required {
				fieldName := r.(string)
				if _, exists := data[fieldName]; !exists {
					result.Errors = append(result.Errors, ValidationError{
						Path:    fmt.Sprintf("$.%s", fieldName),
						Message: fmt.Sprintf("required field '%s' is missing", fieldName),
						Code:    "MISSING_FIELD",
					})
				}
			}
		}

		// Validate field types
		for fieldName, fieldValue := range data {
			if propSchema, ok := props[fieldName].(map[string]interface{}); ok {
				if expectedType, ok := propSchema["type"].(string); ok {
					actualType := v.getJSONType(fieldValue)
					if !v.typesCompatible(expectedType, actualType) {
						result.Errors = append(result.Errors, ValidationError{
							Path:    fmt.Sprintf("$.%s", fieldName),
							Message: fmt.Sprintf("expected type '%s', got '%s'", expectedType, actualType),
							Code:    "TYPE_MISMATCH",
						})
					}
				}
			}
		}
	}

	result.IsValid = len(result.Errors) == 0
	result.ValidationTimeMs = int(time.Since(start).Milliseconds())
	return result
}

func (v *Validator) getJSONType(value interface{}) string {
	if value == nil {
		return "null"
	}
	switch reflect.TypeOf(value).Kind() {
	case reflect.String:
		return "string"
	case reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return "unknown"
	}
}

func (v *Validator) typesCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	// integer is also a number
	if expected == "integer" && actual == "number" {
		return true
	}
	return false
}

// BreakingChangeDetector detects breaking changes between versions
type BreakingChangeDetector struct{}

// NewBreakingChangeDetector creates a new detector
func NewBreakingChangeDetector() *BreakingChangeDetector {
	return &BreakingChangeDetector{}
}

// DetectChanges compares two schema versions and detects breaking changes
func (d *BreakingChangeDetector) DetectChanges(oldSchema, newSchema json.RawMessage) (*BreakingChange, error) {
	var oldMap, newMap map[string]interface{}

	if err := json.Unmarshal(oldSchema, &oldMap); err != nil {
		return nil, fmt.Errorf("failed to parse old schema: %w", err)
	}
	if err := json.Unmarshal(newSchema, &newMap); err != nil {
		return nil, fmt.Errorf("failed to parse new schema: %w", err)
	}

	changes := d.compareSchemas(oldMap, newMap, "$")

	result := &BreakingChange{
		Changes:    changes,
		DetectedAt: time.Now(),
	}

	// Determine if any changes are breaking
	var affectedFields []string
	for _, c := range changes {
		if c.IsBreaking {
			result.HasBreakingChanges = true
			affectedFields = append(affectedFields, c.Path)
		}
	}
	result.AffectedFields = affectedFields

	// Generate migration steps
	if result.HasBreakingChanges {
		result.MigrationSteps = d.generateMigrationSteps(changes)
		result.ImpactAssessment = d.assessImpact(changes)
	}

	return result, nil
}

func (d *BreakingChangeDetector) compareSchemas(old, new map[string]interface{}, path string) []Change {
	var changes []Change

	// Check for removed fields (breaking)
	oldProps := d.getProperties(old)
	newProps := d.getProperties(new)

	for field := range oldProps {
		if _, exists := newProps[field]; !exists {
			changes = append(changes, Change{
				Type:        "field_removed",
				Path:        fmt.Sprintf("%s.%s", path, field),
				OldValue:    oldProps[field],
				IsBreaking:  true,
				Description: fmt.Sprintf("Field '%s' was removed", field),
			})
		}
	}

	// Check for added required fields (breaking)
	oldRequired := d.getRequired(old)
	newRequired := d.getRequired(new)

	for _, field := range newRequired {
		if !contains(oldRequired, field) {
			if _, existed := oldProps[field]; !existed {
				changes = append(changes, Change{
					Type:        "required_added",
					Path:        fmt.Sprintf("%s.%s", path, field),
					IsBreaking:  true,
					Description: fmt.Sprintf("New required field '%s' was added", field),
				})
			}
		}
	}

	// Check for type changes (breaking)
	for field, oldProp := range oldProps {
		if newProp, exists := newProps[field]; exists {
			oldType := d.getType(oldProp)
			newType := d.getType(newProp)
			if oldType != newType {
				changes = append(changes, Change{
					Type:        "type_changed",
					Path:        fmt.Sprintf("%s.%s", path, field),
					OldValue:    oldType,
					NewValue:    newType,
					IsBreaking:  true,
					Description: fmt.Sprintf("Field '%s' type changed from '%s' to '%s'", field, oldType, newType),
				})
			}
		}
	}

	// Check for new optional fields (non-breaking)
	for field := range newProps {
		if _, existed := oldProps[field]; !existed && !contains(newRequired, field) {
			changes = append(changes, Change{
				Type:        "field_added",
				Path:        fmt.Sprintf("%s.%s", path, field),
				NewValue:    newProps[field],
				IsBreaking:  false,
				Description: fmt.Sprintf("Optional field '%s' was added", field),
			})
		}
	}

	return changes
}

func (d *BreakingChangeDetector) getProperties(schema map[string]interface{}) map[string]interface{} {
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		return props
	}
	return make(map[string]interface{})
}

func (d *BreakingChangeDetector) getRequired(schema map[string]interface{}) []string {
	var result []string
	if req, ok := schema["required"].([]interface{}); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}

func (d *BreakingChangeDetector) getType(prop interface{}) string {
	if m, ok := prop.(map[string]interface{}); ok {
		if t, ok := m["type"].(string); ok {
			return t
		}
	}
	return "unknown"
}

func (d *BreakingChangeDetector) generateMigrationSteps(changes []Change) []string {
	var steps []string
	for _, c := range changes {
		if c.IsBreaking {
			switch c.Type {
			case "field_removed":
				steps = append(steps, fmt.Sprintf("Remove usage of field at '%s' before upgrading", c.Path))
			case "required_added":
				steps = append(steps, fmt.Sprintf("Ensure field '%s' is always provided in requests", c.Path))
			case "type_changed":
				steps = append(steps, fmt.Sprintf("Update field '%s' to use new type '%v'", c.Path, c.NewValue))
			}
		}
	}
	return steps
}

func (d *BreakingChangeDetector) assessImpact(changes []Change) string {
	breakingCount := 0
	for _, c := range changes {
		if c.IsBreaking {
			breakingCount++
		}
	}

	if breakingCount == 0 {
		return "No breaking changes detected. Safe to upgrade."
	} else if breakingCount <= 2 {
		return "Minor breaking changes detected. Review migration steps before upgrading."
	}
	return "Significant breaking changes detected. Careful migration planning required."
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// TestRunner executes contract test suites
type TestRunner struct {
	validator *Validator
}

// NewTestRunner creates a new test runner
func NewTestRunner() *TestRunner {
	return &TestRunner{
		validator: NewValidator(),
	}
}

// RunSuite executes a test suite
func (r *TestRunner) RunSuite(ctx context.Context, suite *TestSuite, contract *Contract) *TestResult {
	start := time.Now()
	result := &TestResult{
		TestSuiteID: suite.ID,
		RunID:       fmt.Sprintf("run-%d", time.Now().UnixNano()),
		TotalTests:  len(suite.TestCases),
		StartedAt:   start,
		Status:      "running",
	}

	for _, tc := range suite.TestCases {
		if tc.Skip {
			result.Skipped++
			result.Results = append(result.Results, CaseResult{
				Name:   tc.Name,
				Status: "skipped",
			})
			continue
		}

		caseResult := r.runTestCase(ctx, tc, contract)
		result.Results = append(result.Results, caseResult)

		if caseResult.Status == "passed" {
			result.Passed++
		} else {
			result.Failed++
		}
	}

	now := time.Now()
	result.CompletedAt = &now
	result.DurationMs = int(time.Since(start).Milliseconds())

	if result.Failed > 0 {
		result.Status = "failed"
	} else {
		result.Status = "passed"
	}

	return result
}

func (r *TestRunner) runTestCase(ctx context.Context, tc TestCase, contract *Contract) CaseResult {
	start := time.Now()
	result := CaseResult{
		Name:   tc.Name,
		Status: "passed",
	}

	// Validate input against contract
	validationResult := r.validator.ValidatePayload(contract, tc.Input, nil)
	if !validationResult.IsValid {
		result.Status = "failed"
		var errs []string
		for _, e := range validationResult.Errors {
			errs = append(errs, e.Message)
		}
		result.Error = strings.Join(errs, "; ")
	}

	// Run assertions
	for _, assertion := range tc.Assertions {
		ar := r.runAssertion(assertion, tc.Input)
		result.Assertions = append(result.Assertions, ar)
		if !ar.Passed {
			result.Status = "failed"
		}
	}

	result.DurationMs = int(time.Since(start).Milliseconds())
	return result
}

func (r *TestRunner) runAssertion(a Assertion, payload json.RawMessage) AssertionResult {
	result := AssertionResult{
		Type:     a.Type,
		Path:     a.Path,
		Expected: fmt.Sprintf("%v", a.Expected),
	}

	// Simple path extraction
	var data map[string]interface{}
	json.Unmarshal(payload, &data)

	path := strings.TrimPrefix(a.Path, "$.")
	value, exists := data[path]

	switch a.Type {
	case "exists":
		result.Passed = exists
		if !exists {
			result.Message = fmt.Sprintf("field '%s' does not exist", path)
		}
	case "equals":
		result.Actual = fmt.Sprintf("%v", value)
		result.Passed = fmt.Sprintf("%v", value) == fmt.Sprintf("%v", a.Expected)
		if !result.Passed {
			result.Message = fmt.Sprintf("expected '%v', got '%v'", a.Expected, value)
		}
	case "type_is":
		actualType := r.validator.getJSONType(value)
		result.Actual = actualType
		result.Passed = actualType == a.Expected.(string)
		if !result.Passed {
			result.Message = fmt.Sprintf("expected type '%v', got '%v'", a.Expected, actualType)
		}
	default:
		result.Passed = false
		result.Message = fmt.Sprintf("unknown assertion type: %s", a.Type)
	}

	return result
}
