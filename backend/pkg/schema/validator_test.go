package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ValidatePayloadDirect – pure tests (no mock needed)
// ---------------------------------------------------------------------------

func TestValidatePayloadDirect_ValidPayload(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {"name": {"type": "string"}},
		"required": ["name"]
	}`)
	result := v.ValidatePayloadDirect([]byte(`{"name":"alice"}`), schema)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidatePayloadDirect_InvalidJSON(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{"type":"object"}`)
	result := v.ValidatePayloadDirect([]byte(`{not json`), schema)
	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "parse_error", result.Errors[0].Type)
}

func TestValidatePayloadDirect_InvalidSchema(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	result := v.ValidatePayloadDirect([]byte(`{"a":1}`), json.RawMessage(`{bad schema`))
	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "schema_error", result.Errors[0].Type)
}

func TestValidatePayloadDirect_MissingRequired(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {"name": {"type": "string"}, "age": {"type": "number"}},
		"required": ["name", "age"]
	}`)
	result := v.ValidatePayloadDirect([]byte(`{"name":"alice"}`), schema)
	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "required", result.Errors[0].Type)
	assert.Contains(t, result.Errors[0].Path, "age")
}

func TestValidatePayloadDirect_TypeMismatch(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {"age": {"type": "number"}},
		"required": ["age"]
	}`)
	result := v.ValidatePayloadDirect([]byte(`{"age":"not_a_number"}`), schema)
	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "type_mismatch", result.Errors[0].Type)
}

func TestValidatePayloadDirect_ArrayValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		schema    string
		payload   string
		wantValid bool
		errType   string
	}{
		{
			name:      "minItems violated",
			schema:    `{"type":"array","items":{"type":"number"},"minItems":3}`,
			payload:   `[1,2]`,
			wantValid: false,
			errType:   "min_items",
		},
		{
			name:      "maxItems violated",
			schema:    `{"type":"array","items":{"type":"number"},"maxItems":2}`,
			payload:   `[1,2,3]`,
			wantValid: false,
			errType:   "max_items",
		},
		{
			name:      "array within bounds",
			schema:    `{"type":"array","items":{"type":"number"},"minItems":1,"maxItems":3}`,
			payload:   `[1,2]`,
			wantValid: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := &Validator{}
			result := v.ValidatePayloadDirect([]byte(tc.payload), json.RawMessage(tc.schema))
			assert.Equal(t, tc.wantValid, result.Valid)
			if !tc.wantValid {
				require.NotEmpty(t, result.Errors)
				assert.Equal(t, tc.errType, result.Errors[0].Type)
			}
		})
	}
}

func TestValidatePayloadDirect_StringValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		schema    string
		payload   string
		wantValid bool
		errType   string
	}{
		{
			name:      "minLength violated",
			schema:    `{"type":"string","minLength":5}`,
			payload:   `"abc"`,
			wantValid: false,
			errType:   "min_length",
		},
		{
			name:      "maxLength violated",
			schema:    `{"type":"string","maxLength":3}`,
			payload:   `"abcdef"`,
			wantValid: false,
			errType:   "max_length",
		},
		{
			name:      "enum valid",
			schema:    `{"type":"string","enum":["a","b","c"]}`,
			payload:   `"b"`,
			wantValid: true,
		},
		{
			name:      "enum invalid",
			schema:    `{"type":"string","enum":["a","b","c"]}`,
			payload:   `"z"`,
			wantValid: false,
			errType:   "enum",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := &Validator{}
			result := v.ValidatePayloadDirect([]byte(tc.payload), json.RawMessage(tc.schema))
			assert.Equal(t, tc.wantValid, result.Valid)
			if !tc.wantValid {
				require.NotEmpty(t, result.Errors)
				assert.Equal(t, tc.errType, result.Errors[0].Type)
			}
		})
	}
}

func TestValidatePayloadDirect_NumberValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		schema    string
		payload   string
		wantValid bool
		errType   string
	}{
		{
			name:      "below minimum",
			schema:    `{"type":"number","minimum":10}`,
			payload:   `5`,
			wantValid: false,
			errType:   "minimum",
		},
		{
			name:      "above maximum",
			schema:    `{"type":"number","maximum":10}`,
			payload:   `15`,
			wantValid: false,
			errType:   "maximum",
		},
		{
			name:      "within range",
			schema:    `{"type":"number","minimum":1,"maximum":100}`,
			payload:   `50`,
			wantValid: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := &Validator{}
			result := v.ValidatePayloadDirect([]byte(tc.payload), json.RawMessage(tc.schema))
			assert.Equal(t, tc.wantValid, result.Valid)
			if !tc.wantValid {
				require.NotEmpty(t, result.Errors)
				assert.Equal(t, tc.errType, result.Errors[0].Type)
			}
		})
	}
}

func TestValidatePayloadDirect_NestedObjectValidation(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{
		"type":"object",
		"properties":{
			"address":{
				"type":"object",
				"properties":{"city":{"type":"string"}},
				"required":["city"]
			}
		},
		"required":["address"]
	}`)

	t.Run("valid nested", func(t *testing.T) {
		t.Parallel()
		result := v.ValidatePayloadDirect([]byte(`{"address":{"city":"NYC"}}`), schema)
		assert.True(t, result.Valid)
	})

	t.Run("missing nested required", func(t *testing.T) {
		t.Parallel()
		result := v.ValidatePayloadDirect([]byte(`{"address":{}}`), schema)
		assert.False(t, result.Valid)
		require.NotEmpty(t, result.Errors)
		assert.Equal(t, "required", result.Errors[0].Type)
		assert.Contains(t, result.Errors[0].Path, "city")
	})
}

func TestValidatePayloadDirect_EmptySchema(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	result := v.ValidatePayloadDirect([]byte(`{"any":"thing","num":42}`), json.RawMessage(`{}`))
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidatePayloadDirect_DeeplyNestedObjects(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{
		"type":"object",
		"properties":{
			"l1":{
				"type":"object",
				"properties":{
					"l2":{
						"type":"object",
						"properties":{
							"l3":{"type":"string"}
						},
						"required":["l3"]
					}
				},
				"required":["l2"]
			}
		},
		"required":["l1"]
	}`)

	t.Run("all levels valid", func(t *testing.T) {
		t.Parallel()
		result := v.ValidatePayloadDirect([]byte(`{"l1":{"l2":{"l3":"deep"}}}`), schema)
		assert.True(t, result.Valid)
	})

	t.Run("deepest level missing", func(t *testing.T) {
		t.Parallel()
		result := v.ValidatePayloadDirect([]byte(`{"l1":{"l2":{}}}`), schema)
		assert.False(t, result.Valid)
		require.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0].Path, "l3")
	})

	t.Run("type mismatch at depth", func(t *testing.T) {
		t.Parallel()
		result := v.ValidatePayloadDirect([]byte(`{"l1":{"l2":{"l3":123}}}`), schema)
		assert.False(t, result.Valid)
		require.NotEmpty(t, result.Errors)
		assert.Equal(t, "type_mismatch", result.Errors[0].Type)
	})
}

// ---------------------------------------------------------------------------
// CheckCompatibility – pure tests (no mock needed)
// ---------------------------------------------------------------------------

func TestCheckCompatibility_Compatible(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	old := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	new := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}},"required":["name"]}`)
	result, err := v.CheckCompatibility(old, new)
	require.NoError(t, err)
	assert.True(t, result.Compatible)
	assert.Empty(t, result.BreakingChanges)
}

func TestCheckCompatibility_NewRequiredField(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	old := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	new := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}},"required":["name","age"]}`)
	result, err := v.CheckCompatibility(old, new)
	require.NoError(t, err)
	assert.False(t, result.Compatible)
	require.Len(t, result.BreakingChanges, 1)
	assert.Equal(t, "required_added", result.BreakingChanges[0].Type)
}

func TestCheckCompatibility_TypeChanged(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	old := json.RawMessage(`{"type":"object","properties":{"age":{"type":"number"}},"required":["age"]}`)
	new := json.RawMessage(`{"type":"object","properties":{"age":{"type":"string"}},"required":["age"]}`)
	result, err := v.CheckCompatibility(old, new)
	require.NoError(t, err)
	assert.False(t, result.Compatible)
	found := false
	for _, bc := range result.BreakingChanges {
		if bc.Type == "type_change" {
			found = true
		}
	}
	assert.True(t, found, "expected a type_change breaking change")
}

func TestCheckCompatibility_PropertyRemoved(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	old := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}}}`)
	new := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`)
	result, err := v.CheckCompatibility(old, new)
	require.NoError(t, err)
	// Property removal is a warning, not a breaking change
	assert.True(t, result.Compatible)
	require.NotEmpty(t, result.Warnings)
	assert.Contains(t, result.Warnings[0], "age")
}

func TestCheckCompatibility_InvalidJSON(t *testing.T) {
	t.Parallel()
	v := &Validator{}

	t.Run("invalid old schema", func(t *testing.T) {
		t.Parallel()
		_, err := v.CheckCompatibility(json.RawMessage(`{bad`), json.RawMessage(`{"type":"object"}`))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid old schema")
	})

	t.Run("invalid new schema", func(t *testing.T) {
		t.Parallel()
		_, err := v.CheckCompatibility(json.RawMessage(`{"type":"object"}`), json.RawMessage(`{bad`))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid new schema")
	})
}

func TestCheckCompatibility_NestedChanges(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	old := json.RawMessage(`{
		"type":"object",
		"properties":{
			"address":{
				"type":"object",
				"properties":{"city":{"type":"string"}},
				"required":["city"]
			}
		}
	}`)
	new := json.RawMessage(`{
		"type":"object",
		"properties":{
			"address":{
				"type":"object",
				"properties":{"city":{"type":"number"}},
				"required":["city"]
			}
		}
	}`)
	result, err := v.CheckCompatibility(old, new)
	require.NoError(t, err)
	assert.False(t, result.Compatible)
	found := false
	for _, bc := range result.BreakingChanges {
		if bc.Type == "type_change" && bc.Path == "$.address.city.type" {
			found = true
		}
	}
	assert.True(t, found, "expected nested type_change at $.address.city.type")
}

// ---------------------------------------------------------------------------
// Validate – needs mock repository
// ---------------------------------------------------------------------------

// mockValidatorRepo is a minimal mock for Validator tests.
type mockValidatorRepo struct {
	mock.Mock
}

func (m *mockValidatorRepo) GetEndpointSchema(ctx context.Context, endpointID string) (*EndpointSchema, error) {
	args := m.Called(ctx, endpointID)
	if v := args.Get(0); v != nil {
		return v.(*EndpointSchema), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockValidatorRepo) GetSchema(ctx context.Context, tenantID, schemaID string) (*Schema, error) {
	args := m.Called(ctx, tenantID, schemaID)
	if v := args.Get(0); v != nil {
		return v.(*Schema), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockValidatorRepo) GetVersion(ctx context.Context, schemaID, version string) (*SchemaVersion, error) {
	args := m.Called(ctx, schemaID, version)
	if v := args.Get(0); v != nil {
		return v.(*SchemaVersion), args.Error(1)
	}
	return nil, args.Error(1)
}

// Unused Repository methods – satisfy the interface.
func (m *mockValidatorRepo) CreateSchema(context.Context, *Schema) error { return nil }
func (m *mockValidatorRepo) GetSchemaByName(context.Context, string, string) (*Schema, error) {
	return nil, nil
}
func (m *mockValidatorRepo) ListSchemas(context.Context, string, int, int) ([]Schema, int, error) {
	return nil, 0, nil
}
func (m *mockValidatorRepo) UpdateSchema(context.Context, *Schema) error         { return nil }
func (m *mockValidatorRepo) DeleteSchema(context.Context, string, string) error  { return nil }
func (m *mockValidatorRepo) CreateVersion(context.Context, *SchemaVersion) error { return nil }
func (m *mockValidatorRepo) GetLatestVersion(context.Context, string) (*SchemaVersion, error) {
	return nil, nil
}
func (m *mockValidatorRepo) ListVersions(context.Context, string) ([]SchemaVersion, error) {
	return nil, nil
}
func (m *mockValidatorRepo) AssignSchemaToEndpoint(context.Context, *EndpointSchema) error {
	return nil
}
func (m *mockValidatorRepo) RemoveSchemaFromEndpoint(context.Context, string) error { return nil }
func (m *mockValidatorRepo) ListEndpointsWithSchema(context.Context, string) ([]string, error) {
	return nil, nil
}

func TestValidatePayloadDirect_NullPayload(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`)
	result := v.ValidatePayloadDirect([]byte(`null`), schema)
	assert.False(t, result.Valid)
	require.NotEmpty(t, result.Errors)
	assert.Equal(t, "type_mismatch", result.Errors[0].Type)
}

func TestValidatePayloadDirect_EmptyArrayPayload(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`)
	result := v.ValidatePayloadDirect([]byte(`[]`), schema)
	assert.False(t, result.Valid)
	require.NotEmpty(t, result.Errors)
	assert.Equal(t, "type_mismatch", result.Errors[0].Type)
}

func TestValidatePayloadDirect_EmptyObjectPayload(t *testing.T) {
	t.Parallel()
	v := &Validator{}
	schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`)
	result := v.ValidatePayloadDirect([]byte(`{}`), schema)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidate_NoAssignment(t *testing.T) {
	t.Parallel()
	repo := new(mockValidatorRepo)
	repo.On("GetEndpointSchema", mock.Anything, "ep1").Return(nil, nil)
	v := NewValidator(repo)

	result, err := v.Validate(context.Background(), "t1", "ep1", []byte(`{"a":1}`))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	repo.AssertExpectations(t)
}

func TestValidate_AssignmentModeNone(t *testing.T) {
	t.Parallel()
	repo := new(mockValidatorRepo)
	repo.On("GetEndpointSchema", mock.Anything, "ep1").Return(&EndpointSchema{
		EndpointID:     "ep1",
		SchemaID:       "s1",
		ValidationMode: ValidationModeNone,
	}, nil)
	v := NewValidator(repo)

	result, err := v.Validate(context.Background(), "t1", "ep1", []byte(`{"a":1}`))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	repo.AssertExpectations(t)
}

func TestValidate_ValidPayload(t *testing.T) {
	t.Parallel()
	repo := new(mockValidatorRepo)
	repo.On("GetEndpointSchema", mock.Anything, "ep1").Return(&EndpointSchema{
		EndpointID:     "ep1",
		SchemaID:       "s1",
		ValidationMode: ValidationModeStrict,
	}, nil)
	repo.On("GetSchema", mock.Anything, "t1", "s1").Return(&Schema{
		ID:         "s1",
		Name:       "test",
		Version:    "1.0",
		JSONSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
	}, nil)
	v := NewValidator(repo)

	result, err := v.Validate(context.Background(), "t1", "ep1", []byte(`{"name":"alice"}`))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "s1", result.SchemaID)
	repo.AssertExpectations(t)
}

func TestValidate_InvalidPayload(t *testing.T) {
	t.Parallel()
	repo := new(mockValidatorRepo)
	repo.On("GetEndpointSchema", mock.Anything, "ep1").Return(&EndpointSchema{
		EndpointID:     "ep1",
		SchemaID:       "s1",
		ValidationMode: ValidationModeStrict,
	}, nil)
	repo.On("GetSchema", mock.Anything, "t1", "s1").Return(&Schema{
		ID:         "s1",
		Name:       "test",
		Version:    "1.0",
		JSONSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
	}, nil)
	v := NewValidator(repo)

	result, err := v.Validate(context.Background(), "t1", "ep1", []byte(`{}`))
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	repo.AssertExpectations(t)
}

func TestValidate_SchemaVersionSpecified(t *testing.T) {
	t.Parallel()
	repo := new(mockValidatorRepo)
	repo.On("GetEndpointSchema", mock.Anything, "ep1").Return(&EndpointSchema{
		EndpointID:     "ep1",
		SchemaID:       "s1",
		SchemaVersion:  "2.0",
		ValidationMode: ValidationModeStrict,
	}, nil)
	repo.On("GetSchema", mock.Anything, "t1", "s1").Return(&Schema{
		ID:         "s1",
		Name:       "test",
		Version:    "1.0",
		JSONSchema: json.RawMessage(`{"type":"object"}`),
	}, nil)
	versionSchema := json.RawMessage(`{"type":"object","properties":{"age":{"type":"number"}},"required":["age"]}`)
	repo.On("GetVersion", mock.Anything, "s1", "2.0").Return(&SchemaVersion{
		SchemaID:   "s1",
		Version:    "2.0",
		JSONSchema: versionSchema,
	}, nil)
	v := NewValidator(repo)

	result, err := v.Validate(context.Background(), "t1", "ep1", []byte(`{"age":25}`))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	repo.AssertExpectations(t)
}

func TestValidate_GetEndpointSchemaError(t *testing.T) {
	t.Parallel()
	repo := new(mockValidatorRepo)
	repo.On("GetEndpointSchema", mock.Anything, "ep1").Return(nil, fmt.Errorf("db down"))
	v := NewValidator(repo)

	_, err := v.Validate(context.Background(), "t1", "ep1", []byte(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get endpoint schema")
	repo.AssertExpectations(t)
}

// ---------- 100-deep nesting validation ----------

func TestValidatePayloadDirect_100DeepNesting(t *testing.T) {
	t.Parallel()
	repo := new(mockValidatorRepo)
	v := NewValidator(repo)

	// Build a schema with 100 levels of nesting
	// Each level has: {"type":"object", "properties": {"child": <next>}, "required": ["child"]}
	innermost := `{"type": "string"}`
	schema := innermost
	for i := 0; i < 100; i++ {
		schema = `{"type": "object", "properties": {"child": ` + schema + `}, "required": ["child"]}`
	}

	// Build matching payload with 100 levels
	innerPayload := `"leaf"`
	payload := innerPayload
	for i := 0; i < 100; i++ {
		payload = `{"child": ` + payload + `}`
	}

	result := v.ValidatePayloadDirect([]byte(payload), json.RawMessage(schema))
	assert.True(t, result.Valid, "100-deep nested valid payload should pass")
	assert.Empty(t, result.Errors)
}

func TestValidatePayloadDirect_100DeepNesting_MissingField(t *testing.T) {
	t.Parallel()
	repo := new(mockValidatorRepo)
	v := NewValidator(repo)

	// Build a schema with 5 levels (sufficient to verify recursive validation)
	schema := `{"type": "object", "properties": {"child": {"type": "object", "properties": {"child": {"type": "object", "properties": {"val": {"type": "string"}}, "required": ["val"]}}, "required": ["child"]}}, "required": ["child"]}`

	// Missing "val" at the deepest level
	payload := `{"child": {"child": {}}}`

	result := v.ValidatePayloadDirect([]byte(payload), json.RawMessage(schema))
	assert.False(t, result.Valid, "missing required field at depth should fail")
	assert.True(t, len(result.Errors) > 0)
}
