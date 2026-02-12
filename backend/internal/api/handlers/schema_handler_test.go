package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/schema"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSchemaRepository implements schema.Repository for testing
type MockSchemaRepository struct {
	mock.Mock
}

func (m *MockSchemaRepository) CreateSchema(ctx context.Context, s *schema.Schema) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSchemaRepository) GetSchema(ctx context.Context, tenantID, schemaID string) (*schema.Schema, error) {
	args := m.Called(ctx, tenantID, schemaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Schema), args.Error(1)
}

func (m *MockSchemaRepository) GetSchemaByName(ctx context.Context, tenantID, name string) (*schema.Schema, error) {
	args := m.Called(ctx, tenantID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Schema), args.Error(1)
}

func (m *MockSchemaRepository) ListSchemas(ctx context.Context, tenantID string, limit, offset int) ([]schema.Schema, int, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]schema.Schema), args.Int(1), args.Error(2)
}

func (m *MockSchemaRepository) UpdateSchema(ctx context.Context, s *schema.Schema) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSchemaRepository) DeleteSchema(ctx context.Context, tenantID, schemaID string) error {
	args := m.Called(ctx, tenantID, schemaID)
	return args.Error(0)
}

func (m *MockSchemaRepository) CreateVersion(ctx context.Context, version *schema.SchemaVersion) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockSchemaRepository) GetVersion(ctx context.Context, schemaID, version string) (*schema.SchemaVersion, error) {
	args := m.Called(ctx, schemaID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.SchemaVersion), args.Error(1)
}

func (m *MockSchemaRepository) GetLatestVersion(ctx context.Context, schemaID string) (*schema.SchemaVersion, error) {
	args := m.Called(ctx, schemaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.SchemaVersion), args.Error(1)
}

func (m *MockSchemaRepository) ListVersions(ctx context.Context, schemaID string) ([]schema.SchemaVersion, error) {
	args := m.Called(ctx, schemaID)
	return args.Get(0).([]schema.SchemaVersion), args.Error(1)
}

func (m *MockSchemaRepository) AssignSchemaToEndpoint(ctx context.Context, assignment *schema.EndpointSchema) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockSchemaRepository) GetEndpointSchema(ctx context.Context, endpointID string) (*schema.EndpointSchema, error) {
	args := m.Called(ctx, endpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.EndpointSchema), args.Error(1)
}

func (m *MockSchemaRepository) RemoveSchemaFromEndpoint(ctx context.Context, endpointID string) error {
	args := m.Called(ctx, endpointID)
	return args.Error(0)
}

func (m *MockSchemaRepository) ListEndpointsWithSchema(ctx context.Context, schemaID string) ([]string, error) {
	args := m.Called(ctx, schemaID)
	return args.Get(0).([]string), args.Error(1)
}

func setupSchemaRouter(mockRepo *MockSchemaRepository, withTenant bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	svc := schema.NewService(mockRepo)
	logger := utils.NewLogger("test")
	handler := NewSchemaHandler(svc, logger)

	group := r.Group("/api/v1")
	if withTenant {
		group.Use(func(c *gin.Context) {
			c.Set("tenant_id", "tenant-123")
			c.Next()
		})
	}
	RegisterSchemaRoutes(group, handler)
	return r
}

// --- CreateSchema tests ---

func TestCreateSchema_Valid(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	// GetSchemaByName check (no duplicate)
	mockRepo.On("GetSchemaByName", mock.Anything, "tenant-123", "test-schema").Return(nil, nil)
	// CreateSchema
	mockRepo.On("CreateSchema", mock.Anything, mock.AnythingOfType("*schema.Schema")).
		Run(func(args mock.Arguments) {
			s := args.Get(1).(*schema.Schema)
			s.ID = "schema-1"
		}).Return(nil)
	// CreateVersion (initial version)
	mockRepo.On("CreateVersion", mock.Anything, mock.AnythingOfType("*schema.SchemaVersion")).Return(nil)

	body := map[string]interface{}{
		"name":        "test-schema",
		"version":     "1.0.0",
		"description": "A test schema",
		"json_schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/schemas", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestCreateSchema_MissingTenantID(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, false)

	body := map[string]interface{}{
		"name":    "test-schema",
		"version": "1.0.0",
		"json_schema": map[string]interface{}{
			"type": "object",
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/schemas", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateSchema_InvalidJSON(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/schemas", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- ListSchemas tests ---

func TestListSchemas_Valid(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	schemas := []schema.Schema{
		{ID: "s1", TenantID: "tenant-123", Name: "schema-1", Version: "1.0.0"},
		{ID: "s2", TenantID: "tenant-123", Name: "schema-2", Version: "1.0.0"},
	}
	mockRepo.On("ListSchemas", mock.Anything, "tenant-123", 50, 0).Return(schemas, 2, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/schemas", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []schema.Schema
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

// --- GetSchema tests ---

func TestGetSchema_Valid(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	s := &schema.Schema{ID: "schema-1", TenantID: "tenant-123", Name: "test-schema", Version: "1.0.0"}
	mockRepo.On("GetSchema", mock.Anything, "tenant-123", "schema-1").Return(s, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/schemas/schema-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result schema.Schema
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "schema-1", result.ID)
	mockRepo.AssertExpectations(t)
}

func TestGetSchema_NotFound(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	mockRepo.On("GetSchema", mock.Anything, "tenant-123", "nonexistent").Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/schemas/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

// --- DeleteSchema tests ---

func TestDeleteSchema_Valid(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	mockRepo.On("ListEndpointsWithSchema", mock.Anything, "schema-1").Return([]string{}, nil)
	mockRepo.On("DeleteSchema", mock.Anything, "tenant-123", "schema-1").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/schemas/schema-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockRepo.AssertExpectations(t)
}

// --- ValidatePayload tests ---

// --- CreateSchemaVersion tests ---

func TestCreateSchemaVersion_Valid(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	// GetSchema check (schema must exist)
	s := &schema.Schema{
		ID:         "schema-1",
		TenantID:   "tenant-123",
		Name:       "test-schema",
		Version:    "1.0.0",
		JSONSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
	}
	mockRepo.On("GetSchema", mock.Anything, "tenant-123", "schema-1").Return(s, nil)
	// GetVersion check (no duplicate — nil,nil means version doesn't exist yet)
	mockRepo.On("GetVersion", mock.Anything, "schema-1", "2.0.0").Return(nil, nil)
	// CreateVersion
	mockRepo.On("CreateVersion", mock.Anything, mock.AnythingOfType("*schema.SchemaVersion")).Return(nil)

	body := map[string]interface{}{
		"version": "2.0.0",
		"json_schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":  map[string]interface{}{"type": "string"},
				"email": map[string]interface{}{"type": "string"},
			},
		},
		"changelog": "Added email field",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/schemas/schema-1/versions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

// --- ListSchemaVersions tests ---

func TestListSchemaVersions_Valid(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	// GetSchema check (schema must exist)
	s := &schema.Schema{ID: "schema-1", TenantID: "tenant-123", Name: "test-schema", Version: "1.0.0"}
	mockRepo.On("GetSchema", mock.Anything, "tenant-123", "schema-1").Return(s, nil)
	// ListVersions
	versions := []schema.SchemaVersion{
		{ID: "v1", SchemaID: "schema-1", Version: "1.0.0"},
		{ID: "v2", SchemaID: "schema-1", Version: "2.0.0"},
	}
	mockRepo.On("ListVersions", mock.Anything, "schema-1").Return(versions, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/schemas/schema-1/versions", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []schema.SchemaVersion
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

// --- ValidatePayload tests ---

func TestValidatePayload_Valid(t *testing.T) {
	mockRepo := new(MockSchemaRepository)
	router := setupSchemaRouter(mockRepo, true)

	jsonSchema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`)
	s := &schema.Schema{
		ID:         "schema-1",
		TenantID:   "tenant-123",
		Name:       "test-schema",
		Version:    "1.0.0",
		JSONSchema: jsonSchema,
	}
	mockRepo.On("GetSchema", mock.Anything, "tenant-123", "schema-1").Return(s, nil)

	body := map[string]interface{}{
		"payload": map[string]interface{}{
			"name": "test",
		},
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/schemas/schema-1/validate", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result schema.ValidationResult
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.True(t, result.Valid)
	mockRepo.AssertExpectations(t)
}
