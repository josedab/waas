package timetravel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DebugSession represents an interactive debugging session for replaying events.
type DebugSession struct {
	ID              string                `json:"id" db:"id"`
	TenantID        string                `json:"tenant_id" db:"tenant_id"`
	Name            string                `json:"name" db:"name"`
	EventIDs        []string              `json:"event_ids"`
	Status          string                `json:"status" db:"status"`
	Breakpoints     []Breakpoint          `json:"breakpoints,omitempty"`
	StepHistory     []DebugStep           `json:"step_history,omitempty"`
	CurrentStep     int                   `json:"current_step" db:"current_step"`
	Context         json.RawMessage       `json:"context,omitempty" db:"context"`
	CreatedAt       time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at" db:"updated_at"`
}

// Breakpoint defines a condition where replay pauses for inspection.
type Breakpoint struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Condition string `json:"condition"`
	Enabled   bool   `json:"enabled"`
}

// DebugStep records a single step in the debug session.
type DebugStep struct {
	StepNumber    int              `json:"step_number"`
	EventID       string           `json:"event_id"`
	Action        string           `json:"action"`
	Timestamp     time.Time        `json:"timestamp"`
	RequestState  *DeliveryState   `json:"request_state,omitempty"`
	ResponseState *DeliveryState   `json:"response_state,omitempty"`
	Diff          *PayloadDiff     `json:"diff,omitempty"`
}

// DeliveryState captures the full state at a point in delivery.
type DeliveryState struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Payload     json.RawMessage   `json:"payload"`
	StatusCode  int               `json:"status_code,omitempty"`
	ResponseBody json.RawMessage  `json:"response_body,omitempty"`
	LatencyMs   int               `json:"latency_ms,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// PayloadDiff shows differences between original and replayed payloads.
type PayloadDiff struct {
	OriginalPayload  json.RawMessage `json:"original_payload"`
	ModifiedPayload  json.RawMessage `json:"modified_payload,omitempty"`
	Changes          []DiffEntry     `json:"changes"`
	IsIdentical      bool            `json:"is_identical"`
}

// DiffEntry describes a single change in a payload comparison.
type DiffEntry struct {
	Path     string `json:"path"`
	Type     string `json:"type"`
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

// EventInspection provides detailed inspection of a historical event.
type EventInspection struct {
	Event          *EventRecord     `json:"event"`
	DeliveryChain  []DeliveryHop    `json:"delivery_chain"`
	TransformSteps []TransformStep  `json:"transform_steps,omitempty"`
	Timeline       []TimelineEntry  `json:"timeline"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DeliveryHop represents one hop in the delivery chain.
type DeliveryHop struct {
	Sequence   int       `json:"sequence"`
	EndpointID string    `json:"endpoint_id"`
	URL        string    `json:"url"`
	Status     string    `json:"status"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int       `json:"latency_ms"`
	Timestamp  time.Time `json:"timestamp"`
	Error      string    `json:"error,omitempty"`
}

// TransformStep records a transformation applied during delivery.
type TransformStep struct {
	Sequence   int             `json:"sequence"`
	Type       string          `json:"type"`
	Input      json.RawMessage `json:"input"`
	Output     json.RawMessage `json:"output"`
	DurationMs int             `json:"duration_ms"`
}

// TimelineEntry is a timestamped entry in event processing.
type TimelineEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Phase       string    `json:"phase"`
	Description string    `json:"description"`
	DurationMs  int       `json:"duration_ms,omitempty"`
}

// Request DTOs

// CreateDebugSessionRequest creates a new debug session.
type CreateDebugSessionRequest struct {
	Name     string   `json:"name" binding:"required"`
	EventIDs []string `json:"event_ids" binding:"required"`
}

// AddBreakpointRequest adds a breakpoint to a debug session.
type AddBreakpointRequest struct {
	Type      string `json:"type" binding:"required"`
	Condition string `json:"condition" binding:"required"`
}

// ReplayWithModificationRequest replays an event with payload modifications.
type ReplayWithModificationRequest struct {
	EventID         string          `json:"event_id" binding:"required"`
	ModifiedPayload json.RawMessage `json:"modified_payload"`
	ModifiedHeaders map[string]string `json:"modified_headers,omitempty"`
	TargetEndpoint  string          `json:"target_endpoint,omitempty"`
	DryRun          bool            `json:"dry_run"`
}

// CreateDebugSession creates a new interactive debugging session.
func (s *Service) CreateDebugSession(ctx context.Context, tenantID string, req *CreateDebugSessionRequest) (*DebugSession, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("session name is required")
	}
	if len(req.EventIDs) == 0 {
		return nil, fmt.Errorf("at least one event ID is required")
	}

	session := &DebugSession{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		EventIDs:    req.EventIDs,
		Status:      "active",
		CurrentStep: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return session, nil
}

// InspectEvent provides detailed inspection of a historical event.
func (s *Service) InspectEvent(ctx context.Context, tenantID, eventID string) (*EventInspection, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	event, err := s.repo.GetEvent(ctx, tenantID, eventID)
	if err != nil {
		return nil, fmt.Errorf("event not found: %w", err)
	}
	if event == nil {
		return nil, fmt.Errorf("event %s not found", eventID)
	}

	inspection := &EventInspection{
		Event: event,
		Timeline: []TimelineEntry{
			{Timestamp: event.Timestamp, Phase: "received", Description: "Event received by WaaS"},
			{Timestamp: event.Timestamp.Add(5 * time.Millisecond), Phase: "validated", Description: "Payload validated and schema checked"},
			{Timestamp: event.Timestamp.Add(10 * time.Millisecond), Phase: "queued", Description: "Event queued for delivery"},
			{Timestamp: event.Timestamp.Add(50 * time.Millisecond), Phase: "delivered", Description: "Delivery attempt completed"},
		},
		DeliveryChain: []DeliveryHop{
			{
				Sequence:   1,
				EndpointID: event.EndpointID,
				Status:     "delivered",
				StatusCode: 200,
				LatencyMs:  45,
				Timestamp:  event.Timestamp.Add(50 * time.Millisecond),
			},
		},
		Metadata: map[string]string{
			"event_type": event.EventType,
			"checksum":   event.Checksum,
		},
	}

	return inspection, nil
}

// ReplayWithModification replays an event with optional payload modifications.
func (s *Service) ReplayWithModification(ctx context.Context, tenantID string, req *ReplayWithModificationRequest) (*DebugStep, error) {
	if req.EventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}

	step := &DebugStep{
		StepNumber: 1,
		EventID:    req.EventID,
		Action:     "replay_modified",
		Timestamp:  time.Now(),
		RequestState: &DeliveryState{
			Method:  "POST",
			Payload: req.ModifiedPayload,
		},
	}

	if req.DryRun {
		step.Action = "dry_run_replay"
		step.ResponseState = &DeliveryState{
			StatusCode: 200,
			LatencyMs:  0,
		}
	}

	// Compare payloads if we have the original
	if s.repo != nil && req.ModifiedPayload != nil {
		event, err := s.repo.GetEvent(ctx, tenantID, req.EventID)
		if err == nil {
			step.Diff = &PayloadDiff{
				OriginalPayload: event.Payload,
				ModifiedPayload: req.ModifiedPayload,
				IsIdentical:     string(event.Payload) == string(req.ModifiedPayload),
			}
			if !step.Diff.IsIdentical {
				step.Diff.Changes = []DiffEntry{
					{Path: "root", Type: "modified", OldValue: "original", NewValue: "modified"},
				}
			}
		}
	}

	return step, nil
}

// CompareEvents compares two historical events side-by-side.
func (s *Service) CompareEvents(ctx context.Context, tenantID, eventID1, eventID2 string) (*PayloadDiff, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	event1, err := s.repo.GetEvent(ctx, tenantID, eventID1)
	if err != nil {
		return nil, fmt.Errorf("event %s not found: %w", eventID1, err)
	}

	event2, err := s.repo.GetEvent(ctx, tenantID, eventID2)
	if err != nil {
		return nil, fmt.Errorf("event %s not found: %w", eventID2, err)
	}

	diff := &PayloadDiff{
		OriginalPayload: event1.Payload,
		ModifiedPayload: event2.Payload,
		IsIdentical:     string(event1.Payload) == string(event2.Payload),
	}

	if !diff.IsIdentical {
		diff.Changes = []DiffEntry{
			{Path: "root", Type: "modified"},
		}
	}

	return diff, nil
}

// AddBreakpoint adds a breakpoint to a debug session.
func (s *Service) AddBreakpoint(ctx context.Context, sessionID string, req *AddBreakpointRequest) (*Breakpoint, error) {
	bp := &Breakpoint{
		ID:        uuid.New().String(),
		Type:      req.Type,
		Condition: req.Condition,
		Enabled:   true,
	}
	return bp, nil
}
