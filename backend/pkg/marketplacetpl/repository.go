package marketplacetpl

import "context"

// Repository defines the data access interface for the marketplace
type Repository interface {
	// Templates
	CreateTemplate(ctx context.Context, template *Template) error
	GetTemplate(ctx context.Context, templateID string) (*Template, error)
	ListTemplates(ctx context.Context, category string, limit, offset int) ([]Template, int, error)
	SearchTemplates(ctx context.Context, query string, limit, offset int) ([]Template, int, error)
	IncrementInstallCount(ctx context.Context, templateID string) error
	UpdateTemplateRating(ctx context.Context, templateID string, avgRating float64) error

	// Installations
	CreateInstallation(ctx context.Context, install *Installation) error
	GetInstallation(ctx context.Context, tenantID, installID string) (*Installation, error)
	ListInstallations(ctx context.Context, tenantID string) ([]Installation, error)
	DeleteInstallation(ctx context.Context, tenantID, installID string) error

	// Reviews
	CreateReview(ctx context.Context, review *Review) error
	ListReviews(ctx context.Context, templateID string, limit, offset int) ([]Review, error)
	GetAverageRating(ctx context.Context, templateID string) (float64, int, error)

	// Stats
	GetStats(ctx context.Context) (*MarketplaceStats, error)
}
