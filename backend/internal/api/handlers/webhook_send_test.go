package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"webhook-platform/pkg/models"
	"webhook-platform/pkg/queue"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendWebhookUnit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockWebhookRepo := NewMockWebhookRepository()
	mockDeliveryRepo := NewMockDeliveryAttemptRepository()
	mockPublisher := queue.NewTestPublisher()
	logger := utils.NewLogger("test")

	handler := NewWebhookHandler(mockWebhookRepo, mockDeliveryRepo, mockPublisher, logger)

	tenantID := uuid.New()
	endpointID := uuid.New()

	// Create test endpoint
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		TenantID: tenantID,
		URL:      "https://api.example.com/webhook",
		SecretHash: "secret-hash",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
	}

	mockWebhookRepo.Create(context.Background(), endpoint)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	router.POST("/webhooks/send", handler.SendWebhook)

	t.Run("Send webhook successfully", func(t *testing.T) {
		mockPublisher.Reset()

		payload := json.RawMessage(`{"event": "test", "data": "value"}`)
		requestBody := SendWebhookRequest{
			EndpointID: &endpointID,
			Payload:    payload,
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
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
		assert.Equal(t, endpointID, response.EndpointID)
		assert.Equal(t, queue.StatusPending, response.Status)

		// Verify message was published
		messages := mockPublisher.GetDeliveryMessages()
		assert.Len(t, messages, 1)
		assert.Equal(t, response.DeliveryID, messages[0].DeliveryID)
		assert.Equal(t, endpointID, messages[0].EndpointID)
		assert.Equal(t, tenantID, messages[0].TenantID)
	})

	t.Run("Send webhook with missing payload", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"endpoint_id": endpointID,
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
		assert.Equal(t, "INVALID_REQUEST", errorObj["code"])
	})

	t.Run("Send webhook without tenant context", func(t *testing.T) {
		// Create router without tenant context
		routerNoTenant := gin.New()
		routerNoTenant.POST("/webhooks/send", handler.SendWebhook)

		payload := json.RawMessage(`{"event": "test"}`)
		requestBody := SendWebhookRequest{
			EndpointID: &endpointID,
			Payload:    payload,
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/webhooks/send", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		routerNoTenant.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "UNAUTHORIZED", errorObj["code"])
	})
}

func TestBatchSendWebhookUnit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockWebhookRepo := NewMockWebhookRepository()
	mockDeliveryRepo := NewMockDeliveryAttemptRepository()
	mockPublisher := queue.NewTestPublisher()
	logger := utils.NewLogger("test")

	handler := NewWebhookHandler(mockWebhookRepo, mockDeliveryRepo, mockPublisher, logger)

	tenantID := uuid.New()
	endpoint1ID := uuid.New()
	endpoint2ID := uuid.New()

	// Create test endpoints
	endpoint1 := &models.WebhookEndpoint{
		ID:       endpoint1ID,
		TenantID: tenantID,
		URL:      "https://api1.example.com/webhook",
		SecretHash: "secret-hash-1",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
	}

	endpoint2 := &models.WebhookEndpoint{
		ID:       endpoint2ID,
		TenantID: tenantID,
		URL:      "https://api2.example.com/webhook",
		SecretHash: "secret-hash-2",
		IsActive: true,
		RetryConfig: models.RetryConfiguration{
			MaxAttempts:       3,
			InitialDelayMs:    500,
			MaxDelayMs:        60000,
			BackoffMultiplier: 2,
		},
	}

	mockWebhookRepo.Create(context.Background(), endpoint1)
	mockWebhookRepo.Create(context.Background(), endpoint2)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	router.POST("/webhooks/send/batch", handler.BatchSendWebhook)

	t.Run("Batch send to specific endpoints", func(t *testing.T) {
		mockPublisher.Reset()

		payload := json.RawMessage(`{"event": "batch_test", "data": "value"}`)
		requestBody := BatchSendWebhookRequest{
			EndpointIDs: []uuid.UUID{endpoint1ID, endpoint2ID},
			Payload:     payload,
			Headers: map[string]string{
				"X-Batch-Header": "batch-value",
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

		assert.Equal(t, 2, response.Total)
		assert.Equal(t, 2, response.Queued)
		assert.Equal(t, 0, response.Failed)
		assert.Len(t, response.Deliveries, 2)

		// Verify messages were published
		messages := mockPublisher.GetDeliveryMessages()
		assert.Len(t, messages, 2)
	})

	t.Run("Batch send to all active endpoints", func(t *testing.T) {
		mockPublisher.Reset()

		payload := json.RawMessage(`{"event": "broadcast", "data": "all"}`)
		requestBody := BatchSendWebhookRequest{
			Payload: payload,
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

		assert.Equal(t, 2, response.Total)
		assert.Equal(t, 2, response.Queued)
		assert.Equal(t, 0, response.Failed)
		assert.Len(t, response.Deliveries, 2)

		// Verify messages were published
		messages := mockPublisher.GetDeliveryMessages()
		assert.Len(t, messages, 2)
	})
}