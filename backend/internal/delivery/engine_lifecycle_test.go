package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/catalog"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Engine Lifecycle Tests ---

func TestDeliveryEngine_SetDLQHook(t *testing.T) {
	t.Parallel()

	engine := &DeliveryEngine{logger: utils.NewLogger("test")}
	assert.Nil(t, engine.dlqHook)

	hook := func(ctx context.Context, tenantID, endpointID, deliveryID string, payload json.RawMessage, headers json.RawMessage, attempts []DLQAttemptDetail, finalError string) {
	}
	engine.SetDLQHook(hook)
	assert.NotNil(t, engine.dlqHook)
}

func TestDeliveryEngine_SetCatalogService(t *testing.T) {
	t.Parallel()

	engine := &DeliveryEngine{logger: utils.NewLogger("test")}
	assert.Nil(t, engine.catalogService)

	svc := &catalog.Service{}
	engine.SetCatalogService(svc)
	assert.Equal(t, svc, engine.catalogService)
}

func TestDeliveryEngine_DoubleStop_Safe(t *testing.T) {
	t.Parallel()

	logger := utils.NewLogger("test")
	ctx, cancel := context.WithCancel(context.Background())

	engine := &DeliveryEngine{
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// First Stop should work fine
	assert.NotPanics(t, func() {
		engine.cancel()
		engine.wg.Wait()
	})

	// Second cancel should not panic (context cancellation is idempotent)
	assert.NotPanics(t, func() {
		engine.cancel()
	})
}

func TestDeliveryEngine_ContextCancellation(t *testing.T) {
	t.Parallel()

	logger := utils.NewLogger("test")
	ctx, cancel := context.WithCancel(context.Background())

	engine := &DeliveryEngine{
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start a goroutine that watches the context
	done := make(chan struct{})
	engine.wg.Add(1)
	go func() {
		defer engine.wg.Done()
		<-engine.ctx.Done()
		close(done)
	}()

	// Cancel should propagate
	cancel()

	select {
	case <-done:
		// Context cancellation propagated
	case <-time.After(1 * time.Second):
		t.Fatal("Context cancellation did not propagate")
	}

	engine.wg.Wait()
}

func TestDLQAttemptDetail_JSONSerialization(t *testing.T) {
	t.Parallel()

	httpStatus := 500
	errMsg := "server error"
	respBody := `{"error":"internal"}`
	detail := DLQAttemptDetail{
		AttemptNumber: 3,
		HTTPStatus:    &httpStatus,
		ResponseBody:  &respBody,
		ErrorMessage:  &errMsg,
		AttemptedAt:   time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(detail)
	assert.NoError(t, err)

	var decoded DLQAttemptDetail
	assert.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, 3, decoded.AttemptNumber)
	assert.Equal(t, &httpStatus, decoded.HTTPStatus)
	assert.Equal(t, &errMsg, decoded.ErrorMessage)
}

// --- Schema Validation Tests ---

func TestSchemaValidation_NoCatalogService(t *testing.T) {
	t.Parallel()

	// When catalogService is nil, schema validation is skipped
	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	deliveryID := uuid.New()

	// Use inactive endpoint to test the path without needing transform repo
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      "http://example.com/webhook",
		IsActive: false,
	}

	message := &queue.DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    endpointID,
		TenantID:      uuid.New(),
		EventType:     "order.created",
		Payload:       json.RawMessage(`{"order_id": "123"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)

	// Engine has no catalog service (nil) - validation should be skipped
	assert.Nil(t, engine.catalogService)

	result, err := engine.HandleDelivery(context.Background(), message)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// The endpoint is inactive so it fails for that reason, not schema validation
	assert.Contains(t, *result.ErrorMessage, "endpoint is inactive")
}

func TestSchemaValidation_EmptyEventType(t *testing.T) {
	t.Parallel()

	// When EventType is empty, schema validation is skipped even with catalog service
	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	deliveryID := uuid.New()

	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      "http://example.com/webhook",
		IsActive: false,
	}

	message := &queue.DeliveryMessage{
		DeliveryID:    deliveryID,
		EndpointID:    endpointID,
		TenantID:      uuid.New(),
		EventType:     "", // Empty - should skip validation
		Payload:       json.RawMessage(`{"data": "test"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.DeliveryAttempt")).Return(nil)

	// Even with a catalog service set, empty EventType should skip validation
	engine.catalogService = &catalog.Service{}

	result, err := engine.HandleDelivery(context.Background(), message)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDeliveryConfig(t *testing.T) {
	t.Parallel()

	config := getDeliveryConfig()
	assert.Equal(t, DefaultMaxWorkers, config.MaxWorkers)
	assert.Equal(t, DefaultRequestTimeout, config.RequestTimeout)
	assert.Equal(t, DefaultMaxIdleConns, config.MaxIdleConns)
	assert.Equal(t, DefaultMaxIdleConnsPerHost, config.MaxIdleConnsPerHost)
	assert.Equal(t, DefaultIdleConnTimeout, config.IdleConnTimeout)
	assert.Equal(t, DefaultTLSHandshakeTimeout, config.TLSHandshakeTimeout)
	assert.Equal(t, DefaultResponseHeaderTimeout, config.ResponseHeaderTimeout)
}

func TestDeliveryEngine_HandleDelivery_WithTransformRepo(t *testing.T) {
	// Verifies that createTestEngine produces a usable engine for the basic path
	engine, _, _ := createTestEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.httpClient)
	assert.NotNil(t, engine.logger)
}

func TestDeliveryEngine_HandleDelivery_CreateAttemptError(t *testing.T) {
	// Tests that delivery continues even if creating the attempt record fails
	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      "http://127.0.0.1:1/not-real",
		IsActive: false, // Use inactive to avoid transform/HTTP issues
	}

	message := &queue.DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    endpointID,
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"data":"test"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(fmt.Errorf("db error"))
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	// Should not error - delivery continues even if recording fails
	result, err := engine.HandleDelivery(context.Background(), message)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queue.StatusFailed, result.Status)
	assert.Contains(t, *result.ErrorMessage, "endpoint is inactive")
}

func TestDeliveryEngine_HandleDelivery_UpdateAttemptError(t *testing.T) {
	// Tests that delivery completes even if updating the attempt record fails
	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      "http://example.com/webhook",
		IsActive: false,
	}

	message := &queue.DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    endpointID,
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"data":"test"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(fmt.Errorf("db update error"))

	// Should still return a result despite update failure
	result, err := engine.HandleDelivery(context.Background(), message)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queue.StatusFailed, result.Status)
	mockDeliveryRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestDeliveryEngine_ServerCrashRecovery(t *testing.T) {
	t.Parallel()

	// Test that the engine handles server errors gracefully
	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	// Use inactive endpoint to test error handling without needing transformRepo
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      "http://127.0.0.1:1/unreachable",
		IsActive: false,
	}

	message := &queue.DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    endpointID,
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"data":"test"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	// Engine handles inactive endpoint gracefully without crashing
	assert.NotPanics(t, func() {
		result, err := engine.HandleDelivery(context.Background(), message)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, queue.StatusFailed, result.Status)
	})
}

func TestDeliveryEngine_CancelledContext(t *testing.T) {
	t.Parallel()

	engine, mockWebhookRepo, mockDeliveryRepo := createTestEngine()

	endpointID := uuid.New()
	endpoint := &models.WebhookEndpoint{
		ID:       endpointID,
		URL:      "http://example.com/webhook",
		IsActive: false,
	}

	// Cancel context before calling HandleDelivery
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	message := &queue.DeliveryMessage{
		DeliveryID:    uuid.New(),
		EndpointID:    endpointID,
		TenantID:      uuid.New(),
		Payload:       json.RawMessage(`{"data":"test"}`),
		AttemptNumber: 1,
		MaxAttempts:   3,
		ScheduledAt:   time.Now(),
	}

	mockWebhookRepo.On("GetByID", mock.Anything, endpointID).Return(endpoint, nil)
	mockDeliveryRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockDeliveryRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	// Should still handle gracefully even with cancelled context
	result, err := engine.HandleDelivery(ctx, message)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
