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
// MockRepository – full mock for Service tests
// ---------------------------------------------------------------------------

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateSchema(ctx context.Context, schema *Schema) error {
	return m.Called(ctx, schema).Error(0)
}

func (m *MockRepository) GetSchema(ctx context.Context, tenantID, schemaID string) (*Schema, error) {
	args := m.Called(ctx, tenantID, schemaID)
	if v := args.Get(0); v != nil {
		return v.(*Schema), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) GetSchemaByName(ctx context.Context, tenantID, name string) (*Schema, error) {
	args := m.Called(ctx, tenantID, name)
	if v := args.Get(0); v != nil {
		return v.(*Schema), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) ListSchemas(ctx context.Context, tenantID string, limit, offset int) ([]Schema, int, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]Schema), args.Int(1), args.Error(2)
}

func (m *MockRepository) UpdateSchema(ctx context.Context, schema *Schema) error {
	return m.Called(ctx, schema).Error(0)
}

func (m *MockRepository) DeleteSchema(ctx context.Context, tenantID, schemaID string) error {
	return m.Called(ctx, tenantID, schemaID).Error(0)
}

func (m *MockRepository) CreateVersion(ctx context.Context, version *SchemaVersion) error {
	return m.Called(ctx, version).Error(0)
}

func (m *MockRepository) GetVersion(ctx context.Context, schemaID, version string) (*SchemaVersion, error) {
	args := m.Called(ctx, schemaID, version)
	if v := args.Get(0); v != nil {
		return v.(*SchemaVersion), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) GetLatestVersion(ctx context.Context, schemaID string) (*SchemaVersion, error) {
	args := m.Called(ctx, schemaID)
	if v := args.Get(0); v != nil {
		return v.(*SchemaVersion), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) ListVersions(ctx context.Context, schemaID string) ([]SchemaVersion, error) {
	args := m.Called(ctx, schemaID)
	return args.Get(0).([]SchemaVersion), args.Error(1)
}

func (m *MockRepository) AssignSchemaToEndpoint(ctx context.Context, assignment *EndpointSchema) error {
	return m.Called(ctx, assignment).Error(0)
}

func (m *MockRepository) GetEndpointSchema(ctx context.Context, endpointID string) (*EndpointSchema, error) {
	args := m.Called(ctx, endpointID)
	if v := args.Get(0); v != nil {
		return v.(*EndpointSchema), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) RemoveSchemaFromEndpoint(ctx context.Context, endpointID string) error {
	return m.Called(ctx, endpointID).Error(0)
}

func (m *MockRepository) ListEndpointsWithSchema(ctx context.Context, schemaID string) ([]string, error) {
	args := m.Called(ctx, schemaID)
	if v := args.Get(0); v != nil {
		return v.([]string), args.Error(1)
	}
	return nil, args.Error(1)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func validJSONSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
}

// ---------------------------------------------------------------------------
// CreateSchema
// ---------------------------------------------------------------------------

func TestService_CreateSchema_Valid(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchemaByName", ctx, "t1", "webhook-v1").Return(nil, nil)
	repo.On("CreateSchema", ctx, mock.AnythingOfType("*schema.Schema")).Return(nil)
	repo.On("CreateVersion", ctx, mock.AnythingOfType("*schema.SchemaVersion")).Return(nil)

	s, err := svc.CreateSchema(ctx, "t1", &CreateSchemaRequest{
		Name:       "webhook-v1",
		Version:    "1.0",
		JSONSchema: validJSONSchema(),
	})
	require.NoError(t, err)
	assert.Equal(t, "webhook-v1", s.Name)
	assert.True(t, s.IsActive)
	repo.AssertExpectations(t)
}

func TestService_CreateSchema_DuplicateName(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchemaByName", ctx, "t1", "dup").Return(&Schema{Name: "dup"}, nil)

	_, err := svc.CreateSchema(ctx, "t1", &CreateSchemaRequest{
		Name:       "dup",
		Version:    "1.0",
		JSONSchema: validJSONSchema(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	repo.AssertExpectations(t)
}

func TestService_CreateSchema_InvalidJSONSchema(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	// Payload "not_an_object" does not match {"type":"object"} used by service
	_, err := svc.CreateSchema(ctx, "t1", &CreateSchemaRequest{
		Name:       "bad",
		Version:    "1.0",
		JSONSchema: json.RawMessage(`"not_an_object"`),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON schema")
}

func TestService_CreateSchema_CreateVersionErrorSwallowed(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchemaByName", ctx, "t1", "s1").Return(nil, nil)
	repo.On("CreateSchema", ctx, mock.AnythingOfType("*schema.Schema")).Return(nil)
	repo.On("CreateVersion", ctx, mock.AnythingOfType("*schema.SchemaVersion")).Return(fmt.Errorf("version error"))

	s, err := svc.CreateSchema(ctx, "t1", &CreateSchemaRequest{
		Name:       "s1",
		Version:    "1.0",
		JSONSchema: validJSONSchema(),
	})
	// Schema should still be returned despite version creation error
	require.NoError(t, err)
	assert.NotNil(t, s)
	repo.AssertExpectations(t)
}

func TestService_CreateSchema_ConcurrentDuplicateName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// First call: name does not exist yet → succeeds
	repo1 := new(MockRepository)
	svc1 := NewService(repo1)
	repo1.On("GetSchemaByName", ctx, "t1", "concurrent").Return(nil, nil)
	repo1.On("CreateSchema", ctx, mock.AnythingOfType("*schema.Schema")).Return(nil)
	repo1.On("CreateVersion", ctx, mock.AnythingOfType("*schema.SchemaVersion")).Return(nil)

	s, err := svc1.CreateSchema(ctx, "t1", &CreateSchemaRequest{
		Name:       "concurrent",
		Version:    "1.0",
		JSONSchema: validJSONSchema(),
	})
	require.NoError(t, err)
	assert.Equal(t, "concurrent", s.Name)
	repo1.AssertExpectations(t)

	// Second call: name already exists → fails
	repo2 := new(MockRepository)
	svc2 := NewService(repo2)
	repo2.On("GetSchemaByName", ctx, "t1", "concurrent").Return(&Schema{Name: "concurrent"}, nil)

	_, err = svc2.CreateSchema(ctx, "t1", &CreateSchemaRequest{
		Name:       "concurrent",
		Version:    "1.0",
		JSONSchema: validJSONSchema(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	repo2.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// UpdateSchema
// ---------------------------------------------------------------------------

func TestService_UpdateSchema_Valid(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	existing := &Schema{ID: "s1", TenantID: "t1", Name: "test", IsActive: true}
	repo.On("GetSchema", ctx, "t1", "s1").Return(existing, nil)
	repo.On("UpdateSchema", ctx, mock.AnythingOfType("*schema.Schema")).Return(nil)

	result, err := svc.UpdateSchema(ctx, "t1", "s1", &UpdateSchemaRequest{
		Description: "updated",
		IsActive:    false,
		IsDefault:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, "updated", result.Description)
	assert.False(t, result.IsActive)
	assert.True(t, result.IsDefault)
	repo.AssertExpectations(t)
}

func TestService_UpdateSchema_NotFound(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchema", ctx, "t1", "s1").Return(nil, nil)

	_, err := svc.UpdateSchema(ctx, "t1", "s1", &UpdateSchemaRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema not found")
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteSchema
// ---------------------------------------------------------------------------

func TestService_DeleteSchema_NotInUse(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListEndpointsWithSchema", ctx, "s1").Return([]string{}, nil)
	repo.On("DeleteSchema", ctx, "t1", "s1").Return(nil)

	err := svc.DeleteSchema(ctx, "t1", "s1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_DeleteSchema_InUse(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("ListEndpointsWithSchema", ctx, "s1").Return([]string{"ep1", "ep2"}, nil)

	err := svc.DeleteSchema(ctx, "t1", "s1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "assigned to 2 endpoints")
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ListSchemas – limit clamping
// ---------------------------------------------------------------------------

func TestService_ListSchemas_LimitClamping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero defaults to 20", 0, 20},
		{"negative defaults to 20", -5, 20},
		{"within range", 50, 50},
		{"exceeds max clamped to 100", 200, 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := new(MockRepository)
			svc := NewService(repo)
			ctx := context.Background()

			repo.On("ListSchemas", ctx, "t1", tc.expectedLimit, 0).Return([]Schema{}, 0, nil)

			_, _, err := svc.ListSchemas(ctx, "t1", tc.inputLimit, 0)
			assert.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// CreateVersion
// ---------------------------------------------------------------------------

func TestService_CreateVersion_Valid(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	existingSchema := &Schema{
		ID:         "s1",
		TenantID:   "t1",
		JSONSchema: validJSONSchema(),
	}
	repo.On("GetSchema", ctx, "t1", "s1").Return(existingSchema, nil)
	repo.On("GetVersion", ctx, "s1", "2.0").Return(nil, nil)
	repo.On("CreateVersion", ctx, mock.AnythingOfType("*schema.SchemaVersion")).Return(nil)

	newSchema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}`)
	ver, compat, err := svc.CreateVersion(ctx, "t1", "s1", &CreateVersionRequest{
		Version:    "2.0",
		JSONSchema: newSchema,
		Changelog:  "added email",
	})
	require.NoError(t, err)
	assert.Equal(t, "2.0", ver.Version)
	assert.True(t, compat.Compatible)
	repo.AssertExpectations(t)
}

func TestService_CreateVersion_DuplicateVersion(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchema", ctx, "t1", "s1").Return(&Schema{ID: "s1", JSONSchema: validJSONSchema()}, nil)
	repo.On("GetVersion", ctx, "s1", "1.0").Return(&SchemaVersion{Version: "1.0"}, nil)

	_, _, err := svc.CreateVersion(ctx, "t1", "s1", &CreateVersionRequest{
		Version:    "1.0",
		JSONSchema: validJSONSchema(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	repo.AssertExpectations(t)
}

func TestService_CreateVersion_CompatibilityCheck(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchema", ctx, "t1", "s1").Return(&Schema{
		ID:         "s1",
		JSONSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
	}, nil)
	repo.On("GetVersion", ctx, "s1", "2.0").Return(nil, nil)
	repo.On("CreateVersion", ctx, mock.AnythingOfType("*schema.SchemaVersion")).Return(nil)

	// Add a new required field → breaking change
	newSchema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}},"required":["name","age"]}`)
	ver, compat, err := svc.CreateVersion(ctx, "t1", "s1", &CreateVersionRequest{
		Version:    "2.0",
		JSONSchema: newSchema,
	})
	require.NoError(t, err)
	assert.NotNil(t, ver)
	assert.False(t, compat.Compatible)
	assert.NotEmpty(t, compat.BreakingChanges)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// AssignSchemaToEndpoint
// ---------------------------------------------------------------------------

func TestService_AssignSchemaToEndpoint_Valid(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchema", ctx, "t1", "s1").Return(&Schema{ID: "s1"}, nil)
	repo.On("AssignSchemaToEndpoint", ctx, mock.AnythingOfType("*schema.EndpointSchema")).Return(nil)

	err := svc.AssignSchemaToEndpoint(ctx, "t1", "ep1", &AssignSchemaRequest{
		SchemaID:       "s1",
		ValidationMode: ValidationModeStrict,
	})
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_AssignSchemaToEndpoint_SchemaNotFound(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchema", ctx, "t1", "s1").Return(nil, nil)

	err := svc.AssignSchemaToEndpoint(ctx, "t1", "ep1", &AssignSchemaRequest{
		SchemaID:       "s1",
		ValidationMode: ValidationModeStrict,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema not found")
	repo.AssertExpectations(t)
}

func TestService_AssignSchemaToEndpoint_VersionNotFound(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchema", ctx, "t1", "s1").Return(&Schema{ID: "s1"}, nil)
	repo.On("GetVersion", ctx, "s1", "9.9").Return(nil, nil)

	err := svc.AssignSchemaToEndpoint(ctx, "t1", "ep1", &AssignSchemaRequest{
		SchemaID:       "s1",
		SchemaVersion:  "9.9",
		ValidationMode: ValidationModeStrict,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version '9.9' not found")
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ValidatePayloadDirect (service-level delegation)
// ---------------------------------------------------------------------------

func TestService_ValidatePayloadDirect(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchema", ctx, "t1", "s1").Return(&Schema{
		ID:         "s1",
		Name:       "test",
		Version:    "1.0",
		JSONSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
	}, nil)

	result, err := svc.ValidatePayloadDirect(ctx, "t1", "s1", []byte(`{"name":"alice"}`))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, "s1", result.SchemaID)
	assert.Equal(t, "test", result.SchemaName)
	repo.AssertExpectations(t)
}

func TestService_ValidatePayloadDirect_SchemaNotFound(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	repo.On("GetSchema", ctx, "t1", "s1").Return(nil, nil)

	_, err := svc.ValidatePayloadDirect(ctx, "t1", "s1", []byte(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema not found")
	repo.AssertExpectations(t)
}

// ---------- Schema version '0.0.0' ----------

func TestService_CreateSchema_VersionZero(t *testing.T) {
	t.Parallel()
	repo := new(MockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	req := &CreateSchemaRequest{
		Name:       "zero-version-schema",
		Version:    "0.0.0",
		JSONSchema: json.RawMessage(`{"type": "object"}`),
	}

	repo.On("GetSchemaByName", ctx, "t1", "zero-version-schema").Return(nil, nil)
	repo.On("CreateSchema", ctx, mock.AnythingOfType("*schema.Schema")).Return(nil)
	repo.On("CreateVersion", ctx, mock.AnythingOfType("*schema.SchemaVersion")).Return(nil)

	result, err := svc.CreateSchema(ctx, "t1", req)

	require.NoError(t, err)
	assert.Equal(t, "0.0.0", result.Version)
	assert.True(t, result.IsActive)
	repo.AssertExpectations(t)
}
