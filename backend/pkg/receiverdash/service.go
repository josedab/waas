package receiverdash

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ServiceConfig configures the receiver dashboard service.
type ServiceConfig struct {
	MaxTokensPerTenant   int
	MaxEndpointsPerToken int
	DefaultTokenExpiry   time.Duration
	MaxPayloadRetention  time.Duration
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxTokensPerTenant:   50,
		MaxEndpointsPerToken: 20,
		DefaultTokenExpiry:   30 * 24 * time.Hour, // 30 days
		MaxPayloadRetention:  7 * 24 * time.Hour,  // 7 days
	}
}

// Service implements the receiver dashboard business logic.
type Service struct {
	repo   Repository
	config *ServiceConfig
}

// NewService creates a new receiver dashboard service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{repo: repo, config: config}
}

// CreateToken generates a new receiver-scoped read-only token.
func (s *Service) CreateToken(tenantID string, req *CreateTokenRequest) (*ReceiverToken, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if len(req.EndpointIDs) == 0 {
		return nil, fmt.Errorf("at least one endpoint_id is required")
	}
	if len(req.EndpointIDs) > s.config.MaxEndpointsPerToken {
		return nil, fmt.Errorf("maximum %d endpoints per token", s.config.MaxEndpointsPerToken)
	}
	if req.Label == "" {
		return nil, fmt.Errorf("label is required")
	}

	// Validate scopes
	validSet := make(map[string]bool)
	for _, sc := range ValidScopes {
		validSet[sc] = true
	}
	for _, sc := range req.Scopes {
		if !validSet[sc] {
			return nil, fmt.Errorf("invalid scope: %s", sc)
		}
	}

	// Check token limit
	existing, err := s.repo.ListTokens(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing tokens: %w", err)
	}
	activeCount := 0
	for _, t := range existing {
		if t.RevokedAt == nil {
			activeCount++
		}
	}
	if activeCount >= s.config.MaxTokensPerTenant {
		return nil, fmt.Errorf("maximum %d active tokens per tenant", s.config.MaxTokensPerTenant)
	}

	// Compute expiry
	expiry := s.config.DefaultTokenExpiry
	if req.ExpiresIn != "" {
		parsed, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			return nil, fmt.Errorf("invalid expires_in duration: %w", err)
		}
		expiry = parsed
	}

	token := &ReceiverToken{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Token:       generateReceiverToken(),
		EndpointIDs: req.EndpointIDs,
		Label:       req.Label,
		Scopes:      req.Scopes,
		ExpiresAt:   time.Now().Add(expiry),
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateToken(token); err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}
	return token, nil
}

// ValidateToken validates a receiver token and returns it if valid.
func (s *Service) ValidateToken(tokenValue string) (*ReceiverToken, error) {
	token, err := s.repo.GetTokenByValue(tokenValue)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if token.RevokedAt != nil {
		return nil, fmt.Errorf("token has been revoked")
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token has expired")
	}
	return token, nil
}

// GetToken retrieves a token by ID.
func (s *Service) GetToken(tokenID string) (*ReceiverToken, error) {
	return s.repo.GetToken(tokenID)
}

// ListTokens returns all tokens for a tenant.
func (s *Service) ListTokens(tenantID string) ([]*ReceiverToken, error) {
	return s.repo.ListTokens(tenantID)
}

// RevokeToken revokes an existing token.
func (s *Service) RevokeToken(tokenID string) error {
	return s.repo.RevokeToken(tokenID)
}

// GetDeliveryHistory returns delivery history for endpoints authorized by the token.
func (s *Service) GetDeliveryHistory(token *ReceiverToken, req *DeliveryHistoryRequest) ([]*DeliveryRecord, int, error) {
	if !hasScope(token, "read:deliveries") {
		return nil, 0, fmt.Errorf("token lacks read:deliveries scope")
	}
	endpointIDs := token.EndpointIDs
	if req.EndpointID != "" {
		if !isEndpointAllowed(token, req.EndpointID) {
			return nil, 0, fmt.Errorf("endpoint %s not authorized", req.EndpointID)
		}
		endpointIDs = []string{req.EndpointID}
	}
	return s.repo.GetDeliveryHistory(endpointIDs, req)
}

// GetRetryStatus returns retry status for authorized endpoints.
func (s *Service) GetRetryStatus(token *ReceiverToken, activeOnly bool) ([]*RetryStatus, error) {
	if !hasScope(token, "read:retries") {
		return nil, fmt.Errorf("token lacks read:retries scope")
	}
	return s.repo.GetRetryStatus(token.EndpointIDs, activeOnly)
}

// InspectPayload returns the payload for a specific delivery.
func (s *Service) InspectPayload(token *ReceiverToken, deliveryID string) (*PayloadInspection, error) {
	if !hasScope(token, "read:payloads") {
		return nil, fmt.Errorf("token lacks read:payloads scope")
	}
	return s.repo.GetDeliveryPayload(deliveryID)
}

// GetEndpointHealth returns health metrics for a single endpoint.
func (s *Service) GetEndpointHealth(token *ReceiverToken, endpointID string, period string) (*EndpointHealth, error) {
	if !hasScope(token, "read:health") {
		return nil, fmt.Errorf("token lacks read:health scope")
	}
	if !isEndpointAllowed(token, endpointID) {
		return nil, fmt.Errorf("endpoint %s not authorized", endpointID)
	}
	if period == "" {
		period = "24h"
	}
	return s.repo.GetEndpointHealth(endpointID, period)
}

// GetHealthSummary returns a health summary across all authorized endpoints.
func (s *Service) GetHealthSummary(token *ReceiverToken, period string) (*HealthSummary, error) {
	if !hasScope(token, "read:health") {
		return nil, fmt.Errorf("token lacks read:health scope")
	}
	if period == "" {
		period = "24h"
	}
	return s.repo.GetHealthSummary(token.EndpointIDs, period)
}

func hasScope(token *ReceiverToken, scope string) bool {
	for _, s := range token.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func isEndpointAllowed(token *ReceiverToken, endpointID string) bool {
	for _, id := range token.EndpointIDs {
		if id == endpointID {
			return true
		}
	}
	return false
}

func generateReceiverToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "rcv_" + hex.EncodeToString(b)
}
