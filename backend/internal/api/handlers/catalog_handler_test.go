package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/catalog"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// mockCatalogRepo implements catalog.CatalogRepository in-memory.
type mockCatalogRepo struct {
	mu            sync.Mutex
	eventTypes    map[uuid.UUID]*catalog.EventType
	versions      map[uuid.UUID][]*catalog.EventVersion
	categories    map[uuid.UUID]*catalog.EventCategory
	subscriptions map[string]*catalog.EventSubscription // key: endpointID+eventTypeID
	docs          map[uuid.UUID][]*catalog.EventDocumentation
	validationCfg map[uuid.UUID]*catalog.SchemaValidationConfig
}

func newMockCatalogRepo() *mockCatalogRepo {
	return &mockCatalogRepo{
		eventTypes:    make(map[uuid.UUID]*catalog.EventType),
		versions:      make(map[uuid.UUID][]*catalog.EventVersion),
		categories:    make(map[uuid.UUID]*catalog.EventCategory),
		subscriptions: make(map[string]*catalog.EventSubscription),
		docs:          make(map[uuid.UUID][]*catalog.EventDocumentation),
		validationCfg: make(map[uuid.UUID]*catalog.SchemaValidationConfig),
	}
}

func (m *mockCatalogRepo) CreateEventType(_ context.Context, et *catalog.EventType) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	et.ID = uuid.New()
	m.eventTypes[et.ID] = et
	return nil
}

func (m *mockCatalogRepo) GetEventType(_ context.Context, id uuid.UUID) (*catalog.EventType, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	et, ok := m.eventTypes[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return et, nil
}

func (m *mockCatalogRepo) GetEventTypeBySlug(_ context.Context, tenantID uuid.UUID, slug string) (*catalog.EventType, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, et := range m.eventTypes {
		if et.TenantID == tenantID && et.Slug == slug {
			return et, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockCatalogRepo) GetEventTypeByName(_ context.Context, tenantID uuid.UUID, name string) (*catalog.EventType, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, et := range m.eventTypes {
		if et.TenantID == tenantID && et.Name == name {
			return et, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockCatalogRepo) UpdateEventType(_ context.Context, et *catalog.EventType) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventTypes[et.ID] = et
	return nil
}

func (m *mockCatalogRepo) DeleteEventType(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.eventTypes, id)
	return nil
}

func (m *mockCatalogRepo) SearchEventTypes(_ context.Context, params *catalog.CatalogSearchParams) (*catalog.CatalogSearchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []*catalog.EventType
	for _, et := range m.eventTypes {
		if et.TenantID == params.TenantID {
			results = append(results, et)
		}
	}
	return &catalog.CatalogSearchResult{EventTypes: results, Total: len(results)}, nil
}

func (m *mockCatalogRepo) CreateEventVersion(_ context.Context, ev *catalog.EventVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ev.ID = uuid.New()
	m.versions[ev.EventTypeID] = append(m.versions[ev.EventTypeID], ev)
	return nil
}

func (m *mockCatalogRepo) ListEventVersions(_ context.Context, eventTypeID uuid.UUID) ([]*catalog.EventVersion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.versions[eventTypeID], nil
}

func (m *mockCatalogRepo) CreateCategory(_ context.Context, cat *catalog.EventCategory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cat.ID = uuid.New()
	m.categories[cat.ID] = cat
	return nil
}

func (m *mockCatalogRepo) ListCategories(_ context.Context, tenantID uuid.UUID) ([]*catalog.EventCategory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []*catalog.EventCategory
	for _, c := range m.categories {
		if c.TenantID == tenantID {
			results = append(results, c)
		}
	}
	return results, nil
}

func (m *mockCatalogRepo) CreateSubscription(_ context.Context, sub *catalog.EventSubscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub.ID = uuid.New()
	key := sub.EndpointID.String() + ":" + sub.EventTypeID.String()
	m.subscriptions[key] = sub
	return nil
}

func (m *mockCatalogRepo) DeleteSubscription(_ context.Context, endpointID, eventTypeID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := endpointID.String() + ":" + eventTypeID.String()
	delete(m.subscriptions, key)
	return nil
}

func (m *mockCatalogRepo) ListEndpointSubscriptions(_ context.Context, endpointID uuid.UUID) ([]*catalog.EventSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []*catalog.EventSubscription
	for _, s := range m.subscriptions {
		if s.EndpointID == endpointID {
			results = append(results, s)
		}
	}
	return results, nil
}

func (m *mockCatalogRepo) ListEventTypeSubscriptions(_ context.Context, eventTypeID uuid.UUID) ([]*catalog.EventSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []*catalog.EventSubscription
	for _, s := range m.subscriptions {
		if s.EventTypeID == eventTypeID {
			results = append(results, s)
		}
	}
	return results, nil
}

func (m *mockCatalogRepo) GetSubscriberCount(_ context.Context, eventTypeID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, s := range m.subscriptions {
		if s.EventTypeID == eventTypeID {
			count++
		}
	}
	return count, nil
}

func (m *mockCatalogRepo) SaveDocumentation(_ context.Context, doc *catalog.EventDocumentation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc.ID = uuid.New()
	m.docs[doc.EventTypeID] = append(m.docs[doc.EventTypeID], doc)
	return nil
}

func (m *mockCatalogRepo) GetDocumentation(_ context.Context, eventTypeID uuid.UUID) ([]*catalog.EventDocumentation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.docs[eventTypeID], nil
}

func (m *mockCatalogRepo) GetSchemaValidationConfig(_ context.Context, tenantID uuid.UUID) (*catalog.SchemaValidationConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, ok := m.validationCfg[tenantID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return cfg, nil
}

func (m *mockCatalogRepo) SaveSchemaValidationConfig(_ context.Context, config *catalog.SchemaValidationConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validationCfg[config.TenantID] = config
	return nil
}

func setupCatalogHappyTest() (*CatalogHandler, *gin.Engine, *mockCatalogRepo, uuid.UUID) {
	gin.SetMode(gin.TestMode)
	repo := newMockCatalogRepo()
	svc := catalog.NewService(repo)
	logger := utils.NewLogger("test")
	handler := NewCatalogHandler(svc, logger)
	router := gin.New()
	tenantID := uuid.New()
	return handler, router, repo, tenantID
}

// seedEventType inserts a pre-existing event type into the mock repo and returns its ID.
func seedEventType(repo *mockCatalogRepo, tenantID uuid.UUID) uuid.UUID {
	et := &catalog.EventType{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     "Order Created",
		Slug:     "order.created",
		Status:   catalog.StatusActive,
		Version:  "1.0.0",
	}
	repo.mu.Lock()
	repo.eventTypes[et.ID] = et
	repo.mu.Unlock()
	return et.ID
}

func setupCatalogTest() (*CatalogHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewCatalogHandler(nil, logger)
	router := gin.New()
	return handler, router
}

func TestCatalogHandler_CreateEventType_Unauthorized(t *testing.T) {
	handler, router := setupCatalogTest()
	router.POST("/event-types", handler.CreateEventType)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{"name": "test"})
	req, _ := http.NewRequest("POST", "/event-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCatalogHandler_GetEventType_InvalidID(t *testing.T) {
	handler, router := setupCatalogTest()
	router.GET("/event-types/:id", handler.GetEventType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/event-types/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCatalogHandler_SearchEventTypes_Unauthorized(t *testing.T) {
	handler, router := setupCatalogTest()
	router.GET("/event-types", handler.SearchEventTypes)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/event-types?q=test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCatalogHandler_DeleteEventType_InvalidID(t *testing.T) {
	handler, router := setupCatalogTest()
	router.DELETE("/event-types/:id", handler.DeleteEventType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/event-types/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCatalogHandler_ListCategories_Unauthorized(t *testing.T) {
	handler, router := setupCatalogTest()
	router.GET("/categories", handler.ListCategories)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/categories", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCatalogHandler_PublishVersion_InvalidID(t *testing.T) {
	handler, router := setupCatalogTest()
	router.POST("/event-types/:id/versions", handler.PublishVersion)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/event-types/not-a-uuid/versions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Happy-path tests ---

func TestCatalogHandler_CreateEventType_Valid(t *testing.T) {
	handler, router, _, tenantID := setupCatalogHappyTest()
	router.POST("/events", func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	}, handler.CreateEventType)

	body, _ := json.Marshal(map[string]interface{}{"name": "User Signed Up", "description": "Fires when a user signs up"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "User Signed Up", resp["name"])
	assert.NotEmpty(t, resp["id"])
}

func TestCatalogHandler_GetEventTypeBySlug_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	seedEventType(repo, tenantID)

	router.GET("/events/slug/:slug", func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	}, handler.GetEventTypeBySlug)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events/slug/order.created", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "order.created", resp["slug"])
}

func TestCatalogHandler_SearchEventTypes_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	seedEventType(repo, tenantID)

	router.GET("/events", func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	}, handler.SearchEventTypes)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events?q=order", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp catalog.CatalogSearchResult
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.GreaterOrEqual(t, resp.Total, 1)
}

func TestCatalogHandler_UpdateEventType_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	etID := seedEventType(repo, tenantID)

	router.PATCH("/events/:id", handler.UpdateEventType)

	body, _ := json.Marshal(map[string]interface{}{"description": "Updated description"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/events/"+etID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Updated description", resp["description"])
}

func TestCatalogHandler_DeleteEventType_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	etID := seedEventType(repo, tenantID)

	router.DELETE("/events/:id", handler.DeleteEventType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/events/"+etID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCatalogHandler_DeprecateEventType_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	etID := seedEventType(repo, tenantID)

	router.POST("/events/:id/deprecate", handler.DeprecateEventType)

	body, _ := json.Marshal(map[string]interface{}{"message": "Use order.placed instead"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/events/"+etID.String()+"/deprecate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "deprecated", resp["status"])
}

func TestCatalogHandler_PublishVersion_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	etID := seedEventType(repo, tenantID)

	router.POST("/events/:id/versions", handler.PublishVersion)

	body, _ := json.Marshal(map[string]interface{}{"version": "2.0.0", "changelog": "Breaking payload change", "is_breaking_change": true})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/events/"+etID.String()+"/versions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "2.0.0", resp["version"])
}

func TestCatalogHandler_GetVersions_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	etID := seedEventType(repo, tenantID)
	// Seed a version
	repo.versions[etID] = []*catalog.EventVersion{{ID: uuid.New(), EventTypeID: etID, Version: "1.0.0"}}

	router.GET("/events/:id/versions", handler.GetVersions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events/"+etID.String()+"/versions", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestCatalogHandler_CreateCategory_Valid(t *testing.T) {
	handler, router, _, tenantID := setupCatalogHappyTest()
	router.POST("/categories", func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	}, handler.CreateCategory)

	body, _ := json.Marshal(map[string]interface{}{"name": "Payments", "description": "Payment events"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/categories", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Payments", resp["name"])
}

func TestCatalogHandler_ListCategories_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	// Seed a category
	cat := &catalog.EventCategory{ID: uuid.New(), TenantID: tenantID, Name: "Billing", Slug: "billing"}
	repo.categories[cat.ID] = cat

	router.GET("/categories", func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	}, handler.ListCategories)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/categories", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestCatalogHandler_SubscribeToEvent_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	etID := seedEventType(repo, tenantID)
	endpointID := uuid.New()

	router.POST("/events/:id/subscribe", handler.SubscribeToEvent)

	body, _ := json.Marshal(map[string]interface{}{"endpoint_id": endpointID.String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/events/"+etID.String()+"/subscribe", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, endpointID.String(), resp["endpoint_id"])
}

func TestCatalogHandler_UnsubscribeFromEvent_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	etID := seedEventType(repo, tenantID)
	endpointID := uuid.New()
	// Seed subscription
	key := endpointID.String() + ":" + etID.String()
	repo.subscriptions[key] = &catalog.EventSubscription{ID: uuid.New(), EndpointID: endpointID, EventTypeID: etID, IsActive: true}

	router.DELETE("/events/:id/subscribe/:endpoint_id", handler.UnsubscribeFromEvent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/events/"+etID.String()+"/subscribe/"+endpointID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCatalogHandler_GetDocumentation_Valid(t *testing.T) {
	handler, router, repo, tenantID := setupCatalogHappyTest()
	etID := seedEventType(repo, tenantID)
	// Seed documentation
	repo.docs[etID] = []*catalog.EventDocumentation{{ID: uuid.New(), EventTypeID: etID, Section: "overview", Content: "Some docs"}}

	router.GET("/events/:id/docs", handler.GetDocumentation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events/"+etID.String()+"/docs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "Some docs", resp[0]["content"])
}
