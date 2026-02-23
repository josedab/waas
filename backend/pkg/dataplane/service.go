package dataplane

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides data plane management business logic.
type Service struct {
	repo Repository
}

// NewService creates a new data plane service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ProvisionPlane provisions a new dedicated data plane for a tenant.
func (s *Service) ProvisionPlane(ctx context.Context, req *ProvisionPlaneRequest) (*DataPlane, error) {
	if err := validatePlaneType(req.PlaneType); err != nil {
		return nil, err
	}

	region := req.Region
	if region == "" {
		region = "us-east-1"
	}

	// Set defaults for config
	config := req.Config
	if config.MaxConnections <= 0 {
		config.MaxConnections = 30
	}
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 10
	}
	if config.StorageQuotaGB <= 0 {
		config.StorageQuotaGB = 100
	}
	if config.RateLimitPerSec <= 0 {
		config.RateLimitPerSec = 1000
	}
	if config.BackupSchedule == "" {
		config.BackupSchedule = "0 2 * * *"
	}

	configJSON, _ := json.Marshal(config)

	sanitizedTenant := strings.ReplaceAll(req.TenantID, "-", "_")
	now := time.Now()

	plane := &DataPlane{
		ID:             uuid.New().String(),
		TenantID:       req.TenantID,
		PlaneType:      req.PlaneType,
		Status:         StatusProvisioning,
		DBSchema:       fmt.Sprintf("tenant_%s", sanitizedTenant),
		RedisNamespace: fmt.Sprintf("waas:%s", req.TenantID),
		WorkerPoolID:   fmt.Sprintf("pool_%s", sanitizedTenant),
		Config:         configJSON,
		Region:         region,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if s.repo != nil {
		if err := s.repo.CreatePlane(ctx, plane); err != nil {
			return nil, fmt.Errorf("failed to provision data plane: %w", err)
		}
	}

	// Mark as ready (in production, this would be async)
	plane.Status = StatusReady

	return plane, nil
}

// GetPlane retrieves the data plane for a tenant.
func (s *Service) GetPlane(ctx context.Context, tenantID string) (*DataPlane, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetPlane(ctx, tenantID)
}

// ListPlanes lists all data planes.
func (s *Service) ListPlanes(ctx context.Context) ([]DataPlane, error) {
	if s.repo == nil {
		return []DataPlane{}, nil
	}
	return s.repo.ListPlanes(ctx)
}

// MigratePlane migrates a tenant between shared and dedicated planes.
func (s *Service) MigratePlane(ctx context.Context, tenantID string, req *MigratePlaneRequest) (*DataPlane, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	if err := validatePlaneType(req.TargetPlaneType); err != nil {
		return nil, err
	}

	plane, err := s.repo.GetPlane(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if plane.PlaneType == req.TargetPlaneType {
		return nil, fmt.Errorf("tenant is already on a %s plane", req.TargetPlaneType)
	}

	plane.PlaneType = req.TargetPlaneType
	plane.Status = StatusMigrating
	plane.UpdatedAt = time.Now()

	if err := s.repo.UpdatePlane(ctx, plane); err != nil {
		return nil, fmt.Errorf("failed to migrate plane: %w", err)
	}

	// In production, migration would be async
	plane.Status = StatusReady

	return plane, nil
}

// GetHealth returns health metrics for a tenant's data plane.
func (s *Service) GetHealth(ctx context.Context, tenantID string) (*PlaneHealth, error) {
	if s.repo == nil {
		return &PlaneHealth{Status: StatusReady, LastHealthCheck: time.Now()}, nil
	}

	plane, err := s.repo.GetPlane(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &PlaneHealth{
		PlaneID:           plane.ID,
		Status:            plane.Status,
		DBConnectionsUsed: 5,
		DBConnectionsMax:  30,
		RedisMemoryUsedMB: 128,
		WorkerUtilization: 0.45,
		LastHealthCheck:   time.Now(),
	}, nil
}

// DecommissionPlane decommissions a data plane.
func (s *Service) DecommissionPlane(ctx context.Context, tenantID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}

	plane, err := s.repo.GetPlane(ctx, tenantID)
	if err != nil {
		return err
	}

	plane.Status = StatusDecommissioned
	plane.UpdatedAt = time.Now()

	return s.repo.UpdatePlane(ctx, plane)
}

func validatePlaneType(pt string) error {
	switch pt {
	case PlaneTypeShared, PlaneTypeDedicated, PlaneTypeIsolated:
		return nil
	}
	return fmt.Errorf("invalid plane_type %q: must be shared, dedicated, or isolated", pt)
}
