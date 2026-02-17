package collabdebug

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the data access interface for collaborative debugging
type Repository interface {
	// Sessions
	CreateSession(ctx context.Context, session *DebugSession) error
	GetSession(ctx context.Context, tenantID, sessionID string) (*DebugSession, error)
	UpdateSession(ctx context.Context, session *DebugSession) error
	ListSessions(ctx context.Context, tenantID string, limit, offset int) ([]DebugSession, int, error)
	CloseSession(ctx context.Context, tenantID, sessionID string) error

	// Participants
	AddParticipant(ctx context.Context, sessionID string, participant *Participant) error
	RemoveParticipant(ctx context.Context, sessionID, userID string) error
	UpdateParticipant(ctx context.Context, sessionID string, participant *Participant) error
	GetParticipants(ctx context.Context, sessionID string) ([]Participant, error)

	// Annotations
	CreateAnnotation(ctx context.Context, annotation *Annotation) error
	GetAnnotations(ctx context.Context, sessionID string) ([]Annotation, error)
	ResolveAnnotation(ctx context.Context, annotationID string) error
	DeleteAnnotation(ctx context.Context, annotationID string) error

	// Activities
	SaveActivity(ctx context.Context, activity *SessionActivity) error
	GetActivities(ctx context.Context, sessionID string, limit, offset int) ([]SessionActivity, error)

	// Recordings
	SaveRecording(ctx context.Context, recording *SessionRecording) error
	GetRecording(ctx context.Context, sessionID string) (*SessionRecording, error)

	// Summary
	GetSessionSummary(ctx context.Context, tenantID string) (*SessionSummary, error)
}

// PostgresRepository implements Repository with PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session *DebugSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	now := time.Now()
	session.CreatedAt = now
	session.ExpiresAt = now.Add(24 * time.Hour)
	session.Status = SessionStatusActive

	query := `INSERT INTO collab_sessions (id, tenant_id, name, delivery_id, webhook_id, status, created_by, created_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.TenantID, session.Name, session.DeliveryID,
		session.WebhookID, session.Status, session.CreatedBy, session.CreatedAt, session.ExpiresAt)
	return err
}

func (r *PostgresRepository) GetSession(ctx context.Context, tenantID, sessionID string) (*DebugSession, error) {
	var session DebugSession
	err := r.db.GetContext(ctx, &session,
		`SELECT * FROM collab_sessions WHERE id = $1 AND tenant_id = $2`, sessionID, tenantID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return &session, err
}

func (r *PostgresRepository) UpdateSession(ctx context.Context, session *DebugSession) error {
	query := `UPDATE collab_sessions SET name=$1, status=$2 WHERE id=$3`
	_, err := r.db.ExecContext(ctx, query, session.Name, session.Status, session.ID)
	return err
}

func (r *PostgresRepository) ListSessions(ctx context.Context, tenantID string, limit, offset int) ([]DebugSession, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM collab_sessions WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, 0, err
	}
	var sessions []DebugSession
	err = r.db.SelectContext(ctx, &sessions,
		`SELECT * FROM collab_sessions WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	return sessions, total, err
}

func (r *PostgresRepository) CloseSession(ctx context.Context, tenantID, sessionID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE collab_sessions SET status = 'closed' WHERE id = $1 AND tenant_id = $2`,
		sessionID, tenantID)
	return err
}

func (r *PostgresRepository) AddParticipant(ctx context.Context, sessionID string, p *Participant) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now()
	p.JoinedAt = now
	p.LastActiveAt = now
	p.IsOnline = true
	query := `INSERT INTO collab_participants (id, tenant_id, session_id, user_id, display_name, role, joined_at, last_active_at, is_online)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.TenantID, sessionID, p.UserID, p.DisplayName, p.Role, p.JoinedAt, p.LastActiveAt, p.IsOnline)
	return err
}

func (r *PostgresRepository) RemoveParticipant(ctx context.Context, sessionID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE collab_participants SET is_online = false WHERE session_id = $1 AND user_id = $2`,
		sessionID, userID)
	return err
}

func (r *PostgresRepository) UpdateParticipant(ctx context.Context, sessionID string, p *Participant) error {
	p.LastActiveAt = time.Now()
	query := `UPDATE collab_participants SET cursor_position=$1, last_active_at=$2, is_online=$3 WHERE session_id=$4 AND user_id=$5`
	_, err := r.db.ExecContext(ctx, query, p.CursorPosition, p.LastActiveAt, p.IsOnline, sessionID, p.UserID)
	return err
}

func (r *PostgresRepository) GetParticipants(ctx context.Context, sessionID string) ([]Participant, error) {
	var participants []Participant
	err := r.db.SelectContext(ctx, &participants,
		`SELECT * FROM collab_participants WHERE session_id = $1 ORDER BY joined_at ASC`, sessionID)
	return participants, err
}

func (r *PostgresRepository) CreateAnnotation(ctx context.Context, a *Annotation) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	a.CreatedAt = time.Now()
	query := `INSERT INTO collab_annotations (id, session_id, author_id, author_name, target_field, content, annotation_type, resolved, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.SessionID, a.AuthorID, a.AuthorName, a.TargetField, a.Content, a.AnnotationType, false, a.CreatedAt)
	return err
}

func (r *PostgresRepository) GetAnnotations(ctx context.Context, sessionID string) ([]Annotation, error) {
	var annotations []Annotation
	err := r.db.SelectContext(ctx, &annotations,
		`SELECT * FROM collab_annotations WHERE session_id = $1 ORDER BY created_at ASC`, sessionID)
	return annotations, err
}

func (r *PostgresRepository) ResolveAnnotation(ctx context.Context, annotationID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE collab_annotations SET resolved = true, resolved_at = $1 WHERE id = $2`,
		now, annotationID)
	return err
}

func (r *PostgresRepository) DeleteAnnotation(ctx context.Context, annotationID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collab_annotations WHERE id = $1`, annotationID)
	return err
}

func (r *PostgresRepository) SaveActivity(ctx context.Context, a *SessionActivity) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	a.Timestamp = time.Now()
	query := `INSERT INTO collab_activities (id, session_id, user_id, action, details, timestamp)
		VALUES ($1,$2,$3,$4,$5,$6)`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.SessionID, a.UserID, a.Action, a.Details, a.Timestamp)
	return err
}

func (r *PostgresRepository) GetActivities(ctx context.Context, sessionID string, limit, offset int) ([]SessionActivity, error) {
	var activities []SessionActivity
	err := r.db.SelectContext(ctx, &activities,
		`SELECT * FROM collab_activities WHERE session_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`,
		sessionID, limit, offset)
	return activities, err
}

func (r *PostgresRepository) SaveRecording(ctx context.Context, recording *SessionRecording) error {
	if recording.ID == "" {
		recording.ID = uuid.New().String()
	}
	recording.CreatedAt = time.Now()
	query := `INSERT INTO collab_recordings (id, session_id, duration, created_at) VALUES ($1,$2,$3,$4)`
	_, err := r.db.ExecContext(ctx, query, recording.ID, recording.SessionID, recording.Duration, recording.CreatedAt)
	return err
}

func (r *PostgresRepository) GetRecording(ctx context.Context, sessionID string) (*SessionRecording, error) {
	var recording SessionRecording
	err := r.db.GetContext(ctx, &recording,
		`SELECT * FROM collab_recordings WHERE session_id = $1`, sessionID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &recording, err
}

func (r *PostgresRepository) GetSessionSummary(ctx context.Context, tenantID string) (*SessionSummary, error) {
	summary := &SessionSummary{}
	// best-effort queries: individual count failures leave zero-value, which is acceptable for summary
	_ = r.db.GetContext(ctx, &summary.TotalSessions,
		`SELECT COUNT(*) FROM collab_sessions WHERE tenant_id = $1`, tenantID)
	_ = r.db.GetContext(ctx, &summary.ActiveSessions,
		`SELECT COUNT(*) FROM collab_sessions WHERE tenant_id = $1 AND status = 'active'`, tenantID)
	_ = r.db.GetContext(ctx, &summary.AnnotationsCreated,
		`SELECT COUNT(*) FROM collab_annotations a JOIN collab_sessions s ON a.session_id = s.id WHERE s.tenant_id = $1`, tenantID)
	return summary, nil
}
