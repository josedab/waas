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
	"webhook-platform/pkg/queue"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockWebhookRepository implements the WebhookEndpointRepository interface for testing
type MockWebhookRepository struct {
	endpoints map[uuid.UUID]*models.WebhookEndpoint
	tenants   map[uuid.UUID][]*models.WebhookEndpoint
}

func NewMockWebhookRepository() *MockWebhookRepository {
	return &MockWebhookRepository{
		endpoints: make(map[uuid.UUID]*models.WebhookEndpoint),
		tenants:   make(map[uuid.UUID][]*models.WebhookEndpoint),
	}
}

func (m *MockWebhookRepository) Create(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	if endpoint.ID == uuid.Nil {
		endpoint.ID = uuid.New()
	}
	m.endpoints[endpoint.ID] = endpoint
	m.tenants[endpoint.TenantID] = append(m.tenants[endpoint.TenantID], endpoint)
	return nil
}

func (m *MockWebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	endpoint, exists := m.endpoints[id]
	if !exists {
		return nil, fmt.Errorf("webhook endpoint not found")
	}
	return endpoint, nil
}

func (m *MockWebhookRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.WebhookEndpoint, error) {
	endpoints := m.tenants[tenantID]
	if offset >= len(endpoints) {
		return []*models.WebhookEndpoint{}, nil
	}
	
	end := offset + limit
	if end > len(endpoints) {
		end = len(endpoints)
	}
	
	return endpoints[offset:end], nil
}

func (m *MockWebhookRepository) GetActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	var active []*models.WebhookEndpoint
	for _, endpoint := range m.tenants[tenantID] {
		if endpoint.IsActive {
			active = append(active, endpoint)
		}
	}
	return active, nil
}

func (m *MockWebhookRepository) Update(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	if _, exists := m.endpoints[endpoint.ID]; !exists {
		return fmt.Errorf("webhook endpoint not found")
	}
	m.endpoints[endpoint.ID] = endpoint
	
	// Update in tenant list
	for i, ep := range m.tenants[endpoint.TenantID] {
		if ep.ID == endpoint.ID {
			m.tenants[endpoint.TenantID][i] = endpoint
			break
		}
	}
	return nil
}

func (m *MockWebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	endpoint, exists := m.endpoints[id]
	if !exists {
		return fmt.Errorf("webhook endpoint not found")
	}
	
	delete(m.endpoints, id)
	
	// Remove from tenant list
	tenantEndpoints := m.tenants[endpoint.TenantID]
	for i, ep := range tenantEndpoints {
		if ep.ID == id {
			m.tenants[endpoint.TenantID] = append(tenantEndpoints[:i], tenantEndpoints[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockWebhookRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	endpoint, exists := m.endpoints[id]
	if !exists {
		return fmt.Errorf("webhook endpoint not found")
	}
	endpoint.IsActive = active
	return nil
}

func (m *MockWebhookRepository) UpdateStatus(ctx context.Context, id uuid.UUID, active bool) error {
	return m.SetActive(ctx, id, active)
}

// SetEndpoint is a helper method for tests to set up mock data
func (m *MockWebhookRepository) SetEndpoint(id uuid.UUID, endpoint *models.WebhookEndpoint) {
	m.endpoints[id] = endpoint
	if m.tenants[endpoint.TenantID] == nil {
		m.tenants[endpoint.TenantID] = []*models.WebhookEndpoint{}
	}
	// Check if endpoint already exists in tenant list
	found := false
	for i, ep := range m.tenants[endpoint.TenantID] {
		if ep.ID == id {
			m.tenants[endpoint.TenantID][i] = endpoint
			found = true
			break
		}
	}
	if !found {
		m.tenants[endpoint.TenantID] = append(m.tenants[endpoint.TenantID], endpoint)
	}
}

// MockDeliveryAttemptRepository implements the DeliveryAttemptRepository interface for testing
type MockDeliveryAttemptRepository struct {
	attempts map[uuid.UUID]*models.DeliveryAttempt
}

func NewMockDeliveryAttemptRepository() *MockDeliveryAttemptRepository {
	return &MockDeliveryAttemptRepository{
		attempts: make(map[uuid.UUID]*models.DeliveryAttempt),
	}
}

func (m *MockDeliveryAttemptRepository) Create(ctx context.Context, attempt *models.DeliveryAttempt) error {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	attempt.CreatedAt = time.Now()
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *MockDeliveryAttemptRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DeliveryAttempt, error) {
	attempt, exists := m.attempts[id]
	if !exists {
		return nil, fmt.Errorf("delivery attempt not found")
	}
	return attempt, nil
}

func (m *MockDeliveryAttemptRepository) GetByEndpointID(ctx context.Context, endpointID uuid.UUID, limit, offset int) ([]*models.DeliveryAttempt, error) {
	var results []*models.DeliveryAttempt
	for _, attempt := range m.attempts {
		if attempt.EndpointID == endpointID {
			results = append(results, attempt)
		}
	}
	return results, nil
}

func (m *MockDeliveryAttemptRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	var results []*models.DeliveryAttempt
	for _, attempt := range m.attempts {
		if attempt.Status == status {
			results = append(results, attempt)
		}
	}
	return results, nil
}

func (m *MockDeliveryAttemptRepository) GetPendingDeliveries(ctx context.Context, limit int) ([]*models.DeliveryAttempt, error) {
	return m.GetByStatus(ctx, queue.StatusPending, limit, 0)
}

func (m *MockDeliveryAttemptRepository) Update(ctx context.Context, attempt *models.DeliveryAttempt) error {
	if _, exists := m.attempts[attempt.ID]; !exists {
		return fmt.Errorf("delivery attempt not found")
	}
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *MockDeliveryAttemptRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	attempt, exists := m.attempts[id]
	if !exists {
		return fmt.Errorf("delivery attempt not found")
	}
	attempt.Status = status
	return nil
}

func (m *MockDeliveryAttemptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.attempts, id)
	return nil
}

func (m *MockDeliveryAttemptRepository) GetDeliveryHistory(ctx context.Context, endpointID uuid.UUID, statuses []string, limit, offset int) ([]*models.DeliveryAttempt, error) {
	var results []*models.DeliveryAttempt
	for _, attempt := range m.attempts {
		if attempt.EndpointID == endpointID {
			if len(statuses) == 0 {
				results = append(results, attempt)
			} else {
				for _, status := range statuses {
					if attempt.Status == status {
						results = append(results, attempt)
						break
					}
				}
			}
		}
	}
	return results, nil
}

func (m *MockDeliveryAttemptRepository) GetDeliveryHistoryWithFilters(ctx context.Context, tenantID uuid.UUID, filters repository.DeliveryHistoryFilters, limit, offset int) ([]*models.DeliveryAttempt, int, error) {
	// This is a simplified implementation for testing
	var results []*models.DeliveryAttempt
	for _, attempt := range m.attempts {
		// For testing purposes, we'll just return all attempts
		// In a real implementation, this would filter by tenant, endpoints, statuses, and dates
		results = append(results, attempt)
	}
	return results, len(results), nil
}

func (m *MockDeliveryAttemptRepository) GetDeliveryAttemptsByDeliveryID(ctx context.Context, deliveryID uuid.UUID, tenantID uuid.UUID) ([]*models.DeliveryAttempt, error) {
	var results []*models.DeliveryAttempt
	for _, attempt := range m.attempts {
		if attempt.ID == deliveryID {
			results = append(results, attempt)
		}
	}
	return results, nil
}

// WebhookMockPublisher implements the PublisherInterface for testing
type WebhookMockPublisher struct {
	deliveryMessages []queue.DeliveryMessage
	delayedMessages  []queue.DeliveryMessage
	dlqMessages      []queue.DeliveryMessage
}

func NewWebhookMockPublisher() *WebhookMockPublisher {
	return &WebhookMockPublisher{
		deliveryMessages: make([]queue.DeliveryMessage, 0),
		delayedMessages:  make([]queue.DeliveryMessage, 0),
		dlqMessages:      make([]queue.DeliveryMessage, 0),
	}
}

func (m *WebhookMockPublisher) PublishDelivery(ctx context.Context, message *queue.DeliveryMessage) error {
	m.deliveryMessages = append(m.deliveryMessages, *message)
	return nil
}

func (m *WebhookMockPublisher) PublishDelayedDelivery(ctx context.Context, message *queue.DeliveryMessage, delay time.Duration) error {
	m.delayedMessages = append(m.delayedMessages, *message)
	return nil
}

func (m *WebhookMockPublisher) PublishToDeadLetter(ctx context.Context, message *queue.DeliveryMessage, reason string) error {
	m.dlqMessages = append(m.dlqMessages, *message)
	return nil
}

func (m *WebhookMockPublisher) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	switch queueName {
	case queue.DeliveryQueue:
		return int64(len(m.deliveryMessages)), nil
	case queue.RetryQueue:
		return int64(len(m.delayedMessages)), nil
	case queue.DeadLetterQueue:
		return int64(len(m.dlqMessages)), nil
	default:
		return 0, nil
	}
}

func (m *WebhookMockPublisher) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	return map[string]int64{
		queue.DeliveryQueue:    int64(len(m.deliveryMessages)),
		queue.RetryQueue:       int64(len(m.delayedMessages)),
		queue.DeadLetterQueue:  int64(len(m.dlqMessages)),
		queue.ProcessingQueue:  0,
	}, nil
}

func (m *WebhookMockPublisher) GetDeliveryMessages() []queue.DeliveryMessage {
	return m.deliveryMessages
}

func (m *WebhookMockPublisher) Reset() {
	m.deliveryMessages = make([]queue.DeliveryMessage, 0)
	m.delayedMessages = make([]queue.DeliveryMessage, 0)
	m.dlqMessages = make([]queue.DeliveryMessage, 0)
}

func TestWebhookEndpointCRUDIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup
	mockRepo := NewMockWebhookRepository()
	mockDeliveryRepo := NewMockDeliveryAttemptRepository()
	mockPublisher := NewWebhookMockPublisher()
	logger := utils.NewLogger("test")
	handler := NewWebhookHandler(mockRepo, mockDeliveryRepo, mockPublisher, logger)
	
	tenantID := uuid.New()
	
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	
	// Setup routes
	router.POST("/webhooks/endpoints", handler.CreateWebhookEndpoint)
	router.GET("/webhooks/endpoints", handler.GetWebhookEndpoints)
	router.GET("/webhooks/endpoints/:id", handler.GetWebhookEndpoint)
	router.PUT("/webhooks/endpoints/:id", handler.UpdateWebhookEndpoint)
	router.DELETE("/webhooks/endpoints/:id", handler.DeleteWebhookEndpoint)
	
	var createdEndpointID string
	
	// Test 1: Create webhook endpoint
	t.Run("Create webhook endpoint", func(t *testing.T) {
		requestBody := CreateWebhookEndpointRequest{
			URL: "https://api.example.com/webhook",
			CustomHeaders: map[string]string{
				"Authorization": "Bearer token123",
			},
		}
		
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)
		
		req, err := http.NewRequest("POST", "/webhooks/endpoints", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response WebhookEndpointResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.NotEmpty(t, response.ID)
		assert.Equal(t, requestBody.URL, response.URL)
		assert.NotEmpty(t, response.Secret)
		assert.True(t, response.IsActive)
		assert.Equal(t, requestBody.CustomHeaders, response.CustomHeaders)
		
		createdEndpointID = response.ID.String()
	})
	
	// Test 2: Get webhook endpoints list
	t.Run("Get webhook endpoints", func(t *testing.T) {
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
		assert.Equal(t, createdEndpointID, endpoint["id"])
		
		// Verify no secret is returned in list
		_, hasSecret := endpoint["secret"]
		assert.False(t, hasSecret)
	})
	
	// Test 3: Get specific webhook endpoint
	t.Run("Get specific webhook endpoint", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/webhooks/endpoints/"+createdEndpointID, nil)
		require.NoError(t, err)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response WebhookEndpointResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, createdEndpointID, response.ID.String())
		assert.Equal(t, "https://api.example.com/webhook", response.URL)
		
		// Verify no secret is returned
		assert.Empty(t, response.Secret)
	})
	
	// Test 4: Update webhook endpoint
	t.Run("Update webhook endpoint", func(t *testing.T) {
		updateRequest := UpdateWebhookEndpointRequest{
			URL:      stringPtr("https://updated.example.com/webhook"),
			IsActive: boolPtr(false),
		}
		
		body, err := json.Marshal(updateRequest)
		require.NoError(t, err)
		
		req, err := http.NewRequest("PUT", "/webhooks/endpoints/"+createdEndpointID, bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response WebhookEndpointResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, *updateRequest.URL, response.URL)
		assert.Equal(t, *updateRequest.IsActive, response.IsActive)
	})
	
	// Test 5: Delete webhook endpoint
	t.Run("Delete webhook endpoint", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", "/webhooks/endpoints/"+createdEndpointID, nil)
		require.NoError(t, err)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNoContent, w.Code)
		
		// Verify endpoint is deleted by trying to get it
		req, err = http.NewRequest("GET", "/webhooks/endpoints/"+createdEndpointID, nil)
		require.NoError(t, err)
		
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestWebhookEndpointValidationIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockRepo := NewMockWebhookRepository()
	mockDeliveryRepo := NewMockDeliveryAttemptRepository()
	mockPublisher := NewWebhookMockPublisher()
	logger := utils.NewLogger("test")
	handler := NewWebhookHandler(mockRepo, mockDeliveryRepo, mockPublisher, logger)
	
	tenantID := uuid.New()
	
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	router.POST("/webhooks/endpoints", handler.CreateWebhookEndpoint)
	
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
			name:           "HTTP URL not allowed",
			url:            "http://api.example.com/webhook",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
		{
			name:           "localhost not allowed",
			url:            "https://localhost:8080/webhook",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_URL",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := CreateWebhookEndpointRequest{
				URL: tt.url,
			}
			
			body, err := json.Marshal(requestBody)
			require.NoError(t, err)
			
			req, err := http.NewRequest("POST", "/webhooks/endpoints", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus != http.StatusCreated {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				errorObj := response["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorObj["code"])
			}
		})
	}
}
func TestWebhookSendingIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup
	mockRepo := NewMockWebhookRepository()
	mockDeliveryRepo := NewMockDeliveryAttemptRepository()
	mockPublisher := NewWebhookMockPublisher()
	logger := utils.NewLogger("test")
	handler := NewWebhookHandler(mockRepo, mockDeliveryRepo, mockPublisher, logger)
	
	tenantID := uuid.New()
	
	// Create test endpoints
	endpoint1 := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: tenantID,
		URL:      "https://api.example.com/webhook1",
		SecretHash: "secret1hash",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
	}
	
	endpoint2 := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: tenantID,
		URL:      "https://api.example.com/webhook2",
		SecretHash: "secret2hash",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    500,
			MaxDelayMs:        60000,
			BackoffMultiplier: 2,
		},
	}
	
	inactiveEndpoint := &models.WebhookEndpoint{
		ID:       uuid.New(),
		TenantID: tenantID,
		URL:      "https://api.example.com/webhook3",
		SecretHash: "secret3hash",
		IsActive: false,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
	}
	
	// Add endpoints to mock repository
	mockRepo.Create(context.Background(), endpoint1)
	mockRepo.Create(context.Background(), endpoint2)
	mockRepo.Create(context.Background(), inactiveEndpoint)
	
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	
	// Setup routes
	router.POST("/webhooks/send", handler.SendWebhook)
	router.POST("/webhooks/send/batch", handler.BatchSendWebhook)
	
	// Test 1: Send webhook to specific endpoint
	t.Run("Send webhook to specific endpoint", func(t *testing.T) {
		mockPublisher.Reset()
		
		payload := json.RawMessage(`{"event": "user.created", "user_id": "123"}`)
		requestBody := SendWebhookRequest{
			EndpointID: &endpoint1.ID,
			Payload:    payload,
			Headers: map[string]string{
				"X-Event-Type": "user.created",
			},
		}
		
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)
		
		req, err := http.NewRequest("POST", "/webhooks/send", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusAccepted, w.Code)
		
		var response SendWebhookResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.NotEmpty(t, response.DeliveryID)
		assert.Equal(t, endpoint1.ID, response.EndpointID)
		assert.Equal(t, queue.StatusPending, response.Status)
		
		// Verify message was published
		messages := mockPublisher.GetDeliveryMessages()
		assert.Len(t, messages, 1)
		assert.Equal(t, response.DeliveryID, messages[0].DeliveryID)
		assert.Equal(t, endpoint1.ID, messages[0].EndpointID)
		assert.Equal(t, tenantID, messages[0].TenantID)
		// Compare JSON content, not exact bytes (formatting may differ)
		var expectedPayload, actualPayload map[string]interface{}
		json.Unmarshal(payload, &expectedPayload)
		json.Unmarshal(messages[0].Payload, &actualPayload)
		assert.Equal(t, expectedPayload, actualPayload)
		assert.Equal(t, 1, messages[0].AttemptNumber)
		assert.Equal(t, 5, messages[0].MaxAttempts)
	})
	
	// Test 2: Send webhook to all active endpoints
	t.Run("Send webhook to all active endpoints", func(t *testing.T) {
		mockPublisher.Reset()
		
		payload := json.RawMessage(`{"event": "order.completed", "order_id": "456"}`)
		requestBody := SendWebhookRequest{
			Payload: payload,
			Headers: map[string]string{
				"X-Event-Type": "order.completed",
			},
		}
		
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)
		
		req, err := http.NewRequest("POST", "/webhooks/send", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusAccepted, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		deliveries := response["deliveries"].([]interface{})
		assert.Len(t, deliveries, 2) // Only active endpoints
		assert.Equal(t, float64(2), response["total"])
		
		// Verify messages were published for both active endpoints
		messages := mockPublisher.GetDeliveryMessages()
		assert.Len(t, messages, 2)
		
		endpointIDs := []uuid.UUID{messages[0].EndpointID, messages[1].EndpointID}
		assert.Contains(t, endpointIDs, endpoint1.ID)
		assert.Contains(t, endpointIDs, endpoint2.ID)
		assert.NotContains(t, endpointIDs, inactiveEndpoint.ID)
	})
	
	// Test 3: Batch send webhooks
	t.Run("Batch send webhooks", func(t *testing.T) {
		mockPublisher.Reset()
		
		payload := json.RawMessage(`{"event": "system.maintenance", "scheduled_at": "2024-01-01T00:00:00Z"}`)
		requestBody := BatchSendWebhookRequest{
			EndpointIDs: []uuid.UUID{endpoint1.ID, endpoint2.ID, inactiveEndpoint.ID},
			Payload:     payload,
			Headers: map[string]string{
				"X-Event-Type": "system.maintenance",
			},
		}
		
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)
		
		req, err := http.NewRequest("POST", "/webhooks/send/batch", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusAccepted, w.Code)
		
		var response BatchSendWebhookResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, 2, response.Total)   // Only 2 active endpoints processed
		assert.Equal(t, 2, response.Queued) // 2 active endpoints queued
		assert.Equal(t, 0, response.Failed) // No failures
		assert.Len(t, response.Deliveries, 2) // Only active endpoints in response
		
		// Verify messages were published only for active endpoints
		messages := mockPublisher.GetDeliveryMessages()
		assert.Len(t, messages, 2)
	})
	
	// Test 4: Send webhook with invalid JSON payload
	t.Run("Send webhook with invalid JSON payload", func(t *testing.T) {
		// Create a request with invalid JSON in the payload field by manually crafting the JSON
		requestBody := `{
			"endpoint_id": "` + endpoint1.ID.String() + `",
			"payload": invalid json content
		}`
		
		req, err := http.NewRequest("POST", "/webhooks/send", bytes.NewBufferString(requestBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// This should fail at the Gin binding level, not our validation
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_REQUEST", errorObj["code"])
	})
	
	// Test 5: Send webhook with payload too large
	t.Run("Send webhook with payload too large", func(t *testing.T) {
		// Create a large JSON payload (larger than 1MB)
		largeData := make([]byte, 1024*1024+100) // Definitely larger than 1MB
		for i := range largeData {
			largeData[i] = 'a'
		}
		
		largePayload := fmt.Sprintf(`{"data": "%s"}`, string(largeData))
		
		requestBody := SendWebhookRequest{
			EndpointID: &endpoint1.ID,
			Payload:    json.RawMessage(largePayload),
		}
		
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)
		
		req, err := http.NewRequest("POST", "/webhooks/send", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "PAYLOAD_TOO_LARGE", errorObj["code"])
	})
	
	// Test 6: Send webhook to non-existent endpoint
	t.Run("Send webhook to non-existent endpoint", func(t *testing.T) {
		nonExistentID := uuid.New()
		payload := json.RawMessage(`{"event": "test"}`)
		requestBody := SendWebhookRequest{
			EndpointID: &nonExistentID,
			Payload:    payload,
		}
		
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)
		
		req, err := http.NewRequest("POST", "/webhooks/send", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "ENDPOINT_NOT_FOUND", errorObj["code"])
	})
	
	// Test 7: Send webhook to inactive endpoint
	t.Run("Send webhook to inactive endpoint", func(t *testing.T) {
		payload := json.RawMessage(`{"event": "test"}`)
		requestBody := SendWebhookRequest{
			EndpointID: &inactiveEndpoint.ID,
			Payload:    payload,
		}
		
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)
		
		req, err := http.NewRequest("POST", "/webhooks/send", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "ENDPOINT_INACTIVE", errorObj["code"])
	})
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}