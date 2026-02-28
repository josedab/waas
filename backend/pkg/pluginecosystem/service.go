package pluginecosystem

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrPluginNotFound       = errors.New("plugin not found")
	ErrInstallationNotFound = errors.New("plugin installation not found")
	ErrAlreadyInstalled     = errors.New("plugin already installed")
	ErrInvalidRating        = errors.New("rating must be between 1 and 5")
)

// ServiceConfig holds configuration for the plugin ecosystem.
type ServiceConfig struct {
	MaxPluginsPerDeveloper int
	MaxInstallsPerTenant   int
	RequireReview          bool
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxPluginsPerDeveloper: 50,
		MaxInstallsPerTenant:   100,
		RequireReview:          true,
	}
}

// Service provides plugin ecosystem operations.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
}

// NewService creates a new plugin ecosystem service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{repo: repo, config: config, logger: utils.NewLogger("pluginecosystem")}
}

// PublishPlugin creates and submits a new plugin for review.
func (s *Service) PublishPlugin(ctx context.Context, developerID string, req *PublishPluginRequest) (*Plugin, error) {
	if req.Name == "" || req.Description == "" {
		return nil, errors.New("name and description are required")
	}
	if req.Manifest == nil {
		return nil, errors.New("manifest is required")
	}

	status := PluginDraft
	if s.config.RequireReview {
		status = PluginInReview
	} else {
		status = PluginPublished
	}

	plugin := &Plugin{
		ID:            uuid.New().String(),
		DeveloperID:   developerID,
		Name:          req.Name,
		Slug:          slugify(req.Name),
		Description:   req.Description,
		Type:          req.Type,
		Status:        status,
		Version:       req.Version,
		Pricing:       req.Pricing,
		PriceAmtCents: req.PriceAmtCents,
		Tags:          req.Tags,
		Manifest:      req.Manifest,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if plugin.Pricing == "" {
		plugin.Pricing = PricingFree
	}

	if err := s.repo.CreatePlugin(ctx, plugin); err != nil {
		return nil, fmt.Errorf("create plugin: %w", err)
	}
	return plugin, nil
}

// GetPlugin retrieves a plugin by ID.
func (s *Service) GetPlugin(ctx context.Context, id string) (*Plugin, error) {
	return s.repo.GetPlugin(ctx, id)
}

// SearchPlugins searches the marketplace.
func (s *Service) SearchPlugins(ctx context.Context, req *SearchPluginsRequest) ([]Plugin, error) {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	return s.repo.ListPlugins(ctx, req)
}

// InstallPlugin installs a plugin for a tenant.
func (s *Service) InstallPlugin(ctx context.Context, tenantID string, req *InstallPluginRequest) (*PluginInstallation, error) {
	plugin, err := s.repo.GetPlugin(ctx, req.PluginID)
	if err != nil {
		return nil, err
	}
	if plugin.Status != PluginPublished {
		return nil, errors.New("plugin is not available for installation")
	}

	// Check not already installed
	if existing, _ := s.repo.GetInstallation(ctx, tenantID, req.PluginID); existing != nil {
		return nil, ErrAlreadyInstalled
	}

	inst := &PluginInstallation{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		PluginID:    req.PluginID,
		Version:     plugin.Version,
		Config:      req.Config,
		Enabled:     true,
		InstalledAt: time.Now().UTC(),
	}

	if err := s.repo.CreateInstallation(ctx, inst); err != nil {
		return nil, err
	}

	// Increment install count
	plugin.Installs++
	s.repo.UpdatePlugin(ctx, plugin)

	return inst, nil
}

// UninstallPlugin removes a plugin installation.
func (s *Service) UninstallPlugin(ctx context.Context, tenantID, pluginID string) error {
	return s.repo.DeleteInstallation(ctx, tenantID, pluginID)
}

// ListInstallations returns plugins installed by a tenant.
func (s *Service) ListInstallations(ctx context.Context, tenantID string) ([]PluginInstallation, error) {
	return s.repo.ListInstallations(ctx, tenantID)
}

// AddReview adds a review for a plugin.
func (s *Service) AddReview(ctx context.Context, tenantID, pluginID string, rating int, comment string) (*PluginReview, error) {
	if rating < 1 || rating > 5 {
		return nil, ErrInvalidRating
	}
	if _, err := s.repo.GetPlugin(ctx, pluginID); err != nil {
		return nil, err
	}

	review := &PluginReview{
		ID:        uuid.New().String(),
		PluginID:  pluginID,
		TenantID:  tenantID,
		Rating:    rating,
		Comment:   comment,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateReview(ctx, review); err != nil {
		return nil, err
	}
	return review, nil
}

// ListReviews returns reviews for a plugin.
func (s *Service) ListReviews(ctx context.Context, pluginID string, limit int) ([]PluginReview, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListReviews(ctx, pluginID, limit)
}

// ApprovePlugin approves a plugin for publication (admin operation).
func (s *Service) ApprovePlugin(ctx context.Context, pluginID string) error {
	plugin, err := s.repo.GetPlugin(ctx, pluginID)
	if err != nil {
		return err
	}
	plugin.Status = PluginPublished
	plugin.UpdatedAt = time.Now().UTC()
	return s.repo.UpdatePlugin(ctx, plugin)
}

func slugify(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	return slug
}
