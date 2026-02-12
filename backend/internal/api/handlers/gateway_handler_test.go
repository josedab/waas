package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/gateway"
	"github.com/josedab/waas/pkg/utils"
)

// mockGatewayRepo implements gateway.Repository for testing
type mockGatewayRepo struct {
	providers map[string]*gateway.Provider
	rules     map[string]*gateway.RoutingRule
	webhooks  map[string]*gateway.InboundWebhook
	err       error
}

func newMockGatewayRepo() *mockGatewayRepo {
	return &mockGatewayRepo{
		providers: make(map[string]*gateway.Provider),
		rules:     make(map[string]*gateway.RoutingRule),
		webhooks:  make(map[string]*gateway.InboundWebhook),
	}
}

func (m *mockGatewayRepo) CreateProvider(_ context.Context, p *gateway.Provider) error {
	if m.err != nil {
		return m.err
	}
	if p.ID == "" {
		p.ID = "test-provider-id"
	}
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.providers[p.ID] = p
	return nil
}

func (m *mockGatewayRepo) GetProvider(_ context.Context, tenantID, providerID string) (*gateway.Provider, error) {
	if m.err != nil {
		return nil, m.err
	}
	p, ok := m.providers[providerID]
	if !ok {
		return nil, nil
	}
	if p.TenantID != tenantID {
		return nil, nil
	}
	return p, nil
}

func (m *mockGatewayRepo) ListProviders(_ context.Context, tenantID string, limit, offset int) ([]gateway.Provider, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	var result []gateway.Provider
	for _, p := range m.providers {
		if p.TenantID == tenantID {
			result = append(result, *p)
		}
	}
	return result, len(result), nil
}

func (m *mockGatewayRepo) UpdateProvider(_ context.Context, p *gateway.Provider) error {
	if m.err != nil {
		return m.err
	}
	m.providers[p.ID] = p
	return nil
}

func (m *mockGatewayRepo) DeleteProvider(_ context.Context, tenantID, providerID string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.providers, providerID)
	return nil
}

func (m *mockGatewayRepo) CreateRoutingRule(_ context.Context, rule *gateway.RoutingRule) error {
	if m.err != nil {
		return m.err
	}
	if rule.ID == "" {
		rule.ID = "test-rule-id"
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockGatewayRepo) GetRoutingRule(_ context.Context, tenantID, ruleID string) (*gateway.RoutingRule, error) {
	if m.err != nil {
		return nil, m.err
	}
	r, ok := m.rules[ruleID]
	if !ok {
		return nil, nil
	}
	if r.TenantID != tenantID {
		return nil, nil
	}
	return r, nil
}

func (m *mockGatewayRepo) ListRoutingRules(_ context.Context, tenantID, providerID string) ([]gateway.RoutingRule, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []gateway.RoutingRule
	for _, r := range m.rules {
		if r.TenantID == tenantID && r.ProviderID == providerID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockGatewayRepo) UpdateRoutingRule(_ context.Context, rule *gateway.RoutingRule) error {
	if m.err != nil {
		return m.err
	}
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockGatewayRepo) DeleteRoutingRule(_ context.Context, tenantID, ruleID string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.rules, ruleID)
	return nil
}

func (m *mockGatewayRepo) SaveInboundWebhook(_ context.Context, wh *gateway.InboundWebhook) error {
	if m.err != nil {
		return m.err
	}
	m.webhooks[wh.ID] = wh
	return nil
}

func (m *mockGatewayRepo) GetInboundWebhook(_ context.Context, tenantID, webhookID string) (*gateway.InboundWebhook, error) {
	if m.err != nil {
		return nil, m.err
	}
	wh, ok := m.webhooks[webhookID]
	if !ok {
		return nil, nil
	}
	if wh.TenantID != tenantID {
		return nil, nil
	}
	return wh, nil
}

func (m *mockGatewayRepo) ListInboundWebhooks(_ context.Context, tenantID, providerID string, limit, offset int) ([]gateway.InboundWebhook, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	var result []gateway.InboundWebhook
	for _, wh := range m.webhooks {
		if wh.TenantID == tenantID {
			if providerID == "" || wh.ProviderID == providerID {
				result = append(result, *wh)
			}
		}
	}
	return result, len(result), nil
}

// mockDeliveryPublisher implements gateway.DeliveryPublisher
type mockDeliveryPublisher struct{}

func (m *mockDeliveryPublisher) Publish(_ context.Context, _, _ string, _ []byte, _ map[string]string) (string, error) {
	return "delivery-id", nil
}

func setupGatewayTestRouter(repo *mockGatewayRepo) (*gin.Engine, *GatewayHandler) {
	gin.SetMode(gin.TestMode)
	service := gateway.NewService(repo, &mockDeliveryPublisher{})
	logger := utils.NewLogger("test")
	handler := NewGatewayHandler(service, logger)

	router := gin.New()
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		if tid := c.GetHeader("X-Tenant-ID"); tid != "" {
			c.Set("tenant_id", tid)
		}
		c.Next()
	})
	RegisterGatewayRoutes(api, handler)
	return router, handler
}

// --- CreateProvider tests ---

func TestCreateProvider_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	body := `{"name":"Stripe","type":"stripe","description":"Stripe webhooks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/providers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp gateway.Provider
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Name != "Stripe" {
		t.Errorf("expected name Stripe, got %s", resp.Name)
	}
	if resp.Type != "stripe" {
		t.Errorf("expected type stripe, got %s", resp.Type)
	}
	if !resp.IsActive {
		t.Error("expected provider to be active")
	}
}

func TestCreateProvider_MissingTenantID(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	body := `{"name":"Stripe","type":"stripe"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/providers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateProvider_InvalidJSON(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/providers", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateProvider_RepoError(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.err = errors.New("db error")
	router, _ := setupGatewayTestRouter(repo)

	body := `{"name":"Stripe","type":"stripe"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/providers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ListProviders tests ---

func TestListProviders_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.providers["p1"] = &gateway.Provider{ID: "p1", TenantID: "tenant-1", Name: "Stripe", Type: "stripe", IsActive: true}
	repo.providers["p2"] = &gateway.Provider{ID: "p2", TenantID: "tenant-1", Name: "GitHub", Type: "github", IsActive: true}
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/providers", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []gateway.Provider
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 providers, got %d", len(resp))
	}
}

func TestListProviders_MissingTenantID(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/providers", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetProvider tests ---

func TestGetProvider_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.providers["p1"] = &gateway.Provider{ID: "p1", TenantID: "tenant-1", Name: "Stripe", Type: "stripe"}
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/providers/p1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp gateway.Provider
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.ID != "p1" {
		t.Errorf("expected id p1, got %s", resp.ID)
	}
}

func TestGetProvider_NotFound(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.err = errors.New("not found")
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/providers/nonexistent", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- DeleteProvider tests ---

func TestDeleteProvider_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.providers["p1"] = &gateway.Provider{ID: "p1", TenantID: "tenant-1", Name: "Stripe"}
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/gateway/providers/p1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProvider_RepoError(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.err = errors.New("db error")
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/gateway/providers/p1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- CreateRoutingRule tests ---

func TestCreateRoutingRule_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.providers["p1"] = &gateway.Provider{ID: "p1", TenantID: "tenant-1", Name: "Stripe", Type: "stripe"}
	router, _ := setupGatewayTestRouter(repo)

	body := `{"provider_id":"p1","name":"Route All","destinations":[{"type":"endpoint","endpoint_id":"ep1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp gateway.RoutingRule
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Name != "Route All" {
		t.Errorf("expected name 'Route All', got %s", resp.Name)
	}
}

func TestCreateRoutingRule_MissingTenantID(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	body := `{"provider_id":"p1","name":"Rule","destinations":[{"type":"endpoint","endpoint_id":"ep1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateRoutingRule_InvalidJSON(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/rules", bytes.NewBufferString(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ListRoutingRules tests ---

func TestListRoutingRules_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.rules["r1"] = &gateway.RoutingRule{ID: "r1", TenantID: "tenant-1", ProviderID: "p1", Name: "Rule 1"}
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/rules?provider_id=p1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []gateway.RoutingRule
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 rule, got %d", len(resp))
	}
}

func TestListRoutingRules_MissingProviderID(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/rules", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ListInboundWebhooks tests ---

func TestListInboundWebhooks_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.webhooks["wh1"] = &gateway.InboundWebhook{ID: "wh1", TenantID: "tenant-1", ProviderID: "p1", EventType: "charge.created"}
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/webhooks", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []gateway.InboundWebhook
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(resp))
	}
}

func TestListInboundWebhooks_MissingTenantID(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/webhooks", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetInboundWebhook tests ---

func TestGetInboundWebhook_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.webhooks["wh1"] = &gateway.InboundWebhook{ID: "wh1", TenantID: "tenant-1", ProviderID: "p1"}
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/webhooks/wh1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetInboundWebhook_NotFound(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.err = errors.New("not found")
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/webhooks/nonexistent", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- DeleteRoutingRule tests ---

func TestDeleteRoutingRule_Success(t *testing.T) {
	repo := newMockGatewayRepo()
	repo.rules["r1"] = &gateway.RoutingRule{ID: "r1", TenantID: "tenant-1"}
	router, _ := setupGatewayTestRouter(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/gateway/rules/r1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ReceiveWebhook tests ---

func TestReceiveWebhook_MissingProviderID(t *testing.T) {
	repo := newMockGatewayRepo()
	router, _ := setupGatewayTestRouter(repo)

	// POST to /gateway/receive/ without a provider_id segment → route won't match → 404
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/receive/", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Gin returns 404 when the :provider_id param is missing from the path
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Fatalf("expected 404 or 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReceiveWebhook_ProviderNotFound(t *testing.T) {
	repo := newMockGatewayRepo()
	// No providers in repo → service.ProcessInboundWebhook will return "provider not found"
	router, _ := setupGatewayTestRouter(repo)

	body := `{"event":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/receive/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}
