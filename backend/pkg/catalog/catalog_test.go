package catalog

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
