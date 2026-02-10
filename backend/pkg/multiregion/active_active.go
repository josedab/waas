package multiregion

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ActiveActiveMode defines the delivery mode for active-active
type ActiveActiveMode string

const (
	ActiveActiveModeAll      ActiveActiveMode = "all"      // Deliver from all regions
	ActiveActiveModeNearest  ActiveActiveMode = "nearest"  // Deliver from nearest region
	ActiveActiveModeWeighted ActiveActiveMode = "weighted" // Weighted distribution
	ActiveActiveModeLatency  ActiveActiveMode = "latency"  // Lowest latency wins
)

// EventSyncStatus represents event synchronization status
type EventSyncStatus string

const (
	EventSyncPending  EventSyncStatus = "pending"
	EventSyncSynced   EventSyncStatus = "synced"
	EventSyncFailed   EventSyncStatus = "failed"
	EventSyncConflict EventSyncStatus = "conflict"
)

// ActiveActiveConfig defines active-active delivery configuration
type ActiveActiveConfig struct {
	ID               string           `json:"id" db:"id"`
	TenantID         string           `json:"tenant_id" db:"tenant_id"`
	Mode             ActiveActiveMode `json:"mode" db:"mode"`
	Regions          []string         `json:"regions" db:"regions"`
	Weights          map[string]int   `json:"weights,omitempty" db:"weights"`
	ConflictStrategy ConflictStrategy `json:"conflict_strategy" db:"conflict_strategy"`
	SyncEnabled      bool             `json:"sync_enabled" db:"sync_enabled"`
	SyncIntervalMs   int              `json:"sync_interval_ms" db:"sync_interval_ms"`
	HealthThreshold  float64          `json:"health_threshold" db:"health_threshold"`
	Enabled          bool             `json:"enabled" db:"enabled"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`
}

// CrossRegionEvent represents an event synced across regions
type CrossRegionEvent struct {
	ID            string           `json:"id" db:"id"`
	TenantID      string           `json:"tenant_id" db:"tenant_id"`
	SourceRegion  string           `json:"source_region" db:"source_region"`
	TargetRegions []string         `json:"target_regions" db:"target_regions"`
	EventType     string           `json:"event_type" db:"event_type"`
	Payload       json.RawMessage  `json:"payload" db:"payload"`
	Status        EventSyncStatus  `json:"status" db:"status"`
	VectorClock   map[string]int64 `json:"vector_clock" db:"vector_clock"`
	SyncedRegions []string         `json:"synced_regions" db:"synced_regions"`
	FailedRegions []string         `json:"failed_regions,omitempty" db:"failed_regions"`
	ErrorMessage  string           `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     time.Time        `json:"created_at" db:"created_at"`
	SyncedAt      *time.Time       `json:"synced_at,omitempty" db:"synced_at"`
}

// RegionEndpoint represents a region-aware endpoint for routing
type RegionEndpoint struct {
	ID          string  `json:"id" db:"id"`
	RegionCode  string  `json:"region_code" db:"region_code"`
	URL         string  `json:"url" db:"url"`
	LatencyMs   float64 `json:"latency_ms" db:"latency_ms"`
	SuccessRate float64 `json:"success_rate" db:"success_rate"`
	IsHealthy   bool    `json:"is_healthy" db:"is_healthy"`
	Weight      int     `json:"weight" db:"weight"`
	Priority    int     `json:"priority" db:"priority"`
}

// FailoverConfig defines automatic failover configuration
type AutoFailoverConfig struct {
	ID                string    `json:"id" db:"id"`
	TenantID          string    `json:"tenant_id" db:"tenant_id"`
	Enabled           bool      `json:"enabled" db:"enabled"`
	HealthCheckURL    string    `json:"health_check_url" db:"health_check_url"`
	CheckIntervalSec  int       `json:"check_interval_sec" db:"check_interval_sec"`
	FailureThreshold  int       `json:"failure_threshold" db:"failure_threshold"`
	RecoveryThreshold int       `json:"recovery_threshold" db:"recovery_threshold"`
	AutoFailback      bool      `json:"auto_failback" db:"auto_failback"`
	FailbackDelaySec  int       `json:"failback_delay_sec" db:"failback_delay_sec"`
	NotifyOnFailover  bool      `json:"notify_on_failover" db:"notify_on_failover"`
	NotifyChannels    []string  `json:"notify_channels,omitempty" db:"notify_channels"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// CreateActiveActiveRequest represents a request to create active-active config
type CreateActiveActiveRequest struct {
	Mode             ActiveActiveMode `json:"mode" binding:"required"`
	Regions          []string         `json:"regions" binding:"required,min=2"`
	Weights          map[string]int   `json:"weights,omitempty"`
	ConflictStrategy ConflictStrategy `json:"conflict_strategy,omitempty"`
	SyncEnabled      bool             `json:"sync_enabled"`
	SyncIntervalMs   int              `json:"sync_interval_ms,omitempty"`
	HealthThreshold  float64          `json:"health_threshold,omitempty"`
}

// CreateFailoverConfigRequest represents a request to create failover config
type CreateFailoverConfigRequest struct {
	HealthCheckURL    string   `json:"health_check_url" binding:"required"`
	CheckIntervalSec  int      `json:"check_interval_sec,omitempty"`
	FailureThreshold  int      `json:"failure_threshold,omitempty"`
	RecoveryThreshold int      `json:"recovery_threshold,omitempty"`
	AutoFailback      bool     `json:"auto_failback"`
	FailbackDelaySec  int      `json:"failback_delay_sec,omitempty"`
	NotifyOnFailover  bool     `json:"notify_on_failover"`
	NotifyChannels    []string `json:"notify_channels,omitempty"`
}

// ActiveActiveRepository defines storage for active-active operations
type ActiveActiveRepository interface {
	CreateActiveActiveConfig(ctx context.Context, config *ActiveActiveConfig) error
	GetActiveActiveConfig(ctx context.Context, tenantID string) (*ActiveActiveConfig, error)
	UpdateActiveActiveConfig(ctx context.Context, config *ActiveActiveConfig) error
	DeleteActiveActiveConfig(ctx context.Context, tenantID string) error
	CreateCrossRegionEvent(ctx context.Context, event *CrossRegionEvent) error
	GetCrossRegionEvent(ctx context.Context, eventID string) (*CrossRegionEvent, error)
	ListCrossRegionEvents(ctx context.Context, tenantID string, limit int) ([]CrossRegionEvent, error)
	UpdateEventSyncStatus(ctx context.Context, eventID string, status EventSyncStatus, syncedRegions []string) error
	CreateFailoverConfig(ctx context.Context, config *AutoFailoverConfig) error
	GetFailoverConfig(ctx context.Context, tenantID string) (*AutoFailoverConfig, error)
	UpdateFailoverConfig(ctx context.Context, config *AutoFailoverConfig) error
}

// ActiveActiveService manages active-active delivery and cross-region sync
type ActiveActiveService struct {
	repo       ActiveActiveRepository
	regionRepo Repository
	mu         sync.RWMutex
}

// NewActiveActiveService creates a new active-active service
func NewActiveActiveService(repo ActiveActiveRepository, regionRepo Repository) *ActiveActiveService {
	return &ActiveActiveService{
		repo:       repo,
		regionRepo: regionRepo,
	}
}

// CreateConfig creates an active-active configuration
func (s *ActiveActiveService) CreateConfig(ctx context.Context, tenantID string, req *CreateActiveActiveRequest) (*ActiveActiveConfig, error) {
	if len(req.Regions) < 2 {
		return nil, fmt.Errorf("active-active requires at least 2 regions")
	}

	conflictStrategy := req.ConflictStrategy
	if conflictStrategy == "" {
		conflictStrategy = ConflictStrategyLastWrite
	}
	syncInterval := req.SyncIntervalMs
	if syncInterval == 0 {
		syncInterval = 1000
	}
	healthThreshold := req.HealthThreshold
	if healthThreshold == 0 {
		healthThreshold = 0.95
	}

	now := time.Now()
	config := &ActiveActiveConfig{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		Mode:             req.Mode,
		Regions:          req.Regions,
		Weights:          req.Weights,
		ConflictStrategy: conflictStrategy,
		SyncEnabled:      req.SyncEnabled,
		SyncIntervalMs:   syncInterval,
		HealthThreshold:  healthThreshold,
		Enabled:          true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.CreateActiveActiveConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create active-active config: %w", err)
	}

	return config, nil
}

// SelectRegion selects the best region for delivery based on active-active config
func (s *ActiveActiveService) SelectRegion(ctx context.Context, tenantID string, sourceLatitude, sourceLongitude float64) (string, error) {
	config, err := s.repo.GetActiveActiveConfig(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("active-active config not found: %w", err)
	}

	if !config.Enabled {
		if len(config.Regions) > 0 {
			return config.Regions[0], nil
		}
		return "", fmt.Errorf("no regions configured")
	}

	// Get health for all configured regions
	healths, err := s.regionRepo.GetAllRegionHealth(ctx)
	if err != nil {
		return config.Regions[0], nil
	}

	healthMap := make(map[string]*RegionHealth)
	for _, h := range healths {
		healthMap[h.RegionID] = h
	}

	// Filter to only healthy regions
	var healthyRegions []string
	for _, r := range config.Regions {
		if h, ok := healthMap[r]; ok && h.SuccessRate >= config.HealthThreshold*100 {
			healthyRegions = append(healthyRegions, r)
		}
	}
	if len(healthyRegions) == 0 {
		healthyRegions = config.Regions
	}

	switch config.Mode {
	case ActiveActiveModeLatency:
		return s.selectByLatency(healthyRegions, healthMap), nil
	case ActiveActiveModeNearest:
		return s.selectByProximity(ctx, healthyRegions, sourceLatitude, sourceLongitude), nil
	case ActiveActiveModeWeighted:
		return s.selectByWeight(healthyRegions, config.Weights), nil
	default:
		if len(healthyRegions) > 0 {
			return healthyRegions[0], nil
		}
		return config.Regions[0], nil
	}
}

func (s *ActiveActiveService) selectByLatency(regions []string, healthMap map[string]*RegionHealth) string {
	best := regions[0]
	bestLatency := time.Duration(math.MaxInt64)

	for _, r := range regions {
		if h, ok := healthMap[r]; ok && h.Latency < bestLatency {
			bestLatency = h.Latency
			best = r
		}
	}
	return best
}

func (s *ActiveActiveService) selectByProximity(ctx context.Context, regions []string, lat, lon float64) string {
	if lat == 0 && lon == 0 {
		return regions[0]
	}

	allRegions, err := s.regionRepo.ListActiveRegions(ctx)
	if err != nil {
		return regions[0]
	}

	regionCoords := make(map[string]*GeoCoord)
	for _, r := range allRegions {
		if r.Metadata.Coordinates != nil {
			regionCoords[r.Code] = r.Metadata.Coordinates
		}
	}

	best := regions[0]
	bestDist := math.MaxFloat64
	for _, r := range regions {
		if coord, ok := regionCoords[r]; ok {
			dist := haversineDistance(lat, lon, coord.Latitude, coord.Longitude)
			if dist < bestDist {
				bestDist = dist
				best = r
			}
		}
	}
	return best
}

func (s *ActiveActiveService) selectByWeight(regions []string, weights map[string]int) string {
	if len(weights) == 0 {
		return regions[0]
	}

	type entry struct {
		region string
		weight int
	}
	var entries []entry
	for _, r := range regions {
		w := weights[r]
		if w == 0 {
			w = 1
		}
		entries = append(entries, entry{r, w})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].weight > entries[j].weight
	})

	return entries[0].region
}

// haversineDistance calculates distance between two coordinates in km
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// SyncEvent synchronizes an event across regions
func (s *ActiveActiveService) SyncEvent(ctx context.Context, tenantID, sourceRegion, eventType string, payload json.RawMessage) (*CrossRegionEvent, error) {
	config, err := s.repo.GetActiveActiveConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("active-active config not found: %w", err)
	}

	var targetRegions []string
	for _, r := range config.Regions {
		if r != sourceRegion {
			targetRegions = append(targetRegions, r)
		}
	}

	now := time.Now()
	event := &CrossRegionEvent{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		SourceRegion:  sourceRegion,
		TargetRegions: targetRegions,
		EventType:     eventType,
		Payload:       payload,
		Status:        EventSyncPending,
		VectorClock:   map[string]int64{sourceRegion: now.UnixMilli()},
		SyncedRegions: []string{sourceRegion},
		CreatedAt:     now,
	}

	if err := s.repo.CreateCrossRegionEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create cross-region event: %w", err)
	}

	return event, nil
}

// CreateFailoverConfig creates a failover configuration
func (s *ActiveActiveService) CreateFailoverConfig(ctx context.Context, tenantID string, req *CreateFailoverConfigRequest) (*AutoFailoverConfig, error) {
	checkInterval := req.CheckIntervalSec
	if checkInterval == 0 {
		checkInterval = 30
	}
	failureThreshold := req.FailureThreshold
	if failureThreshold == 0 {
		failureThreshold = 3
	}
	recoveryThreshold := req.RecoveryThreshold
	if recoveryThreshold == 0 {
		recoveryThreshold = 2
	}

	now := time.Now()
	config := &AutoFailoverConfig{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		Enabled:           true,
		HealthCheckURL:    req.HealthCheckURL,
		CheckIntervalSec:  checkInterval,
		FailureThreshold:  failureThreshold,
		RecoveryThreshold: recoveryThreshold,
		AutoFailback:      req.AutoFailback,
		FailbackDelaySec:  req.FailbackDelaySec,
		NotifyOnFailover:  req.NotifyOnFailover,
		NotifyChannels:    req.NotifyChannels,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateFailoverConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create failover config: %w", err)
	}

	return config, nil
}

// GetActiveActiveConfig retrieves active-active configuration
func (s *ActiveActiveService) GetActiveActiveConfig(ctx context.Context, tenantID string) (*ActiveActiveConfig, error) {
	return s.repo.GetActiveActiveConfig(ctx, tenantID)
}

// GetFailoverConfig retrieves failover configuration
func (s *ActiveActiveService) GetFailoverConfig(ctx context.Context, tenantID string) (*AutoFailoverConfig, error) {
	return s.repo.GetFailoverConfig(ctx, tenantID)
}

// ListCrossRegionEvents lists cross-region sync events
func (s *ActiveActiveService) ListCrossRegionEvents(ctx context.Context, tenantID string, limit int) ([]CrossRegionEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListCrossRegionEvents(ctx, tenantID, limit)
}
