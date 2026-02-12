package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type MockRepository struct{ mock.Mock }

func (m *MockRepository) CreateAction(ctx context.Context, action *RemediationAction) error {
	return m.Called(ctx, action).Error(0)
}
func (m *MockRepository) GetAction(ctx context.Context, tenantID, actionID string) (*RemediationAction, error) {
	args := m.Called(ctx, tenantID, actionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RemediationAction), args.Error(1)
}
func (m *MockRepository) UpdateAction(ctx context.Context, action *RemediationAction) error {
	return m.Called(ctx, action).Error(0)
}
func (m *MockRepository) DeleteAction(ctx context.Context, tenantID, actionID string) error {
	return m.Called(ctx, tenantID, actionID).Error(0)
}
func (m *MockRepository) ListActions(ctx context.Context, tenantID string, filters *ActionFilters) ([]RemediationAction, int, error) {
	args := m.Called(ctx, tenantID, filters)
	return args.Get(0).([]RemediationAction), args.Int(1), args.Error(2)
}
func (m *MockRepository) GetPendingActions(ctx context.Context, tenantID string) ([]RemediationAction, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]RemediationAction), args.Error(1)
}
func (m *MockRepository) GetExpiredActions(ctx context.Context) ([]RemediationAction, error) {
	args := m.Called(ctx)
	return args.Get(0).([]RemediationAction), args.Error(1)
}
func (m *MockRepository) CreatePolicy(ctx context.Context, policy *RemediationPolicy) error {
	return m.Called(ctx, policy).Error(0)
}
func (m *MockRepository) GetPolicy(ctx context.Context, tenantID string) (*RemediationPolicy, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RemediationPolicy), args.Error(1)
}
func (m *MockRepository) UpdatePolicy(ctx context.Context, policy *RemediationPolicy) error {
	return m.Called(ctx, policy).Error(0)
}
func (m *MockRepository) DeletePolicy(ctx context.Context, tenantID string) error {
	return m.Called(ctx, tenantID).Error(0)
}
func (m *MockRepository) CreateAuditLog(ctx context.Context, log *ActionAuditLog) error {
	return m.Called(ctx, log).Error(0)
}
func (m *MockRepository) GetAuditLogs(ctx context.Context, actionID string) ([]ActionAuditLog, error) {
	args := m.Called(ctx, actionID)
	return args.Get(0).([]ActionAuditLog), args.Error(1)
}
func (m *MockRepository) SaveMetrics(ctx context.Context, metrics *RemediationMetrics) error {
	return m.Called(ctx, metrics).Error(0)
}
func (m *MockRepository) GetMetrics(ctx context.Context, tenantID, period string) (*RemediationMetrics, error) {
	args := m.Called(ctx, tenantID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RemediationMetrics), args.Error(1)
}
func (m *MockRepository) SetCooldown(ctx context.Context, tenantID, endpointID string, duration time.Duration) error {
	return m.Called(ctx, tenantID, endpointID, duration).Error(0)
}
func (m *MockRepository) HasCooldown(ctx context.Context, tenantID, endpointID string) (bool, error) {
	args := m.Called(ctx, tenantID, endpointID)
	return args.Bool(0), args.Error(1)
}
func (m *MockRepository) IncrementAutoActionCount(ctx context.Context, tenantID string) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockRepository) GetAutoActionCount(ctx context.Context, tenantID string, since time.Time) (int, error) {
	args := m.Called(ctx, tenantID, since)
	return args.Int(0), args.Error(1)
}

type MockEndpointUpdater struct{ mock.Mock }

func (m *MockEndpointUpdater) UpdateURL(ctx context.Context, tenantID, endpointID, newURL string) error {
	return m.Called(ctx, tenantID, endpointID, newURL).Error(0)
}
func (m *MockEndpointUpdater) UpdateHeaders(ctx context.Context, tenantID, endpointID string, headers map[string]string) error {
	return m.Called(ctx, tenantID, endpointID, headers).Error(0)
}
func (m *MockEndpointUpdater) UpdateTimeout(ctx context.Context, tenantID, endpointID string, timeoutMs int) error {
	return m.Called(ctx, tenantID, endpointID, timeoutMs).Error(0)
}
func (m *MockEndpointUpdater) UpdateRetryConfig(ctx context.Context, tenantID, endpointID string, maxRetries, backoffMs int) error {
	return m.Called(ctx, tenantID, endpointID, maxRetries, backoffMs).Error(0)
}
func (m *MockEndpointUpdater) DisableEndpoint(ctx context.Context, tenantID, endpointID string) error {
	return m.Called(ctx, tenantID, endpointID).Error(0)
}
func (m *MockEndpointUpdater) GetEndpointState(ctx context.Context, tenantID, endpointID string) (map[string]interface{}, error) {
	args := m.Called(ctx, tenantID, endpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

type MockAIAnalyzer struct{ mock.Mock }

func (m *MockAIAnalyzer) AnalyzeFailure(ctx context.Context, tenantID, deliveryID string) (*AnalysisWithRemediation, error) {
	args := m.Called(ctx, tenantID, deliveryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AnalysisWithRemediation), args.Error(1)
}
func (m *MockAIAnalyzer) GenerateSuggestions(ctx context.Context, tenantID, endpointID string, errorPattern string) ([]RemediationSuggestion, error) {
	args := m.Called(ctx, tenantID, endpointID, errorPattern)
	return args.Get(0).([]RemediationSuggestion), args.Error(1)
}

type MockNotifier struct{ mock.Mock }

func (m *MockNotifier) SendActionCreated(ctx context.Context, action *RemediationAction) error {
	return m.Called(ctx, action).Error(0)
}
func (m *MockNotifier) SendActionApproved(ctx context.Context, action *RemediationAction) error {
	return m.Called(ctx, action).Error(0)
}
func (m *MockNotifier) SendActionExecuted(ctx context.Context, action *RemediationAction, success bool) error {
	return m.Called(ctx, action, success).Error(0)
}
func (m *MockNotifier) SendActionPending(ctx context.Context, action *RemediationAction) error {
	return m.Called(ctx, action).Error(0)
}

// --- Tests ---

func TestAnalyzeAndSuggest_NilAnalyzer(t *testing.T) {
	svc := NewService(&MockRepository{}, nil, nil, nil, nil)

	_, err := svc.AnalyzeAndSuggest(context.Background(), "t1", "d1")
	assert.ErrorContains(t, err, "not configured")
}

func TestAnalyzeAndSuggest_ValidAnalysis(t *testing.T) {
	analyzer := new(MockAIAnalyzer)
	analysis := &AnalysisWithRemediation{
		AnalysisID: "a1",
		Suggestions: []RemediationSuggestion{
			{ConfidenceScore: 0.9, Reversible: true, ActionType: ActionTypeURLUpdate},
		},
	}
	analyzer.On("AnalyzeFailure", mock.Anything, "t1", "d1").Return(analysis, nil)

	svc := NewService(&MockRepository{}, nil, analyzer, nil, nil)
	result, err := svc.AnalyzeAndSuggest(context.Background(), "t1", "d1")

	require.NoError(t, err)
	assert.True(t, result.AutoRemediable)
	analyzer.AssertExpectations(t)
}

func TestAnalyzeAndSuggest_NotAutoRemediable(t *testing.T) {
	analyzer := new(MockAIAnalyzer)
	analysis := &AnalysisWithRemediation{
		AnalysisID: "a1",
		Suggestions: []RemediationSuggestion{
			{ConfidenceScore: 0.5, Reversible: true},
			{ConfidenceScore: 0.9, Reversible: false},
		},
	}
	analyzer.On("AnalyzeFailure", mock.Anything, "t1", "d1").Return(analysis, nil)

	svc := NewService(&MockRepository{}, nil, analyzer, nil, nil)
	result, err := svc.AnalyzeAndSuggest(context.Background(), "t1", "d1")

	require.NoError(t, err)
	assert.False(t, result.AutoRemediable)
}

func TestCreateAction_PendingStatus(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)

	endpointState := map[string]interface{}{"url": "https://old.example.com"}
	updater.On("GetEndpointState", mock.Anything, "t1", "ep1").Return(endpointState, nil)

	// Policy not found → default manual policy → no auto-approve
	repo.On("GetPolicy", mock.Anything, "t1").Return(nil, ErrPolicyNotFound)
	repo.On("GetAutoActionCount", mock.Anything, "t1", mock.Anything).Return(0, nil).Maybe()
	repo.On("HasCooldown", mock.Anything, "t1", "ep1").Return(false, nil).Maybe()
	repo.On("CreateAction", mock.Anything, mock.AnythingOfType("*remediation.RemediationAction")).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()

	// Async goroutine calls — use Maybe()
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionPending", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionApproved", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, updater, nil, notifier, nil)
	req := &CreateActionRequest{
		EndpointID:      "ep1",
		ActionType:      ActionTypeURLUpdate,
		Description:     "test",
		ConfidenceScore: 0.5,
		Parameters:      ActionParams{NewURL: "https://new.example.com"},
	}

	action, err := svc.CreateAction(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.Equal(t, ActionStatusPending, action.Status)
	assert.Equal(t, "t1", action.TenantID)
	assert.NotEmpty(t, action.ID)
	assert.NotNil(t, action.PreviousState)

	// Verify synchronous calls
	repo.AssertCalled(t, "CreateAction", mock.Anything, mock.Anything)
}

func TestCreateAction_CapturesPreviousState(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)

	state := map[string]interface{}{"url": "https://old.example.com", "timeout_ms": float64(5000)}
	updater.On("GetEndpointState", mock.Anything, "t1", "ep1").Return(state, nil)

	repo.On("GetPolicy", mock.Anything, "t1").Return(nil, ErrPolicyNotFound)
	repo.On("CreateAction", mock.Anything, mock.AnythingOfType("*remediation.RemediationAction")).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionPending", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, updater, nil, notifier, nil)
	action, err := svc.CreateAction(context.Background(), "t1", &CreateActionRequest{
		EndpointID: "ep1",
		ActionType: ActionTypeURLUpdate,
		Parameters: ActionParams{NewURL: "https://new.example.com"},
	})

	require.NoError(t, err)
	require.NotNil(t, action.PreviousState)

	var prevState map[string]interface{}
	err = json.Unmarshal(action.PreviousState, &prevState)
	require.NoError(t, err)
	assert.Equal(t, "https://old.example.com", prevState["url"])
}

func TestCreateAction_AuditLogCreated(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)

	updater.On("GetEndpointState", mock.Anything, "t1", "ep1").Return(map[string]interface{}{}, nil)

	repo.On("GetPolicy", mock.Anything, "t1").Return(nil, ErrPolicyNotFound)
	repo.On("CreateAction", mock.Anything, mock.AnythingOfType("*remediation.RemediationAction")).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.MatchedBy(func(log *ActionAuditLog) bool {
		return log.Event == "created" && log.Actor == "system"
	})).Return(nil).Once()
	// Allow additional audit log calls from async goroutine
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionPending", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, updater, nil, notifier, nil)
	_, err := svc.CreateAction(context.Background(), "t1", &CreateActionRequest{
		EndpointID: "ep1",
		ActionType: ActionTypeAlertOnly,
	})
	require.NoError(t, err)
}

func TestApproveAction_Success(t *testing.T) {
	repo := new(MockRepository)
	action := &RemediationAction{
		ID:       "a1",
		TenantID: "t1",
		Status:   ActionStatusPending,
	}

	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)
	repo.On("UpdateAction", mock.Anything, mock.MatchedBy(func(a *RemediationAction) bool {
		return a.Status == ActionStatusApproved && a.ApprovedBy == "admin"
	})).Return(nil).Once()
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	// Async goroutine calls
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("GetPolicy", mock.Anything, mock.Anything).Return(nil, ErrPolicyNotFound).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionApproved", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, nil, nil, notifier, nil)
	result, err := svc.ApproveAction(context.Background(), "t1", "a1", "admin")

	require.NoError(t, err)
	assert.Equal(t, ActionStatusApproved, result.Status)
	assert.Equal(t, "admin", result.ApprovedBy)
	assert.NotNil(t, result.ApprovedAt)
}

func TestApproveAction_NonPending(t *testing.T) {
	repo := new(MockRepository)
	action := &RemediationAction{
		ID:       "a1",
		TenantID: "t1",
		Status:   ActionStatusCompleted,
	}
	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)

	svc := NewService(repo, nil, nil, nil, nil)
	_, err := svc.ApproveAction(context.Background(), "t1", "a1", "admin")

	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestApproveAction_Expired(t *testing.T) {
	repo := new(MockRepository)
	expiry := time.Now().Add(-1 * time.Hour)
	action := &RemediationAction{
		ID:        "a1",
		TenantID:  "t1",
		Status:    ActionStatusPending,
		ExpiresAt: &expiry,
	}
	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)

	svc := NewService(repo, nil, nil, nil, nil)
	_, err := svc.ApproveAction(context.Background(), "t1", "a1", "admin")

	assert.ErrorIs(t, err, ErrActionExpired)
}

func TestApproveAction_AuditLog(t *testing.T) {
	repo := new(MockRepository)
	action := &RemediationAction{
		ID:       "a1",
		TenantID: "t1",
		Status:   ActionStatusPending,
	}

	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("CreateAuditLog", mock.Anything, mock.MatchedBy(func(log *ActionAuditLog) bool {
		return log.Event == "approved" && log.Actor == "admin"
	})).Return(nil).Once()
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	repo.On("GetPolicy", mock.Anything, mock.Anything).Return(nil, ErrPolicyNotFound).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionApproved", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, nil, nil, notifier, nil)
	_, err := svc.ApproveAction(context.Background(), "t1", "a1", "admin")
	require.NoError(t, err)
}

func TestRejectAction_Success(t *testing.T) {
	repo := new(MockRepository)
	action := &RemediationAction{
		ID:       "a1",
		TenantID: "t1",
		Status:   ActionStatusPending,
	}

	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)
	repo.On("UpdateAction", mock.Anything, mock.MatchedBy(func(a *RemediationAction) bool {
		return a.Status == ActionStatusRejected && a.RejectedBy == "admin" && a.RejectionReason == "not needed"
	})).Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()

	svc := NewService(repo, nil, nil, nil, nil)
	result, err := svc.RejectAction(context.Background(), "t1", "a1", "admin", "not needed")

	require.NoError(t, err)
	assert.Equal(t, ActionStatusRejected, result.Status)
	assert.Equal(t, "admin", result.RejectedBy)
	assert.Equal(t, "not needed", result.RejectionReason)
	assert.NotNil(t, result.RejectedAt)
}

func TestRejectAction_NonPending(t *testing.T) {
	repo := new(MockRepository)
	action := &RemediationAction{
		ID:       "a1",
		TenantID: "t1",
		Status:   ActionStatusApproved,
	}
	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)

	svc := NewService(repo, nil, nil, nil, nil)
	_, err := svc.RejectAction(context.Background(), "t1", "a1", "admin", "reason")

	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestRollbackAction_Success(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)
	prevState, _ := json.Marshal(map[string]interface{}{"url": "https://old.example.com"})
	action := &RemediationAction{
		ID:            "a1",
		TenantID:      "t1",
		EndpointID:    "ep1",
		Status:        ActionStatusCompleted,
		ActionType:    ActionTypeURLUpdate,
		RollbackAvail: true,
		PreviousState: prevState,
	}

	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)
	updater.On("UpdateURL", mock.Anything, "t1", "ep1", "https://old.example.com").Return(nil)
	repo.On("UpdateAction", mock.Anything, mock.MatchedBy(func(a *RemediationAction) bool {
		return a.Status == ActionStatusRolledBack && !a.RollbackAvail
	})).Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()

	svc := NewService(repo, updater, nil, nil, nil)
	result, err := svc.RollbackAction(context.Background(), "t1", "a1", "admin")

	require.NoError(t, err)
	assert.Equal(t, ActionStatusRolledBack, result.Status)
	assert.False(t, result.RollbackAvail)
	updater.AssertExpectations(t)
}

func TestRollbackAction_NonCompleted(t *testing.T) {
	repo := new(MockRepository)
	action := &RemediationAction{
		ID:       "a1",
		TenantID: "t1",
		Status:   ActionStatusPending,
	}
	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)

	svc := NewService(repo, nil, nil, nil, nil)
	_, err := svc.RollbackAction(context.Background(), "t1", "a1", "admin")

	assert.ErrorContains(t, err, "can only rollback completed actions")
}

func TestRollbackAction_NoPreviousState(t *testing.T) {
	repo := new(MockRepository)
	action := &RemediationAction{
		ID:            "a1",
		TenantID:      "t1",
		Status:        ActionStatusCompleted,
		RollbackAvail: true,
		PreviousState: nil,
	}
	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)

	svc := NewService(repo, nil, nil, nil, nil)
	_, err := svc.RollbackAction(context.Background(), "t1", "a1", "admin")

	assert.ErrorContains(t, err, "no previous state to rollback to")
}

func TestRollbackAction_NotAvailable(t *testing.T) {
	repo := new(MockRepository)
	action := &RemediationAction{
		ID:            "a1",
		TenantID:      "t1",
		Status:        ActionStatusCompleted,
		RollbackAvail: false,
	}
	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)

	svc := NewService(repo, nil, nil, nil, nil)
	_, err := svc.RollbackAction(context.Background(), "t1", "a1", "admin")

	assert.ErrorContains(t, err, "rollback not available")
}

func TestGetPolicy_NotFound_ReturnsDefault(t *testing.T) {
	repo := new(MockRepository)
	repo.On("GetPolicy", mock.Anything, "t1").Return(nil, ErrPolicyNotFound)

	svc := NewService(repo, nil, nil, nil, nil)
	policy, err := svc.GetPolicy(context.Background(), "t1")

	require.NoError(t, err)
	assert.Equal(t, "t1", policy.TenantID)
	assert.Equal(t, ApprovalPolicyManual, policy.ApprovalPolicy)
	assert.True(t, policy.Enabled)
}

func TestCleanupExpiredActions(t *testing.T) {
	repo := new(MockRepository)
	expired := []RemediationAction{
		{ID: "a1", TenantID: "t1", Status: ActionStatusPending},
		{ID: "a2", TenantID: "t2", Status: ActionStatusPending},
	}

	repo.On("GetExpiredActions", mock.Anything).Return(expired, nil)
	repo.On("UpdateAction", mock.Anything, mock.MatchedBy(func(a *RemediationAction) bool {
		return a.Status == ActionStatusFailed && a.ErrorMessage == "Action expired"
	})).Return(nil).Times(2)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()

	svc := NewService(repo, nil, nil, nil, nil)
	err := svc.CleanupExpiredActions(context.Background())

	require.NoError(t, err)
}

// --- DefaultExecutor Tests ---

func TestDefaultExecutor_Execute_URLUpdate(t *testing.T) {
	updater := new(MockEndpointUpdater)
	updater.On("UpdateURL", mock.Anything, "t1", "ep1", "https://new.example.com").Return(nil)

	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		TenantID:   "t1",
		EndpointID: "ep1",
		ActionType: ActionTypeURLUpdate,
		Parameters: ActionParams{NewURL: "https://new.example.com"},
	})

	assert.NoError(t, err)
	updater.AssertExpectations(t)
}

func TestDefaultExecutor_Execute_TimeoutAdjust(t *testing.T) {
	updater := new(MockEndpointUpdater)
	updater.On("UpdateTimeout", mock.Anything, "t1", "ep1", 10000).Return(nil)

	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		TenantID:   "t1",
		EndpointID: "ep1",
		ActionType: ActionTypeTimeoutAdjust,
		Parameters: ActionParams{NewTimeoutMs: 10000},
	})

	assert.NoError(t, err)
	updater.AssertExpectations(t)
}

func TestDefaultExecutor_Execute_AlertOnly(t *testing.T) {
	updater := new(MockEndpointUpdater)
	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		ActionType: ActionTypeAlertOnly,
	})

	assert.NoError(t, err)
}

func TestDefaultExecutor_Execute_NilUpdater(t *testing.T) {
	executor := NewDefaultExecutor(nil)
	err := executor.Execute(context.Background(), &RemediationAction{
		ActionType: ActionTypeURLUpdate,
		Parameters: ActionParams{NewURL: "https://new.example.com"},
	})

	assert.ErrorContains(t, err, "endpoint updater not configured")
}

func TestDefaultExecutor_Execute_UnsupportedType(t *testing.T) {
	updater := new(MockEndpointUpdater)
	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		ActionType: ActionType("unknown_type"),
	})

	assert.ErrorContains(t, err, "unsupported action type")
}

func TestDefaultExecutor_Rollback_URLUpdate(t *testing.T) {
	updater := new(MockEndpointUpdater)
	prevState, _ := json.Marshal(map[string]interface{}{"url": "https://old.example.com"})
	updater.On("UpdateURL", mock.Anything, "t1", "ep1", "https://old.example.com").Return(nil)

	executor := NewDefaultExecutor(updater)
	err := executor.Rollback(context.Background(), &RemediationAction{
		TenantID:      "t1",
		EndpointID:    "ep1",
		ActionType:    ActionTypeURLUpdate,
		PreviousState: prevState,
	})

	assert.NoError(t, err)
	updater.AssertExpectations(t)
}

func TestDefaultExecutor_Rollback_NoPreviousState(t *testing.T) {
	executor := NewDefaultExecutor(nil)
	err := executor.Rollback(context.Background(), &RemediationAction{
		PreviousState: nil,
	})

	assert.ErrorContains(t, err, "no previous state")
}

func TestCreateAction_PolicyNotFound_UsesDefault(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)

	updater.On("GetEndpointState", mock.Anything, "t1", "ep1").Return(map[string]interface{}{}, nil)
	repo.On("GetPolicy", mock.Anything, "t1").Return(nil, ErrPolicyNotFound)
	repo.On("CreateAction", mock.Anything, mock.MatchedBy(func(a *RemediationAction) bool {
		// Default policy is manual, so high-confidence actions remain pending
		return a.Status == ActionStatusPending
	})).Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionPending", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionApproved", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, updater, nil, notifier, nil)
	action, err := svc.CreateAction(context.Background(), "t1", &CreateActionRequest{
		EndpointID:      "ep1",
		ActionType:      ActionTypeURLUpdate,
		ConfidenceScore: 0.95,
		Parameters:      ActionParams{NewURL: "https://new.example.com"},
	})

	require.NoError(t, err)
	// Default policy is manual → stays pending even with high confidence
	assert.Equal(t, ActionStatusPending, action.Status)
	assert.False(t, action.AutoApproved)
}

func TestDefaultExecutor_Execute_URLUpdate_MissingURL(t *testing.T) {
	updater := new(MockEndpointUpdater)
	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		ActionType: ActionTypeURLUpdate,
		Parameters: ActionParams{NewURL: ""},
	})
	assert.ErrorContains(t, err, "new URL is required")
}

func TestDefaultExecutor_Execute_TimeoutAdjust_InvalidTimeout(t *testing.T) {
	updater := new(MockEndpointUpdater)
	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		ActionType: ActionTypeTimeoutAdjust,
		Parameters: ActionParams{NewTimeoutMs: 0},
	})
	assert.ErrorContains(t, err, "new timeout must be positive")
}

func TestDefaultExecutor_Execute_UpdateError(t *testing.T) {
	updater := new(MockEndpointUpdater)
	updater.On("UpdateURL", mock.Anything, "t1", "ep1", "https://new.example.com").
		Return(fmt.Errorf("connection refused"))

	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		TenantID:   "t1",
		EndpointID: "ep1",
		ActionType: ActionTypeURLUpdate,
		Parameters: ActionParams{NewURL: "https://new.example.com"},
	})

	assert.ErrorContains(t, err, "connection refused")
}

func TestCreateAction_NilEndpointUpdater(t *testing.T) {
	repo := new(MockRepository)

	repo.On("GetPolicy", mock.Anything, "t1").Return(nil, ErrPolicyNotFound)
	repo.On("CreateAction", mock.Anything, mock.AnythingOfType("*remediation.RemediationAction")).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionPending", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, nil, nil, notifier, nil)
	action, err := svc.CreateAction(context.Background(), "t1", &CreateActionRequest{
		EndpointID:      "ep1",
		ActionType:      ActionTypeURLUpdate,
		ConfidenceScore: 0.5,
		Parameters:      ActionParams{NewURL: "https://new.example.com"},
	})

	require.NoError(t, err)
	assert.Nil(t, action.PreviousState)
}

func TestCreateAction_AutoApproveHighConfidence(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)

	updater.On("GetEndpointState", mock.Anything, "t1", "ep1").Return(map[string]interface{}{}, nil).Maybe()
	updater.On("UpdateURL", mock.Anything, "t1", "ep1", "https://new.example.com").Return(nil).Maybe()

	policy := &RemediationPolicy{
		TenantID:              "t1",
		Enabled:               true,
		ApprovalPolicy:        ApprovalPolicyAuto,
		AutoApproveThreshold:  0.8,
		MaxAutoActionsPerHour: 10,
		NotifyOnAction:        true,
		ActionExpirySec:       86400,
	}
	repo.On("GetPolicy", mock.Anything, "t1").Return(policy, nil)
	repo.On("GetAutoActionCount", mock.Anything, "t1", mock.Anything).Return(0, nil)
	repo.On("HasCooldown", mock.Anything, "t1", "ep1").Return(false, nil)
	repo.On("CreateAction", mock.Anything, mock.AnythingOfType("*remediation.RemediationAction")).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionApproved", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, updater, nil, notifier, nil)
	action, err := svc.CreateAction(context.Background(), "t1", &CreateActionRequest{
		EndpointID:      "ep1",
		ActionType:      ActionTypeURLUpdate,
		ConfidenceScore: 0.9,
		Parameters:      ActionParams{NewURL: "https://new.example.com"},
	})

	require.NoError(t, err)
	assert.Equal(t, ActionStatusApproved, action.Status)
	assert.True(t, action.AutoApproved)
}

func TestCreateAction_AutoApproveBlockedByRateLimit(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)

	updater.On("GetEndpointState", mock.Anything, "t1", "ep1").Return(map[string]interface{}{}, nil).Maybe()

	policy := &RemediationPolicy{
		TenantID:              "t1",
		Enabled:               true,
		ApprovalPolicy:        ApprovalPolicyAuto,
		AutoApproveThreshold:  0.8,
		MaxAutoActionsPerHour: 10,
		NotifyOnAction:        true,
		ActionExpirySec:       86400,
	}
	repo.On("GetPolicy", mock.Anything, "t1").Return(policy, nil)
	repo.On("GetAutoActionCount", mock.Anything, "t1", mock.Anything).Return(10, nil)
	repo.On("HasCooldown", mock.Anything, "t1", "ep1").Return(false, nil).Maybe()
	repo.On("CreateAction", mock.Anything, mock.AnythingOfType("*remediation.RemediationAction")).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionPending", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, updater, nil, notifier, nil)
	action, err := svc.CreateAction(context.Background(), "t1", &CreateActionRequest{
		EndpointID:      "ep1",
		ActionType:      ActionTypeURLUpdate,
		ConfidenceScore: 0.9,
		Parameters:      ActionParams{NewURL: "https://new.example.com"},
	})

	require.NoError(t, err)
	assert.Equal(t, ActionStatusPending, action.Status)
	assert.False(t, action.AutoApproved)
}

func TestCreateAction_AutoApproveBlockedByCooldown(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)

	updater.On("GetEndpointState", mock.Anything, "t1", "ep1").Return(map[string]interface{}{}, nil).Maybe()

	policy := &RemediationPolicy{
		TenantID:              "t1",
		Enabled:               true,
		ApprovalPolicy:        ApprovalPolicyAuto,
		AutoApproveThreshold:  0.8,
		MaxAutoActionsPerHour: 10,
		NotifyOnAction:        true,
		ActionExpirySec:       86400,
	}
	repo.On("GetPolicy", mock.Anything, "t1").Return(policy, nil)
	repo.On("GetAutoActionCount", mock.Anything, "t1", mock.Anything).Return(0, nil)
	repo.On("HasCooldown", mock.Anything, "t1", "ep1").Return(true, nil)
	repo.On("CreateAction", mock.Anything, mock.AnythingOfType("*remediation.RemediationAction")).
		Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()
	repo.On("UpdateAction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(1, nil).Maybe()

	notifier := new(MockNotifier)
	notifier.On("SendActionPending", mock.Anything, mock.Anything).Return(nil).Maybe()
	notifier.On("SendActionExecuted", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := NewService(repo, updater, nil, notifier, nil)
	action, err := svc.CreateAction(context.Background(), "t1", &CreateActionRequest{
		EndpointID:      "ep1",
		ActionType:      ActionTypeURLUpdate,
		ConfidenceScore: 0.9,
		Parameters:      ActionParams{NewURL: "https://new.example.com"},
	})

	require.NoError(t, err)
	assert.Equal(t, ActionStatusPending, action.Status)
	assert.False(t, action.AutoApproved)
}

func TestDefaultExecutor_Execute_EndpointDisable(t *testing.T) {
	updater := new(MockEndpointUpdater)
	updater.On("DisableEndpoint", mock.Anything, "t1", "ep1").Return(nil)

	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		TenantID:   "t1",
		EndpointID: "ep1",
		ActionType: ActionTypeEndpointDisable,
	})

	assert.NoError(t, err)
	updater.AssertExpectations(t)
}

func TestDefaultExecutor_Execute_RetryConfig(t *testing.T) {
	updater := new(MockEndpointUpdater)
	updater.On("UpdateRetryConfig", mock.Anything, "t1", "ep1", 5, 1000).Return(nil)

	executor := NewDefaultExecutor(updater)
	err := executor.Execute(context.Background(), &RemediationAction{
		TenantID:   "t1",
		EndpointID: "ep1",
		ActionType: ActionTypeRetryConfig,
		Parameters: ActionParams{NewMaxRetries: 5, NewBackoffMs: 1000},
	})

	assert.NoError(t, err)
	updater.AssertExpectations(t)
}

func TestRollbackAction_PreviousStateRestoresTimeout(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)
	prevState, _ := json.Marshal(map[string]interface{}{"timeout_ms": float64(3000)})
	action := &RemediationAction{
		ID:            "a1",
		TenantID:      "t1",
		EndpointID:    "ep1",
		Status:        ActionStatusCompleted,
		ActionType:    ActionTypeTimeoutAdjust,
		RollbackAvail: true,
		PreviousState: prevState,
	}

	repo.On("GetAction", mock.Anything, "t1", "a1").Return(action, nil)
	updater.On("UpdateTimeout", mock.Anything, "t1", "ep1", 3000).Return(nil)
	repo.On("UpdateAction", mock.Anything, mock.MatchedBy(func(a *RemediationAction) bool {
		return a.Status == ActionStatusRolledBack && !a.RollbackAvail
	})).Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).
		Return(nil).Maybe()

	svc := NewService(repo, updater, nil, nil, nil)
	result, err := svc.RollbackAction(context.Background(), "t1", "a1", "admin")

	require.NoError(t, err)
	assert.Equal(t, ActionStatusRolledBack, result.Status)
	assert.False(t, result.RollbackAvail)
	updater.AssertExpectations(t)
}

// ---------- Concurrent approve and execute ----------

func TestApproveAction_ConcurrentCallsOnlyOneSucceeds(t *testing.T) {
	// Verifies that if two ApproveAction calls race, only one transitions from pending.
	// We simulate this sequentially since testify mocks aren't thread-safe.
	repo := new(MockRepository)
	svc := NewService(repo, nil, nil, nil, nil)
	ctx := context.Background()

	action := &RemediationAction{
		ID:       "a-race",
		TenantID: "t1",
		Status:   ActionStatusPending,
	}

	// First call succeeds
	repo.On("GetAction", ctx, "t1", "a-race").Return(action, nil).Once()
	repo.On("UpdateAction", mock.Anything, mock.AnythingOfType("*remediation.RemediationAction")).Return(nil).Maybe()
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*remediation.ActionAuditLog")).Return(nil).Maybe()
	repo.On("GetPolicy", mock.Anything, "t1").Return(nil, ErrPolicyNotFound).Maybe()
	repo.On("SetCooldown", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("IncrementAutoActionCount", mock.Anything, mock.Anything).Return(0, nil).Maybe()

	result1, err1 := svc.ApproveAction(ctx, "t1", "a-race", "user1")
	require.NoError(t, err1)
	assert.Equal(t, ActionStatusApproved, result1.Status)

	// Second call: action is now approved, so it should fail
	approvedAction := &RemediationAction{
		ID:       "a-race",
		TenantID: "t1",
		Status:   ActionStatusApproved,
	}
	repo.On("GetAction", ctx, "t1", "a-race").Return(approvedAction, nil).Once()

	_, err2 := svc.ApproveAction(ctx, "t1", "a-race", "user2")
	assert.ErrorIs(t, err2, ErrInvalidTransition)
}

// ---------- Large state serialization ----------

func TestCreateAction_LargeEndpointState(t *testing.T) {
	repo := new(MockRepository)
	updater := new(MockEndpointUpdater)
	svc := NewService(repo, updater, nil, nil, nil)
	ctx := context.Background()

	// Return a large endpoint state
	largeState := map[string]interface{}{
		"url":        "https://example.com/webhook",
		"headers":    map[string]interface{}{"Authorization": "Bearer " + string(make([]byte, 4096))},
		"timeout_ms": 30000,
	}
	updater.On("GetEndpointState", ctx, "t1", "ep1").Return(largeState, nil)

	repo.On("GetPolicy", ctx, "t1").Return(nil, ErrPolicyNotFound)
	repo.On("CreateAction", ctx, mock.MatchedBy(func(a *RemediationAction) bool {
		return a.PreviousState != nil && len(a.PreviousState) > 0
	})).Return(nil)
	repo.On("CreateAuditLog", ctx, mock.AnythingOfType("*remediation.ActionAuditLog")).Return(nil).Maybe()

	req := &CreateActionRequest{
		EndpointID:      "ep1",
		ActionType:      ActionTypeURLUpdate,
		ConfidenceScore: 0.5,
	}

	result, err := svc.CreateAction(ctx, "t1", req)

	require.NoError(t, err)
	assert.NotNil(t, result.PreviousState)
	assert.True(t, len(result.PreviousState) > 100, "large state should be serialized")
}
