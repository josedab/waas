package autoremediation

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// FailureClassifier classifies delivery failures using ML-based pattern matching
// and produces remediation recommendations based on historical outcomes.
type FailureClassifier struct {
	patterns     map[string]*ClassifiedPattern
	outcomes     []RemediationOutcome
	mu           sync.RWMutex
	config       *ClassifierConfig
}

// ClassifierConfig configures the failure classifier
type ClassifierConfig struct {
	MinSamplesForClassification int     `json:"min_samples"`
	ConfidenceThreshold         float64 `json:"confidence_threshold"`
	DecayFactor                 float64 `json:"decay_factor"`
	MaxPatterns                 int     `json:"max_patterns"`
	LearningEnabled             bool    `json:"learning_enabled"`
}

// DefaultClassifierConfig returns sensible defaults
func DefaultClassifierConfig() *ClassifierConfig {
	return &ClassifierConfig{
		MinSamplesForClassification: 5,
		ConfidenceThreshold:         0.7,
		DecayFactor:                 0.95,
		MaxPatterns:                 1000,
		LearningEnabled:             true,
	}
}

// FailureCategory represents a classified failure type
type FailureCategory string

const (
	FailureCategoryTransient   FailureCategory = "transient"   // Temporary, will self-resolve
	FailureCategoryEndpoint    FailureCategory = "endpoint"    // Endpoint is down/misconfigured
	FailureCategoryPayload     FailureCategory = "payload"     // Payload format/content issue
	FailureCategoryRateLimit   FailureCategory = "rate_limit"  // Endpoint rate limiting
	FailureCategoryTimeout     FailureCategory = "timeout"     // Endpoint too slow
	FailureCategoryCertificate FailureCategory = "certificate" // TLS/SSL issue
	FailureCategoryDNS         FailureCategory = "dns"         // DNS resolution failure
	FailureCategoryAuth        FailureCategory = "auth"        // Authentication/authorization failure
	FailureCategoryUnknown     FailureCategory = "unknown"
)

// ClassifiedPattern represents a failure pattern with classification metadata
type ClassifiedPattern struct {
	PatternKey     string            `json:"pattern_key"`
	Category       FailureCategory   `json:"category"`
	SubCategory    string            `json:"sub_category,omitempty"`
	StatusCode     int               `json:"status_code"`
	ErrorSignature string            `json:"error_signature"`
	Occurrences    int               `json:"occurrences"`
	SuccessRate    float64           `json:"success_rate_after_remediation"`
	BestAction     string            `json:"best_action"`
	BestConfig     string            `json:"best_config"`
	Confidence     float64           `json:"confidence"`
	Features       map[string]float64 `json:"features"`
	LastSeenAt     time.Time         `json:"last_seen_at"`
}

// FailureSignal represents a single failure event for classification
type FailureSignal struct {
	EndpointID   string            `json:"endpoint_id"`
	TenantID     string            `json:"tenant_id"`
	StatusCode   int               `json:"status_code"`
	ErrorMessage string            `json:"error_message"`
	LatencyMs    float64           `json:"latency_ms"`
	AttemptNum   int               `json:"attempt_num"`
	Headers      map[string]string `json:"headers,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// ClassificationResult is the output of failure classification
type ClassificationResult struct {
	Category        FailureCategory   `json:"category"`
	SubCategory     string            `json:"sub_category,omitempty"`
	Confidence      float64           `json:"confidence"`
	IsRetryable     bool              `json:"is_retryable"`
	SuggestedAction *SuggestedAction  `json:"suggested_action"`
	SimilarPatterns []string          `json:"similar_patterns,omitempty"`
	Reasoning       string            `json:"reasoning"`
}

// SuggestedAction recommends a remediation action
type SuggestedAction struct {
	Type            string  `json:"type"` // retry_adjust, backoff, disable, alert, transform
	Config          string  `json:"config"`
	Confidence      float64 `json:"confidence"`
	EstimatedImpact string  `json:"estimated_impact"`
	AutoApply       bool    `json:"auto_apply"`
}

// RemediationOutcome records the outcome of a remediation for learning
type RemediationOutcome struct {
	PatternKey   string    `json:"pattern_key"`
	ActionType   string    `json:"action_type"`
	ActionConfig string    `json:"action_config"`
	Success      bool      `json:"success"`
	ImprovementPct float64 `json:"improvement_pct"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewFailureClassifier creates a new ML-based failure classifier
func NewFailureClassifier(config *ClassifierConfig) *FailureClassifier {
	if config == nil {
		config = DefaultClassifierConfig()
	}
	return &FailureClassifier{
		patterns: make(map[string]*ClassifiedPattern),
		outcomes: make([]RemediationOutcome, 0),
		config:   config,
	}
}

// Classify classifies a failure signal and returns a remediation recommendation
func (fc *FailureClassifier) Classify(signal *FailureSignal) *ClassificationResult {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	category := fc.classifyByRules(signal)
	confidence := fc.computeConfidence(signal, category)

	result := &ClassificationResult{
		Category:    category,
		Confidence:  confidence,
		IsRetryable: isRetryable(category),
		Reasoning:   fc.generateReasoning(signal, category),
	}

	// Look for similar known patterns
	patternKey := fc.computePatternKey(signal)
	if known, exists := fc.patterns[patternKey]; exists {
		result.Confidence = math.Max(result.Confidence, known.Confidence)
		if known.BestAction != "" {
			result.SuggestedAction = &SuggestedAction{
				Type:       known.BestAction,
				Config:     known.BestConfig,
				Confidence: known.SuccessRate,
				AutoApply:  known.SuccessRate > 0.9 && known.Occurrences > 10,
			}
		}
	} else {
		result.SuggestedAction = fc.defaultAction(category, signal)
	}

	// Find similar patterns
	result.SimilarPatterns = fc.findSimilarPatterns(signal, 3)

	return result
}

// RecordOutcome records a remediation outcome for the learning loop
func (fc *FailureClassifier) RecordOutcome(outcome RemediationOutcome) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.outcomes = append(fc.outcomes, outcome)

	// Update pattern with outcome data
	pattern, exists := fc.patterns[outcome.PatternKey]
	if !exists {
		return
	}

	pattern.Occurrences++
	if outcome.Success {
		// Increase confidence in this action
		successWeight := 1.0 / float64(pattern.Occurrences)
		pattern.SuccessRate = pattern.SuccessRate*(1-successWeight) + successWeight
		pattern.BestAction = outcome.ActionType
		pattern.BestConfig = outcome.ActionConfig
	} else {
		successWeight := 1.0 / float64(pattern.Occurrences)
		pattern.SuccessRate = pattern.SuccessRate * (1 - successWeight)
	}

	pattern.Confidence = math.Min(1.0, float64(pattern.Occurrences)/20.0) * pattern.SuccessRate
}

// LearnFromSignal records a failure signal for pattern learning
func (fc *FailureClassifier) LearnFromSignal(signal *FailureSignal) {
	if !fc.config.LearningEnabled {
		return
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	key := fc.computePatternKey(signal)
	pattern, exists := fc.patterns[key]
	if !exists {
		category := fc.classifyByRules(signal)
		pattern = &ClassifiedPattern{
			PatternKey:     key,
			Category:       category,
			StatusCode:     signal.StatusCode,
			ErrorSignature: fc.extractErrorSignature(signal.ErrorMessage),
			Features:       make(map[string]float64),
		}
		fc.patterns[key] = pattern
	}

	pattern.Occurrences++
	pattern.LastSeenAt = signal.Timestamp

	// Update feature vector
	pattern.Features["status_code"] = float64(signal.StatusCode)
	pattern.Features["latency_ms"] = signal.LatencyMs
	pattern.Features["attempt_num"] = float64(signal.AttemptNum)

	// Prune old patterns if over limit
	if len(fc.patterns) > fc.config.MaxPatterns {
		fc.pruneOldPatterns()
	}
}

func (fc *FailureClassifier) classifyByRules(signal *FailureSignal) FailureCategory {
	errLower := strings.ToLower(signal.ErrorMessage)

	// Status code-based classification
	switch {
	case signal.StatusCode == 429:
		return FailureCategoryRateLimit
	case signal.StatusCode == 401 || signal.StatusCode == 403:
		return FailureCategoryAuth
	case signal.StatusCode == 400 || signal.StatusCode == 422:
		return FailureCategoryPayload
	case signal.StatusCode >= 500 && signal.StatusCode < 600:
		if signal.StatusCode == 503 || signal.StatusCode == 502 {
			return FailureCategoryTransient
		}
		return FailureCategoryEndpoint
	case signal.StatusCode == 0 && signal.LatencyMs > 30000:
		return FailureCategoryTimeout
	}

	// Error message-based classification
	switch {
	case strings.Contains(errLower, "timeout") || strings.Contains(errLower, "deadline exceeded"):
		return FailureCategoryTimeout
	case strings.Contains(errLower, "dns") || strings.Contains(errLower, "no such host"):
		return FailureCategoryDNS
	case strings.Contains(errLower, "certificate") || strings.Contains(errLower, "tls") || strings.Contains(errLower, "x509"):
		return FailureCategoryCertificate
	case strings.Contains(errLower, "connection refused") || strings.Contains(errLower, "connection reset"):
		return FailureCategoryEndpoint
	case strings.Contains(errLower, "rate limit") || strings.Contains(errLower, "throttl"):
		return FailureCategoryRateLimit
	case strings.Contains(errLower, "unauthorized") || strings.Contains(errLower, "forbidden"):
		return FailureCategoryAuth
	}

	return FailureCategoryUnknown
}

func (fc *FailureClassifier) computeConfidence(signal *FailureSignal, category FailureCategory) float64 {
	base := 0.5

	// Higher confidence for well-known status codes
	if signal.StatusCode == 429 || signal.StatusCode == 401 || signal.StatusCode == 403 {
		base = 0.95
	} else if signal.StatusCode >= 500 {
		base = 0.85
	} else if signal.StatusCode == 400 || signal.StatusCode == 422 {
		base = 0.90
	}

	// Boost for error message match
	if category != FailureCategoryUnknown {
		base = math.Min(1.0, base+0.1)
	}

	return base
}

func isRetryable(category FailureCategory) bool {
	switch category {
	case FailureCategoryTransient, FailureCategoryRateLimit, FailureCategoryTimeout:
		return true
	case FailureCategoryEndpoint:
		return true // May come back
	default:
		return false
	}
}

func (fc *FailureClassifier) generateReasoning(signal *FailureSignal, category FailureCategory) string {
	switch category {
	case FailureCategoryTransient:
		return "Temporary server error; endpoint likely experiencing intermittent issues"
	case FailureCategoryRateLimit:
		return "Endpoint is rate limiting requests; reduce delivery rate"
	case FailureCategoryTimeout:
		return "Endpoint is too slow to respond; consider increasing timeout or reducing payload size"
	case FailureCategoryPayload:
		return "Endpoint rejected the payload; check schema compatibility"
	case FailureCategoryAuth:
		return "Authentication/authorization failure; verify API credentials"
	case FailureCategoryDNS:
		return "DNS resolution failed; endpoint hostname may be incorrect or DNS outage"
	case FailureCategoryCertificate:
		return "TLS certificate error; endpoint certificate may be expired or invalid"
	case FailureCategoryEndpoint:
		return "Endpoint is unreachable or returning server errors"
	default:
		return "Unable to determine root cause; manual investigation recommended"
	}
}

func (fc *FailureClassifier) defaultAction(category FailureCategory, signal *FailureSignal) *SuggestedAction {
	switch category {
	case FailureCategoryTransient:
		return &SuggestedAction{
			Type:            ActionTypeRetryStrategyChange,
			Config:          `{"max_retries": 5, "backoff_factor": 2, "initial_delay_ms": 1000}`,
			Confidence:      0.8,
			EstimatedImpact: "high - transient errors typically resolve with retries",
			AutoApply:       true,
		}
	case FailureCategoryRateLimit:
		return &SuggestedAction{
			Type:            ActionTypeRetryStrategyChange,
			Config:          `{"max_retries": 10, "backoff_factor": 3, "initial_delay_ms": 5000, "respect_retry_after": true}`,
			Confidence:      0.9,
			EstimatedImpact: "high - backing off allows rate limit window to reset",
			AutoApply:       true,
		}
	case FailureCategoryTimeout:
		return &SuggestedAction{
			Type:            ActionTypeRetryStrategyChange,
			Config:          `{"timeout_ms": 60000, "max_retries": 3}`,
			Confidence:      0.7,
			EstimatedImpact: "medium - longer timeout may allow slow endpoints to respond",
			AutoApply:       false,
		}
	case FailureCategoryPayload:
		return &SuggestedAction{
			Type:            ActionTypeAlert,
			Config:          `{"severity": "warning", "message": "payload rejected by endpoint"}`,
			Confidence:      0.85,
			EstimatedImpact: "low - requires payload format investigation",
			AutoApply:       false,
		}
	case FailureCategoryEndpoint:
		return &SuggestedAction{
			Type:            ActionTypeEndpointDisable,
			Config:          `{"disable_after_failures": 50, "check_interval_minutes": 5}`,
			Confidence:      0.6,
			EstimatedImpact: "medium - prevents wasted delivery attempts",
			AutoApply:       false,
		}
	default:
		return &SuggestedAction{
			Type:            ActionTypeAlert,
			Config:          `{"severity": "info", "message": "unclassified failure"}`,
			Confidence:      0.3,
			EstimatedImpact: "unknown - requires manual investigation",
			AutoApply:       false,
		}
	}
}

func (fc *FailureClassifier) computePatternKey(signal *FailureSignal) string {
	sig := fc.extractErrorSignature(signal.ErrorMessage)
	return strings.Join([]string{
		signal.EndpointID,
		string(rune(signal.StatusCode)),
		sig,
	}, ":")
}

func (fc *FailureClassifier) extractErrorSignature(errorMsg string) string {
	msg := strings.ToLower(errorMsg)
	// Normalize dynamic parts
	for _, token := range []string{"timeout", "connection refused", "rate limit", "tls", "dns", "certificate", "unauthorized"} {
		if strings.Contains(msg, token) {
			return token
		}
	}
	// Truncate to first 50 chars as signature
	if len(msg) > 50 {
		return msg[:50]
	}
	return msg
}

func (fc *FailureClassifier) findSimilarPatterns(signal *FailureSignal, maxResults int) []string {
	var similar []string
	targetCategory := fc.classifyByRules(signal)

	type scored struct {
		key   string
		score float64
	}
	var scored_ []scored

	for key, pattern := range fc.patterns {
		score := 0.0
		if pattern.Category == targetCategory {
			score += 0.5
		}
		if pattern.StatusCode == signal.StatusCode {
			score += 0.3
		}
		if pattern.ErrorSignature == fc.extractErrorSignature(signal.ErrorMessage) {
			score += 0.2
		}
		if score > 0.3 {
			scored_ = append(scored_, scored{key: key, score: score})
		}
	}

	sort.Slice(scored_, func(i, j int) bool {
		return scored_[i].score > scored_[j].score
	})

	for i, s := range scored_ {
		if i >= maxResults {
			break
		}
		similar = append(similar, s.key)
	}

	return similar
}

func (fc *FailureClassifier) pruneOldPatterns() {
	// Remove least-recently-seen patterns
	type entry struct {
		key  string
		time time.Time
	}
	var entries []entry
	for k, p := range fc.patterns {
		entries = append(entries, entry{key: k, time: p.LastSeenAt})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].time.Before(entries[j].time)
	})

	toRemove := len(fc.patterns) - fc.config.MaxPatterns + fc.config.MaxPatterns/10
	for i := 0; i < toRemove && i < len(entries); i++ {
		delete(fc.patterns, entries[i].key)
	}
}

// GetStats returns classifier statistics
func (fc *FailureClassifier) GetStats() map[string]interface{} {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	categoryCounts := make(map[FailureCategory]int)
	for _, p := range fc.patterns {
		categoryCounts[p.Category]++
	}

	totalOutcomes := len(fc.outcomes)
	successfulOutcomes := 0
	for _, o := range fc.outcomes {
		if o.Success {
			successfulOutcomes++
		}
	}

	successRate := 0.0
	if totalOutcomes > 0 {
		successRate = float64(successfulOutcomes) / float64(totalOutcomes)
	}

	return map[string]interface{}{
		"total_patterns":          len(fc.patterns),
		"category_distribution":  categoryCounts,
		"total_outcomes":         totalOutcomes,
		"remediation_success_rate": successRate,
		"learning_enabled":       fc.config.LearningEnabled,
	}
}
