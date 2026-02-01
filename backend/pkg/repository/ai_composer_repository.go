package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/models"
)

type AIComposerRepository interface {
	// Session operations
	CreateSession(ctx context.Context, session *models.AIComposerSession) error
	GetSession(ctx context.Context, id uuid.UUID) (*models.AIComposerSession, error)
	GetSessionsByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.AIComposerSession, error)
	UpdateSession(ctx context.Context, session *models.AIComposerSession) error
	DeleteSession(ctx context.Context, id uuid.UUID) error
	CleanupExpiredSessions(ctx context.Context) (int64, error)

	// Message operations
	AddMessage(ctx context.Context, message *models.AIComposerMessage) error
	GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerMessage, error)

	// Generated config operations
	SaveGeneratedConfig(ctx context.Context, config *models.AIComposerGeneratedConfig) error
	GetGeneratedConfig(ctx context.Context, id uuid.UUID) (*models.AIComposerGeneratedConfig, error)
	GetSessionConfigs(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerGeneratedConfig, error)
	UpdateConfigValidation(ctx context.Context, id uuid.UUID, status string, errors []string) error
	MarkConfigApplied(ctx context.Context, id uuid.UUID) error

	// Template operations
	GetTemplates(ctx context.Context, category string) ([]*models.AIComposerTemplate, error)
	GetTemplate(ctx context.Context, id uuid.UUID) (*models.AIComposerTemplate, error)
	IncrementTemplateUsage(ctx context.Context, id uuid.UUID) error

	// Feedback operations
	SaveFeedback(ctx context.Context, feedback *models.AIComposerFeedback) error
	GetSessionFeedback(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerFeedback, error)
}

type aiComposerRepository struct {
	db *database.DB
}

func NewAIComposerRepository(db *database.DB) AIComposerRepository {
	return &aiComposerRepository{db: db}
}

func (r *aiComposerRepository) CreateSession(ctx context.Context, session *models.AIComposerSession) error {
	query := `
		INSERT INTO ai_composer_sessions (id, tenant_id, status, context, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = now.Add(24 * time.Hour)
	}
	if session.Context == nil {
		session.Context = make(map[string]interface{})
	}

	contextJSON, err := json.Marshal(session.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx, query,
		session.ID, session.TenantID, session.Status, contextJSON,
		session.CreatedAt, session.UpdatedAt, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create AI composer session: %w", err)
	}
	return nil
}

func (r *aiComposerRepository) GetSession(ctx context.Context, id uuid.UUID) (*models.AIComposerSession, error) {
	query := `
		SELECT id, tenant_id, status, context, created_at, updated_at, expires_at
		FROM ai_composer_sessions WHERE id = $1
	`

	var session models.AIComposerSession
	var contextJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&session.ID, &session.TenantID, &session.Status, &contextJSON,
		&session.CreatedAt, &session.UpdatedAt, &session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if err := json.Unmarshal(contextJSON, &session.Context); err != nil {
		session.Context = make(map[string]interface{})
	}

	return &session, nil
}

func (r *aiComposerRepository) GetSessionsByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.AIComposerSession, error) {
	query := `
		SELECT id, tenant_id, status, context, created_at, updated_at, expires_at
		FROM ai_composer_sessions 
		WHERE tenant_id = $1 AND status != 'expired'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.AIComposerSession
	for rows.Next() {
		var session models.AIComposerSession
		var contextJSON []byte

		if err := rows.Scan(&session.ID, &session.TenantID, &session.Status, &contextJSON,
			&session.CreatedAt, &session.UpdatedAt, &session.ExpiresAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if err := json.Unmarshal(contextJSON, &session.Context); err != nil {
			session.Context = make(map[string]interface{})
		}
		sessions = append(sessions, &session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate sessions: %w", err)
	}

	return sessions, nil
}

func (r *aiComposerRepository) UpdateSession(ctx context.Context, session *models.AIComposerSession) error {
	query := `
		UPDATE ai_composer_sessions 
		SET status = $2, context = $3, updated_at = $4
		WHERE id = $1
	`

	session.UpdatedAt = time.Now()
	contextJSON, err := json.Marshal(session.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx, query, session.ID, session.Status, contextJSON, session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

func (r *aiComposerRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_composer_sessions WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *aiComposerRepository) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	query := `DELETE FROM ai_composer_sessions WHERE expires_at < NOW()`
	result, err := r.db.Pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return result.RowsAffected(), nil
}

func (r *aiComposerRepository) AddMessage(ctx context.Context, message *models.AIComposerMessage) error {
	query := `
		INSERT INTO ai_composer_messages (id, session_id, role, content, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}
	message.CreatedAt = time.Now()
	if message.Metadata == nil {
		message.Metadata = make(map[string]interface{})
	}

	metadataJSON, err := json.Marshal(message.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx, query,
		message.ID, message.SessionID, message.Role, message.Content, metadataJSON, message.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}
	return nil
}

func (r *aiComposerRepository) GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerMessage, error) {
	query := `
		SELECT id, session_id, role, content, metadata, created_at
		FROM ai_composer_messages 
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.AIComposerMessage
	for rows.Next() {
		var msg models.AIComposerMessage
		var metadataJSON []byte

		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &metadataJSON, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
			msg.Metadata = make(map[string]interface{})
		}
		messages = append(messages, &msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate messages: %w", err)
	}

	return messages, nil
}

func (r *aiComposerRepository) SaveGeneratedConfig(ctx context.Context, config *models.AIComposerGeneratedConfig) error {
	query := `
		INSERT INTO ai_composer_generated_configs 
		(id, session_id, tenant_id, config_type, generated_config, transformation_code, validation_status, validation_errors, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	config.CreatedAt = time.Now()
	if config.ValidationStatus == "" {
		config.ValidationStatus = models.AIValidationPending
	}

	configJSON, err := json.Marshal(config.GeneratedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	errorsJSON, err := json.Marshal(config.ValidationErrors)
	if err != nil {
		return fmt.Errorf("failed to marshal errors: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx, query,
		config.ID, config.SessionID, config.TenantID, config.ConfigType,
		configJSON, config.TransformationCode, config.ValidationStatus, errorsJSON, config.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save generated config: %w", err)
	}
	return nil
}

func (r *aiComposerRepository) GetGeneratedConfig(ctx context.Context, id uuid.UUID) (*models.AIComposerGeneratedConfig, error) {
	query := `
		SELECT id, session_id, tenant_id, config_type, generated_config, transformation_code, 
		       validation_status, validation_errors, applied, applied_at, created_at
		FROM ai_composer_generated_configs WHERE id = $1
	`

	var config models.AIComposerGeneratedConfig
	var configJSON, errorsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.SessionID, &config.TenantID, &config.ConfigType,
		&configJSON, &config.TransformationCode, &config.ValidationStatus,
		&errorsJSON, &config.Applied, &config.AppliedAt, &config.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	json.Unmarshal(configJSON, &config.GeneratedConfig)
	json.Unmarshal(errorsJSON, &config.ValidationErrors)

	return &config, nil
}

func (r *aiComposerRepository) GetSessionConfigs(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerGeneratedConfig, error) {
	query := `
		SELECT id, session_id, tenant_id, config_type, generated_config, transformation_code,
		       validation_status, validation_errors, applied, applied_at, created_at
		FROM ai_composer_generated_configs 
		WHERE session_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get configs: %w", err)
	}
	defer rows.Close()

	var configs []*models.AIComposerGeneratedConfig
	for rows.Next() {
		var config models.AIComposerGeneratedConfig
		var configJSON, errorsJSON []byte

		if err := rows.Scan(&config.ID, &config.SessionID, &config.TenantID, &config.ConfigType,
			&configJSON, &config.TransformationCode, &config.ValidationStatus,
			&errorsJSON, &config.Applied, &config.AppliedAt, &config.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}

		json.Unmarshal(configJSON, &config.GeneratedConfig)
		json.Unmarshal(errorsJSON, &config.ValidationErrors)
		configs = append(configs, &config)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate configs: %w", err)
	}

	return configs, nil
}

func (r *aiComposerRepository) UpdateConfigValidation(ctx context.Context, id uuid.UUID, status string, errors []string) error {
	query := `
		UPDATE ai_composer_generated_configs 
		SET validation_status = $2, validation_errors = $3
		WHERE id = $1
	`

	errorsJSON, err := json.Marshal(errors)
	if err != nil {
		return fmt.Errorf("failed to marshal errors: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx, query, id, status, errorsJSON)
	if err != nil {
		return fmt.Errorf("failed to update validation: %w", err)
	}
	return nil
}

func (r *aiComposerRepository) MarkConfigApplied(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE ai_composer_generated_configs SET applied = true, applied_at = NOW() WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark config applied: %w", err)
	}
	return nil
}

func (r *aiComposerRepository) GetTemplates(ctx context.Context, category string) ([]*models.AIComposerTemplate, error) {
	query := `
		SELECT id, name, description, category, prompt_template, example_input, example_output, 
		       is_active, usage_count, created_at, updated_at
		FROM ai_composer_templates 
		WHERE is_active = true
	`
	args := []interface{}{}

	if category != "" {
		query += " AND category = $1"
		args = append(args, category)
	}
	query += " ORDER BY usage_count DESC"

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}
	defer rows.Close()

	var templates []*models.AIComposerTemplate
	for rows.Next() {
		var t models.AIComposerTemplate
		var exampleOutputJSON []byte

		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.PromptTemplate,
			&t.ExampleInput, &exampleOutputJSON, &t.IsActive, &t.UsageCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}

		json.Unmarshal(exampleOutputJSON, &t.ExampleOutput)
		templates = append(templates, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate templates: %w", err)
	}

	return templates, nil
}

func (r *aiComposerRepository) GetTemplate(ctx context.Context, id uuid.UUID) (*models.AIComposerTemplate, error) {
	query := `
		SELECT id, name, description, category, prompt_template, example_input, example_output,
		       is_active, usage_count, created_at, updated_at
		FROM ai_composer_templates WHERE id = $1
	`

	var t models.AIComposerTemplate
	var exampleOutputJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.Category, &t.PromptTemplate,
		&t.ExampleInput, &exampleOutputJSON, &t.IsActive, &t.UsageCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	json.Unmarshal(exampleOutputJSON, &t.ExampleOutput)
	return &t, nil
}

func (r *aiComposerRepository) IncrementTemplateUsage(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE ai_composer_templates SET usage_count = usage_count + 1 WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

func (r *aiComposerRepository) SaveFeedback(ctx context.Context, feedback *models.AIComposerFeedback) error {
	query := `
		INSERT INTO ai_composer_feedback (id, session_id, config_id, rating, feedback_text, worked_as_expected, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if feedback.ID == uuid.Nil {
		feedback.ID = uuid.New()
	}
	feedback.CreatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, query,
		feedback.ID, feedback.SessionID, feedback.ConfigID, feedback.Rating,
		feedback.FeedbackText, feedback.WorkedAsExpected, feedback.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save feedback: %w", err)
	}
	return nil
}

func (r *aiComposerRepository) GetSessionFeedback(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerFeedback, error) {
	query := `
		SELECT id, session_id, config_id, rating, feedback_text, worked_as_expected, created_at
		FROM ai_composer_feedback WHERE session_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}
	defer rows.Close()

	var feedbacks []*models.AIComposerFeedback
	for rows.Next() {
		var f models.AIComposerFeedback
		if err := rows.Scan(&f.ID, &f.SessionID, &f.ConfigID, &f.Rating,
			&f.FeedbackText, &f.WorkedAsExpected, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan feedback: %w", err)
		}
		feedbacks = append(feedbacks, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate feedback: %w", err)
	}

	return feedbacks, nil
}
