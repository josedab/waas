package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// Service provides remediation operations
type Service struct {
	repo            Repository
	endpointUpdater EndpointUpdater
	aiAnalyzer      AIAnalyzer
	notifier        Notifier
	executor        *DefaultExecutor
	config          *ServiceConfig
	logger          *utils.Logger
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	EnableAutoRemediation bool
	DefaultExpirySec      int
	MaxPendingActions     int
	CleanupIntervalSec    int
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		EnableAutoRemediation: true,
		DefaultExpirySec:      86400, // 24 hours
		MaxPendingActions:     100,
		CleanupIntervalSec:    3600, // 1 hour
	}
}

// NewService creates a new remediation service
func NewService(repo Repository, endpointUpdater EndpointUpdater, aiAnalyzer AIAnalyzer, notifier Notifier, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	s := &Service{
		repo:            repo,
		endpointUpdater: endpointUpdater,
		aiAnalyzer:      aiAnalyzer,
		notifier:        notifier,
		config:          config,
		logger:          utils.NewLogger("remediation"),
	}

	s.executor = NewDefaultExecutor(endpointUpdater)

	return s
}

// AnalyzeAndSuggest analyzes a failure and returns remediation suggestions
func (s *Service) AnalyzeAndSuggest(ctx context.Context, tenantID, deliveryID string) (*AnalysisWithRemediation, error) {
	if s.aiAnalyzer == nil {
		return nil, fmt.Errorf("AI analyzer not configured")
	}

	analysis, err := s.aiAnalyzer.AnalyzeFailure(ctx, tenantID, deliveryID)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Determine if auto-remediable based on suggestions
	for _, suggestion := range analysis.Suggestions {
		if suggestion.ConfidenceScore >= 0.8 && suggestion.Reversible {
			analysis.AutoRemediable = true
			break
		}
	}

	return analysis, nil
}

// CreateAction creates a new remediation action
func (s *Service) CreateAction(ctx context.Context, tenantID string, req *CreateActionRequest) (*RemediationAction, error) {
	// Get policy for tenant
	policy, err := s.repo.GetPolicy(ctx, tenantID)
	if err != nil {
		// Use default policy
		policy = DefaultRemediationPolicy(tenantID)
	}

	if !policy.Enabled && !s.config.EnableAutoRemediation {
		return nil, fmt.Errorf("remediation is disabled")
	}

	// Get current endpoint state for rollback
	var previousState json.RawMessage
	if s.endpointUpdater != nil {
		state, err := s.endpointUpdater.GetEndpointState(ctx, tenantID, req.EndpointID)
		if err == nil {
			marshaled, marshalErr := json.Marshal(state)
			if marshalErr != nil {
				s.logger.Error("failed to marshal previous state", map[string]interface{}{"endpoint_id": req.EndpointID, "error": marshalErr.Error()})
			} else {
				previousState = marshaled
			}
		}
	}

	now := time.Now()
	confidenceLevel := GetConfidenceLevel(req.ConfidenceScore)

	// Calculate expiry
	var expiresAt *time.Time
	if policy.ActionExpirySec > 0 {
		expiry := now.Add(time.Duration(policy.ActionExpirySec) * time.Second)
		expiresAt = &expiry
	}

	action := &RemediationAction{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		EndpointID:      req.EndpointID,
		DeliveryID:      req.DeliveryID,
		ActionType:      req.ActionType,
		Status:          ActionStatusPending,
		Description:     req.Description,
		Reason:          fmt.Sprintf("Auto-generated from analysis"),
		ConfidenceScore: req.ConfidenceScore,
		ConfidenceLevel: confidenceLevel,
		Parameters:      req.Parameters,
		PreviousState:   previousState,
		RollbackAvail:   true,
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Check if action should be auto-approved
	shouldAutoApprove := s.shouldAutoApprove(ctx, action, policy)

	if shouldAutoApprove {
		action.AutoApproved = true
		action.Status = ActionStatusApproved
		approvedAt := now
		action.ApprovedAt = &approvedAt
		action.ApprovedBy = "system"
	}

	// Save action
	if err := s.repo.CreateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	// Create audit log
	s.createAuditLog(ctx, action, "created", "system", nil)

	// Send notification
	if s.notifier != nil && policy.NotifyOnAction {
		if shouldAutoApprove {
			if err := s.notifier.SendActionApproved(ctx, action); err != nil {
				s.logger.Error("failed to send action approved notification", map[string]interface{}{"action_id": action.ID, "error": err.Error()})
			}
		} else {
			if err := s.notifier.SendActionPending(ctx, action); err != nil {
				s.logger.Error("failed to send action pending notification", map[string]interface{}{"action_id": action.ID, "error": err.Error()})
			}
		}
	}

	// Auto-execute if approved
	if action.Status == ActionStatusApproved {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.executeActionAsync(ctx, action)
		}()
	}

	return action, nil
}

// shouldAutoApprove determines if an action should be auto-approved
func (s *Service) shouldAutoApprove(ctx context.Context, action *RemediationAction, policy *RemediationPolicy) bool {
	// Manual policy requires approval for everything
	if policy.ApprovalPolicy == ApprovalPolicyManual {
		return false
	}

	// Check if action type requires approval
	for _, requiredType := range policy.RequireApprovalFor {
		if action.ActionType == requiredType {
			return false
		}
	}

	// Check confidence threshold
	if action.ConfidenceScore < policy.AutoApproveThreshold {
		return false
	}

	// Check rate limiting
	count, err := s.repo.GetAutoActionCount(ctx, action.TenantID, time.Now().Add(-1*time.Hour))
	if err == nil && count >= policy.MaxAutoActionsPerHour {
		return false
	}

	// Check cooldown
	hasCooldown, err := s.repo.HasCooldown(ctx, action.TenantID, action.EndpointID)
	if err == nil && hasCooldown {
		return false
	}

	return true
}

// ApproveAction approves a pending remediation action
func (s *Service) ApproveAction(ctx context.Context, tenantID, actionID, approvedBy string) (*RemediationAction, error) {
	action, err := s.repo.GetAction(ctx, tenantID, actionID)
	if err != nil {
		return nil, err
	}

	if action.Status != ActionStatusPending {
		return nil, ErrInvalidTransition
	}

	// Check expiry
	if action.ExpiresAt != nil && time.Now().After(*action.ExpiresAt) {
		return nil, ErrActionExpired
	}

	now := time.Now()
	action.Status = ActionStatusApproved
	action.ApprovedBy = approvedBy
	action.ApprovedAt = &now
	action.UpdatedAt = now

	if err := s.repo.UpdateAction(ctx, action); err != nil {
		return nil, err
	}

	s.createAuditLog(ctx, action, "approved", approvedBy, nil)

	if s.notifier != nil {
		if err := s.notifier.SendActionApproved(ctx, action); err != nil {
			s.logger.Error("failed to send action approved notification", map[string]interface{}{"action_id": action.ID, "error": err.Error()})
		}
	}

	// Execute async
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.executeActionAsync(ctx, action)
	}()

	return action, nil
}

// RejectAction rejects a pending remediation action
func (s *Service) RejectAction(ctx context.Context, tenantID, actionID, rejectedBy, reason string) (*RemediationAction, error) {
	action, err := s.repo.GetAction(ctx, tenantID, actionID)
	if err != nil {
		return nil, err
	}

	if action.Status != ActionStatusPending {
		return nil, ErrInvalidTransition
	}

	now := time.Now()
	action.Status = ActionStatusRejected
	action.RejectedBy = rejectedBy
	action.RejectedAt = &now
	action.RejectionReason = reason
	action.UpdatedAt = now

	if err := s.repo.UpdateAction(ctx, action); err != nil {
		return nil, err
	}

	s.createAuditLog(ctx, action, "rejected", rejectedBy, map[string]string{"reason": reason})

	return action, nil
}

// executeActionAsync executes an action asynchronously
func (s *Service) executeActionAsync(ctx context.Context, action *RemediationAction) {
	// Update status to executing
	action.Status = ActionStatusExecuting
	now := time.Now()
	action.ExecutedAt = &now
	action.UpdatedAt = now
	if err := s.repo.UpdateAction(ctx, action); err != nil {
		s.logger.Error("failed to update action to executing", map[string]interface{}{"action_id": action.ID, "error": err.Error()})
	}

	// Execute the action
	err := s.executor.Execute(ctx, action)

	if err != nil {
		action.Status = ActionStatusFailed
		action.ErrorMessage = err.Error()
		s.createAuditLog(ctx, action, "failed", "system", map[string]string{"error": err.Error()})
	} else {
		action.Status = ActionStatusCompleted
		completedAt := time.Now()
		action.CompletedAt = &completedAt

		// Capture new state
		if s.endpointUpdater != nil {
			state, err := s.endpointUpdater.GetEndpointState(ctx, action.TenantID, action.EndpointID)
			if err == nil {
				marshaled, marshalErr := json.Marshal(state)
				if marshalErr != nil {
					s.logger.Error("failed to marshal new state", map[string]interface{}{"action_id": action.ID, "error": marshalErr.Error()})
				} else {
					action.NewState = marshaled
				}
			}
		}

		s.createAuditLog(ctx, action, "executed", "system", nil)
	}

	action.UpdatedAt = time.Now()
	if err := s.repo.UpdateAction(ctx, action); err != nil {
		s.logger.Error("failed to update action after execution", map[string]interface{}{"action_id": action.ID, "error": err.Error()})
	}

	// Notify
	if s.notifier != nil {
		if err := s.notifier.SendActionExecuted(ctx, action, action.Status == ActionStatusCompleted); err != nil {
			s.logger.Error("failed to send execution notification", map[string]interface{}{"action_id": action.ID, "error": err.Error()})
		}
	}

	// Set cooldown
	policy, _ := s.repo.GetPolicy(ctx, action.TenantID)
	if policy != nil && policy.CooldownPeriodSec > 0 {
		if err := s.repo.SetCooldown(ctx, action.TenantID, action.EndpointID,
			time.Duration(policy.CooldownPeriodSec)*time.Second); err != nil {
			s.logger.Error("failed to set cooldown", map[string]interface{}{"action_id": action.ID, "error": err.Error()})
		}
	}

	// Increment auto action count if auto-approved
	if action.AutoApproved {
		if _, err := s.repo.IncrementAutoActionCount(ctx, action.TenantID); err != nil {
			s.logger.Error("failed to increment auto action count", map[string]interface{}{"tenant_id": action.TenantID, "error": err.Error()})
		}
	}
}

// RollbackAction rolls back a completed action
func (s *Service) RollbackAction(ctx context.Context, tenantID, actionID, rolledBackBy string) (*RemediationAction, error) {
	action, err := s.repo.GetAction(ctx, tenantID, actionID)
	if err != nil {
		return nil, err
	}

	if action.Status != ActionStatusCompleted {
		return nil, fmt.Errorf("can only rollback completed actions")
	}

	if !action.RollbackAvail {
		return nil, fmt.Errorf("rollback not available for this action")
	}

	if action.PreviousState == nil {
		return nil, fmt.Errorf("no previous state to rollback to")
	}

	// Perform rollback
	err = s.executor.Rollback(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	action.Status = ActionStatusRolledBack
	action.UpdatedAt = time.Now()
	action.RollbackAvail = false

	if err := s.repo.UpdateAction(ctx, action); err != nil {
		return nil, err
	}

	s.createAuditLog(ctx, action, "rolled_back", rolledBackBy, nil)

	return action, nil
}

// GetAction retrieves an action
func (s *Service) GetAction(ctx context.Context, tenantID, actionID string) (*RemediationAction, error) {
	return s.repo.GetAction(ctx, tenantID, actionID)
}

// ListActions lists actions with filters
func (s *Service) ListActions(ctx context.Context, tenantID string, filters *ActionFilters) (*ListActionsResponse, error) {
	actions, total, err := s.repo.ListActions(ctx, tenantID, filters)
	if err != nil {
		return nil, err
	}

	page := 1
	pageSize := 20
	if filters != nil {
		if filters.Page > 0 {
			page = filters.Page
		}
		if filters.PageSize > 0 {
			pageSize = filters.PageSize
		}
	}

	return &ListActionsResponse{
		Actions:    actions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}

// GetPendingActions gets all pending actions
func (s *Service) GetPendingActions(ctx context.Context, tenantID string) ([]RemediationAction, error) {
	return s.repo.GetPendingActions(ctx, tenantID)
}

// GetPolicy retrieves the remediation policy
func (s *Service) GetPolicy(ctx context.Context, tenantID string) (*RemediationPolicy, error) {
	policy, err := s.repo.GetPolicy(ctx, tenantID)
	if err == ErrPolicyNotFound {
		return DefaultRemediationPolicy(tenantID), nil
	}
	return policy, err
}

// UpdatePolicy updates the remediation policy
func (s *Service) UpdatePolicy(ctx context.Context, tenantID string, policy *RemediationPolicy) error {
	policy.TenantID = tenantID
	policy.UpdatedAt = time.Now()
	return s.repo.UpdatePolicy(ctx, policy)
}

// GetMetrics retrieves remediation metrics
func (s *Service) GetMetrics(ctx context.Context, tenantID, period string) (*RemediationMetrics, error) {
	return s.repo.GetMetrics(ctx, tenantID, period)
}

// GetAuditLogs retrieves audit logs for an action
func (s *Service) GetAuditLogs(ctx context.Context, actionID string) ([]ActionAuditLog, error) {
	return s.repo.GetAuditLogs(ctx, actionID)
}

// createAuditLog creates an audit log entry
func (s *Service) createAuditLog(ctx context.Context, action *RemediationAction, event, actor string, details map[string]string) {
	var detailsJSON json.RawMessage
	if details != nil {
		marshaled, marshalErr := json.Marshal(details)
		if marshalErr != nil {
			s.logger.Error("failed to marshal audit log details", map[string]interface{}{"action_id": action.ID, "error": marshalErr.Error()})
		} else {
			detailsJSON = marshaled
		}
	}

	auditLog := &ActionAuditLog{
		ID:        uuid.New().String(),
		ActionID:  action.ID,
		TenantID:  action.TenantID,
		Event:     event,
		Actor:     actor,
		Details:   detailsJSON,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateAuditLog(ctx, auditLog); err != nil {
		s.logger.Error("failed to create audit log", map[string]interface{}{"action_id": action.ID, "event": event, "error": err.Error()})
	}
}

// CleanupExpiredActions marks expired pending actions
func (s *Service) CleanupExpiredActions(ctx context.Context) error {
	expired, err := s.repo.GetExpiredActions(ctx)
	if err != nil {
		return err
	}

	for _, action := range expired {
		action.Status = ActionStatusFailed
		action.ErrorMessage = "Action expired"
		action.UpdatedAt = time.Now()
		if err := s.repo.UpdateAction(ctx, &action); err != nil {
			s.logger.Error("failed to mark expired action as failed", map[string]interface{}{"action_id": action.ID, "error": err.Error()})
		}
		s.createAuditLog(ctx, &action, "expired", "system", nil)
	}

	return nil
}

// DefaultExecutor implements ActionExecutor
type DefaultExecutor struct {
	updater EndpointUpdater
}

// NewDefaultExecutor creates a new executor
func NewDefaultExecutor(updater EndpointUpdater) *DefaultExecutor {
	return &DefaultExecutor{updater: updater}
}

// Execute executes a remediation action
func (e *DefaultExecutor) Execute(ctx context.Context, action *RemediationAction) error {
	if e.updater == nil {
		return fmt.Errorf("endpoint updater not configured")
	}

	switch action.ActionType {
	case ActionTypeURLUpdate:
		if action.Parameters.NewURL == "" {
			return fmt.Errorf("new URL is required")
		}
		return e.updater.UpdateURL(ctx, action.TenantID, action.EndpointID, action.Parameters.NewURL)

	case ActionTypeHeaderUpdate:
		headers := make(map[string]string)
		if action.Parameters.HeaderName != "" {
			if action.Parameters.RemoveHeader {
				headers[action.Parameters.HeaderName] = "" // Empty to remove
			} else {
				headers[action.Parameters.HeaderName] = action.Parameters.NewValue
			}
		}
		return e.updater.UpdateHeaders(ctx, action.TenantID, action.EndpointID, headers)

	case ActionTypeTimeoutAdjust:
		if action.Parameters.NewTimeoutMs <= 0 {
			return fmt.Errorf("new timeout must be positive")
		}
		return e.updater.UpdateTimeout(ctx, action.TenantID, action.EndpointID, action.Parameters.NewTimeoutMs)

	case ActionTypeRetryConfig:
		return e.updater.UpdateRetryConfig(ctx, action.TenantID, action.EndpointID,
			action.Parameters.NewMaxRetries, action.Parameters.NewBackoffMs)

	case ActionTypeEndpointDisable:
		return e.updater.DisableEndpoint(ctx, action.TenantID, action.EndpointID)

	case ActionTypeAlertOnly:
		// No action needed, just create awareness
		return nil

	default:
		return fmt.Errorf("unsupported action type: %s", action.ActionType)
	}
}

// Rollback rolls back a remediation action
func (e *DefaultExecutor) Rollback(ctx context.Context, action *RemediationAction) error {
	if action.PreviousState == nil {
		return fmt.Errorf("no previous state available")
	}

	var state map[string]interface{}
	if err := json.Unmarshal(action.PreviousState, &state); err != nil {
		return fmt.Errorf("failed to parse previous state: %w", err)
	}

	// Restore based on action type
	switch action.ActionType {
	case ActionTypeURLUpdate:
		if url, ok := state["url"].(string); ok {
			return e.updater.UpdateURL(ctx, action.TenantID, action.EndpointID, url)
		}
	case ActionTypeTimeoutAdjust:
		if timeout, ok := state["timeout_ms"].(float64); ok {
			return e.updater.UpdateTimeout(ctx, action.TenantID, action.EndpointID, int(timeout))
		}
	case ActionTypeHeaderUpdate:
		if headers, ok := state["headers"].(map[string]interface{}); ok {
			stringHeaders := make(map[string]string)
			for k, v := range headers {
				stringHeaders[k] = fmt.Sprintf("%v", v)
			}
			return e.updater.UpdateHeaders(ctx, action.TenantID, action.EndpointID, stringHeaders)
		}
	}

	return nil
}

// Validate validates a remediation action
func (e *DefaultExecutor) Validate(ctx context.Context, action *RemediationAction) error {
	switch action.ActionType {
	case ActionTypeURLUpdate:
		if action.Parameters.NewURL == "" {
			return fmt.Errorf("new URL is required")
		}
	case ActionTypeTimeoutAdjust:
		if action.Parameters.NewTimeoutMs <= 0 {
			return fmt.Errorf("new timeout must be positive")
		}
	}
	return nil
}
