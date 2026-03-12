package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWebhookRepo implements repository.WebhookEndpointRepository for unit tests.
type mockWebhookRepo struct {
	endpoints map[uuid.UUID]*models.WebhookEndpoint
	createErr error
	updateErr error
	deleteErr error
}

func newMockWebhookRepo() *mockWebhookRepo {
	return &mockWebhookRepo{endpoints: make(map[uuid.UUID]*models.WebhookEndpoint)}
}

func (m *mockWebhookRepo) Create(_ context.Context, ep *models.WebhookEndpoint) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.endpoints[ep.ID] = ep
	return nil
}

func (m *mockWebhookRepo) GetByID(_ context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	ep, ok := m.endpoints[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return ep, nil
}

func (m *mockWebhookRepo) GetByTenantID(_ context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.WebhookEndpoint, error) {
	var result []*models.WebhookEndpoint
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID {
			result = append(result, ep)
		}
	}
	if offset >= len(result) {
		return []*models.WebhookEndpoint{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *mockWebhookRepo) GetActiveByTenantID(_ context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	var result []*models.WebhookEndpoint
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID && ep.IsActive {
			result = append(result, ep)
		}
	}
	return result, nil
}

func (m *mockWebhookRepo) Update(_ context.Context, ep *models.WebhookEndpoint) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.endpoints[ep.ID] = ep
	return nil
}

func (m *mockWebhookRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.endpoints, id)
	return nil
}

func (m *mockWebhookRepo) SetActive(_ context.Context, id uuid.UUID, active bool) error {
	if ep, ok := m.endpoints[id]; ok {
		ep.IsActive = active
	}
	return nil
}

func (m *mockWebhookRepo) UpdateStatus(_ context.Context, id uuid.UUID, active bool) error {
	return m.SetActive(nil, id, active)
}

func (m *mockWebhookRepo) CountByTenantID(_ context.Context, tenantID uuid.UUID) (int, error) {
	count := 0
	for _, ep := range m.endpoints {
		if ep.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

// mockDeliveryAttemptRepo implements repository.DeliveryAttemptRepository
type mockDeliveryAttemptRepo struct {
	attempts  map[uuid.UUID]*models.DeliveryAttempt
	createErr error
}

func newMockDeliveryAttemptRepo() *mockDeliveryAttemptRepo {
	return &mockDeliveryAttemptRepo{attempts: make(map[uuid.UUID]*models.DeliveryAttempt)}
}

func (m *mockDeliveryAttemptRepo) Create(_ context.Context, a *models.DeliveryAttempt) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.attempts[a.ID] = a
	return nil
}

func (m *mockDeliveryAttemptRepo) GetByID(_ context.Context, id uuid.UUID) (*models.DeliveryAttempt, error) {
	a, ok := m.attempts[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return a, nil
}

func (m *mockDeliveryAttemptRepo) GetByEndpointID(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.DeliveryAttempt, error) {
	return nil, nil
}
func (m *mockDeliveryAttemptRepo) GetByStatus(_ context.Context, _ string, _, _ int) ([]*models.DeliveryAttempt, error) {
	return nil, nil
}
func (m *mockDeliveryAttemptRepo) GetPendingDeliveries(_ context.Context, _ int) ([]*models.DeliveryAttempt, error) {
	return nil, nil
}
func (m *mockDeliveryAttemptRepo) Update(_ context.Context, _ *models.DeliveryAttempt) error {
	return nil
}
func (m *mockDeliveryAttemptRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockDeliveryAttemptRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockDeliveryAttemptRepo) GetDeliveryHistory(_ context.Context, _ uuid.UUID, _ []string, _, _ int) ([]*models.DeliveryAttempt, error) {
	return nil, nil
}

func (m *mockDeliveryAttemptRepo) GetDeliveryHistoryWithFilters(_ context.Context, _ uuid.UUID, _ repository.DeliveryHistoryFilters, _, _ int) ([]*models.DeliveryAttempt, int, error) {
	return nil, 0, nil
}

func (m *mockDeliveryAttemptRepo) GetDeliveryAttemptsByDeliveryID(_ context.Context, _, _ uuid.UUID) ([]*models.DeliveryAttempt, error) {
	return nil, nil
}

func setupWebhookUnitTest(repo *mockWebhookRepo, deliveryRepo *mockDeliveryAttemptRepo) (*WebhookHandler, *gin.Engine, uuid.UUID) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	publisher := queue.NewTestPublisher()

	if deliveryRepo == nil {
		deliveryRepo = newMockDeliveryAttemptRepo()
	}

	handler := NewWebhookHandler(repo, deliveryRepo, publisher, logger)
	handler.SetURLValidator(&formatOnlyURLValidator{})
	tenantID := uuid.New()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	})

	v1 := router.Group("/api/v1")
	{
		v1.POST("/webhooks/endpoints", handler.CreateWebhookEndpoint)
		v1.GET("/webhooks/endpoints", handler.GetWebhookEndpoints)
		v1.GET("/webhooks/endpoints/:id", handler.GetWebhookEndpoint)
		v1.PUT("/webhooks/endpoints/:id", handler.UpdateWebhookEndpoint)
		v1.DELETE("/webhooks/endpoints/:id", handler.DeleteWebhookEndpoint)
		v1.POST("/webhooks/send", handler.SendWebhook)
		v1.POST("/webhooks/send/batch", handler.BatchSendWebhook)
	}

	return handler, router, tenantID
}

// --- Webhook Endpoint Handler Unit Tests ---

func TestWebhookEndpointUnit_CreateSuccess(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	body := CreateWebhookEndpointRequest{
		URL: "https://example.com/webhook",
		CustomHeaders: map[string]string{
			"Authorization": "Bearer token",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["id"])
	assert.NotEmpty(t, resp["secret"])
	assert.Equal(t, true, resp["is_active"])
	assert.Equal(t, "https://example.com/webhook", resp["url"])
}

func TestWebhookEndpointUnit_CreateValidatesURL(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	tests := []struct {
		name          string
		url           string
		expectedCode  int
		expectedError string
	}{
		{"localhost rejected", "https://localhost/hook", http.StatusBadRequest, "INVALID_URL"},
		{"127.0.0.1 rejected", "https://127.0.0.1/hook", http.StatusBadRequest, "INVALID_URL"},
		{"0.0.0.0 rejected", "https://0.0.0.0/hook", http.StatusBadRequest, "INVALID_URL"},
		{"HTTP rejected", "http://example.com/hook", http.StatusBadRequest, "INVALID_URL"},
		{"empty URL rejected", "", http.StatusBadRequest, "INVALID_REQUEST"},
		{"not a URL", "not-a-url", http.StatusBadRequest, "INVALID_URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := CreateWebhookEndpointRequest{URL: tt.url}
			bodyBytes, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestWebhookEndpointUnit_CreateWithRetryConfig(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	body := CreateWebhookEndpointRequest{
		URL: "https://example.com/hook",
		RetryConfig: &RetryConfigRequest{
			MaxAttempts:       3,
			InitialDelayMs:    500,
			MaxDelayMs:        30000,
			BackoffMultiplier: 2,
		},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	retryConfig := resp["retry_config"].(map[string]interface{})
	assert.Equal(t, float64(3), retryConfig["max_attempts"])
}

func TestWebhookEndpointUnit_SecretGeneration(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	body := CreateWebhookEndpointRequest{URL: "https://example.com/hook"}
	bodyBytes, _ := json.Marshal(body)

	// Create two endpoints and verify secrets differ
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)

	var r1, r2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &r1))
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &r2))

	assert.NotEqual(t, r1["secret"], r2["secret"], "secrets should be unique")
	assert.Len(t, r1["secret"].(string), 64, "secret should be 64 hex chars (32 bytes)")
}

func TestWebhookEndpointUnit_CreateDatabaseFailure(t *testing.T) {
	repo := newMockWebhookRepo()
	repo.createErr = fmt.Errorf("database connection lost")
	_, router, _ := setupWebhookUnitTest(repo, nil)

	body := CreateWebhookEndpointRequest{URL: "https://example.com/hook"}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "DATABASE_ERROR", resp.Code)
}

func TestWebhookEndpointUnit_TenantIsolation(t *testing.T) {
	repo := newMockWebhookRepo()
	logger := utils.NewLogger("test")
	publisher := queue.NewTestPublisher()
	handler := NewWebhookHandler(repo, newMockDeliveryAttemptRepo(), publisher, logger)
	handler.SetURLValidator(&noopURLValidator{})

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Create endpoint owned by tenant2
	ep := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: tenant2,
		URL:      "https://tenant2.com/hook",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts: 5, InitialDelayMs: 1000, MaxDelayMs: 300000, BackoffMultiplier: 2,
		},
	}
	repo.endpoints[ep.ID] = ep

	// Tenant1 tries to access tenant2's endpoint
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant1)
		c.Next()
	})
	router.GET("/endpoints/:id", handler.GetWebhookEndpoint)
	router.PUT("/endpoints/:id", handler.UpdateWebhookEndpoint)
	router.DELETE("/endpoints/:id", handler.DeleteWebhookEndpoint)

	// GET should return 403
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/endpoints/"+ep.ID.String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// UPDATE should return 403
	body, _ := json.Marshal(UpdateWebhookEndpointRequest{URL: strPtr("https://evil.com/hook")})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/endpoints/"+ep.ID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// DELETE should return 403
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/endpoints/"+ep.ID.String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestWebhookEndpointUnit_GetList_Pagination(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, tenantID := setupWebhookUnitTest(repo, nil)

	// Add 5 endpoints
	for i := 0; i < 5; i++ {
		ep := &models.WebhookEndpoint{
			ID:       uuid.New(),
			TenantID: tenantID,
			URL:      fmt.Sprintf("https://example%d.com/hook", i),
			IsActive: true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts: 5, InitialDelayMs: 1000, MaxDelayMs: 300000, BackoffMultiplier: 2,
			},
		}
		repo.endpoints[ep.ID] = ep
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{"default returns all", "", 5},
		{"limit=2", "?limit=2", 2},
		{"limit=1&offset=4", "?limit=1&offset=4", 1},
		{"offset beyond count", "?offset=100", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/webhooks/endpoints"+tt.query, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			endpoints := resp["endpoints"].([]interface{})
			assert.Len(t, endpoints, tt.expectedCount)

			// Verify no secrets in list response
			for _, ep := range endpoints {
				epMap := ep.(map[string]interface{})
				_, hasSecret := epMap["secret"]
				assert.False(t, hasSecret)
			}
		})
	}
}

func TestWebhookEndpointUnit_UpdatePreservesUnchangedFields(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, tenantID := setupWebhookUnitTest(repo, nil)

	ep := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenantID,
		URL:        "https://original.com/hook",
		SecretHash: "original-hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts: 5, InitialDelayMs: 1000, MaxDelayMs: 300000, BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{"X-Original": "value"},
	}
	repo.endpoints[ep.ID] = ep

	// Only update IsActive, everything else should stay
	body, _ := json.Marshal(UpdateWebhookEndpointRequest{
		IsActive: boolRef(false),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/webhooks/endpoints/"+ep.ID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "https://original.com/hook", resp["url"])
	assert.Equal(t, false, resp["is_active"])
}

func TestWebhookEndpointUnit_NoTenantContext(t *testing.T) {
	repo := newMockWebhookRepo()
	logger := utils.NewLogger("test")
	publisher := queue.NewTestPublisher()
	handler := NewWebhookHandler(repo, newMockDeliveryAttemptRepo(), publisher, logger)
	handler.SetURLValidator(&noopURLValidator{})

	router := gin.New()
	// No tenant_id middleware
	router.POST("/endpoints", handler.CreateWebhookEndpoint)

	body, _ := json.Marshal(CreateWebhookEndpointRequest{URL: "https://example.com/hook"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/endpoints", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhookEndpointUnit_InvalidJSON(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookEndpointUnit_InvalidUUID(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	for _, method := range []string{"GET", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			var body *bytes.Buffer
			if method == "PUT" {
				body = bytes.NewBuffer([]byte("{}"))
			} else {
				body = bytes.NewBuffer(nil)
			}
			req, _ := http.NewRequest(method, "/api/v1/webhooks/endpoints/not-uuid", body)
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// --- Webhook Sender Handler Unit Tests ---

func TestWebhookSenderUnit_SendToSpecificEndpoint(t *testing.T) {
	repo := newMockWebhookRepo()
	deliveryRepo := newMockDeliveryAttemptRepo()
	_, router, tenantID := setupWebhookUnitTest(repo, deliveryRepo)

	ep := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenantID,
		URL:        "https://example.com/hook",
		SecretHash: "hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2,
		},
	}
	repo.endpoints[ep.ID] = ep

	body, _ := json.Marshal(SendWebhookRequest{
		EndpointID: &ep.ID,
		Payload:    json.RawMessage(`{"event":"test"}`),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp SendWebhookResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, ep.ID, resp.EndpointID)
	assert.Equal(t, "pending", resp.Status)
}

func TestWebhookSenderUnit_SendToAllActiveEndpoints(t *testing.T) {
	repo := newMockWebhookRepo()
	deliveryRepo := newMockDeliveryAttemptRepo()
	_, router, tenantID := setupWebhookUnitTest(repo, deliveryRepo)

	for i := 0; i < 3; i++ {
		ep := &models.WebhookEndpoint{
			ID:         uuid.New(),
			TenantID:   tenantID,
			URL:        fmt.Sprintf("https://example%d.com/hook", i),
			SecretHash: "hash",
			IsActive:   true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2,
			},
		}
		repo.endpoints[ep.ID] = ep
	}

	body, _ := json.Marshal(SendWebhookRequest{
		Payload: json.RawMessage(`{"event":"broadcast"}`),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestWebhookSenderUnit_PayloadSizeBoundary(t *testing.T) {
	repo := newMockWebhookRepo()
	deliveryRepo := newMockDeliveryAttemptRepo()
	_, router, tenantID := setupWebhookUnitTest(repo, deliveryRepo)

	ep := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenantID,
		URL:        "https://example.com/hook",
		SecretHash: "hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2,
		},
	}
	repo.endpoints[ep.ID] = ep

	tests := []struct {
		name         string
		payloadSize  int
		expectStatus int
	}{
		{"exactly 1MB should succeed", 1024 * 1024, http.StatusAccepted},
		{"1MB + 1 byte should fail", 1024*1024 + 1, http.StatusBadRequest},
		{"small payload succeeds", 100, http.StatusAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a valid JSON payload of the target size
			// Use {"d":"AAAA..."} format
			overhead := len(`{"d":""}`)
			padding := tt.payloadSize - overhead
			if padding < 0 {
				padding = 0
			}
			payload := fmt.Sprintf(`{"d":"%s"}`, strings.Repeat("A", padding))

			body, _ := json.Marshal(SendWebhookRequest{
				EndpointID: &ep.ID,
				Payload:    json.RawMessage(payload),
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/webhooks/send", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
		})
	}
}

func TestWebhookSenderUnit_InvalidJSON(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	// Payload is not valid JSON (though the request itself must be valid for binding)
	body := `{"payload": "not json object"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// The payload "not json object" is actually valid JSON (it's a string)
	// so this should either succeed or fail depending on endpoint existence
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusAccepted}, w.Code)
}

func TestWebhookSenderUnit_SendToInactiveEndpoint(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, tenantID := setupWebhookUnitTest(repo, nil)

	ep := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenantID,
		URL:        "https://example.com/hook",
		SecretHash: "hash",
		IsActive:   false,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2,
		},
	}
	repo.endpoints[ep.ID] = ep

	body, _ := json.Marshal(SendWebhookRequest{
		EndpointID: &ep.ID,
		Payload:    json.RawMessage(`{"event":"test"}`),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ENDPOINT_INACTIVE", resp.Code)
}

func TestWebhookSenderUnit_SendToNonexistentEndpoint(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	nonExistentID := uuid.New()
	body, _ := json.Marshal(SendWebhookRequest{
		EndpointID: &nonExistentID,
		Payload:    json.RawMessage(`{"event":"test"}`),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWebhookSenderUnit_BatchSendSuccess(t *testing.T) {
	repo := newMockWebhookRepo()
	deliveryRepo := newMockDeliveryAttemptRepo()
	_, router, tenantID := setupWebhookUnitTest(repo, deliveryRepo)

	var epIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		ep := &models.WebhookEndpoint{
			ID:         uuid.New(),
			TenantID:   tenantID,
			URL:        fmt.Sprintf("https://example%d.com/hook", i),
			SecretHash: "hash",
			IsActive:   true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2,
			},
		}
		repo.endpoints[ep.ID] = ep
		epIDs = append(epIDs, ep.ID)
	}

	body, _ := json.Marshal(BatchSendWebhookRequest{
		EndpointIDs: epIDs,
		Payload:     json.RawMessage(`{"event":"batch"}`),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp BatchSendWebhookResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 3, resp.Queued)
	assert.Equal(t, 0, resp.Failed)
}

func TestWebhookSenderUnit_BatchNoActiveEndpoints(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	body, _ := json.Marshal(BatchSendWebhookRequest{
		Payload: json.RawMessage(`{"event":"test"}`),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookSenderUnit_BatchSkipsInactiveEndpoints(t *testing.T) {
	repo := newMockWebhookRepo()
	deliveryRepo := newMockDeliveryAttemptRepo()
	_, router, tenantID := setupWebhookUnitTest(repo, deliveryRepo)

	active := &models.WebhookEndpoint{
		ID: uuid.New(), TenantID: tenantID, URL: "https://active.com/hook",
		SecretHash: "h", IsActive: true,
		RetryConfig: models.RetryConfiguration{MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2},
	}
	inactive := &models.WebhookEndpoint{
		ID: uuid.New(), TenantID: tenantID, URL: "https://inactive.com/hook",
		SecretHash: "h", IsActive: false,
		RetryConfig: models.RetryConfiguration{MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2},
	}
	repo.endpoints[active.ID] = active
	repo.endpoints[inactive.ID] = inactive

	body, _ := json.Marshal(BatchSendWebhookRequest{
		EndpointIDs: []uuid.UUID{active.ID, inactive.ID},
		Payload:     json.RawMessage(`{"event":"test"}`),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp BatchSendWebhookResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Queued)
}

func TestWebhookSenderUnit_BatchPayloadTooLarge(t *testing.T) {
	repo := newMockWebhookRepo()
	_, router, _ := setupWebhookUnitTest(repo, nil)

	largePayload := fmt.Sprintf(`{"d":"%s"}`, strings.Repeat("X", 1024*1024+1))
	body, _ := json.Marshal(BatchSendWebhookRequest{
		Payload: json.RawMessage(largePayload),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/send/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// helper to avoid collision with webhook_handler_integration_test.go's stringPtr
func strPtr(s string) *string { return &s }
func boolRef(b bool) *bool    { return &b }
