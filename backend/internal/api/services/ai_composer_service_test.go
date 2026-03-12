package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock AIComposerRepository ---
type MockAIComposerRepo struct {
	mock.Mock
}

func (m *MockAIComposerRepo) CreateSession(ctx context.Context, session *models.AIComposerSession) error {
	return m.Called(ctx, session).Error(0)
}
func (m *MockAIComposerRepo) GetSession(ctx context.Context, id uuid.UUID) (*models.AIComposerSession, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AIComposerSession), args.Error(1)
}
func (m *MockAIComposerRepo) GetSessionsByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.AIComposerSession, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]*models.AIComposerSession), args.Error(1)
}
func (m *MockAIComposerRepo) UpdateSession(ctx context.Context, session *models.AIComposerSession) error {
	return m.Called(ctx, session).Error(0)
}
func (m *MockAIComposerRepo) DeleteSession(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockAIComposerRepo) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockAIComposerRepo) AddMessage(ctx context.Context, message *models.AIComposerMessage) error {
	return m.Called(ctx, message).Error(0)
}
func (m *MockAIComposerRepo) GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerMessage, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]*models.AIComposerMessage), args.Error(1)
}
func (m *MockAIComposerRepo) SaveGeneratedConfig(ctx context.Context, config *models.AIComposerGeneratedConfig) error {
	return m.Called(ctx, config).Error(0)
}
func (m *MockAIComposerRepo) GetGeneratedConfig(ctx context.Context, id uuid.UUID) (*models.AIComposerGeneratedConfig, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AIComposerGeneratedConfig), args.Error(1)
}
func (m *MockAIComposerRepo) GetSessionConfigs(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerGeneratedConfig, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]*models.AIComposerGeneratedConfig), args.Error(1)
}
func (m *MockAIComposerRepo) UpdateConfigValidation(ctx context.Context, id uuid.UUID, status string, errors []string) error {
	return m.Called(ctx, id, status, errors).Error(0)
}
func (m *MockAIComposerRepo) MarkConfigApplied(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockAIComposerRepo) GetTemplates(ctx context.Context, category string) ([]*models.AIComposerTemplate, error) {
	args := m.Called(ctx, category)
	return args.Get(0).([]*models.AIComposerTemplate), args.Error(1)
}
func (m *MockAIComposerRepo) GetTemplate(ctx context.Context, id uuid.UUID) (*models.AIComposerTemplate, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AIComposerTemplate), args.Error(1)
}
func (m *MockAIComposerRepo) IncrementTemplateUsage(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockAIComposerRepo) SaveFeedback(ctx context.Context, feedback *models.AIComposerFeedback) error {
	return m.Called(ctx, feedback).Error(0)
}
func (m *MockAIComposerRepo) GetSessionFeedback(ctx context.Context, sessionID uuid.UUID) ([]*models.AIComposerFeedback, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]*models.AIComposerFeedback), args.Error(1)
}

// --- Mock WebhookEndpointRepository ---
type MockWebhookEndpointRepo struct {
	mock.Mock
}

func (m *MockWebhookEndpointRepo) Create(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	return m.Called(ctx, endpoint).Error(0)
}
func (m *MockWebhookEndpointRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.WebhookEndpoint), args.Error(1)
}
func (m *MockWebhookEndpointRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.WebhookEndpoint, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	return args.Get(0).([]*models.WebhookEndpoint), args.Error(1)
}
func (m *MockWebhookEndpointRepo) GetActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.WebhookEndpoint), args.Error(1)
}
func (m *MockWebhookEndpointRepo) Update(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	return m.Called(ctx, endpoint).Error(0)
}
func (m *MockWebhookEndpointRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockWebhookEndpointRepo) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	return m.Called(ctx, id, active).Error(0)
}
func (m *MockWebhookEndpointRepo) UpdateStatus(ctx context.Context, id uuid.UUID, active bool) error {
	return m.Called(ctx, id, active).Error(0)
}

func (m *MockWebhookEndpointRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

// --- Mock LLMClient ---
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Complete(ctx context.Context, messages []LLMMessage, options LLMOptions) (*LLMResponse, error) {
	args := m.Called(ctx, messages, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LLMResponse), args.Error(1)
}

// --- AI Composer Service Tests ---

func TestAIComposerService_JavaScriptValidator_ValidCode(t *testing.T) {
	t.Parallel()
	v := NewJavaScriptValidator()

	valid, errs := v.Validate(`return { event: payload.type, data: payload };`)
	assert.True(t, valid)
	assert.Empty(t, errs)
}

func TestAIComposerService_JavaScriptValidator_DangerousEval(t *testing.T) {
	t.Parallel()
	v := NewJavaScriptValidator()

	valid, errs := v.Validate(`eval("alert(1)")`)
	assert.False(t, valid)
	assert.NotEmpty(t, errs)
}

func TestAIComposerService_JavaScriptValidator_DangerousRequire(t *testing.T) {
	t.Parallel()
	v := NewJavaScriptValidator()

	valid, errs := v.Validate(`const fs = require("fs")`)
	assert.False(t, valid)
	assert.NotEmpty(t, errs)
}

func TestAIComposerService_JavaScriptValidator_UnbalancedBraces(t *testing.T) {
	t.Parallel()
	v := NewJavaScriptValidator()

	valid, errs := v.Validate(`return { event: payload.type`)
	assert.False(t, valid)
	assert.NotEmpty(t, errs)
}

func TestAIComposerService_JavaScriptValidator_ProtoPattern(t *testing.T) {
	t.Parallel()
	v := NewJavaScriptValidator()

	valid, errs := v.Validate(`obj.__proto__.polluted = true`)
	assert.False(t, valid)
	assert.NotEmpty(t, errs)
}

func TestAIComposerService_JavaScriptValidator_FunctionConstructor(t *testing.T) {
	t.Parallel()
	v := NewJavaScriptValidator()

	valid, errs := v.Validate(`new Function ("return this")()`)
	assert.False(t, valid)
	assert.NotEmpty(t, errs)
}

func TestAIComposerService_GetTemplates(t *testing.T) {
	t.Parallel()
	repo := &MockAIComposerRepo{}
	webhookRepo := &MockWebhookEndpointRepo{}
	llm := &MockLLMClient{}
	logger := utils.NewLogger("test")
	svc := NewAIComposerService(repo, webhookRepo, llm, logger)

	templates := []*models.AIComposerTemplate{
		{ID: uuid.New(), Name: "Slack Webhook", Category: "integrations"},
		{ID: uuid.New(), Name: "Stripe Payment", Category: "integrations"},
	}
	repo.On("GetTemplates", mock.Anything, "integrations").Return(templates, nil)

	result, err := svc.GetTemplates(context.Background(), "integrations")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestAIComposerService_GetTemplates_RepoError(t *testing.T) {
	t.Parallel()
	repo := &MockAIComposerRepo{}
	webhookRepo := &MockWebhookEndpointRepo{}
	llm := &MockLLMClient{}
	logger := utils.NewLogger("test")
	svc := NewAIComposerService(repo, webhookRepo, llm, logger)

	repo.On("GetTemplates", mock.Anything, "unknown").Return(([]*models.AIComposerTemplate)(nil), fmt.Errorf("db error"))

	_, err := svc.GetTemplates(context.Background(), "unknown")
	assert.Error(t, err)
}
