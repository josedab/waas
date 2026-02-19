package georouting

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/josedab/waas/pkg/utils"
)

// HealthTracker monitors region health
type HealthTracker struct {
	repo       Repository
	httpClient *http.Client
	mu         sync.RWMutex
	health     map[string]*RegionHealth
	stopCh     chan struct{}
	logger     *utils.Logger
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
		logger: utils.NewLogger("georouting"),
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
		h.logger.Error("failed to list regions", map[string]interface{}{"error": err.Error()})
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

// CheckRegionHealth returns detailed health for a specific region
func (h *HealthTracker) CheckRegionHealth(ctx context.Context, regionName string) (*GeoRegionHealth, error) {
	h.mu.RLock()
	rh, ok := h.health[regionName]
	h.mu.RUnlock()

	result := &GeoRegionHealth{
		RegionName:  regionName,
		LastChecked: time.Now(),
	}

	if ok && rh != nil {
		result.AvgLatency = rh.AvgLatencyMs
		result.LastChecked = rh.LastCheck
		if rh.IsHealthy {
			result.Status = "active"
			result.SuccessRate = 100.0 - rh.ErrorRate
		} else {
			result.Status = "offline"
			result.SuccessRate = 0
		}
	} else {
		// Try from repo
		dbHealth, err := h.repo.GetRegionHealth(ctx, regionName)
		if err != nil || dbHealth == nil {
			result.Status = "unknown"
			return result, nil
		}
		result.AvgLatency = dbHealth.AvgLatencyMs
		result.LastChecked = dbHealth.LastCheck
		if dbHealth.IsHealthy {
			result.Status = "active"
			result.SuccessRate = 100.0 - dbHealth.ErrorRate
		} else {
			result.Status = "offline"
		}
	}

	return result, nil
}

// GetAllRegionHealthStatus returns health status for all tracked regions
func (h *HealthTracker) GetAllRegionHealthStatus(ctx context.Context) ([]GeoRegionHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []GeoRegionHealth
	for name, rh := range h.health {
		grh := GeoRegionHealth{
			RegionName:  name,
			AvgLatency:  rh.AvgLatencyMs,
			LastChecked: rh.LastCheck,
		}
		if rh.IsHealthy {
			grh.Status = "active"
			grh.SuccessRate = 100.0 - rh.ErrorRate
		} else {
			grh.Status = "offline"
			grh.SuccessRate = 0
		}
		result = append(result, grh)
	}
	return result, nil
}

// UpdateRegionMetrics updates latency and success metrics for a region
func (h *HealthTracker) UpdateRegionMetrics(ctx context.Context, regionName string, latency int, success bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	rh, ok := h.health[regionName]
	if !ok {
		rh = &RegionHealth{
			RegionID:  regionName,
			IsHealthy: true,
		}
		h.health[regionName] = rh
	}

	rh.LastCheck = time.Now()
	rh.AvgLatencyMs = (rh.AvgLatencyMs + latency) / 2

	if success {
		rh.ConsecutiveOK++
		rh.ConsecutiveFail = 0
		if rh.ConsecutiveOK >= 2 {
			rh.IsHealthy = true
		}
	} else {
		rh.ConsecutiveFail++
		rh.ConsecutiveOK = 0
		if rh.ConsecutiveFail >= 3 {
			rh.IsHealthy = false
		}
	}

	return nil
}

// DetectDegradedRegions returns health info for regions that are degraded or offline
func (h *HealthTracker) DetectDegradedRegions(ctx context.Context) ([]GeoRegionHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var degraded []GeoRegionHealth
	for name, rh := range h.health {
		if !rh.IsHealthy || rh.ConsecutiveFail > 0 {
			status := "degraded"
			if !rh.IsHealthy {
				status = "offline"
			}
			degraded = append(degraded, GeoRegionHealth{
				RegionName:  name,
				Status:      status,
				AvgLatency:  rh.AvgLatencyMs,
				SuccessRate: 100.0 - rh.ErrorRate,
				LastChecked: rh.LastCheck,
			})
		}
	}
	return degraded, nil
}
