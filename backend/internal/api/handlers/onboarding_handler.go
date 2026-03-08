package handlers

import (
	"errors"
	"net/http"

	"github.com/josedab/waas/pkg/cloud"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
)

// OnboardingHandler handles self-service onboarding HTTP requests
type OnboardingHandler struct {
	service *cloud.OnboardingService
	logger  *utils.Logger
}

// NewOnboardingHandler creates a new onboarding handler
func NewOnboardingHandler(service *cloud.OnboardingService, logger *utils.Logger) *OnboardingHandler {
	return &OnboardingHandler{
		service: service,
		logger:  logger,
	}
}

// StartOnboardingRequest represents the initial signup request
type StartOnboardingRequest struct {
	Email          string            `json:"email" binding:"required,email"`
	ReferralSource string            `json:"referral_source,omitempty"`
	UTMParams      map[string]string `json:"utm_params,omitempty"`
}

// StartOnboardingResponse contains the session ID and verification info
type StartOnboardingResponse struct {
	SessionID         string `json:"session_id"`
	Email             string `json:"email"`
	CurrentStep       string `json:"current_step"`
	VerificationSent  bool   `json:"verification_sent"`
	VerificationToken string `json:"verification_token,omitempty"` // Only in dev mode
}

// VerifyEmailRequest verifies the email with token
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// SetupOrganizationRequest sets up the organization
type SetupOrganizationRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

// SelectPlanRequest selects a subscription plan
type SelectPlanRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// CompletePaymentRequest completes payment setup
type CompletePaymentRequest struct {
	PaymentMethodID string `json:"payment_method_id" binding:"required"`
}

// OnboardingSessionResponse is the standard session response
type OnboardingSessionResponse struct {
	ID             string   `json:"id"`
	Email          string   `json:"email"`
	OrganizationID string   `json:"organization_id,omitempty"`
	CurrentStep    string   `json:"current_step"`
	CompletedSteps []string `json:"completed_steps"`
}

// StartOnboarding begins the onboarding flow
// @Summary Start onboarding
// @Description Begin self-service signup with email verification
// @Tags onboarding
// @Accept json
// @Produce json
// @Param request body StartOnboardingRequest true "Signup request"
// @Success 201 {object} StartOnboardingResponse
// @Failure 400 {object} ErrorResponse
// @Router /onboarding/start [post]
func (h *OnboardingHandler) StartOnboarding(c *gin.Context) {
	var req StartOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	session, token, err := h.service.StartOnboarding(
		c.Request.Context(),
		req.Email,
		ipAddress,
		userAgent,
		req.ReferralSource,
		req.UTMParams,
	)
	if err != nil {
		h.logger.Error("Failed to start onboarding", map[string]interface{}{
			"email": req.Email,
			"error": err.Error(),
		})

		if errors.Is(err, cloud.ErrInvalidEmail) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_EMAIL",
				Message: "Invalid email address format",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "ONBOARDING_FAILED",
			Message: "Failed to start onboarding",
		})
		return
	}

	response := StartOnboardingResponse{
		SessionID:        session.ID,
		Email:            session.Email,
		CurrentStep:      session.CurrentStep,
		VerificationSent: true,
	}

	// In development, return the token for testing
	if c.GetHeader("X-Dev-Mode") == "true" {
		response.VerificationToken = token
	}

	// TODO(#10): Send verification email with token — https://github.com/josedab/waas/issues/10

	h.logger.Info("Onboarding started", map[string]interface{}{
		"session_id": session.ID,
		"email":      req.Email,
	})

	c.JSON(http.StatusCreated, response)
}

// VerifyEmail verifies the email address
// @Summary Verify email
// @Description Verify email address with token from email
// @Tags onboarding
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Param request body VerifyEmailRequest true "Verification request"
// @Success 200 {object} OnboardingSessionResponse
// @Failure 400 {object} ErrorResponse
// @Router /onboarding/{session_id}/verify [post]
func (h *OnboardingHandler) VerifyEmail(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	session, err := h.service.VerifyEmail(c.Request.Context(), sessionID, req.Token)
	if err != nil {
		if errors.Is(err, cloud.ErrInvalidVerification) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_TOKEN",
				Message: "Invalid or expired verification token",
			})
			return
		}
		if errors.Is(err, cloud.ErrOnboardingNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Code:    "SESSION_NOT_FOUND",
				Message: "Onboarding session not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "VERIFICATION_FAILED",
			Message: "Email verification failed",
		})
		return
	}

	c.JSON(http.StatusOK, OnboardingSessionResponse{
		ID:             session.ID,
		Email:          session.Email,
		OrganizationID: session.OrganizationID,
		CurrentStep:    session.CurrentStep,
		CompletedSteps: session.CompletedSteps,
	})
}

// SetupOrganization creates the organization
// @Summary Setup organization
// @Description Create the organization during onboarding
// @Tags onboarding
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Param request body SetupOrganizationRequest true "Organization setup"
// @Success 200 {object} OnboardingSessionResponse
// @Failure 400 {object} ErrorResponse
// @Router /onboarding/{session_id}/organization [post]
func (h *OnboardingHandler) SetupOrganization(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req SetupOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	session, org, err := h.service.SetupOrganization(c.Request.Context(), sessionID, req.Name)
	if err != nil {
		if errors.Is(err, cloud.ErrStepNotComplete) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "STEP_INCOMPLETE",
				Message: "Please complete email verification first",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "SETUP_FAILED",
			Message: "Organization setup failed",
		})
		return
	}

	h.logger.Info("Organization created", map[string]interface{}{
		"session_id": session.ID,
		"org_id":     org.ID,
		"org_slug":   org.Slug,
	})

	c.JSON(http.StatusOK, gin.H{
		"session": OnboardingSessionResponse{
			ID:             session.ID,
			Email:          session.Email,
			OrganizationID: session.OrganizationID,
			CurrentStep:    session.CurrentStep,
			CompletedSteps: session.CompletedSteps,
		},
		"organization": org,
	})
}

// SelectPlan selects a subscription plan
// @Summary Select plan
// @Description Select a subscription plan during onboarding
// @Tags onboarding
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Param request body SelectPlanRequest true "Plan selection"
// @Success 200 {object} OnboardingSessionResponse
// @Failure 400 {object} ErrorResponse
// @Router /onboarding/{session_id}/plan [post]
func (h *OnboardingHandler) SelectPlan(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req SelectPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	session, err := h.service.SelectPlan(c.Request.Context(), sessionID, req.PlanID)
	if err != nil {
		if errors.Is(err, cloud.ErrStepNotComplete) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "STEP_INCOMPLETE",
				Message: "Please complete organization setup first",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "PLAN_SELECTION_FAILED",
			Message: "Plan selection failed",
		})
		return
	}

	c.JSON(http.StatusOK, OnboardingSessionResponse{
		ID:             session.ID,
		Email:          session.Email,
		OrganizationID: session.OrganizationID,
		CurrentStep:    session.CurrentStep,
		CompletedSteps: session.CompletedSteps,
	})
}

// CompletePayment completes payment setup
// @Summary Complete payment
// @Description Add payment method during onboarding
// @Tags onboarding
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Param request body CompletePaymentRequest true "Payment info"
// @Success 200 {object} OnboardingSessionResponse
// @Failure 400 {object} ErrorResponse
// @Router /onboarding/{session_id}/payment [post]
func (h *OnboardingHandler) CompletePayment(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req CompletePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	session, err := h.service.CompletePayment(c.Request.Context(), sessionID, req.PaymentMethodID)
	if err != nil {
		if errors.Is(err, cloud.ErrStepNotComplete) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "STEP_INCOMPLETE",
				Message: "Please complete plan selection first",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "PAYMENT_FAILED",
			Message: "Payment setup failed",
		})
		return
	}

	c.JSON(http.StatusOK, OnboardingSessionResponse{
		ID:             session.ID,
		Email:          session.Email,
		OrganizationID: session.OrganizationID,
		CurrentStep:    session.CurrentStep,
		CompletedSteps: session.CompletedSteps,
	})
}

// CompleteOnboarding marks onboarding as complete
// @Summary Complete onboarding
// @Description Mark onboarding as complete and activate account
// @Tags onboarding
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} OnboardingSessionResponse
// @Failure 400 {object} ErrorResponse
// @Router /onboarding/{session_id}/complete [post]
func (h *OnboardingHandler) CompleteOnboarding(c *gin.Context) {
	sessionID := c.Param("session_id")

	session, err := h.service.CompleteOnboarding(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "COMPLETION_FAILED",
			Message: "Failed to complete onboarding",
		})
		return
	}

	h.logger.Info("Onboarding completed", map[string]interface{}{
		"session_id": session.ID,
		"org_id":     session.OrganizationID,
	})

	c.JSON(http.StatusOK, OnboardingSessionResponse{
		ID:             session.ID,
		Email:          session.Email,
		OrganizationID: session.OrganizationID,
		CurrentStep:    session.CurrentStep,
		CompletedSteps: session.CompletedSteps,
	})
}

// GetSession retrieves the current onboarding session
// @Summary Get onboarding session
// @Description Get current onboarding session status
// @Tags onboarding
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} OnboardingSessionResponse
// @Failure 404 {object} ErrorResponse
// @Router /onboarding/{session_id} [get]
func (h *OnboardingHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	session, err := h.service.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "SESSION_NOT_FOUND",
			Message: "Onboarding session not found",
		})
		return
	}

	c.JSON(http.StatusOK, OnboardingSessionResponse{
		ID:             session.ID,
		Email:          session.Email,
		OrganizationID: session.OrganizationID,
		CurrentStep:    session.CurrentStep,
		CompletedSteps: session.CompletedSteps,
	})
}

// GetPlans returns available subscription plans
// @Summary Get plans
// @Description Get available subscription plans for selection
// @Tags onboarding
// @Produce json
// @Success 200 {array} cloud.Plan
// @Router /onboarding/plans [get]
func (h *OnboardingHandler) GetPlans(c *gin.Context) {
	plans := h.service.GetAvailablePlans()
	c.JSON(http.StatusOK, plans)
}

// RegisterOnboardingRoutes registers onboarding routes (public, no auth required)
func RegisterOnboardingRoutes(r *gin.RouterGroup, h *OnboardingHandler) {
	onboarding := r.Group("/onboarding")
	{
		onboarding.POST("/start", h.StartOnboarding)
		onboarding.GET("/plans", h.GetPlans)
		onboarding.GET("/:session_id", h.GetSession)
		onboarding.POST("/:session_id/verify", h.VerifyEmail)
		onboarding.POST("/:session_id/organization", h.SetupOrganization)
		onboarding.POST("/:session_id/plan", h.SelectPlan)
		onboarding.POST("/:session_id/payment", h.CompletePayment)
		onboarding.POST("/:session_id/complete", h.CompleteOnboarding)
	}
}
