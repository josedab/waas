package georouting

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// HealthTracker monitors region health
type HealthTracker struct {
	repo       Repository
	httpClient *http.Client
	mu         sync.RWMutex
	health     map[string]*RegionHealth
	stopCh     chan struct{}
}

// NewHealthTracker creates a new health tracker
func NewHealthTracker(repo Repository) *HealthTracker {
	return &HealthTracker{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		health: make(map[string]*RegionHealth),
		stopCh: make(chan struct{}),
	}
}

// Start begins health monitoring
func (h *HealthTracker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial check
	h.checkAllRegions(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.checkAllRegions(ctx)
		}
	}
}

// Stop stops health monitoring
func (h *HealthTracker) Stop() {
	close(h.stopCh)
}

// GetHealth returns health status for a region
func (h *HealthTracker) GetHealth(ctx context.Context, regionID string) (*RegionHealth, error) {
	h.mu.RLock()
	if health, ok := h.health[regionID]; ok {
		h.mu.RUnlock()
		return health, nil
	}
	h.mu.RUnlock()

	// Try to get from database
	return h.repo.GetRegionHealth(ctx, regionID)
}

// checkAllRegions checks health of all configured regions
func (h *HealthTracker) checkAllRegions(ctx context.Context) {
	regions, err := h.repo.ListRegionConfigs(ctx)
	if err != nil {
		log.Printf("[georouting] Failed to list regions: %v", err)
		return
	}

	for _, region := range regions {
		if !region.IsActive {
			continue
		}
		go h.checkRegion(ctx, &region)
	}
}

// checkRegion performs a health check on a region
func (h *HealthTracker) checkRegion(ctx context.Context, region *RegionConfig) {
	start := time.Now()

	healthEndpoint := region.Endpoint + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", healthEndpoint, nil)
	if err != nil {
		h.recordFailure(ctx, region.ID, err.Error())
		return
	}

	resp, err := h.httpClient.Do(req)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		h.recordFailure(ctx, region.ID, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		h.recordSuccess(ctx, region.ID, latency)
	} else {
		h.recordFailure(ctx, region.ID, resp.Status)
	}
}

// recordSuccess records a successful health check
func (h *HealthTracker) recordSuccess(ctx context.Context, regionID string, latencyMs int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	health, ok := h.health[regionID]
	if !ok {
		health = &RegionHealth{
			RegionID:  regionID,
			IsHealthy: true,
		}
		h.health[regionID] = health
	}

	health.LastCheck = time.Now()
	health.ConsecutiveOK++
	health.ConsecutiveFail = 0
	health.AvgLatencyMs = (health.AvgLatencyMs + latencyMs) / 2

	// Mark healthy after threshold
	if health.ConsecutiveOK >= 2 {
		health.IsHealthy = true
	}

	// Save to database
	h.repo.UpdateRegionHealth(ctx, health)
}

// recordFailure records a failed health check
func (h *HealthTracker) recordFailure(ctx context.Context, regionID, errMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	health, ok := h.health[regionID]
	if !ok {
		health = &RegionHealth{
			RegionID:  regionID,
			IsHealthy: true,
		}
		h.health[regionID] = health
	}

	now := time.Now()
	health.LastCheck = now
	health.ConsecutiveFail++
	health.ConsecutiveOK = 0
	health.LastError = errMsg
	health.LastErrorAt = &now

	// Mark unhealthy after threshold
	if health.ConsecutiveFail >= 3 {
		health.IsHealthy = false
	}

	// Save to database
	h.repo.UpdateRegionHealth(ctx, health)
}

// GetAllHealth returns health status for all regions
func (h *HealthTracker) GetAllHealth() map[string]*RegionHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*RegionHealth, len(h.health))
	for k, v := range h.health {
		health := *v
		result[k] = &health
	}
	return result
}

// IsHealthy returns whether a region is currently healthy
func (h *HealthTracker) IsHealthy(regionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if health, ok := h.health[regionID]; ok {
		return health.IsHealthy
	}
	return true // Assume healthy if no data
}
