package onboarding

import "context"

// Repository defines the data access interface for onboarding.
type Repository interface {
	CreateSession(ctx context.Context, session *OnboardingSession) error
	GetSession(ctx context.Context, tenantID string) (*OnboardingSession, error)
	UpdateSession(ctx context.Context, session *OnboardingSession) error
	GetAnalytics(ctx context.Context) (*OnboardingAnalytics, error)
}
