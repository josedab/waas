package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeadLetterReason categorizes why an event was dead-lettered
type DeadLetterReason string

const (
	DeadLetterReasonMaxRetries    DeadLetterReason = "max_retries_exceeded"
	DeadLetterReasonTransformFail DeadLetterReason = "transform_failed"
	DeadLetterReasonSchemaInvalid DeadLetterReason = "schema_invalid"
	DeadLetterReasonPayloadSize   DeadLetterReason = "payload_too_large"
	DeadLetterReasonRoutingFail   DeadLetterReason = "routing_failed"
	DeadLetterReasonTimeout       DeadLetterReason = "timeout"
	DeadLetterReasonManual        DeadLetterReason = "manual"
)

// DeadLetterPolicy defines how failed events are handled per broker
type DeadLetterPolicy struct {
	ID             string           `json:"id"`
	TenantID       string           `json:"tenant_id"`
	BridgeID       string           `json:"bridge_id"`
	Enabled        bool             `json:"enabled"`
	MaxRetries     int              `json:"max_retries"`
	RetryBackoffMs int              `json:"retry_backoff_ms"`
	TargetType     string           `json:"target_type"` // topic, queue, webhook, storage
	TargetConfig   *DLQTargetConfig `json:"target_config"`
	RetentionDays  int              `json:"retention_days"`
	AlertOnDLQ     bool             `json:"alert_on_dlq"`
	AlertThreshold int              `json:"alert_threshold"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// DLQTargetConfig defines the dead-letter target destination
type DLQTargetConfig struct {
	TopicName  string `json:"topic_name,omitempty"`
	QueueURL   string `json:"queue_url,omitempty"`
	WebhookURL string `json:"webhook_url,omitempty"`
	BucketName string `json:"bucket_name,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
}

// DeadLetterEvent represents an event that has been dead-lettered
type DeadLetterEvent struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	BridgeID       string            `json:"bridge_id"`
	OriginalEvent  json.RawMessage   `json:"original_event"`
	Reason         DeadLetterReason  `json:"reason"`
	ErrorMessage   string            `json:"error_message"`
	RetryCount     int               `json:"retry_count"`
	Headers        map[string]string `json:"headers,omitempty"`
	Reprocessed    bool              `json:"reprocessed"`
	ReprocessedAt  *time.Time        `json:"reprocessed_at,omitempty"`
	ReprocessError string            `json:"reprocess_error,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	ExpiresAt      time.Time         `json:"expires_at"`
}

// CreateDeadLetterPolicyRequest represents a request to create a DLQ policy
type CreateDeadLetterPolicyRequest struct {
	BridgeID       string           `json:"bridge_id" binding:"required"`
	Enabled        bool             `json:"enabled"`
	MaxRetries     int              `json:"max_retries,omitempty"`
	RetryBackoffMs int              `json:"retry_backoff_ms,omitempty"`
	TargetType     string           `json:"target_type" binding:"required"`
	TargetConfig   *DLQTargetConfig `json:"target_config" binding:"required"`
	RetentionDays  int              `json:"retention_days,omitempty"`
	AlertOnDLQ     bool             `json:"alert_on_dlq,omitempty"`
	AlertThreshold int              `json:"alert_threshold,omitempty"`
}

// DeadLetterRouter manages dead-letter routing for streaming bridges
type DeadLetterRouter struct {
	repo     DeadLetterRepository
	policies map[string]*DeadLetterPolicy // bridgeID -> policy
	mu       sync.RWMutex
}

// DeadLetterRepository defines storage for dead-letter operations
type DeadLetterRepository interface {
	CreatePolicy(ctx context.Context, policy *DeadLetterPolicy) error
	GetPolicy(ctx context.Context, tenantID, bridgeID string) (*DeadLetterPolicy, error)
	UpdatePolicy(ctx context.Context, policy *DeadLetterPolicy) error
	DeletePolicy(ctx context.Context, tenantID, bridgeID string) error
	SaveDeadLetterEvent(ctx context.Context, event *DeadLetterEvent) error
	GetDeadLetterEvent(ctx context.Context, tenantID, eventID string) (*DeadLetterEvent, error)
	ListDeadLetterEvents(ctx context.Context, tenantID, bridgeID string, filters *DLQFilters) ([]DeadLetterEvent, int, error)
	MarkReprocessed(ctx context.Context, eventID string, errMsg string) error
	CountByReason(ctx context.Context, tenantID, bridgeID string) (map[DeadLetterReason]int64, error)
	PurgeExpired(ctx context.Context) (int64, error)
}

// DLQFilters contains filter options for listing dead-letter events
type DLQFilters struct {
	Reason      *DeadLetterReason `json:"reason,omitempty"`
	Reprocessed *bool             `json:"reprocessed,omitempty"`
	StartTime   *time.Time        `json:"start_time,omitempty"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
	Page        int               `json:"page,omitempty"`
	PageSize    int               `json:"page_size,omitempty"`
}

// NewDeadLetterRouter creates a new dead-letter router
func NewDeadLetterRouter(repo DeadLetterRepository) *DeadLetterRouter {
	return &DeadLetterRouter{
		repo:     repo,
		policies: make(map[string]*DeadLetterPolicy),
	}
}

// CreatePolicy creates a dead-letter policy for a bridge
func (r *DeadLetterRouter) CreatePolicy(ctx context.Context, tenantID string, req *CreateDeadLetterPolicyRequest) (*DeadLetterPolicy, error) {
	if err := r.validatePolicy(req); err != nil {
		return nil, err
	}

	maxRetries := req.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}
	retryBackoff := req.RetryBackoffMs
	if retryBackoff == 0 {
		retryBackoff = 1000
	}
	retention := req.RetentionDays
	if retention == 0 {
		retention = 14
	}

	now := time.Now()
	policy := &DeadLetterPolicy{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		BridgeID:       req.BridgeID,
		Enabled:        req.Enabled,
		MaxRetries:     maxRetries,
		RetryBackoffMs: retryBackoff,
		TargetType:     req.TargetType,
		TargetConfig:   req.TargetConfig,
		RetentionDays:  retention,
		AlertOnDLQ:     req.AlertOnDLQ,
		AlertThreshold: req.AlertThreshold,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := r.repo.CreatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create dead-letter policy: %w", err)
	}

	r.mu.Lock()
	r.policies[policy.BridgeID] = policy
	r.mu.Unlock()

	return policy, nil
}

// RouteToDeadLetter sends a failed event to the dead-letter queue
func (r *DeadLetterRouter) RouteToDeadLetter(ctx context.Context, tenantID, bridgeID string, originalEvent json.RawMessage, reason DeadLetterReason, errMsg string, retryCount int, headers map[string]string) (*DeadLetterEvent, error) {
	r.mu.RLock()
	policy, ok := r.policies[bridgeID]
	r.mu.RUnlock()

	if !ok {
		var err error
		policy, err = r.repo.GetPolicy(ctx, tenantID, bridgeID)
		if err != nil {
			return nil, fmt.Errorf("no dead-letter policy for bridge: %w", err)
		}
		r.mu.Lock()
		r.policies[bridgeID] = policy
		r.mu.Unlock()
	}

	if !policy.Enabled {
		return nil, nil
	}

	now := time.Now()
	dlEvent := &DeadLetterEvent{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		BridgeID:      bridgeID,
		OriginalEvent: originalEvent,
		Reason:        reason,
		ErrorMessage:  errMsg,
		RetryCount:    retryCount,
		Headers:       headers,
		CreatedAt:     now,
		ExpiresAt:     now.Add(time.Duration(policy.RetentionDays) * 24 * time.Hour),
	}

	if err := r.repo.SaveDeadLetterEvent(ctx, dlEvent); err != nil {
		return nil, fmt.Errorf("failed to save dead-letter event: %w", err)
	}

	return dlEvent, nil
}

// ReprocessEvent attempts to reprocess a dead-lettered event
func (r *DeadLetterRouter) ReprocessEvent(ctx context.Context, tenantID, eventID string) error {
	event, err := r.repo.GetDeadLetterEvent(ctx, tenantID, eventID)
	if err != nil {
		return fmt.Errorf("dead-letter event not found: %w", err)
	}

	if event.Reprocessed {
		return fmt.Errorf("event already reprocessed")
	}

	// Mark as reprocessed (the actual reprocessing is handled by the caller)
	return r.repo.MarkReprocessed(ctx, eventID, "")
}

// GetPolicy retrieves a dead-letter policy for a bridge
func (r *DeadLetterRouter) GetPolicy(ctx context.Context, tenantID, bridgeID string) (*DeadLetterPolicy, error) {
	return r.repo.GetPolicy(ctx, tenantID, bridgeID)
}

// ListDeadLetterEvents lists dead-letter events for a bridge
func (r *DeadLetterRouter) ListDeadLetterEvents(ctx context.Context, tenantID, bridgeID string, filters *DLQFilters) ([]DeadLetterEvent, int, error) {
	return r.repo.ListDeadLetterEvents(ctx, tenantID, bridgeID, filters)
}

// GetDeadLetterStats returns dead-letter statistics by reason
func (r *DeadLetterRouter) GetDeadLetterStats(ctx context.Context, tenantID, bridgeID string) (map[DeadLetterReason]int64, error) {
	return r.repo.CountByReason(ctx, tenantID, bridgeID)
}

func (r *DeadLetterRouter) validatePolicy(req *CreateDeadLetterPolicyRequest) error {
	validTargets := map[string]bool{
		"topic": true, "queue": true, "webhook": true, "storage": true,
	}
	if !validTargets[req.TargetType] {
		return fmt.Errorf("%w: invalid target type: %s", ErrInvalidConfig, req.TargetType)
	}
	if req.TargetConfig == nil {
		return fmt.Errorf("%w: target config is required", ErrInvalidConfig)
	}
	return nil
}
