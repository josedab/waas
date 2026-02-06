package livemigration

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ShadowMode implements dual-write shadow delivery for migration verification.
// During shadow mode, deliveries are sent to both the old and new platforms
// and results are compared to verify equivalence.
type ShadowMode struct {
	jobID    string
	tenantID string
	enabled  bool
	config   *ShadowConfig
	results  []ShadowResult
	mu       sync.RWMutex
	metrics  ShadowMetrics
}

// ShadowConfig configures shadow mode behavior
type ShadowConfig struct {
	SampleRate      float64 `json:"sample_rate"`       // 0.0-1.0, fraction of traffic to shadow
	CompareResponse bool    `json:"compare_response"`  // Compare response bodies
	CompareStatus   bool    `json:"compare_status"`    // Compare HTTP status codes
	CompareLatency  bool    `json:"compare_latency"`   // Compare latency within threshold
	LatencyThreshMs int64   `json:"latency_thresh_ms"` // Max acceptable latency diff
	MaxShadowQPS    int     `json:"max_shadow_qps"`    // Rate limit for shadow traffic
}

// DefaultShadowConfig returns sensible defaults
func DefaultShadowConfig() *ShadowConfig {
	return &ShadowConfig{
		SampleRate:      0.1, // 10% of traffic
		CompareResponse: false,
		CompareStatus:   true,
		CompareLatency:  true,
		LatencyThreshMs: 2000,
		MaxShadowQPS:    100,
	}
}

// ShadowResult captures the result of a single shadow delivery comparison
type ShadowResult struct {
	ID               string    `json:"id"`
	EndpointID       string    `json:"endpoint_id"`
	EventID          string    `json:"event_id"`
	PrimaryStatus    int       `json:"primary_status"`
	ShadowStatus     int       `json:"shadow_status"`
	PrimaryLatencyMs int64     `json:"primary_latency_ms"`
	ShadowLatencyMs  int64     `json:"shadow_latency_ms"`
	StatusMatch      bool      `json:"status_match"`
	LatencyMatch     bool      `json:"latency_match"`
	ResponseMatch    bool      `json:"response_match"`
	Discrepancy      string    `json:"discrepancy,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

// ShadowMetrics aggregates shadow mode metrics
type ShadowMetrics struct {
	TotalCompared     int64   `json:"total_compared"`
	StatusMatches     int64   `json:"status_matches"`
	LatencyMatches    int64   `json:"latency_matches"`
	ResponseMatches   int64   `json:"response_matches"`
	Discrepancies     int64   `json:"discrepancies"`
	MatchRate         float64 `json:"match_rate"`
	AvgPrimaryLatency float64 `json:"avg_primary_latency_ms"`
	AvgShadowLatency  float64 `json:"avg_shadow_latency_ms"`
}

// NewShadowMode creates a new shadow mode instance
func NewShadowMode(jobID, tenantID string, config *ShadowConfig) *ShadowMode {
	if config == nil {
		config = DefaultShadowConfig()
	}
	return &ShadowMode{
		jobID:    jobID,
		tenantID: tenantID,
		enabled:  true,
		config:   config,
	}
}

// ShouldShadow returns true if this delivery should be shadowed based on sample rate
func (sm *ShadowMode) ShouldShadow() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if !sm.enabled {
		return false
	}
	return rand.Float64() < sm.config.SampleRate
}

// RecordComparison records the result of a shadow delivery comparison
func (sm *ShadowMode) RecordComparison(endpointID, eventID string, primaryStatus, shadowStatus int, primaryLatency, shadowLatency int64, responseMatch bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	statusMatch := primaryStatus == shadowStatus
	latencyMatch := abs64(primaryLatency-shadowLatency) <= sm.config.LatencyThreshMs

	result := ShadowResult{
		ID:               uuid.New().String(),
		EndpointID:       endpointID,
		EventID:          eventID,
		PrimaryStatus:    primaryStatus,
		ShadowStatus:     shadowStatus,
		PrimaryLatencyMs: primaryLatency,
		ShadowLatencyMs:  shadowLatency,
		StatusMatch:      statusMatch,
		LatencyMatch:     latencyMatch,
		ResponseMatch:    responseMatch,
		Timestamp:        time.Now(),
	}

	if !statusMatch {
		result.Discrepancy = fmt.Sprintf("status mismatch: primary=%d shadow=%d", primaryStatus, shadowStatus)
	} else if !latencyMatch {
		result.Discrepancy = fmt.Sprintf("latency delta %dms exceeds threshold %dms", abs64(primaryLatency-shadowLatency), sm.config.LatencyThreshMs)
	}

	sm.results = append(sm.results, result)

	// Update metrics
	sm.metrics.TotalCompared++
	if statusMatch {
		sm.metrics.StatusMatches++
	}
	if latencyMatch {
		sm.metrics.LatencyMatches++
	}
	if responseMatch {
		sm.metrics.ResponseMatches++
	}
	if !statusMatch || !latencyMatch || !responseMatch {
		sm.metrics.Discrepancies++
	}

	n := float64(sm.metrics.TotalCompared)
	sm.metrics.MatchRate = float64(sm.metrics.StatusMatches) / n
	sm.metrics.AvgPrimaryLatency = (sm.metrics.AvgPrimaryLatency*(n-1) + float64(primaryLatency)) / n
	sm.metrics.AvgShadowLatency = (sm.metrics.AvgShadowLatency*(n-1) + float64(shadowLatency)) / n
}

// GetMetrics returns current shadow mode metrics
func (sm *ShadowMode) GetMetrics() ShadowMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.metrics
}

// Disable stops shadow mode
func (sm *ShadowMode) Disable() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.enabled = false
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// --- Traffic Splitting ---

// TrafficSplitter implements gradual traffic migration from source to destination.
// Traffic is split between old and new platforms based on configurable percentages.
type TrafficSplitter struct {
	jobID    string
	tenantID string
	config   *SplitConfig
	mu       sync.RWMutex
}

// SplitConfig configures traffic splitting behavior
type SplitConfig struct {
	DestinationPct   int     `json:"destination_pct"`     // 0-100, percentage to new destination
	RampUpStepPct    int     `json:"ramp_up_step_pct"`    // Step size for automatic ramp-up
	RampUpIntervalMs int     `json:"ramp_up_interval_ms"` // Time between ramp-up steps
	MinSuccessRate   float64 `json:"min_success_rate"`    // Min success rate to continue ramp-up
	AutoRampUp       bool    `json:"auto_ramp_up"`        // Enable automatic ramp-up
	RollbackOnError  bool    `json:"rollback_on_error"`   // Auto-rollback if errors exceed threshold
	ErrorThreshold   float64 `json:"error_threshold"`     // Error rate that triggers rollback
}

// DefaultSplitConfig returns defaults for gradual traffic migration
func DefaultSplitConfig() *SplitConfig {
	return &SplitConfig{
		DestinationPct:   1,      // Start at 1%
		RampUpStepPct:    5,      // 5% increments
		RampUpIntervalMs: 300000, // 5 minutes
		MinSuccessRate:   0.99,
		AutoRampUp:       true,
		RollbackOnError:  true,
		ErrorThreshold:   0.05, // 5% error rate triggers rollback
	}
}

// NewTrafficSplitter creates a new traffic splitter
func NewTrafficSplitter(jobID, tenantID string, config *SplitConfig) *TrafficSplitter {
	if config == nil {
		config = DefaultSplitConfig()
	}
	return &TrafficSplitter{
		jobID:    jobID,
		tenantID: tenantID,
		config:   config,
	}
}

// SplitDecision represents the routing decision for a single delivery
type SplitDecision struct {
	UseDestination bool   `json:"use_destination"`
	Reason         string `json:"reason"`
	CurrentPct     int    `json:"current_pct"`
}

// Route decides whether to route a delivery to the destination (new) platform
func (ts *TrafficSplitter) Route() SplitDecision {
	ts.mu.RLock()
	pct := ts.config.DestinationPct
	ts.mu.RUnlock()

	useNew := rand.Intn(100) < pct

	return SplitDecision{
		UseDestination: useNew,
		Reason:         fmt.Sprintf("traffic split at %d%%", pct),
		CurrentPct:     pct,
	}
}

// RampUp increases the destination percentage by one step
func (ts *TrafficSplitter) RampUp() (int, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	newPct := ts.config.DestinationPct + ts.config.RampUpStepPct
	if newPct > 100 {
		newPct = 100
	}
	ts.config.DestinationPct = newPct
	return newPct, nil
}

// Rollback resets the destination percentage to 0
func (ts *TrafficSplitter) Rollback() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.config.DestinationPct = 0
}

// SetPercentage sets the exact split percentage
func (ts *TrafficSplitter) SetPercentage(pct int) error {
	if pct < 0 || pct > 100 {
		return fmt.Errorf("percentage must be 0-100, got %d", pct)
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.config.DestinationPct = pct
	return nil
}

// GetPercentage returns the current destination percentage
func (ts *TrafficSplitter) GetPercentage() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.config.DestinationPct
}

// IsFullyCutover returns true if 100% of traffic goes to destination
func (ts *TrafficSplitter) IsFullyCutover() bool {
	return ts.GetPercentage() == 100
}

// --- Verification ---

// VerificationReport summarizes migration verification results
type VerificationReport struct {
	JobID             string         `json:"job_id"`
	TotalEndpoints    int            `json:"total_endpoints"`
	VerifiedEndpoints int            `json:"verified_endpoints"`
	FailedEndpoints   int            `json:"failed_endpoints"`
	ShadowMetrics     *ShadowMetrics `json:"shadow_metrics,omitempty"`
	TrafficSplit      int            `json:"traffic_split_pct"`
	ReadyForCutover   bool           `json:"ready_for_cutover"`
	Blockers          []string       `json:"blockers,omitempty"`
	Warnings          []string       `json:"warnings,omitempty"`
	GeneratedAt       time.Time      `json:"generated_at"`
}

// GenerateVerificationReport creates a verification report for a migration job
func GenerateVerificationReport(job *MigrationJob, shadowMetrics *ShadowMetrics, splitPct int, endpointResults []MigrationEndpoint) *VerificationReport {
	report := &VerificationReport{
		JobID:          job.ID,
		TotalEndpoints: job.EndpointCount,
		ShadowMetrics:  shadowMetrics,
		TrafficSplit:   splitPct,
		GeneratedAt:    time.Now(),
	}

	for _, ep := range endpointResults {
		if ep.Status == EndpointStatusValidated || ep.Status == EndpointStatusActive {
			report.VerifiedEndpoints++
		} else if ep.Status == EndpointStatusFailed {
			report.FailedEndpoints++
		}
	}

	// Determine readiness
	report.ReadyForCutover = true

	if report.FailedEndpoints > 0 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("%d endpoint(s) failed verification", report.FailedEndpoints))
		report.ReadyForCutover = false
	}

	if shadowMetrics != nil && shadowMetrics.MatchRate < 0.95 {
		report.Blockers = append(report.Blockers, fmt.Sprintf("shadow match rate %.1f%% is below 95%% threshold", shadowMetrics.MatchRate*100))
		report.ReadyForCutover = false
	}

	if shadowMetrics != nil && shadowMetrics.TotalCompared < 100 {
		report.Warnings = append(report.Warnings, "fewer than 100 shadow comparisons — consider running longer")
	}

	if report.VerifiedEndpoints < report.TotalEndpoints {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d endpoint(s) not yet verified", report.TotalEndpoints-report.VerifiedEndpoints))
	}

	return report
}
