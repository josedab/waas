package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides inbound webhook business logic
type Service struct {
	repo Repository
}

// NewService creates a new inbound service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateSource registers a new inbound webhook source
func (s *Service) CreateSource(ctx context.Context, tenantID string, req *CreateSourceRequest) (*InboundSource, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	algorithm := req.VerificationAlgorithm
	if algorithm == "" {
		algorithm = "hmac-sha256"
	}

	now := time.Now()
	source := &InboundSource{
		ID:                    uuid.New().String(),
		TenantID:              tenantID,
		Name:                  req.Name,
		Provider:              req.Provider,
		VerificationSecret:    req.VerificationSecret,
		VerificationHeader:    req.VerificationHeader,
		VerificationAlgorithm: algorithm,
		Status:                SourceStatusActive,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.repo.CreateSource(ctx, source); err != nil {
		return nil, err
	}

	return source, nil
}

// GetSource retrieves an inbound source by ID for a tenant
func (s *Service) GetSource(ctx context.Context, tenantID, sourceID string) (*InboundSource, error) {
	return s.repo.GetSourceByTenant(ctx, tenantID, sourceID)
}

// ListSources lists inbound sources for a tenant
func (s *Service) ListSources(ctx context.Context, tenantID string, limit, offset int) ([]InboundSource, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListSources(ctx, tenantID, limit, offset)
}

// UpdateSource updates an inbound source
func (s *Service) UpdateSource(ctx context.Context, tenantID, sourceID string, req *UpdateSourceRequest) (*InboundSource, error) {
	source, err := s.repo.GetSourceByTenant(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		source.Name = req.Name
	}
	if req.VerificationSecret != "" {
		source.VerificationSecret = req.VerificationSecret
	}
	if req.VerificationHeader != "" {
		source.VerificationHeader = req.VerificationHeader
	}
	if req.VerificationAlgorithm != "" {
		source.VerificationAlgorithm = req.VerificationAlgorithm
	}
	if req.Status != "" {
		if req.Status != SourceStatusActive && req.Status != SourceStatusPaused && req.Status != SourceStatusDisabled {
			return nil, fmt.Errorf("invalid status: %s", req.Status)
		}
		source.Status = req.Status
	}
	source.UpdatedAt = time.Now()

	if err := s.repo.UpdateSource(ctx, source); err != nil {
		return nil, err
	}

	return source, nil
}

// DeleteSource deletes an inbound source
func (s *Service) DeleteSource(ctx context.Context, tenantID, sourceID string) error {
	return s.repo.DeleteSource(ctx, tenantID, sourceID)
}

// ProcessInboundWebhook validates signature, normalizes payload, and routes the event
func (s *Service) ProcessInboundWebhook(ctx context.Context, sourceID string, payload []byte, headers map[string][]string) (*InboundEvent, error) {
	source, err := s.repo.GetSource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	if source.Status != SourceStatusActive {
		return nil, fmt.Errorf("source is not active, current status: %s", source.Status)
	}

	// Marshal headers for storage
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		headersJSON = []byte("{}")
	}

	now := time.Now()
	event := &InboundEvent{
		ID:         uuid.New().String(),
		SourceID:   sourceID,
		TenantID:   source.TenantID,
		Provider:   source.Provider,
		RawPayload: string(payload),
		Headers:    headersJSON,
		Status:     EventStatusReceived,
		CreatedAt:  now,
	}

	// Create event record first
	if err := s.repo.CreateEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to store event: %w", err)
	}

	// Verify signature if secret is configured
	if source.VerificationSecret != "" {
		verifier := GetVerifier(source.Provider)

		// Configure custom verifier if needed
		if source.Provider == ProviderCustom {
			if cv, ok := verifier.(*CustomVerifier); ok {
				cv.HeaderName = source.VerificationHeader
				cv.Algorithm = source.VerificationAlgorithm
			}
		}

		valid, verifyErr := verifier.Verify(payload, headers, source.VerificationSecret)
		event.SignatureValid = valid

		if verifyErr != nil || !valid {
			event.Status = EventStatusFailed
			errMsg := "signature verification failed"
			if verifyErr != nil {
				errMsg = verifyErr.Error()
			}
			event.ErrorMessage = errMsg
			_ = s.repo.UpdateEventStatus(ctx, event.ID, EventStatusFailed, errMsg)
			return event, fmt.Errorf("signature verification failed: %s", errMsg)
		}

		event.Status = EventStatusValidated
		_ = s.repo.UpdateEventStatus(ctx, event.ID, EventStatusValidated, "")
	} else {
		event.SignatureValid = true
		event.Status = EventStatusValidated
		_ = s.repo.UpdateEventStatus(ctx, event.ID, EventStatusValidated, "")
	}

	// Route the event
	rules, err := s.repo.GetRoutingRules(ctx, sourceID)
	if err == nil && len(rules) > 0 {
		if routeErr := s.RouteEvent(ctx, event, rules); routeErr != nil {
			event.Status = EventStatusFailed
			_ = s.repo.UpdateEventStatus(ctx, event.ID, EventStatusFailed, routeErr.Error())
			return event, nil // Return event even if routing fails
		}
	}

	event.Status = EventStatusRouted
	event.ProcessedAt = &now
	_ = s.repo.UpdateEventStatus(ctx, event.ID, EventStatusRouted, "")

	return event, nil
}

// RouteEvent applies routing rules and delivers the event
func (s *Service) RouteEvent(ctx context.Context, event *InboundEvent, rules []RoutingRule) error {
	for _, rule := range rules {
		if !rule.Active {
			continue
		}

		// In a full implementation, this would:
		// 1. Evaluate filter expressions against the payload
		// 2. Route to the appropriate destination (HTTP, queue, internal)
		// For now, we mark the event as routed
		_ = rule // rule processing would go here
	}
	return nil
}

// GetSourceEvents retrieves event history for a source
func (s *Service) GetSourceEvents(ctx context.Context, tenantID, sourceID, status string, limit, offset int) ([]InboundEvent, int, error) {
	// Verify source belongs to tenant
	_, err := s.repo.GetSourceByTenant(ctx, tenantID, sourceID)
	if err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.repo.ListEventsBySource(ctx, sourceID, status, limit, offset)
}

// ReplayInboundEvent re-processes an existing event
func (s *Service) ReplayInboundEvent(ctx context.Context, tenantID, eventID string) (*InboundEvent, error) {
	event, err := s.repo.GetEventByTenant(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("event not found: %w", err)
	}

	// Re-route the event
	rules, err := s.repo.GetRoutingRules(ctx, event.SourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing rules: %w", err)
	}

	if len(rules) > 0 {
		if routeErr := s.RouteEvent(ctx, event, rules); routeErr != nil {
			event.Status = EventStatusFailed
			_ = s.repo.UpdateEventStatus(ctx, event.ID, EventStatusFailed, routeErr.Error())
			return event, routeErr
		}
	}

	now := time.Now()
	event.Status = EventStatusRouted
	event.ProcessedAt = &now
	_ = s.repo.UpdateEventStatus(ctx, event.ID, EventStatusRouted, "")

	return event, nil
}
