package collabdebug

import "time"

// SessionStatus constants
const (
	SessionStatusActive = "active"
	SessionStatusPaused = "paused"
	SessionStatusClosed = "closed"
)

// ParticipantRole constants
const (
	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// AnnotationType constants
const (
	AnnotationComment   = "comment"
	AnnotationHighlight = "highlight"
	AnnotationSuggestion = "suggestion"
	AnnotationBugReport = "bug_report"
)

// DebugSession represents a collaborative debugging session
type DebugSession struct {
	ID           string        `json:"id" db:"id"`
	TenantID     string        `json:"tenant_id" db:"tenant_id"`
	Name         string        `json:"name" db:"name"`
	DeliveryID   string        `json:"delivery_id" db:"delivery_id"`
	WebhookID    string        `json:"webhook_id" db:"webhook_id"`
	Status       string        `json:"status" db:"status"`
	Participants []Participant `json:"participants"`
	Annotations  []Annotation  `json:"annotations"`
	CreatedBy    string        `json:"created_by" db:"created_by"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	ExpiresAt    time.Time     `json:"expires_at" db:"expires_at"`
}

// Participant represents a user in a collaborative debugging session
type Participant struct {
	ID             string    `json:"id" db:"id"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	UserID         string    `json:"user_id" db:"user_id"`
	DisplayName    string    `json:"display_name" db:"display_name"`
	Role           string    `json:"role" db:"role"`
	CursorPosition string    `json:"cursor_position" db:"cursor_position"`
	JoinedAt       time.Time `json:"joined_at" db:"joined_at"`
	LastActiveAt   time.Time `json:"last_active_at" db:"last_active_at"`
	IsOnline       bool      `json:"is_online" db:"is_online"`
}

// Annotation represents a note or comment on a specific field in a debug session
type Annotation struct {
	ID             string     `json:"id" db:"id"`
	SessionID      string     `json:"session_id" db:"session_id"`
	AuthorID       string     `json:"author_id" db:"author_id"`
	AuthorName     string     `json:"author_name" db:"author_name"`
	TargetField    string     `json:"target_field" db:"target_field"`
	Content        string     `json:"content" db:"content"`
	AnnotationType string     `json:"annotation_type" db:"annotation_type"`
	Resolved       bool       `json:"resolved" db:"resolved"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// SharedState represents the shared view state across all participants
type SharedState struct {
	SessionID         string            `json:"session_id" db:"session_id"`
	SelectedDelivery  string            `json:"selected_delivery" db:"selected_delivery"`
	Filters           map[string]string `json:"filters"`
	FiltersJSON       string            `json:"-" db:"filters"`
	ViewMode          string            `json:"view_mode" db:"view_mode"`
	HighlightedFields []string          `json:"highlighted_fields"`
	HighlightedJSON   string            `json:"-" db:"highlighted_fields"`
	UpdatedAt         time.Time         `json:"updated_at" db:"updated_at"`
}

// SessionActivity represents a single action taken by a user in a session
type SessionActivity struct {
	ID        string    `json:"id" db:"id"`
	SessionID string    `json:"session_id" db:"session_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Action    string    `json:"action" db:"action"`
	Details   string    `json:"details" db:"details"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

// SessionRecording captures the full activity history for replay
type SessionRecording struct {
	ID         string            `json:"id" db:"id"`
	SessionID  string            `json:"session_id" db:"session_id"`
	Activities []SessionActivity `json:"activities"`
	Duration   int               `json:"duration_seconds" db:"duration_seconds"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
}

// SessionSummary provides aggregate statistics for collaborative debugging
type SessionSummary struct {
	TotalSessions      int     `json:"total_sessions"`
	ActiveSessions     int     `json:"active_sessions"`
	AvgParticipants    float64 `json:"avg_participants"`
	AvgDuration        float64 `json:"avg_duration_seconds"`
	AnnotationsCreated int     `json:"annotations_created"`
}

// CreateSessionRequest is the request DTO for creating a collaborative debug session
type CreateSessionRequest struct {
	Name       string `json:"name" binding:"required"`
	DeliveryID string `json:"delivery_id,omitempty"`
	WebhookID  string `json:"webhook_id,omitempty"`
}

// JoinSessionRequest is the request DTO for joining a session
type JoinSessionRequest struct {
	UserID      string `json:"user_id" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Role        string `json:"role,omitempty"`
}

// CreateAnnotationRequest is the request DTO for adding an annotation
type CreateAnnotationRequest struct {
	AuthorID       string `json:"author_id" binding:"required"`
	AuthorName     string `json:"author_name" binding:"required"`
	TargetField    string `json:"target_field" binding:"required"`
	Content        string `json:"content" binding:"required"`
	AnnotationType string `json:"annotation_type,omitempty"`
}

// UpdateCursorRequest is the request DTO for updating a participant's cursor position
type UpdateCursorRequest struct {
	UserID         string `json:"user_id" binding:"required"`
	CursorPosition string `json:"cursor_position" binding:"required"`
}
