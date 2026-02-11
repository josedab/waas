package contracts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Validator.ValidatePayload
// ---------------------------------------------------------------------------

func TestValidator_ValidatePayload(t *testing.T) {
	t.Parallel()
	v := NewValidator()

	baseSchema := json.RawMessage(`{
		"properties": {
			"name":  {"type": "string"},
			"age":   {"type": "number"},
			"email": {"type": "string"}
		},
		"required": ["name", "age"]
	}`)

	tests := []struct {
		name       string
		contract   *Contract
		payload    json.RawMessage
		headers    map[string]string
		wantValid  bool
		wantCode   string // expected error code (empty = no specific check)
		wantErrLen int    // -1 means don't check length
	}{
		{
			name: "valid payload with all required fields",
			contract: &Contract{
				ID: "c1", Version: "1.0",
				RequestSchema: baseSchema,
			},
			payload:    json.RawMessage(`{"name":"Alice","age":30}`),
			headers:    map[string]string{},
			wantValid:  true,
			wantErrLen: 0,
		},
		{
			name: "missing required header",
			contract: &Contract{
				ID: "c2", Version: "1.0",
				RequestSchema:   baseSchema,
				RequiredHeaders: []string{"X-Api-Key"},
			},
			payload:    json.RawMessage(`{"name":"Alice","age":30}`),
			headers:    map[string]string{},
			wantValid:  false,
			wantCode:   "MISSING_HEADER",
			wantErrLen: -1,
		},
		{
			name: "invalid schema JSON",
			contract: &Contract{
				ID: "c3", Version: "1.0",
				RequestSchema: json.RawMessage(`not-json`),
			},
			payload:    json.RawMessage(`{"name":"Alice"}`),
			headers:    map[string]string{},
			wantValid:  false,
			wantCode:   "INVALID_SCHEMA",
			wantErrLen: -1,
		},
		{
			name: "invalid payload JSON",
			contract: &Contract{
				ID: "c4", Version: "1.0",
				RequestSchema: baseSchema,
			},
			payload:    json.RawMessage(`not-json`),
			headers:    map[string]string{},
			wantValid:  false,
			wantCode:   "INVALID_JSON",
			wantErrLen: -1,
		},
		{
			name: "missing required field",
			contract: &Contract{
				ID: "c5", Version: "1.0",
				RequestSchema: baseSchema,
			},
			payload:    json.RawMessage(`{"name":"Alice"}`),
			headers:    map[string]string{},
			wantValid:  false,
			wantCode:   "MISSING_FIELD",
			wantErrLen: 1,
		},
		{
			name: "type mismatch",
			contract: &Contract{
				ID: "c6", Version: "1.0",
				RequestSchema: baseSchema,
			},
			payload:    json.RawMessage(`{"name":123,"age":30}`),
			headers:    map[string]string{},
			wantValid:  false,
			wantCode:   "TYPE_MISMATCH",
			wantErrLen: -1,
		},
		{
			name: "integer field matches number type",
			contract: &Contract{
				ID: "c7", Version: "1.0",
				RequestSchema: json.RawMessage(`{
					"properties": {"count": {"type": "integer"}},
					"required": ["count"]
				}`),
			},
			payload:    json.RawMessage(`{"count":42}`),
			headers:    map[string]string{},
			wantValid:  true,
			wantErrLen: 0,
		},
		{
			name: "nil headers with no required headers",
			contract: &Contract{
				ID: "c8", Version: "1.0",
				RequestSchema: baseSchema,
			},
			payload:    json.RawMessage(`{"name":"Bob","age":25}`),
			headers:    nil,
			wantValid:  true,
			wantErrLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := v.ValidatePayload(tt.contract, tt.payload, tt.headers)

			assert.Equal(t, tt.wantValid, result.IsValid, "IsValid mismatch")

			if tt.wantErrLen >= 0 {
				assert.Len(t, result.Errors, tt.wantErrLen)
			}
			if tt.wantCode != "" {
				require.NotEmpty(t, result.Errors, "expected at least one error")
				codes := make([]string, len(result.Errors))
				for i, e := range result.Errors {
					codes[i] = e.Code
				}
				assert.Contains(t, codes, tt.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BreakingChangeDetector.DetectChanges
// ---------------------------------------------------------------------------

func TestBreakingChangeDetector_DetectChanges(t *testing.T) {
	t.Parallel()
	d := NewBreakingChangeDetector()

	tests := []struct {
		name            string
		oldSchema       json.RawMessage
		newSchema       json.RawMessage
		wantErr         bool
		wantBreaking    bool
		wantChangeType  string // first matching change type
		wantChangeCount int    // -1 means don't check
	}{
		{
			name: "field removed is breaking",
			oldSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "age": {"type":"number"}},
				"required": ["name"]
			}`),
			newSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
			wantBreaking:    true,
			wantChangeType:  "field_removed",
			wantChangeCount: -1,
		},
		{
			name: "new required field added is breaking",
			oldSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
			newSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "email": {"type":"string"}},
				"required": ["name", "email"]
			}`),
			wantBreaking:    true,
			wantChangeType:  "required_added",
			wantChangeCount: -1,
		},
		{
			name: "type changed is breaking",
			oldSchema: json.RawMessage(`{
				"properties": {"age": {"type":"number"}},
				"required": ["age"]
			}`),
			newSchema: json.RawMessage(`{
				"properties": {"age": {"type":"string"}},
				"required": ["age"]
			}`),
			wantBreaking:    true,
			wantChangeType:  "type_changed",
			wantChangeCount: -1,
		},
		{
			name: "optional field added is non-breaking",
			oldSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
			newSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "nickname": {"type":"string"}},
				"required": ["name"]
			}`),
			wantBreaking:    false,
			wantChangeType:  "field_added",
			wantChangeCount: -1,
		},
		{
			name: "no changes",
			oldSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
			newSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
			wantBreaking:    false,
			wantChangeCount: 0,
		},
		{
			name:      "invalid old schema JSON",
			oldSchema: json.RawMessage(`not-json`),
			newSchema: json.RawMessage(`{"properties":{}}`),
			wantErr:   true,
		},
		{
			name:      "invalid new schema JSON",
			oldSchema: json.RawMessage(`{"properties":{}}`),
			newSchema: json.RawMessage(`not-json`),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := d.DetectChanges(tt.oldSchema, tt.newSchema)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantBreaking, result.HasBreakingChanges)

			if tt.wantChangeCount >= 0 {
				assert.Len(t, result.Changes, tt.wantChangeCount)
			}

			if tt.wantChangeType != "" {
				types := make([]string, len(result.Changes))
				for i, c := range result.Changes {
					types[i] = c.Type
				}
				assert.Contains(t, types, tt.wantChangeType)
			}
		})
	}
}

func TestBreakingChangeDetector_MigrationStepsGenerated(t *testing.T) {
	t.Parallel()
	d := NewBreakingChangeDetector()

	oldSchema := json.RawMessage(`{
		"properties": {"name": {"type":"string"}, "age": {"type":"number"}},
		"required": ["name"]
	}`)
	newSchema := json.RawMessage(`{
		"properties": {"name": {"type":"string"}},
		"required": ["name"]
	}`)

	result, err := d.DetectChanges(oldSchema, newSchema)
	require.NoError(t, err)
	assert.True(t, result.HasBreakingChanges)
	assert.NotEmpty(t, result.MigrationSteps, "migration steps should be generated for breaking changes")
}

func TestBreakingChangeDetector_ImpactAssessment(t *testing.T) {
	t.Parallel()
	d := NewBreakingChangeDetector()

	t.Run("minor impact with few breaking changes", func(t *testing.T) {
		t.Parallel()
		old := json.RawMessage(`{
			"properties": {"a": {"type":"string"}, "b": {"type":"number"}},
			"required": ["a"]
		}`)
		new := json.RawMessage(`{
			"properties": {"a": {"type":"string"}},
			"required": ["a"]
		}`)
		result, err := d.DetectChanges(old, new)
		require.NoError(t, err)
		assert.Contains(t, result.ImpactAssessment, "Minor")
	})

	t.Run("significant impact with many breaking changes", func(t *testing.T) {
		t.Parallel()
		old := json.RawMessage(`{
			"properties": {"a": {"type":"string"}, "b": {"type":"number"}, "c": {"type":"boolean"}},
			"required": ["a"]
		}`)
		new := json.RawMessage(`{
			"properties": {},
			"required": ["a"]
		}`)
		result, err := d.DetectChanges(old, new)
		require.NoError(t, err)
		assert.Contains(t, result.ImpactAssessment, "Significant")
	})
}

// ---------------------------------------------------------------------------
// TestRunner.RunSuite
// ---------------------------------------------------------------------------

func TestTestRunner_RunSuite(t *testing.T) {
	t.Parallel()
	runner := NewTestRunner()
	ctx := context.Background()

	contract := &Contract{
		ID: "c1", Version: "1.0",
		RequestSchema: json.RawMessage(`{
			"properties": {
				"name":  {"type":"string"},
				"value": {"type":"number"}
			},
			"required": ["name","value"]
		}`),
	}

	tests := []struct {
		name       string
		suite      *TestSuite
		wantStatus string
		wantPassed int
		wantFailed int
		wantSkip   int
	}{
		{
			name: "all tests pass",
			suite: &TestSuite{
				ID: "s1", Name: "pass suite",
				TestCases: []TestCase{
					{
						Name:  "valid input",
						Input: json.RawMessage(`{"name":"Alice","value":42}`),
						Assertions: []Assertion{
							{Type: "exists", Path: "$.name"},
							{Type: "equals", Path: "$.name", Expected: "Alice"},
							{Type: "type_is", Path: "$.value", Expected: "number"},
						},
					},
				},
			},
			wantStatus: "passed",
			wantPassed: 1,
			wantFailed: 0,
			wantSkip:   0,
		},
		{
			name: "skipped test counted",
			suite: &TestSuite{
				ID: "s2", Name: "skip suite",
				TestCases: []TestCase{
					{
						Name:  "valid",
						Input: json.RawMessage(`{"name":"Bob","value":1}`),
					},
					{
						Name:  "skipped test",
						Input: json.RawMessage(`{}`),
						Skip:  true,
					},
				},
			},
			wantStatus: "passed",
			wantPassed: 1,
			wantFailed: 0,
			wantSkip:   1,
		},
		{
			name: "failed assertion",
			suite: &TestSuite{
				ID: "s3", Name: "fail suite",
				TestCases: []TestCase{
					{
						Name:  "wrong value",
						Input: json.RawMessage(`{"name":"Alice","value":42}`),
						Assertions: []Assertion{
							{Type: "equals", Path: "$.name", Expected: "Bob"},
						},
					},
				},
			},
			wantStatus: "failed",
			wantPassed: 0,
			wantFailed: 1,
			wantSkip:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := runner.RunSuite(ctx, tt.suite, contract)

			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantPassed, result.Passed)
			assert.Equal(t, tt.wantFailed, result.Failed)
			assert.Equal(t, tt.wantSkip, result.Skipped)
			assert.Equal(t, len(tt.suite.TestCases), result.TotalTests)
			assert.NotNil(t, result.CompletedAt)
		})
	}
}

// ---------------------------------------------------------------------------
// Validator: deeply nested schema
// ---------------------------------------------------------------------------

func TestValidator_DeeplyNestedSchema(t *testing.T) {
	t.Parallel()
	v := NewValidator()

	nestedSchema := json.RawMessage(`{
		"properties": {
			"user":    {"type": "object"},
			"address": {"type": "object"}
		},
		"required": ["user"]
	}`)
	contract := &Contract{
		ID: "nested1", Version: "1.0",
		RequestSchema: nestedSchema,
	}

	t.Run("valid nested payload", func(t *testing.T) {
		t.Parallel()
		payload := json.RawMessage(`{
			"user": {"name": "Alice", "age": 30},
			"address": {"city": "NYC", "zip": "10001"}
		}`)
		result := v.ValidatePayload(contract, payload, nil)
		assert.True(t, result.IsValid)
		assert.Empty(t, result.Errors)
	})

	t.Run("missing required nested field", func(t *testing.T) {
		t.Parallel()
		payload := json.RawMessage(`{"address": {"city": "NYC"}}`)
		result := v.ValidatePayload(contract, payload, nil)
		assert.False(t, result.IsValid)
		codes := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			codes[i] = e.Code
		}
		assert.Contains(t, codes, "MISSING_FIELD")
	})

	t.Run("nested field with wrong type", func(t *testing.T) {
		t.Parallel()
		payload := json.RawMessage(`{"user": "not-an-object"}`)
		result := v.ValidatePayload(contract, payload, nil)
		assert.False(t, result.IsValid)
		codes := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			codes[i] = e.Code
		}
		assert.Contains(t, codes, "TYPE_MISMATCH")
	})
}

// ---------------------------------------------------------------------------
// Validator: empty schema
// ---------------------------------------------------------------------------

func TestValidator_EmptySchema(t *testing.T) {
	t.Parallel()
	v := NewValidator()

	contract := &Contract{
		ID: "empty1", Version: "1.0",
		RequestSchema: json.RawMessage(`{}`),
	}

	t.Run("any payload passes empty schema", func(t *testing.T) {
		t.Parallel()
		payload := json.RawMessage(`{"anything": "goes", "number": 42}`)
		result := v.ValidatePayload(contract, payload, nil)
		assert.True(t, result.IsValid)
		assert.Empty(t, result.Errors)
	})

	t.Run("empty payload passes empty schema", func(t *testing.T) {
		t.Parallel()
		payload := json.RawMessage(`{}`)
		result := v.ValidatePayload(contract, payload, nil)
		assert.True(t, result.IsValid)
		assert.Empty(t, result.Errors)
	})
}

// ---------------------------------------------------------------------------
// Validator: extra fields are NOT flagged in standard mode
// ---------------------------------------------------------------------------

func TestValidator_ExtraFieldsNotFlaggedInStandardMode(t *testing.T) {
	t.Parallel()
	v := NewValidator()

	schema := json.RawMessage(`{
		"properties": {"name": {"type": "string"}},
		"required": ["name"]
	}`)
	contract := &Contract{
		ID: "std1", Version: "1.0",
		RequestSchema: schema,
	}

	payload := json.RawMessage(`{"name": "Alice", "extra": "value", "another": 123}`)
	result := v.ValidatePayload(contract, payload, nil)
	assert.True(t, result.IsValid, "extra fields should not cause errors in standard mode")
	assert.Empty(t, result.Errors)
}

// ---------------------------------------------------------------------------
// BreakingChangeDetector: identical schemas produce no changes
// ---------------------------------------------------------------------------

func TestBreakingChangeDetector_IdenticalSchemasNoChanges(t *testing.T) {
	t.Parallel()
	d := NewBreakingChangeDetector()

	schema := json.RawMessage(`{
		"properties": {
			"name":   {"type": "string"},
			"age":    {"type": "number"},
			"active": {"type": "boolean"}
		},
		"required": ["name", "age"]
	}`)

	result, err := d.DetectChanges(schema, schema)
	require.NoError(t, err)
	assert.False(t, result.HasBreakingChanges, "identical schemas should have no breaking changes")
	assert.Empty(t, result.Changes, "identical schemas should produce no changes")
	assert.Empty(t, result.MigrationSteps, "no migration steps needed")
	assert.Empty(t, result.AffectedFields, "no affected fields")
}

// ---------------------------------------------------------------------------
// TestRunner: assertion edge cases
// ---------------------------------------------------------------------------

func TestTestRunner_AssertionEdgeCases(t *testing.T) {
	t.Parallel()
	runner := NewTestRunner()
	ctx := context.Background()

	contract := &Contract{
		ID: "c1", Version: "1.0",
		RequestSchema: json.RawMessage(`{
			"properties": {
				"name":   {"type": "string"},
				"count":  {"type": "number"},
				"active": {"type": "boolean"}
			},
			"required": ["name"]
		}`),
	}

	t.Run("exists assertion fails for missing field", func(t *testing.T) {
		t.Parallel()
		suite := &TestSuite{
			ID: "s-edge1", Name: "exists edge",
			TestCases: []TestCase{
				{
					Name:  "missing field",
					Input: json.RawMessage(`{"name": "Alice"}`),
					Assertions: []Assertion{
						{Type: "exists", Path: "$.missing_field"},
					},
				},
			},
		}
		result := runner.RunSuite(ctx, suite, contract)
		assert.Equal(t, "failed", result.Status)
		assert.Equal(t, 1, result.Failed)
		require.NotEmpty(t, result.Results[0].Assertions)
		assert.False(t, result.Results[0].Assertions[0].Passed)
	})

	t.Run("equals assertion with wrong value", func(t *testing.T) {
		t.Parallel()
		suite := &TestSuite{
			ID: "s-edge2", Name: "equals edge",
			TestCases: []TestCase{
				{
					Name:  "wrong value",
					Input: json.RawMessage(`{"name": "Alice"}`),
					Assertions: []Assertion{
						{Type: "equals", Path: "$.name", Expected: "Bob"},
					},
				},
			},
		}
		result := runner.RunSuite(ctx, suite, contract)
		assert.Equal(t, "failed", result.Status)
		require.NotEmpty(t, result.Results[0].Assertions)
		assert.False(t, result.Results[0].Assertions[0].Passed)
		assert.Contains(t, result.Results[0].Assertions[0].Message, "expected")
	})

	t.Run("type_is assertion fails for wrong type", func(t *testing.T) {
		t.Parallel()
		suite := &TestSuite{
			ID: "s-edge3", Name: "type_is edge",
			TestCases: []TestCase{
				{
					Name:  "wrong type",
					Input: json.RawMessage(`{"name": "Alice"}`),
					Assertions: []Assertion{
						{Type: "type_is", Path: "$.name", Expected: "number"},
					},
				},
			},
		}
		result := runner.RunSuite(ctx, suite, contract)
		assert.Equal(t, "failed", result.Status)
		require.NotEmpty(t, result.Results[0].Assertions)
		assert.False(t, result.Results[0].Assertions[0].Passed)
		assert.Equal(t, "string", result.Results[0].Assertions[0].Actual)
	})

	t.Run("exists assertion passes for present field", func(t *testing.T) {
		t.Parallel()
		suite := &TestSuite{
			ID: "s-edge4", Name: "exists pass",
			TestCases: []TestCase{
				{
					Name:  "field present",
					Input: json.RawMessage(`{"name": "Alice"}`),
					Assertions: []Assertion{
						{Type: "exists", Path: "$.name"},
					},
				},
			},
		}
		result := runner.RunSuite(ctx, suite, contract)
		assert.Equal(t, "passed", result.Status)
		require.NotEmpty(t, result.Results[0].Assertions)
		assert.True(t, result.Results[0].Assertions[0].Passed)
	})
}

// ---------------------------------------------------------------------------
// TestRunner: all tests skipped
// ---------------------------------------------------------------------------

func TestTestRunner_AllTestsSkipped(t *testing.T) {
	t.Parallel()
	runner := NewTestRunner()
	ctx := context.Background()

	contract := &Contract{
		ID: "c1", Version: "1.0",
		RequestSchema: json.RawMessage(`{
			"properties": {"name": {"type": "string"}},
			"required": ["name"]
		}`),
	}

	suite := &TestSuite{
		ID: "s-skip-all", Name: "all skipped",
		TestCases: []TestCase{
			{Name: "skip1", Input: json.RawMessage(`{"name": "A"}`), Skip: true},
			{Name: "skip2", Input: json.RawMessage(`{"name": "B"}`), Skip: true},
			{Name: "skip3", Input: json.RawMessage(`{}`), Skip: true},
		},
	}

	result := runner.RunSuite(ctx, suite, contract)
	assert.Equal(t, "passed", result.Status, "all-skipped suite should report passed")
	assert.Equal(t, 3, result.TotalTests)
	assert.Equal(t, 0, result.Passed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 3, result.Skipped)
	assert.NotNil(t, result.CompletedAt)
}

// ---------- Circular $ref and edge cases ----------

func TestValidator_CircularRefSchema(t *testing.T) {
	t.Parallel()
	v := NewValidator()

	// Schema with $ref (Validator doesn't resolve $ref, so it should not crash)
	contract := &Contract{
		ID: "c-ref", Version: "1.0",
		RequestSchema: json.RawMessage(`{
"properties": {
"parent": {"$ref": "#"},
"name": {"type": "string"}
},
"required": ["name"]
}`),
	}
	payload := json.RawMessage(`{"name": "test", "parent": {"name": "child"}}`)

	result := v.ValidatePayload(contract, payload, nil)

	// Should not crash; $ref is not resolved, so validation passes for the fields it does check
	assert.True(t, result.IsValid, "validator should not crash on $ref schemas")
}

func TestValidator_NullPayload(t *testing.T) {
	t.Parallel()
	v := NewValidator()

	contract := &Contract{
		ID: "c-null", Version: "1.0",
		RequestSchema: json.RawMessage(`{"properties": {"name": {"type": "string"}}}`),
	}
	payload := json.RawMessage(`null`)

	result := v.ValidatePayload(contract, payload, nil)

	// JSON null unmarshals to nil map, so validator treats it as valid (no fields to check)
	// This documents the current behavior - null payloads pass validation
	assert.True(t, result.IsValid, "null payload passes since no required fields")
}

func TestBreakingChangeDetector_InvalidOldSchema(t *testing.T) {
	t.Parallel()
	d := NewBreakingChangeDetector()

	_, err := d.DetectChanges(json.RawMessage(`not-json`), json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse old schema")
}

func TestBreakingChangeDetector_InvalidNewSchema(t *testing.T) {
	t.Parallel()
	d := NewBreakingChangeDetector()

	_, err := d.DetectChanges(json.RawMessage(`{}`), json.RawMessage(`not-json`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse new schema")
}
