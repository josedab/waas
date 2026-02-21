package contracts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CIValidator.ValidateForCI
// ---------------------------------------------------------------------------

func TestCIValidator_ValidateForCI(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cv := NewCIValidator()

	schema := json.RawMessage(`{
		"properties": {"name": {"type":"string"}, "age": {"type":"number"}},
		"required": ["name"]
	}`)

	tests := []struct {
		name         string
		contracts    []Contract
		payloads     map[string]json.RawMessage
		wantPassed   bool
		wantFailed   int
		wantWarnings int
		wantChecks   int
	}{
		{
			name: "all contracts pass",
			contracts: []Contract{
				{ID: "c1", Name: "Contract 1", Version: "1.0", RequestSchema: schema},
			},
			payloads: map[string]json.RawMessage{
				"c1": json.RawMessage(`{"name":"Alice","age":30}`),
			},
			wantPassed: true,
			wantFailed: 0,
			wantChecks: 1,
		},
		{
			name: "validation failure",
			contracts: []Contract{
				{ID: "c1", Name: "Contract 1", Version: "1.0", RequestSchema: schema},
			},
			payloads: map[string]json.RawMessage{
				"c1": json.RawMessage(`{"age":"not-a-number"}`),
			},
			wantPassed: false,
			wantFailed: 1,
			wantChecks: 1,
		},
		{
			name: "missing payload generates warning",
			contracts: []Contract{
				{ID: "c1", Name: "Contract 1", Version: "1.0", RequestSchema: schema},
			},
			payloads:     map[string]json.RawMessage{},
			wantPassed:   true,
			wantFailed:   0,
			wantWarnings: 1,
			wantChecks:   1,
		},
		{
			name:       "empty contracts list",
			contracts:  []Contract{},
			payloads:   map[string]json.RawMessage{},
			wantPassed: true,
			wantFailed: 0,
			wantChecks: 0,
		},
		{
			name: "multiple contracts mixed results",
			contracts: []Contract{
				{ID: "c1", Name: "Contract 1", Version: "1.0", RequestSchema: schema},
				{ID: "c2", Name: "Contract 2", Version: "1.0", RequestSchema: schema},
			},
			payloads: map[string]json.RawMessage{
				"c1": json.RawMessage(`{"name":"Alice"}`),
				"c2": json.RawMessage(`{"age":30}`),
			},
			wantPassed: false,
			wantFailed: 1,
			wantChecks: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cv.ValidateForCI(ctx, tt.contracts, tt.payloads)

			assert.Equal(t, tt.wantPassed, result.Passed)
			assert.Equal(t, tt.wantFailed, result.FailedChecks)
			assert.Equal(t, tt.wantChecks, result.TotalChecks)
			assert.Equal(t, tt.wantWarnings, result.Warnings)
			assert.NotEmpty(t, result.ID)
			assert.NotEmpty(t, result.RunID)
			assert.NotZero(t, result.ValidatedAt)
		})
	}
}

// ---------------------------------------------------------------------------
// CIValidator.GenerateSARIF
// ---------------------------------------------------------------------------

func TestCIValidator_GenerateSARIF(t *testing.T) {
	t.Parallel()
	cv := NewCIValidator()

	t.Run("report with violations", func(t *testing.T) {
		t.Parallel()
		result := &CIValidationResult{
			ID:    "r1",
			RunID: "run-1",
			Violations: []CIViolation{
				{RuleID: "MISSING_FIELD", Level: "error", Message: "required field 'name' is missing", Path: "$.name", ContractID: "c1"},
				{RuleID: "TYPE_MISMATCH", Level: "error", Message: "expected type 'number'", Path: "$.age", ContractID: "c1"},
			},
		}

		sarif := cv.GenerateSARIF(result)

		assert.Equal(t, "2.1.0", sarif.Version)
		assert.NotEmpty(t, sarif.Schema)
		require.Len(t, sarif.Runs, 1)
		assert.Equal(t, "waas-contract-validator", sarif.Runs[0].Tool.Driver.Name)
		assert.Len(t, sarif.Runs[0].Results, 2)
		assert.NotEmpty(t, sarif.Runs[0].Tool.Driver.Rules)

		// Verify rules are deduplicated
		ruleIDs := make(map[string]bool)
		for _, rule := range sarif.Runs[0].Tool.Driver.Rules {
			ruleIDs[rule.ID] = true
		}
		assert.True(t, ruleIDs["MISSING_FIELD"])
		assert.True(t, ruleIDs["TYPE_MISMATCH"])
	})

	t.Run("report with no violations", func(t *testing.T) {
		t.Parallel()
		result := &CIValidationResult{
			ID:     "r2",
			RunID:  "run-2",
			Passed: true,
		}

		sarif := cv.GenerateSARIF(result)

		assert.Equal(t, "2.1.0", sarif.Version)
		require.Len(t, sarif.Runs, 1)
		assert.Empty(t, sarif.Runs[0].Results)
		assert.Empty(t, sarif.Runs[0].Tool.Driver.Rules)
	})

	t.Run("SARIF locations use logical locations for JSON paths", func(t *testing.T) {
		t.Parallel()
		result := &CIValidationResult{
			Violations: []CIViolation{
				{RuleID: "MISSING_FIELD", Level: "error", Message: "missing", Path: "$.name"},
			},
		}

		sarif := cv.GenerateSARIF(result)
		require.Len(t, sarif.Runs[0].Results, 1)
		require.NotEmpty(t, sarif.Runs[0].Results[0].Locations)
		loc := sarif.Runs[0].Results[0].Locations[0]
		require.NotEmpty(t, loc.LogicalLocations)
		assert.Equal(t, "$.name", loc.LogicalLocations[0].Name)
		assert.Equal(t, "jsonPath", loc.LogicalLocations[0].Kind)
	})

	t.Run("SARIF is valid JSON", func(t *testing.T) {
		t.Parallel()
		result := &CIValidationResult{
			Violations: []CIViolation{
				{RuleID: "TEST", Level: "warning", Message: "test warning"},
			},
		}

		sarif := cv.GenerateSARIF(result)
		data, err := json.Marshal(sarif)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var parsed SARIFReport
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		assert.Equal(t, "2.1.0", parsed.Version)
	})
}

// ---------------------------------------------------------------------------
// BreakingChangeChecker.CheckBreakingChanges
// ---------------------------------------------------------------------------

func TestBreakingChangeChecker_CheckBreakingChanges(t *testing.T) {
	t.Parallel()
	checker := NewBreakingChangeChecker()

	t.Run("detects breaking change when field removed", func(t *testing.T) {
		t.Parallel()
		oldContract := &Contract{
			ID: "c1", Version: "1.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "age": {"type":"number"}},
				"required": ["name"]
			}`),
		}
		newContract := &Contract{
			ID: "c1", Version: "2.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
		}

		result, err := checker.CheckBreakingChanges(oldContract, newContract)
		require.NoError(t, err)
		assert.True(t, result.HasBreakingChanges)
		assert.Equal(t, "c1", result.ContractID)
		assert.Equal(t, "1.0", result.OldVersion)
		assert.Equal(t, "2.0", result.NewVersion)
		assert.NotEmpty(t, result.ID)
	})

	t.Run("no breaking changes for optional field addition", func(t *testing.T) {
		t.Parallel()
		oldContract := &Contract{
			ID: "c1", Version: "1.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
		}
		newContract := &Contract{
			ID: "c1", Version: "1.1",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "email": {"type":"string"}},
				"required": ["name"]
			}`),
		}

		result, err := checker.CheckBreakingChanges(oldContract, newContract)
		require.NoError(t, err)
		assert.False(t, result.HasBreakingChanges)
	})

	t.Run("invalid schema returns error", func(t *testing.T) {
		t.Parallel()
		oldContract := &Contract{
			ID: "c1", Version: "1.0",
			RequestSchema: json.RawMessage(`not-json`),
		}
		newContract := &Contract{
			ID: "c1", Version: "2.0",
			RequestSchema: json.RawMessage(`{"properties":{}}`),
		}

		_, err := checker.CheckBreakingChanges(oldContract, newContract)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check breaking changes")
	})
}

// ---------------------------------------------------------------------------
// SchemaRegistryClient
// ---------------------------------------------------------------------------

func TestSchemaRegistryClient_RegisterSchema(t *testing.T) {
	t.Parallel()

	t.Run("register first schema version", func(t *testing.T) {
		t.Parallel()
		registry := NewSchemaRegistryClient()

		contract := &Contract{
			ID: "c1", Version: "1.0",
			RequestSchema: json.RawMessage(`{"properties": {"name": {"type":"string"}}}`),
		}

		entry, err := registry.RegisterSchema(contract)
		require.NoError(t, err)
		assert.NotEmpty(t, entry.ID)
		assert.Equal(t, "c1", entry.ContractID)
		assert.Equal(t, "1.0", entry.Version)
		assert.False(t, entry.IsBreaking)
		assert.NotZero(t, entry.RegisteredAt)
	})

	t.Run("register breaking schema version", func(t *testing.T) {
		t.Parallel()
		registry := NewSchemaRegistryClient()

		v1 := &Contract{
			ID: "c1", Version: "1.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "age": {"type":"number"}},
				"required": ["name"]
			}`),
		}
		_, err := registry.RegisterSchema(v1)
		require.NoError(t, err)

		v2 := &Contract{
			ID: "c1", Version: "2.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
		}
		entry, err := registry.RegisterSchema(v2)
		require.NoError(t, err)
		assert.True(t, entry.IsBreaking)
		assert.Equal(t, "2.0", entry.Version)
	})

	t.Run("nil contract returns error", func(t *testing.T) {
		t.Parallel()
		registry := NewSchemaRegistryClient()

		_, err := registry.RegisterSchema(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "contract must not be nil")
	})

	t.Run("invalid schema JSON returns error", func(t *testing.T) {
		t.Parallel()
		registry := NewSchemaRegistryClient()

		contract := &Contract{
			ID: "c1", Version: "1.0",
			RequestSchema: json.RawMessage(`not-json`),
		}

		_, err := registry.RegisterSchema(contract)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "schema must be valid JSON")
	})
}

func TestSchemaRegistryClient_GetVersions(t *testing.T) {
	t.Parallel()
	registry := NewSchemaRegistryClient()

	schema := json.RawMessage(`{"properties": {"name": {"type":"string"}}}`)
	for _, v := range []string{"1.0", "1.1", "2.0"} {
		_, err := registry.RegisterSchema(&Contract{
			ID: "c1", Version: v, RequestSchema: schema,
		})
		require.NoError(t, err)
	}

	versions := registry.GetVersions("c1")
	assert.Len(t, versions, 3)
	assert.Equal(t, "1.0", versions[0].Version)
	assert.Equal(t, "2.0", versions[2].Version)
}

func TestSchemaRegistryClient_GetVersions_Empty(t *testing.T) {
	t.Parallel()
	registry := NewSchemaRegistryClient()

	versions := registry.GetVersions("nonexistent")
	assert.Empty(t, versions)
}

func TestSchemaRegistryClient_GetLatestVersion(t *testing.T) {
	t.Parallel()
	registry := NewSchemaRegistryClient()

	schema := json.RawMessage(`{"properties": {"name": {"type":"string"}}}`)
	for _, v := range []string{"1.0", "2.0"} {
		_, err := registry.RegisterSchema(&Contract{
			ID: "c1", Version: v, RequestSchema: schema,
		})
		require.NoError(t, err)
	}

	latest := registry.GetLatestVersion("c1")
	require.NotNil(t, latest)
	assert.Equal(t, "2.0", latest.Version)
}

func TestSchemaRegistryClient_GetLatestVersion_Empty(t *testing.T) {
	t.Parallel()
	registry := NewSchemaRegistryClient()

	latest := registry.GetLatestVersion("nonexistent")
	assert.Nil(t, latest)
}

// ---------------------------------------------------------------------------
// GenerateChangelog
// ---------------------------------------------------------------------------

func TestGenerateChangelog(t *testing.T) {
	t.Parallel()

	t.Run("changelog from multiple versions", func(t *testing.T) {
		t.Parallel()
		registry := NewSchemaRegistryClient()

		v1 := &Contract{
			ID: "c1", Version: "1.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "age": {"type":"number"}},
				"required": ["name"]
			}`),
		}
		_, err := registry.RegisterSchema(v1)
		require.NoError(t, err)

		v2 := &Contract{
			ID: "c1", Version: "2.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "email": {"type":"string"}},
				"required": ["name"]
			}`),
		}
		_, err = registry.RegisterSchema(v2)
		require.NoError(t, err)

		entries := registry.GetVersions("c1")
		changelog := GenerateChangelog(entries)

		require.Len(t, changelog, 1)
		assert.Equal(t, "2.0", changelog[0].Version)
		assert.True(t, changelog[0].IsBreaking, "removing 'age' is breaking")
		assert.Contains(t, changelog[0].Summary, "BREAKING")
		assert.NotEmpty(t, changelog[0].Changes)
	})

	t.Run("non-breaking changelog", func(t *testing.T) {
		t.Parallel()
		registry := NewSchemaRegistryClient()

		v1 := &Contract{
			ID: "c1", Version: "1.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}},
				"required": ["name"]
			}`),
		}
		_, err := registry.RegisterSchema(v1)
		require.NoError(t, err)

		v2 := &Contract{
			ID: "c1", Version: "1.1",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "email": {"type":"string"}},
				"required": ["name"]
			}`),
		}
		_, err = registry.RegisterSchema(v2)
		require.NoError(t, err)

		entries := registry.GetVersions("c1")
		changelog := GenerateChangelog(entries)

		require.Len(t, changelog, 1)
		assert.Equal(t, "1.1", changelog[0].Version)
		assert.False(t, changelog[0].IsBreaking)
		assert.NotContains(t, changelog[0].Summary, "BREAKING")
	})

	t.Run("single entry returns nil", func(t *testing.T) {
		t.Parallel()
		entries := []SchemaRegistryEntry{
			{Version: "1.0", Schema: json.RawMessage(`{}`)},
		}
		changelog := GenerateChangelog(entries)
		assert.Nil(t, changelog)
	})

	t.Run("empty entries returns nil", func(t *testing.T) {
		t.Parallel()
		changelog := GenerateChangelog(nil)
		assert.Nil(t, changelog)
	})
}

// ---------------------------------------------------------------------------
// End-to-end: validate → SARIF → breaking changes → registry → changelog
// ---------------------------------------------------------------------------

func TestCIIntegration_EndToEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cv := NewCIValidator()
	checker := NewBreakingChangeChecker()
	registry := NewSchemaRegistryClient()

	// Register v1
	v1 := &Contract{
		ID: "c1", Name: "Order Contract", Version: "1.0",
		RequestSchema: json.RawMessage(`{
			"properties": {"order_id": {"type":"string"}, "amount": {"type":"number"}},
			"required": ["order_id", "amount"]
		}`),
	}
	_, err := registry.RegisterSchema(v1)
	require.NoError(t, err)

	// Validate v1 payload
	result := cv.ValidateForCI(ctx, []Contract{*v1}, map[string]json.RawMessage{
		"c1": json.RawMessage(`{"order_id":"ORD-001","amount":99.99}`),
	})
	assert.True(t, result.Passed)

	// Generate SARIF for the result
	sarif := cv.GenerateSARIF(result)
	assert.Equal(t, "2.1.0", sarif.Version)
	assert.Empty(t, sarif.Runs[0].Results)

	// Register v2 with a breaking change (removed field)
	v2 := &Contract{
		ID: "c1", Name: "Order Contract", Version: "2.0",
		RequestSchema: json.RawMessage(`{
			"properties": {"order_id": {"type":"string"}},
			"required": ["order_id"]
		}`),
	}
	entry, err := registry.RegisterSchema(v2)
	require.NoError(t, err)
	assert.True(t, entry.IsBreaking)

	// Check breaking changes
	bc, err := checker.CheckBreakingChanges(v1, v2)
	require.NoError(t, err)
	assert.True(t, bc.HasBreakingChanges)

	// Generate changelog
	entries := registry.GetVersions("c1")
	changelog := GenerateChangelog(entries)
	require.Len(t, changelog, 1)
	assert.True(t, changelog[0].IsBreaking)
}
