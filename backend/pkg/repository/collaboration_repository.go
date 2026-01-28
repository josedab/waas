package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/models"
)

type CollaborationRepository interface {
	// Team operations
	CreateTeam(ctx context.Context, team *models.Team) error
	GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error)
	GetTeamsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.Team, error)
	UpdateTeam(ctx context.Context, team *models.Team) error
	DeleteTeam(ctx context.Context, id uuid.UUID) error

	// Team member operations
	AddTeamMember(ctx context.Context, member *models.CollabTeamMember) error
	GetTeamMember(ctx context.Context, id uuid.UUID) (*models.CollabTeamMember, error)
	GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*models.CollabTeamMember, error)
	GetTeamMemberByEmail(ctx context.Context, teamID uuid.UUID, email string) (*models.CollabTeamMember, error)
	UpdateTeamMember(ctx context.Context, member *models.CollabTeamMember) error
	RemoveTeamMember(ctx context.Context, id uuid.UUID) error

	// Shared configuration operations
	CreateSharedConfig(ctx context.Context, config *models.SharedConfiguration) error
	GetSharedConfig(ctx context.Context, id uuid.UUID) (*models.SharedConfiguration, error)
	GetTeamConfigs(ctx context.Context, teamID uuid.UUID) ([]*models.SharedConfiguration, error)
	UpdateSharedConfig(ctx context.Context, config *models.SharedConfiguration) error
	DeleteSharedConfig(ctx context.Context, id uuid.UUID) error
	LockConfig(ctx context.Context, id, lockedBy uuid.UUID) error
	UnlockConfig(ctx context.Context, id uuid.UUID) error

	// Config version operations
	SaveConfigVersion(ctx context.Context, version *models.ConfigVersion) error
	GetConfigVersions(ctx context.Context, configID uuid.UUID) ([]*models.ConfigVersion, error)
	GetConfigVersion(ctx context.Context, configID uuid.UUID, version int) (*models.ConfigVersion, error)

	// Change request operations
	CreateChangeRequest(ctx context.Context, cr *models.ChangeRequest) error
	GetChangeRequest(ctx context.Context, id uuid.UUID) (*models.ChangeRequest, error)
	GetTeamChangeRequests(ctx context.Context, teamID uuid.UUID, status string) ([]*models.ChangeRequest, error)
	UpdateChangeRequest(ctx context.Context, cr *models.ChangeRequest) error

	// Review operations
	AddReview(ctx context.Context, review *models.ChangeRequestReview) error
	GetReviews(ctx context.Context, changeRequestID uuid.UUID) ([]*models.ChangeRequestReview, error)

	// Comment operations
	AddComment(ctx context.Context, comment *models.ChangeRequestComment) error
	GetComments(ctx context.Context, changeRequestID uuid.UUID) ([]*models.ChangeRequestComment, error)

	// Activity feed operations
	AddActivity(ctx context.Context, activity *models.ActivityFeedItem) error
	GetTeamActivity(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*models.ActivityFeedItem, error)

	// Notification operations
	SaveNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error
	GetNotificationPreferences(ctx context.Context, memberID uuid.UUID) ([]*models.NotificationPreference, error)
	CreateNotificationIntegration(ctx context.Context, integration *models.NotificationIntegration) error
	GetNotificationIntegrations(ctx context.Context, teamID uuid.UUID) ([]*models.NotificationIntegration, error)
	RecordSentNotification(ctx context.Context, notification *models.SentNotification) error
}

type collaborationRepository struct {
	db *database.DB
}

func NewCollaborationRepository(db *database.DB) CollaborationRepository {
	return &collaborationRepository{db: db}
}

// Team operations

func (r *collaborationRepository) CreateTeam(ctx context.Context, team *models.Team) error {
	query := `
		INSERT INTO teams (id, tenant_id, name, description, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if team.ID == uuid.Nil {
		team.ID = uuid.New()
	}
	now := time.Now()
	team.CreatedAt = now
	team.UpdatedAt = now

	settingsJSON, _ := json.Marshal(team.Settings)

	_, err := r.db.Pool.Exec(ctx, query,
		team.ID, team.TenantID, team.Name, team.Description, settingsJSON, team.CreatedAt, team.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

func (r *collaborationRepository) GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	query := `SELECT id, tenant_id, name, description, settings, created_at, updated_at FROM teams WHERE id = $1`

	var team models.Team
	var settingsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&team.ID, &team.TenantID, &team.Name, &team.Description, &settingsJSON, &team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	json.Unmarshal(settingsJSON, &team.Settings)
	return &team, nil
}

func (r *collaborationRepository) GetTeamsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.Team, error) {
	query := `SELECT id, tenant_id, name, description, settings, created_at, updated_at FROM teams WHERE tenant_id = $1 ORDER BY name`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		var team models.Team
		var settingsJSON []byte
		if err := rows.Scan(&team.ID, &team.TenantID, &team.Name, &team.Description, &settingsJSON, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(settingsJSON, &team.Settings)
		teams = append(teams, &team)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *collaborationRepository) UpdateTeam(ctx context.Context, team *models.Team) error {
	query := `UPDATE teams SET name = $2, description = $3, settings = $4, updated_at = $5 WHERE id = $1`
	team.UpdatedAt = time.Now()
	settingsJSON, _ := json.Marshal(team.Settings)
	_, err := r.db.Pool.Exec(ctx, query, team.ID, team.Name, team.Description, settingsJSON, team.UpdatedAt)
	return err
}

func (r *collaborationRepository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}

// Team member operations

func (r *collaborationRepository) AddTeamMember(ctx context.Context, member *models.CollabTeamMember) error {
	query := `
		INSERT INTO team_members (id, team_id, user_id, email, role, permissions, invited_by, joined_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	if member.ID == uuid.Nil {
		member.ID = uuid.New()
	}
	member.CreatedAt = time.Now()
	permissionsJSON, _ := json.Marshal(member.Permissions)

	_, err := r.db.Pool.Exec(ctx, query,
		member.ID, member.TeamID, member.UserID, member.Email, member.Role,
		permissionsJSON, member.InvitedBy, member.JoinedAt, member.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}
	return nil
}

func (r *collaborationRepository) GetTeamMember(ctx context.Context, id uuid.UUID) (*models.CollabTeamMember, error) {
	query := `SELECT id, team_id, user_id, email, role, permissions, invited_by, joined_at, created_at FROM team_members WHERE id = $1`

	var member models.CollabTeamMember
	var permissionsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&member.ID, &member.TeamID, &member.UserID, &member.Email, &member.Role,
		&permissionsJSON, &member.InvitedBy, &member.JoinedAt, &member.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get team member: %w", err)
	}

	json.Unmarshal(permissionsJSON, &member.Permissions)
	return &member, nil
}

func (r *collaborationRepository) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*models.CollabTeamMember, error) {
	query := `SELECT id, team_id, user_id, email, role, permissions, invited_by, joined_at, created_at FROM team_members WHERE team_id = $1 ORDER BY created_at`

	rows, err := r.db.Pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.CollabTeamMember
	for rows.Next() {
		var member models.CollabTeamMember
		var permissionsJSON []byte
		if err := rows.Scan(&member.ID, &member.TeamID, &member.UserID, &member.Email, &member.Role,
			&permissionsJSON, &member.InvitedBy, &member.JoinedAt, &member.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(permissionsJSON, &member.Permissions)
		members = append(members, &member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return members, nil
}

func (r *collaborationRepository) GetTeamMemberByEmail(ctx context.Context, teamID uuid.UUID, email string) (*models.CollabTeamMember, error) {
	query := `SELECT id, team_id, user_id, email, role, permissions, invited_by, joined_at, created_at FROM team_members WHERE team_id = $1 AND email = $2`

	var member models.CollabTeamMember
	var permissionsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, teamID, email).Scan(
		&member.ID, &member.TeamID, &member.UserID, &member.Email, &member.Role,
		&permissionsJSON, &member.InvitedBy, &member.JoinedAt, &member.CreatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(permissionsJSON, &member.Permissions)
	return &member, nil
}

func (r *collaborationRepository) UpdateTeamMember(ctx context.Context, member *models.CollabTeamMember) error {
	query := `UPDATE team_members SET role = $2, permissions = $3, joined_at = $4 WHERE id = $1`
	permissionsJSON, _ := json.Marshal(member.Permissions)
	_, err := r.db.Pool.Exec(ctx, query, member.ID, member.Role, permissionsJSON, member.JoinedAt)
	return err
}

func (r *collaborationRepository) RemoveTeamMember(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM team_members WHERE id = $1`, id)
	return err
}

// Shared configuration operations

func (r *collaborationRepository) CreateSharedConfig(ctx context.Context, config *models.SharedConfiguration) error {
	query := `
		INSERT INTO shared_configurations (id, team_id, name, description, config_type, config_data, version, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
	config.Version = 1

	configDataJSON, _ := json.Marshal(config.ConfigData)

	_, err := r.db.Pool.Exec(ctx, query,
		config.ID, config.TeamID, config.Name, config.Description, config.ConfigType,
		configDataJSON, config.Version, config.CreatedBy, config.CreatedAt, config.UpdatedAt)
	return err
}

func (r *collaborationRepository) GetSharedConfig(ctx context.Context, id uuid.UUID) (*models.SharedConfiguration, error) {
	query := `
		SELECT id, team_id, name, description, config_type, config_data, version, is_locked, locked_by, locked_at, created_by, created_at, updated_at
		FROM shared_configurations WHERE id = $1
	`

	var config models.SharedConfiguration
	var configDataJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.TeamID, &config.Name, &config.Description, &config.ConfigType,
		&configDataJSON, &config.Version, &config.IsLocked, &config.LockedBy, &config.LockedAt,
		&config.CreatedBy, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(configDataJSON, &config.ConfigData)
	return &config, nil
}

func (r *collaborationRepository) GetTeamConfigs(ctx context.Context, teamID uuid.UUID) ([]*models.SharedConfiguration, error) {
	query := `
		SELECT id, team_id, name, description, config_type, config_data, version, is_locked, locked_by, locked_at, created_by, created_at, updated_at
		FROM shared_configurations WHERE team_id = $1 ORDER BY name
	`

	rows, err := r.db.Pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.SharedConfiguration
	for rows.Next() {
		var config models.SharedConfiguration
		var configDataJSON []byte
		if err := rows.Scan(&config.ID, &config.TeamID, &config.Name, &config.Description, &config.ConfigType,
			&configDataJSON, &config.Version, &config.IsLocked, &config.LockedBy, &config.LockedAt,
			&config.CreatedBy, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(configDataJSON, &config.ConfigData)
		configs = append(configs, &config)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *collaborationRepository) UpdateSharedConfig(ctx context.Context, config *models.SharedConfiguration) error {
	query := `
		UPDATE shared_configurations 
		SET name = $2, description = $3, config_data = $4, version = version + 1, updated_at = $5
		WHERE id = $1
		RETURNING version
	`
	config.UpdatedAt = time.Now()
	configDataJSON, _ := json.Marshal(config.ConfigData)
	return r.db.Pool.QueryRow(ctx, query, config.ID, config.Name, config.Description, configDataJSON, config.UpdatedAt).Scan(&config.Version)
}

func (r *collaborationRepository) DeleteSharedConfig(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM shared_configurations WHERE id = $1`, id)
	return err
}

func (r *collaborationRepository) LockConfig(ctx context.Context, id, lockedBy uuid.UUID) error {
	query := `UPDATE shared_configurations SET is_locked = true, locked_by = $2, locked_at = $3 WHERE id = $1 AND is_locked = false`
	result, err := r.db.Pool.Exec(ctx, query, id, lockedBy, time.Now())
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("config is already locked or not found")
	}
	return nil
}

func (r *collaborationRepository) UnlockConfig(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE shared_configurations SET is_locked = false, locked_by = NULL, locked_at = NULL WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

// Config version operations

func (r *collaborationRepository) SaveConfigVersion(ctx context.Context, version *models.ConfigVersion) error {
	query := `
		INSERT INTO config_versions (id, config_id, version, config_data, change_summary, changed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if version.ID == uuid.Nil {
		version.ID = uuid.New()
	}
	version.CreatedAt = time.Now()
	configDataJSON, _ := json.Marshal(version.ConfigData)

	_, err := r.db.Pool.Exec(ctx, query,
		version.ID, version.ConfigID, version.Version, configDataJSON, version.ChangeSummary, version.ChangedBy, version.CreatedAt)
	return err
}

func (r *collaborationRepository) GetConfigVersions(ctx context.Context, configID uuid.UUID) ([]*models.ConfigVersion, error) {
	query := `SELECT id, config_id, version, config_data, change_summary, changed_by, created_at FROM config_versions WHERE config_id = $1 ORDER BY version DESC`

	rows, err := r.db.Pool.Query(ctx, query, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*models.ConfigVersion
	for rows.Next() {
		var v models.ConfigVersion
		var configDataJSON []byte
		if err := rows.Scan(&v.ID, &v.ConfigID, &v.Version, &configDataJSON, &v.ChangeSummary, &v.ChangedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(configDataJSON, &v.ConfigData)
		versions = append(versions, &v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *collaborationRepository) GetConfigVersion(ctx context.Context, configID uuid.UUID, version int) (*models.ConfigVersion, error) {
	query := `SELECT id, config_id, version, config_data, change_summary, changed_by, created_at FROM config_versions WHERE config_id = $1 AND version = $2`

	var v models.ConfigVersion
	var configDataJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, configID, version).Scan(
		&v.ID, &v.ConfigID, &v.Version, &configDataJSON, &v.ChangeSummary, &v.ChangedBy, &v.CreatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(configDataJSON, &v.ConfigData)
	return &v, nil
}

// Change request operations

func (r *collaborationRepository) CreateChangeRequest(ctx context.Context, cr *models.ChangeRequest) error {
	query := `
		INSERT INTO change_requests (id, team_id, config_id, title, description, status, proposed_changes, base_version, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if cr.ID == uuid.Nil {
		cr.ID = uuid.New()
	}
	now := time.Now()
	cr.CreatedAt = now
	cr.UpdatedAt = now
	cr.Status = models.ChangeRequestPending

	proposedJSON, _ := json.Marshal(cr.ProposedChanges)

	_, err := r.db.Pool.Exec(ctx, query,
		cr.ID, cr.TeamID, cr.ConfigID, cr.Title, cr.Description, cr.Status,
		proposedJSON, cr.BaseVersion, cr.CreatedBy, cr.CreatedAt, cr.UpdatedAt)
	return err
}

func (r *collaborationRepository) GetChangeRequest(ctx context.Context, id uuid.UUID) (*models.ChangeRequest, error) {
	query := `
		SELECT id, team_id, config_id, title, description, status, proposed_changes, base_version, created_by, created_at, updated_at
		FROM change_requests WHERE id = $1
	`

	var cr models.ChangeRequest
	var proposedJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&cr.ID, &cr.TeamID, &cr.ConfigID, &cr.Title, &cr.Description, &cr.Status,
		&proposedJSON, &cr.BaseVersion, &cr.CreatedBy, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(proposedJSON, &cr.ProposedChanges)
	return &cr, nil
}

func (r *collaborationRepository) GetTeamChangeRequests(ctx context.Context, teamID uuid.UUID, status string) ([]*models.ChangeRequest, error) {
	query := `
		SELECT id, team_id, config_id, title, description, status, proposed_changes, base_version, created_by, created_at, updated_at
		FROM change_requests WHERE team_id = $1
	`
	args := []interface{}{teamID}

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var crs []*models.ChangeRequest
	for rows.Next() {
		var cr models.ChangeRequest
		var proposedJSON []byte
		if err := rows.Scan(&cr.ID, &cr.TeamID, &cr.ConfigID, &cr.Title, &cr.Description, &cr.Status,
			&proposedJSON, &cr.BaseVersion, &cr.CreatedBy, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(proposedJSON, &cr.ProposedChanges)
		crs = append(crs, &cr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return crs, nil
}

func (r *collaborationRepository) UpdateChangeRequest(ctx context.Context, cr *models.ChangeRequest) error {
	query := `UPDATE change_requests SET title = $2, description = $3, status = $4, updated_at = $5 WHERE id = $1`
	cr.UpdatedAt = time.Now()
	_, err := r.db.Pool.Exec(ctx, query, cr.ID, cr.Title, cr.Description, cr.Status, cr.UpdatedAt)
	return err
}

// Review operations

func (r *collaborationRepository) AddReview(ctx context.Context, review *models.ChangeRequestReview) error {
	query := `INSERT INTO change_request_reviews (id, change_request_id, reviewer_id, status, comments, reviewed_at) VALUES ($1, $2, $3, $4, $5, $6)`

	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}
	review.ReviewedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, query, review.ID, review.ChangeRequestID, review.ReviewerID, review.Status, review.Comments, review.ReviewedAt)
	return err
}

func (r *collaborationRepository) GetReviews(ctx context.Context, changeRequestID uuid.UUID) ([]*models.ChangeRequestReview, error) {
	query := `SELECT id, change_request_id, reviewer_id, status, comments, reviewed_at FROM change_request_reviews WHERE change_request_id = $1 ORDER BY reviewed_at`

	rows, err := r.db.Pool.Query(ctx, query, changeRequestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*models.ChangeRequestReview
	for rows.Next() {
		var review models.ChangeRequestReview
		if err := rows.Scan(&review.ID, &review.ChangeRequestID, &review.ReviewerID, &review.Status, &review.Comments, &review.ReviewedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, &review)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reviews, nil
}

// Comment operations

func (r *collaborationRepository) AddComment(ctx context.Context, comment *models.ChangeRequestComment) error {
	query := `INSERT INTO change_request_comments (id, change_request_id, author_id, content, parent_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
	}
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	_, err := r.db.Pool.Exec(ctx, query, comment.ID, comment.ChangeRequestID, comment.AuthorID, comment.Content, comment.ParentID, comment.CreatedAt, comment.UpdatedAt)
	return err
}

func (r *collaborationRepository) GetComments(ctx context.Context, changeRequestID uuid.UUID) ([]*models.ChangeRequestComment, error) {
	query := `SELECT id, change_request_id, author_id, content, parent_id, created_at, updated_at FROM change_request_comments WHERE change_request_id = $1 ORDER BY created_at`

	rows, err := r.db.Pool.Query(ctx, query, changeRequestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*models.ChangeRequestComment
	for rows.Next() {
		var c models.ChangeRequestComment
		if err := rows.Scan(&c.ID, &c.ChangeRequestID, &c.AuthorID, &c.Content, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

// Activity feed operations

func (r *collaborationRepository) AddActivity(ctx context.Context, activity *models.ActivityFeedItem) error {
	query := `INSERT INTO activity_feed (id, team_id, actor_id, action_type, resource_type, resource_id, resource_name, details, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	if activity.ID == uuid.Nil {
		activity.ID = uuid.New()
	}
	activity.CreatedAt = time.Now()
	detailsJSON, _ := json.Marshal(activity.Details)

	_, err := r.db.Pool.Exec(ctx, query,
		activity.ID, activity.TeamID, activity.ActorID, activity.ActionType,
		activity.ResourceType, activity.ResourceID, activity.ResourceName, detailsJSON, activity.CreatedAt)
	return err
}

func (r *collaborationRepository) GetTeamActivity(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*models.ActivityFeedItem, error) {
	query := `SELECT id, team_id, actor_id, action_type, resource_type, resource_id, resource_name, details, created_at FROM activity_feed WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Pool.Query(ctx, query, teamID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*models.ActivityFeedItem
	for rows.Next() {
		var a models.ActivityFeedItem
		var detailsJSON []byte
		if err := rows.Scan(&a.ID, &a.TeamID, &a.ActorID, &a.ActionType, &a.ResourceType, &a.ResourceID, &a.ResourceName, &detailsJSON, &a.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(detailsJSON, &a.Details)
		activities = append(activities, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return activities, nil
}

// Notification operations

func (r *collaborationRepository) SaveNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error {
	query := `
		INSERT INTO notification_preferences (id, team_member_id, channel, event_types, is_enabled, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (team_member_id, channel) DO UPDATE SET event_types = $4, is_enabled = $5, settings = $6, updated_at = $8
	`

	if pref.ID == uuid.Nil {
		pref.ID = uuid.New()
	}
	now := time.Now()
	pref.CreatedAt = now
	pref.UpdatedAt = now

	eventTypesJSON, _ := json.Marshal(pref.EventTypes)
	settingsJSON, _ := json.Marshal(pref.Settings)

	_, err := r.db.Pool.Exec(ctx, query,
		pref.ID, pref.TeamMemberID, pref.Channel, eventTypesJSON, pref.IsEnabled, settingsJSON, pref.CreatedAt, pref.UpdatedAt)
	return err
}

func (r *collaborationRepository) GetNotificationPreferences(ctx context.Context, memberID uuid.UUID) ([]*models.NotificationPreference, error) {
	query := `SELECT id, team_member_id, channel, event_types, is_enabled, settings, created_at, updated_at FROM notification_preferences WHERE team_member_id = $1`

	rows, err := r.db.Pool.Query(ctx, query, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []*models.NotificationPreference
	for rows.Next() {
		var p models.NotificationPreference
		var eventTypesJSON, settingsJSON []byte
		if err := rows.Scan(&p.ID, &p.TeamMemberID, &p.Channel, &eventTypesJSON, &p.IsEnabled, &settingsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(eventTypesJSON, &p.EventTypes)
		json.Unmarshal(settingsJSON, &p.Settings)
		prefs = append(prefs, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prefs, nil
}

func (r *collaborationRepository) CreateNotificationIntegration(ctx context.Context, integration *models.NotificationIntegration) error {
	query := `
		INSERT INTO notification_integrations (id, team_id, integration_type, name, config, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	if integration.ID == uuid.Nil {
		integration.ID = uuid.New()
	}
	now := time.Now()
	integration.CreatedAt = now
	integration.UpdatedAt = now

	configJSON, _ := json.Marshal(integration.Config)

	_, err := r.db.Pool.Exec(ctx, query,
		integration.ID, integration.TeamID, integration.IntegrationType, integration.Name, configJSON, integration.IsActive, integration.CreatedAt, integration.UpdatedAt)
	return err
}

func (r *collaborationRepository) GetNotificationIntegrations(ctx context.Context, teamID uuid.UUID) ([]*models.NotificationIntegration, error) {
	query := `SELECT id, team_id, integration_type, name, config, is_active, created_at, updated_at FROM notification_integrations WHERE team_id = $1`

	rows, err := r.db.Pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var integrations []*models.NotificationIntegration
	for rows.Next() {
		var i models.NotificationIntegration
		var configJSON []byte
		if err := rows.Scan(&i.ID, &i.TeamID, &i.IntegrationType, &i.Name, &configJSON, &i.IsActive, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(configJSON, &i.Config)
		integrations = append(integrations, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return integrations, nil
}

func (r *collaborationRepository) RecordSentNotification(ctx context.Context, notification *models.SentNotification) error {
	query := `
		INSERT INTO sent_notifications (id, team_id, integration_id, recipient_id, channel, event_type, subject, content, status, sent_at, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	notification.CreatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, query,
		notification.ID, notification.TeamID, notification.IntegrationID, notification.RecipientID,
		notification.Channel, notification.EventType, notification.Subject, notification.Content,
		notification.Status, notification.SentAt, notification.ErrorMessage, notification.CreatedAt)
	return err
}
