package smartlimit

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// HealthStatus constants
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusDegraded  = "degraded"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusUnknown   = "unknown"
)

// AdaptiveStrategy constants
const (
	StrategyAIMD       = "aimd"       // Additive Increase Multiplicative Decrease
	StrategyToken      = "token"      // Token bucket with dynamic refill
	StrategyLeaky      = "leaky"      // Leaky bucket with health-based drain
	StrategyCongestion = "congestion" // TCP-like congestion control
)

// ReceiverHealth tracks the health of a webhook receiver endpoint.
type ReceiverHealth struct {
	EndpointID        string     `json:"endpoint_id"`
	TenantID          string     `json:"tenant_id"`
	Status            string     `json:"status"`
	AvgResponseTimeMs float64    `json:"avg_response_time_ms"`
	P95ResponseTimeMs float64    `json:"p95_response_time_ms"`
	SuccessRate       float64    `json:"success_rate"`
	ErrorRate         float64    `json:"error_rate"`
	RateLimitHits     int64      `json:"rate_limit_hits"`
	ConsecutiveErrors int        `json:"consecutive_errors"`
	LastCheckedAt     time.Time  `json:"last_checked_at"`
	LastSuccessAt     *time.Time `json:"last_success_at,omitempty"`
	WindowDuration    string     `json:"window_duration"`
}

// AdaptiveRateConfig holds the adaptive rate limit configuration for an endpoint.
type RecvAdaptiveConfig struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	EndpointID         string    `json:"endpoint_id"`
	Strategy           string    `json:"strategy"`
	BaseRatePerSecond  float64   `json:"base_rate_per_second"`
	CurrentRate        float64   `json:"current_rate"`
	MinRate            float64   `json:"min_rate"`
	MaxRate            float64   `json:"max_rate"`
	IncreaseStep       float64   `json:"increase_step"`
	DecreaseMultiplier float64   `json:"decrease_multiplier"`
	HealthThreshold    float64   `json:"health_threshold"`
	Enabled            bool      `json:"enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// RateAdjustment records a rate limit adjustment event.
type RecvRateAdjustment struct {
	ID           string    `json:"id"`
	EndpointID   string    `json:"endpoint_id"`
	TenantID     string    `json:"tenant_id"`
	PreviousRate float64   `json:"previous_rate"`
	NewRate      float64   `json:"new_rate"`
	Reason       string    `json:"reason"`
	HealthScore  float64   `json:"health_score"`
	Timestamp    time.Time `json:"timestamp"`
}

// AdaptiveStats provides statistics about adaptive rate limiting.
type RecvAdaptiveStats struct {
	EndpointID        string               `json:"endpoint_id"`
	CurrentRate       float64              `json:"current_rate"`
	BaseRate          float64              `json:"base_rate"`
	HealthScore       float64              `json:"health_score"`
	AdjustmentCount   int                  `json:"adjustment_count"`
	RecentAdjustments []RecvRateAdjustment `json:"recent_adjustments"`
}

// CapacitySignal represents a signal about receiver capacity.
type CapacitySignal struct {
	EndpointID string    `json:"endpoint_id"`
	SignalType string    `json:"signal_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
}

// Request DTOs

type CreateRecvAdaptiveConfigRequest struct {
	EndpointID        string  `json:"endpoint_id" binding:"required"`
	Strategy          string  `json:"strategy,omitempty"`
	BaseRatePerSecond float64 `json:"base_rate_per_second" binding:"required"`
	MinRate           float64 `json:"min_rate,omitempty"`
	MaxRate           float64 `json:"max_rate,omitempty"`
}

type RecvDeliveryResultRequest struct {
	EndpointID     string `json:"endpoint_id" binding:"required"`
	StatusCode     int    `json:"status_code" binding:"required"`
	ResponseTimeMs int    `json:"response_time_ms" binding:"required"`
	Success        bool   `json:"success"`
}

// adaptiveState stores in-memory state for adaptive rate limiting
type adaptiveState struct {
	mu             sync.RWMutex
	configs        map[string]*RecvAdaptiveConfig
	health         map[string]*ReceiverHealth
	adjustments    map[string][]RecvRateAdjustment
	deliveryWindow map[string]*deliveryWindow
}

type deliveryWindow struct {
	successes  int
	failures   int
	totalMs    int64
	count      int
	maxMs      int64
	p95Samples []int
}

// newAdaptiveState creates a fresh adaptiveState instance.
func newAdaptiveState() *adaptiveState {
	return &adaptiveState{
		configs:        make(map[string]*RecvAdaptiveConfig),
		health:         make(map[string]*ReceiverHealth),
		adjustments:    make(map[string][]RecvRateAdjustment),
		deliveryWindow: make(map[string]*deliveryWindow),
	}
}

// resetAdaptiveState resets the adaptive state; used by tests to prevent state leakage.
func (s *Service) resetAdaptiveState() {
	s.adaptive.mu.Lock()
	defer s.adaptive.mu.Unlock()
	s.adaptive.configs = make(map[string]*RecvAdaptiveConfig)
	s.adaptive.health = make(map[string]*ReceiverHealth)
	s.adaptive.adjustments = make(map[string][]RecvRateAdjustment)
	s.adaptive.deliveryWindow = make(map[string]*deliveryWindow)
}

// CreateAdaptiveConfig creates an adaptive rate limit configuration.
func (s *Service) CreateAdaptiveConfig(ctx context.Context, tenantID string, req *CreateRecvAdaptiveConfigRequest) (*RecvAdaptiveConfig, error) {
	if req.EndpointID == "" {
		return nil, fmt.Errorf("endpoint_id is required")
	}
	if req.BaseRatePerSecond <= 0 {
		return nil, fmt.Errorf("base_rate_per_second must be positive")
	}

	strategy := req.Strategy
	if strategy == "" {
		strategy = StrategyAIMD
	}

	config := &RecvAdaptiveConfig{
		ID:                 uuid.New().String(),
		TenantID:           tenantID,
		EndpointID:         req.EndpointID,
		Strategy:           strategy,
		BaseRatePerSecond:  req.BaseRatePerSecond,
		CurrentRate:        req.BaseRatePerSecond,
		MinRate:            req.MinRate,
		MaxRate:            req.MaxRate,
		IncreaseStep:       req.BaseRatePerSecond * 0.1,
		DecreaseMultiplier: 0.5,
		HealthThreshold:    0.8,
		Enabled:            true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if config.MinRate <= 0 {
		config.MinRate = 1
	}
	if config.MaxRate <= 0 {
		config.MaxRate = req.BaseRatePerSecond * 5
	}

	s.adaptive.mu.Lock()
	s.adaptive.configs[req.EndpointID] = config
	s.adaptive.mu.Unlock()

	return config, nil
}

// GetAdaptiveConfig returns the adaptive config for an endpoint.
func (s *Service) GetAdaptiveConfig(ctx context.Context, tenantID, endpointID string) (*RecvAdaptiveConfig, error) {
	s.adaptive.mu.RLock()
	defer s.adaptive.mu.RUnlock()

	config, exists := s.adaptive.configs[endpointID]
	if !exists {
		return nil, fmt.Errorf("no adaptive config for endpoint %s", endpointID)
	}
	return config, nil
}

// RecordDeliveryResult records a delivery result and adjusts the rate.
func (s *Service) RecordRecvDeliveryResult(ctx context.Context, tenantID string, req *RecvDeliveryResultRequest) (*ReceiverHealth, error) {
	if req.EndpointID == "" {
		return nil, fmt.Errorf("endpoint_id is required")
	}

	s.adaptive.mu.Lock()
	defer s.adaptive.mu.Unlock()

	// Update delivery window
	window, exists := s.adaptive.deliveryWindow[req.EndpointID]
	if !exists {
		window = &deliveryWindow{}
		s.adaptive.deliveryWindow[req.EndpointID] = window
	}

	window.count++
	window.totalMs += int64(req.ResponseTimeMs)
	if req.Success {
		window.successes++
	} else {
		window.failures++
	}
	if int64(req.ResponseTimeMs) > window.maxMs {
		window.maxMs = int64(req.ResponseTimeMs)
	}

	// Compute health
	health := s.computeHealth(req.EndpointID, tenantID, window)
	s.adaptive.health[req.EndpointID] = health

	// Adjust rate based on health
	if config, exists := s.adaptive.configs[req.EndpointID]; exists && config.Enabled {
		s.adjustRate(config, health)
	}

	return health, nil
}

// GetReceiverHealth returns the current health status of a receiver.
func (s *Service) GetReceiverHealth(ctx context.Context, tenantID, endpointID string) (*ReceiverHealth, error) {
	s.adaptive.mu.RLock()
	defer s.adaptive.mu.RUnlock()

	health, exists := s.adaptive.health[endpointID]
	if !exists {
		return &ReceiverHealth{
			EndpointID: endpointID,
			TenantID:   tenantID,
			Status:     HealthStatusUnknown,
		}, nil
	}
	return health, nil
}

// GetAdaptiveStats returns statistics about adaptive rate limiting for an endpoint.
func (s *Service) GetAdaptiveStats(ctx context.Context, tenantID, endpointID string) (*RecvAdaptiveStats, error) {
	s.adaptive.mu.RLock()
	defer s.adaptive.mu.RUnlock()

	stats := &RecvAdaptiveStats{EndpointID: endpointID}

	if config, exists := s.adaptive.configs[endpointID]; exists {
		stats.CurrentRate = config.CurrentRate
		stats.BaseRate = config.BaseRatePerSecond
	}

	if health, exists := s.adaptive.health[endpointID]; exists {
		stats.HealthScore = health.SuccessRate
	}

	if adjustments, exists := s.adaptive.adjustments[endpointID]; exists {
		stats.AdjustmentCount = len(adjustments)
		limit := 10
		if len(adjustments) < limit {
			limit = len(adjustments)
		}
		stats.RecentAdjustments = adjustments[len(adjustments)-limit:]
	}

	return stats, nil
}

func (s *Service) computeHealth(endpointID, tenantID string, window *deliveryWindow) *ReceiverHealth {
	health := &ReceiverHealth{
		EndpointID:     endpointID,
		TenantID:       tenantID,
		LastCheckedAt:  time.Now(),
		WindowDuration: "5m",
	}

	if window.count == 0 {
		health.Status = HealthStatusUnknown
		return health
	}

	health.SuccessRate = float64(window.successes) / float64(window.count)
	health.ErrorRate = float64(window.failures) / float64(window.count)
	health.AvgResponseTimeMs = float64(window.totalMs) / float64(window.count)
	health.P95ResponseTimeMs = float64(window.maxMs) * 0.95

	now := time.Now()
	if window.successes > 0 {
		health.LastSuccessAt = &now
	}

	// Determine status
	if health.SuccessRate >= 0.95 && health.AvgResponseTimeMs < 1000 {
		health.Status = HealthStatusHealthy
	} else if health.SuccessRate >= 0.8 {
		health.Status = HealthStatusDegraded
	} else {
		health.Status = HealthStatusUnhealthy
	}

	return health
}

func (s *Service) adjustRate(config *RecvAdaptiveConfig, health *ReceiverHealth) {
	previousRate := config.CurrentRate
	var reason string

	switch config.Strategy {
	case StrategyAIMD:
		if health.SuccessRate >= config.HealthThreshold {
			// Additive increase
			config.CurrentRate = math.Min(config.CurrentRate+config.IncreaseStep, config.MaxRate)
			reason = "healthy: additive increase"
		} else {
			// Multiplicative decrease
			config.CurrentRate = math.Max(config.CurrentRate*config.DecreaseMultiplier, config.MinRate)
			reason = "degraded: multiplicative decrease"
		}
	case StrategyCongestion:
		if health.SuccessRate >= 0.95 {
			config.CurrentRate = math.Min(config.CurrentRate*1.1, config.MaxRate)
			reason = "healthy: slow start increase"
		} else if health.SuccessRate >= 0.8 {
			reason = "stable: holding rate"
		} else {
			config.CurrentRate = math.Max(config.CurrentRate*0.5, config.MinRate)
			reason = "congestion: halving rate"
		}
	default:
		if health.SuccessRate >= config.HealthThreshold {
			config.CurrentRate = math.Min(config.CurrentRate+config.IncreaseStep, config.MaxRate)
			reason = "healthy: linear increase"
		} else {
			config.CurrentRate = math.Max(config.CurrentRate*0.7, config.MinRate)
			reason = "unhealthy: reducing rate"
		}
	}

	config.UpdatedAt = time.Now()

	if previousRate != config.CurrentRate {
		adjustment := RecvRateAdjustment{
			ID:           uuid.New().String(),
			EndpointID:   config.EndpointID,
			TenantID:     config.TenantID,
			PreviousRate: previousRate,
			NewRate:      config.CurrentRate,
			Reason:       reason,
			HealthScore:  health.SuccessRate,
			Timestamp:    time.Now(),
		}
		s.adaptive.adjustments[config.EndpointID] = append(
			s.adaptive.adjustments[config.EndpointID], adjustment,
		)
	}
}
