package remediation

import (
	"context"
	"errors"
	"time"
)

var (
	ErrActionNotFound       = errors.New("remediation action not found")
	ErrPolicyNotFound       = errors.New("remediation policy not found")
	ErrActionAlreadyApproved = errors.New("action already approved")
	ErrActionExpired        = errors.New("action has expired")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrCooldownActive       = errors.New("cooldown period active for endpoint")
	ErrMaxActionsExceeded   = errors.New("maximum auto actions exceeded")
)

// Repository defines the interface for remediation storage
type Repository interface {
	// Actions
	CreateAction(ctx context.Context, action *RemediationAction) error
	GetAction(ctx context.Context, tenantID, actionID string) (*RemediationAction, error)
	UpdateAction(ctx context.Context, action *RemediationAction) error
	DeleteAction(ctx context.Context, tenantID, actionID string) error
	ListActions(ctx context.Context, tenantID string, filters *ActionFilters) ([]RemediationAction, int, error)
	GetPendingActions(ctx context.Context, tenantID string) ([]RemediationAction, error)
	GetExpiredActions(ctx context.Context) ([]RemediationAction, error)

	// Policies
	CreatePolicy(ctx context.Context, policy *RemediationPolicy) error
	GetPolicy(ctx context.Context, tenantID string) (*RemediationPolicy, error)
	UpdatePolicy(ctx context.Context, policy *RemediationPolicy) error
	DeletePolicy(ctx context.Context, tenantID string) error

	// Audit
	CreateAuditLog(ctx context.Context, log *ActionAuditLog) error
	GetAuditLogs(ctx context.Context, actionID string) ([]ActionAuditLog, error)

	// Metrics
	SaveMetrics(ctx context.Context, metrics *RemediationMetrics) error
	GetMetrics(ctx context.Context, tenantID, period string) (*RemediationMetrics, error)

	// Cooldown tracking
	SetCooldown(ctx context.Context, tenantID, endpointID string, duration time.Duration) error
	HasCooldown(ctx context.Context, tenantID, endpointID string) (bool, error)

	// Rate limiting for auto-actions
	IncrementAutoActionCount(ctx context.Context, tenantID string) (int, error)
	GetAutoActionCount(ctx context.Context, tenantID string, since time.Time) (int, error)
}

// EndpointUpdater defines interface for updating webhook endpoints
type EndpointUpdater interface {
	UpdateURL(ctx context.Context, tenantID, endpointID, newURL string) error
	UpdateHeaders(ctx context.Context, tenantID, endpointID string, headers map[string]string) error
	UpdateTimeout(ctx context.Context, tenantID, endpointID string, timeoutMs int) error
	UpdateRetryConfig(ctx context.Context, tenantID, endpointID string, maxRetries, backoffMs int) error
	DisableEndpoint(ctx context.Context, tenantID, endpointID string) error
	GetEndpointState(ctx context.Context, tenantID, endpointID string) (map[string]interface{}, error)
}

// AIAnalyzer defines interface for AI analysis integration
type AIAnalyzer interface {
	AnalyzeFailure(ctx context.Context, tenantID, deliveryID string) (*AnalysisWithRemediation, error)
	GenerateSuggestions(ctx context.Context, tenantID, endpointID string, errorPattern string) ([]RemediationSuggestion, error)
}

// Notifier defines interface for sending notifications
type Notifier interface {
	SendActionCreated(ctx context.Context, action *RemediationAction) error
	SendActionApproved(ctx context.Context, action *RemediationAction) error
	SendActionExecuted(ctx context.Context, action *RemediationAction, success bool) error
	SendActionPending(ctx context.Context, action *RemediationAction) error
}

// ActionExecutor defines interface for executing remediation actions
type ActionExecutor interface {
	Execute(ctx context.Context, action *RemediationAction) error
	Rollback(ctx context.Context, action *RemediationAction) error
	Validate(ctx context.Context, action *RemediationAction) error
}
