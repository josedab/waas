package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock FederatedMeshRepository ---
type MockFederatedMeshRepo struct {
	mock.Mock
}

func (m *MockFederatedMeshRepo) GetAllRegions(ctx context.Context) ([]*models.Region, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Region), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetRegion(ctx context.Context, id uuid.UUID) (*models.Region, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Region), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetRegionByCode(ctx context.Context, code string) (*models.Region, error) {
	args := m.Called(ctx, code)
	return args.Get(0).(*models.Region), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetActiveRegions(ctx context.Context) ([]*models.Region, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Region), args.Error(1)
}
func (m *MockFederatedMeshRepo) UpdateRegionHealth(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockFederatedMeshRepo) UpdateRegionLoad(ctx context.Context, id uuid.UUID, load int) error {
	return m.Called(ctx, id, load).Error(0)
}
func (m *MockFederatedMeshRepo) CreateCluster(ctx context.Context, cluster *models.RegionCluster) error {
	return m.Called(ctx, cluster).Error(0)
}
func (m *MockFederatedMeshRepo) GetCluster(ctx context.Context, id uuid.UUID) (*models.RegionCluster, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.RegionCluster), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetClusters(ctx context.Context) ([]*models.RegionCluster, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.RegionCluster), args.Error(1)
}
func (m *MockFederatedMeshRepo) AddClusterMember(ctx context.Context, member *models.RegionClusterMember) error {
	return m.Called(ctx, member).Error(0)
}
func (m *MockFederatedMeshRepo) GetClusterMembers(ctx context.Context, clusterID uuid.UUID) ([]*models.RegionClusterMember, error) {
	args := m.Called(ctx, clusterID)
	return args.Get(0).([]*models.RegionClusterMember), args.Error(1)
}
func (m *MockFederatedMeshRepo) CreateTenantRegion(ctx context.Context, tr *models.MeshTenantRegion) error {
	return m.Called(ctx, tr).Error(0)
}
func (m *MockFederatedMeshRepo) GetTenantRegion(ctx context.Context, tenantID uuid.UUID) (*models.MeshTenantRegion, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MeshTenantRegion), args.Error(1)
}
func (m *MockFederatedMeshRepo) UpdateTenantRegion(ctx context.Context, tr *models.MeshTenantRegion) error {
	return m.Called(ctx, tr).Error(0)
}
func (m *MockFederatedMeshRepo) CreateRoutingRule(ctx context.Context, rule *models.GeoRoutingRule) error {
	return m.Called(ctx, rule).Error(0)
}
func (m *MockFederatedMeshRepo) GetRoutingRule(ctx context.Context, id uuid.UUID) (*models.GeoRoutingRule, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.GeoRoutingRule), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetRoutingRulesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.GeoRoutingRule, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.GeoRoutingRule), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetEnabledRoutingRules(ctx context.Context, tenantID uuid.UUID) ([]*models.GeoRoutingRule, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.GeoRoutingRule), args.Error(1)
}
func (m *MockFederatedMeshRepo) UpdateRoutingRule(ctx context.Context, rule *models.GeoRoutingRule) error {
	return m.Called(ctx, rule).Error(0)
}
func (m *MockFederatedMeshRepo) DeleteRoutingRule(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockFederatedMeshRepo) CreateReplicationStream(ctx context.Context, stream *models.ReplicationStream) error {
	return m.Called(ctx, stream).Error(0)
}
func (m *MockFederatedMeshRepo) GetReplicationStream(ctx context.Context, id uuid.UUID) (*models.ReplicationStream, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.ReplicationStream), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetReplicationStreamsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ReplicationStream, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.ReplicationStream), args.Error(1)
}
func (m *MockFederatedMeshRepo) UpdateReplicationLag(ctx context.Context, id uuid.UUID, lagMs int64, lastEventID *uuid.UUID) error {
	return m.Called(ctx, id, lagMs, lastEventID).Error(0)
}
func (m *MockFederatedMeshRepo) SetReplicationError(ctx context.Context, id uuid.UUID, errorMsg string) error {
	return m.Called(ctx, id, errorMsg).Error(0)
}
func (m *MockFederatedMeshRepo) CreateRoutingDecision(ctx context.Context, decision *models.RegionalRoutingDecision) error {
	return m.Called(ctx, decision).Error(0)
}
func (m *MockFederatedMeshRepo) GetRoutingDecisions(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.RegionalRoutingDecision, error) {
	args := m.Called(ctx, tenantID, limit)
	return args.Get(0).([]*models.RegionalRoutingDecision), args.Error(1)
}
func (m *MockFederatedMeshRepo) CreateHealthMetric(ctx context.Context, metric *models.RegionHealthMetric) error {
	return m.Called(ctx, metric).Error(0)
}
func (m *MockFederatedMeshRepo) GetLatestHealthMetrics(ctx context.Context, regionID uuid.UUID) ([]*models.RegionHealthMetric, error) {
	args := m.Called(ctx, regionID)
	return args.Get(0).([]*models.RegionHealthMetric), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetHealthMetricHistory(ctx context.Context, regionID uuid.UUID, metricType string, since time.Time) ([]*models.RegionHealthMetric, error) {
	args := m.Called(ctx, regionID, metricType, since)
	return args.Get(0).([]*models.RegionHealthMetric), args.Error(1)
}
func (m *MockFederatedMeshRepo) CreateResidencyAudit(ctx context.Context, audit *models.DataResidencyAudit) error {
	return m.Called(ctx, audit).Error(0)
}
func (m *MockFederatedMeshRepo) GetResidencyAudits(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.DataResidencyAudit, error) {
	args := m.Called(ctx, tenantID, limit)
	return args.Get(0).([]*models.DataResidencyAudit), args.Error(1)
}
func (m *MockFederatedMeshRepo) CreateFailoverEvent(ctx context.Context, event *models.FailoverEvent) error {
	return m.Called(ctx, event).Error(0)
}
func (m *MockFederatedMeshRepo) GetFailoverEvent(ctx context.Context, id uuid.UUID) (*models.FailoverEvent, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.FailoverEvent), args.Error(1)
}
func (m *MockFederatedMeshRepo) GetRecentFailovers(ctx context.Context, limit int) ([]*models.FailoverEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*models.FailoverEvent), args.Error(1)
}
func (m *MockFederatedMeshRepo) CompleteFailover(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockFederatedMeshRepo) CreateConfigSync(ctx context.Context, sync *models.RegionalConfigSync) error {
	return m.Called(ctx, sync).Error(0)
}
func (m *MockFederatedMeshRepo) GetConfigSync(ctx context.Context, configType string, configID, regionID uuid.UUID) (*models.RegionalConfigSync, error) {
	args := m.Called(ctx, configType, configID, regionID)
	return args.Get(0).(*models.RegionalConfigSync), args.Error(1)
}
func (m *MockFederatedMeshRepo) UpdateConfigSyncStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockFederatedMeshRepo) GetPendingSyncs(ctx context.Context, tenantID uuid.UUID) ([]*models.RegionalConfigSync, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.RegionalConfigSync), args.Error(1)
}

// --- Federated Mesh Service Tests ---

func TestFederatedMeshService_SetupTenantRegion_Valid(t *testing.T) {
	t.Parallel()
	repo := &MockFederatedMeshRepo{}
	logger := utils.NewLogger("test")
	svc := NewFederatedMeshService(repo, logger)

	tenantID := uuid.New()
	primaryRegionID := uuid.New()

	repo.On("GetRegion", mock.Anything, primaryRegionID).Return(&models.Region{
		ID:           primaryRegionID,
		Code:         "us-east-1",
		Status:       models.RegionStatusActive,
		HealthStatus: models.RegionHealthHealthy,
	}, nil)
	repo.On("GetTenantRegion", mock.Anything, tenantID).Return(nil, fmt.Errorf("not found"))
	repo.On("CreateTenantRegion", mock.Anything, mock.AnythingOfType("*models.MeshTenantRegion")).Return(nil)

	tr, err := svc.SetupTenantRegion(context.Background(), tenantID, &models.CreateTenantRegionRequest{
		PrimaryRegionID:     primaryRegionID.String(),
		DataResidencyPolicy: models.DataResidencyFlexible,
		ReplicationMode:     models.ReplicationModeAsync,
	})
	require.NoError(t, err)
	assert.Equal(t, tenantID, tr.TenantID)
	assert.Equal(t, primaryRegionID, tr.PrimaryRegionID)
	assert.Equal(t, models.DataResidencyFlexible, tr.DataResidencyPolicy)
}

func TestFederatedMeshService_SetupTenantRegion_InvalidPrimaryRegion(t *testing.T) {
	t.Parallel()
	repo := &MockFederatedMeshRepo{}
	logger := utils.NewLogger("test")
	svc := NewFederatedMeshService(repo, logger)

	regionID := uuid.New()
	repo.On("GetRegion", mock.Anything, regionID).Return((*models.Region)(nil), fmt.Errorf("not found"))

	_, err := svc.SetupTenantRegion(context.Background(), uuid.New(), &models.CreateTenantRegionRequest{
		PrimaryRegionID: regionID.String(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "primary region not found")
}

func TestFederatedMeshService_CreateRoutingRule_ValidLatency(t *testing.T) {
	t.Parallel()
	repo := &MockFederatedMeshRepo{}
	logger := utils.NewLogger("test")
	svc := NewFederatedMeshService(repo, logger)

	tenantID := uuid.New()
	targetRegionID := uuid.New()

	repo.On("CreateRoutingRule", mock.Anything, mock.AnythingOfType("*models.GeoRoutingRule")).Return(nil)

	rule, err := svc.CreateRoutingRule(context.Background(), tenantID, &models.CreateGeoRoutingRuleRequest{
		Name:           "latency-rule",
		RuleType:       models.GeoRuleLatency,
		Priority:       1,
		TargetRegionID: targetRegionID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, tenantID, rule.TenantID)
	assert.Equal(t, models.GeoRuleLatency, rule.RuleType)
	assert.True(t, rule.Enabled)
}

func TestFederatedMeshService_CreateRoutingRule_InvalidRuleType(t *testing.T) {
	t.Parallel()
	repo := &MockFederatedMeshRepo{}
	logger := utils.NewLogger("test")
	svc := NewFederatedMeshService(repo, logger)

	_, err := svc.CreateRoutingRule(context.Background(), uuid.New(), &models.CreateGeoRoutingRuleRequest{
		Name:     "bad-rule",
		RuleType: "invalid_type",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid rule_type")
}

func TestFederatedMeshService_RouteEvent_NoTenantConfig(t *testing.T) {
	t.Parallel()
	repo := &MockFederatedMeshRepo{}
	logger := utils.NewLogger("test")
	svc := NewFederatedMeshService(repo, logger)

	tenantID := uuid.New()
	eventID := uuid.New()
	sourceRegionID := uuid.New()

	repo.On("GetTenantRegion", mock.Anything, tenantID).Return(nil, fmt.Errorf("not found"))
	repo.On("CreateRoutingDecision", mock.Anything, mock.AnythingOfType("*models.RegionalRoutingDecision")).Return(nil).Maybe()

	decision, err := svc.RouteEvent(context.Background(), tenantID, &models.RouteEventRequest{
		EventID:        eventID.String(),
		SourceRegionID: sourceRegionID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, sourceRegionID, decision.SourceRegionID)
	assert.Equal(t, sourceRegionID, decision.TargetRegionID)
	assert.Equal(t, "no_tenant_region_config", decision.DecisionReason)
}

func TestFederatedMeshService_RecordHealthMetric(t *testing.T) {
	t.Parallel()
	repo := &MockFederatedMeshRepo{}
	logger := utils.NewLogger("test")
	svc := NewFederatedMeshService(repo, logger)

	regionID := uuid.New()
	repo.On("CreateHealthMetric", mock.Anything, mock.AnythingOfType("*models.RegionHealthMetric")).Return(nil)

	err := svc.RecordHealthMetric(context.Background(), regionID, "latency", 42.5)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
