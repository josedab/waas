package models

import (
	"time"

	"github.com/google/uuid"
)

// Team roles
const (
	TeamRoleOwner  = "owner"
	TeamRoleAdmin  = "admin"
	TeamRoleMember = "member"
	TeamRoleViewer = "viewer"
)

// Change request statuses
const (
	ChangeRequestPending  = "pending"
	ChangeRequestApproved = "approved"
	ChangeRequestRejected = "rejected"
	ChangeRequestMerged   = "merged"
	ChangeRequestClosed   = "closed"
)

// Review statuses
const (
	ReviewApproved         = "approved"
	ReviewChangesRequested = "changes_requested"
	ReviewCommented        = "commented"
)

// Notification channels
const (
	NotificationChannelEmail   = "email"
	NotificationChannelSlack   = "slack"
	NotificationChannelTeams   = "teams"
	NotificationChannelWebhook = "webhook"
)

// Team represents a collaboration workspace
type Team struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	TenantID    uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Settings    map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// CollabTeamMember represents a member of a team
type CollabTeamMember struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TeamID      uuid.UUID  `json:"team_id" db:"team_id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Email       string     `json:"email" db:"email"`
	Role        string     `json:"role" db:"role"`
	Permissions []string   `json:"permissions" db:"permissions"`
	InvitedBy   *uuid.UUID `json:"invited_by,omitempty" db:"invited_by"`
	JoinedAt    *time.Time `json:"joined_at,omitempty" db:"joined_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// SharedConfiguration represents a team-shared webhook configuration
type SharedConfiguration struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	TeamID      uuid.UUID              `json:"team_id" db:"team_id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	ConfigType  string                 `json:"config_type" db:"config_type"`
	ConfigData  map[string]interface{} `json:"config_data" db:"config_data"`
	Version     int                    `json:"version" db:"version"`
	IsLocked    bool                   `json:"is_locked" db:"is_locked"`
	LockedBy    *uuid.UUID             `json:"locked_by,omitempty" db:"locked_by"`
	LockedAt    *time.Time             `json:"locked_at,omitempty" db:"locked_at"`
	CreatedBy   uuid.UUID              `json:"created_by" db:"created_by"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// ConfigVersion represents a historical version of a configuration
type ConfigVersion struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	ConfigID      uuid.UUID              `json:"config_id" db:"config_id"`
	Version       int                    `json:"version" db:"version"`
	ConfigData    map[string]interface{} `json:"config_data" db:"config_data"`
	ChangeSummary string                 `json:"change_summary" db:"change_summary"`
	ChangedBy     uuid.UUID              `json:"changed_by" db:"changed_by"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// ChangeRequest represents a PR-style change request
type ChangeRequest struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	TeamID          uuid.UUID              `json:"team_id" db:"team_id"`
	ConfigID        uuid.UUID              `json:"config_id" db:"config_id"`
	Title           string                 `json:"title" db:"title"`
	Description     string                 `json:"description" db:"description"`
	Status          string                 `json:"status" db:"status"`
	ProposedChanges map[string]interface{} `json:"proposed_changes" db:"proposed_changes"`
	BaseVersion     int                    `json:"base_version" db:"base_version"`
	CreatedBy       uuid.UUID              `json:"created_by" db:"created_by"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// ChangeRequestReview represents a review on a change request
type ChangeRequestReview struct {
	ID              uuid.UUID `json:"id" db:"id"`
	ChangeRequestID uuid.UUID `json:"change_request_id" db:"change_request_id"`
	ReviewerID      uuid.UUID `json:"reviewer_id" db:"reviewer_id"`
	Status          string    `json:"status" db:"status"`
	Comments        string    `json:"comments" db:"comments"`
	ReviewedAt      time.Time `json:"reviewed_at" db:"reviewed_at"`
}

// ChangeRequestComment represents a comment on a change request
type ChangeRequestComment struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	ChangeRequestID uuid.UUID  `json:"change_request_id" db:"change_request_id"`
	AuthorID        uuid.UUID  `json:"author_id" db:"author_id"`
	Content         string     `json:"content" db:"content"`
	ParentID        *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// ActivityFeedItem represents an activity in the team feed
type ActivityFeedItem struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	TeamID       uuid.UUID              `json:"team_id" db:"team_id"`
	ActorID      *uuid.UUID             `json:"actor_id,omitempty" db:"actor_id"`
	ActionType   string                 `json:"action_type" db:"action_type"`
	ResourceType string                 `json:"resource_type" db:"resource_type"`
	ResourceID   *uuid.UUID             `json:"resource_id,omitempty" db:"resource_id"`
	ResourceName string                 `json:"resource_name" db:"resource_name"`
	Details      map[string]interface{} `json:"details" db:"details"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// NotificationPreference represents user notification settings
type NotificationPreference struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	TeamMemberID uuid.UUID              `json:"team_member_id" db:"team_member_id"`
	Channel      string                 `json:"channel" db:"channel"`
	EventTypes   []string               `json:"event_types" db:"event_types"`
	IsEnabled    bool                   `json:"is_enabled" db:"is_enabled"`
	Settings     map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// NotificationIntegration represents a team notification integration (Slack, Teams, etc.)
type NotificationIntegration struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	TeamID          uuid.UUID              `json:"team_id" db:"team_id"`
	IntegrationType string                 `json:"integration_type" db:"integration_type"`
	Name            string                 `json:"name" db:"name"`
	Config          map[string]interface{} `json:"config" db:"config"`
	IsActive        bool                   `json:"is_active" db:"is_active"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// SentNotification represents a sent notification record
type SentNotification struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TeamID        uuid.UUID  `json:"team_id" db:"team_id"`
	IntegrationID *uuid.UUID `json:"integration_id,omitempty" db:"integration_id"`
	RecipientID   *uuid.UUID `json:"recipient_id,omitempty" db:"recipient_id"`
	Channel       string     `json:"channel" db:"channel"`
	EventType     string     `json:"event_type" db:"event_type"`
	Subject       string     `json:"subject" db:"subject"`
	Content       string     `json:"content" db:"content"`
	Status        string     `json:"status" db:"status"`
	SentAt        *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	ErrorMessage  string     `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// Request/Response types

type CreateTeamRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Settings    map[string]interface{} `json:"settings"`
}

type InviteMemberRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required"`
}

type CreateChangeRequestRequest struct {
	ConfigID        string                 `json:"config_id" binding:"required"`
	Title           string                 `json:"title" binding:"required"`
	Description     string                 `json:"description"`
	ProposedChanges map[string]interface{} `json:"proposed_changes" binding:"required"`
}

type ReviewChangeRequestRequest struct {
	Status   string `json:"status" binding:"required"`
	Comments string `json:"comments"`
}

type UpdateNotificationPreferenceRequest struct {
	Channel    string                 `json:"channel" binding:"required"`
	EventTypes []string               `json:"event_types"`
	IsEnabled  bool                   `json:"is_enabled"`
	Settings   map[string]interface{} `json:"settings"`
}
