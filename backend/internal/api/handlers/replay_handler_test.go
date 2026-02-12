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
	"github.com/josedab/waas/pkg/replay"
	"github.com/josedab/waas/pkg/utils"
)

// mockReplayRepo implements replay.Repository for testing
type mockReplayRepo struct {
	archives  map[string]*replay.DeliveryArchive
	snapshots map[string]*replay.Snapshot
	err       error
}

func newMockReplayRepo() *mockReplayRepo {
	return &mockReplayRepo{
		archives:  make(map[string]*replay.DeliveryArchive),
		snapshots: make(map[string]*replay.Snapshot),
	}
}

func (m *mockReplayRepo) GetDeliveryArchive(_ context.Context, tenantID, deliveryID string) (*replay.DeliveryArchive, error) {
	if m.err != nil {
		return nil, m.err
	}
	a, ok := m.archives[deliveryID]
	if !ok {
		return nil, nil
	}
	if a.TenantID != tenantID {
		return nil, nil
	}
	return a, nil
}

func (m *mockReplayRepo) ListDeliveryArchives(_ context.Context, tenantID string, _ *replay.BulkReplayRequest) ([]replay.DeliveryArchive, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	var result []replay.DeliveryArchive
	for _, a := range m.archives {
		if a.TenantID == tenantID {
			result = append(result, *a)
		}
	}
	return result, len(result), nil
}

func (m *mockReplayRepo) ArchiveDelivery(_ context.Context, archive *replay.DeliveryArchive) error {
	if m.err != nil {
		return m.err
	}
	m.archives[archive.ID] = archive
	return nil
}

func (m *mockReplayRepo) CreateSnapshot(_ context.Context, snapshot *replay.Snapshot, _ []string) error {
	if m.err != nil {
		return m.err
	}
	m.snapshots[snapshot.ID] = snapshot
	return nil
}

func (m *mockReplayRepo) GetSnapshot(_ context.Context, tenantID, snapshotID string) (*replay.Snapshot, error) {
	if m.err != nil {
		return nil, m.err
	}
	s, ok := m.snapshots[snapshotID]
	if !ok {
		return nil, nil
	}
	if s.TenantID != tenantID {
		return nil, nil
	}
	return s, nil
}

func (m *mockReplayRepo) ListSnapshots(_ context.Context, tenantID string, _, _ int) ([]replay.Snapshot, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	var result []replay.Snapshot
	for _, s := range m.snapshots {
		if s.TenantID == tenantID {
			result = append(result, *s)
		}
	}
	return result, len(result), nil
}

func (m *mockReplayRepo) GetSnapshotDeliveryIDs(_ context.Context, _ string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []string{}, nil
}

func (m *mockReplayRepo) DeleteSnapshot(_ context.Context, tenantID, snapshotID string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.snapshots, snapshotID)
	return nil
}

func (m *mockReplayRepo) CleanupExpiredSnapshots(_ context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return 0, nil
}

// mockReplayPublisher implements replay.DeliveryPublisher
type mockReplayPublisher struct {
	err error
}

func (m *mockReplayPublisher) Publish(_ context.Context, _, _ string, _ []byte, _ map[string]string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "new-delivery-id", nil
}

func setupReplayTestRouter(repo *mockReplayRepo, publisher *mockReplayPublisher) (*gin.Engine, *ReplayHandler) {
	gin.SetMode(gin.TestMode)
	service := replay.NewService(repo, publisher)
	logger := utils.NewLogger("test")
	handler := NewReplayHandler(service, logger)

	router := gin.New()
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		if tid := c.GetHeader("X-Tenant-ID"); tid != "" {
			c.Set("tenant_id", tid)
		}
		c.Next()
	})
	RegisterReplayRoutes(api, handler)
	return router, handler
}

// --- ListReplays tests ---

func TestListReplays_Success(t *testing.T) {
	repo := newMockReplayRepo()
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/replay", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
}

func TestListReplays_MissingTenantID(t *testing.T) {
	repo := newMockReplayRepo()
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/replay", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// ListReplays returns 200 with empty array regardless of tenant
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetReplay tests ---

func TestGetReplay_ReturnsNotImplemented(t *testing.T) {
	repo := newMockReplayRepo()
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/replay/some-id", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetReplay_NoTenant_StillReturnsNotImplemented(t *testing.T) {
	repo := newMockReplayRepo()
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/replay/some-id", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// GetReplay always returns 501 regardless of tenant
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ReplayDelivery tests ---

func TestReplayDelivery_Success(t *testing.T) {
	repo := newMockReplayRepo()
	repo.archives["del-1"] = &replay.DeliveryArchive{
		ID:          "del-1",
		TenantID:    "tenant-1",
		EndpointID:  "ep-1",
		EndpointURL: "https://example.com/webhook",
		Payload:     json.RawMessage(`{"event":"test"}`),
		Headers:     map[string]string{"Content-Type": "application/json"},
		Status:      "delivered",
		CreatedAt:   time.Now(),
	}
	publisher := &mockReplayPublisher{}
	router, _ := setupReplayTestRouter(repo, publisher)

	body := `{"delivery_id":"del-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/replay", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp replay.ReplayResult
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.OriginalDeliveryID != "del-1" {
		t.Errorf("expected original_delivery_id del-1, got %s", resp.OriginalDeliveryID)
	}
	if resp.NewDeliveryID != "new-delivery-id" {
		t.Errorf("expected new_delivery_id new-delivery-id, got %s", resp.NewDeliveryID)
	}
	if resp.Status != "queued" {
		t.Errorf("expected status queued, got %s", resp.Status)
	}
}

func TestReplayDelivery_MissingTenantID(t *testing.T) {
	repo := newMockReplayRepo()
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	body := `{"delivery_id":"del-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/replay", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReplayDelivery_InvalidJSON(t *testing.T) {
	repo := newMockReplayRepo()
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/replay", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReplayDelivery_MissingDeliveryID(t *testing.T) {
	repo := newMockReplayRepo()
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/replay", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReplayDelivery_DeliveryNotFound(t *testing.T) {
	repo := newMockReplayRepo()
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	body := `{"delivery_id":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/replay", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReplayDelivery_RepoError(t *testing.T) {
	repo := newMockReplayRepo()
	repo.err = errors.New("db error")
	router, _ := setupReplayTestRouter(repo, &mockReplayPublisher{})

	body := `{"delivery_id":"del-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/replay", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
