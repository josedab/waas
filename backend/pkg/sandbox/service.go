package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides sandbox management functionality
type Service struct {
	repo Repository
}

// NewService creates a new sandbox service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateSandbox creates a new sandbox environment
func (s *Service) CreateSandbox(ctx context.Context, tenantID string, req *CreateSandboxRequest) (*SandboxEnvironment, error) {
	if req.TTLMinutes < 1 {
		return nil, fmt.Errorf("ttl_minutes must be at least 1")
	}

	rulesJSON, err := json.Marshal(req.MaskingRules)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal masking rules: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(req.TTLMinutes) * time.Minute)

	sandbox := &SandboxEnvironment{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		Status:       StatusActive,
		TargetURL:    req.TargetURL,
		MaskingRules: string(rulesJSON),
		TTLMinutes:   req.TTLMinutes,
		ExpiresAt:    &expiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateSandbox(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	return sandbox, nil
}

// GetSandbox retrieves a sandbox environment by ID
func (s *Service) GetSandbox(ctx context.Context, tenantID, sandboxID string) (*SandboxEnvironment, error) {
	return s.repo.GetSandbox(ctx, tenantID, sandboxID)
}

// ListSandboxes retrieves all sandbox environments for a tenant
func (s *Service) ListSandboxes(ctx context.Context, tenantID string) ([]SandboxEnvironment, error) {
	return s.repo.ListSandboxes(ctx, tenantID)
}

// ReplayEvents replays webhook events in the sandbox with masking applied
func (s *Service) ReplayEvents(ctx context.Context, tenantID, sandboxID string, req *ReplayRequest) ([]ReplaySession, error) {
	sandbox, err := s.repo.GetSandbox(ctx, tenantID, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("sandbox not found: %w", err)
	}

	if sandbox.Status != StatusActive {
		return nil, fmt.Errorf("sandbox is not active, current status: %s", sandbox.Status)
	}

	if sandbox.ExpiresAt != nil && time.Now().After(*sandbox.ExpiresAt) {
		return nil, fmt.Errorf("sandbox has expired")
	}

	var rules []MaskingRule
	if sandbox.MaskingRules != "" {
		if err := json.Unmarshal([]byte(sandbox.MaskingRules), &rules); err != nil {
			return nil, fmt.Errorf("failed to parse masking rules: %w", err)
		}
	}

	var sessions []ReplaySession
	for _, eventID := range req.SourceEventIDs {
		originalPayload := fmt.Sprintf(`{"event_id":"%s","data":"sample_payload"}`, eventID)

		maskedPayload := applyMaskingRules(originalPayload, rules)

		if req.ModifyPayload != nil {
			maskedPayload, err = applyPayloadModifications(maskedPayload, req.ModifyPayload)
			if err != nil {
				return nil, fmt.Errorf("failed to apply payload modifications: %w", err)
			}
		}

		// Simulate response (no actual HTTP call)
		responseLatency := int64(50 + rand.Intn(200))
		responseStatus := 200
		responseBody := fmt.Sprintf(`{"status":"ok","event_id":"%s","sandbox":true}`, eventID)

		session := ReplaySession{
			ID:                uuid.New().String(),
			TenantID:          tenantID,
			SandboxID:         sandboxID,
			SourceEventID:     eventID,
			OriginalPayload:   originalPayload,
			MaskedPayload:     maskedPayload,
			ResponseStatus:    responseStatus,
			ResponseBody:      responseBody,
			ResponseLatencyMs: responseLatency,
			ComparisonResult:  "pending",
			Status:            ReplayStatusReplayed,
			CreatedAt:         time.Now(),
		}

		if err := s.repo.CreateReplaySession(ctx, &session); err != nil {
			session.Status = ReplayStatusFailed
			session.ComparisonResult = fmt.Sprintf("error: %s", err.Error())
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetComparisonReport aggregates replay results for a sandbox
func (s *Service) GetComparisonReport(ctx context.Context, tenantID, sandboxID string) (*ComparisonReport, error) {
	sessions, err := s.repo.GetReplaySessionsForComparison(ctx, tenantID, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get replay sessions: %w", err)
	}

	report := &ComparisonReport{
		SandboxID:    sandboxID,
		TotalReplays: len(sessions),
		Details:      make([]ComparisonDetail, 0),
	}

	var totalLatency int64
	for _, session := range sessions {
		totalLatency += session.ResponseLatencyMs

		switch session.Status {
		case ReplayStatusReplayed:
			if session.ResponseStatus >= 200 && session.ResponseStatus < 300 {
				report.Matches++
			} else {
				report.Mismatches++
				report.Details = append(report.Details, ComparisonDetail{
					EventID:    session.SourceEventID,
					FieldPath:  "response_status",
					Expected:   "2xx",
					Actual:     fmt.Sprintf("%d", session.ResponseStatus),
					IsMismatch: true,
				})
			}
		case ReplayStatusFailed:
			report.Errors++
			report.Details = append(report.Details, ComparisonDetail{
				EventID:    session.SourceEventID,
				FieldPath:  "status",
				Expected:   ReplayStatusReplayed,
				Actual:     ReplayStatusFailed,
				IsMismatch: true,
			})
		}
	}

	if report.TotalReplays > 0 {
		report.AvgLatencyDelta = totalLatency / int64(report.TotalReplays)
	}

	return report, nil
}

// TerminateSandbox terminates a sandbox environment early
func (s *Service) TerminateSandbox(ctx context.Context, tenantID, sandboxID string) (*SandboxEnvironment, error) {
	sandbox, err := s.repo.GetSandbox(ctx, tenantID, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("sandbox not found: %w", err)
	}

	sandbox.Status = StatusTerminated
	sandbox.UpdatedAt = time.Now()

	if err := s.repo.UpdateSandbox(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to terminate sandbox: %w", err)
	}

	return sandbox, nil
}

// CleanupExpired removes all expired sandbox environments for a tenant
func (s *Service) CleanupExpired(ctx context.Context, tenantID string) (int64, error) {
	count, err := s.repo.DeleteExpiredSandboxes(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sandboxes: %w", err)
	}
	return count, nil
}

// applyMaskingRules applies masking rules to a JSON payload string
func applyMaskingRules(payload string, rules []MaskingRule) string {
	if len(rules) == 0 {
		return payload
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return payload
	}

	for _, rule := range rules {
		if val, ok := data[rule.FieldPath]; ok {
			switch rule.Strategy {
			case StrategyRedact:
				data[rule.FieldPath] = "[REDACTED]"
			case StrategyHash:
				hash := sha256.Sum256([]byte(fmt.Sprintf("%v", val)))
				data[rule.FieldPath] = fmt.Sprintf("%x", hash[:8])
			case StrategyFake:
				if rule.Replacement != "" {
					data[rule.FieldPath] = rule.Replacement
				} else {
					data[rule.FieldPath] = "fake_" + rule.FieldPath
				}
			case StrategyPreserve:
				// No change
			}
		}
	}

	masked, err := json.Marshal(data)
	if err != nil {
		return payload
	}
	return string(masked)
}

// applyPayloadModifications merges user-provided modifications into the payload
func applyPayloadModifications(payload string, modifications map[string]interface{}) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return payload, fmt.Errorf("failed to parse payload: %w", err)
	}

	for key, val := range modifications {
		data[key] = val
	}

	modified, err := json.Marshal(data)
	if err != nil {
		return payload, fmt.Errorf("failed to marshal modified payload: %w", err)
	}
	return string(modified), nil
}

// HandleMockRequest processes a request against a mock endpoint with failure injection and request capture
func (s *Service) HandleMockRequest(ctx context.Context, sandboxID, endpointID, method, url string, headers map[string]string, body string) (*CapturedRequest, int, string, error) {
	endpoint, err := s.repo.GetMockEndpoint(ctx, sandboxID, endpointID)
	if err != nil {
		return nil, http.StatusNotFound, "", fmt.Errorf("mock endpoint not found: %w", err)
	}

	captured := &CapturedRequest{
		ID:             uuid.New().String(),
		SandboxID:      sandboxID,
		EndpointID:     endpointID,
		Method:         method,
		URL:            url,
		RequestHeaders: headers,
		RequestBody:    body,
		CapturedAt:     time.Now(),
	}

	// Check failure injection
	injectedFailure, failureType := s.checkFailureInjection(endpoint)
	if injectedFailure {
		captured.FailureInjected = true
		captured.FailureType = failureType
		switch failureType {
		case FailureTimeout:
			captured.ResponseStatus = http.StatusGatewayTimeout
			captured.ResponseBody = `{"error":"gateway timeout (simulated)"}`
			captured.LatencyMs = 30000
		case Failure500Error:
			captured.ResponseStatus = http.StatusInternalServerError
			captured.ResponseBody = `{"error":"internal server error (simulated)"}`
		case FailureRateLimit:
			captured.ResponseStatus = http.StatusTooManyRequests
			captured.ResponseBody = `{"error":"rate limit exceeded (simulated)"}`
		case FailureDNSFailure:
			captured.ResponseStatus = http.StatusBadGateway
			captured.ResponseBody = `{"error":"dns resolution failed (simulated)"}`
		}
	} else {
		// Simulate latency
		if endpoint.Latency != nil {
			latency := SimulateLatency(*endpoint.Latency)
			captured.LatencyMs = int64(latency)
		}
		captured.ResponseStatus = endpoint.ResponseStatus
		captured.ResponseBody = endpoint.ResponseBody
	}

	if err := s.repo.CreateCapturedRequest(ctx, captured); err != nil {
		return nil, 0, "", fmt.Errorf("failed to capture request: %w", err)
	}

	return captured, captured.ResponseStatus, captured.ResponseBody, nil
}

// checkFailureInjection determines if a failure should be injected based on endpoint config
func (s *Service) checkFailureInjection(endpoint *MockEndpointConfig) (bool, string) {
	// Check global failure rate
	if endpoint.FailureRate > 0 && rand.Float64() < endpoint.FailureRate {
		if len(endpoint.Failures) > 0 {
			// Pick a random failure scenario
			f := endpoint.Failures[rand.Intn(len(endpoint.Failures))]
			return true, f.Type
		}
		return true, Failure500Error
	}

	// Check individual failure scenarios
	for _, f := range endpoint.Failures {
		if rand.Float64() < f.Probability {
			return true, f.Type
		}
	}

	return false, ""
}

// SimulateLatency calculates simulated latency based on distribution configuration
func SimulateLatency(config LatencySimulation) int {
	if config.MinMs >= config.MaxMs {
		return config.MinMs
	}

	switch config.DistributionType {
	case DistributionNormal:
		mean := float64(config.MinMs+config.MaxMs) / 2.0
		stddev := float64(config.MaxMs-config.MinMs) / 6.0
		val := rand.NormFloat64()*stddev + mean
		return int(math.Max(float64(config.MinMs), math.Min(float64(config.MaxMs), val)))
	default: // uniform
		return config.MinMs + rand.Intn(config.MaxMs-config.MinMs+1)
	}
}

// GetCapturedRequests retrieves captured requests for a sandbox
func (s *Service) GetCapturedRequests(ctx context.Context, sandboxID string, limit, offset int) ([]CapturedRequest, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListCapturedRequests(ctx, sandboxID, limit, offset)
}

// CreateTestScenario creates a new automated test scenario
func (s *Service) CreateTestScenario(ctx context.Context, tenantID string, req *CreateTestScenarioRequest) (*TestScenario, error) {
	now := time.Now()
	scenario := &TestScenario{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		SandboxID:   req.SandboxID,
		Name:        req.Name,
		Description: req.Description,
		Status:      ScenarioStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	for i, step := range req.Steps {
		step.ID = uuid.New().String()
		step.ScenarioID = scenario.ID
		step.Order = i + 1
		scenario.Steps = append(scenario.Steps, step)
	}

	if err := s.repo.CreateTestScenario(ctx, scenario); err != nil {
		return nil, fmt.Errorf("failed to create test scenario: %w", err)
	}

	return scenario, nil
}

// RunTestScenario executes a test scenario step by step
func (s *Service) RunTestScenario(ctx context.Context, scenarioID string) (*ScenarioResult, error) {
	scenario, err := s.repo.GetTestScenario(ctx, scenarioID)
	if err != nil {
		return nil, fmt.Errorf("scenario not found: %w", err)
	}

	scenario.Status = ScenarioStatusRunning
	scenario.UpdatedAt = time.Now()
	_ = s.repo.UpdateTestScenario(ctx, scenario)

	result := &ScenarioResult{
		ID:         uuid.New().String(),
		ScenarioID: scenarioID,
		Status:     ScenarioStatusRunning,
		TotalSteps: len(scenario.Steps),
		StartedAt:  time.Now(),
		Steps:      make([]StepResult, 0, len(scenario.Steps)),
	}

	for _, step := range scenario.Steps {
		stepResult := s.executeStep(ctx, scenario, step)
		result.Steps = append(result.Steps, stepResult)
		if stepResult.Status == ScenarioStatusFailed {
			result.FailedSteps++
		} else {
			result.PassedSteps++
		}
	}

	result.CompletedAt = time.Now()
	if result.FailedSteps > 0 {
		result.Status = ScenarioStatusFailed
		scenario.Status = ScenarioStatusFailed
	} else {
		result.Status = ScenarioStatusCompleted
		scenario.Status = ScenarioStatusCompleted
	}

	scenario.UpdatedAt = time.Now()
	_ = s.repo.UpdateTestScenario(ctx, scenario)
	_ = s.repo.CreateScenarioResult(ctx, result)

	return result, nil
}

// executeStep runs a single test step and returns the result
func (s *Service) executeStep(ctx context.Context, scenario *TestScenario, step TestStep) StepResult {
	sr := StepResult{
		ID:         uuid.New().String(),
		ScenarioID: scenario.ID,
		StepID:     step.ID,
		StepOrder:  step.Order,
		ExecutedAt: time.Now(),
	}

	switch step.Type {
	case StepSendWebhook:
		captured, _, _, err := s.HandleMockRequest(ctx, scenario.SandboxID, step.EndpointID, "POST", "/webhook", nil, step.Payload)
		if err != nil {
			sr.Status = ScenarioStatusFailed
			sr.ErrorMessage = err.Error()
			return sr
		}
		sr.ActualStatus = captured.ResponseStatus
		sr.ActualBody = captured.ResponseBody
		sr.LatencyMs = captured.LatencyMs

		if step.ExpectedStatus > 0 && captured.ResponseStatus != step.ExpectedStatus {
			sr.Status = ScenarioStatusFailed
			sr.ErrorMessage = fmt.Sprintf("expected status %d, got %d", step.ExpectedStatus, captured.ResponseStatus)
			return sr
		}
		if step.ExpectedBody != "" && !strings.Contains(captured.ResponseBody, step.ExpectedBody) {
			sr.Status = ScenarioStatusFailed
			sr.ErrorMessage = fmt.Sprintf("response body does not contain expected content")
			return sr
		}
		sr.Status = ScenarioStatusCompleted

	case StepWait:
		if step.WaitMs > 0 {
			time.Sleep(time.Duration(step.WaitMs) * time.Millisecond)
		}
		sr.Status = ScenarioStatusCompleted

	case StepAssertDelivery:
		requests, err := s.repo.ListCapturedRequests(ctx, scenario.SandboxID, 100, 0)
		if err != nil {
			sr.Status = ScenarioStatusFailed
			sr.ErrorMessage = fmt.Sprintf("failed to list captured requests: %s", err.Error())
			return sr
		}
		if len(requests) == 0 {
			sr.Status = ScenarioStatusFailed
			sr.ErrorMessage = "no captured requests found"
			return sr
		}
		sr.ActualStatus = requests[len(requests)-1].ResponseStatus
		sr.Status = ScenarioStatusCompleted

	default:
		sr.Status = ScenarioStatusFailed
		sr.ErrorMessage = fmt.Sprintf("unknown step type: %s", step.Type)
	}

	return sr
}

// GetScenarioResults retrieves results for a test scenario
func (s *Service) GetScenarioResults(ctx context.Context, scenarioID string) (*ScenarioResult, error) {
	return s.repo.GetScenarioResult(ctx, scenarioID)
}

// InjectChaos adds failure injection dynamically to a sandbox mock endpoint
func (s *Service) InjectChaos(ctx context.Context, sandboxID, failureType string, probability float64) error {
	endpoints, err := s.repo.ListMockEndpoints(ctx, sandboxID)
	if err != nil {
		return fmt.Errorf("failed to list mock endpoints: %w", err)
	}

	failure := FailureScenario{
		Name:        fmt.Sprintf("chaos_%s", failureType),
		Type:        failureType,
		Probability: probability,
	}

	for i := range endpoints {
		endpoints[i].Failures = append(endpoints[i].Failures, failure)
		endpoints[i].UpdatedAt = time.Now()
		if err := s.repo.UpdateMockEndpoint(ctx, &endpoints[i]); err != nil {
			return fmt.Errorf("failed to update endpoint %s: %w", endpoints[i].ID, err)
		}
	}

	return nil
}

// ListTestScenarios retrieves all test scenarios for a tenant
func (s *Service) ListTestScenarios(ctx context.Context, tenantID string) ([]TestScenario, error) {
	return s.repo.ListTestScenarios(ctx, tenantID)
}

// CreateMockEndpoint creates a new mock endpoint config for a sandbox
func (s *Service) CreateMockEndpoint(ctx context.Context, sandboxID string, req *CreateMockEndpointRequest) (*MockEndpointConfig, error) {
	now := time.Now()
	endpoint := &MockEndpointConfig{
		ID:              uuid.New().String(),
		SandboxID:       sandboxID,
		Path:            req.Path,
		Method:          req.Method,
		ResponseStatus:  req.ResponseStatus,
		ResponseBody:    req.ResponseBody,
		ResponseHeaders: req.ResponseHeaders,
		Latency:         req.Latency,
		FailureRate:     req.FailureRate,
		Failures:        req.Failures,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateMockEndpoint(ctx, endpoint); err != nil {
		return nil, fmt.Errorf("failed to create mock endpoint: %w", err)
	}

	return endpoint, nil
}
