package eventlineage

import "context"

// Repository defines the data access interface for event lineage.
type Repository interface {
	RecordEntry(ctx context.Context, entry *LineageEntry) error
	GetEntry(ctx context.Context, tenantID, eventID string) (*LineageEntry, error)
	GetDescendants(ctx context.Context, tenantID, eventID string) ([]LineageEntry, error)
	GetAncestors(ctx context.Context, tenantID, eventID string) ([]LineageEntry, error)
	ListEntries(ctx context.Context, tenantID string, limit, offset int) ([]LineageEntry, error)
	GetStats(ctx context.Context, tenantID string) (*LineageStats, error)
}
