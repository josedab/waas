package pluginmarket

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service implements plugin marketplace business logic
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreatePlugin(ctx context.Context, authorID, authorName string, req *CreatePluginRequest) (*Plugin, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("plugin name is required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("plugin type is required")
	}
	if req.Version == "" {
		return nil, fmt.Errorf("plugin version is required")
	}

	plugin := &Plugin{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Description:  req.Description,
		LongDesc:     req.LongDesc,
		AuthorID:     authorID,
		AuthorName:   authorName,
		Type:         req.Type,
		Status:       PluginStatusDraft,
		Pricing:      req.Pricing,
		PriceMonthly: req.PriceMonthly,
		Version:      req.Version,
		IconURL:      req.IconURL,
		SourceURL:    req.SourceURL,
		DocURL:       req.DocURL,
		Tags:         req.Tags,
		Categories:   req.Categories,
	}

	if err := s.repo.CreatePlugin(ctx, plugin); err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	// Create initial version
	version := &PluginVersion{
		PluginID:    plugin.ID,
		Version:     req.Version,
		Changelog:   "Initial release",
		MinPlatform: "1.0.0",
		IsLatest:    true,
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create initial version: %w", err)
	}

	return plugin, nil
}

func (s *Service) GetPlugin(ctx context.Context, id string) (*Plugin, error) {
	return s.repo.GetPlugin(ctx, id)
}

func (s *Service) GetPluginBySlug(ctx context.Context, slug string) (*Plugin, error) {
	return s.repo.GetPluginBySlug(ctx, slug)
}

func (s *Service) SearchPlugins(ctx context.Context, params *PluginSearchParams) (*PluginSearchResult, error) {
	return s.repo.SearchPlugins(ctx, params)
}

func (s *Service) PublishPlugin(ctx context.Context, pluginID, authorID string) (*Plugin, error) {
	plugin, err := s.repo.GetPlugin(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	if plugin.AuthorID != authorID {
		return nil, fmt.Errorf("unauthorized: only the author can publish this plugin")
	}
	if plugin.Status != PluginStatusDraft && plugin.Status != PluginStatusReview {
		return nil, fmt.Errorf("plugin cannot be published from status: %s", plugin.Status)
	}

	now := time.Now()
	plugin.Status = PluginStatusPublished
	plugin.PublishedAt = &now

	if err := s.repo.UpdatePlugin(ctx, plugin); err != nil {
		return nil, fmt.Errorf("failed to publish plugin: %w", err)
	}
	return plugin, nil
}

func (s *Service) SubmitForReview(ctx context.Context, pluginID, authorID string) (*Plugin, error) {
	plugin, err := s.repo.GetPlugin(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	if plugin.AuthorID != authorID {
		return nil, fmt.Errorf("unauthorized: only the author can submit for review")
	}
	if plugin.Status != PluginStatusDraft {
		return nil, fmt.Errorf("only draft plugins can be submitted for review")
	}

	plugin.Status = PluginStatusReview
	if err := s.repo.UpdatePlugin(ctx, plugin); err != nil {
		return nil, fmt.Errorf("failed to submit for review: %w", err)
	}
	return plugin, nil
}

func (s *Service) InstallPlugin(ctx context.Context, tenantID string, req *InstallPluginRequest) (*PluginInstallation, error) {
	plugin, err := s.repo.GetPlugin(ctx, req.PluginID)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}
	if plugin.Status != PluginStatusPublished {
		return nil, fmt.Errorf("plugin is not available for installation")
	}

	// Check if already installed
	existing, err := s.repo.GetInstallation(ctx, tenantID, req.PluginID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("plugin already installed")
	}

	// Get version to install
	versionID := req.VersionID
	if versionID == "" {
		latest, err := s.repo.GetLatestVersion(ctx, req.PluginID)
		if err != nil {
			return nil, fmt.Errorf("no versions available: %w", err)
		}
		versionID = latest.ID
	}

	install := &PluginInstallation{
		TenantID:  tenantID,
		PluginID:  req.PluginID,
		VersionID: versionID,
		Config:    req.Config,
	}

	if err := s.repo.InstallPlugin(ctx, install); err != nil {
		return nil, fmt.Errorf("failed to install plugin: %w", err)
	}
	return install, nil
}

func (s *Service) UninstallPlugin(ctx context.Context, tenantID, pluginID string) error {
	existing, err := s.repo.GetInstallation(ctx, tenantID, pluginID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("plugin is not installed")
	}
	return s.repo.UninstallPlugin(ctx, tenantID, pluginID)
}

func (s *Service) ListInstallations(ctx context.Context, tenantID string) ([]PluginInstallation, error) {
	return s.repo.ListInstallations(ctx, tenantID)
}

func (s *Service) CreateReview(ctx context.Context, tenantID, pluginID string, req *CreateReviewRequest) (*PluginReview, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	// Verify plugin exists
	if _, err := s.repo.GetPlugin(ctx, pluginID); err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}

	// Verify tenant has installed the plugin
	install, err := s.repo.GetInstallation(ctx, tenantID, pluginID)
	if err != nil {
		return nil, err
	}
	if install == nil {
		return nil, fmt.Errorf("you must install the plugin before reviewing it")
	}

	review := &PluginReview{
		PluginID: pluginID,
		TenantID: tenantID,
		Rating:   req.Rating,
		Title:    req.Title,
		Body:     req.Body,
	}

	if err := s.repo.CreateReview(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}
	return review, nil
}

func (s *Service) GetReviews(ctx context.Context, pluginID string, page, pageSize int) ([]PluginReview, int, error) {
	return s.repo.GetReviews(ctx, pluginID, page, pageSize)
}

func (s *Service) GetVersions(ctx context.Context, pluginID string) ([]PluginVersion, error) {
	return s.repo.GetVersions(ctx, pluginID)
}

func (s *Service) CreateVersion(ctx context.Context, pluginID, authorID string, version, changelog string) (*PluginVersion, error) {
	plugin, err := s.repo.GetPlugin(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	if plugin.AuthorID != authorID {
		return nil, fmt.Errorf("unauthorized: only the author can create versions")
	}

	v := &PluginVersion{
		PluginID:    pluginID,
		Version:     version,
		Changelog:   changelog,
		MinPlatform: "1.0.0",
		IsLatest:    true,
	}

	if err := s.repo.CreateVersion(ctx, v); err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	// Update plugin version
	plugin.Version = version
	if err := s.repo.UpdatePlugin(ctx, plugin); err != nil {
		return nil, fmt.Errorf("failed to update plugin version: %w", err)
	}

	return v, nil
}

func (s *Service) GetMarketplaceStats(ctx context.Context) (*PluginStats, error) {
	return s.repo.GetMarketplaceStats(ctx)
}

func (s *Service) RegisterHook(ctx context.Context, pluginID string, hookPoint HookPoint, priority int) (*PluginHook, error) {
	if _, err := s.repo.GetPlugin(ctx, pluginID); err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}

	hook := &PluginHook{
		PluginID:  pluginID,
		HookPoint: hookPoint,
		Priority:  priority,
	}
	if err := s.repo.CreatePluginHook(ctx, hook); err != nil {
		return nil, fmt.Errorf("failed to register hook: %w", err)
	}
	return hook, nil
}

func (s *Service) GetPluginHooks(ctx context.Context, pluginID string) ([]PluginHook, error) {
	return s.repo.GetPluginHooks(ctx, pluginID)
}

// ExecuteHook runs a plugin at the specified hook point (sandbox execution stub)
func (s *Service) ExecuteHook(ctx context.Context, tenantID string, hookPoint HookPoint, payload map[string]any) ([]PluginExecResult, error) {
	installs, err := s.repo.ListInstallations(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var results []PluginExecResult
	for _, install := range installs {
		hooks, err := s.repo.GetPluginHooks(ctx, install.PluginID)
		if err != nil {
			continue
		}
		for _, hook := range hooks {
			if hook.HookPoint != hookPoint {
				continue
			}
			start := time.Now()
			// Plugin execution would happen here via sandbox
			result := PluginExecResult{
				PluginID:   install.PluginID,
				HookPoint:  hookPoint,
				Success:    true,
				DurationMs: time.Since(start).Milliseconds(),
				Output:     payload,
			}
			results = append(results, result)
		}
	}
	return results, nil
}
