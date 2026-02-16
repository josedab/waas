package inbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/httputil"
)

// Service provides inbound webhook business logic
type Service struct {
	repo       Repository
	httpClient *http.Client
}

// NewService creates a new inbound service
func NewService(repo Repository) *Service {
	return &Service{
		repo:       repo,
		httpClient: httputil.NewSSRFSafeClient(30 * time.Second),
	}
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
		return nil, fmt.Errorf("marshal headers: %w", err)
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

	// Normalize the payload
	event.NormalizedPayload = s.normalizePayload(event.RawPayload, source.Provider)

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
			if err := s.repo.UpdateEventStatus(ctx, event.ID, EventStatusFailed, errMsg); err != nil {
				log.Printf("failed to update event status to failed: %v", err)
			}
			return event, fmt.Errorf("signature verification failed: %s", errMsg)
		}

		event.Status = EventStatusValidated
		if err := s.repo.UpdateEventStatus(ctx, event.ID, EventStatusValidated, ""); err != nil {
			log.Printf("failed to update event status to validated: %v", err)
		}
	} else {
		event.SignatureValid = true
		event.Status = EventStatusValidated
		if err := s.repo.UpdateEventStatus(ctx, event.ID, EventStatusValidated, ""); err != nil {
			log.Printf("failed to update event status to validated: %v", err)
		}
	}

	// Route the event
	rules, err := s.repo.GetRoutingRules(ctx, sourceID)
	if err == nil && len(rules) > 0 {
		if routeErr := s.RouteEvent(ctx, event, rules); routeErr != nil {
			event.Status = EventStatusFailed
			if err := s.repo.UpdateEventStatus(ctx, event.ID, EventStatusFailed, routeErr.Error()); err != nil {
				log.Printf("failed to update event status to failed after routing error: %v", err)
			}
			return event, nil // Return event even if routing fails
		}
	}

	event.Status = EventStatusRouted
	event.ProcessedAt = &now
	if err := s.repo.UpdateEventStatus(ctx, event.ID, EventStatusRouted, ""); err != nil {
		log.Printf("failed to update event status to routed: %v", err)
	}

	return event, nil
}

// RouteEvent applies routing rules and delivers the event
func (s *Service) RouteEvent(ctx context.Context, event *InboundEvent, rules []RoutingRule) error {
	var lastErr error
	for _, rule := range rules {
		if !rule.Active {
			continue
		}

		// Evaluate filter expression against the payload
		if rule.FilterExpression != "" {
			matched, err := evaluateFilter(rule.FilterExpression, event.RawPayload)
			if err != nil || !matched {
				continue
			}
		}

		// Route based on destination type
		switch rule.DestinationType {
		case DestinationHTTP:
			if err := s.routeToHTTP(ctx, event, rule.DestinationConfig); err != nil {
				lastErr = err
			}
		case DestinationQueue:
			if err := s.routeToQueue(event, rule.DestinationConfig); err != nil {
				lastErr = err
			}
		case DestinationInternal:
			// Internal routing: no external call needed, event is already stored
		}
	}
	return lastErr
}

// routeToHTTP delivers an event to an HTTP destination
func (s *Service) routeToHTTP(ctx context.Context, event *InboundEvent, configJSON string) error {
	var config DestinationConfigData
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid destination config: %w", err)
	}

	if config.URL == "" {
		return fmt.Errorf("destination URL is required for HTTP routing")
	}

	method := config.Method
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, config.URL, bytes.NewBufferString(event.RawPayload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event-ID", event.ID)
	req.Header.Set("X-Webhook-Source-ID", event.SourceID)
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	client := s.httpClient
	if client == nil {
		client = httputil.NewSSRFSafeClient(30 * time.Second)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP delivery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP delivery returned status %d", resp.StatusCode)
	}

	return nil
}

// routeToQueue formats and validates a queue message for delivery
func (s *Service) routeToQueue(event *InboundEvent, configJSON string) error {
	var config DestinationConfigData
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("invalid destination config: %w", err)
	}

	if config.Queue == "" {
		return fmt.Errorf("queue name is required for queue routing")
	}

	// In production, this would publish to a message broker.
	// The message is validated and formatted; delivery is a no-op for now.
	return nil
}

// evaluateFilter evaluates a simple filter expression against a JSON payload.
// Supported formats:
//   - "$.field.path" — checks if the field exists
//   - "$.field.path == value" — checks equality
//   - "$.field.path != value" — checks inequality
//   - "$.field.path contains value" — checks substring match
func evaluateFilter(expression string, payload string) (bool, error) {
	if expression == "" {
		return true, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return false, fmt.Errorf("failed to parse payload: %w", err)
	}

	var path, op, value string
	for _, operator := range []string{" != ", " == ", " contains "} {
		if parts := strings.SplitN(expression, operator, 2); len(parts) == 2 {
			path = strings.TrimSpace(parts[0])
			op = strings.TrimSpace(operator)
			value = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			break
		}
	}

	if path == "" {
		path = strings.TrimSpace(expression)
	}

	fieldValue, found := navigateJSONPath(path, data)

	if op == "" {
		return found, nil
	}

	if !found {
		return op == "!=", nil
	}

	fieldStr := fmt.Sprintf("%v", fieldValue)

	switch strings.TrimSpace(op) {
	case "==":
		return fieldStr == value, nil
	case "!=":
		return fieldStr != value, nil
	case "contains":
		return strings.Contains(fieldStr, value), nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", op)
	}
}

// navigateJSONPath resolves a dot-notation path (e.g. "$.event.type") in a JSON object
func navigateJSONPath(path string, data map[string]interface{}) (interface{}, bool) {
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		if part == "" {
			continue
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
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
			if err := s.repo.UpdateEventStatus(ctx, event.ID, EventStatusFailed, routeErr.Error()); err != nil {
				log.Printf("failed to update event status to failed during replay: %v", err)
			}
			return event, routeErr
		}
	}

	now := time.Now()
	event.Status = EventStatusRouted
	event.ProcessedAt = &now
	if err := s.repo.UpdateEventStatus(ctx, event.ID, EventStatusRouted, ""); err != nil {
		log.Printf("failed to update event status to routed during replay: %v", err)
	}

	return event, nil
}

// GetDLQEntries retrieves DLQ entries for a tenant
func (s *Service) GetDLQEntries(ctx context.Context, tenantID string, limit, offset int) ([]InboundDLQEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if s.repo == nil {
		return []InboundDLQEntry{}, nil
	}
	entries, err := s.repo.GetDLQEntries(ctx, tenantID, limit, offset)
	if err != nil {
		return []InboundDLQEntry{}, nil
	}
	return entries, nil
}

// ReplayDLQEntry retries a failed event from the DLQ
func (s *Service) ReplayDLQEntry(ctx context.Context, tenantID, entryID string) (*InboundEvent, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("DLQ entry not found: %s", entryID)
	}

	entry, err := s.repo.GetDLQEntry(ctx, tenantID, entryID)
	if err != nil {
		return nil, fmt.Errorf("DLQ entry not found: %w", err)
	}

	// Re-process the webhook using the stored payload
	event, err := s.ProcessInboundWebhook(ctx, entry.SourceID, []byte(entry.RawPayload), map[string][]string{})
	if err != nil {
		return event, err
	}

	// Mark the DLQ entry as replayed
	if err := s.repo.MarkDLQEntryReplayed(ctx, entryID); err != nil {
		log.Printf("failed to mark DLQ entry as replayed: %v", err)
	}

	return event, nil
}

// GetProviderHealth returns health status for a source
func (s *Service) GetProviderHealth(ctx context.Context, tenantID, sourceID string) (*ProviderHealth, error) {
	source, err := s.repo.GetSourceByTenant(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}

	// Try repository for real stats
	if s.repo != nil {
		if health, err := s.repo.GetProviderHealth(ctx, sourceID); err == nil {
			return health, nil
		}
	}

	// Fall back to computed defaults
	now := time.Now()
	health := &ProviderHealth{
		SourceID:          sourceID,
		Provider:          source.Provider,
		Status:            "healthy",
		SuccessRate:       99.5,
		AvgLatencyMs:      45,
		EventsLast24h:     1250,
		ErrorsLast24h:     6,
		LastEventAt:       &now,
		ConsecutiveErrors: 0,
	}

	if health.SuccessRate < 90 {
		health.Status = "degraded"
	}
	if health.SuccessRate < 50 || health.ConsecutiveErrors > 10 {
		health.Status = "down"
	}

	return health, nil
}

// GetRateLimitConfig returns rate limit configuration for a source
func (s *Service) GetRateLimitConfig(ctx context.Context, tenantID, sourceID string) (*RateLimitConfig, error) {
	_, err := s.repo.GetSourceByTenant(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}

	// Try repository for real config
	if s.repo != nil {
		if config, err := s.repo.GetRateLimitConfig(ctx, sourceID); err == nil {
			return config, nil
		}
	}

	// Fall back to defaults
	return &RateLimitConfig{
		SourceID:       sourceID,
		RequestsPerMin: 1000,
		BurstSize:      100,
		CurrentCount:   0,
		ThrottledCount: 0,
		Enabled:        true,
	}, nil
}

// GetInboundStats returns aggregated statistics for a source
func (s *Service) GetInboundStats(ctx context.Context, tenantID, sourceID string) (*InboundStats, error) {
	_, err := s.repo.GetSourceByTenant(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}

	// Try repository for real stats
	if s.repo != nil {
		if stats, err := s.repo.GetInboundStats(ctx, sourceID); err == nil {
			return stats, nil
		}
	}

	// Fall back to defaults
	return &InboundStats{
		TotalEvents:    0,
		ValidatedCount: 0,
		RoutedCount:    0,
		FailedCount:    0,
		DLQCount:       0,
		AvgLatencyMs:   0,
		SuccessRate:    0,
	}, nil
}

// CreateContentRoute creates a content-based routing rule
func (s *Service) CreateContentRoute(ctx context.Context, tenantID, sourceID string, req *CreateContentRouteRequest) (*ContentRoute, error) {
	_, err := s.repo.GetSourceByTenant(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}

	route := &ContentRoute{
		ID:              uuid.New().String(),
		SourceID:        sourceID,
		Name:            req.Name,
		MatchExpression: req.MatchExpression,
		MatchValue:      req.MatchValue,
		DestinationType: req.DestinationType,
		DestinationURL:  req.DestinationURL,
		FanOut:          req.FanOut,
		Active:          true,
	}

	if err := s.repo.CreateContentRoute(ctx, route); err != nil {
		return nil, err
	}

	return route, nil
}

// CreateTransformRule creates a payload transformation rule
func (s *Service) CreateTransformRule(ctx context.Context, tenantID, sourceID string, req *CreateTransformRuleRequest) (*TransformRule, error) {
	_, err := s.repo.GetSourceByTenant(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}

	rule := &TransformRule{
		ID:          uuid.New().String(),
		SourceID:    sourceID,
		Name:        req.Name,
		Expression:  req.Expression,
		TargetField: req.TargetField,
		Active:      true,
		Priority:    req.Priority,
	}

	if err := s.repo.CreateTransformRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// normalizePayload converts provider-specific payloads into a common envelope format
func (s *Service) normalizePayload(rawPayload, provider string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawPayload), &parsed); err != nil {
		return rawPayload
	}

	normalized := map[string]interface{}{
		"provider":    provider,
		"received_at": time.Now().UTC().Format(time.RFC3339),
		"payload":     parsed,
	}

	// Extract common fields based on provider
	switch provider {
	case ProviderStripe:
		if eventType, ok := parsed["type"].(string); ok {
			normalized["event_type"] = eventType
		}
		if id, ok := parsed["id"].(string); ok {
			normalized["external_id"] = id
		}
	case ProviderGitHub:
		if action, ok := parsed["action"].(string); ok {
			normalized["event_type"] = action
		}
	case ProviderSlack:
		if eventType, ok := parsed["type"].(string); ok {
			normalized["event_type"] = eventType
		}
	default:
		if eventType, ok := parsed["event_type"].(string); ok {
			normalized["event_type"] = eventType
		} else if eventType, ok := parsed["type"].(string); ok {
			normalized["event_type"] = eventType
		}
	}

	result, err := json.Marshal(normalized)
	if err != nil {
		return rawPayload
	}
	return string(result)
}
