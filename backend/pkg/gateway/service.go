package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// DeliveryPublisher defines the interface for publishing deliveries
type DeliveryPublisher interface {
	Publish(ctx context.Context, tenantID, endpointID string, payload []byte, headers map[string]string) (string, error)
}

// Service provides gateway functionality
type Service struct {
	repo      Repository
	verifiers *VerifierRegistry
	publisher DeliveryPublisher
}

// NewService creates a new gateway service
func NewService(repo Repository, publisher DeliveryPublisher) *Service {
	return &Service{
		repo:      repo,
		verifiers: NewVerifierRegistry(),
		publisher: publisher,
	}
}

// CreateProvider creates a new webhook provider
func (s *Service) CreateProvider(ctx context.Context, tenantID string, req *CreateProviderRequest) (*Provider, error) {
	var sigConfig json.RawMessage
	if req.SignatureConfig != nil {
		sigConfig, _ = json.Marshal(req.SignatureConfig)
	}

	provider := &Provider{
		TenantID:        tenantID,
		Name:            req.Name,
		Type:            req.Type,
		Description:     req.Description,
		IsActive:        true,
		SignatureConfig: sigConfig,
	}

	if err := s.repo.CreateProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return provider, nil
}

// GetProvider retrieves a provider
func (s *Service) GetProvider(ctx context.Context, tenantID, providerID string) (*Provider, error) {
	return s.repo.GetProvider(ctx, tenantID, providerID)
}

// ListProviders lists all providers for a tenant
func (s *Service) ListProviders(ctx context.Context, tenantID string, limit, offset int) ([]Provider, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListProviders(ctx, tenantID, limit, offset)
}

// DeleteProvider deletes a provider
func (s *Service) DeleteProvider(ctx context.Context, tenantID, providerID string) error {
	return s.repo.DeleteProvider(ctx, tenantID, providerID)
}

// CreateRoutingRule creates a new routing rule
func (s *Service) CreateRoutingRule(ctx context.Context, tenantID string, req *CreateRoutingRuleRequest) (*RoutingRule, error) {
	// Validate provider exists
	provider, err := s.repo.GetProvider(ctx, tenantID, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider not found")
	}

	conditionsJSON, _ := json.Marshal(req.Conditions)
	destinationsJSON, _ := json.Marshal(req.Destinations)

	rule := &RoutingRule{
		TenantID:     tenantID,
		ProviderID:   req.ProviderID,
		Name:         req.Name,
		Description:  req.Description,
		Priority:     req.Priority,
		IsActive:     true,
		Conditions:   conditionsJSON,
		Destinations: destinationsJSON,
		Transform:    req.Transform,
	}

	if err := s.repo.CreateRoutingRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create routing rule: %w", err)
	}

	return rule, nil
}

// GetRoutingRule retrieves a routing rule
func (s *Service) GetRoutingRule(ctx context.Context, tenantID, ruleID string) (*RoutingRule, error) {
	return s.repo.GetRoutingRule(ctx, tenantID, ruleID)
}

// ListRoutingRules lists routing rules for a provider
func (s *Service) ListRoutingRules(ctx context.Context, tenantID, providerID string) ([]RoutingRule, error) {
	return s.repo.ListRoutingRules(ctx, tenantID, providerID)
}

// UpdateRoutingRule updates a routing rule
func (s *Service) UpdateRoutingRule(ctx context.Context, tenantID, ruleID string, req *UpdateRoutingRuleRequest) (*RoutingRule, error) {
	rule, err := s.repo.GetRoutingRule(ctx, tenantID, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("routing rule not found")
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.Priority != 0 {
		rule.Priority = req.Priority
	}
	rule.IsActive = req.IsActive

	if req.Conditions != nil {
		rule.Conditions, _ = json.Marshal(req.Conditions)
	}
	if req.Destinations != nil {
		rule.Destinations, _ = json.Marshal(req.Destinations)
	}
	if req.Transform != nil {
		rule.Transform = req.Transform
	}

	if err := s.repo.UpdateRoutingRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update rule: %w", err)
	}

	return rule, nil
}

// DeleteRoutingRule deletes a routing rule
func (s *Service) DeleteRoutingRule(ctx context.Context, tenantID, ruleID string) error {
	return s.repo.DeleteRoutingRule(ctx, tenantID, ruleID)
}

// ProcessInboundWebhook processes an incoming webhook
func (s *Service) ProcessInboundWebhook(ctx context.Context, tenantID, providerID string, payload []byte, headers map[string]string) (*FanoutResult, error) {
	// Get provider
	provider, err := s.repo.GetProvider(ctx, tenantID, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider not found")
	}
	if !provider.IsActive {
		return nil, fmt.Errorf("provider is inactive")
	}

	// Verify signature
	var sigConfig SignatureConfig
	if provider.SignatureConfig != nil {
		json.Unmarshal(provider.SignatureConfig, &sigConfig)
	}

	verifier := s.verifiers.Get(provider.Type)
	signatureValid, err := verifier.Verify(payload, headers, &sigConfig)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Extract event type from payload (provider-specific)
	eventType := extractEventType(provider.Type, payload, headers)

	// Save inbound webhook
	webhook := &InboundWebhook{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		ProviderID:     providerID,
		ProviderType:   provider.Type,
		EventType:      eventType,
		Payload:        payload,
		Headers:        headers,
		RawBody:        payload,
		SignatureValid: signatureValid,
	}

	if err := s.repo.SaveInboundWebhook(ctx, webhook); err != nil {
		// Log but continue
	}

	// Get routing rules
	rules, err := s.repo.ListRoutingRules(ctx, tenantID, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing rules: %w", err)
	}

	// Fan out to matching destinations
	result := &FanoutResult{
		InboundID: webhook.ID,
	}

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		// Check conditions
		if !s.matchesConditions(payload, headers, eventType, rule.Conditions) {
			continue
		}

		// Route to destinations
		var destinations []RoutingDestination
		json.Unmarshal(rule.Destinations, &destinations)

		for _, dest := range destinations {
			destResult := s.routeToDestination(ctx, tenantID, payload, headers, &rule, &dest)
			result.Destinations = append(result.Destinations, destResult)

			if destResult.Status == "queued" {
				result.TotalRouted++
			} else {
				result.TotalFailed++
			}
		}
	}

	return result, nil
}

func (s *Service) matchesConditions(payload []byte, headers map[string]string, eventType string, conditionsJSON json.RawMessage) bool {
	if conditionsJSON == nil || string(conditionsJSON) == "null" || string(conditionsJSON) == "[]" {
		return true // No conditions = match all
	}

	var conditions []RoutingCondition
	if err := json.Unmarshal(conditionsJSON, &conditions); err != nil {
		return true
	}

	var payloadMap map[string]interface{}
	json.Unmarshal(payload, &payloadMap)

	for _, cond := range conditions {
		value := s.extractValue(cond.Field, payloadMap, headers, eventType)

		switch cond.Operator {
		case "equals":
			if value != cond.Value {
				return false
			}
		case "not_equals":
			if value == cond.Value {
				return false
			}
		case "contains":
			if !strings.Contains(value, cond.Value) {
				return false
			}
		case "matches":
			matched, _ := regexp.MatchString(cond.Value, value)
			if !matched {
				return false
			}
		case "exists":
			if value == "" {
				return false
			}
		}
	}

	return true
}

func (s *Service) extractValue(field string, payload map[string]interface{}, headers map[string]string, eventType string) string {
	if field == "event_type" || field == "type" {
		return eventType
	}

	if strings.HasPrefix(field, "header.") {
		headerName := strings.TrimPrefix(field, "header.")
		return headers[headerName]
	}

	// JSON path extraction (simplified)
	parts := strings.Split(field, ".")
	var current interface{} = payload

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return ""
		}
	}

	if str, ok := current.(string); ok {
		return str
	}

	return fmt.Sprintf("%v", current)
}

func (s *Service) routeToDestination(ctx context.Context, tenantID string, payload []byte, headers map[string]string, rule *RoutingRule, dest *RoutingDestination) DestinationResult {
	result := DestinationResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Type:     dest.Type,
	}

	switch dest.Type {
	case "endpoint":
		result.Target = dest.EndpointID
		deliveryID, err := s.publisher.Publish(ctx, tenantID, dest.EndpointID, payload, headers)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		} else {
			result.Status = "queued"
			result.DeliveryID = deliveryID
		}

	default:
		result.Status = "failed"
		result.Error = fmt.Sprintf("unsupported destination type: %s", dest.Type)
	}

	return result
}

// GetInboundWebhook retrieves an inbound webhook
func (s *Service) GetInboundWebhook(ctx context.Context, tenantID, webhookID string) (*InboundWebhook, error) {
	return s.repo.GetInboundWebhook(ctx, tenantID, webhookID)
}

// ListInboundWebhooks lists inbound webhooks
func (s *Service) ListInboundWebhooks(ctx context.Context, tenantID, providerID string, limit, offset int) ([]InboundWebhook, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListInboundWebhooks(ctx, tenantID, providerID, limit, offset)
}

// GetProviderEndpointURL returns the endpoint URL for receiving webhooks from a provider
func (s *Service) GetProviderEndpointURL(baseURL, tenantID, providerID string) string {
	return fmt.Sprintf("%s/gateway/%s/%s", baseURL, tenantID, providerID)
}

func extractEventType(providerType string, payload []byte, headers map[string]string) string {
	var data map[string]interface{}
	json.Unmarshal(payload, &data)

	switch providerType {
	case ProviderTypeStripe:
		if t, ok := data["type"].(string); ok {
			return t
		}
	case ProviderTypeGitHub:
		return headers["X-GitHub-Event"]
	case ProviderTypeShopify:
		return headers["X-Shopify-Topic"]
	case ProviderTypeSlack:
		if t, ok := data["type"].(string); ok {
			return t
		}
	case ProviderTypePaddle:
		if t, ok := data["event_type"].(string); ok {
			return t
		}
	case ProviderTypeLinear:
		if t, ok := data["type"].(string); ok {
			return t
		}
		if t, ok := data["action"].(string); ok {
			return t
		}
	case ProviderTypeIntercom:
		if t, ok := data["topic"].(string); ok {
			return t
		}
	case ProviderTypeDiscord:
		if t, ok := data["t"].(string); ok {
			return t
		}
	}

	// Generic extraction
	if t, ok := data["type"].(string); ok {
		return t
	}
	if t, ok := data["event"].(string); ok {
		return t
	}
	if t, ok := data["event_type"].(string); ok {
		return t
	}

	return ""
}
