package smartlimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// memoryRepository is an in-memory implementation of Repository for testing
type memoryRepository struct {
	mu            sync.RWMutex
	behaviors     map[string]*EndpointBehavior
	configs       map[string]*AdaptiveRateConfig
	states        map[string]*RateLimitState
	learningData  map[string][]LearningDataPoint
	models        map[string]*PredictionModel
	events        map[string][]RateLimitEvent
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		behaviors:    make(map[string]*EndpointBehavior),
		configs:      make(map[string]*AdaptiveRateConfig),
		states:       make(map[string]*RateLimitState),
		learningData: make(map[string][]LearningDataPoint),
		models:       make(map[string]*PredictionModel),
		events:       make(map[string][]RateLimitEvent),
	}
}

func key(tenantID, endpointID string) string { return tenantID + ":" + endpointID }

func (r *memoryRepository) SaveBehavior(ctx context.Context, b *EndpointBehavior) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.behaviors[key(b.TenantID, b.EndpointID)] = b
	return nil
}

func (r *memoryRepository) GetBehavior(ctx context.Context, tenantID, endpointID string) (*EndpointBehavior, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if b, ok := r.behaviors[key(tenantID, endpointID)]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("not found")
}

func (r *memoryRepository) GetRecentBehaviors(ctx context.Context, tenantID, endpointID string, limit int) ([]EndpointBehavior, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if b, ok := r.behaviors[key(tenantID, endpointID)]; ok {
		return []EndpointBehavior{*b}, nil
	}
	return nil, nil
}

func (r *memoryRepository) SaveConfig(ctx context.Context, c *AdaptiveRateConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[key(c.TenantID, c.EndpointID)] = c
	return nil
}

func (r *memoryRepository) GetConfig(ctx context.Context, tenantID, endpointID string) (*AdaptiveRateConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.configs[key(tenantID, endpointID)]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("not found")
}

func (r *memoryRepository) ListConfigs(ctx context.Context, tenantID string) ([]AdaptiveRateConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []AdaptiveRateConfig
	for _, c := range r.configs {
		if c.TenantID == tenantID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (r *memoryRepository) DeleteConfig(ctx context.Context, tenantID, endpointID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.configs, key(tenantID, endpointID))
	return nil
}

func (r *memoryRepository) SaveState(ctx context.Context, s *RateLimitState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states[key(s.TenantID, s.EndpointID)] = s
	return nil
}

func (r *memoryRepository) GetState(ctx context.Context, tenantID, endpointID string) (*RateLimitState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.states[key(tenantID, endpointID)]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("not found")
}

func (r *memoryRepository) SaveLearningData(ctx context.Context, data *LearningDataPoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(data.TenantID, data.EndpointID)
	r.learningData[k] = append(r.learningData[k], *data)
	return nil
}

func (r *memoryRepository) GetLearningData(ctx context.Context, tenantID, endpointID string, start, end time.Time) ([]LearningDataPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []LearningDataPoint
	for _, d := range r.learningData[key(tenantID, endpointID)] {
		if !d.Timestamp.Before(start) && !d.Timestamp.After(end) {
			result = append(result, d)
		}
	}
	return result, nil
}

func (r *memoryRepository) SaveModel(ctx context.Context, m *PredictionModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[key(m.TenantID, m.EndpointID)] = m
	return nil
}

func (r *memoryRepository) GetActiveModel(ctx context.Context, tenantID, endpointID string) (*PredictionModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if m, ok := r.models[key(tenantID, endpointID)]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("not found")
}

func (r *memoryRepository) SaveEvent(ctx context.Context, e *RateLimitEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(e.TenantID, e.EndpointID)
	r.events[k] = append(r.events[k], *e)
	return nil
}

func (r *memoryRepository) GetEvents(ctx context.Context, tenantID, endpointID string, start, end time.Time) ([]RateLimitEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []RateLimitEvent
	for _, e := range r.events[key(tenantID, endpointID)] {
		if !e.Timestamp.Before(start) && !e.Timestamp.After(end) {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *memoryRepository) GetStats(ctx context.Context, tenantID string, start, end time.Time) (*SmartLimitStats, error) {
	return &SmartLimitStats{TenantID: tenantID, GeneratedAt: time.Now()}, nil
}
