package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func setupOnboardingTest() (*OnboardingHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewOnboardingHandler(nil, logger)
	router := gin.New()
	return handler, router
}

func TestOnboardingHandler_StartOnboarding_InvalidBody(t *testing.T) {
	handler, router := setupOnboardingTest()
	router.POST("/onboarding", handler.StartOnboarding)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/onboarding", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOnboardingHandler_VerifyEmail_InvalidBody(t *testing.T) {
	handler, router := setupOnboardingTest()
	router.POST("/onboarding/verify-email", handler.VerifyEmail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/onboarding/verify-email", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOnboardingHandler_SetupOrganization_InvalidBody(t *testing.T) {
	handler, router := setupOnboardingTest()
	router.POST("/onboarding/organization", handler.SetupOrganization)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/onboarding/organization", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOnboardingHandler_SelectPlan_InvalidBody(t *testing.T) {
	handler, router := setupOnboardingTest()
	router.POST("/onboarding/plan", handler.SelectPlan)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/onboarding/plan", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TODO: Add tests with mock OnboardingService for GetSession, GetPlans
