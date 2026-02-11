package cdc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveConnector(ctx context.Context, conn *CDCConnector) error {
	args := m.Called(ctx, conn)
	return args.Error(0)
}

func (m *MockRepository) GetConnector(ctx context.Context, tenantID, connID string) (*CDCConnector, error) {
	args := m.Called(ctx, tenantID, connID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CDCConnector), args.Error(1)
}

func (m *MockRepository) ListConnectors(ctx context.Context, tenantID string, status *ConnectorStatus) ([]CDCConnector, error) {
	args := m.Called(ctx, tenantID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]CDCConnector), args.Error(1)
}

func (m *MockRepository) DeleteConnector(ctx context.Context, tenantID, connID string) error {
	args := m.Called(ctx, tenantID, connID)
	return args.Error(0)
}

func (m *MockRepository) UpdateConnectorStatus(ctx context.Context, tenantID, connID string, status ConnectorStatus, errMsg string) error {
	args := m.Called(ctx, tenantID, connID, status, errMsg)
	return args.Error(0)
}

func (m *MockRepository) SaveOffset(ctx context.Context, state *OffsetState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockRepository) GetOffset(ctx context.Context, tenantID, connID string) (*OffsetState, error) {
	args := m.Called(ctx, tenantID, connID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OffsetState), args.Error(1)
}

func (m *MockRepository) SaveEventHistory(ctx context.Context, history *EventHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockRepository) GetEventHistory(ctx context.Context, tenantID, connID string, limit, offset int) ([]EventHistory, error) {
	args := m.Called(ctx, tenantID, connID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]EventHistory), args.Error(1)
}

func (m *MockRepository) GetConnectorMetrics(ctx context.Context, tenantID, connID string) (*CDCMetrics, error) {
	args := m.Called(ctx, tenantID, connID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CDCMetrics), args.Error(1)
}

func (m *MockRepository) IncrementEventCount(ctx context.Context, tenantID, connID string, success bool) error {
	args := m.Called(ctx, tenantID, connID, success)
	return args.Error(0)
}

type MockSecretStore struct {
	mock.Mock
}

func (m *MockSecretStore) StoreConnectionCredentials(ctx context.Context, connID string, password string) error {
	args := m.Called(ctx, connID, password)
	return args.Error(0)
}

func (m *MockSecretStore) GetConnectionCredentials(ctx context.Context, connID string) (string, error) {
	args := m.Called(ctx, connID)
	return args.String(0), args.Error(1)
}

func (m *MockSecretStore) DeleteConnectionCredentials(ctx context.Context, connID string) error {
	args := m.Called(ctx, connID)
	return args.Error(0)
}

// --- Helpers ---

func newTestService(repo *MockRepository, secrets *MockSecretStore) *Service {
	return NewService(repo, secrets, DefaultServiceConfig())
}

func validCreateRequest() *CreateConnectorRequest {
	return &CreateConnectorRequest{
		Name: "test-connector",
		Type: ConnectorPostgreSQL,
		ConnectionConfig: ConnectionConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			Username: "user",
		},
		CaptureConfig: CaptureConfig{
			Tables: []TableConfig{{Schema: "public", Table: "users"}},
		},
		WebhookConfig: WebhookMapping{
			EndpointID:       "ep-1",
			EventTypePattern: "db.{schema}.{table}.{operation}",
		},
	}
}

// --- CreateConnector Tests ---

func TestCreateConnector_Valid(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("ListConnectors", ctx, "tenant-1", (*ConnectorStatus)(nil)).Return([]CDCConnector{}, nil)
	repo.On("SaveConnector", ctx, mock.AnythingOfType("*cdc.CDCConnector")).Return(nil)

	conn, err := svc.CreateConnector(ctx, "tenant-1", validCreateRequest())

	require.NoError(t, err)
	assert.NotEmpty(t, conn.ID)
	assert.Equal(t, "tenant-1", conn.TenantID)
	assert.Equal(t, StatusCreated, conn.Status)
	assert.Equal(t, []ChangeOperation{OpInsert, OpUpdate, OpDelete}, conn.CaptureConfig.Operations)
	assert.Equal(t, "never", conn.CaptureConfig.SnapshotMode)
	assert.Equal(t, "database", conn.OffsetConfig.StorageType)
	assert.Equal(t, 1000, conn.OffsetConfig.FlushIntervalMs)
	assert.True(t, conn.OffsetConfig.CommitOnSuccess)
	repo.AssertExpectations(t)
}

func TestCreateConnector_LimitReached(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	cfg := DefaultServiceConfig()
	cfg.MaxConnectorsPerTenant = 2
	svc := NewService(repo, secrets, cfg)
	ctx := context.Background()

	existing := []CDCConnector{{ID: "c1"}, {ID: "c2"}}
	repo.On("ListConnectors", ctx, "tenant-1", (*ConnectorStatus)(nil)).Return(existing, nil)

	_, err := svc.CreateConnector(ctx, "tenant-1", validCreateRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum connectors reached")
	repo.AssertExpectations(t)
}

func TestCreateConnector_WithPassword(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("ListConnectors", ctx, "tenant-1", (*ConnectorStatus)(nil)).Return([]CDCConnector{}, nil)
	secrets.On("StoreConnectionCredentials", ctx, mock.AnythingOfType("string"), "secret-pass").Return(nil)
	repo.On("SaveConnector", ctx, mock.AnythingOfType("*cdc.CDCConnector")).Return(nil)

	req := validCreateRequest()
	req.ConnectionConfig.Password = "secret-pass"

	conn, err := svc.CreateConnector(ctx, "tenant-1", req)

	require.NoError(t, err)
	assert.Empty(t, conn.ConnectionConfig.Password)
	secrets.AssertCalled(t, "StoreConnectionCredentials", ctx, conn.ID, "secret-pass")
	repo.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestCreateConnector_WithoutPassword(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("ListConnectors", ctx, "tenant-1", (*ConnectorStatus)(nil)).Return([]CDCConnector{}, nil)
	repo.On("SaveConnector", ctx, mock.AnythingOfType("*cdc.CDCConnector")).Return(nil)

	conn, err := svc.CreateConnector(ctx, "tenant-1", validCreateRequest())

	require.NoError(t, err)
	assert.Empty(t, conn.ConnectionConfig.Password)
	secrets.AssertNotCalled(t, "StoreConnectionCredentials", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateConnector_WithOffsetConfig(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("ListConnectors", ctx, "tenant-1", (*ConnectorStatus)(nil)).Return([]CDCConnector{}, nil)
	repo.On("SaveConnector", ctx, mock.AnythingOfType("*cdc.CDCConnector")).Return(nil)

	req := validCreateRequest()
	req.OffsetConfig = &OffsetConfig{
		StorageType:     "kafka",
		FlushIntervalMs: 5000,
		CommitOnSuccess: false,
	}

	conn, err := svc.CreateConnector(ctx, "tenant-1", req)

	require.NoError(t, err)
	assert.Equal(t, "kafka", conn.OffsetConfig.StorageType)
	assert.Equal(t, 5000, conn.OffsetConfig.FlushIntervalMs)
	assert.False(t, conn.OffsetConfig.CommitOnSuccess)
}

// --- GetConnector / ListConnectors Tests ---

func TestGetConnector_Delegates(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	expected := &CDCConnector{ID: "conn-1", TenantID: "tenant-1", Name: "my-conn"}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(expected, nil)

	conn, err := svc.GetConnector(ctx, "tenant-1", "conn-1")

	require.NoError(t, err)
	assert.Equal(t, expected, conn)
	repo.AssertExpectations(t)
}

func TestListConnectors_Delegates(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	expected := []CDCConnector{{ID: "c1"}, {ID: "c2"}}
	repo.On("ListConnectors", ctx, "tenant-1", (*ConnectorStatus)(nil)).Return(expected, nil)

	conns, err := svc.ListConnectors(ctx, "tenant-1", nil)

	require.NoError(t, err)
	assert.Len(t, conns, 2)
	repo.AssertExpectations(t)
}

// --- UpdateConnector Tests ---

func TestUpdateConnector_ValidOnStopped(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{
		ID:       "conn-1",
		TenantID: "tenant-1",
		Name:     "old-name",
		Status:   StatusStopped,
	}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)
	repo.On("SaveConnector", ctx, mock.AnythingOfType("*cdc.CDCConnector")).Return(nil)

	newName := "new-name"
	conn, err := svc.UpdateConnector(ctx, "tenant-1", "conn-1", &UpdateConnectorRequest{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "new-name", conn.Name)
	repo.AssertExpectations(t)
}

func TestUpdateConnector_CannotUpdateRunning(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{ID: "conn-1", TenantID: "tenant-1", Status: StatusRunning}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)

	_, err := svc.UpdateConnector(ctx, "tenant-1", "conn-1", &UpdateConnectorRequest{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update running connector")
}

func TestUpdateConnector_OnlyUpdatesProvidedFields(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{
		ID:       "conn-1",
		TenantID: "tenant-1",
		Name:     "original",
		Status:   StatusStopped,
		ConnectionConfig: ConnectionConfig{
			Host: "original-host",
			Port: 5432,
		},
		CaptureConfig: CaptureConfig{
			Tables: []TableConfig{{Table: "users"}},
		},
	}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)
	repo.On("SaveConnector", ctx, mock.AnythingOfType("*cdc.CDCConnector")).Return(nil)

	newCfg := &ConnectionConfig{Host: "new-host", Port: 3306, Database: "newdb", Username: "u"}
	conn, err := svc.UpdateConnector(ctx, "tenant-1", "conn-1", &UpdateConnectorRequest{
		ConnectionConfig: newCfg,
	})

	require.NoError(t, err)
	assert.Equal(t, "original", conn.Name)
	assert.Equal(t, "new-host", conn.ConnectionConfig.Host)
	assert.Equal(t, 3306, conn.ConnectionConfig.Port)
}

func TestUpdateConnector_PasswordStoredAndCleared(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{ID: "conn-1", TenantID: "tenant-1", Status: StatusStopped}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)
	secrets.On("StoreConnectionCredentials", ctx, "conn-1", "new-pass").Return(nil)
	repo.On("SaveConnector", ctx, mock.AnythingOfType("*cdc.CDCConnector")).Return(nil)

	newCfg := &ConnectionConfig{Host: "h", Port: 1, Database: "d", Username: "u", Password: "new-pass"}
	conn, err := svc.UpdateConnector(ctx, "tenant-1", "conn-1", &UpdateConnectorRequest{
		ConnectionConfig: newCfg,
	})

	require.NoError(t, err)
	assert.Empty(t, conn.ConnectionConfig.Password)
	secrets.AssertCalled(t, "StoreConnectionCredentials", ctx, "conn-1", "new-pass")
}

// --- DeleteConnector Tests ---

func TestDeleteConnector_Stopped(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{ID: "conn-1", TenantID: "tenant-1", Status: StatusStopped}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)
	secrets.On("DeleteConnectionCredentials", ctx, "conn-1").Return(nil)
	repo.On("DeleteConnector", ctx, "tenant-1", "conn-1").Return(nil)

	err := svc.DeleteConnector(ctx, "tenant-1", "conn-1")

	require.NoError(t, err)
	repo.AssertExpectations(t)
	secrets.AssertExpectations(t)
}

func TestDeleteConnector_RunningStopsFirst(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	runningConn := &CDCConnector{ID: "conn-1", TenantID: "tenant-1", Status: StatusRunning}
	// First call from DeleteConnector, second from StopConnector
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(runningConn, nil)
	repo.On("UpdateConnectorStatus", ctx, "tenant-1", "conn-1", StatusStopped, "").Return(nil)
	secrets.On("DeleteConnectionCredentials", ctx, "conn-1").Return(nil)
	repo.On("DeleteConnector", ctx, "tenant-1", "conn-1").Return(nil)

	err := svc.DeleteConnector(ctx, "tenant-1", "conn-1")

	require.NoError(t, err)
	repo.AssertCalled(t, "UpdateConnectorStatus", ctx, "tenant-1", "conn-1", StatusStopped, "")
	repo.AssertCalled(t, "DeleteConnector", ctx, "tenant-1", "conn-1")
	secrets.AssertCalled(t, "DeleteConnectionCredentials", ctx, "conn-1")
}

// --- StopConnector Tests ---

func TestStopConnector_NoRunner(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{ID: "conn-1", TenantID: "tenant-1", Status: StatusRunning}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)
	repo.On("UpdateConnectorStatus", ctx, "tenant-1", "conn-1", StatusStopped, "").Return(nil)

	conn, err := svc.StopConnector(ctx, "tenant-1", "conn-1")

	require.NoError(t, err)
	assert.Equal(t, StatusStopped, conn.Status)
	repo.AssertExpectations(t)
}

// --- PauseConnector Tests ---

func TestPauseConnector(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{ID: "conn-1", TenantID: "tenant-1", Status: StatusRunning}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)
	repo.On("UpdateConnectorStatus", ctx, "tenant-1", "conn-1", StatusPaused, "").Return(nil)

	conn, err := svc.PauseConnector(ctx, "tenant-1", "conn-1")

	require.NoError(t, err)
	assert.Equal(t, StatusPaused, conn.Status)
	repo.AssertExpectations(t)
}

// --- GetEventHistory Tests ---

func TestGetEventHistory_DefaultLimit(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("GetEventHistory", ctx, "tenant-1", "conn-1", 50, 0).Return([]EventHistory{}, nil)

	_, err := svc.GetEventHistory(ctx, "tenant-1", "conn-1", 0, 0)

	require.NoError(t, err)
	repo.AssertCalled(t, "GetEventHistory", ctx, "tenant-1", "conn-1", 50, 0)
}

func TestGetEventHistory_NegativeLimit(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("GetEventHistory", ctx, "tenant-1", "conn-1", 50, 0).Return([]EventHistory{}, nil)

	_, err := svc.GetEventHistory(ctx, "tenant-1", "conn-1", -10, 0)

	require.NoError(t, err)
	repo.AssertCalled(t, "GetEventHistory", ctx, "tenant-1", "conn-1", 50, 0)
}

func TestGetEventHistory_MaxLimit(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("GetEventHistory", ctx, "tenant-1", "conn-1", 500, 5).Return([]EventHistory{}, nil)

	_, err := svc.GetEventHistory(ctx, "tenant-1", "conn-1", 1000, 5)

	require.NoError(t, err)
	repo.AssertCalled(t, "GetEventHistory", ctx, "tenant-1", "conn-1", 500, 5)
}

func TestGetEventHistory_ValidLimit(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("GetEventHistory", ctx, "tenant-1", "conn-1", 100, 10).Return([]EventHistory{}, nil)

	_, err := svc.GetEventHistory(ctx, "tenant-1", "conn-1", 100, 10)

	require.NoError(t, err)
	repo.AssertCalled(t, "GetEventHistory", ctx, "tenant-1", "conn-1", 100, 10)
}

// --- GetHealth Tests ---

func TestGetHealth_WithoutRunner(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{
		ID:       "conn-1",
		TenantID: "tenant-1",
		Status:   StatusStopped,
	}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)

	health, err := svc.GetHealth(ctx, "tenant-1", "conn-1")

	require.NoError(t, err)
	assert.Equal(t, "conn-1", health.ConnectorID)
	assert.True(t, health.Healthy)
	assert.False(t, health.ConnectionOK)
}

func TestGetHealth_ErrorStatus(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{
		ID:       "conn-1",
		TenantID: "tenant-1",
		Status:   StatusError,
	}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)

	health, err := svc.GetHealth(ctx, "tenant-1", "conn-1")

	require.NoError(t, err)
	assert.False(t, health.Healthy)
}

func TestGetHealth_ConnectorNotFound(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(nil, fmt.Errorf("not found"))

	_, err := svc.GetHealth(ctx, "tenant-1", "conn-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- GetSupportedConnectors Tests ---

func TestGetSupportedConnectors(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)

	connectors := svc.GetSupportedConnectors()

	assert.Len(t, connectors, 4)

	types := make([]ConnectorType, len(connectors))
	for i, c := range connectors {
		types[i] = c.Type
	}
	assert.Contains(t, types, ConnectorPostgreSQL)
	assert.Contains(t, types, ConnectorMySQL)
	assert.Contains(t, types, ConnectorMongoDB)
	assert.Contains(t, types, ConnectorSQLServer)
}

// --- buildDSN Tests ---

func TestBuildDSN_PostgreSQL(t *testing.T) {
	cfg := &ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "mydb",
		SSLMode:  "require",
	}

	dsn := buildDSN(ConnectorPostgreSQL, cfg)

	assert.Equal(t, "host=localhost port=5432 user=user password=pass dbname=mydb sslmode=require", dsn)
}

func TestBuildDSN_PostgreSQL_DefaultSSL(t *testing.T) {
	cfg := &ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "mydb",
	}

	dsn := buildDSN(ConnectorPostgreSQL, cfg)

	assert.Contains(t, dsn, "sslmode=disable")
}

func TestBuildDSN_MySQL(t *testing.T) {
	cfg := &ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "user",
		Password: "pass",
		Database: "mydb",
	}

	dsn := buildDSN(ConnectorMySQL, cfg)

	assert.Equal(t, "user:pass@tcp(localhost:3306)/mydb", dsn)
}

func TestBuildDSN_Unsupported(t *testing.T) {
	cfg := &ConnectionConfig{Host: "h", Port: 1, Database: "d"}

	dsn := buildDSN(ConnectorType("unknown"), cfg)

	assert.Empty(t, dsn)
}

// --- getDriverName Tests ---

func TestGetDriverName(t *testing.T) {
	assert.Equal(t, "pgx", getDriverName(ConnectorPostgreSQL))
	assert.Equal(t, "mysql", getDriverName(ConnectorMySQL))
	assert.Empty(t, getDriverName(ConnectorMongoDB))
	assert.Empty(t, getDriverName(ConnectorSQLServer))
	assert.Empty(t, getDriverName(ConnectorType("unknown")))
}

// --- Additional Coverage Tests ---

func TestCreateConnector_CredentialStoreFailure(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("ListConnectors", ctx, "tenant-1", (*ConnectorStatus)(nil)).Return([]CDCConnector{}, nil)
	secrets.On("StoreConnectionCredentials", ctx, mock.AnythingOfType("string"), "secret-pass").Return(fmt.Errorf("vault unavailable"))

	req := validCreateRequest()
	req.ConnectionConfig.Password = "secret-pass"

	_, err := svc.CreateConnector(ctx, "tenant-1", req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store credentials")
	repo.AssertNotCalled(t, "SaveConnector", mock.Anything, mock.Anything)
}

func TestBuildDSN_EmptyHost(t *testing.T) {
	cfg := &ConnectionConfig{
		Host:     "",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "mydb",
	}

	dsn := buildDSN(ConnectorPostgreSQL, cfg)

	assert.Equal(t, "host= port=5432 user=user password=pass dbname=mydb sslmode=disable", dsn)
}

func TestBuildDSN_EmptyDatabase(t *testing.T) {
	cfg := &ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "",
	}

	dsn := buildDSN(ConnectorPostgreSQL, cfg)

	assert.Equal(t, "host=localhost port=5432 user=user password=pass dbname= sslmode=disable", dsn)
}

func TestBuildDSN_SpecialCharsInPassword(t *testing.T) {
	specialPass := "p@ss!@#$%^&*()'word"
	cfg := &ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: specialPass,
		Database: "mydb",
	}

	dsn := buildDSN(ConnectorPostgreSQL, cfg)

	assert.Contains(t, dsn, "password="+specialPass)

	dsnMySQL := buildDSN(ConnectorMySQL, cfg)

	assert.Contains(t, dsnMySQL, specialPass)
}

func TestStartConnector_AlreadyRunning(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	existing := &CDCConnector{ID: "conn-1", TenantID: "tenant-1", Status: StatusRunning}
	repo.On("GetConnector", ctx, "tenant-1", "conn-1").Return(existing, nil)

	conn, err := svc.StartConnector(ctx, "tenant-1", "conn-1")

	require.NoError(t, err)
	assert.Equal(t, StatusRunning, conn.Status)
	assert.Equal(t, "conn-1", conn.ID)
	secrets.AssertNotCalled(t, "GetConnectionCredentials", mock.Anything, mock.Anything)
}

func TestGetEventHistory_PaginationUpperBound(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("GetEventHistory", ctx, "tenant-1", "conn-1", 500, 0).Return([]EventHistory{}, nil)

	_, err := svc.GetEventHistory(ctx, "tenant-1", "conn-1", 1000, 0)

	require.NoError(t, err)
	repo.AssertCalled(t, "GetEventHistory", ctx, "tenant-1", "conn-1", 500, 0)
}

// ---------- Duplicate connector name via repo error ----------

func TestCreateConnector_DuplicateName(t *testing.T) {
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)
	ctx := context.Background()

	repo.On("ListConnectors", ctx, "tenant-1", (*ConnectorStatus)(nil)).Return([]CDCConnector{}, nil)
	secrets.On("StoreConnectionCredentials", ctx, mock.Anything, mock.Anything).Return(nil)
	repo.On("SaveConnector", ctx, mock.AnythingOfType("*cdc.CDCConnector")).
		Return(fmt.Errorf("unique constraint violation: name"))

	req := &CreateConnectorRequest{
		Name: "duplicate-name",
		Type: ConnectorPostgreSQL,
		ConnectionConfig: ConnectionConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			Username: "user",
			Password: "pass",
		},
		CaptureConfig: CaptureConfig{},
		WebhookConfig: WebhookMapping{},
	}

	result, err := svc.CreateConnector(ctx, "tenant-1", req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unique constraint violation")
}

// ---------- TestConnection requires real DB ----------
// TestConnection (service.go line 253) opens a real database connection via sql.Open.
// It cannot be unit-tested without a running database. This is documented as a
// limitation. Integration tests would be needed with a test database container.

// ---------- Concurrent start/stop safety (sync.Map) ----------

func TestService_StopConnector_MultipleCallsSafe(t *testing.T) {
	// Verifies multiple sequential StopConnector calls on same connector don't error.
	// The service uses sync.Map for connector storage which is inherently thread-safe.
	// Testify mocks are NOT thread-safe so concurrent calls cause mock races.
	repo := new(MockRepository)
	secrets := new(MockSecretStore)
	svc := newTestService(repo, secrets)

	connID := "conn-multi-stop"
	conn := &CDCConnector{
		ID:       connID,
		TenantID: "t1",
		Status:   StatusStopped,
		Type:     ConnectorPostgreSQL,
		ConnectionConfig: ConnectionConfig{
			Host: "localhost", Port: 5432, Database: "db", Username: "u",
		},
	}

	repo.On("GetConnector", mock.Anything, "t1", connID).Return(conn, nil)
	repo.On("UpdateConnectorStatus", mock.Anything, "t1", connID, StatusStopped, "").Return(nil)

	// Multiple sequential stop calls should all succeed
	for i := 0; i < 5; i++ {
		result, err := svc.StopConnector(context.Background(), "t1", connID)
		require.NoError(t, err)
		assert.Equal(t, StatusStopped, result.Status)
	}
}
