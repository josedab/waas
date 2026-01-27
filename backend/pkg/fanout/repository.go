package fanout

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the data access interface for fan-out resources
type Repository interface {
	// Topics
	CreateTopic(ctx context.Context, topic *Topic) error
	GetTopic(ctx context.Context, tenantID, topicID uuid.UUID) (*Topic, error)
	GetTopicByName(ctx context.Context, tenantID uuid.UUID, name string) (*Topic, error)
	ListTopics(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Topic, int, error)
	UpdateTopic(ctx context.Context, topic *Topic) error
	DeleteTopic(ctx context.Context, tenantID, topicID uuid.UUID) error

	// Subscriptions
	CreateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscription(ctx context.Context, subID uuid.UUID) (*Subscription, error)
	ListSubscriptions(ctx context.Context, topicID uuid.UUID, limit, offset int) ([]Subscription, int, error)
	GetActiveSubscriptions(ctx context.Context, topicID uuid.UUID) ([]Subscription, error)
	DeleteSubscription(ctx context.Context, subID uuid.UUID) error
	CountSubscriptions(ctx context.Context, topicID uuid.UUID) (int, error)

	// Events
	CreateEvent(ctx context.Context, event *TopicEvent) error
	GetEvent(ctx context.Context, eventID uuid.UUID) (*TopicEvent, error)
	ListEvents(ctx context.Context, topicID uuid.UUID, limit, offset int) ([]TopicEvent, int, error)
	UpdateEventStatus(ctx context.Context, eventID uuid.UUID, status string, fanOutCount int) error

	// Routing Rules
	CreateRoutingRule(ctx context.Context, rule *RoutingRule) error
	GetRoutingRule(ctx context.Context, tenantID, ruleID uuid.UUID) (*RoutingRule, error)
	ListRoutingRules(ctx context.Context, tenantID, topicID uuid.UUID) ([]RoutingRule, error)
	UpdateRoutingRule(ctx context.Context, rule *RoutingRule) error
	DeleteRoutingRule(ctx context.Context, tenantID, ruleID uuid.UUID) error

	// Rule Versions
	CreateRuleVersion(ctx context.Context, version *RuleVersion) error
	GetRuleVersions(ctx context.Context, ruleID uuid.UUID) ([]RuleVersion, error)
	GetRuleVersion(ctx context.Context, ruleID uuid.UUID, version int) (*RuleVersion, error)
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateTopic(ctx context.Context, topic *Topic) error {
	query := `INSERT INTO fanout_topics (id, tenant_id, name, description, status, max_subscribers, retention_days, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query,
		topic.ID, topic.TenantID, topic.Name, topic.Description, topic.Status,
		topic.MaxSubscribers, topic.RetentionDays, topic.CreatedAt, topic.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetTopic(ctx context.Context, tenantID, topicID uuid.UUID) (*Topic, error) {
	var topic Topic
	err := r.db.GetContext(ctx, &topic,
		`SELECT * FROM fanout_topics WHERE id = $1 AND tenant_id = $2`, topicID, tenantID)
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *PostgresRepository) GetTopicByName(ctx context.Context, tenantID uuid.UUID, name string) (*Topic, error) {
	var topic Topic
	err := r.db.GetContext(ctx, &topic,
		`SELECT * FROM fanout_topics WHERE tenant_id = $1 AND name = $2`, tenantID, name)
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

func (r *PostgresRepository) ListTopics(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Topic, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM fanout_topics WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var topics []Topic
	err = r.db.SelectContext(ctx, &topics,
		`SELECT * FROM fanout_topics WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return topics, total, nil
}

func (r *PostgresRepository) UpdateTopic(ctx context.Context, topic *Topic) error {
	query := `UPDATE fanout_topics SET name = $1, description = $2, status = $3, max_subscribers = $4, retention_days = $5, updated_at = $6
		WHERE id = $7 AND tenant_id = $8`
	_, err := r.db.ExecContext(ctx, query,
		topic.Name, topic.Description, topic.Status, topic.MaxSubscribers, topic.RetentionDays, topic.UpdatedAt,
		topic.ID, topic.TenantID)
	return err
}

func (r *PostgresRepository) DeleteTopic(ctx context.Context, tenantID, topicID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM fanout_topics WHERE id = $1 AND tenant_id = $2`, topicID, tenantID)
	return err
}

func (r *PostgresRepository) CreateSubscription(ctx context.Context, sub *Subscription) error {
	query := `INSERT INTO fanout_subscriptions (id, topic_id, tenant_id, endpoint_id, filter_expression, active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID, sub.TopicID, sub.TenantID, sub.EndpointID, sub.FilterExpr, sub.Active, sub.CreatedAt)
	return err
}

func (r *PostgresRepository) GetSubscription(ctx context.Context, subID uuid.UUID) (*Subscription, error) {
	var sub Subscription
	err := r.db.GetContext(ctx, &sub,
		`SELECT * FROM fanout_subscriptions WHERE id = $1`, subID)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *PostgresRepository) ListSubscriptions(ctx context.Context, topicID uuid.UUID, limit, offset int) ([]Subscription, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM fanout_subscriptions WHERE topic_id = $1`, topicID)
	if err != nil {
		return nil, 0, err
	}

	var subs []Subscription
	err = r.db.SelectContext(ctx, &subs,
		`SELECT * FROM fanout_subscriptions WHERE topic_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		topicID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return subs, total, nil
}

func (r *PostgresRepository) GetActiveSubscriptions(ctx context.Context, topicID uuid.UUID) ([]Subscription, error) {
	var subs []Subscription
	err := r.db.SelectContext(ctx, &subs,
		`SELECT * FROM fanout_subscriptions WHERE topic_id = $1 AND active = true`, topicID)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (r *PostgresRepository) DeleteSubscription(ctx context.Context, subID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM fanout_subscriptions WHERE id = $1`, subID)
	return err
}

func (r *PostgresRepository) CountSubscriptions(ctx context.Context, topicID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM fanout_subscriptions WHERE topic_id = $1 AND active = true`, topicID)
	return count, err
}

func (r *PostgresRepository) CreateEvent(ctx context.Context, event *TopicEvent) error {
	query := `INSERT INTO fanout_events (id, topic_id, tenant_id, event_type, payload, metadata, fan_out_count, status, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.TopicID, event.TenantID, event.EventType, event.Payload, event.Metadata,
		event.FanOutCount, event.Status, event.PublishedAt)
	return err
}

func (r *PostgresRepository) GetEvent(ctx context.Context, eventID uuid.UUID) (*TopicEvent, error) {
	var event TopicEvent
	err := r.db.GetContext(ctx, &event,
		`SELECT * FROM fanout_events WHERE id = $1`, eventID)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *PostgresRepository) ListEvents(ctx context.Context, topicID uuid.UUID, limit, offset int) ([]TopicEvent, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM fanout_events WHERE topic_id = $1`, topicID)
	if err != nil {
		return nil, 0, err
	}

	var events []TopicEvent
	err = r.db.SelectContext(ctx, &events,
		`SELECT * FROM fanout_events WHERE topic_id = $1 ORDER BY published_at DESC LIMIT $2 OFFSET $3`,
		topicID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *PostgresRepository) UpdateEventStatus(ctx context.Context, eventID uuid.UUID, status string, fanOutCount int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE fanout_events SET status = $1, fan_out_count = $2 WHERE id = $3`,
		status, fanOutCount, eventID)
	return err
}

func (r *PostgresRepository) CreateRoutingRule(ctx context.Context, rule *RoutingRule) error {
	condJSON, _ := json.Marshal(rule.Conditions)
	actJSON, _ := json.Marshal(rule.Actions)
	query := `INSERT INTO fanout_routing_rules (id, tenant_id, topic_id, name, description, version, conditions, actions, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query,
		rule.ID, rule.TenantID, rule.TopicID, rule.Name, rule.Description,
		rule.Version, condJSON, actJSON, rule.Priority, rule.Enabled,
		rule.CreatedAt, rule.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetRoutingRule(ctx context.Context, tenantID, ruleID uuid.UUID) (*RoutingRule, error) {
	var rule RoutingRule
	var condJSON, actJSON []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, topic_id, name, description, version, conditions, actions, priority, enabled, created_at, updated_at
		FROM fanout_routing_rules WHERE id = $1 AND tenant_id = $2`, ruleID, tenantID).
		Scan(&rule.ID, &rule.TenantID, &rule.TopicID, &rule.Name, &rule.Description,
			&rule.Version, &condJSON, &actJSON, &rule.Priority, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(condJSON, &rule.Conditions)
	json.Unmarshal(actJSON, &rule.Actions)
	return &rule, nil
}

func (r *PostgresRepository) ListRoutingRules(ctx context.Context, tenantID, topicID uuid.UUID) ([]RoutingRule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, topic_id, name, description, version, conditions, actions, priority, enabled, created_at, updated_at
		FROM fanout_routing_rules WHERE tenant_id = $1 AND topic_id = $2 ORDER BY priority ASC`, tenantID, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []RoutingRule
	for rows.Next() {
		var rule RoutingRule
		var condJSON, actJSON []byte
		if err := rows.Scan(&rule.ID, &rule.TenantID, &rule.TopicID, &rule.Name, &rule.Description,
			&rule.Version, &condJSON, &actJSON, &rule.Priority, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(condJSON, &rule.Conditions)
		json.Unmarshal(actJSON, &rule.Actions)
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *PostgresRepository) UpdateRoutingRule(ctx context.Context, rule *RoutingRule) error {
	condJSON, _ := json.Marshal(rule.Conditions)
	actJSON, _ := json.Marshal(rule.Actions)
	query := `UPDATE fanout_routing_rules SET name = $1, description = $2, version = $3, conditions = $4, actions = $5, priority = $6, enabled = $7, updated_at = $8
		WHERE id = $9 AND tenant_id = $10`
	_, err := r.db.ExecContext(ctx, query,
		rule.Name, rule.Description, rule.Version, condJSON, actJSON,
		rule.Priority, rule.Enabled, rule.UpdatedAt, rule.ID, rule.TenantID)
	return err
}

func (r *PostgresRepository) DeleteRoutingRule(ctx context.Context, tenantID, ruleID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM fanout_routing_rules WHERE id = $1 AND tenant_id = $2`, ruleID, tenantID)
	return err
}

func (r *PostgresRepository) CreateRuleVersion(ctx context.Context, v *RuleVersion) error {
	condJSON, _ := json.Marshal(v.Conditions)
	actJSON, _ := json.Marshal(v.Actions)
	query := `INSERT INTO fanout_rule_versions (id, rule_id, version, conditions, actions, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		v.ID, v.RuleID, v.Version, condJSON, actJSON, v.CreatedAt, v.CreatedBy)
	return err
}

func (r *PostgresRepository) GetRuleVersions(ctx context.Context, ruleID uuid.UUID) ([]RuleVersion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, rule_id, version, conditions, actions, created_at, created_by
		FROM fanout_rule_versions WHERE rule_id = $1 ORDER BY version DESC`, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []RuleVersion
	for rows.Next() {
		var v RuleVersion
		var condJSON, actJSON []byte
		if err := rows.Scan(&v.ID, &v.RuleID, &v.Version, &condJSON, &actJSON, &v.CreatedAt, &v.CreatedBy); err != nil {
			return nil, err
		}
		json.Unmarshal(condJSON, &v.Conditions)
		json.Unmarshal(actJSON, &v.Actions)
		versions = append(versions, v)
	}
	return versions, nil
}

func (r *PostgresRepository) GetRuleVersion(ctx context.Context, ruleID uuid.UUID, version int) (*RuleVersion, error) {
	var v RuleVersion
	var condJSON, actJSON []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT id, rule_id, version, conditions, actions, created_at, created_by
		FROM fanout_rule_versions WHERE rule_id = $1 AND version = $2`, ruleID, version).
		Scan(&v.ID, &v.RuleID, &v.Version, &condJSON, &actJSON, &v.CreatedAt, &v.CreatedBy)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(condJSON, &v.Conditions)
	json.Unmarshal(actJSON, &v.Actions)
	return &v, nil
}
