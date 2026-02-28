package pluginecosystem

import (
	"context"
	"strings"
)

// Repository defines data access operations for the plugin ecosystem.
type Repository interface {
	CreatePlugin(ctx context.Context, p *Plugin) error
	GetPlugin(ctx context.Context, id string) (*Plugin, error)
	ListPlugins(ctx context.Context, req *SearchPluginsRequest) ([]Plugin, error)
	UpdatePlugin(ctx context.Context, p *Plugin) error
	DeletePlugin(ctx context.Context, id string) error

	CreateInstallation(ctx context.Context, inst *PluginInstallation) error
	GetInstallation(ctx context.Context, tenantID, pluginID string) (*PluginInstallation, error)
	ListInstallations(ctx context.Context, tenantID string) ([]PluginInstallation, error)
	DeleteInstallation(ctx context.Context, tenantID, pluginID string) error

	CreateReview(ctx context.Context, review *PluginReview) error
	ListReviews(ctx context.Context, pluginID string, limit int) ([]PluginReview, error)
}

type memoryRepository struct {
	plugins       map[string]*Plugin
	installations map[string]*PluginInstallation
	reviews       map[string][]PluginReview
}

// NewMemoryRepository creates an in-memory plugin ecosystem repository.
func NewMemoryRepository() Repository {
	return &memoryRepository{
		plugins:       make(map[string]*Plugin),
		installations: make(map[string]*PluginInstallation),
		reviews:       make(map[string][]PluginReview),
	}
}

func (r *memoryRepository) CreatePlugin(_ context.Context, p *Plugin) error {
	r.plugins[p.ID] = p
	return nil
}

func (r *memoryRepository) GetPlugin(_ context.Context, id string) (*Plugin, error) {
	p, ok := r.plugins[id]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return p, nil
}

func (r *memoryRepository) ListPlugins(_ context.Context, req *SearchPluginsRequest) ([]Plugin, error) {
	var results []Plugin
	for _, p := range r.plugins {
		if p.Status != PluginPublished {
			continue
		}
		if req.Query != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(req.Query)) &&
			!strings.Contains(strings.ToLower(p.Description), strings.ToLower(req.Query)) {
			continue
		}
		if req.Type != "" && p.Type != req.Type {
			continue
		}
		if req.Pricing != "" && p.Pricing != req.Pricing {
			continue
		}
		results = append(results, *p)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := req.Offset
	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end], nil
}

func (r *memoryRepository) UpdatePlugin(_ context.Context, p *Plugin) error {
	if _, ok := r.plugins[p.ID]; !ok {
		return ErrPluginNotFound
	}
	r.plugins[p.ID] = p
	return nil
}

func (r *memoryRepository) DeletePlugin(_ context.Context, id string) error {
	delete(r.plugins, id)
	return nil
}

func instKey(tenantID, pluginID string) string { return tenantID + ":" + pluginID }

func (r *memoryRepository) CreateInstallation(_ context.Context, inst *PluginInstallation) error {
	r.installations[instKey(inst.TenantID, inst.PluginID)] = inst
	return nil
}

func (r *memoryRepository) GetInstallation(_ context.Context, tenantID, pluginID string) (*PluginInstallation, error) {
	inst, ok := r.installations[instKey(tenantID, pluginID)]
	if !ok {
		return nil, ErrInstallationNotFound
	}
	return inst, nil
}

func (r *memoryRepository) ListInstallations(_ context.Context, tenantID string) ([]PluginInstallation, error) {
	var results []PluginInstallation
	for _, inst := range r.installations {
		if inst.TenantID == tenantID {
			results = append(results, *inst)
		}
	}
	return results, nil
}

func (r *memoryRepository) DeleteInstallation(_ context.Context, tenantID, pluginID string) error {
	delete(r.installations, instKey(tenantID, pluginID))
	return nil
}

func (r *memoryRepository) CreateReview(_ context.Context, review *PluginReview) error {
	r.reviews[review.PluginID] = append(r.reviews[review.PluginID], *review)
	return nil
}

func (r *memoryRepository) ListReviews(_ context.Context, pluginID string, limit int) ([]PluginReview, error) {
	reviews := r.reviews[pluginID]
	if limit > 0 && limit < len(reviews) {
		reviews = reviews[:limit]
	}
	return reviews, nil
}
