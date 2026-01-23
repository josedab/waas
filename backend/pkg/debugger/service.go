package debugger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides webhook debugging and time-travel replay functionality
type Service struct {
	repo Repository
}

// NewService creates a new debugger service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetTrace retrieves the full delivery trace for step-through debugging
func (s *Service) GetTrace(ctx context.Context, tenantID, deliveryID string) (*DeliveryTrace, error) {
	return s.repo.GetTrace(ctx, tenantID, deliveryID)
}

// ListTraces lists delivery traces for a tenant
func (s *Service) ListTraces(ctx context.Context, tenantID, endpointID string, limit, offset int) ([]DeliveryTrace, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListTraces(ctx, tenantID, endpointID, limit, offset)
}

// DiffPayloads computes the difference between two delivery payloads (e.g., pre- and post-transform)
func (s *Service) DiffPayloads(ctx context.Context, tenantID, deliveryID string) (*PayloadDiff, error) {
	trace, err := s.repo.GetTrace(ctx, tenantID, deliveryID)
	if err != nil {
		return nil, fmt.Errorf("trace not found: %w", err)
	}

	diff := &PayloadDiff{
		DeliveryID: deliveryID,
		Identical:  true,
	}

	// Find pre-transform and post-transform stages
	var preInput, postOutput string
	for _, stage := range trace.Stages {
		if stage.Name == StageTransform {
			preInput = stage.Input
		}
		if stage.Name == StagePostTransform {
			postOutput = stage.Output
		}
	}

	if preInput == "" || postOutput == "" {
		// No transform stages, compare received vs delivered
		for _, stage := range trace.Stages {
			if stage.Name == StageReceived {
				preInput = stage.Input
			}
			if stage.Name == StageDelivery {
				postOutput = stage.Input
			}
		}
	}

	if preInput != "" && postOutput != "" {
		diff.Diffs = computeJSONDiff(preInput, postOutput)
		diff.Identical = len(diff.Diffs) == 0
	}

	return diff, nil
}

// ReplayWithModifications replays a delivery with optional payload/header/endpoint overrides
func (s *Service) ReplayWithModifications(ctx context.Context, tenantID string, req *ReplayWithModRequest) (*DeliveryTrace, error) {
	payload, headers, endpointID, err := s.repo.GetDeliveryPayload(ctx, tenantID, req.DeliveryID)
	if err != nil {
		return nil, fmt.Errorf("original delivery not found: %w", err)
	}

	// Apply overrides
	if req.PayloadOverride != "" {
		payload = req.PayloadOverride
	}
	if req.EndpointOverride != "" {
		endpointID = req.EndpointOverride
	}
	for k, v := range req.HeaderOverride {
		headers[k] = v
	}

	// Create a new trace for the replay
	trace := &DeliveryTrace{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		DeliveryID:  req.DeliveryID + "-replay-" + uuid.New().String()[:8],
		EndpointID:  endpointID,
		FinalStatus: "replayed",
		CreatedAt:   time.Now(),
		Stages: []TraceStage{
			{
				Name:      StageReceived,
				Status:    "ok",
				Input:     payload,
				Timestamp: time.Now(),
				Metadata:  map[string]string{"source": "replay", "original_delivery": req.DeliveryID},
			},
		},
	}

	_ = headers // Would be used in actual delivery

	if err := s.repo.SaveTrace(ctx, trace); err != nil {
		return nil, fmt.Errorf("failed to save replay trace: %w", err)
	}

	return trace, nil
}

// BulkReplay replays multiple deliveries matching the given criteria
func (s *Service) BulkReplay(ctx context.Context, tenantID string, req *BulkReplayRequest) (*BulkReplayResult, error) {
	result := &BulkReplayResult{DryRun: req.DryRun}

	if len(req.DeliveryIDs) > 0 {
		result.TotalFound = len(req.DeliveryIDs)
	} else {
		// In production, this would query deliveries matching the filters
		traces, total, err := s.repo.ListTraces(ctx, tenantID, req.EndpointID, 100, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to list deliveries: %w", err)
		}
		result.TotalFound = total
		for _, t := range traces {
			if req.StatusFilter == "" || t.FinalStatus == req.StatusFilter {
				req.DeliveryIDs = append(req.DeliveryIDs, t.DeliveryID)
			}
		}
	}

	if req.DryRun {
		return result, nil
	}

	for _, id := range req.DeliveryIDs {
		replayReq := &ReplayWithModRequest{DeliveryID: id}
		if _, err := s.ReplayWithModifications(ctx, tenantID, replayReq); err != nil {
			result.TotalFailed++
		} else {
			result.TotalReplayed++
		}
	}

	return result, nil
}

// CreateDebugSession creates an interactive debugging session
func (s *Service) CreateDebugSession(ctx context.Context, tenantID string, req *CreateDebugSessionRequest) (*DebugSession, error) {
	// Verify the delivery exists
	if _, err := s.repo.GetTrace(ctx, tenantID, req.DeliveryID); err != nil {
		return nil, fmt.Errorf("delivery trace not found: %w", err)
	}

	session := &DebugSession{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		DeliveryID:  req.DeliveryID,
		CurrentStep: 0,
		Breakpoints: req.Breakpoints,
		Status:      DebugStatusActive,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	}

	if err := s.repo.CreateDebugSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create debug session: %w", err)
	}

	return session, nil
}

// StepDebugSession advances the debug session to the next stage
func (s *Service) StepDebugSession(ctx context.Context, tenantID, sessionID string) (*DebugSession, *TraceStage, error) {
	session, err := s.repo.GetDebugSession(ctx, tenantID, sessionID)
	if err != nil {
		return nil, nil, err
	}

	if session.Status == DebugStatusExpired || time.Now().After(session.ExpiresAt) {
		return nil, nil, fmt.Errorf("debug session has expired")
	}

	trace, err := s.repo.GetTrace(ctx, tenantID, session.DeliveryID)
	if err != nil {
		return nil, nil, err
	}

	if session.CurrentStep >= len(trace.Stages) {
		return session, nil, nil // No more stages
	}

	currentStage := trace.Stages[session.CurrentStep]
	session.CurrentStep++
	session.Status = DebugStatusActive

	// Check for breakpoints
	for _, bp := range session.Breakpoints {
		if bp == currentStage.Name {
			session.Status = DebugStatusPaused
			break
		}
	}

	if err := s.repo.UpdateDebugSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return session, &currentStage, nil
}

// GetDebugSession retrieves a debug session
func (s *Service) GetDebugSession(ctx context.Context, tenantID, sessionID string) (*DebugSession, error) {
	return s.repo.GetDebugSession(ctx, tenantID, sessionID)
}

// ListDebugSessions lists active debug sessions
func (s *Service) ListDebugSessions(ctx context.Context, tenantID string) ([]DebugSession, error) {
	return s.repo.ListDebugSessions(ctx, tenantID)
}

func computeJSONDiff(a, b string) []FieldDiff {
	var diffs []FieldDiff

	var mapA, mapB map[string]interface{}
	if err := json.Unmarshal([]byte(a), &mapA); err != nil {
		return diffs
	}
	if err := json.Unmarshal([]byte(b), &mapB); err != nil {
		return diffs
	}

	// Find removed/changed fields
	for k, vA := range mapA {
		vB, exists := mapB[k]
		if !exists {
			diffs = append(diffs, FieldDiff{
				Path: k, Type: "removed",
				OldValue: fmt.Sprintf("%v", vA),
			})
		} else if fmt.Sprintf("%v", vA) != fmt.Sprintf("%v", vB) {
			diffs = append(diffs, FieldDiff{
				Path: k, Type: "changed",
				OldValue: fmt.Sprintf("%v", vA),
				NewValue: fmt.Sprintf("%v", vB),
			})
		}
	}

	// Find added fields
	for k, vB := range mapB {
		if _, exists := mapA[k]; !exists {
			diffs = append(diffs, FieldDiff{
				Path: k, Type: "added",
				NewValue: fmt.Sprintf("%v", vB),
			})
		}
	}

	return diffs
}
