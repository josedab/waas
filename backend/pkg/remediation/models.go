// Package remediation provides AI-powered auto-remediation for webhook failures
package remediation

import (
	"encoding/json"
	"time"
)

// ActionType represents the type of remediation action
type ActionType string

const (
	ActionTypeURLUpdate        ActionType = "url_update"
	ActionTypeHeaderUpdate     ActionType = "header_update"
	ActionTypeCredentialRotate ActionType = "credential_rotate"
	ActionTypeTimeoutAdjust    ActionType = "timeout_adjust"
	ActionTypeRetryConfig      ActionType = "retry_config"
	ActionTypePayloadFix       ActionType = "payload_fix"
	ActionTypeEndpointDisable  ActionType = "endpoint_disable"
	ActionTypeAlertOnly        ActionType = "alert_only"
	ActionTypeTransformFix     ActionType = "transform_fix"
	ActionTypeRateLimitAdjust  ActionType = "rate_limit_adjust"
)

// ActionStatus represents the status of a remediation action
type ActionStatus string

const (
	ActionStatusPending   ActionStatus = "pending"
	ActionStatusApproved  ActionStatus = "approved"
	ActionStatusRejected  ActionStatus = "rejected"
	ActionStatusExecuting ActionStatus = "executing"
	ActionStatusCompleted ActionStatus = "completed"
	ActionStatusFailed    ActionStatus = "failed"
	ActionStatusRolledBack ActionStatus = "rolled_back"
)

// ConfidenceLevel categorizes confidence scores
type ConfidenceLevel string

const (
	ConfidenceLow    ConfidenceLevel = "low"    // < 0.5 - requires approval
	ConfidenceMedium ConfidenceLevel = "medium" // 0.5-0.8 - suggest with approval
	ConfidenceHigh   ConfidenceLevel = "high"   // > 0.8 - auto-execute if enabled
)

// ApprovalPolicy determines how actions are approved
type ApprovalPolicy string

const (
	ApprovalPolicyManual ApprovalPolicy = "manual"     // All actions require approval
	ApprovalPolicyAuto   ApprovalPolicy = "auto"       // High confidence auto-executes
	ApprovalPolicyNotify ApprovalPolicy = "notify"     // Auto-execute with notification
)

// RemediationAction represents a single remediation action
type RemediationAction struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	EndpointID      string          `json:"endpoint_id"`
	DeliveryID      string          `json:"delivery_id,omitempty"`
	AnalysisID      string          `json:"analysis_id,omitempty"`
	ActionType      ActionType      `json:"action_type"`
	Status          ActionStatus    `json:"status"`
	Description     string          `json:"description"`
	Reason          string          `json:"reason"`
	ConfidenceScore float64         `json:"confidence_score"`
	ConfidenceLevel ConfidenceLevel `json:"confidence_level"`
	Parameters      ActionParams    `json:"parameters"`
	PreviousState   json.RawMessage `json:"previous_state,omitempty"`
	NewState        json.RawMessage `json:"new_state,omitempty"`
	ApprovedBy      string          `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time      `json:"approved_at,omitempty"`
	RejectedBy      string          `json:"rejected_by,omitempty"`
	RejectedAt      *time.Time      `json:"rejected_at,omitempty"`
	RejectionReason string          `json:"rejection_reason,omitempty"`
	ExecutedAt      *time.Time      `json:"executed_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	RollbackAvail   bool            `json:"rollback_available"`
	AutoApproved    bool            `json:"auto_approved"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ActionParams contains parameters for different action types
type ActionParams struct {
	// URL Update
	OldURL string `json:"old_url,omitempty"`
	NewURL string `json:"new_url,omitempty"`

	// Header Update
	HeaderName  string `json:"header_name,omitempty"`
	OldValue    string `json:"old_value,omitempty"`
	NewValue    string `json:"new_value,omitempty"`
	AddHeader   bool   `json:"add_header,omitempty"`
	RemoveHeader bool  `json:"remove_header,omitempty"`

	// Credential Rotation
	CredentialType string `json:"credential_type,omitempty"`
	CredentialID   string `json:"credential_id,omitempty"`

	// Timeout Adjustment
	OldTimeoutMs int `json:"old_timeout_ms,omitempty"`
	NewTimeoutMs int `json:"new_timeout_ms,omitempty"`

	// Retry Configuration
	OldMaxRetries int `json:"old_max_retries,omitempty"`
	NewMaxRetries int `json:"new_max_retries,omitempty"`
	OldBackoffMs  int `json:"old_backoff_ms,omitempty"`
	NewBackoffMs  int `json:"new_backoff_ms,omitempty"`

	// Payload Fix
	TransformScript string `json:"transform_script,omitempty"`
	FieldPath       string `json:"field_path,omitempty"`
	FieldValue      string `json:"field_value,omitempty"`

	// Rate Limit
	OldRateLimit int `json:"old_rate_limit,omitempty"`
	NewRateLimit int `json:"new_rate_limit,omitempty"`
}

// RemediationPolicy defines how remediation is handled for a tenant
type RemediationPolicy struct {
	ID                    string         `json:"id"`
	TenantID              string         `json:"tenant_id"`
	Enabled               bool           `json:"enabled"`
	ApprovalPolicy        ApprovalPolicy `json:"approval_policy"`
	AutoApproveThreshold  float64        `json:"auto_approve_threshold"` // Confidence threshold for auto-approval
	MaxAutoActionsPerHour int            `json:"max_auto_actions_per_hour"`
	RequireApprovalFor    []ActionType   `json:"require_approval_for"` // Actions that always need approval
	NotifyOnAction        bool           `json:"notify_on_action"`
	NotifyChannels        []string       `json:"notify_channels"` // email, slack, webhook
	NotifyRecipients      []string       `json:"notify_recipients"`
	ActionExpirySec       int            `json:"action_expiry_sec"` // Pending actions expire after this
	CooldownPeriodSec     int            `json:"cooldown_period_sec"` // Between auto-actions on same endpoint
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

// ApprovalRequest represents a request to approve a remediation
type ApprovalRequest struct {
	ActionID string `json:"action_id" binding:"required"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

// RemediationSuggestion represents a suggested remediation
type RemediationSuggestion struct {
	ActionType      ActionType   `json:"action_type"`
	Description     string       `json:"description"`
	Reason          string       `json:"reason"`
	ConfidenceScore float64      `json:"confidence_score"`
	Parameters      ActionParams `json:"parameters"`
	Impact          string       `json:"impact"`          // Description of change impact
	Risk            string       `json:"risk"`            // Low, Medium, High
	Reversible      bool         `json:"reversible"`
}

// AnalysisWithRemediation extends AI analysis with remediation suggestions
type AnalysisWithRemediation struct {
	AnalysisID     string                  `json:"analysis_id"`
	DeliveryID     string                  `json:"delivery_id"`
	EndpointID     string                  `json:"endpoint_id"`
	Classification string                  `json:"classification"`
	RootCause      string                  `json:"root_cause"`
	Suggestions    []RemediationSuggestion `json:"suggestions"`
	AutoRemediable bool                    `json:"auto_remediable"`
	CreatedAt      time.Time               `json:"created_at"`
}

// RemediationMetrics tracks remediation effectiveness
type RemediationMetrics struct {
	TenantID           string    `json:"tenant_id"`
	Period             string    `json:"period"` // daily, weekly, monthly
	TotalActions       int       `json:"total_actions"`
	AutoApproved       int       `json:"auto_approved"`
	ManuallyApproved   int       `json:"manually_approved"`
	Rejected           int       `json:"rejected"`
	Successful         int       `json:"successful"`
	Failed             int       `json:"failed"`
	RolledBack         int       `json:"rolled_back"`
	AvgTimeToRemediate float64   `json:"avg_time_to_remediate_sec"`
	SuccessRate        float64   `json:"success_rate"`
	ByActionType       map[ActionType]int `json:"by_action_type"`
	IssuesResolved     int       `json:"issues_resolved"`
	CollectedAt        time.Time `json:"collected_at"`
}

// ActionAuditLog records audit trail for actions
type ActionAuditLog struct {
	ID         string          `json:"id"`
	ActionID   string          `json:"action_id"`
	TenantID   string          `json:"tenant_id"`
	Event      string          `json:"event"` // created, approved, rejected, executed, failed, rolled_back
	Actor      string          `json:"actor"` // user_id or "system"
	Details    json.RawMessage `json:"details,omitempty"`
	IPAddress  string          `json:"ip_address,omitempty"`
	UserAgent  string          `json:"user_agent,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// CreateActionRequest represents a request to create a remediation action
type CreateActionRequest struct {
	EndpointID      string       `json:"endpoint_id" binding:"required"`
	DeliveryID      string       `json:"delivery_id,omitempty"`
	ActionType      ActionType   `json:"action_type" binding:"required"`
	Description     string       `json:"description"`
	Parameters      ActionParams `json:"parameters"`
	ConfidenceScore float64      `json:"confidence_score,omitempty"`
}

// ListActionsResponse represents paginated action list
type ListActionsResponse struct {
	Actions    []RemediationAction `json:"actions"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// ActionFilters for listing actions
type ActionFilters struct {
	EndpointID string        `json:"endpoint_id,omitempty"`
	ActionType *ActionType   `json:"action_type,omitempty"`
	Status     *ActionStatus `json:"status,omitempty"`
	StartTime  *time.Time    `json:"start_time,omitempty"`
	EndTime    *time.Time    `json:"end_time,omitempty"`
	Page       int           `json:"page,omitempty"`
	PageSize   int           `json:"page_size,omitempty"`
}

// GetConfidenceLevel returns the confidence level for a score
func GetConfidenceLevel(score float64) ConfidenceLevel {
	if score >= 0.8 {
		return ConfidenceHigh
	}
	if score >= 0.5 {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

// DefaultRemediationPolicy returns a default policy
func DefaultRemediationPolicy(tenantID string) *RemediationPolicy {
	return &RemediationPolicy{
		TenantID:              tenantID,
		Enabled:               true,
		ApprovalPolicy:        ApprovalPolicyManual,
		AutoApproveThreshold:  0.85,
		MaxAutoActionsPerHour: 10,
		RequireApprovalFor: []ActionType{
			ActionTypeEndpointDisable,
			ActionTypeCredentialRotate,
		},
		NotifyOnAction:    true,
		ActionExpirySec:   86400, // 24 hours
		CooldownPeriodSec: 300,   // 5 minutes
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}
