package protocols

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service manages webhook protocol configurations
type Service struct {
	repo     *Repository
	registry *Registry
}

// NewService creates a new protocol service
func NewService(repo *Repository, registry *Registry) *Service {
	if registry == nil {
		registry = DefaultRegistry()
	}
	return &Service{
		repo:     repo,
		registry: registry,
	}
}

// CreateConfig creates a new protocol configuration
func (s *Service) CreateConfig(ctx context.Context, tenantID string, req *CreateConfigRequest) (*DeliveryConfig, error) {
	if !IsValidProtocol(req.Protocol) {
		return nil, fmt.Errorf("invalid protocol: %s", req.Protocol)
	}

	config := &DeliveryConfig{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EndpointID: req.EndpointID,
		Protocol:   req.Protocol,
		Target:     req.Target,
		Options:    req.Options,
		Headers:    req.Headers,
		TLS:        req.TLS,
		Auth:       req.Auth,
		Timeout:    req.Timeout,
		Retries:    req.Retries,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.Retries == 0 {
		config.Retries = 3
	}

	// Validate config with deliverer
	if err := s.registry.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if err := s.repo.Create(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return config, nil
}

// GetConfig retrieves a protocol configuration
func (s *Service) GetConfig(ctx context.Context, tenantID, configID string) (*DeliveryConfig, error) {
	return s.repo.GetByID(ctx, tenantID, configID)
}

// ListConfigs lists protocol configurations
func (s *Service) ListConfigs(ctx context.Context, tenantID string, limit, offset int) ([]*DeliveryConfig, int, error) {
	return s.repo.List(ctx, tenantID, limit, offset)
}

// ListByProtocol lists configurations for a specific protocol
func (s *Service) ListByProtocol(ctx context.Context, tenantID string, protocol Protocol, limit, offset int) ([]*DeliveryConfig, int, error) {
	return s.repo.ListByProtocol(ctx, tenantID, protocol, limit, offset)
}

// UpdateConfig updates a protocol configuration
func (s *Service) UpdateConfig(ctx context.Context, tenantID, configID string, req *UpdateConfigRequest) (*DeliveryConfig, error) {
	config, err := s.repo.GetByID(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("config not found")
	}

	if req.Protocol != nil {
		if !IsValidProtocol(*req.Protocol) {
			return nil, fmt.Errorf("invalid protocol: %s", *req.Protocol)
		}
		config.Protocol = *req.Protocol
	}
	if req.Target != nil {
		config.Target = *req.Target
	}
	if req.Options != nil {
		config.Options = req.Options
	}
	if req.Headers != nil {
		config.Headers = req.Headers
	}
	if req.TLS != nil {
		config.TLS = req.TLS
	}
	if req.Auth != nil {
		config.Auth = req.Auth
	}
	if req.Timeout != nil {
		config.Timeout = *req.Timeout
	}
	if req.Retries != nil {
		config.Retries = *req.Retries
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	config.UpdatedAt = time.Now()

	// Validate config
	if err := s.registry.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if err := s.repo.Update(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return config, nil
}

// DeleteConfig deletes a protocol configuration
func (s *Service) DeleteConfig(ctx context.Context, tenantID, configID string) error {
	return s.repo.Delete(ctx, tenantID, configID)
}

// GetEndpointConfigs retrieves all protocol configs for an endpoint
func (s *Service) GetEndpointConfigs(ctx context.Context, tenantID, endpointID string) ([]*DeliveryConfig, error) {
	return s.repo.GetByEndpoint(ctx, tenantID, endpointID)
}

// Deliver performs a delivery using the appropriate protocol
func (s *Service) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	return s.registry.Deliver(ctx, config, request)
}

// DeliverToEndpoint delivers to all configured protocols for an endpoint
func (s *Service) DeliverToEndpoint(ctx context.Context, tenantID, endpointID string, request *DeliveryRequest) ([]*DeliveryResponse, error) {
	configs, err := s.repo.GetByEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}

	responses := make([]*DeliveryResponse, 0, len(configs))
	for _, config := range configs {
		resp, err := s.registry.Deliver(ctx, config, request)
		if err != nil {
			resp = &DeliveryResponse{
				Success:   false,
				Error:     err.Error(),
				ErrorType: ErrorTypeProtocol,
			}
		}
		resp.ProtocolInfo["config_id"] = config.ID
		resp.ProtocolInfo["protocol"] = string(config.Protocol)
		responses = append(responses, resp)
	}

	return responses, nil
}

// TestConfig tests a protocol configuration
func (s *Service) TestConfig(ctx context.Context, tenantID, configID string) (*DeliveryResponse, error) {
	config, err := s.repo.GetByID(ctx, tenantID, configID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("config not found")
	}

	// Create test request
	testRequest := &DeliveryRequest{
		ID:            uuid.New().String(),
		WebhookID:     "test-webhook",
		EndpointID:    config.EndpointID,
		Payload:       []byte(`{"test": true, "message": "WAAS connection test"}`),
		ContentType:   "application/json",
		Headers:       map[string]string{},
		Metadata:      map[string]any{"test": true},
		AttemptNumber: 1,
	}

	return s.registry.Deliver(ctx, config, testRequest)
}

// SupportedProtocols returns information about supported protocols
func (s *Service) SupportedProtocols() []ProtocolInfo {
	return SupportedProtocols()
}

// GetRegistry returns the protocol registry
func (s *Service) GetRegistry() *Registry {
	return s.registry
}

// Close closes the service and registry
func (s *Service) Close() error {
	return s.registry.Close()
}
