package eventlineage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides event lineage business logic.
type Service struct {
	repo Repository
}

// NewService creates a new event lineage service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RecordLineage records a new event in the lineage graph.
func (s *Service) RecordLineage(ctx context.Context, tenantID string, req *RecordLineageRequest) (*LineageEntry, error) {
	if err := validateOperation(req.Operation); err != nil {
		return nil, err
	}

	entry := &LineageEntry{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		EventID:       req.EventID,
		ParentEventID: req.ParentEventID,
		EventType:     req.EventType,
		Source:        req.Source,
		Operation:     req.Operation,
		Metadata:      req.Metadata,
		PayloadHash:   req.PayloadHash,
		CreatedAt:     time.Now(),
	}

	if s.repo != nil {
		if err := s.repo.RecordEntry(ctx, entry); err != nil {
			return nil, fmt.Errorf("failed to record lineage: %w", err)
		}
	}

	return entry, nil
}

// GetLineageGraph builds the full lineage graph for an event.
func (s *Service) GetLineageGraph(ctx context.Context, tenantID, eventID string) (*LineageGraph, error) {
	if s.repo == nil {
		return &LineageGraph{RootEventID: eventID}, nil
	}

	// Get the root event
	root, err := s.repo.GetEntry(ctx, tenantID, eventID)
	if err != nil {
		return nil, err
	}

	// Get ancestors
	ancestors, _ := s.repo.GetAncestors(ctx, tenantID, eventID)

	// Get descendants
	descendants, _ := s.repo.GetDescendants(ctx, tenantID, eventID)

	// Build graph
	graph := &LineageGraph{
		RootEventID: eventID,
	}

	allEntries := append(ancestors, *root)
	allEntries = append(allEntries, descendants...)

	// Build nodes and edges
	seen := make(map[string]bool)
	for _, e := range allEntries {
		if !seen[e.EventID] {
			seen[e.EventID] = true
			graph.Nodes = append(graph.Nodes, LineageNode{
				ID:        e.ID,
				EventID:   e.EventID,
				EventType: e.EventType,
				Operation: e.Operation,
				Source:    e.Source,
			})
		}

		if e.ParentEventID != "" {
			graph.Edges = append(graph.Edges, LineageEdge{
				FromEventID: e.ParentEventID,
				ToEventID:   e.EventID,
				Operation:   e.Operation,
			})
		}
	}

	// Calculate depth
	graph.TotalDepth = calculateDepth(graph.Edges, eventID)

	return graph, nil
}

// ListEntries lists lineage entries for a tenant.
func (s *Service) ListEntries(ctx context.Context, tenantID string, limit, offset int) ([]LineageEntry, error) {
	if s.repo == nil {
		return []LineageEntry{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListEntries(ctx, tenantID, limit, offset)
}

// GetStats returns lineage statistics for a tenant.
func (s *Service) GetStats(ctx context.Context, tenantID string) (*LineageStats, error) {
	if s.repo == nil {
		return &LineageStats{OperationCounts: map[string]int64{}}, nil
	}
	return s.repo.GetStats(ctx, tenantID)
}

func calculateDepth(edges []LineageEdge, rootID string) int {
	children := make(map[string][]string)
	for _, e := range edges {
		children[e.FromEventID] = append(children[e.FromEventID], e.ToEventID)
	}

	var dfs func(string, int) int
	dfs = func(id string, depth int) int {
		maxDepth := depth
		for _, child := range children[id] {
			d := dfs(child, depth+1)
			if d > maxDepth {
				maxDepth = d
			}
		}
		return maxDepth
	}

	return dfs(rootID, 0)
}

func validateOperation(op string) error {
	switch op {
	case OpIngest, OpTransform, OpFanOut, OpRoute, OpDeliver, OpRetry, OpFilter, OpCorrelate:
		return nil
	}
	return fmt.Errorf("invalid operation %q", op)
}
