package dataplane

import "context"

// Repository defines the data access interface for data planes.
type Repository interface {
	CreatePlane(ctx context.Context, plane *DataPlane) error
	GetPlane(ctx context.Context, tenantID string) (*DataPlane, error)
	ListPlanes(ctx context.Context) ([]DataPlane, error)
	UpdatePlane(ctx context.Context, plane *DataPlane) error
	DeletePlane(ctx context.Context, tenantID string) error
}
