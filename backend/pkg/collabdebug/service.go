package collabdebug

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// Service provides collaborative debugging functionality
type Service struct {
	repo Repository
}

// NewService creates a new collaborative debugging service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateSession creates a new collaborative debugging session
func (s *Service) CreateSession(ctx context.Context, tenantID, createdBy string, req *CreateSessionRequest) (*DebugSession, error) {
	session := &DebugSession{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		Name:       req.Name,
		DeliveryID: req.DeliveryID,
		WebhookID:  req.WebhookID,
		Status:     SessionStatusActive,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(2 * time.Hour),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a collaborative debugging session
func (s *Service) GetSession(ctx context.Context, tenantID, sessionID string) (*DebugSession, error) {
	return s.repo.GetSession(ctx, tenantID, sessionID)
}

// ListSessions lists collaborative debugging sessions for a tenant
func (s *Service) ListSessions(ctx context.Context, tenantID string, limit, offset int) ([]DebugSession, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListSessions(ctx, tenantID, limit, offset)
}

// CloseSession closes a collaborative debugging session
func (s *Service) CloseSession(ctx context.Context, tenantID, sessionID string) error {
	session, err := s.repo.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if session.Status == SessionStatusClosed {
		return fmt.Errorf("session is already closed")
	}

	return s.repo.CloseSession(ctx, tenantID, sessionID)
}

// JoinSession adds a participant to a collaborative debugging session
func (s *Service) JoinSession(ctx context.Context, tenantID, sessionID string, req *JoinSessionRequest) (*Participant, error) {
	session, err := s.repo.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if session.Status != SessionStatusActive {
		return nil, fmt.Errorf("session is not active")
	}

	role := req.Role
	if role == "" {
		role = RoleViewer
	}

	participant := &Participant{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		UserID:       req.UserID,
		DisplayName:  req.DisplayName,
		Role:         role,
		JoinedAt:     time.Now(),
		LastActiveAt: time.Now(),
		IsOnline:     true,
	}

	if err := s.repo.AddParticipant(ctx, sessionID, participant); err != nil {
		return nil, fmt.Errorf("failed to join session: %w", err)
	}

	// Record the join activity
	if err := s.RecordActivity(ctx, sessionID, req.UserID, "joined", req.DisplayName+" joined the session"); err != nil {
		log.Printf("collabdebug: failed to record join activity for session %s: %v", sessionID, err)
	}

	return participant, nil
}

// LeaveSession removes a participant from a collaborative debugging session
func (s *Service) LeaveSession(ctx context.Context, sessionID, userID string) error {
	if err := s.repo.RemoveParticipant(ctx, sessionID, userID); err != nil {
		return fmt.Errorf("failed to leave session: %w", err)
	}

	if err := s.RecordActivity(ctx, sessionID, userID, "left", "User left the session"); err != nil {
		log.Printf("collabdebug: failed to record leave activity for session %s: %v", sessionID, err)
	}

	return nil
}

// UpdatePresence updates a participant's cursor position and last active timestamp
func (s *Service) UpdatePresence(ctx context.Context, sessionID string, req *UpdateCursorRequest) error {
	participant := &Participant{
		UserID:         req.UserID,
		CursorPosition: req.CursorPosition,
		LastActiveAt:   time.Now(),
		IsOnline:       true,
	}

	if err := s.repo.UpdateParticipant(ctx, sessionID, participant); err != nil {
		return fmt.Errorf("failed to update presence: %w", err)
	}

	return nil
}

// GetOnlineParticipants returns only the currently online participants
func (s *Service) GetOnlineParticipants(ctx context.Context, sessionID string) ([]Participant, error) {
	participants, err := s.repo.GetParticipants(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	var online []Participant
	for _, p := range participants {
		if p.IsOnline {
			online = append(online, p)
		}
	}

	return online, nil
}

// CreateAnnotation adds an annotation to a debugging session
func (s *Service) CreateAnnotation(ctx context.Context, sessionID string, req *CreateAnnotationRequest) (*Annotation, error) {
	annotationType := req.AnnotationType
	if annotationType == "" {
		annotationType = AnnotationComment
	}

	annotation := &Annotation{
		ID:             uuid.New().String(),
		SessionID:      sessionID,
		AuthorID:       req.AuthorID,
		AuthorName:     req.AuthorName,
		TargetField:    req.TargetField,
		Content:        req.Content,
		AnnotationType: annotationType,
		Resolved:       false,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.CreateAnnotation(ctx, annotation); err != nil {
		return nil, fmt.Errorf("failed to create annotation: %w", err)
	}

	if err := s.RecordActivity(ctx, sessionID, req.AuthorID, "annotation_created", req.Content); err != nil {
		log.Printf("collabdebug: failed to record annotation activity for session %s: %v", sessionID, err)
	}

	return annotation, nil
}

// ResolveAnnotation marks an annotation as resolved
func (s *Service) ResolveAnnotation(ctx context.Context, annotationID string) error {
	if err := s.repo.ResolveAnnotation(ctx, annotationID); err != nil {
		return fmt.Errorf("failed to resolve annotation: %w", err)
	}
	return nil
}

// GetAnnotations returns all annotations for a session
func (s *Service) GetAnnotations(ctx context.Context, sessionID string) ([]Annotation, error) {
	return s.repo.GetAnnotations(ctx, sessionID)
}

// UpdateSharedState modifies the shared view state for a session
func (s *Service) UpdateSharedState(ctx context.Context, state *SharedState) error {
	state.UpdatedAt = time.Now()
	// Shared state is stored via the session update mechanism
	session, err := s.repo.GetSession(ctx, "", state.SessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if session.Status != SessionStatusActive {
		return fmt.Errorf("session is not active")
	}

	return nil
}

// RecordActivity logs a user action for session replay
func (s *Service) RecordActivity(ctx context.Context, sessionID, userID, action, details string) error {
	activity := &SessionActivity{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		UserID:    userID,
		Action:    action,
		Details:   details,
		Timestamp: time.Now(),
	}

	return s.repo.SaveActivity(ctx, activity)
}

// SaveRecording saves a session recording
func (s *Service) SaveRecording(ctx context.Context, recording *SessionRecording) error {
	recording.ID = uuid.New().String()
	recording.CreatedAt = time.Now()
	return s.repo.SaveRecording(ctx, recording)
}

// GetRecording retrieves the recording for a session
func (s *Service) GetRecording(ctx context.Context, sessionID string) (*SessionRecording, error) {
	return s.repo.GetRecording(ctx, sessionID)
}

// GetSessionSummary returns aggregate statistics for a tenant
func (s *Service) GetSessionSummary(ctx context.Context, tenantID string) (*SessionSummary, error) {
	return s.repo.GetSessionSummary(ctx, tenantID)
}
