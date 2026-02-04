package quickstart

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryQueue_PublishDelivery(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	err := q.PublishDelivery(ctx, nil)
	assert.Error(t, err, "should reject nil message")

	stats, _ := q.GetQueueStats(ctx)
	assert.Equal(t, int64(0), stats["delivery"])
}

func TestMemoryQueue_GetQueueStats(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	stats, err := q.GetQueueStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats["delivery"])
	assert.Equal(t, int64(0), stats["retry"])
	assert.Equal(t, int64(0), stats["dlq"])
}

func TestMemoryQueue_GetQueueLength(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	length, err := q.GetQueueLength(ctx, "delivery")
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)

	length, err = q.GetQueueLength(ctx, "unknown")
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)
}

func TestMemoryQueue_Dequeue_Empty(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()

	msg, err := q.Dequeue(ctx)
	require.NoError(t, err)
	assert.Nil(t, msg)
}

func TestGetTutorialSteps(t *testing.T) {
	steps := GetTutorialSteps()
	assert.True(t, len(steps) >= 5, "should have at least 5 tutorial steps")
	assert.Equal(t, 1, steps[0].Step)
	assert.Equal(t, "Health Check", steps[0].Title)
}

func TestGettingStartedHandler(t *testing.T) {
	handler := GettingStartedHandler()
	req := httptest.NewRequest(http.MethodGet, "/getting-started", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "WaaS Getting Started Guide", response["title"])
	assert.NotNil(t, response["steps"])
}

func TestGettingStartedHTMLHandler(t *testing.T) {
	handler := GettingStartedHTMLHandler()
	req := httptest.NewRequest(http.MethodGet, "/getting-started", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "WaaS")
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, ModeQuickstart, cfg.Mode)
	assert.Equal(t, 8080, cfg.Port)
	assert.True(t, cfg.EnableTutorial)
	assert.True(t, cfg.InMemoryQueue)
}
