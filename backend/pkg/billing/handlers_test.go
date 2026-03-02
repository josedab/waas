package billing

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupBillingTestRouter(repo *mockRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		c.Set("user_id", "user-1")
		c.Next()
	})

	svc := NewService(repo, nil, nil)
	handler := NewHandler(svc)
	api := router.Group("")
	handler.RegisterRoutes(api)

	return router
}

// =====================
// Subscribe
// =====================

func TestHandler_Subscribe_Valid(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	planID := uuid.New()
	body, _ := json.Marshal(SubscribeRequest{PlanID: planID.String()})

	req := httptest.NewRequest("POST", "/billing/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var sub Subscription
	err := json.Unmarshal(w.Body.Bytes(), &sub)
	require.NoError(t, err)
	assert.Equal(t, "active", sub.Status)
}

func TestHandler_Subscribe_MissingPlanID(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest("POST", "/billing/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Subscribe_InvalidPlanID(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	body, _ := json.Marshal(SubscribeRequest{PlanID: "not-a-uuid"})
	req := httptest.NewRequest("POST", "/billing/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =====================
// GetUsageSummary
// =====================

func TestHandler_GetUsageSummary_Valid(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetUsageSummary", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", mock.Anything).Return(&CostUsageSummary{
		TenantID:      "550e8400-e29b-41d4-a716-446655440000",
		TotalRequests: 1000,
		TotalCost:     50.0,
		Currency:      "USD",
	}, nil)

	req := httptest.NewRequest("GET", "/billing/usage", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var summary CostUsageSummary
	err := json.Unmarshal(w.Body.Bytes(), &summary)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), summary.TotalRequests)
}

func TestHandler_GetUsageSummary_WithPeriod(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetUsageSummary", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", "2026-01").Return(&CostUsageSummary{
		TenantID: "550e8400-e29b-41d4-a716-446655440000",
		Period:   "2026-01",
	}, nil)

	req := httptest.NewRequest("GET", "/billing/usage?period=2026-01", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetUsageSummary_RepoError(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetUsageSummary", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/billing/usage", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =====================
// CreateBudget
// =====================

func TestHandler_CreateBudget_Valid(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("SaveBudget", mock.Anything, mock.AnythingOfType("*billing.BudgetConfig")).Return(nil)

	body, _ := json.Marshal(CreateBudgetRequest{
		Name:   "Monthly Budget",
		Amount: 500,
		Period: PeriodMonthly,
	})

	req := httptest.NewRequest("POST", "/billing/budgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var budget BudgetConfig
	err := json.Unmarshal(w.Body.Bytes(), &budget)
	require.NoError(t, err)
	assert.Equal(t, "Monthly Budget", budget.Name)
	assert.Equal(t, 500.0, budget.Amount)
}

func TestHandler_CreateBudget_InvalidJSON(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	req := httptest.NewRequest("POST", "/billing/budgets", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateBudget_MissingRequired(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	// Missing "name" field (required)
	body, _ := json.Marshal(map[string]interface{}{
		"amount": 500,
		"period": "monthly",
	})

	req := httptest.NewRequest("POST", "/billing/budgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =====================
// ListBudgets
// =====================

func TestHandler_ListBudgets(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("ListBudgets", mock.Anything, "550e8400-e29b-41d4-a716-446655440000").Return([]BudgetConfig{
		{ID: "b1", Name: "Monthly", Amount: 1000},
	}, nil)

	req := httptest.NewRequest("GET", "/billing/budgets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// GetBudget
// =====================

func TestHandler_GetBudget_Found(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetBudget", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", "budget-1").Return(&BudgetConfig{
		ID:     "budget-1",
		Name:   "Monthly",
		Amount: 1000,
	}, nil)

	req := httptest.NewRequest("GET", "/billing/budgets/budget-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetBudget_NotFound(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetBudget", mock.Anything, mock.Anything, "budget-999").Return(nil, errors.New("not found"))

	req := httptest.NewRequest("GET", "/billing/budgets/budget-999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =====================
// UpdateBudget
// =====================

func TestHandler_UpdateBudget(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetBudget", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", "budget-1").Return(&BudgetConfig{
		ID:       "budget-1",
		TenantID: "550e8400-e29b-41d4-a716-446655440000",
		Name:     "Original",
		Amount:   1000,
	}, nil)
	repo.On("SaveBudget", mock.Anything, mock.AnythingOfType("*billing.BudgetConfig")).Return(nil)

	body, _ := json.Marshal(UpdateBudgetRequest{Name: "Updated"})
	req := httptest.NewRequest("PUT", "/billing/budgets/budget-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// DeleteBudget
// =====================

func TestHandler_DeleteBudget(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("DeleteBudget", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", "budget-1").Return(nil)

	req := httptest.NewRequest("DELETE", "/billing/budgets/budget-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// =====================
// ListAlerts
// =====================

func TestHandler_ListAlerts(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("ListAlerts", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", (*AlertStatus)(nil)).Return([]BillingAlert{}, nil)

	req := httptest.NewRequest("GET", "/billing/alerts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListAlerts_WithStatus(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	status := AlertPending
	repo.On("ListAlerts", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", &status).Return([]BillingAlert{}, nil)

	req := httptest.NewRequest("GET", "/billing/alerts?status=pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// AcknowledgeAlert
// =====================

func TestHandler_AcknowledgeAlert(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("UpdateAlertStatus", mock.Anything, "alert-1", AlertAcked, "user-1").Return(nil)

	req := httptest.NewRequest("POST", "/billing/alerts/alert-1/ack", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// HandleStripeWebhook
// =====================

func TestHandler_StripeWebhook_ValidEvent(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	body, _ := json.Marshal(map[string]interface{}{
		"type": "invoice.paid",
		"data": map[string]interface{}{},
	})

	req := httptest.NewRequest("POST", "/billing/webhooks/stripe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_StripeWebhook_InvalidJSON(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	req := httptest.NewRequest("POST", "/billing/webhooks/stripe", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StripeWebhook_UnknownEvent(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	body, _ := json.Marshal(map[string]interface{}{
		"type": "unknown.event",
		"data": map[string]interface{}{},
	})

	req := httptest.NewRequest("POST", "/billing/webhooks/stripe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// GetSubscription
// =====================

func TestHandler_GetSubscription(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	req := httptest.NewRequest("GET", "/billing/subscription", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// ListPricingPlans
// =====================

func TestHandler_ListPricingPlans(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	req := httptest.NewRequest("GET", "/billing/plans", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	plans, ok := result["plans"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 4, len(plans)) // free, starter, pro, enterprise
}

// =====================
// GetDashboard
// =====================

func TestHandler_GetDashboard(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	req := httptest.NewRequest("GET", "/billing/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// GetProjection
// =====================

func TestHandler_GetProjection(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	req := httptest.NewRequest("GET", "/billing/projection?days=14", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// ForecastSpend
// =====================

func TestHandler_ForecastSpend(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetUsageSummary", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", mock.Anything).Return(&CostUsageSummary{
		TotalCost: 100.0,
		Currency:  "USD",
	}, nil)
	repo.On("ListBudgets", mock.Anything, "550e8400-e29b-41d4-a716-446655440000").Return([]BudgetConfig{}, nil)

	req := httptest.NewRequest("GET", "/billing/forecast?days=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// CancelSubscription
// =====================

func TestHandler_CancelSubscription(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	req := httptest.NewRequest("DELETE", "/billing/subscription", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// GetCurrentSpend
// =====================

func TestHandler_GetCurrentSpend(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetCurrentSpend", mock.Anything, "550e8400-e29b-41d4-a716-446655440000").Return(42.5, nil)

	req := httptest.NewRequest("GET", "/billing/spend", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, 42.5, result["current_spend"])
}

// =====================
// GetOptimizations
// =====================

func TestHandler_GetOptimizations(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("ListOptimizations", mock.Anything, "550e8400-e29b-41d4-a716-446655440000").Return([]CostOptimization{}, nil)

	req := httptest.NewRequest("GET", "/billing/optimizations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// ListInvoices
// =====================

func TestHandler_ListInvoices(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("ListInvoices", mock.Anything, "550e8400-e29b-41d4-a716-446655440000").Return([]CostInvoice{}, nil)

	req := httptest.NewRequest("GET", "/billing/invoices", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================
// GetInvoice
// =====================

func TestHandler_GetInvoice_NotFound(t *testing.T) {
	repo := new(mockRepository)
	router := setupBillingTestRouter(repo)

	repo.On("GetInvoice", mock.Anything, "550e8400-e29b-41d4-a716-446655440000", "inv-999").Return(nil, errors.New("not found"))

	req := httptest.NewRequest("GET", "/billing/invoices/inv-999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
