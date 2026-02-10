package smartlimit

import (
	"math"
	"sort"
	"sync"
	"time"
)

// CapacityLearner uses observed endpoint behavior to learn capacity limits
// and automatically adjust delivery rates. It implements an online learning
// approach that continuously refines predictions from delivery outcomes.
type CapacityLearner struct {
	models   map[string]*EndpointCapacityModel // key: endpointID
	mu       sync.RWMutex
	config   *CapacityLearnerConfig
}

// CapacityLearnerConfig configures the capacity learning system
type CapacityLearnerConfig struct {
	LearningRate      float64       `json:"learning_rate"`
	ExplorationRate   float64       `json:"exploration_rate"`   // 0-1, probability of probing higher rates
	MinObservations   int           `json:"min_observations"`
	DecayFactor       float64       `json:"decay_factor"`       // exponential decay for old data
	SafetyMargin      float64       `json:"safety_margin"`      // % below learned limit to target
	UpdateInterval    time.Duration `json:"update_interval"`
	ConvergenceThresh float64       `json:"convergence_threshold"`
}

// DefaultCapacityLearnerConfig returns sensible defaults
func DefaultCapacityLearnerConfig() *CapacityLearnerConfig {
	return &CapacityLearnerConfig{
		LearningRate:      0.1,
		ExplorationRate:   0.05,
		MinObservations:   20,
		DecayFactor:       0.95,
		SafetyMargin:      0.15,
		UpdateInterval:    30 * time.Second,
		ConvergenceThresh: 0.01,
	}
}

// EndpointCapacityModel represents the learned capacity model for an endpoint
type EndpointCapacityModel struct {
	EndpointID           string              `json:"endpoint_id"`
	TenantID             string              `json:"tenant_id"`
	LearnedMaxRate       float64             `json:"learned_max_rate"`
	RecommendedRate      float64             `json:"recommended_rate"`
	Confidence           float64             `json:"confidence"` // 0-1
	Observations         int                 `json:"observations"`
	SuccessRateAtLevels  []RateObservation   `json:"success_rate_at_levels"`
	LatencyProfile       *LatencyProfile     `json:"latency_profile"`
	TimeOfDayAdjustments [24]float64         `json:"time_of_day_adjustments"`
	LastUpdated          time.Time           `json:"last_updated"`
	Converged            bool                `json:"converged"`
	ModelVersion         int                 `json:"model_version"`
}

// RateObservation records success rate at a given delivery rate
type RateObservation struct {
	Rate        float64   `json:"rate"`
	SuccessRate float64   `json:"success_rate"`
	AvgLatency  float64   `json:"avg_latency_ms"`
	SampleCount int       `json:"sample_count"`
	ObservedAt  time.Time `json:"observed_at"`
	Weight      float64   `json:"weight"` // Decayed weight
}

// LatencyProfile captures latency characteristics at different load levels
type LatencyProfile struct {
	BaselineLatencyMs float64           `json:"baseline_latency_ms"`
	SaturationPoint   float64           `json:"saturation_point"`  // Rate at which latency starts increasing
	MaxAcceptableMs   float64           `json:"max_acceptable_ms"`
	LatencyCurve      []LatencyCurvePoint `json:"latency_curve"`
}

// LatencyCurvePoint maps a rate to observed latency
type LatencyCurvePoint struct {
	Rate      float64 `json:"rate"`
	LatencyMs float64 `json:"latency_ms"`
}

// RateAdjustment represents a rate adjustment recommendation
type RateAdjustment struct {
	EndpointID      string    `json:"endpoint_id"`
	PreviousRate    float64   `json:"previous_rate"`
	NewRate         float64   `json:"new_rate"`
	Reason          string    `json:"reason"`
	Confidence      float64   `json:"confidence"`
	Direction       string    `json:"direction"` // increase, decrease, hold
	Factor          float64   `json:"factor"`
	ValidUntil      time.Time `json:"valid_until"`
}

// NewCapacityLearner creates a new capacity learning system
func NewCapacityLearner(config *CapacityLearnerConfig) *CapacityLearner {
	if config == nil {
		config = DefaultCapacityLearnerConfig()
	}
	return &CapacityLearner{
		models: make(map[string]*EndpointCapacityModel),
		config: config,
	}
}

// RecordOutcome records a delivery outcome for learning
func (cl *CapacityLearner) RecordOutcome(endpointID, tenantID string, rate float64, success bool, latencyMs float64) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	model, exists := cl.models[endpointID]
	if !exists {
		model = &EndpointCapacityModel{
			EndpointID:          endpointID,
			TenantID:            tenantID,
			SuccessRateAtLevels: make([]RateObservation, 0),
			LatencyProfile:      &LatencyProfile{MaxAcceptableMs: 5000},
		}
		cl.models[endpointID] = model
	}

	model.Observations++
	model.LastUpdated = time.Now()

	// Find or create rate bucket
	bucketIdx := -1
	bucketRange := rate * 0.1 // 10% bucket width
	for i, obs := range model.SuccessRateAtLevels {
		if math.Abs(obs.Rate-rate) <= bucketRange {
			bucketIdx = i
			break
		}
	}

	if bucketIdx >= 0 {
		obs := &model.SuccessRateAtLevels[bucketIdx]
		successVal := 0.0
		if success {
			successVal = 1.0
		}
		// Exponential moving average
		obs.SuccessRate = obs.SuccessRate*(1-cl.config.LearningRate) + successVal*cl.config.LearningRate
		obs.AvgLatency = obs.AvgLatency*(1-cl.config.LearningRate) + latencyMs*cl.config.LearningRate
		obs.SampleCount++
		obs.ObservedAt = time.Now()
	} else {
		successRate := 0.0
		if success {
			successRate = 1.0
		}
		model.SuccessRateAtLevels = append(model.SuccessRateAtLevels, RateObservation{
			Rate:        rate,
			SuccessRate: successRate,
			AvgLatency:  latencyMs,
			SampleCount: 1,
			ObservedAt:  time.Now(),
			Weight:      1.0,
		})
	}

	// Update latency profile
	cl.updateLatencyProfile(model, rate, latencyMs)

	// Recalculate optimal rate if enough observations
	if model.Observations >= cl.config.MinObservations {
		cl.recalculateOptimalRate(model)
	}
}

func (cl *CapacityLearner) updateLatencyProfile(model *EndpointCapacityModel, rate, latencyMs float64) {
	profile := model.LatencyProfile

	// Update baseline (latency at lowest observed rate)
	if len(model.SuccessRateAtLevels) > 0 {
		sort.Slice(model.SuccessRateAtLevels, func(i, j int) bool {
			return model.SuccessRateAtLevels[i].Rate < model.SuccessRateAtLevels[j].Rate
		})
		if model.SuccessRateAtLevels[0].SampleCount > 5 {
			profile.BaselineLatencyMs = model.SuccessRateAtLevels[0].AvgLatency
		}
	}

	// Update latency curve
	found := false
	for i := range profile.LatencyCurve {
		if math.Abs(profile.LatencyCurve[i].Rate-rate) < rate*0.1 {
			profile.LatencyCurve[i].LatencyMs = profile.LatencyCurve[i].LatencyMs*0.9 + latencyMs*0.1
			found = true
			break
		}
	}
	if !found {
		profile.LatencyCurve = append(profile.LatencyCurve, LatencyCurvePoint{
			Rate: rate, LatencyMs: latencyMs,
		})
	}

	// Detect saturation point (where latency > 2x baseline)
	if profile.BaselineLatencyMs > 0 {
		for _, point := range profile.LatencyCurve {
			if point.LatencyMs > profile.BaselineLatencyMs*2 {
				if profile.SaturationPoint == 0 || point.Rate < profile.SaturationPoint {
					profile.SaturationPoint = point.Rate
				}
			}
		}
	}
}

func (cl *CapacityLearner) recalculateOptimalRate(model *EndpointCapacityModel) {
	if len(model.SuccessRateAtLevels) == 0 {
		return
	}

	// Apply time decay to observations
	now := time.Now()
	for i := range model.SuccessRateAtLevels {
		age := now.Sub(model.SuccessRateAtLevels[i].ObservedAt).Hours()
		model.SuccessRateAtLevels[i].Weight = math.Pow(cl.config.DecayFactor, age/24) // Decay per day
	}

	// Find the highest rate with success rate > 95%
	sort.Slice(model.SuccessRateAtLevels, func(i, j int) bool {
		return model.SuccessRateAtLevels[i].Rate < model.SuccessRateAtLevels[j].Rate
	})

	maxSafeRate := 0.0
	totalWeight := 0.0
	for _, obs := range model.SuccessRateAtLevels {
		if obs.SampleCount < 5 || obs.Weight < 0.1 {
			continue
		}
		if obs.SuccessRate >= 0.95 {
			if obs.Rate > maxSafeRate {
				maxSafeRate = obs.Rate
			}
		}
		totalWeight += obs.Weight
	}

	if maxSafeRate > 0 {
		model.LearnedMaxRate = maxSafeRate
		model.RecommendedRate = maxSafeRate * (1 - cl.config.SafetyMargin)

		// Cap by saturation point if known
		if model.LatencyProfile.SaturationPoint > 0 {
			saturationRate := model.LatencyProfile.SaturationPoint * 0.8
			if model.RecommendedRate > saturationRate {
				model.RecommendedRate = saturationRate
			}
		}
	}

	// Compute confidence based on observation count and convergence
	model.Confidence = math.Min(1.0, float64(model.Observations)/float64(cl.config.MinObservations*5))
	if totalWeight > 0 {
		model.Confidence *= math.Min(1.0, totalWeight/10.0)
	}

	// Check convergence
	model.ModelVersion++
	if model.ModelVersion > 10 && model.Confidence > 0.8 {
		model.Converged = true
	}
}

// GetRecommendation returns the current rate recommendation for an endpoint
func (cl *CapacityLearner) GetRecommendation(endpointID string) *RateAdjustment {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	model, exists := cl.models[endpointID]
	if !exists {
		return nil
	}

	if model.Observations < cl.config.MinObservations {
		return &RateAdjustment{
			EndpointID: endpointID,
			NewRate:    0,
			Reason:     "insufficient data for recommendation",
			Direction:  "hold",
			Confidence: 0,
		}
	}

	// Apply time-of-day adjustment
	hour := time.Now().Hour()
	adjustment := model.TimeOfDayAdjustments[hour]
	if adjustment == 0 {
		adjustment = 1.0
	}

	recommendedRate := model.RecommendedRate * adjustment

	direction := "hold"
	if recommendedRate > model.RecommendedRate*1.05 {
		direction = "increase"
	} else if recommendedRate < model.RecommendedRate*0.95 {
		direction = "decrease"
	}

	return &RateAdjustment{
		EndpointID:   endpointID,
		PreviousRate: model.RecommendedRate,
		NewRate:      recommendedRate,
		Reason:       "ML-based capacity learning",
		Confidence:   model.Confidence,
		Direction:    direction,
		Factor:       adjustment,
		ValidUntil:   time.Now().Add(cl.config.UpdateInterval),
	}
}

// GetModel returns the current capacity model for an endpoint
func (cl *CapacityLearner) GetModel(endpointID string) *EndpointCapacityModel {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	if model, exists := cl.models[endpointID]; exists {
		cpy := *model
		return &cpy
	}
	return nil
}

// LearnTimeOfDayPatterns analyzes historical data to learn time-of-day adjustments
func (cl *CapacityLearner) LearnTimeOfDayPatterns(endpointID string, dataPoints []LearningDataPoint) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	model, exists := cl.models[endpointID]
	if !exists {
		return
	}

	// Group data points by hour
	hourCounts := [24]int{}
	hourRates := [24]float64{}
	for _, dp := range dataPoints {
		h := dp.HourOfDay
		if h >= 0 && h < 24 {
			hourCounts[h]++
			hourRates[h] += dp.RequestRate
		}
	}

	// Compute average rate per hour
	overallAvg := 0.0
	activeHours := 0
	for h := 0; h < 24; h++ {
		if hourCounts[h] > 0 {
			hourRates[h] /= float64(hourCounts[h])
			overallAvg += hourRates[h]
			activeHours++
		}
	}
	if activeHours > 0 {
		overallAvg /= float64(activeHours)
	}

	// Compute adjustment factors relative to overall average
	for h := 0; h < 24; h++ {
		if hourCounts[h] > 0 && overallAvg > 0 {
			model.TimeOfDayAdjustments[h] = hourRates[h] / overallAvg
		} else {
			model.TimeOfDayAdjustments[h] = 1.0
		}
	}
}

// Reset clears the learned model for an endpoint
func (cl *CapacityLearner) Reset(endpointID string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	delete(cl.models, endpointID)
}

// Stats returns aggregate statistics across all learned endpoints
func (cl *CapacityLearner) Stats() map[string]interface{} {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	totalEndpoints := len(cl.models)
	convergedCount := 0
	totalObservations := 0
	avgConfidence := 0.0

	for _, model := range cl.models {
		if model.Converged {
			convergedCount++
		}
		totalObservations += model.Observations
		avgConfidence += model.Confidence
	}

	if totalEndpoints > 0 {
		avgConfidence /= float64(totalEndpoints)
	}

	return map[string]interface{}{
		"total_endpoints":    totalEndpoints,
		"converged":          convergedCount,
		"total_observations": totalObservations,
		"avg_confidence":     avgConfidence,
	}
}
