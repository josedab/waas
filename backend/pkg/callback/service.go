package callback

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/josedab/waas/pkg/utils"
)

// ServiceConfig holds configuration for the callback service
type ServiceConfig struct {
	DefaultTimeoutMs int
	DefaultTTL       int // seconds
	MaxTimeoutMs     int
	PollIntervalMs   int
}

// DefaultServiceConfig returns sensible defaults
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		DefaultTimeoutMs: 30000,
		DefaultTTL:       3600,
		MaxTimeoutMs:     300000,
		PollIntervalMs:   500,
	}
}

// Service provides callback business logic
type Service struct {
	repo   Repository
	logger *utils.Logger
	config ServiceConfig
}

// NewService creates a new callback service
func NewService(repo Repository) *Service {
	return &Service{
		repo:   repo,
		logger: utils.NewLogger("callback-service"),
		config: DefaultServiceConfig(),
	}
}

// NewServiceWithConfig creates a new callback service with custom configuration
func NewServiceWithConfig(repo Repository, logger *utils.Logger, config ServiceConfig) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
		config: config,
	}
}

// SendWithCallback sends a webhook and creates a callback expectation with a correlation ID
func (s *Service) SendWithCallback(ctx context.Context, tenantID uuid.UUID, req *CreateCallbackRequest) (*CallbackRequest, error) {
	webhookID, err := uuid.Parse(req.WebhookID)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook ID: %w", err)
	}

	timeoutMs := req.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = s.config.DefaultTimeoutMs
	}
	if timeoutMs > s.config.MaxTimeoutMs {
		timeoutMs = s.config.MaxTimeoutMs
	}

	now := time.Now()
	correlationID := uuid.New().String()

	cbReq := &CallbackRequest{
		ID:            uuid.New(),
		TenantID:      tenantID,
		WebhookID:     webhookID,
		CorrelationID: correlationID,
		CallbackURL:   req.CallbackURL,
		Payload:       req.Payload,
		Headers:       req.Headers,
		TimeoutMs:     timeoutMs,
		Status:        CallbackStatusPending,
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Duration(timeoutMs) * time.Millisecond),
	}

	if err := s.repo.CreateCallbackRequest(ctx, cbReq); err != nil {
		return nil, fmt.Errorf("failed to create callback request: %w", err)
	}

	// Create correlation entry for tracking
	correlation := &CorrelationEntry{
		ID:            uuid.New(),
		CorrelationID: correlationID,
		TenantID:      tenantID,
		RequestID:     cbReq.ID,
		Status:        CallbackStatusPending,
		CreatedAt:     now,
		TTL:           s.config.DefaultTTL,
	}

	if err := s.repo.CreateCorrelation(ctx, correlation); err != nil {
		return nil, fmt.Errorf("failed to create correlation entry: %w", err)
	}

	return cbReq, nil
}

// ReceiveCallback matches a response to a request via correlation ID
func (s *Service) ReceiveCallback(ctx context.Context, correlationID string, statusCode int, body, headers []byte) (*CallbackResponse, error) {
	correlation, err := s.repo.GetCorrelation(ctx, correlationID)
	if err != nil {
		return nil, fmt.Errorf("correlation not found: %w", err)
	}

	if correlation.Status == CallbackStatusReceived {
		return nil, fmt.Errorf("callback already received for correlation %s", correlationID)
	}

	if correlation.Status == CallbackStatusTimeout || correlation.Status == CallbackStatusCancelled {
		return nil, fmt.Errorf("callback %s is no longer active (status: %s)", correlationID, correlation.Status)
	}

	now := time.Now()
	latencyMs := now.Sub(correlation.CreatedAt).Milliseconds()

	resp := &CallbackResponse{
		ID:            uuid.New(),
		RequestID:     correlation.RequestID,
		CorrelationID: correlationID,
		StatusCode:    statusCode,
		Body:          body,
		Headers:       headers,
		ReceivedAt:    now,
		LatencyMs:     latencyMs,
	}

	if err := s.repo.SaveCallbackResponse(ctx, resp); err != nil {
		return nil, fmt.Errorf("failed to save callback response: %w", err)
	}

	// Update correlation with response
	respID := resp.ID
	correlation.ResponseID = &respID
	correlation.Status = CallbackStatusReceived
	if err := s.repo.UpdateCorrelation(ctx, correlation); err != nil {
		return nil, fmt.Errorf("failed to update correlation: %w", err)
	}

	// Update request status
	if err := s.repo.UpdateCallbackStatus(ctx, correlation.RequestID, CallbackStatusReceived); err != nil {
		return nil, fmt.Errorf("failed to update request status: %w", err)
	}

	return resp, nil
}

// WaitForCallback blocks until a response is received or timeout occurs
func (s *Service) WaitForCallback(ctx context.Context, correlationID string, timeoutMs int) (*CallbackResponse, error) {
	if timeoutMs <= 0 {
		timeoutMs = s.config.DefaultTimeoutMs
	}
	if timeoutMs > s.config.MaxTimeoutMs {
		timeoutMs = s.config.MaxTimeoutMs
	}

	deadline := time.Duration(timeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	// Update status to waiting
	correlation, err := s.repo.GetCorrelation(ctx, correlationID)
	if err != nil {
		return nil, fmt.Errorf("correlation not found: %w", err)
	}

	if correlation.Status == CallbackStatusReceived {
		return s.repo.GetResponseByCorrelation(ctx, correlationID)
	}

	correlation.Status = CallbackStatusWaiting
	if err := s.repo.UpdateCorrelation(ctx, correlation); err != nil {
		s.logger.Warn("Failed to update correlation status", map[string]interface{}{"error": err.Error(), "correlation_id": correlationID})
	}
	if err := s.repo.UpdateCallbackStatus(ctx, correlation.RequestID, CallbackStatusWaiting); err != nil {
		s.logger.Warn("Failed to update callback status", map[string]interface{}{"error": err.Error(), "request_id": correlation.RequestID})
	}

	pollInterval := time.Duration(s.config.PollIntervalMs) * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Timeout: update status
			if err := s.repo.UpdateCallbackStatus(ctx, correlation.RequestID, CallbackStatusTimeout); err != nil {
				s.logger.Warn("Failed to update callback status on timeout", map[string]interface{}{"error": err.Error(), "request_id": correlation.RequestID})
			}
			correlation.Status = CallbackStatusTimeout
			updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.repo.UpdateCorrelation(updateCtx, correlation); err != nil {
				s.logger.Warn("Failed to update correlation on timeout", map[string]interface{}{"error": err.Error(), "correlation_id": correlationID})
			}
			return nil, fmt.Errorf("timeout waiting for callback response (correlation: %s)", correlationID)
		case <-ticker.C:
			resp, err := s.repo.GetResponseByCorrelation(ctx, correlationID)
			if err == nil && resp != nil {
				return resp, nil
			}
		}
	}
}

// GetCallbackRequest retrieves a callback request by ID
func (s *Service) GetCallbackRequest(ctx context.Context, tenantID, requestID uuid.UUID) (*CallbackRequest, error) {
	return s.repo.GetCallbackRequest(ctx, tenantID, requestID)
}

// ListCallbackRequests lists all callback requests for a tenant
func (s *Service) ListCallbackRequests(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackRequest, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListCallbackRequests(ctx, tenantID, limit, offset)
}

// CreateLongPollSession creates a new long-polling session
func (s *Service) CreateLongPollSession(ctx context.Context, tenantID uuid.UUID, req *CreateLongPollRequest) (*LongPollSession, error) {
	endpointID, err := uuid.Parse(req.EndpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint ID: %w", err)
	}

	timeoutMs := req.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = s.config.DefaultTimeoutMs
	}

	session := &LongPollSession{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EndpointID: endpointID,
		Filters:    req.Filters,
		TimeoutMs:  timeoutMs,
		Status:     LongPollStatusActive,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateLongPollSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create long-poll session: %w", err)
	}

	return session, nil
}

// PollForEvents returns pending callback responses for a long-poll session or waits until timeout
func (s *Service) PollForEvents(ctx context.Context, sessionID uuid.UUID) ([]CallbackResponse, error) {
	session, err := s.repo.GetLongPollSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if session.Status != LongPollStatusActive {
		return nil, fmt.Errorf("session is not active (status: %s)", session.Status)
	}

	deadline := time.Duration(session.TimeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	pollInterval := time.Duration(s.config.PollIntervalMs) * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return []CallbackResponse{}, nil
		case <-ticker.C:
			// Check for pending callback requests for this tenant/endpoint
			requests, _, err := s.repo.ListCallbackRequests(ctx, session.TenantID, 100, 0)
			if err != nil {
				continue
			}

			var responses []CallbackResponse
			for _, req := range requests {
				if req.Status == CallbackStatusReceived {
					resp, err := s.repo.GetResponseByCorrelation(ctx, req.CorrelationID)
					if err == nil && resp != nil {
						responses = append(responses, *resp)
					}
				}
			}

			if len(responses) > 0 {
				return responses, nil
			}
		}
	}
}

// RegisterPattern registers a reusable callback pattern
func (s *Service) RegisterPattern(ctx context.Context, tenantID uuid.UUID, req *RegisterPatternRequest) (*CallbackPattern, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("pattern name is required")
	}

	timeoutMs := req.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = s.config.DefaultTimeoutMs
	}

	maxRetries := req.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	pattern := &CallbackPattern{
		ID:                     uuid.New(),
		TenantID:               tenantID,
		Name:                   req.Name,
		Description:            req.Description,
		RequestTemplate:        req.RequestTemplate,
		ExpectedResponseSchema: req.ExpectedResponseSchema,
		TimeoutMs:              timeoutMs,
		MaxRetries:             maxRetries,
		CreatedAt:              time.Now(),
	}

	if err := s.repo.SavePattern(ctx, pattern); err != nil {
		return nil, fmt.Errorf("failed to save pattern: %w", err)
	}

	return pattern, nil
}

// GetPatterns lists all callback patterns for a tenant
func (s *Service) GetPatterns(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackPattern, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetPatterns(ctx, tenantID, limit, offset)
}

// GetMetrics retrieves callback metrics for a tenant
func (s *Service) GetMetrics(ctx context.Context, tenantID uuid.UUID) (*CallbackMetrics, error) {
	return s.repo.GetCallbackMetrics(ctx, tenantID)
}

// GetPendingCallbacks retrieves callback requests in pending or waiting status
func (s *Service) GetPendingCallbacks(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CallbackRequest, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListCallbackRequests(ctx, tenantID, limit, offset)
}
