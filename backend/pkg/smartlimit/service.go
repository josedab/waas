package smartlimit

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service provides intelligent rate limiting functionality
type Service struct {
	repo       Repository
	predictor  *Predictor
	analyzer   *PatternAnalyzer
	throttlers sync.Map // map[endpointID]*Throttler
	config     *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	DefaultWindowSeconds int
	DefaultBurstSize     int
	MinDataPointsForML   int
	PredictionInterval   time.Duration
	LearningInterval     time.Duration
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		DefaultWindowSeconds: 60,
		DefaultBurstSize:     10,
		MinDataPointsForML:   100,
		PredictionInterval:   5 * time.Minute,
		LearningInterval:     1 * time.Hour,
	}
}

// NewService creates a new smart limit service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	
	return &Service{
		repo:      repo,
		predictor: NewPredictor(),
		analyzer:  NewPatternAnalyzer(repo, DefaultPatternConfig()),
		config:    config,
	}
}

// CreateConfig creates an adaptive rate config
func (s *Service) CreateConfig(ctx context.Context, tenantID string, req *CreateAdaptiveConfigRequest) (*AdaptiveRateConfig, error) {
	config := &AdaptiveRateConfig{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		EndpointID:      req.EndpointID,
		Enabled:         true,
		Mode:            req.Mode,
		BaseRatePerSec:  req.BaseRatePerSec,
		MinRatePerSec:   req.MinRatePerSec,
		MaxRatePerSec:   req.MaxRatePerSec,
		BurstSize:       req.BurstSize,
		RiskThreshold:   req.RiskThreshold,
		BackoffFactor:   0.5,
		RecoveryFactor:  1.1,
		WindowSeconds:   s.config.DefaultWindowSeconds,
		LearningEnabled: req.LearningEnabled,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Set defaults
	if config.Mode == "" {
		config.Mode = RateModeAdaptive
	}
	if config.BaseRatePerSec == 0 {
		config.BaseRatePerSec = 10
	}
	if config.MinRatePerSec == 0 {
		config.MinRatePerSec = 1
	}
	if config.MaxRatePerSec == 0 {
		config.MaxRatePerSec = 100
	}
	if config.BurstSize == 0 {
		config.BurstSize = s.config.DefaultBurstSize
	}
	if config.RiskThreshold == 0 {
		config.RiskThreshold = 0.7
	}

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return nil, err
	}

	// Initialize throttler
	s.getOrCreateThrottler(tenantID, config)

	return config, nil
}

// GetConfig retrieves an adaptive rate config
func (s *Service) GetConfig(ctx context.Context, tenantID, endpointID string) (*AdaptiveRateConfig, error) {
	return s.repo.GetConfig(ctx, tenantID, endpointID)
}

// ListConfigs lists all adaptive rate configs for a tenant
func (s *Service) ListConfigs(ctx context.Context, tenantID string) ([]AdaptiveRateConfig, error) {
	return s.repo.ListConfigs(ctx, tenantID)
}

// UpdateConfig updates an adaptive rate config
func (s *Service) UpdateConfig(ctx context.Context, tenantID, endpointID string, req *UpdateAdaptiveConfigRequest) (*AdaptiveRateConfig, error) {
	config, err := s.repo.GetConfig(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.Mode != nil {
		config.Mode = *req.Mode
	}
	if req.BaseRatePerSec != nil {
		config.BaseRatePerSec = *req.BaseRatePerSec
	}
	if req.MinRatePerSec != nil {
		config.MinRatePerSec = *req.MinRatePerSec
	}
	if req.MaxRatePerSec != nil {
		config.MaxRatePerSec = *req.MaxRatePerSec
	}
	if req.BurstSize != nil {
		config.BurstSize = *req.BurstSize
	}
	if req.RiskThreshold != nil {
		config.RiskThreshold = *req.RiskThreshold
	}
	if req.LearningEnabled != nil {
		config.LearningEnabled = *req.LearningEnabled
	}

	config.UpdatedAt = time.Now()

	if err := s.repo.SaveConfig(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

// DeleteConfig deletes an adaptive rate config
func (s *Service) DeleteConfig(ctx context.Context, tenantID, endpointID string) error {
	s.throttlers.Delete(tenantID + ":" + endpointID)
	return s.repo.DeleteConfig(ctx, tenantID, endpointID)
}

// ShouldThrottle determines if a request should be throttled
func (s *Service) ShouldThrottle(ctx context.Context, tenantID, endpointID string) (*ThrottleDecision, error) {
	config, err := s.repo.GetConfig(ctx, tenantID, endpointID)
	if err != nil {
		// No config, allow by default
		return &ThrottleDecision{
			EndpointID: endpointID,
			Allowed:    true,
			Reason:     "no rate limit configured",
		}, nil
	}

	if !config.Enabled {
		return &ThrottleDecision{
			EndpointID: endpointID,
			Allowed:    true,
			Reason:     "rate limiting disabled",
		}, nil
	}

	throttler := s.getOrCreateThrottler(tenantID, config)
	return throttler.Allow()
}

// RecordDeliveryResult records the result of a delivery attempt
func (s *Service) RecordDeliveryResult(ctx context.Context, tenantID, endpointID, deliveryID string, statusCode int, latencyMs int64, wasRateLimited bool) error {
	config, _ := s.repo.GetConfig(ctx, tenantID, endpointID)
	
	// Update throttler state
	if throttler, ok := s.throttlers.Load(tenantID + ":" + endpointID); ok {
		t := throttler.(*Throttler)
		t.RecordResult(statusCode, wasRateLimited)
	}

	// Record learning data if enabled
	if config != nil && config.LearningEnabled {
		now := time.Now()
		data := &LearningDataPoint{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			EndpointID:  endpointID,
			Timestamp:   now,
			HourOfDay:   now.Hour(),
			DayOfWeek:   int(now.Weekday()),
			AvgLatency:  float64(latencyMs),
			RateLimited: wasRateLimited,
			ResponseCode: statusCode,
		}
		if statusCode >= 200 && statusCode < 300 {
			data.SuccessRate = 1.0
		}
		s.repo.SaveLearningData(ctx, data)
	}

	// Record rate limit event
	if wasRateLimited || statusCode == 429 {
		event := &RateLimitEvent{
			ID:         uuid.New().String(),
			TenantID:   tenantID,
			EndpointID: endpointID,
			DeliveryID: deliveryID,
			Timestamp:  time.Now(),
			EventType:  "hit",
			StatusCode: statusCode,
		}
		s.repo.SaveEvent(ctx, event)
	}

	return nil
}

// GetPrediction gets a rate limit prediction for an endpoint
func (s *Service) GetPrediction(ctx context.Context, tenantID, endpointID string) (*RateLimitPrediction, error) {
	behavior, err := s.repo.GetBehavior(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}

	model, _ := s.repo.GetActiveModel(ctx, tenantID, endpointID)
	state, _ := s.repo.GetState(ctx, tenantID, endpointID)

	return s.predictor.Predict(behavior, model, state)
}

// GetStats retrieves smart limit statistics
func (s *Service) GetStats(ctx context.Context, tenantID string, start, end time.Time) (*SmartLimitStats, error) {
	return s.repo.GetStats(ctx, tenantID, start, end)
}

// TrainModel trains or updates the prediction model for an endpoint
func (s *Service) TrainModel(ctx context.Context, tenantID, endpointID string) (*PredictionModel, error) {
	// Get learning data
	start := time.Now().Add(-30 * 24 * time.Hour) // Last 30 days
	end := time.Now()
	
	data, err := s.repo.GetLearningData(ctx, tenantID, endpointID, start, end)
	if err != nil {
		return nil, err
	}

	if len(data) < s.config.MinDataPointsForML {
		return nil, nil // Not enough data
	}

	// Get existing model for version
	existingModel, _ := s.repo.GetActiveModel(ctx, tenantID, endpointID)
	version := 1
	if existingModel != nil {
		version = existingModel.Version + 1
	}

	// Train model
	model := s.predictor.Train(tenantID, endpointID, data, version)
	
	if err := s.repo.SaveModel(ctx, model); err != nil {
		return nil, err
	}

	return model, nil
}

func (s *Service) getOrCreateThrottler(tenantID string, config *AdaptiveRateConfig) *Throttler {
	key := tenantID + ":" + config.EndpointID
	
	if throttler, ok := s.throttlers.Load(key); ok {
		return throttler.(*Throttler)
	}

	throttler := NewThrottler(config)
	s.throttlers.Store(key, throttler)
	return throttler
}

// Throttler implements adaptive rate limiting for a single endpoint
type Throttler struct {
	config       *AdaptiveRateConfig
	mu           sync.Mutex
	currentRate  float64
	tokens       float64
	lastUpdate   time.Time
	windowStart  time.Time
	requestCount int64
	successCount int64
	failCount    int64
	rateLimitHits int64
}

// NewThrottler creates a new throttler
func NewThrottler(config *AdaptiveRateConfig) *Throttler {
	return &Throttler{
		config:      config,
		currentRate: config.BaseRatePerSec,
		tokens:      float64(config.BurstSize),
		lastUpdate:  time.Now(),
		windowStart: time.Now(),
	}
}

// Allow checks if a request should be allowed
func (t *Throttler) Allow() (*ThrottleDecision, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastUpdate).Seconds()
	t.lastUpdate = now

	// Refill tokens based on rate
	t.tokens = math.Min(
		float64(t.config.BurstSize),
		t.tokens + elapsed*t.currentRate,
	)

	// Check window reset
	if now.Sub(t.windowStart).Seconds() >= float64(t.config.WindowSeconds) {
		t.windowStart = now
		t.requestCount = 0
		t.successCount = 0
		t.failCount = 0
		t.rateLimitHits = 0
	}

	t.requestCount++

	if t.tokens < 1 {
		waitDuration := time.Duration((1-t.tokens)/t.currentRate*1000) * time.Millisecond
		return &ThrottleDecision{
			EndpointID:     t.config.EndpointID,
			Allowed:        false,
			WaitDuration:   waitDuration,
			Reason:         "rate limit exceeded",
			CurrentRate:    t.currentRate,
			AllowedRate:    t.currentRate,
			RemainingBurst: 0,
			ResetAt:        now.Add(waitDuration),
		}, nil
	}

	t.tokens--

	return &ThrottleDecision{
		EndpointID:     t.config.EndpointID,
		Allowed:        true,
		Reason:         "allowed",
		CurrentRate:    t.currentRate,
		AllowedRate:    t.currentRate,
		RemainingBurst: int(t.tokens),
		ResetAt:        t.windowStart.Add(time.Duration(t.config.WindowSeconds) * time.Second),
	}, nil
}

// RecordResult records a delivery result and adjusts rate
func (t *Throttler) RecordResult(statusCode int, wasRateLimited bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if wasRateLimited || statusCode == 429 {
		t.rateLimitHits++
		t.failCount++
		// Backoff
		t.currentRate = math.Max(
			t.config.MinRatePerSec,
			t.currentRate * t.config.BackoffFactor,
		)
	} else if statusCode >= 200 && statusCode < 300 {
		t.successCount++
		// Gradual recovery if no recent rate limits
		if t.rateLimitHits == 0 && t.successCount > 10 {
			t.currentRate = math.Min(
				t.config.MaxRatePerSec,
				t.currentRate * t.config.RecoveryFactor,
			)
		}
	} else {
		t.failCount++
	}
}

// Predictor handles rate limit predictions
type Predictor struct{}

// NewPredictor creates a new predictor
func NewPredictor() *Predictor {
	return &Predictor{}
}

// Predict generates a rate limit prediction
func (p *Predictor) Predict(behavior *EndpointBehavior, model *PredictionModel, state *RateLimitState) (*RateLimitPrediction, error) {
	prediction := &RateLimitPrediction{
		EndpointID:      behavior.EndpointID,
		ValidUntil:      time.Now().Add(5 * time.Minute),
		PredictedWindow: 60,
	}

	// Calculate risk score based on historical data
	if behavior.TotalRequests > 0 {
		rateLimitRatio := float64(behavior.RateLimitCount) / float64(behavior.TotalRequests)
		prediction.RiskScore = rateLimitRatio

		// Use model if available
		if model != nil && model.Accuracy > 0.7 {
			prediction.Confidence = model.Accuracy
			// Use model coefficients for prediction
			if val, ok := model.Coefficients["base_rate"]; ok {
				prediction.PredictedLimit = int(val)
			}
		} else {
			prediction.Confidence = 0.5
		}

		// Calculate recommended rate
		if state != nil {
			prediction.CurrentRate = state.CurrentRate
		}

		// Estimate limit from successful request rate
		successRate := float64(behavior.SuccessCount) / float64(behavior.TotalRequests)
		if successRate > 0.9 {
			prediction.PredictedLimit = int(float64(behavior.TotalRequests) / behavior.WindowEnd.Sub(behavior.WindowStart).Seconds())
			prediction.RecommendedRate = float64(prediction.PredictedLimit) * 0.8 // 80% of limit
		} else {
			prediction.RecommendedRate = float64(behavior.SuccessCount) / behavior.WindowEnd.Sub(behavior.WindowStart).Seconds()
		}

		// Determine if backoff is needed
		prediction.BackoffRecommended = prediction.RiskScore > 0.3
		if prediction.BackoffRecommended {
			prediction.Reason = "High rate limit risk detected based on historical patterns"
		} else {
			prediction.Reason = "Current rate is sustainable"
		}
	}

	return prediction, nil
}

// Train trains a prediction model from learning data
func (p *Predictor) Train(tenantID, endpointID string, data []LearningDataPoint, version int) *PredictionModel {
	model := &PredictionModel{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		EndpointID:     endpointID,
		ModelType:      "linear",
		Version:        version,
		Features:       []string{"hour_of_day", "day_of_week", "request_rate"},
		Coefficients:   make(map[string]float64),
		DataPointCount: int64(len(data)),
		TrainedAt:      time.Now(),
		IsActive:       true,
		CreatedAt:      time.Now(),
	}

	// Simple linear regression on rate-limited vs not
	var sumRate, sumLatency float64
	var rateLimitedCount, successCount int64
	
	for _, point := range data {
		sumRate += point.RequestRate
		sumLatency += point.AvgLatency
		if point.RateLimited {
			rateLimitedCount++
		} else if point.ResponseCode >= 200 && point.ResponseCode < 300 {
			successCount++
		}
	}

	n := float64(len(data))
	if n > 0 {
		avgRate := sumRate / n
		avgLatency := sumLatency / n
		
		// Store coefficients
		model.Coefficients["avg_rate"] = avgRate
		model.Coefficients["avg_latency"] = avgLatency
		model.Coefficients["rate_limit_ratio"] = float64(rateLimitedCount) / n
		
		// Estimate safe rate (rate where 95% success)
		if successCount > 0 {
			safeRate := avgRate * (float64(successCount) / n) * 0.95
			model.Coefficients["base_rate"] = safeRate
		}

		// Calculate accuracy based on how well we can predict rate limits
		if rateLimitedCount > 0 {
			model.Accuracy = 1 - (float64(rateLimitedCount) / n)
		} else {
			model.Accuracy = 0.9 // No rate limits = good
		}
	}

	return model
}
