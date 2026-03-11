package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/cloud"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inMemoryCloudRepo implements cloud.Repository for unit testing.
type inMemoryCloudRepo struct {
	subscriptions map[string]*cloud.Subscription
	customers     map[string]*cloud.Customer
	members       map[string]*cloud.TeamMember
	invoices      map[string]*cloud.Invoice
	payments      map[string]*cloud.PaymentMethod
	auditLogs     []*cloud.AuditLog
	usage         map[string]*cloud.UsageRecord
}

func newInMemoryCloudRepo() *inMemoryCloudRepo {
	return &inMemoryCloudRepo{
		subscriptions: make(map[string]*cloud.Subscription),
		customers:     make(map[string]*cloud.Customer),
		members:       make(map[string]*cloud.TeamMember),
		invoices:      make(map[string]*cloud.Invoice),
		payments:      make(map[string]*cloud.PaymentMethod),
		usage:         make(map[string]*cloud.UsageRecord),
	}
}

func (r *inMemoryCloudRepo) CreateSubscription(_ context.Context, sub *cloud.Subscription) error {
	r.subscriptions[sub.ID] = sub
	return nil
}
func (r *inMemoryCloudRepo) GetSubscription(_ context.Context, id string) (*cloud.Subscription, error) {
	if s, ok := r.subscriptions[id]; ok {
		return s, nil
	}
	return nil, cloud.ErrSubscriptionNotFound
}
func (r *inMemoryCloudRepo) GetSubscriptionByTenant(_ context.Context, tenantID string) (*cloud.Subscription, error) {
	for _, s := range r.subscriptions {
		if s.TenantID == tenantID {
			return s, nil
		}
	}
	return nil, cloud.ErrSubscriptionNotFound
}
func (r *inMemoryCloudRepo) UpdateSubscription(_ context.Context, sub *cloud.Subscription) error {
	r.subscriptions[sub.ID] = sub
	return nil
}
func (r *inMemoryCloudRepo) ListSubscriptions(_ context.Context, _ cloud.SubscriptionStatus, _ int) ([]*cloud.Subscription, error) {
	return nil, nil
}
func (r *inMemoryCloudRepo) GetUsage(_ context.Context, tenantID, period string) (*cloud.UsageRecord, error) {
	key := tenantID + ":" + period
	if u, ok := r.usage[key]; ok {
		return u, nil
	}
	return nil, cloud.ErrSubscriptionNotFound
}
func (r *inMemoryCloudRepo) IncrementUsage(_ context.Context, _ string, _ string, _ int64) error {
	return nil
}
func (r *inMemoryCloudRepo) ListUsage(_ context.Context, _ string, _ int) ([]*cloud.UsageRecord, error) {
	return nil, nil
}
func (r *inMemoryCloudRepo) CreateInvoice(_ context.Context, inv *cloud.Invoice) error {
	r.invoices[inv.ID] = inv
	return nil
}
func (r *inMemoryCloudRepo) GetInvoice(_ context.Context, id string) (*cloud.Invoice, error) {
	if i, ok := r.invoices[id]; ok {
		return i, nil
	}
	return nil, cloud.ErrSubscriptionNotFound
}
func (r *inMemoryCloudRepo) ListInvoices(_ context.Context, tenantID string, _ int) ([]*cloud.Invoice, error) {
	var result []*cloud.Invoice
	for _, i := range r.invoices {
		if i.TenantID == tenantID {
			result = append(result, i)
		}
	}
	return result, nil
}
func (r *inMemoryCloudRepo) UpdateInvoice(_ context.Context, inv *cloud.Invoice) error {
	r.invoices[inv.ID] = inv
	return nil
}
func (r *inMemoryCloudRepo) CreateCustomer(_ context.Context, c *cloud.Customer) error {
	r.customers[c.TenantID] = c
	return nil
}
func (r *inMemoryCloudRepo) GetCustomer(_ context.Context, tenantID string) (*cloud.Customer, error) {
	if c, ok := r.customers[tenantID]; ok {
		return c, nil
	}
	return nil, cloud.ErrSubscriptionNotFound
}
func (r *inMemoryCloudRepo) UpdateCustomer(_ context.Context, c *cloud.Customer) error {
	r.customers[c.TenantID] = c
	return nil
}
func (r *inMemoryCloudRepo) CreatePaymentMethod(_ context.Context, pm *cloud.PaymentMethod) error {
	r.payments[pm.ID] = pm
	return nil
}
func (r *inMemoryCloudRepo) ListPaymentMethods(_ context.Context, tenantID string) ([]*cloud.PaymentMethod, error) {
	var result []*cloud.PaymentMethod
	for _, pm := range r.payments {
		if pm.TenantID == tenantID {
			result = append(result, pm)
		}
	}
	return result, nil
}
func (r *inMemoryCloudRepo) DeletePaymentMethod(_ context.Context, id string) error {
	delete(r.payments, id)
	return nil
}
func (r *inMemoryCloudRepo) SetDefaultPaymentMethod(_ context.Context, _, _ string) error {
	return nil
}
func (r *inMemoryCloudRepo) CreateTeamMember(_ context.Context, m *cloud.TeamMember) error {
	r.members[m.ID] = m
	return nil
}
func (r *inMemoryCloudRepo) GetTeamMember(_ context.Context, id string) (*cloud.TeamMember, error) {
	if m, ok := r.members[id]; ok {
		return m, nil
	}
	return nil, cloud.ErrSubscriptionNotFound
}
func (r *inMemoryCloudRepo) ListTeamMembers(_ context.Context, tenantID string) ([]*cloud.TeamMember, error) {
	var result []*cloud.TeamMember
	for _, m := range r.members {
		if m.TenantID == tenantID {
			result = append(result, m)
		}
	}
	return result, nil
}
func (r *inMemoryCloudRepo) UpdateTeamMember(_ context.Context, m *cloud.TeamMember) error {
	r.members[m.ID] = m
	return nil
}
func (r *inMemoryCloudRepo) DeleteTeamMember(_ context.Context, id string) error {
	delete(r.members, id)
	return nil
}
func (r *inMemoryCloudRepo) CreateAuditLog(_ context.Context, log *cloud.AuditLog) error {
	r.auditLogs = append(r.auditLogs, log)
	return nil
}
func (r *inMemoryCloudRepo) ListAuditLogs(_ context.Context, _ string, _, _ int) ([]*cloud.AuditLog, error) {
	return r.auditLogs, nil
}

func setupCloudMutationTest(repo *inMemoryCloudRepo) (*CloudHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")

	billingService := cloud.NewBillingService(repo, nil)
	teamService := cloud.NewTeamService(repo)
	auditService := cloud.NewAuditService(repo)

	handler := NewCloudHandler(billingService, teamService, auditService, repo, logger)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant-123")
		c.Set("user_id", "test-user-456")
		c.Next()
	})
	RegisterCloudRoutes(router.Group("/api/v1"), handler)

	return handler, router
}

func TestCloudHandler_CreateSubscription_Success(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	body, _ := json.Marshal(CreateSubscriptionRequest{
		PlanID:       "free",
		BillingCycle: "monthly",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/billing/subscription", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCloudHandler_CreateSubscription_InvalidPlan(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	body, _ := json.Marshal(CreateSubscriptionRequest{
		PlanID:       "nonexistent-plan",
		BillingCycle: "monthly",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/billing/subscription", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCloudHandler_CreateSubscription_InvalidBody(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/billing/subscription", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCloudHandler_CreateSubscription_Unauthorized(t *testing.T) {
	repo := newInMemoryCloudRepo()
	logger := utils.NewLogger("test")
	billingService := cloud.NewBillingService(repo, nil)
	handler := NewCloudHandler(billingService, nil, nil, repo, logger)

	router := gin.New()
	// No tenant_id set
	router.POST("/sub", handler.CreateSubscription)

	body, _ := json.Marshal(CreateSubscriptionRequest{PlanID: "free", BillingCycle: "monthly"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/sub", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCloudHandler_ChangePlan_Success(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	// First create a subscription
	repo.subscriptions["sub-1"] = &cloud.Subscription{
		ID:       "sub-1",
		TenantID: "test-tenant-123",
		PlanID:   "free",
		Status:   cloud.SubscriptionStatusActive,
	}

	body, _ := json.Marshal(ChangePlanRequest{PlanID: "starter"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/billing/subscription/plan", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCloudHandler_CancelSubscription_Success(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	repo.subscriptions["sub-1"] = &cloud.Subscription{
		ID:       "sub-1",
		TenantID: "test-tenant-123",
		PlanID:   "starter",
		Status:   cloud.SubscriptionStatusActive,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/billing/subscription", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCloudHandler_CancelSubscription_Nonexistent(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/billing/subscription", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCloudHandler_InviteMember_Success(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	body, _ := json.Marshal(InviteMemberRequest{
		Email: "newmember@example.com",
		Name:  "New Member",
		Role:  "member",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/team/members", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Len(t, repo.members, 1)
}

func TestCloudHandler_RemoveMember_Success(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	repo.members["member-1"] = &cloud.TeamMember{
		ID:       "member-1",
		TenantID: "test-tenant-123",
		Email:    "remove@example.com",
		Name:     "Remove Me",
		Role:     "member",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/team/members/member-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Len(t, repo.members, 0)
}

func TestCloudHandler_UpdateCustomer_Success(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	repo.customers["test-tenant-123"] = &cloud.Customer{
		ID:       "cust-1",
		TenantID: "test-tenant-123",
		Email:    "old@example.com",
		Name:     "Old Name",
	}

	body, _ := json.Marshal(UpdateCustomerRequest{
		Name:  "New Name",
		Email: "new@example.com",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/billing/customer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "New Name", resp["name"])
	assert.Equal(t, "new@example.com", resp["email"])
}

func TestCloudHandler_UpdateCustomer_PartialUpdate(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	repo.customers["test-tenant-123"] = &cloud.Customer{
		ID:       "cust-1",
		TenantID: "test-tenant-123",
		Email:    "keep@example.com",
		Name:     "Keep Name",
		Company:  "Original Co",
	}

	body, _ := json.Marshal(UpdateCustomerRequest{Company: "New Co"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/billing/customer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "keep@example.com", repo.customers["test-tenant-123"].Email)
	assert.Equal(t, "New Co", repo.customers["test-tenant-123"].Company)
}

func TestCloudHandler_DeletePaymentMethod_Success(t *testing.T) {
	repo := newInMemoryCloudRepo()
	_, router := setupCloudMutationTest(repo)

	repo.payments["pm-1"] = &cloud.PaymentMethod{
		ID:       "pm-1",
		TenantID: "test-tenant-123",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/billing/payment-methods/pm-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Len(t, repo.payments, 0)
}
