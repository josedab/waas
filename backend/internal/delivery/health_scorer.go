package delivery

import (
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// HealthScore represents a 0-100 health score for an endpoint
type HealthScore struct {
	EndpointID        uuid.UUID `json:"endpoint_id"`
	Score             int       `json:"score"`               // 0-100
	Grade             string    `json:"grade"`                // A, B, C, D, F
	SuccessRate       float64   `json:"success_rate"`         // 0-1
	AvgLatencyMs      float64   `json:"avg_latency_ms"`
	P95LatencyMs      float64   `json:"p95_latency_ms"`
	ErrorRate         float64   `json:"error_rate"`           // 0-1
	ConsecutiveErrors int       `json:"consecutive_errors"`
	CircuitState      string    `json:"circuit_state"`        // closed, open, half_open
	TotalDeliveries   int64     `json:"total_deliveries"`
	RecentDeliveries  int64     `json:"recent_deliveries"`    // Last hour
	Trend             string    `json:"trend"`                // improving, stable, degrading
	IsPaused          bool      `json:"is_paused"`
	PausedAt          *time.Time `json:"paused_at,omitempty"`
	PauseReason       string    `json:"pause_reason,omitempty"`
	LastUpdated       time.Time `json:"last_updated"`
}

// HealthScoringConfig configures the health scoring behavior
type HealthScoringConfig struct {
	// Weight factors for score calculation (must sum to 1.0)
	SuccessRateWeight    float64
	LatencyWeight        float64
	ErrorPatternWeight   float64
	ConsistencyWeight    float64

	// Thresholds for auto-pause
	PauseThreshold       int           // Score below which endpoint is auto-paused
	ResumeThreshold      int           // Score above which endpoint is auto-resumed
	MinDeliveriesForScore int          // Min deliveries before scoring is active
	ScoringWindow        time.Duration // Window for recent delivery calculation

	// Latency targets
	TargetLatencyMs      float64
	MaxAcceptableLatency float64
}

// DefaultHealthScoringConfig returns sensible defaults
func DefaultHealthScoringConfig() *HealthScoringConfig {
	return &HealthScoringConfig{
		SuccessRateWeight:    0.40,
		LatencyWeight:        0.20,
		ErrorPatternWeight:   0.25,
		ConsistencyWeight:    0.15,
		PauseThreshold:       20,
		ResumeThreshold:      60,
		MinDeliveriesForScore: 10,
		ScoringWindow:        1 * time.Hour,
		TargetLatencyMs:      500,
		MaxAcceptableLatency: 5000,
	}
}

// HealthScorer calculates and tracks health scores for endpoints
type HealthScorer struct {
	mu       sync.RWMutex
	scores   map[uuid.UUID]*healthState
	config   *HealthScoringConfig
	onPause  func(endpointID uuid.UUID, reason string)
	onResume func(endpointID uuid.UUID)
}

// healthState tracks raw metrics for score calculation
type healthState struct {
	endpointID         uuid.UUID
	recentSuccesses    int64
	recentFailures     int64
	totalSuccesses     int64
	totalFailures      int64
	consecutiveErrors  int
	consecutiveSuccess int
	latencies          []float64 // ring buffer of recent latencies
	latencyIdx         int
	latencyFull        bool
	windowStart        time.Time
	isPaused           bool
	pausedAt           *time.Time
	pauseReason        string
	lastScore          int
	previousScores     []int // last N scores for trend detection
	lastUpdated        time.Time
}

const (
	latencyBufferSize = 100
	trendWindowSize   = 10
)

// NewHealthScorer creates a new health scorer
func NewHealthScorer(config *HealthScoringConfig) *HealthScorer {
	if config == nil {
		config = DefaultHealthScoringConfig()
	}
	return &HealthScorer{
		scores: make(map[uuid.UUID]*healthState),
		config: config,
	}
}

// SetPauseCallback sets the function called when an endpoint is auto-paused
func (hs *HealthScorer) SetPauseCallback(fn func(endpointID uuid.UUID, reason string)) {
	hs.onPause = fn
}

// SetResumeCallback sets the function called when an endpoint is auto-resumed
func (hs *HealthScorer) SetResumeCallback(fn func(endpointID uuid.UUID)) {
	hs.onResume = fn
}

// RecordDelivery records a delivery result for health scoring
func (hs *HealthScorer) RecordDelivery(endpointID uuid.UUID, success bool, latencyMs float64, httpStatus int) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	state, exists := hs.scores[endpointID]
	if !exists {
		state = &healthState{
			endpointID:  endpointID,
			latencies:   make([]float64, latencyBufferSize),
			windowStart: time.Now(),
			lastScore:   100,
		}
		hs.scores[endpointID] = state
	}

	now := time.Now()

	// Reset window if expired
	if now.Sub(state.windowStart) > hs.config.ScoringWindow {
		state.recentSuccesses = 0
		state.recentFailures = 0
		state.windowStart = now
	}

	if success {
		state.recentSuccesses++
		state.totalSuccesses++
		state.consecutiveErrors = 0
		state.consecutiveSuccess++
	} else {
		state.recentFailures++
		state.totalFailures++
		state.consecutiveErrors++
		state.consecutiveSuccess = 0
	}

	// Record latency in ring buffer
	state.latencies[state.latencyIdx] = latencyMs
	state.latencyIdx = (state.latencyIdx + 1) % latencyBufferSize
	if state.latencyIdx == 0 {
		state.latencyFull = true
	}

	state.lastUpdated = now

	// Calculate new score
	score := hs.calculateScore(state)
	state.lastScore = score

	// Track score history for trend
	state.previousScores = append(state.previousScores, score)
	if len(state.previousScores) > trendWindowSize {
		state.previousScores = state.previousScores[len(state.previousScores)-trendWindowSize:]
	}

	// Auto-pause/resume logic
	hs.evaluateCircuitAction(state, score)
}

// GetScore returns the current health score for an endpoint
func (hs *HealthScorer) GetScore(endpointID uuid.UUID) *HealthScore {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	state, exists := hs.scores[endpointID]
	if !exists {
		return &HealthScore{
			EndpointID: endpointID,
			Score:      100,
			Grade:      "A",
			Trend:      "stable",
		}
	}

	score := hs.calculateScore(state)

	recentTotal := state.recentSuccesses + state.recentFailures
	successRate := 0.0
	if recentTotal > 0 {
		successRate = float64(state.recentSuccesses) / float64(recentTotal)
	}

	avgLatency, p95Latency := hs.computeLatencyStats(state)

	return &HealthScore{
		EndpointID:        endpointID,
		Score:             score,
		Grade:             scoreToGrade(score),
		SuccessRate:       math.Round(successRate*10000) / 10000,
		AvgLatencyMs:      math.Round(avgLatency*100) / 100,
		P95LatencyMs:      math.Round(p95Latency*100) / 100,
		ErrorRate:         math.Round((1-successRate)*10000) / 10000,
		ConsecutiveErrors: state.consecutiveErrors,
		TotalDeliveries:   state.totalSuccesses + state.totalFailures,
		RecentDeliveries:  recentTotal,
		Trend:             hs.detectTrend(state),
		IsPaused:          state.isPaused,
		PausedAt:          state.pausedAt,
		PauseReason:       state.pauseReason,
		LastUpdated:       state.lastUpdated,
	}
}

// GetAllScores returns health scores for all tracked endpoints
func (hs *HealthScorer) GetAllScores() map[uuid.UUID]*HealthScore {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	result := make(map[uuid.UUID]*HealthScore, len(hs.scores))
	for id := range hs.scores {
		// Temporarily unlock for GetScore (which takes read lock)
		hs.mu.RUnlock()
		result[id] = hs.GetScore(id)
		hs.mu.RLock()
	}
	return result
}

func (hs *HealthScorer) calculateScore(state *healthState) int {
	recentTotal := state.recentSuccesses + state.recentFailures
	if recentTotal < int64(hs.config.MinDeliveriesForScore) {
		return state.lastScore // Not enough data, keep previous score
	}

	// 1. Success rate component (0-100)
	successRate := float64(state.recentSuccesses) / float64(recentTotal)
	successScore := successRate * 100

	// 2. Latency component (0-100)
	avgLatency, _ := hs.computeLatencyStats(state)
	latencyScore := 100.0
	if avgLatency > hs.config.TargetLatencyMs {
		excess := avgLatency - hs.config.TargetLatencyMs
		maxExcess := hs.config.MaxAcceptableLatency - hs.config.TargetLatencyMs
		if maxExcess > 0 {
			latencyScore = math.Max(0, 100-((excess/maxExcess)*100))
		}
	}

	// 3. Error pattern component (0-100) - penalizes consecutive errors heavily
	errorPatternScore := 100.0
	if state.consecutiveErrors > 0 {
		// Each consecutive error reduces score more aggressively
		penalty := math.Min(100, float64(state.consecutiveErrors)*float64(state.consecutiveErrors)*5)
		errorPatternScore = math.Max(0, 100-penalty)
	}

	// 4. Consistency component (0-100) - standard deviation of recent success
	consistencyScore := 100.0
	if len(state.previousScores) >= 3 {
		variance := 0.0
		mean := 0.0
		for _, s := range state.previousScores {
			mean += float64(s)
		}
		mean /= float64(len(state.previousScores))
		for _, s := range state.previousScores {
			diff := float64(s) - mean
			variance += diff * diff
		}
		variance /= float64(len(state.previousScores))
		stddev := math.Sqrt(variance)
		// High stddev = inconsistent = lower score
		consistencyScore = math.Max(0, 100-stddev*2)
	}

	// Weighted final score
	finalScore := successScore*hs.config.SuccessRateWeight +
		latencyScore*hs.config.LatencyWeight +
		errorPatternScore*hs.config.ErrorPatternWeight +
		consistencyScore*hs.config.ConsistencyWeight

	return int(math.Round(math.Max(0, math.Min(100, finalScore))))
}

func (hs *HealthScorer) computeLatencyStats(state *healthState) (avg, p95 float64) {
	count := latencyBufferSize
	if !state.latencyFull {
		count = state.latencyIdx
	}
	if count == 0 {
		return 0, 0
	}

	// Collect valid latencies
	latencies := make([]float64, 0, count)
	sum := 0.0
	for i := 0; i < count; i++ {
		l := state.latencies[i]
		if l > 0 {
			latencies = append(latencies, l)
			sum += l
		}
	}

	if len(latencies) == 0 {
		return 0, 0
	}

	avg = sum / float64(len(latencies))

	// Simple P95 approximation (sort and pick 95th percentile)
	// Using insertion sort for small buffers
	sorted := make([]float64, len(latencies))
	copy(sorted, latencies)
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	p95Idx := int(float64(len(sorted)) * 0.95)
	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}
	p95 = sorted[p95Idx]

	return avg, p95
}

func (hs *HealthScorer) detectTrend(state *healthState) string {
	scores := state.previousScores
	if len(scores) < 3 {
		return "stable"
	}

	// Compare average of first half vs second half
	half := len(scores) / 2
	firstSum, secondSum := 0.0, 0.0
	for i := 0; i < half; i++ {
		firstSum += float64(scores[i])
	}
	for i := half; i < len(scores); i++ {
		secondSum += float64(scores[i])
	}
	firstAvg := firstSum / float64(half)
	secondAvg := secondSum / float64(len(scores)-half)

	diff := secondAvg - firstAvg
	if diff > 5 {
		return "improving"
	}
	if diff < -5 {
		return "degrading"
	}
	return "stable"
}

func (hs *HealthScorer) evaluateCircuitAction(state *healthState, score int) {
	if !state.isPaused && score <= hs.config.PauseThreshold {
		now := time.Now()
		state.isPaused = true
		state.pausedAt = &now
		state.pauseReason = "health score below threshold"

		if hs.onPause != nil {
			go hs.onPause(state.endpointID, state.pauseReason)
		}
	} else if state.isPaused && score >= hs.config.ResumeThreshold {
		state.isPaused = false
		state.pausedAt = nil
		state.pauseReason = ""

		if hs.onResume != nil {
			go hs.onResume(state.endpointID)
		}
	}
}

func scoreToGrade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 60:
		return "C"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}
