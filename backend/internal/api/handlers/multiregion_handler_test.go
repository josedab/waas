package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/multiregion"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// --- Mock multiregion Repository ---

var errMRNotFound = fmt.Errorf("not found")

type mockMultiRegionRepo struct {
	regions     map[string]*multiregion.Region
	failovers   []*multiregion.FailoverEvent
	policies    map[string]*multiregion.RoutingPolicy
	replConfigs map[string]*multiregion.ReplicationConfig
}

func newMockMRRepo() *mockMultiRegionRepo {
	return &mockMultiRegionRepo{
		regions:     make(map[string]*multiregion.Region),
		policies:    make(map[string]*multiregion.RoutingPolicy),
		replConfigs: make(map[string]*multiregion.ReplicationConfig),
	}
}

func (m *mockMultiRegionRepo) CreateRegion(_ context.Context, region *multiregion.Region) error {
	m.regions[region.ID] = region
	return nil
}
func (m *mockMultiRegionRepo) GetRegion(_ context.Context, id string) (*multiregion.Region, error) {
	r, ok := m.regions[id]
	if !ok {
		return nil, errMRNotFound
	}
	return r, nil
}
func (m *mockMultiRegionRepo) GetRegionByCode(_ context.Context, code string) (*multiregion.Region, error) {
	for _, r := range m.regions {
		if r.Code == code {
			return r, nil
		}
	}
	return nil, errMRNotFound
}
func (m *mockMultiRegionRepo) ListRegions(_ context.Context) ([]*multiregion.Region, error) {
	var result []*multiregion.Region
	for _, r := range m.regions {
		result = append(result, r)
	}
	return result, nil
}
func (m *mockMultiRegionRepo) ListActiveRegions(_ context.Context) ([]*multiregion.Region, error) {
	var result []*multiregion.Region
	for _, r := range m.regions {
		if r.IsActive {
			result = append(result, r)
		}
	}
	return result, nil
}
func (m *mockMultiRegionRepo) UpdateRegion(_ context.Context, region *multiregion.Region) error {
	m.regions[region.ID] = region
	return nil
}
func (m *mockMultiRegionRepo) DeleteRegion(_ context.Context, id string) error {
	delete(m.regions, id)
	return nil
}
func (m *mockMultiRegionRepo) RecordRegionHealth(_ context.Context, health *multiregion.RegionHealth) error {
	return nil
}
func (m *mockMultiRegionRepo) GetRegionHealth(_ context.Context, regionID string) (*multiregion.RegionHealth, error) {
	return nil, nil
}
func (m *mockMultiRegionRepo) GetAllRegionHealth(_ context.Context) ([]*multiregion.RegionHealth, error) {
	return nil, nil
}
func (m *mockMultiRegionRepo) CreateFailoverEvent(_ context.Context, event *multiregion.FailoverEvent) error {
	m.failovers = append(m.failovers, event)
	return nil
}
func (m *mockMultiRegionRepo) UpdateFailoverEvent(_ context.Context, event *multiregion.FailoverEvent) error {
	return nil
}
func (m *mockMultiRegionRepo) GetFailoverEvent(_ context.Context, id string) (*multiregion.FailoverEvent, error) {
	for _, f := range m.failovers {
		if f.ID == id {
			return f, nil
		}
	}
	return nil, errMRNotFound
}
func (m *mockMultiRegionRepo) ListFailoverEvents(_ context.Context, limit int) ([]*multiregion.FailoverEvent, error) {
	if limit > len(m.failovers) {
		return m.failovers, nil
	}
	return m.failovers[:limit], nil
}
func (m *mockMultiRegionRepo) CreateRoutingPolicy(_ context.Context, policy *multiregion.RoutingPolicy) error {
	m.policies[policy.TenantID] = policy
	return nil
}
func (m *mockMultiRegionRepo) GetRoutingPolicy(_ context.Context, tenantID string) (*multiregion.RoutingPolicy, error) {
	p, ok := m.policies[tenantID]
	if !ok {
		return nil, errMRNotFound
	}
	return p, nil
}
func (m *mockMultiRegionRepo) UpdateRoutingPolicy(_ context.Context, policy *multiregion.RoutingPolicy) error {
	m.policies[policy.TenantID] = policy
	return nil
}
func (m *mockMultiRegionRepo) DeleteRoutingPolicy(_ context.Context, tenantID string) error {
	delete(m.policies, tenantID)
	return nil
}
func (m *mockMultiRegionRepo) CreateReplicationConfig(_ context.Context, config *multiregion.ReplicationConfig) error {
	m.replConfigs[config.ID] = config
	return nil
}
func (m *mockMultiRegionRepo) GetReplicationConfig(_ context.Context, id string) (*multiregion.ReplicationConfig, error) {
	c, ok := m.replConfigs[id]
	if !ok {
		return nil, errMRNotFound
	}
	return c, nil
}
func (m *mockMultiRegionRepo) GetReplicationConfigByRegions(_ context.Context, source, target string) (*multiregion.ReplicationConfig, error) {
	return nil, errMRNotFound
}
func (m *mockMultiRegionRepo) ListReplicationConfigs(_ context.Context) ([]*multiregion.ReplicationConfig, error) {
	var result []*multiregion.ReplicationConfig
	for _, c := range m.replConfigs {
		result = append(result, c)
	}
	return result, nil
}
func (m *mockMultiRegionRepo) UpdateReplicationConfig(_ context.Context, config *multiregion.ReplicationConfig) error {
	m.replConfigs[config.ID] = config
	return nil
}

func setupMultiRegionTest() (*MultiRegionHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := &MultiRegionHandler{logger: logger}
	router := gin.New()
	return handler, router
}

func setupMultiRegionWithMocks() (*MultiRegionHandler, *gin.Engine, *mockMultiRegionRepo) {
	gin.SetMode(gin.TestMode)
	repo := newMockMRRepo()
	logger := utils.NewLogger("test")
	hc := multiregion.NewHealthChecker(repo, multiregion.DefaultHealthConfig())
	handler := &MultiRegionHandler{
		repo:          repo,
		healthChecker: hc,
		logger:        logger,
	}
	router := gin.New()
	return handler, router, repo
}

// --- Error-path tests (existing) ---

func TestMultiRegionHandler_CreateRegion_InvalidBody(t *testing.T) {
	handler, router := setupMultiRegionTest()
	router.POST("/regions", handler.CreateRegion)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/regions", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMultiRegionHandler_TriggerFailover_InvalidBody(t *testing.T) {
	handler, router := setupMultiRegionTest()
	router.POST("/failover", handler.TriggerFailover)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/failover", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMultiRegionHandler_GetRoutingPolicy_Unauthorized(t *testing.T) {
	handler, router := setupMultiRegionTest()
	router.GET("/routing-policies/:id", handler.GetRoutingPolicy)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/routing-policies/test-id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Happy-path tests ---

func TestMultiRegionHandler_ListRegions_Empty(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.GET("/regions", handler.ListRegions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/regions", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiRegionHandler_ListRegions_WithRegions(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.regions["r1"] = &multiregion.Region{ID: "r1", Name: "US East", Code: "us-east-1", IsActive: true}
	repo.regions["r2"] = &multiregion.Region{ID: "r2", Name: "EU West", Code: "eu-west-1", IsActive: true}
	router.GET("/regions", handler.ListRegions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/regions", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var regions []*multiregion.Region
	json.Unmarshal(w.Body.Bytes(), &regions)
	assert.Len(t, regions, 2)
}

func TestMultiRegionHandler_GetRegion_Valid(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.regions["r1"] = &multiregion.Region{ID: "r1", Name: "US East", Code: "us-east-1"}
	router.GET("/regions/:id", handler.GetRegion)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/regions/r1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var region multiregion.Region
	json.Unmarshal(w.Body.Bytes(), &region)
	assert.Equal(t, "US East", region.Name)
}

func TestMultiRegionHandler_GetRegion_NotFound(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.GET("/regions/:id", handler.GetRegion)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/regions/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMultiRegionHandler_CreateRegion_Valid(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.POST("/regions", handler.CreateRegion)

	body, _ := json.Marshal(CreateRegionRequest{
		Name:     "US East",
		Code:     "us-east-1",
		Endpoint: "https://us-east.example.com",
		Priority: 1,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/regions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var region multiregion.Region
	json.Unmarshal(w.Body.Bytes(), &region)
	assert.Equal(t, "US East", region.Name)
	assert.True(t, region.IsActive)
}

func TestMultiRegionHandler_UpdateRegion_Valid(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.regions["r1"] = &multiregion.Region{ID: "r1", Name: "Old", Code: "us-east-1"}
	router.PATCH("/regions/:id", handler.UpdateRegion)

	body, _ := json.Marshal(UpdateRegionRequest{Name: "Updated"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/regions/r1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var region multiregion.Region
	json.Unmarshal(w.Body.Bytes(), &region)
	assert.Equal(t, "Updated", region.Name)
}

func TestMultiRegionHandler_UpdateRegion_NotFound(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.PATCH("/regions/:id", handler.UpdateRegion)

	body, _ := json.Marshal(UpdateRegionRequest{Name: "X"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/regions/nope", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMultiRegionHandler_DeleteRegion_Valid(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.regions["r1"] = &multiregion.Region{ID: "r1"}
	router.DELETE("/regions/:id", handler.DeleteRegion)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/regions/r1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestMultiRegionHandler_GetRegionHealth(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.GET("/regions/health", handler.GetRegionHealth)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/regions/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiRegionHandler_GetSingleRegionHealth_NotFound(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.GET("/regions/:id/health", handler.GetSingleRegionHealth)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/regions/r1/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMultiRegionHandler_ListFailoverEvents(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.failovers = []*multiregion.FailoverEvent{
		{ID: "f1", FromRegion: "r1", ToRegion: "r2"},
	}
	router.GET("/failover/events", handler.ListFailoverEvents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/failover/events", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiRegionHandler_GetFailoverEvent_Valid(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.failovers = []*multiregion.FailoverEvent{
		{ID: "f1", FromRegion: "r1", ToRegion: "r2"},
	}
	router.GET("/failover/events/:id", handler.GetFailoverEvent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/failover/events/f1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiRegionHandler_GetFailoverEvent_NotFound(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.GET("/failover/events/:id", handler.GetFailoverEvent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/failover/events/nope", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMultiRegionHandler_CreateRoutingPolicy_Valid(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.POST("/routing/policy", func(c *gin.Context) {
		c.Set("tenant_id", "tenant-1")
		c.Next()
	}, handler.CreateRoutingPolicy)

	body, _ := json.Marshal(CreateRoutingPolicyRequest{
		PolicyType:    "primary_backup",
		PrimaryRegion: "us-east-1",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/routing/policy", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var policy multiregion.RoutingPolicy
	json.Unmarshal(w.Body.Bytes(), &policy)
	assert.Equal(t, "tenant-1", policy.TenantID)
	assert.True(t, policy.Enabled)
}

func TestMultiRegionHandler_GetRoutingPolicy_Valid(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.policies["tenant-1"] = &multiregion.RoutingPolicy{
		TenantID: "tenant-1", PolicyType: multiregion.RoutingTypePrimaryBackup,
	}
	router.GET("/routing/policy", func(c *gin.Context) {
		c.Set("tenant_id", "tenant-1")
		c.Next()
	}, handler.GetRoutingPolicy)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/routing/policy", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiRegionHandler_UpdateRoutingPolicy_Valid(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.policies["tenant-1"] = &multiregion.RoutingPolicy{
		TenantID: "tenant-1", PolicyType: multiregion.RoutingTypePrimaryBackup,
	}
	router.PATCH("/routing/policy", func(c *gin.Context) {
		c.Set("tenant_id", "tenant-1")
		c.Next()
	}, handler.UpdateRoutingPolicy)

	body, _ := json.Marshal(CreateRoutingPolicyRequest{
		PolicyType:    "weighted",
		PrimaryRegion: "eu-west-1",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/routing/policy", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiRegionHandler_DeleteRoutingPolicy(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.policies["tenant-1"] = &multiregion.RoutingPolicy{TenantID: "tenant-1"}
	router.DELETE("/routing/policy", func(c *gin.Context) {
		c.Set("tenant_id", "tenant-1")
		c.Next()
	}, handler.DeleteRoutingPolicy)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/routing/policy", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestMultiRegionHandler_ListReplicationConfigs(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.replConfigs["rc1"] = &multiregion.ReplicationConfig{ID: "rc1", SourceRegion: "r1", TargetRegion: "r2"}
	router.GET("/replication/configs", handler.ListReplicationConfigs)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/replication/configs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiRegionHandler_CreateReplicationConfig_Valid(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.POST("/replication/configs", handler.CreateReplicationConfig)

	body, _ := json.Marshal(CreateReplicationConfigRequest{
		SourceRegion: "us-east-1",
		TargetRegion: "eu-west-1",
		Mode:         "async",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/replication/configs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var config multiregion.ReplicationConfig
	json.Unmarshal(w.Body.Bytes(), &config)
	assert.Equal(t, int64(1000), config.LagThresholdMs) // default
	assert.Equal(t, 30, config.RetentionDays)            // default
	assert.True(t, config.Enabled)
}

func TestMultiRegionHandler_GetReplicationConfig_Valid(t *testing.T) {
	handler, router, repo := setupMultiRegionWithMocks()
	repo.replConfigs["rc1"] = &multiregion.ReplicationConfig{ID: "rc1", SourceRegion: "r1"}
	router.GET("/replication/configs/:id", handler.GetReplicationConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/replication/configs/rc1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiRegionHandler_GetReplicationConfig_NotFound(t *testing.T) {
	handler, router, _ := setupMultiRegionWithMocks()
	router.GET("/replication/configs/:id", handler.GetReplicationConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/replication/configs/nope", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}