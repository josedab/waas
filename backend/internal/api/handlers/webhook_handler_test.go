package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/josedab/waas/pkg/database"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWebhookTestRouter(t *testing.T) (*gin.Engine, *database.DB, repository.TenantRepository, repository.WebhookEndpointRepository, *models.Tenant) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Skip integration tests when database is not available
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("Skipping integration test: TEST_DATABASE_URL not set")
	}

	// Create test database connection
	db, err := database.NewTestConnection()
	require.NoError(t, err)

	// Run migrations
	err = database.RunMigrations(db.GetConnectionString())
	require.NoError(t, err)

	// Create repositories
	tenantRepo := repository.NewTenantRepository(db)
	webhookRepo := repository.NewWebhookEndpointRepository(db)

	// Create logger
	logger := utils.NewLogger("test")

	// Create mock delivery attempt repository and publisher for testing
	deliveryAttemptRepo := repository.NewDeliveryAttemptRepository(db)
	mockPublisher := queue.NewTestPublisher()

	// Create handlers
	webhookHandler := NewWebhookHandler(webhookRepo, deliveryAttemptRepo, mockPublisher, logger)

	// Create test tenant
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		APIKeyHash:         "test-api-key-hash",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 100,
		MonthlyQuota:       10000,
	}
	err = tenantRepo.Create(context.Background(), tenant)
	require.NoError(t, err)

	// Setup router
	router := gin.New()

	// Add middleware to set tenant_id in context for testing
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Next()
	})

	// Setup routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/webhooks/endpoints", webhookHandler.CreateWebhookEndpoint)
		v1.GET("/webhooks/endpoints", webhookHandler.GetWebhookEndpoints)
		v1.GET("/webhooks/endpoints/:id", webhookHandler.GetWebhookEndpoint)
		v1.PUT("/webhooks/endpoints/:id", webhookHandler.UpdateWebhookEndpoint)
		v1.DELETE("/webhooks/endpoints/:id", webhookHandler.DeleteWebhookEndpoint)
	}

	return router, db, tenantRepo, webhookRepo, tenant
}

func TestCreateWebhookEndpoint(t *testing.T) {
	router, db, _, _, _ := setupWebhookTestRouter(t)
	defer db.Close()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid webhook endpoint",
			requestBody: CreateWebhookEndpointRequest{
				URL: "https://example.com/webhook",
				CustomHeaders: map[string]string{
					"Authorization": "Bearer token",
				},
				RetryConfig: &RetryConfigRequest{
					MaxAttempts:       3,
					InitialDelayMs:    500,
					MaxDelayMs:        30000,
					BackoffMultiplier: 2,
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "valid webhook endpoint with defaults",
			requestBody: CreateWebhookEndpointRequest{
				URL: "https://api.example.com/hooks",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid URL - not HTTPS",
			requestBody: CreateWebhookEndpointRequest{
				URL: "http://example.com/webhook",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name: "invalid URL - localhost",
			requestBody: CreateWebhookEndpointRequest{
				URL: "https://localhost:8080/webhook",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name: "missing URL",
			requestBody: CreateWebhookEndpointRequest{
				CustomHeaders: map[string]string{
					"Authorization": "Bearer token",
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req, err := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedStatus == http.StatusCreated {
				// Check successful response
				assert.NotEmpty(t, response["id"])
				assert.NotEmpty(t, response["secret"])
				assert.Equal(t, true, response["is_active"])
				assert.NotEmpty(t, response["created_at"])
				assert.NotEmpty(t, response["updated_at"])

				// Verify URL
				if req, ok := tt.requestBody.(CreateWebhookEndpointRequest); ok {
					assert.Equal(t, req.URL, response["url"])
				}
			} else {
				// Check error response
				errorObj, exists := response["error"]
				require.True(t, exists)
				errorMap := errorObj.(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorMap["code"])
			}
		})
	}
}

func TestGetWebhookEndpoints(t *testing.T) {
	router, db, _, webhookRepo, tenant := setupWebhookTestRouter(t)
	defer db.Close()

	// Create test endpoints
	endpoints := []*models.WebhookEndpoint{
		{
			ID:         uuid.New(),
			TenantID:   tenant.ID,
			URL:        "https://example1.com/webhook",
			SecretHash: "hash1",
			IsActive:   true,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       5,
				InitialDelayMs:    1000,
				MaxDelayMs:        300000,
				BackoffMultiplier: 2,
			},
			CustomHeaders: map[string]string{"Auth": "Bearer token1"},
		},
		{
			ID:         uuid.New(),
			TenantID:   tenant.ID,
			URL:        "https://example2.com/webhook",
			SecretHash: "hash2",
			IsActive:   false,
			RetryConfig: models.RetryConfiguration{
				MaxAttempts:       3,
				InitialDelayMs:    500,
				MaxDelayMs:        60000,
				BackoffMultiplier: 2,
			},
			CustomHeaders: map[string]string{},
		},
	}

	for _, endpoint := range endpoints {
		err := webhookRepo.Create(context.Background(), endpoint)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "get all endpoints",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "get with limit",
			queryParams:    "?limit=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "get with offset",
			queryParams:    "?offset=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "get with limit and offset",
			queryParams:    "?limit=1&offset=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/webhooks/endpoints"+tt.queryParams, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			endpoints := response["endpoints"].([]interface{})
			assert.Len(t, endpoints, tt.expectedCount)

			// Verify pagination info
			pagination := response["pagination"].(map[string]interface{})
			assert.NotNil(t, pagination["limit"])
			assert.NotNil(t, pagination["offset"])
			assert.Equal(t, float64(tt.expectedCount), pagination["count"])

			// Verify no secrets are returned
			for _, endpoint := range endpoints {
				endpointMap := endpoint.(map[string]interface{})
				_, hasSecret := endpointMap["secret"]
				assert.False(t, hasSecret, "Secret should not be returned in list")
			}
		})
	}
}

func TestGetWebhookEndpoint(t *testing.T) {
	router, db, _, webhookRepo, tenant := setupWebhookTestRouter(t)
	defer db.Close()

	// Create test endpoint
	endpoint := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenant.ID,
		URL:        "https://example.com/webhook",
		SecretHash: "test-hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{"Auth": "Bearer token"},
	}
	err := webhookRepo.Create(context.Background(), endpoint)
	require.NoError(t, err)

	// Create endpoint for different tenant
	otherTenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Other Tenant",
		APIKeyHash:         "other-api-key-hash",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 100,
		MonthlyQuota:       10000,
	}

	otherEndpoint := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   otherTenant.ID,
		URL:        "https://other.com/webhook",
		SecretHash: "other-hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{},
	}

	tests := []struct {
		name           string
		endpointID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "get existing endpoint",
			endpointID:     endpoint.ID.String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get non-existent endpoint",
			endpointID:     uuid.New().String(),
			expectedStatus: http.StatusNotFound,
			expectedError:  "ENDPOINT_NOT_FOUND",
		},
		{
			name:           "get endpoint from other tenant",
			endpointID:     otherEndpoint.ID.String(),
			expectedStatus: http.StatusForbidden,
			expectedError:  "FORBIDDEN",
		},
		{
			name:           "invalid endpoint ID",
			endpointID:     "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/webhooks/endpoints/"+tt.endpointID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedStatus == http.StatusOK {
				// Check successful response
				assert.Equal(t, endpoint.ID.String(), response["id"])
				assert.Equal(t, endpoint.URL, response["url"])
				assert.Equal(t, endpoint.IsActive, response["is_active"])

				// Verify no secret is returned
				_, hasSecret := response["secret"]
				assert.False(t, hasSecret, "Secret should not be returned")
			} else {
				// Check error response
				errorObj, exists := response["error"]
				require.True(t, exists)
				errorMap := errorObj.(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorMap["code"])
			}
		})
	}
}

func TestUpdateWebhookEndpoint(t *testing.T) {
	router, db, _, webhookRepo, tenant := setupWebhookTestRouter(t)
	defer db.Close()

	// Create test endpoint
	endpoint := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenant.ID,
		URL:        "https://example.com/webhook",
		SecretHash: "test-hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{"Auth": "Bearer token"},
	}
	err := webhookRepo.Create(context.Background(), endpoint)
	require.NoError(t, err)

	tests := []struct {
		name           string
		endpointID     string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "update URL",
			endpointID: endpoint.ID.String(),
			requestBody: UpdateWebhookEndpointRequest{
				URL: stringPtr("https://newexample.com/webhook"),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "update active status",
			endpointID: endpoint.ID.String(),
			requestBody: UpdateWebhookEndpointRequest{
				IsActive: boolPtr(false),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "update custom headers",
			endpointID: endpoint.ID.String(),
			requestBody: UpdateWebhookEndpointRequest{
				CustomHeaders: map[string]string{
					"Authorization": "Bearer newtoken",
					"X-Custom":      "value",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "update retry config",
			endpointID: endpoint.ID.String(),
			requestBody: UpdateWebhookEndpointRequest{
				RetryConfig: &RetryConfigRequest{
					MaxAttempts:       3,
					InitialDelayMs:    2000,
					MaxDelayMs:        60000,
					BackoffMultiplier: 3,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "update multiple fields",
			endpointID: endpoint.ID.String(),
			requestBody: UpdateWebhookEndpointRequest{
				URL:      stringPtr("https://updated.com/webhook"),
				IsActive: boolPtr(true),
				CustomHeaders: map[string]string{
					"X-Updated": "true",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "invalid URL - not HTTPS",
			endpointID: endpoint.ID.String(),
			requestBody: UpdateWebhookEndpointRequest{
				URL: stringPtr("http://example.com/webhook"),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name:       "invalid URL - localhost",
			endpointID: endpoint.ID.String(),
			requestBody: UpdateWebhookEndpointRequest{
				URL: stringPtr("https://localhost/webhook"),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name:           "non-existent endpoint",
			endpointID:     uuid.New().String(),
			requestBody:    UpdateWebhookEndpointRequest{},
			expectedStatus: http.StatusNotFound,
			expectedError:  "ENDPOINT_NOT_FOUND",
		},
		{
			name:           "invalid endpoint ID",
			endpointID:     "invalid-uuid",
			requestBody:    UpdateWebhookEndpointRequest{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("PUT", "/api/v1/webhooks/endpoints/"+tt.endpointID, bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedStatus == http.StatusOK {
				// Check successful response
				assert.Equal(t, endpoint.ID.String(), response["id"])
				assert.NotEmpty(t, response["updated_at"])

				// Verify no secret is returned
				_, hasSecret := response["secret"]
				assert.False(t, hasSecret, "Secret should not be returned")

				// Verify specific updates
				if updateReq, ok := tt.requestBody.(UpdateWebhookEndpointRequest); ok {
					if updateReq.URL != nil {
						assert.Equal(t, *updateReq.URL, response["url"])
					}
					if updateReq.IsActive != nil {
						assert.Equal(t, *updateReq.IsActive, response["is_active"])
					}
				}
			} else {
				// Check error response
				errorObj, exists := response["error"]
				require.True(t, exists)
				errorMap := errorObj.(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorMap["code"])
			}
		})
	}
}

func TestDeleteWebhookEndpoint(t *testing.T) {
	router, db, _, webhookRepo, tenant := setupWebhookTestRouter(t)
	defer db.Close()

	// Create test endpoint
	endpoint := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenant.ID,
		URL:        "https://example.com/webhook",
		SecretHash: "test-hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{},
	}
	err := webhookRepo.Create(context.Background(), endpoint)
	require.NoError(t, err)

	tests := []struct {
		name           string
		endpointID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "delete existing endpoint",
			endpointID:     endpoint.ID.String(),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "delete non-existent endpoint",
			endpointID:     uuid.New().String(),
			expectedStatus: http.StatusNotFound,
			expectedError:  "ENDPOINT_NOT_FOUND",
		},
		{
			name:           "invalid endpoint ID",
			endpointID:     "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v1/webhooks/endpoints/"+tt.endpointID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusNoContent {
				// Verify endpoint was actually deleted
				_, err := webhookRepo.GetByID(context.Background(), endpoint.ID)
				assert.Error(t, err, "Endpoint should be deleted")
			} else {
				// Check error response
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				errorObj, exists := response["error"]
				require.True(t, exists)
				errorMap := errorObj.(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorMap["code"])
			}
		})
	}
}

func TestWebhookEndpointValidation(t *testing.T) {
	router, db, _, _, _ := setupWebhookTestRouter(t)
	defer db.Close()

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid HTTPS URL",
			url:            "https://api.example.com/webhook",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid HTTPS URL with path",
			url:            "https://api.example.com/webhooks/receive",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid HTTPS URL with query params",
			url:            "https://api.example.com/webhook?token=abc123",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "HTTP URL (not allowed)",
			url:            "http://api.example.com/webhook",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name:           "localhost URL (not allowed)",
			url:            "https://localhost:8080/webhook",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name:           "127.0.0.1 URL (not allowed)",
			url:            "https://127.0.0.1:8080/webhook",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name:           "0.0.0.0 URL (not allowed)",
			url:            "https://0.0.0.0:8080/webhook",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name:           "invalid URL format",
			url:            "not-a-url",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name:           "empty URL",
			url:            "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := CreateWebhookEndpointRequest{
				URL: tt.url,
			}

			body, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/webhooks/endpoints", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedStatus != http.StatusCreated {
				errorObj, exists := response["error"]
				require.True(t, exists)
				errorMap := errorObj.(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorMap["code"])
			}
		})
	}
}

func TestWebhookEndpointTenantIsolation(t *testing.T) {
	// Skip integration tests when database is not available
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("Skipping integration test: TEST_DATABASE_URL not set")
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test database connection
	db, err := database.NewTestConnection()
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	err = database.RunMigrations(db.GetConnectionString())
	require.NoError(t, err)

	// Create repositories
	tenantRepo := repository.NewTenantRepository(db)
	webhookRepo := repository.NewWebhookEndpointRepository(db)
	deliveryAttemptRepo := repository.NewDeliveryAttemptRepository(db)

	// Create logger
	logger := utils.NewLogger("test")

	// Create mock publisher for testing
	mockPublisher := queue.NewTestPublisher()

	// Create handlers
	webhookHandler := NewWebhookHandler(webhookRepo, deliveryAttemptRepo, mockPublisher, logger)

	// Create two test tenants
	tenant1 := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Tenant 1",
		APIKeyHash:         "tenant1-api-key-hash",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 100,
		MonthlyQuota:       10000,
	}
	err = tenantRepo.Create(context.Background(), tenant1)
	require.NoError(t, err)

	tenant2 := &models.Tenant{
		ID:                 uuid.New(),
		Name:               "Tenant 2",
		APIKeyHash:         "tenant2-api-key-hash",
		SubscriptionTier:   "basic",
		RateLimitPerMinute: 100,
		MonthlyQuota:       10000,
	}
	err = tenantRepo.Create(context.Background(), tenant2)
	require.NoError(t, err)

	// Create endpoints for each tenant
	endpoint1 := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenant1.ID,
		URL:        "https://tenant1.com/webhook",
		SecretHash: "tenant1-hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{},
	}
	err = webhookRepo.Create(context.Background(), endpoint1)
	require.NoError(t, err)

	endpoint2 := &models.WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   tenant2.ID,
		URL:        "https://tenant2.com/webhook",
		SecretHash: "tenant2-hash",
		IsActive:   true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{},
	}
	err = webhookRepo.Create(context.Background(), endpoint2)
	require.NoError(t, err)

	// Test tenant 1 can only see their endpoints
	t.Run("tenant 1 isolation", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("tenant_id", tenant1.ID)
			c.Next()
		})
		router.GET("/webhooks/endpoints", webhookHandler.GetWebhookEndpoints)
		router.GET("/webhooks/endpoints/:id", webhookHandler.GetWebhookEndpoint)

		// Test list endpoints
		req, err := http.NewRequest("GET", "/webhooks/endpoints", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		endpoints := response["endpoints"].([]interface{})
		assert.Len(t, endpoints, 1)

		endpoint := endpoints[0].(map[string]interface{})
		assert.Equal(t, endpoint1.ID.String(), endpoint["id"])

		// Test get specific endpoint (should work for own endpoint)
		req, err = http.NewRequest("GET", "/webhooks/endpoints/"+endpoint1.ID.String(), nil)
		require.NoError(t, err)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Test get other tenant's endpoint (should fail)
		req, err = http.NewRequest("GET", "/webhooks/endpoints/"+endpoint2.ID.String(), nil)
		require.NoError(t, err)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// Test tenant 2 can only see their endpoints
	t.Run("tenant 2 isolation", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("tenant_id", tenant2.ID)
			c.Next()
		})
		router.GET("/webhooks/endpoints", webhookHandler.GetWebhookEndpoints)

		req, err := http.NewRequest("GET", "/webhooks/endpoints", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		endpoints := response["endpoints"].([]interface{})
		assert.Len(t, endpoints, 1)

		endpoint := endpoints[0].(map[string]interface{})
		assert.Equal(t, endpoint2.ID.String(), endpoint["id"])
	})
}
