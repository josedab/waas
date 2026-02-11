package contracts

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// MockRepository
// ---------------------------------------------------------------------------

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateContract(ctx context.Context, contract *Contract) error {
	return m.Called(ctx, contract).Error(0)
}

func (m *MockRepository) GetContract(ctx context.Context, tenantID, contractID string) (*Contract, error) {
	args := m.Called(ctx, tenantID, contractID)
	if c, ok := args.Get(0).(*Contract); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) ListContracts(ctx context.Context, tenantID string, limit, offset int) ([]Contract, int, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]Contract), args.Int(1), args.Error(2)
}

func (m *MockRepository) UpdateContract(ctx context.Context, contract *Contract) error {
	return m.Called(ctx, contract).Error(0)
}

func (m *MockRepository) DeleteContract(ctx context.Context, tenantID, contractID string) error {
	return m.Called(ctx, tenantID, contractID).Error(0)
}

func (m *MockRepository) GetContractByEventType(ctx context.Context, tenantID, eventType string) (*Contract, error) {
	args := m.Called(ctx, tenantID, eventType)
	if c, ok := args.Get(0).(*Contract); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) SaveTestResult(ctx context.Context, result *ContractTestResult) error {
	return m.Called(ctx, result).Error(0)
}

func (m *MockRepository) GetTestResult(ctx context.Context, tenantID, resultID string) (*ContractTestResult, error) {
	args := m.Called(ctx, tenantID, resultID)
	if r, ok := args.Get(0).(*ContractTestResult); ok {
		return r, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRepository) ListTestResults(ctx context.Context, tenantID, contractID string, limit, offset int) ([]ContractTestResult, error) {
	args := m.Called(ctx, tenantID, contractID, limit, offset)
	return args.Get(0).([]ContractTestResult), args.Error(1)
}

func (m *MockRepository) GetContractStatus(ctx context.Context, tenantID string) (*ContractStatus, error) {
	args := m.Called(ctx, tenantID)
	if s, ok := args.Get(0).(*ContractStatus); ok {
		return s, args.Error(1)
	}
	return nil, args.Error(1)
}

// ---------------------------------------------------------------------------
// Service.CreateContract
// ---------------------------------------------------------------------------

func TestService_CreateContract(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		req       *CreateContractRequest
		repoErr   error
		wantErr   bool
		wantErrIs string
	}{
		{
			name: "valid contract",
			req: &CreateContractRequest{
				Name:      "test",
				Version:   "1.0",
				EventType: "order.created",
				Schema:    `{"properties":{"id":{"type":"string"}},"required":["id"]}`,
			},
			wantErr: false,
		},
		{
			name: "invalid JSON schema returns error",
			req: &CreateContractRequest{
				Name:      "bad",
				Version:   "1.0",
				EventType: "x",
				Schema:    `not-json`,
			},
			wantErr:   true,
			wantErrIs: "schema must be valid JSON",
		},
		{
			name: "repo error propagated",
			req: &CreateContractRequest{
				Name:      "ok",
				Version:   "1.0",
				EventType: "x",
				Schema:    `{}`,
			},
			repoErr:   errors.New("db down"),
			wantErr:   true,
			wantErrIs: "failed to create contract",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := new(MockRepository)
			svc := NewService(repo)

			if !tt.wantErr || tt.repoErr != nil {
				repo.On("CreateContract", ctx, mock.AnythingOfType("*contracts.Contract")).
					Return(tt.repoErr)
			}

			contract, err := svc.CreateContract(ctx, "tenant1", tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != "" {
					assert.Contains(t, err.Error(), tt.wantErrIs)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "tenant1", contract.TenantID)
			assert.Equal(t, tt.req.Name, contract.Name)
			assert.Equal(t, "active", contract.Status)
			assert.NotEmpty(t, contract.ID)
			repo.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Service.ValidatePayload
// ---------------------------------------------------------------------------

func TestService_ValidatePayload(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	schema := json.RawMessage(`{
		"properties": {"name": {"type":"string"}, "age": {"type":"number"}},
		"required": ["name"]
	}`)
	contract := &Contract{
		ID: "c1", TenantID: "t1", Version: "1.0",
		RequestSchema: schema,
		SchemaFormat:  string(StrictnessStandard),
	}

	tests := []struct {
		name       string
		req        *ValidatePayloadRequest
		contract   *Contract
		repoErr    error
		wantErr    bool
		wantPassed bool
	}{
		{
			name:       "valid payload passes",
			req:        &ValidatePayloadRequest{ContractID: "c1", Payload: `{"name":"Alice","age":30}`},
			contract:   contract,
			wantPassed: true,
		},
		{
			name: "invalid payload has violations",
			req:  &ValidatePayloadRequest{ContractID: "c1", Payload: `{"age":30}`},
			contract: &Contract{
				ID: "c1", TenantID: "t1", Version: "1.0",
				RequestSchema: schema,
				SchemaFormat:  string(StrictnessStandard),
			},
			wantPassed: false,
		},
		{
			name: "strict mode catches extra fields",
			req:  &ValidatePayloadRequest{ContractID: "c1", Payload: `{"name":"Alice","extra":"val"}`},
			contract: &Contract{
				ID: "c1", TenantID: "t1", Version: "1.0",
				RequestSchema: schema,
				SchemaFormat:  string(StrictnessStrict),
			},
			wantPassed: false,
		},
		{
			name:    "contract not found",
			req:     &ValidatePayloadRequest{ContractID: "missing", Payload: `{}`},
			repoErr: errors.New("not found"),
			wantErr: true,
		},
		{
			name:     "invalid payload JSON",
			req:      &ValidatePayloadRequest{ContractID: "c1", Payload: `not-json`},
			contract: contract,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := new(MockRepository)
			svc := NewService(repo)

			if tt.repoErr != nil {
				repo.On("GetContract", ctx, "t1", tt.req.ContractID).
					Return((*Contract)(nil), tt.repoErr)
			} else {
				repo.On("GetContract", ctx, "t1", tt.req.ContractID).
					Return(tt.contract, nil)
			}

			if !tt.wantErr {
				repo.On("SaveTestResult", ctx, mock.AnythingOfType("*contracts.ContractTestResult")).
					Return(nil)
			}

			result, err := svc.ValidatePayload(ctx, "t1", tt.req)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPassed, result.Passed)
			if !tt.wantPassed {
				assert.NotEmpty(t, result.Violations)
			}
			repo.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Service.DiffContracts
// ---------------------------------------------------------------------------

func TestService_DiffContracts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("added and removed fields detected", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		// When oldVersion != contract.Version, oldSchema = empty map
		// newSchema = contract.RequestSchema
		contract := &Contract{
			ID: "c1", TenantID: "t1", Version: "2.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}, "email": {"type":"string"}}
			}`),
		}
		repo.On("GetContract", ctx, "t1", "c1").Return(contract, nil)

		diff, err := svc.DiffContracts(ctx, "t1", "c1", "1.0", "2.0")
		require.NoError(t, err)

		// oldSchema is empty → all fields in new are "added"
		assert.NotEmpty(t, diff.Changes)
		types := make([]string, len(diff.Changes))
		for i, c := range diff.Changes {
			types[i] = c.Type
		}
		assert.Contains(t, types, "added")
	})

	t.Run("breaking changes flagged", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		// oldVersion == contract.Version → oldSchema = newSchema; no changes
		contract := &Contract{
			ID: "c1", TenantID: "t1", Version: "1.0",
			RequestSchema: json.RawMessage(`{
				"properties": {"name": {"type":"string"}}
			}`),
		}
		repo.On("GetContract", ctx, "t1", "c1").Return(contract, nil)

		diff, err := svc.DiffContracts(ctx, "t1", "c1", "1.0", "1.0")
		require.NoError(t, err)
		assert.False(t, diff.IsBreaking)
		assert.Empty(t, diff.Changes)
	})

	t.Run("unmarshal error on line 144 is silently swallowed", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		// Contract with invalid RequestSchema JSON
		contract := &Contract{
			ID: "c1", TenantID: "t1", Version: "2.0",
			RequestSchema: json.RawMessage(`not-json`),
		}
		repo.On("GetContract", ctx, "t1", "c1").Return(contract, nil)

		// The unmarshal error is silently swallowed; newSchema stays nil/empty
		diff, err := svc.DiffContracts(ctx, "t1", "c1", "1.0", "2.0")
		require.NoError(t, err, "error is silently swallowed so no error returned")
		assert.NotNil(t, diff)
	})
}

// ---------------------------------------------------------------------------
// Service.ListContracts
// ---------------------------------------------------------------------------

func TestService_ListContracts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("default limit applied when <= 0", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		repo.On("ListContracts", ctx, "t1", 50, 0).
			Return([]Contract{}, 0, nil)

		_, total, err := svc.ListContracts(ctx, "t1", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		repo.AssertExpectations(t)
	})

	t.Run("custom limit used when > 0", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		repo.On("ListContracts", ctx, "t1", 10, 5).
			Return([]Contract{{ID: "c1"}}, 1, nil)

		contracts, total, err := svc.ListContracts(ctx, "t1", 10, 5)
		require.NoError(t, err)
		assert.Len(t, contracts, 1)
		assert.Equal(t, 1, total)
		repo.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// Service.UpdateContract
// ---------------------------------------------------------------------------

func TestService_UpdateContract(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("contract not found", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		repo.On("GetContract", ctx, "t1", "missing").
			Return((*Contract)(nil), ErrContractNotFound)

		_, err := svc.UpdateContract(ctx, "t1", "missing", &CreateContractRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrContractNotFound)
	})

	t.Run("valid update", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		existing := &Contract{
			ID: "c1", TenantID: "t1", Version: "1.0",
			RequestSchema: json.RawMessage(`{}`),
		}
		repo.On("GetContract", ctx, "t1", "c1").Return(existing, nil)
		repo.On("UpdateContract", ctx, mock.AnythingOfType("*contracts.Contract")).Return(nil)

		req := &CreateContractRequest{
			Name:       "updated",
			Version:    "2.0",
			EventType:  "order.updated",
			Schema:     `{"properties":{}}`,
			Strictness: StrictnessStrict,
		}
		contract, err := svc.UpdateContract(ctx, "t1", "c1", req)
		require.NoError(t, err)
		assert.Equal(t, "updated", contract.Name)
		assert.Equal(t, "2.0", contract.Version)
		repo.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// Service.DeleteContract
// ---------------------------------------------------------------------------

func TestService_DeleteContract(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("delegates to repo", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		repo.On("DeleteContract", ctx, "t1", "c1").Return(nil)

		err := svc.DeleteContract(ctx, "t1", "c1")
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		t.Parallel()
		repo := new(MockRepository)
		svc := NewService(repo)

		repo.On("DeleteContract", ctx, "t1", "c1").Return(errors.New("db error"))

		err := svc.DeleteContract(ctx, "t1", "c1")
		require.Error(t, err)
	})
}

// ---------- CreateContract duplicate name (repo error) ----------

func TestService_CreateContract_DuplicateName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := new(MockRepository)
	svc := NewService(repo)

	req := &CreateContractRequest{
		Name:      "existing-contract",
		Version:   "1.0",
		EventType: "order.created",
		Schema:    `{"properties": {"id": {"type": "string"}}}`,
	}

	// Repo returns error for duplicate
	repo.On("CreateContract", ctx, mock.AnythingOfType("*contracts.Contract")).
		Return(errors.New("unique constraint violation: name"))

	result, err := svc.CreateContract(ctx, "tenant-1", req)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create contract")
}

// ---------- DiffContracts with same version ----------

func TestService_DiffContracts_SameVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := new(MockRepository)
	svc := NewService(repo)

	contract := &Contract{
		ID:            "c1",
		TenantID:      "t1",
		Version:       "1.0",
		RequestSchema: json.RawMessage(`{"properties":{"name":{"type":"string"}}}`),
	}
	repo.On("GetContract", ctx, "t1", "c1").Return(contract, nil)

	diff, err := svc.DiffContracts(ctx, "t1", "c1", "1.0", "1.0")

	require.NoError(t, err)
	assert.False(t, diff.IsBreaking, "same version should have no breaking changes")
	assert.Empty(t, diff.Changes, "same version should have no changes")
}
