package cloud

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidEmail          = errors.New("invalid email address")
	ErrEmailAlreadyExists    = errors.New("email already registered")
	ErrInvalidVerification   = errors.New("invalid or expired verification token")
	ErrOnboardingNotFound    = errors.New("onboarding session not found")
	ErrStepNotComplete       = errors.New("previous steps not completed")
	ErrOrganizationExists    = errors.New("organization slug already taken")
)

// OnboardingService handles self-service signup flow
type OnboardingService struct {
	repo         *OnboardingRepository
	billingRepo  Repository
	stripeClient StripeClient
}

// OnboardingSession represents a signup flow session
type OnboardingSession struct {
	ID                    string            `json:"id"`
	Email                 string            `json:"email"`
	OrganizationID        string            `json:"organization_id,omitempty"`
	CurrentStep           string            `json:"current_step"`
	CompletedSteps        []string          `json:"completed_steps"`
	VerificationToken     string            `json:"-"`
	VerificationExpiresAt *time.Time        `json:"-"`
	FormData              map[string]string `json:"form_data,omitempty"`
	IPAddress             string            `json:"-"`
	UserAgent             string            `json:"-"`
	ReferralSource        string            `json:"referral_source,omitempty"`
	UTMParams             map[string]string `json:"utm_params,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
	CompletedAt           *time.Time        `json:"completed_at,omitempty"`
}

// Organization represents a multi-tenant organization
type Organization struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Slug             string            `json:"slug"`
	BillingEmail     string            `json:"billing_email"`
	StripeCustomerID string            `json:"-"`
	PlanID           string            `json:"plan_id"`
	Status           string            `json:"status"`
	TrialEndsAt      *time.Time        `json:"trial_ends_at,omitempty"`
	Settings         map[string]string `json:"settings,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// OnboardingStep constants
const (
	StepEmailVerification = "email_verification"
	StepOrganizationSetup = "organization_setup"
	StepPlanSelection     = "plan_selection"
	StepPaymentSetup      = "payment_setup"
	StepFirstEndpoint     = "first_endpoint"
	StepCompleted         = "completed"
)

// Organization status constants
const (
	OrgStatusActive    = "active"
	OrgStatusTrial     = "trial"
	OrgStatusSuspended = "suspended"
	OrgStatusCanceled  = "canceled"
)

// NewOnboardingService creates a new onboarding service
func NewOnboardingService(repo *OnboardingRepository, billingRepo Repository, stripeClient StripeClient) *OnboardingService {
	return &OnboardingService{
		repo:         repo,
		billingRepo:  billingRepo,
		stripeClient: stripeClient,
	}
}

// StartOnboarding begins the onboarding flow with email verification
func (s *OnboardingService) StartOnboarding(ctx context.Context, email, ipAddress, userAgent, referralSource string, utmParams map[string]string) (*OnboardingSession, string, error) {
	if !isValidEmail(email) {
		return nil, "", ErrInvalidEmail
	}

	// Check if email already has a session
	existing, _ := s.repo.GetSessionByEmail(ctx, email)
	if existing != nil && existing.CurrentStep != StepEmailVerification {
		return existing, "", nil // Resume existing session
	}

	// Generate verification token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	session := &OnboardingSession{
		ID:                    uuid.New().String(),
		Email:                 email,
		CurrentStep:           StepEmailVerification,
		CompletedSteps:        []string{},
		VerificationToken:     hashToken(token),
		VerificationExpiresAt: &expiresAt,
		IPAddress:             ipAddress,
		UserAgent:             userAgent,
		ReferralSource:        referralSource,
		UTMParams:             utmParams,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, "", err
	}

	return session, token, nil
}

// VerifyEmail verifies the email with the token
func (s *OnboardingService) VerifyEmail(ctx context.Context, sessionID, token string) (*OnboardingSession, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrOnboardingNotFound
	}

	if session.CurrentStep != StepEmailVerification {
		return session, nil // Already verified
	}

	hashedToken := hashToken(token)
	if session.VerificationToken != hashedToken {
		return nil, ErrInvalidVerification
	}

	if session.VerificationExpiresAt != nil && time.Now().After(*session.VerificationExpiresAt) {
		return nil, ErrInvalidVerification
	}

	session.CompletedSteps = append(session.CompletedSteps, StepEmailVerification)
	session.CurrentStep = StepOrganizationSetup
	session.UpdatedAt = time.Now()

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// SetupOrganization creates the organization during onboarding
func (s *OnboardingService) SetupOrganization(ctx context.Context, sessionID, name string) (*OnboardingSession, *Organization, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, ErrOnboardingNotFound
	}

	if !containsStep(session.CompletedSteps, StepEmailVerification) {
		return nil, nil, ErrStepNotComplete
	}

	slug := generateSlug(name)

	// Check slug uniqueness
	existing, _ := s.repo.GetOrganizationBySlug(ctx, slug)
	if existing != nil {
		slug = slug + "-" + generateShortID()
	}

	trialEnd := time.Now().AddDate(0, 0, 14) // 14-day trial

	org := &Organization{
		ID:           uuid.New().String(),
		Name:         name,
		Slug:         slug,
		BillingEmail: session.Email,
		PlanID:       "free", // Start with free plan
		Status:       OrgStatusTrial,
		TrialEndsAt:  &trialEnd,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Create Stripe customer
	if s.stripeClient != nil {
		customerID, err := s.stripeClient.CreateCustomer(ctx, session.Email, name)
		if err == nil {
			org.StripeCustomerID = customerID
		}
	}

	if err := s.repo.CreateOrganization(ctx, org); err != nil {
		return nil, nil, err
	}

	session.OrganizationID = org.ID
	session.CompletedSteps = append(session.CompletedSteps, StepOrganizationSetup)
	session.CurrentStep = StepPlanSelection
	session.UpdatedAt = time.Now()

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, nil, err
	}

	return session, org, nil
}

// SelectPlan selects a subscription plan
func (s *OnboardingService) SelectPlan(ctx context.Context, sessionID, planID string) (*OnboardingSession, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrOnboardingNotFound
	}

	if !containsStep(session.CompletedSteps, StepOrganizationSetup) {
		return nil, ErrStepNotComplete
	}

	// Update organization with selected plan
	org, err := s.repo.GetOrganization(ctx, session.OrganizationID)
	if err != nil {
		return nil, err
	}

	org.PlanID = planID
	org.UpdatedAt = time.Now()

	if err := s.repo.UpdateOrganization(ctx, org); err != nil {
		return nil, err
	}

	session.CompletedSteps = append(session.CompletedSteps, StepPlanSelection)

	// If free plan, skip payment setup
	if planID == "free" {
		session.CurrentStep = StepFirstEndpoint
	} else {
		session.CurrentStep = StepPaymentSetup
	}
	session.UpdatedAt = time.Now()

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// CompletePayment completes the payment setup step
func (s *OnboardingService) CompletePayment(ctx context.Context, sessionID, paymentMethodID string) (*OnboardingSession, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrOnboardingNotFound
	}

	if !containsStep(session.CompletedSteps, StepPlanSelection) {
		return nil, ErrStepNotComplete
	}

	org, err := s.repo.GetOrganization(ctx, session.OrganizationID)
	if err != nil {
		return nil, err
	}

	// Attach payment method in Stripe
	if s.stripeClient != nil && org.StripeCustomerID != "" {
		if err := s.stripeClient.AttachPaymentMethod(ctx, org.StripeCustomerID, paymentMethodID); err != nil {
			return nil, fmt.Errorf("failed to attach payment method: %w", err)
		}
	}

	session.CompletedSteps = append(session.CompletedSteps, StepPaymentSetup)
	session.CurrentStep = StepFirstEndpoint
	session.UpdatedAt = time.Now()

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// CompleteOnboarding marks onboarding as complete
func (s *OnboardingService) CompleteOnboarding(ctx context.Context, sessionID string) (*OnboardingSession, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrOnboardingNotFound
	}

	now := time.Now()
	session.CompletedSteps = append(session.CompletedSteps, StepFirstEndpoint)
	session.CurrentStep = StepCompleted
	session.CompletedAt = &now
	session.UpdatedAt = now

	// Activate organization
	org, err := s.repo.GetOrganization(ctx, session.OrganizationID)
	if err == nil {
		org.Status = OrgStatusActive
		org.UpdatedAt = now
		s.repo.UpdateOrganization(ctx, org)
	}

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves an onboarding session
func (s *OnboardingService) GetSession(ctx context.Context, sessionID string) (*OnboardingSession, error) {
	return s.repo.GetSession(ctx, sessionID)
}

// GetAvailablePlans returns available subscription plans
func (s *OnboardingService) GetAvailablePlans() []*Plan {
	return AvailablePlans
}

// Helper functions

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	// Simple hash for demo - use bcrypt in production
	return fmt.Sprintf("%x", token)
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	slug = re.ReplaceAllString(slug, "")
	return slug
}

func generateShortID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func containsStep(steps []string, step string) bool {
	for _, s := range steps {
		if s == step {
			return true
		}
	}
	return false
}

// OnboardingRepository handles onboarding data persistence
type OnboardingRepository struct {
	sessions      map[string]*OnboardingSession
	organizations map[string]*Organization
}

// NewOnboardingRepository creates a new in-memory onboarding repository
func NewOnboardingRepository() *OnboardingRepository {
	return &OnboardingRepository{
		sessions:      make(map[string]*OnboardingSession),
		organizations: make(map[string]*Organization),
	}
}

func (r *OnboardingRepository) CreateSession(ctx context.Context, session *OnboardingSession) error {
	r.sessions[session.ID] = session
	return nil
}

func (r *OnboardingRepository) GetSession(ctx context.Context, id string) (*OnboardingSession, error) {
	if session, ok := r.sessions[id]; ok {
		return session, nil
	}
	return nil, ErrOnboardingNotFound
}

func (r *OnboardingRepository) GetSessionByEmail(ctx context.Context, email string) (*OnboardingSession, error) {
	for _, session := range r.sessions {
		if session.Email == email {
			return session, nil
		}
	}
	return nil, ErrOnboardingNotFound
}

func (r *OnboardingRepository) UpdateSession(ctx context.Context, session *OnboardingSession) error {
	r.sessions[session.ID] = session
	return nil
}

func (r *OnboardingRepository) CreateOrganization(ctx context.Context, org *Organization) error {
	r.organizations[org.ID] = org
	return nil
}

func (r *OnboardingRepository) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	if org, ok := r.organizations[id]; ok {
		return org, nil
	}
	return nil, errors.New("organization not found")
}

func (r *OnboardingRepository) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	for _, org := range r.organizations {
		if org.Slug == slug {
			return org, nil
		}
	}
	return nil, errors.New("organization not found")
}

func (r *OnboardingRepository) UpdateOrganization(ctx context.Context, org *Organization) error {
	r.organizations[org.ID] = org
	return nil
}
