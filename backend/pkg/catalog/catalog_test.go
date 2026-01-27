package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==========================================
// Mock Repository
// ==========================================

type mockRepository struct {
	eventTypes       map[uuid.UUID]*EventType
	eventVersions    map[uuid.UUID][]*EventVersion
	validationConfig map[uuid.UUID]*SchemaValidationConfig
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		eventTypes:       make(map[uuid.UUID]*EventType),
		eventVersions:    make(map[uuid.UUID][]*EventVersion),
		validationConfig: make(map[uuid.UUID]*SchemaValidationConfig),
	}
}

func (m *mockRepository) CreateEventType(ctx context.Context, et *EventType) error {
	if et.ID == uuid.Nil {
		et.ID = uuid.New()
	}
	et.CreatedAt = time.Now()
	et.UpdatedAt = time.Now()
	m.eventTypes[et.ID] = et
	return nil
}

func (m *mockRepository) GetEventType(ctx context.Context, id uuid.UUID) (*EventType, error) {
	et, ok := m.eventTypes[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return et, nil
}

func (m *mockRepository) GetEventTypeBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*EventType, error) {
	for _, et := range m.eventTypes {
		if et.TenantID == tenantID && et.Slug == slug {
			return et, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockRepository) UpdateEventType(ctx context.Context, et *EventType) error {
	m.eventTypes[et.ID] = et
	return nil
}

func (m *mockRepository) DeleteEventType(ctx context.Context, id uuid.UUID) error {
	delete(m.eventTypes, id)
	return nil
}

func (m *mockRepository) SearchEventTypes(ctx context.Context, params *CatalogSearchParams) (*CatalogSearchResult, error) {
	var results []*EventType
	for _, et := range m.eventTypes {
		if et.TenantID == params.TenantID {
			results = append(results, et)
		}
	}
	return &CatalogSearchResult{EventTypes: results, Total: len(results)}, nil
}

func (m *mockRepository) CreateEventVersion(ctx context.Context, ev *EventVersion) error {
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	ev.PublishedAt = time.Now()
	m.eventVersions[ev.EventTypeID] = append(m.eventVersions[ev.EventTypeID], ev)
	return nil
}

func (m *mockRepository) ListEventVersions(ctx context.Context, eventTypeID uuid.UUID) ([]*EventVersion, error) {
	return m.eventVersions[eventTypeID], nil
}

func (m *mockRepository) CreateCategory(ctx context.Context, cat *EventCategory) error { return nil }
func (m *mockRepository) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*EventCategory, error) {
	return nil, nil
}
func (m *mockRepository) CreateSubscription(ctx context.Context, sub *EventSubscription) error {
	return nil
}
func (m *mockRepository) DeleteSubscription(ctx context.Context, endpointID, eventTypeID uuid.UUID) error {
	return nil
}
func (m *mockRepository) ListEndpointSubscriptions(ctx context.Context, endpointID uuid.UUID) ([]*EventSubscription, error) {
	return nil, nil
}
func (m *mockRepository) ListEventTypeSubscriptions(ctx context.Context, eventTypeID uuid.UUID) ([]*EventSubscription, error) {
	return nil, nil
}
func (m *mockRepository) GetSubscriberCount(ctx context.Context, eventTypeID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockRepository) SaveDocumentation(ctx context.Context, doc *EventDocumentation) error {
	return nil
}
func (m *mockRepository) GetDocumentation(ctx context.Context, eventTypeID uuid.UUID) ([]*EventDocumentation, error) {
	return nil, nil
}

func (m *mockRepository) GetSchemaValidationConfig(ctx context.Context, tenantID uuid.UUID) (*SchemaValidationConfig, error) {
	config, ok := m.validationConfig[tenantID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return config, nil
}

func (m *mockRepository) SaveSchemaValidationConfig(ctx context.Context, config *SchemaValidationConfig) error {
	m.validationConfig[config.TenantID] = config
	return nil
}

// ==========================================
// Schema Compatibility Tests
// ==========================================

func TestBackwardCompatibility_NoChanges(t *testing.T) {
	checker := NewCompatibilityChecker()
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"}
		},
		"required": ["id"]
	}`)

	compatible, issues := checker.CheckBackwardCompatibility(schema, schema)
	assert.True(t, compatible)
	assert.Empty(t, issues)
}

func TestBackwardCompatibility_AddOptionalField(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"required": ["id"]
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"email": {"type": "string"}
		},
		"required": ["id"]
	}`)

	compatible, issues := checker.CheckBackwardCompatibility(oldSchema, newSchema)
	assert.True(t, compatible)
	assert.Empty(t, issues)
}

func TestBackwardCompatibility_RemoveField(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"}
		},
		"required": ["id"]
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"required": ["id"]
	}`)

	compatible, issues := checker.CheckBackwardCompatibility(oldSchema, newSchema)
	assert.False(t, compatible)
	assert.Contains(t, issues[0], "name")
	assert.Contains(t, issues[0], "removed")
}

func TestBackwardCompatibility_AddNewRequiredField(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"required": ["id"]
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"email": {"type": "string"}
		},
		"required": ["id", "email"]
	}`)

	compatible, issues := checker.CheckBackwardCompatibility(oldSchema, newSchema)
	assert.False(t, compatible)
	assert.Contains(t, issues[0], "email")
	assert.Contains(t, issues[0], "required")
}

func TestBackwardCompatibility_ChangeFieldType(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"count": {"type": "integer"}
		}
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"count": {"type": "string"}
		}
	}`)

	compatible, issues := checker.CheckBackwardCompatibility(oldSchema, newSchema)
	assert.False(t, compatible)
	assert.Contains(t, issues[0], "count")
	assert.Contains(t, issues[0], "type changed")
}

func TestForwardCompatibility_NoChanges(t *testing.T) {
	checker := NewCompatibilityChecker()
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"}
		},
		"required": ["id"]
	}`)

	compatible, issues := checker.CheckForwardCompatibility(schema, schema)
	assert.True(t, compatible)
	assert.Empty(t, issues)
}

func TestForwardCompatibility_TypeChange(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"age": {"type": "integer"}
		}
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"age": {"type": "string"}
		}
	}`)

	compatible, issues := checker.CheckForwardCompatibility(oldSchema, newSchema)
	assert.False(t, compatible)
	assert.Contains(t, issues[0], "age")
}

func TestForwardCompatibility_RemovedRequiredField(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"}
		},
		"required": ["id", "name"]
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"required": ["id"]
	}`)

	compatible, issues := checker.CheckForwardCompatibility(oldSchema, newSchema)
	assert.False(t, compatible)
	assert.NotEmpty(t, issues)
}

func TestFullCompatibility_Compatible(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"}
		},
		"required": ["id"]
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"},
			"email": {"type": "string"}
		},
		"required": ["id"]
	}`)

	compatible, issues := checker.CheckFullCompatibility(oldSchema, newSchema)
	assert.True(t, compatible)
	assert.Empty(t, issues)
}

func TestFullCompatibility_Incompatible(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"name": {"type": "string"}
		},
		"required": ["id"]
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "integer"}
		},
		"required": ["id"]
	}`)

	compatible, issues := checker.CheckFullCompatibility(oldSchema, newSchema)
	assert.False(t, compatible)
	assert.NotEmpty(t, issues)
}

func TestCompatibility_InvalidJSON(t *testing.T) {
	checker := NewCompatibilityChecker()
	valid := json.RawMessage(`{"type": "object", "properties": {"id": {"type": "string"}}}`)
	invalid := json.RawMessage(`{invalid json`)

	compatible, issues := checker.CheckBackwardCompatibility(valid, invalid)
	assert.False(t, compatible)
	assert.NotEmpty(t, issues)

	compatible, issues = checker.CheckBackwardCompatibility(invalid, valid)
	assert.False(t, compatible)
	assert.NotEmpty(t, issues)
}

// ==========================================
// Code Generation Tests
// ==========================================

func TestGenerateGoTypes(t *testing.T) {
	et := &EventType{
		Name: "order.created",
		Schema: &EventSchema{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"order_id": {"type": "string"},
					"amount": {"type": "number"},
					"items": {"type": "array", "items": {"type": "string"}}
				},
				"required": ["order_id", "amount"]
			}`),
		},
	}

	code, err := GenerateGoTypes(et)
	require.NoError(t, err)
	assert.Contains(t, code, "package events")
	assert.Contains(t, code, "type OrderCreated struct")
	assert.Contains(t, code, "string")
	assert.Contains(t, code, "float64")
	assert.Contains(t, code, "json:\"order_id\"")
}

func TestGeneratePythonTypes(t *testing.T) {
	et := &EventType{
		Name: "order.created",
		Schema: &EventSchema{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"order_id": {"type": "string"},
					"amount": {"type": "number"},
					"active": {"type": "boolean"}
				},
				"required": ["order_id"]
			}`),
		},
	}

	code, err := GeneratePythonTypes(et)
	require.NoError(t, err)
	assert.Contains(t, code, "@dataclass")
	assert.Contains(t, code, "class OrderCreated")
	assert.Contains(t, code, "order_id: str")
	assert.Contains(t, code, "Optional[")
}

func TestGenerateTypeScriptTypes(t *testing.T) {
	et := &EventType{
		Name: "user.updated",
		Schema: &EventSchema{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"user_id": {"type": "string"},
					"email": {"type": "string"},
					"age": {"type": "integer"}
				},
				"required": ["user_id"]
			}`),
		},
	}

	code, err := GenerateTypeScriptTypes(et)
	require.NoError(t, err)
	assert.Contains(t, code, "export interface UserUpdated")
	assert.Contains(t, code, "user_id: string")
	assert.Contains(t, code, "number")
}

func TestGenerateGoTypes_FromExamplePayload(t *testing.T) {
	et := &EventType{
		Name: "payment.completed",
		ExamplePayload: json.RawMessage(`{
			"payment_id": "pay_123",
			"amount": 99.99,
			"success": true
		}`),
	}

	code, err := GenerateGoTypes(et)
	require.NoError(t, err)
	assert.Contains(t, code, "type PaymentCompleted struct")
	assert.Contains(t, code, "string")
}

func TestGenerateTypes_NoSchema(t *testing.T) {
	et := &EventType{
		Name: "empty.event",
	}

	_, err := GenerateGoTypes(et)
	assert.Error(t, err)
}

func TestGenerateTypeScriptTypes_AllTypes(t *testing.T) {
	et := &EventType{
		Name: "test.event",
		Schema: &EventSchema{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"count": {"type": "integer"},
					"price": {"type": "number"},
					"active": {"type": "boolean"},
					"tags": {"type": "array", "items": {"type": "string"}},
					"metadata": {"type": "object"}
				},
				"required": ["name"]
			}`),
		},
	}

	code, err := GenerateTypeScriptTypes(et)
	require.NoError(t, err)
	assert.Contains(t, code, "string")
	assert.Contains(t, code, "number")
	assert.Contains(t, code, "boolean")
	assert.Contains(t, code, "string[]")
	assert.Contains(t, code, "Record<string, unknown>")
}

// ==========================================
// Payload Validation Tests
// ==========================================

func TestValidateAgainstSchema_Valid(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":   map[string]interface{}{"type": "string"},
			"name": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"id"},
	}

	data := map[string]interface{}{
		"id":   "123",
		"name": "test",
	}

	valid, issues := validateAgainstSchema(data, schema)
	assert.True(t, valid)
	assert.Empty(t, issues)
}

func TestValidateAgainstSchema_MissingRequired(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":   map[string]interface{}{"type": "string"},
			"name": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"id", "name"},
	}

	data := map[string]interface{}{
		"id": "123",
	}

	valid, issues := validateAgainstSchema(data, schema)
	assert.False(t, valid)
	assert.Len(t, issues, 1)
	assert.Contains(t, issues[0], "name")
	assert.Contains(t, issues[0], "required")
}

func TestValidateAgainstSchema_WrongType(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":    map[string]interface{}{"type": "string"},
			"count": map[string]interface{}{"type": "string"},
		},
	}

	data := map[string]interface{}{
		"id":    "123",
		"count": float64(42),
	}

	valid, issues := validateAgainstSchema(data, schema)
	assert.False(t, valid)
	assert.Contains(t, issues[0], "count")
	assert.Contains(t, issues[0], "expected type")
}

func TestValidateAgainstSchema_NumberIntegerCompat(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{"type": "integer"},
		},
	}

	data := map[string]interface{}{
		"count": float64(42),
	}

	valid, issues := validateAgainstSchema(data, schema)
	assert.True(t, valid)
	assert.Empty(t, issues)
}

// ==========================================
// Search & Helper Tests
// ==========================================

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Order Created", "order.created"},
		{"user.updated", "user.updated"},
		{"Payment Completed!", "payment.completed"},
	}

	for _, tt := range tests {
		slug := generateSlug(tt.input)
		assert.Equal(t, tt.expected, slug)
	}
}

func TestIsValidSemver(t *testing.T) {
	assert.True(t, isValidSemver("1.0.0"))
	assert.True(t, isValidSemver("2.1.3"))
	assert.True(t, isValidSemver("1.0.0-beta.1"))
	assert.False(t, isValidSemver("1.0"))
	assert.False(t, isValidSemver("abc"))
	assert.False(t, isValidSemver(""))
}

// ==========================================
// Model Tests
// ==========================================

func TestEventTypeStatus(t *testing.T) {
	assert.Equal(t, "active", StatusActive)
	assert.Equal(t, "deprecated", StatusDeprecated)
	assert.Equal(t, "draft", StatusDraft)
}

func TestSDKLanguages(t *testing.T) {
	assert.Equal(t, "typescript", LangTypeScript)
	assert.Equal(t, "python", LangPython)
	assert.Equal(t, "go", LangGo)
}

// ==========================================
// Helper function tests
// ==========================================

func TestToGoName(t *testing.T) {
	assert.Equal(t, "OrderCreated", toGoName("order.created"))
	assert.Equal(t, "UserUpdated", toGoName("user_updated"))
	assert.Equal(t, "PaymentCompleted", toGoName("payment-completed"))
	assert.Equal(t, "Simple", toGoName("simple"))
}

func TestToSnakeCase(t *testing.T) {
	assert.Equal(t, "order_id", toSnakeCase("orderId"))
	assert.Equal(t, "user_name", toSnakeCase("userName"))
	assert.Equal(t, "simple", toSnakeCase("simple"))
}

func TestInferGoJSONType(t *testing.T) {
	assert.Equal(t, "string", inferGoJSONType("hello"))
	assert.Equal(t, "number", inferGoJSONType(float64(42)))
	assert.Equal(t, "boolean", inferGoJSONType(true))
	assert.Equal(t, "array", inferGoJSONType([]interface{}{}))
	assert.Equal(t, "object", inferGoJSONType(map[string]interface{}{}))
	assert.Equal(t, "null", inferGoJSONType(nil))
}

func TestIsTypeCompatible(t *testing.T) {
	assert.True(t, isTypeCompatible("string", "string"))
	assert.True(t, isTypeCompatible("integer", "number"))
	assert.True(t, isTypeCompatible("number", "integer"))
	assert.False(t, isTypeCompatible("string", "number"))
	assert.False(t, isTypeCompatible("boolean", "string"))
}

// ==========================================
// Additional Compatibility Tests
// ==========================================

func TestBackwardCompatibility_RemovedRequiredField(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"email": {"type": "string"}
		},
		"required": ["id", "email"]
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"required": ["id"]
	}`)

	compatible, issues := checker.CheckBackwardCompatibility(oldSchema, newSchema)
	assert.False(t, compatible)
	assert.Contains(t, issues[0], "email")
	assert.Contains(t, issues[0], "removed")
}

func TestBackwardCompatibility_TypeChangeStringToInteger(t *testing.T) {
	checker := NewCompatibilityChecker()
	oldSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"status": {"type": "string"}
		}
	}`)
	newSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string"},
			"status": {"type": "integer"}
		}
	}`)

	compatible, issues := checker.CheckBackwardCompatibility(oldSchema, newSchema)
	assert.False(t, compatible)
	assert.Contains(t, issues[0], "status")
	assert.Contains(t, issues[0], "type changed")
}

// ==========================================
// Additional Slug Tests
// ==========================================

func TestGenerateSlug_SpecialCharacters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World!", "hello.world"},
		{"foo--bar", "foobar"},
		{"Multiple   Spaces", "multiple...spaces"},
		{"UPPER CASE", "upper.case"},
		{"already.dotted", "already.dotted"},
	}
	for _, tt := range tests {
		slug := generateSlug(tt.input)
		assert.Equal(t, tt.expected, slug, "input: %s", tt.input)
	}
}

// ==========================================
// Additional Semver Tests
// ==========================================

func TestIsValidSemver_EdgeCases(t *testing.T) {
	assert.True(t, isValidSemver("0.0.1"))
	assert.True(t, isValidSemver("10.20.30"))
	assert.True(t, isValidSemver("1.0.0-alpha"))
	assert.True(t, isValidSemver("1.0.0-rc.1"))
	assert.False(t, isValidSemver("v1.0.0"))
	assert.False(t, isValidSemver("1.0.0.0"))
	assert.False(t, isValidSemver("1.0"))
	assert.False(t, isValidSemver("1"))
	assert.False(t, isValidSemver(".1.0"))
}

// ==========================================
// Additional TypeScript Codegen Tests
// ==========================================

func TestGenerateTypeScriptTypes_OptionalFields(t *testing.T) {
	et := &EventType{
		Name: "order.shipped",
		Schema: &EventSchema{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"order_id": {"type": "string"},
					"tracking_number": {"type": "string"},
					"shipped_at": {"type": "string"}
				},
				"required": ["order_id"]
			}`),
		},
	}

	code, err := GenerateTypeScriptTypes(et)
	require.NoError(t, err)
	assert.Contains(t, code, "export interface OrderShipped")
	assert.Contains(t, code, "order_id: string")
	// Optional fields should have ? marker
	assert.Contains(t, code, "?")
}

func TestGenerateTypeScriptTypes_NoSchema(t *testing.T) {
	et := &EventType{Name: "empty.event"}
	_, err := GenerateTypeScriptTypes(et)
	assert.Error(t, err)
}

// ==========================================
// Changelog Generation Tests (with mock)
// ==========================================

func TestGenerateChangelog(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	et := &EventType{
		TenantID: tenantID,
		Name:     "order.created",
		Slug:     "order.created",
		Version:  "1.0.0",
		Status:   StatusActive,
	}
	require.NoError(t, repo.CreateEventType(ctx, et))

	// Add versions
	require.NoError(t, repo.CreateEventVersion(ctx, &EventVersion{
		EventTypeID: et.ID, Version: "1.0.0", Changelog: "Initial version",
	}))
	require.NoError(t, repo.CreateEventVersion(ctx, &EventVersion{
		EventTypeID: et.ID, Version: "1.1.0", Changelog: "Added email field",
	}))
	require.NoError(t, repo.CreateEventVersion(ctx, &EventVersion{
		EventTypeID: et.ID, Version: "2.0.0", Changelog: "Breaking schema change", IsBreakingChange: true,
	}))

	changelog, err := svc.GenerateChangelog(ctx, et.ID)
	require.NoError(t, err)
	assert.Equal(t, et.ID, changelog.EventTypeID)
	assert.Equal(t, "order.created", changelog.EventName)
	assert.Len(t, changelog.Entries, 3)
	assert.True(t, changelog.Entries[2].Breaking)
	assert.Equal(t, "2.0.0", changelog.Entries[2].Version)
}

func TestGenerateChangelog_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	_, err := svc.GenerateChangelog(context.Background(), uuid.New())
	assert.Error(t, err)
}

// ==========================================
// ValidateForDelivery Tests (with mock)
// ==========================================

func TestValidateForDelivery_UnknownEventType(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	result := svc.ValidateForDelivery(ctx, uuid.New(), "unknown.event", json.RawMessage(`{}`))
	assert.True(t, result.Valid)
	assert.Equal(t, string(ValidationModeNone), result.Mode)
}

func TestValidateForDelivery_ModeNone(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	repo.validationConfig[tenantID] = &SchemaValidationConfig{
		TenantID: tenantID, Mode: ValidationModeNone,
	}

	et := &EventType{
		TenantID: tenantID, Name: "test.event", Slug: "test.event",
		Version: "1.0.0", Status: StatusActive,
	}
	require.NoError(t, repo.CreateEventType(ctx, et))

	result := svc.ValidateForDelivery(ctx, tenantID, "test.event", json.RawMessage(`{}`))
	assert.True(t, result.Valid)
	assert.Equal(t, string(ValidationModeNone), result.Mode)
}

func TestValidateForDelivery_ModeWarn(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	repo.validationConfig[tenantID] = &SchemaValidationConfig{
		TenantID: tenantID, Mode: ValidationModeWarn,
	}

	et := &EventType{
		TenantID: tenantID, Name: "test.event", Slug: "test.event",
		Version: "1.0.0", Status: StatusActive,
		Schema: &EventSchema{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {"id": {"type": "string"}},
				"required": ["id"]
			}`),
		},
	}
	require.NoError(t, repo.CreateEventType(ctx, et))

	// Missing required field — warn mode should still pass
	result := svc.ValidateForDelivery(ctx, tenantID, "test.event", json.RawMessage(`{}`))
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings)
}

func TestValidateForDelivery_ModeStrict(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	repo.validationConfig[tenantID] = &SchemaValidationConfig{
		TenantID: tenantID, Mode: ValidationModeStrict,
	}

	et := &EventType{
		TenantID: tenantID, Name: "test.event", Slug: "test.event",
		Version: "1.0.0", Status: StatusActive,
		Schema: &EventSchema{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {"id": {"type": "string"}},
				"required": ["id"]
			}`),
		},
	}
	require.NoError(t, repo.CreateEventType(ctx, et))

	// Missing required field — strict mode should fail
	result := svc.ValidateForDelivery(ctx, tenantID, "test.event", json.RawMessage(`{}`))
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Issues)
}

// ==========================================
// GetValidationConfig Tests (with mock)
// ==========================================

func TestGetValidationConfig_FallbackToDefaults(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	config := svc.GetValidationConfig(context.Background(), uuid.New())
	assert.Equal(t, ValidationModeWarn, config.Mode)
	assert.Equal(t, 1024*1024, config.MaxPayloadBytes)
}

func TestGetValidationConfig_FromRepo(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	saved := &SchemaValidationConfig{
		TenantID: tenantID, Mode: ValidationModeStrict,
		RejectUnknown: true, CoerceTypes: false, MaxPayloadBytes: 512,
	}
	repo.validationConfig[tenantID] = saved

	config := svc.GetValidationConfig(context.Background(), tenantID)
	assert.Equal(t, ValidationModeStrict, config.Mode)
	assert.True(t, config.RejectUnknown)
	assert.Equal(t, 512, config.MaxPayloadBytes)
}

// ==========================================
// CreateEventType Service Tests (with mock)
// ==========================================

func TestCreateEventType_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	req := &CreateEventTypeRequest{Name: "Order Created", Description: "New order"}
	et, err := svc.CreateEventType(context.Background(), tenantID, req)
	require.NoError(t, err)
	assert.Equal(t, "order.created", et.Slug)
	assert.Equal(t, "1.0.0", et.Version)
	assert.Equal(t, StatusActive, et.Status)
	// Version should have been created
	versions := repo.eventVersions[et.ID]
	assert.Len(t, versions, 1)
}
