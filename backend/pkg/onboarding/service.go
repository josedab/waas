package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides onboarding business logic.
type Service struct {
	repo Repository
}

// NewService creates a new onboarding service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// StartOnboarding begins a new onboarding session for a tenant.
func (s *Service) StartOnboarding(ctx context.Context, req *StartOnboardingRequest) (*OnboardingProgress, error) {
	now := time.Now()
	session := &OnboardingSession{
		ID:             uuid.New().String(),
		TenantID:       req.TenantID,
		CurrentStep:    StepCreateTenant,
		CompletedSteps: []string{},
		Metadata:       req.Metadata,
		IsComplete:     false,
		StartedAt:      now,
		UpdatedAt:      now,
	}

	if s.repo != nil {
		if err := s.repo.CreateSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to start onboarding: %w", err)
		}
	}

	return s.buildProgress(session), nil
}

// GetProgress retrieves the current onboarding progress for a tenant.
func (s *Service) GetProgress(ctx context.Context, tenantID string) (*OnboardingProgress, error) {
	if s.repo == nil {
		return &OnboardingProgress{
			Session: &OnboardingSession{TenantID: tenantID, CurrentStep: StepCreateTenant},
			Steps:   s.buildStepDetails(StepCreateTenant, nil),
		}, nil
	}

	session, err := s.repo.GetSession(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return s.buildProgress(session), nil
}

// CompleteStep marks an onboarding step as completed and advances to next.
func (s *Service) CompleteStep(ctx context.Context, tenantID string, req *CompleteStepRequest) (*OnboardingProgress, error) {
	if !isValidStep(req.StepID) {
		return nil, fmt.Errorf("invalid step %q", req.StepID)
	}

	var session *OnboardingSession

	if s.repo != nil {
		var err error
		session, err = s.repo.GetSession(ctx, tenantID)
		if err != nil {
			return nil, err
		}
	} else {
		session = &OnboardingSession{
			ID:             uuid.New().String(),
			TenantID:       tenantID,
			CurrentStep:    req.StepID,
			CompletedSteps: []string{},
			StartedAt:      time.Now(),
		}
	}

	if session.IsComplete {
		return s.buildProgress(session), nil
	}

	// Mark step as completed
	if !contains(session.CompletedSteps, req.StepID) {
		session.CompletedSteps = append(session.CompletedSteps, req.StepID)
	}

	// Advance to next step
	nextStep := getNextStep(req.StepID)
	if nextStep != "" {
		session.CurrentStep = nextStep
	} else {
		session.IsComplete = true
		now := time.Now()
		session.CompletedAt = &now
	}
	session.UpdatedAt = time.Now()

	if s.repo != nil {
		_ = s.repo.UpdateSession(ctx, session)
	}

	return s.buildProgress(session), nil
}

// SkipStep skips a step and advances to the next one.
func (s *Service) SkipStep(ctx context.Context, tenantID, stepID string) (*OnboardingProgress, error) {
	return s.CompleteStep(ctx, tenantID, &CompleteStepRequest{StepID: stepID})
}

// GetSnippets returns language-specific code snippets with tenant details.
func (s *Service) GetSnippets(ctx context.Context, req *GetSnippetsRequest) ([]CodeSnippet, error) {
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = "<YOUR_TENANT_ID>"
	}
	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = "<YOUR_API_KEY>"
	}
	endpoint := req.EndpointURL
	if endpoint == "" {
		endpoint = "https://your-app.com/webhooks"
	}

	switch req.Language {
	case "typescript", "javascript":
		return []CodeSnippet{
			{Language: "typescript", Title: "Install SDK", Code: "npm install @waas/sdk"},
			{Language: "typescript", Title: "Initialize Client", Code: fmt.Sprintf(
				"import { WaaSClient } from '@waas/sdk';\n\nconst client = new WaaSClient({\n  apiKey: '%s',\n  tenantId: '%s'\n});", apiKey, tenantID)},
			{Language: "typescript", Title: "Create Endpoint", Code: fmt.Sprintf(
				"const endpoint = await client.endpoints.create({\n  url: '%s',\n  eventTypes: ['order.created']\n});", endpoint)},
			{Language: "typescript", Title: "Send Test Webhook", Code: "await client.webhooks.send({\n  eventType: 'order.created',\n  payload: { order_id: '123', amount: 99.99 }\n});"},
		}, nil

	case "python":
		return []CodeSnippet{
			{Language: "python", Title: "Install SDK", Code: "pip install waas-sdk"},
			{Language: "python", Title: "Initialize Client", Code: fmt.Sprintf(
				"from waas import WaaSClient\n\nclient = WaaSClient(\n    api_key='%s',\n    tenant_id='%s'\n)", apiKey, tenantID)},
			{Language: "python", Title: "Create Endpoint", Code: fmt.Sprintf(
				"endpoint = client.endpoints.create(\n    url='%s',\n    event_types=['order.created']\n)", endpoint)},
			{Language: "python", Title: "Send Test Webhook", Code: "client.webhooks.send(\n    event_type='order.created',\n    payload={'order_id': '123', 'amount': 99.99}\n)"},
		}, nil

	case "go":
		return []CodeSnippet{
			{Language: "go", Title: "Install SDK", Code: "go get github.com/josedab/waas/sdk/go"},
			{Language: "go", Title: "Initialize Client", Code: fmt.Sprintf(
				"client := waas.NewClient(\n\twaas.WithAPIKey(\"%s\"),\n\twaas.WithTenantID(\"%s\"),\n)", apiKey, tenantID)},
			{Language: "go", Title: "Send Test Webhook", Code: "client.Webhooks.Send(ctx, &waas.SendRequest{\n\tEventType: \"order.created\",\n\tPayload:   map[string]interface{}{\"order_id\": \"123\"},\n})"},
		}, nil

	default:
		return []CodeSnippet{}, nil
	}
}

// GetAnalytics returns onboarding funnel analytics.
func (s *Service) GetAnalytics(ctx context.Context) (*OnboardingAnalytics, error) {
	if s.repo == nil {
		return &OnboardingAnalytics{StepDropoff: map[string]int64{}}, nil
	}
	return s.repo.GetAnalytics(ctx)
}

func (s *Service) buildProgress(session *OnboardingSession) *OnboardingProgress {
	steps := s.buildStepDetails(session.CurrentStep, session.CompletedSteps)
	completedCount := len(session.CompletedSteps)
	progressPercent := (completedCount * 100) / len(AllSteps)

	elapsed := time.Since(session.StartedAt).Round(time.Second).String()

	return &OnboardingProgress{
		Session:         session,
		Steps:           steps,
		ProgressPercent: progressPercent,
		TimeElapsed:     elapsed,
	}
}

func (s *Service) buildStepDetails(currentStep string, completedSteps []string) []StepDetail {
	stepNames := map[string]string{
		StepCreateTenant:   "Create Tenant",
		StepConfigEndpoint: "Configure Endpoint",
		StepSendTest:       "Send Test Webhook",
		StepVerifyDelivery: "Verify Delivery",
		StepInstallSDK:     "Install SDK",
	}

	stepDescs := map[string]string{
		StepCreateTenant:   "Create your first tenant and get your API key",
		StepConfigEndpoint: "Configure a webhook endpoint URL to receive events",
		StepSendTest:       "Send a test webhook to verify your setup",
		StepVerifyDelivery: "Verify the webhook was delivered successfully",
		StepInstallSDK:     "Install the WaaS SDK in your preferred language",
	}

	var details []StepDetail
	for i, step := range AllSteps {
		details = append(details, StepDetail{
			ID:          step,
			Name:        stepNames[step],
			Description: stepDescs[step],
			Order:       i + 1,
			IsCompleted: contains(completedSteps, step),
			IsCurrent:   step == currentStep,
		})
	}
	return details
}

func isValidStep(step string) bool {
	for _, s := range AllSteps {
		if s == step {
			return true
		}
	}
	return false
}

func getNextStep(currentStep string) string {
	for i, step := range AllSteps {
		if step == currentStep && i+1 < len(AllSteps) {
			return AllSteps[i+1]
		}
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
