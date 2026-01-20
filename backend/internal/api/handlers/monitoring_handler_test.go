package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/monitoring"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockDeliveryAttemptRepositoryWithFilters extends the existing mock with new methods
type MockDeliveryAttemptRepositoryWithFilters struct {
	*MockDeliveryAttemptRepository
	deliveryHistoryWithFilters map[string][]*models.DeliveryAttempt
	deliveryHistoryCount       map[string]int
	deliveryAttemptsByID       map[uuid.UUID][]*models.DeliveryAttempt
}

func NewMockDeliveryAttemptRepositoryWithFilters() *MockDeliveryAttemptRepositoryWithFilters {
	return &MockDeliveryAttemptRepositoryWithFilters{
		MockDeliveryAttemptRepository: NewMockDeliveryAttemptRepository(),
		deliveryHistoryWithFilters:    make(map[string][]*models.DeliveryAttempt),
		deliveryHistoryCount:          make(map[string]int),
		deliveryAttemptsByID:          make(map[uuid.UUID][]*models.DeliveryAttempt),
	}
}

func (m *MockDeliveryAttemptRepositoryWithFilters) GetDeliveryHistoryWithFilters(ctx context.Context, tenantID uuid.UUID, filters repository.DeliveryHistoryFilters, limit, offset int) ([]*models.DeliveryAttempt, int, error) {
	key := fmt.Sprintf("%s-%d-%d", tenantID.String(), limit, offset)
	if attempts, exists := m.deliveryHistoryWithFilters[key]; exists {
		count := m.deliveryHistoryCount[key]
		return attempts, count, nil
	}
	return []*models.DeliveryAttempt{}, 0, fmt.Errorf("no data configured for key: %s", key)
}

func (m *MockDeliveryAttemptRepositoryWithFilters) GetDeliveryAttemptsByDeliveryID(ctx context.Context, deliveryID uuid.UUID, tenantID uuid.UUID) ([]*models.DeliveryAttempt, error) {
	if attempts, exists := m.deliveryAttemptsByID[deliveryID]; exists {
		return attempts, nil
	}
	return []*models.DeliveryAttempt{}, nil
}

func (m *MockDeliveryAttemptRepositoryWithFilters) SetDeliveryHistoryWithFilters(tenantID uuid.UUID, limit, offset int, attempts []*models.DeliveryAttempt, totalCount int) {
	key := fmt.Sprintf("%s-%d-%d", tenantID.String(), limit, offset)
	m.deliveryHistoryWithFilters[key] = attempts
	m.deliveryHistoryCount[key] = totalCount
}

func (m *MockDeliveryAttemptRepositoryWithFilters) SetDeliveryAttemptsByID(deliveryID uuid.UUID, attempts []*models.DeliveryAttempt) {
	m.deliveryAttemptsByID[deliveryID] = attempts
}

func TestMonitoringHandler_GetDeliveryHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		tenantID       uuid.UUID
		queryParams    string
		setupMocks     func(*MockDeliveryAttemptRepositoryWithFilters, *MockWebhookRepository)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:        "successful delivery history retrieval",
			tenantID:    uuid.New(),
			queryParams: "?limit=10&offset=0",
			setupMocks: func(deliveryRepo *MockDeliveryAttemptRepositoryWithFilters, webhookRepo *MockWebhookRepository) {
				attempts := []*models.DeliveryAttempt{
					{
						ID:            uuid.New(),
						EndpointID:    uuid.New(),
						PayloadHash:   "hash1",
						PayloadSize:   100,
						Status:        "delivered",
						HTTPStatus:    &[]int{200}[0],
						AttemptNumber: 1,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
					{
						ID:            uuid.New(),
						EndpointID:    uuid.New(),
						PayloadHash:   "hash2",
						PayloadSize:   150,
						Status:        "failed",
						HTTPStatus:    &[]int{500}[0],
						AttemptNumber: 3,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
				}
				deliveryRepo.SetDeliveryHistoryWithFilters(uuid.New(), 10, 0, attempts, 2)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:        "delivery history with filters",
			tenantID:    uuid.New(),
			queryParams: "?statuses=delivered,failed&limit=5",
			setupMocks: func(deliveryRepo *MockDeliveryAttemptRepositoryWithFilters, webhookRepo *MockWebhookRepository) {
				attempts := []*models.DeliveryAttempt{
					{
						ID:            uuid.New(),
						EndpointID:    uuid.New(),
						PayloadHash:   "hash1",
						PayloadSize:   100,
						Status:        "delivered",
						HTTPStatus:    &[]int{200}[0],
						AttemptNumber: 1,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
				}
				deliveryRepo.SetDeliveryHistoryWithFilters(uuid.New(), 5, 0, attempts, 1)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			deliveryRepo := NewMockDeliveryAttemptRepositoryWithFilters()
			webhookRepo := NewMockWebhookRepository()
			logger := utils.NewLogger("test")

			tt.setupMocks(deliveryRepo, webhookRepo)

			// Update the mock to use the test tenant ID
			if tt.expectedCount == 2 {
				deliveryRepo.SetDeliveryHistoryWithFilters(tt.tenantID, 10, 0, []*models.DeliveryAttempt{
					{
						ID:            uuid.New(),
						EndpointID:    uuid.New(),
						PayloadHash:   "hash1",
						PayloadSize:   100,
						Status:        "delivered",
						HTTPStatus:    &[]int{200}[0],
						AttemptNumber: 1,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
					{
						ID:            uuid.New(),
						EndpointID:    uuid.New(),
						PayloadHash:   "hash2",
						PayloadSize:   150,
						Status:        "failed",
						HTTPStatus:    &[]int{500}[0],
						AttemptNumber: 3,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
				}, tt.expectedCount)
			} else {
				deliveryRepo.SetDeliveryHistoryWithFilters(tt.tenantID, 10, 0, []*models.DeliveryAttempt{
					{
						ID:            uuid.New(),
						EndpointID:    uuid.New(),
						PayloadHash:   "hash1",
						PayloadSize:   100,
						Status:        "delivered",
						HTTPStatus:    &[]int{200}[0],
						AttemptNumber: 1,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
				}, tt.expectedCount)
			}

			if tt.queryParams == "?statuses=delivered,failed&limit=5" {
				deliveryRepo.SetDeliveryHistoryWithFilters(tt.tenantID, 5, 0, []*models.DeliveryAttempt{
					{
						ID:            uuid.New(),
						EndpointID:    uuid.New(),
						PayloadHash:   "hash1",
						PayloadSize:   100,
						Status:        "delivered",
						HTTPStatus:    &[]int{200}[0],
						AttemptNumber: 1,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
				}, 1)
			}

			// Create handler
			handler := NewMonitoringHandler(deliveryRepo, webhookRepo, logger, nil, nil, nil)

			// Setup Gin
			router := gin.New()
			router.GET("/webhooks/deliveries", func(c *gin.Context) {
				c.Set("tenant_id", tt.tenantID)
				handler.GetDeliveryHistory(c)
			})

			// Create request
			req, _ := http.NewRequest("GET", "/webhooks/deliveries"+tt.queryParams, nil)
			req.Header.Set("X-Correlation-ID", "test-correlation-id")
			req.Header.Set("X-Request-ID", "test-request-id")

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response DeliveryHistoryResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(response.Deliveries))
				assert.NotNil(t, response.Pagination)
			}
		})
	}
}

func TestMonitoringHandler_GetDeliveryDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		tenantID       uuid.UUID
		deliveryID     uuid.UUID
		setupMocks     func(*MockDeliveryAttemptRepositoryWithFilters, *MockWebhookRepository, uuid.UUID, uuid.UUID)
		expectedStatus int
	}{
		{
			name:       "successful delivery details retrieval",
			tenantID:   uuid.New(),
			deliveryID: uuid.New(),
			setupMocks: func(deliveryRepo *MockDeliveryAttemptRepositoryWithFilters, webhookRepo *MockWebhookRepository, tenantID, deliveryID uuid.UUID) {
				attempts := []*models.DeliveryAttempt{
					{
						ID:            deliveryID,
						EndpointID:    uuid.New(),
						PayloadHash:   "hash1",
						PayloadSize:   100,
						Status:        "pending",
						AttemptNumber: 1,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
					{
						ID:            deliveryID,
						EndpointID:    uuid.New(),
						PayloadHash:   "hash1",
						PayloadSize:   100,
						Status:        "delivered",
						HTTPStatus:    &[]int{200}[0],
						AttemptNumber: 2,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
				}
				deliveryRepo.SetDeliveryAttemptsByID(deliveryID, attempts)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "delivery not found",
			tenantID:   uuid.New(),
			deliveryID: uuid.New(),
			setupMocks: func(deliveryRepo *MockDeliveryAttemptRepositoryWithFilters, webhookRepo *MockWebhookRepository, tenantID, deliveryID uuid.UUID) {
				deliveryRepo.SetDeliveryAttemptsByID(deliveryID, []*models.DeliveryAttempt{})
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			deliveryRepo := NewMockDeliveryAttemptRepositoryWithFilters()
			webhookRepo := NewMockWebhookRepository()
			logger := utils.NewLogger("test")

			tt.setupMocks(deliveryRepo, webhookRepo, tt.tenantID, tt.deliveryID)

			// Create handler
			handler := NewMonitoringHandler(deliveryRepo, webhookRepo, logger, nil, nil, nil)

			// Setup Gin
			router := gin.New()
			router.GET("/webhooks/deliveries/:id", func(c *gin.Context) {
				c.Set("tenant_id", tt.tenantID)
				handler.GetDeliveryDetails(c)
			})

			// Create request
			req, _ := http.NewRequest("GET", "/webhooks/deliveries/"+tt.deliveryID.String(), nil)
			req.Header.Set("X-Correlation-ID", "test-correlation-id")
			req.Header.Set("X-Request-ID", "test-request-id")

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response DeliveryDetailResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.deliveryID, response.DeliveryID)
				assert.NotEmpty(t, response.Attempts)
				assert.NotNil(t, response.Summary)
			}
		})
	}
}

func TestMonitoringHandler_GetEndpointDeliveryHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		tenantID       uuid.UUID
		endpointID     uuid.UUID
		queryParams    string
		setupMocks     func(*MockDeliveryAttemptRepositoryWithFilters, *MockWebhookRepository, uuid.UUID, uuid.UUID)
		expectedStatus int
	}{
		{
			name:        "successful endpoint delivery history",
			tenantID:    uuid.New(),
			endpointID:  uuid.New(),
			queryParams: "?limit=10&status=delivered",
			setupMocks: func(deliveryRepo *MockDeliveryAttemptRepositoryWithFilters, webhookRepo *MockWebhookRepository, tenantID, endpointID uuid.UUID) {
				endpoint := &models.WebhookEndpoint{
					ID:       endpointID,
					TenantID: tenantID,
					URL:      "https://example.com/webhook",
					IsActive: true,
				}
				webhookRepo.SetEndpoint(endpointID, endpoint)

				attempts := []*models.DeliveryAttempt{
					{
						ID:            uuid.New(),
						EndpointID:    endpointID,
						PayloadHash:   "hash1",
						PayloadSize:   100,
						Status:        "delivered",
						HTTPStatus:    &[]int{200}[0],
						AttemptNumber: 1,
						ScheduledAt:   time.Now(),
						CreatedAt:     time.Now(),
					},
				}
				deliveryRepo.SetDeliveryHistoryForEndpoint(endpointID, []string{"delivered"}, 10, 0, attempts)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "endpoint not found",
			tenantID:    uuid.New(),
			endpointID:  uuid.New(),
			queryParams: "",
			setupMocks: func(deliveryRepo *MockDeliveryAttemptRepositoryWithFilters, webhookRepo *MockWebhookRepository, tenantID, endpointID uuid.UUID) {
				// Don't set any endpoint - will return not found
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "access denied to endpoint",
			tenantID:    uuid.New(),
			endpointID:  uuid.New(),
			queryParams: "",
			setupMocks: func(deliveryRepo *MockDeliveryAttemptRepositoryWithFilters, webhookRepo *MockWebhookRepository, tenantID, endpointID uuid.UUID) {
				endpoint := &models.WebhookEndpoint{
					ID:       endpointID,
					TenantID: uuid.New(), // Different tenant ID
					URL:      "https://example.com/webhook",
					IsActive: true,
				}
				webhookRepo.SetEndpoint(endpointID, endpoint)
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			deliveryRepo := NewMockDeliveryAttemptRepositoryWithFilters()
			webhookRepo := NewMockWebhookRepository()
			logger := utils.NewLogger("test")

			tt.setupMocks(deliveryRepo, webhookRepo, tt.tenantID, tt.endpointID)

			// Create handler
			handler := NewMonitoringHandler(deliveryRepo, webhookRepo, logger, nil, nil, nil)

			// Setup Gin
			router := gin.New()
			router.GET("/webhooks/endpoints/:id/deliveries", func(c *gin.Context) {
				c.Set("tenant_id", tt.tenantID)
				handler.GetEndpointDeliveryHistory(c)
			})

			// Create request
			req, _ := http.NewRequest("GET", "/webhooks/endpoints/"+tt.endpointID.String()+"/deliveries"+tt.queryParams, nil)
			req.Header.Set("X-Correlation-ID", "test-correlation-id")
			req.Header.Set("X-Request-ID", "test-request-id")

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestMonitoringHandler_BuildDeliverySummary(t *testing.T) {
	handler := &MonitoringHandler{}

	tests := []struct {
		name     string
		attempts []*models.DeliveryAttempt
		expected DeliverySummary
	}{
		{
			name:     "empty attempts",
			attempts: []*models.DeliveryAttempt{},
			expected: DeliverySummary{},
		},
		{
			name: "single successful attempt",
			attempts: []*models.DeliveryAttempt{
				{
					ID:            uuid.New(),
					Status:        "delivered",
					HTTPStatus:    &[]int{200}[0],
					AttemptNumber: 1,
					CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
			expected: DeliverySummary{
				TotalAttempts:   1,
				Status:          "delivered",
				FirstAttemptAt:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				LastAttemptAt:   &[]time.Time{time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}[0],
				FinalHTTPStatus: &[]int{200}[0],
			},
		},
		{
			name: "multiple attempts with final failure",
			attempts: []*models.DeliveryAttempt{
				{
					ID:            uuid.New(),
					Status:        "pending",
					AttemptNumber: 1,
					CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					ID:            uuid.New(),
					Status:        "failed",
					HTTPStatus:    &[]int{500}[0],
					ErrorMessage:  &[]string{"Internal server error"}[0],
					AttemptNumber: 2,
					CreatedAt:     time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
				},
			},
			expected: DeliverySummary{
				TotalAttempts:   2,
				Status:          "failed",
				FirstAttemptAt:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				LastAttemptAt:   &[]time.Time{time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC)}[0],
				FinalHTTPStatus: &[]int{500}[0],
				FinalErrorMsg:   &[]string{"Internal server error"}[0],
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.buildDeliverySummary(tt.attempts)

			assert.Equal(t, tt.expected.TotalAttempts, result.TotalAttempts)
			assert.Equal(t, tt.expected.Status, result.Status)
			assert.Equal(t, tt.expected.FirstAttemptAt, result.FirstAttemptAt)

			if tt.expected.LastAttemptAt != nil {
				assert.NotNil(t, result.LastAttemptAt)
				assert.Equal(t, *tt.expected.LastAttemptAt, *result.LastAttemptAt)
			} else {
				assert.Nil(t, result.LastAttemptAt)
			}

			if tt.expected.FinalHTTPStatus != nil {
				assert.NotNil(t, result.FinalHTTPStatus)
				assert.Equal(t, *tt.expected.FinalHTTPStatus, *result.FinalHTTPStatus)
			} else {
				assert.Nil(t, result.FinalHTTPStatus)
			}

			if tt.expected.FinalErrorMsg != nil {
				assert.NotNil(t, result.FinalErrorMsg)
				assert.Equal(t, *tt.expected.FinalErrorMsg, *result.FinalErrorMsg)
			} else {
				assert.Nil(t, result.FinalErrorMsg)
			}
		})
	}
}

func TestMonitoringHandler_TruncateResponseBody(t *testing.T) {
	handler := &MonitoringHandler{}

	tests := []struct {
		name     string
		input    *string
		expected *string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "short response body",
			input:    &[]string{"Short response"}[0],
			expected: &[]string{"Short response"}[0],
		},
		{
			name:     "long response body",
			input:    &[]string{string(bytes.Repeat([]byte("a"), 1500))}[0],
			expected: &[]string{string(bytes.Repeat([]byte("a"), 1000)) + "... [truncated]"}[0],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.truncateResponseBody(tt.input)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

// Helper method for the mock
func (m *MockDeliveryAttemptRepositoryWithFilters) SetDeliveryHistoryForEndpoint(endpointID uuid.UUID, statuses []string, limit, offset int, attempts []*models.DeliveryAttempt) {
	// Store in the base mock's delivery history
	key := fmt.Sprintf("%s-%v-%d-%d", endpointID.String(), statuses, limit, offset)
	m.deliveryHistoryWithFilters[key] = attempts
}

// TestMonitoringHandler_HealthEndpoints tests health check endpoints
func TestMonitoringHandler_HealthEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		endpoint       string
		setupHandler   func() *MonitoringHandler
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "health status with health checker",
			endpoint: "/health",
			setupHandler: func() *MonitoringHandler {
				logger := utils.NewLogger("test")
				healthChecker := monitoring.NewHealthChecker(nil, nil, logger, "test-1.0.0")
				return NewMonitoringHandler(nil, nil, logger, healthChecker, nil, nil)
			},
			expectedStatus: http.StatusServiceUnavailable, // DB will be unhealthy
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response monitoring.HealthCheckResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "test-1.0.0", response.Version)
				assert.Contains(t, response.Components, "database")
			},
		},
		{
			name:     "health status without health checker",
			endpoint: "/health",
			setupHandler: func() *MonitoringHandler {
				logger := utils.NewLogger("test")
				return NewMonitoringHandler(nil, nil, logger, nil, nil, nil)
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			},
		},
		{
			name:     "readiness status",
			endpoint: "/ready",
			setupHandler: func() *MonitoringHandler {
				logger := utils.NewLogger("test")
				healthChecker := monitoring.NewHealthChecker(nil, nil, logger, "test-1.0.0")
				return NewMonitoringHandler(nil, nil, logger, healthChecker, nil, nil)
			},
			expectedStatus: http.StatusServiceUnavailable,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "not ready", response["status"])
			},
		},
		{
			name:     "liveness status",
			endpoint: "/live",
			setupHandler: func() *MonitoringHandler {
				logger := utils.NewLogger("test")
				healthChecker := monitoring.NewHealthChecker(nil, nil, logger, "test-1.0.0")
				return NewMonitoringHandler(nil, nil, logger, healthChecker, nil, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "alive", response["status"])
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupHandler()
			
			router := gin.New()
			router.GET(tt.endpoint, func(c *gin.Context) {
				switch tt.endpoint {
				case "/health":
					handler.GetHealthStatus(c)
				case "/ready":
					handler.GetReadinessStatus(c)
				case "/live":
					handler.GetLivenessStatus(c)
				}
			})
			
			req, _ := http.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestMonitoringHandler_AlertEndpoints tests alert management endpoints
func TestMonitoringHandler_AlertEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		endpoint       string
		queryParams    string
		setupHandler   func() *MonitoringHandler
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "get active alerts with alert manager",
			endpoint: "/alerts/active",
			setupHandler: func() *MonitoringHandler {
				logger := utils.NewLogger("test")
				alertManager := monitoring.NewAlertManager(logger)
				
				// Trigger an alert
				alertManager.EvaluateMetric("test_metric", 10.0, map[string]string{
					"component": "test",
				})
				
				return NewMonitoringHandler(nil, nil, logger, nil, alertManager, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "alerts")
				assert.Contains(t, response, "count")
			},
		},
		{
			name:     "get active alerts without alert manager",
			endpoint: "/alerts/active",
			setupHandler: func() *MonitoringHandler {
				logger := utils.NewLogger("test")
				return NewMonitoringHandler(nil, nil, logger, nil, nil, nil)
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			},
		},
		{
			name:        "get alert history with filters",
			endpoint:    "/alerts/history",
			queryParams: "?limit=10&severity=critical",
			setupHandler: func() *MonitoringHandler {
				logger := utils.NewLogger("test")
				alertManager := monitoring.NewAlertManager(logger)
				return NewMonitoringHandler(nil, nil, logger, nil, alertManager, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "alerts")
				assert.Contains(t, response, "count")
				assert.Contains(t, response, "limit")
			},
		},
		{
			name:        "get alert history with invalid severity",
			endpoint:    "/alerts/history",
			queryParams: "?severity=invalid",
			setupHandler: func() *MonitoringHandler {
				logger := utils.NewLogger("test")
				alertManager := monitoring.NewAlertManager(logger)
				return NewMonitoringHandler(nil, nil, logger, nil, alertManager, nil)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupHandler()
			
			router := gin.New()
			router.GET(tt.endpoint, func(c *gin.Context) {
				switch tt.endpoint {
				case "/alerts/active":
					handler.GetActiveAlerts(c)
				case "/alerts/history":
					handler.GetAlertHistory(c)
				}
			})
			
			url := tt.endpoint + tt.queryParams
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestMonitoringHandler_MetricsIntegration tests metrics integration
func TestMonitoringHandler_MetricsIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	logger := utils.NewLogger("test")
	metricsRecorder := monitoring.NewMetricsRecorder()
	alertManager := monitoring.NewAlertManager(logger)
	
	// Add test notifier to capture alerts
	testNotifier := &TestNotifier{alerts: make([]*monitoring.Alert, 0)}
	alertManager.AddNotifier(testNotifier)
	
	handler := NewMonitoringHandler(nil, nil, logger, nil, alertManager, metricsRecorder)
	
	// Record some metrics that should trigger alerts
	metricsRecorder.RecordWebhookDeliveryError("tenant-1", "endpoint-1", "timeout", "408")
	metricsRecorder.RecordWebhookDeliveryError("tenant-1", "endpoint-1", "timeout", "408")
	metricsRecorder.RecordWebhookDeliveryError("tenant-1", "endpoint-1", "timeout", "408")
	
	// Simulate high failure rate
	alertManager.EvaluateMetric("webhook_delivery_failure_rate", 0.1, map[string]string{
		"tenant_id":   "tenant-1",
		"endpoint_id": "endpoint-1",
	})
	
	// Wait for alert processing
	time.Sleep(100 * time.Millisecond)
	
	// Check that alerts were generated
	router := gin.New()
	router.GET("/alerts/active", handler.GetActiveAlerts)
	
	req, _ := http.NewRequest("GET", "/alerts/active", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	count, ok := response["count"].(float64)
	assert.True(t, ok)
	assert.Greater(t, int(count), 0, "Expected active alerts")
}

// TestNotifier for testing alert notifications
type TestNotifier struct {
	alerts []*monitoring.Alert
}

func (tn *TestNotifier) SendAlert(ctx context.Context, alert *monitoring.Alert) error {
	tn.alerts = append(tn.alerts, alert)
	return nil
}

func (tn *TestNotifier) GetName() string {
	return "test"
}