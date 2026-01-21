package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TransformEngine defines the interface for executing transforms
type TransformEngine interface {
	Execute(script string, event interface{}, config interface{}) (interface{}, error)
}

// Service provides connector functionality
type Service struct {
	registry        *Registry
	repo            Repository
	transformEngine TransformEngine
}

// NewService creates a new connector service
func NewService(repo Repository, transformEngine TransformEngine) *Service {
	return &Service{
		registry:        NewRegistry(),
		repo:            repo,
		transformEngine: transformEngine,
	}
}

// ListMarketplace lists available connectors in the marketplace
func (s *Service) ListMarketplace(ctx context.Context, filters *MarketplaceListRequest) ([]*Connector, error) {
	connectors := s.registry.List(filters)

	// Apply search filter
	if filters.Search != "" {
		var filtered []*Connector
		search := filters.Search
		for _, c := range connectors {
			if contains(c.Name, search) || contains(c.Description, search) {
				filtered = append(filtered, c)
			}
		}
		connectors = filtered
	}

	// Apply pagination
	offset := filters.Offset
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	if offset >= len(connectors) {
		return []*Connector{}, nil
	}

	end := offset + limit
	if end > len(connectors) {
		end = len(connectors)
	}

	return connectors[offset:end], nil
}

// GetConnector retrieves a connector from the marketplace
func (s *Service) GetConnector(ctx context.Context, connectorID string) (*Connector, error) {
	connector := s.registry.Get(connectorID)
	if connector == nil {
		return nil, fmt.Errorf("connector not found: %s", connectorID)
	}
	return connector, nil
}

// InstallConnector installs a connector for a tenant
func (s *Service) InstallConnector(ctx context.Context, tenantID string, req *InstallConnectorRequest) (*InstalledConnector, error) {
	// Validate connector exists
	connector := s.registry.Get(req.ConnectorID)
	if connector == nil {
		return nil, fmt.Errorf("connector not found: %s", req.ConnectorID)
	}

	// Validate config against schema
	if err := s.validateConfig(connector.ConfigSchema, req.Config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	installed := &InstalledConnector{
		TenantID:    tenantID,
		ConnectorID: req.ConnectorID,
		Name:        req.Name,
		Config:      req.Config,
		IsActive:    true,
	}

	if err := s.repo.InstallConnector(ctx, installed); err != nil {
		return nil, fmt.Errorf("failed to install connector: %w", err)
	}

	return installed, nil
}

// GetInstalledConnector retrieves an installed connector
func (s *Service) GetInstalledConnector(ctx context.Context, tenantID, id string) (*InstalledConnector, error) {
	return s.repo.GetInstalledConnector(ctx, tenantID, id)
}

// ListInstalledConnectors lists all installed connectors for a tenant
func (s *Service) ListInstalledConnectors(ctx context.Context, tenantID string, limit, offset int) ([]InstalledConnector, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListInstalledConnectors(ctx, tenantID, limit, offset)
}

// UpdateInstalledConnector updates an installed connector
func (s *Service) UpdateInstalledConnector(ctx context.Context, tenantID, id string, req *UpdateConnectorRequest) (*InstalledConnector, error) {
	installed, err := s.repo.GetInstalledConnector(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get connector: %w", err)
	}
	if installed == nil {
		return nil, fmt.Errorf("installed connector not found")
	}

	if req.Name != "" {
		installed.Name = req.Name
	}
	if req.Config != nil {
		// Validate config
		connector := s.registry.Get(installed.ConnectorID)
		if connector != nil {
			if err := s.validateConfig(connector.ConfigSchema, req.Config); err != nil {
				return nil, fmt.Errorf("invalid configuration: %w", err)
			}
		}
		installed.Config = req.Config
	}
	installed.IsActive = req.IsActive

	if err := s.repo.UpdateInstalledConnector(ctx, installed); err != nil {
		return nil, fmt.Errorf("failed to update connector: %w", err)
	}

	return installed, nil
}

// UninstallConnector uninstalls a connector
func (s *Service) UninstallConnector(ctx context.Context, tenantID, id string) error {
	return s.repo.UninstallConnector(ctx, tenantID, id)
}

// ExecuteConnector executes a connector transformation
func (s *Service) ExecuteConnector(ctx context.Context, tenantID, installedID string, eventType string, payload []byte) ([]byte, error) {
	startTime := time.Now()

	// Get installed connector
	installed, err := s.repo.GetInstalledConnector(ctx, tenantID, installedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connector: %w", err)
	}
	if installed == nil {
		return nil, fmt.Errorf("installed connector not found")
	}
	if !installed.IsActive {
		return nil, fmt.Errorf("connector is inactive")
	}

	// Get connector definition
	connector := s.registry.Get(installed.ConnectorID)
	if connector == nil {
		return nil, fmt.Errorf("connector definition not found")
	}

	// Parse payload and config
	var event interface{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	var config interface{}
	if installed.Config != nil {
		json.Unmarshal(installed.Config, &config)
	}

	// Execute transformation
	var result interface{}
	var execError string
	status := "success"

	if s.transformEngine != nil {
		result, err = s.transformEngine.Execute(connector.Transform, event, config)
		if err != nil {
			execError = err.Error()
			status = "error"
		}
	} else {
		// No transform engine, pass through
		result = event
	}

	// Convert result to JSON
	var outputPayload []byte
	if result != nil {
		outputPayload, _ = json.Marshal(result)
	}

	// Log execution
	exec := &ConnectorExecution{
		InstalledConnectorID: installedID,
		EventType:            eventType,
		InputPayload:         payload,
		OutputPayload:        outputPayload,
		Status:               status,
		Error:                execError,
		Duration:             time.Since(startTime).Milliseconds(),
	}
	s.repo.LogExecution(ctx, exec)

	if status == "error" {
		return nil, fmt.Errorf("transform execution failed: %s", execError)
	}

	return outputPayload, nil
}

// ListExecutions lists connector executions
func (s *Service) ListExecutions(ctx context.Context, tenantID, installedID string, limit, offset int) ([]ConnectorExecution, int, error) {
	// Verify ownership
	installed, err := s.repo.GetInstalledConnector(ctx, tenantID, installedID)
	if err != nil || installed == nil {
		return nil, 0, fmt.Errorf("installed connector not found")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.repo.ListExecutions(ctx, installedID, limit, offset)
}

func (s *Service) validateConfig(schema json.RawMessage, config json.RawMessage) error {
	if schema == nil || string(schema) == "null" {
		return nil
	}

	// Basic validation - check required fields
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return nil // Skip validation if schema is invalid
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(config, &configMap); err != nil {
		return fmt.Errorf("config must be a JSON object")
	}

	// Check required fields
	if required, ok := schemaMap["required"].([]interface{}); ok {
		for _, r := range required {
			field := r.(string)
			if _, exists := configMap[field]; !exists {
				return fmt.Errorf("missing required field: %s", field)
			}
		}
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
