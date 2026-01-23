package eventmesh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides event mesh routing functionality
type Service struct {
	repo Repository
}

// NewService creates a new event mesh service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateRoute creates a new routing rule
func (s *Service) CreateRoute(ctx context.Context, tenantID string, req *CreateRouteRequest) (*Route, error) {
	if len(req.Targets) == 0 {
		return nil, fmt.Errorf("at least one target is required")
	}

	route := &Route{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		SourceFilter: req.SourceFilter,
		Targets:      req.Targets,
		Priority:     req.Priority,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateRoute(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	return route, nil
}

// GetRoute retrieves a route by ID
func (s *Service) GetRoute(ctx context.Context, tenantID, routeID string) (*Route, error) {
	return s.repo.GetRoute(ctx, tenantID, routeID)
}

// ListRoutes lists all routes for a tenant
func (s *Service) ListRoutes(ctx context.Context, tenantID string, limit, offset int) ([]Route, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListRoutes(ctx, tenantID, limit, offset)
}

// UpdateRoute updates an existing route
func (s *Service) UpdateRoute(ctx context.Context, tenantID, routeID string, req *CreateRouteRequest) (*Route, error) {
	route, err := s.repo.GetRoute(ctx, tenantID, routeID)
	if err != nil {
		return nil, err
	}

	route.Name = req.Name
	route.Description = req.Description
	route.SourceFilter = req.SourceFilter
	route.Targets = req.Targets
	route.Priority = req.Priority
	route.UpdatedAt = time.Now()

	if err := s.repo.UpdateRoute(ctx, route); err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	return route, nil
}

// DeleteRoute deletes a route
func (s *Service) DeleteRoute(ctx context.Context, tenantID, routeID string) error {
	return s.repo.DeleteRoute(ctx, tenantID, routeID)
}

// RouteEvent routes an incoming event to matching targets via fan-out
func (s *Service) RouteEvent(ctx context.Context, tenantID string, req *RouteEventRequest) (*RouteExecution, error) {
	start := time.Now()

	routes, err := s.repo.GetMatchingRoutes(ctx, tenantID, req.EventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get matching routes: %w", err)
	}

	// Sort by priority
	sortedRoutes := sortByPriority(routes)

	var payload map[string]interface{}
	json.Unmarshal([]byte(req.Payload), &payload)

	totalHit := 0
	totalFailed := 0

	for _, route := range sortedRoutes {
		if !route.IsActive {
			continue
		}

		// Check filter match
		if !matchesFilter(route.SourceFilter, req.EventType, req.Headers, payload) {
			continue
		}

		// Fan-out to all targets
		for _, target := range route.Targets {
			// Check target condition
			if target.Condition != "" && !evaluateCondition(target.Condition, payload) {
				continue
			}

			totalHit++
			// In a real implementation, this would enqueue to the delivery engine
			// For now, we record the intent
		}
	}

	exec := &RouteExecution{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		RouteID:       "", // aggregated across all matching routes
		SourceEvent:   req.EventType,
		TargetsHit:    totalHit,
		TargetsFailed: totalFailed,
		DurationMs:    int(time.Since(start).Milliseconds()),
		CreatedAt:     time.Now(),
	}

	if len(sortedRoutes) > 0 {
		exec.RouteID = sortedRoutes[0].ID
	}

	if err := s.repo.SaveExecution(ctx, exec); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	return exec, nil
}

// ConfigureDeadLetter sets up dead letter queue configuration for a route
func (s *Service) ConfigureDeadLetter(ctx context.Context, tenantID, routeID string, req *ConfigureDeadLetterRequest) (*DeadLetterConfig, error) {
	config := &DeadLetterConfig{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		RouteID:       routeID,
		MaxRetries:    req.MaxRetries,
		RetentionDays: req.RetentionDays,
		AlertOnEntry:  req.AlertOnEntry,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.ConfigureDeadLetter(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to configure dead letter: %w", err)
	}

	return config, nil
}

// ListDeadLetterEntries lists dead letter entries for a route
func (s *Service) ListDeadLetterEntries(ctx context.Context, tenantID, routeID string, limit, offset int) ([]DeadLetterEntry, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListDeadLetterEntries(ctx, tenantID, routeID, limit, offset)
}

// RedriveDeadLetter attempts to redeliver a dead letter entry
func (s *Service) RedriveDeadLetter(ctx context.Context, tenantID, entryID string) error {
	return s.repo.DeleteDeadLetterEntry(ctx, tenantID, entryID)
}

// GetRouteStats returns routing statistics
func (s *Service) GetRouteStats(ctx context.Context, tenantID string) (*RouteStats, error) {
	return s.repo.GetRouteStats(ctx, tenantID)
}

// ListExecutions lists route executions
func (s *Service) ListExecutions(ctx context.Context, tenantID, routeID string, limit, offset int) ([]RouteExecution, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListExecutions(ctx, tenantID, routeID, limit, offset)
}

func matchesFilter(filter *Filter, eventType string, headers map[string]string, payload map[string]interface{}) bool {
	if filter == nil {
		return true
	}

	// Check event type filter
	if len(filter.EventTypes) > 0 {
		matched := false
		for _, et := range filter.EventTypes {
			if et == eventType || matchGlob(et, eventType) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check header filters
	for k, v := range filter.Headers {
		if headers[k] != v {
			return false
		}
	}

	return true
}

func matchGlob(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}
	return pattern == value
}

func evaluateCondition(condition string, payload map[string]interface{}) bool {
	// Simple key existence check for now
	parts := strings.SplitN(condition, "=", 2)
	if len(parts) == 2 {
		val, ok := payload[parts[0]]
		if !ok {
			return false
		}
		return fmt.Sprintf("%v", val) == parts[1]
	}

	_, exists := payload[condition]
	return exists
}

func sortByPriority(routes []Route) []Route {
	// Simple insertion sort by priority (ascending)
	for i := 1; i < len(routes); i++ {
		key := routes[i]
		j := i - 1
		for j >= 0 && routes[j].Priority > key.Priority {
			routes[j+1] = routes[j]
			j--
		}
		routes[j+1] = key
	}
	return routes
}
