package pluginmarket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/httputil"
)

// Extended plugin type constants (beyond what exists in models.go)
const (
	PluginTypeEnricher  = "enricher"
	PluginTypeValidator = "validator"
)

// MarketplacePlugin represents a plugin in the extended marketplace
type MarketplacePlugin struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Slug         string            `json:"slug"`
	Description  string            `json:"description"`
	LongDesc     string            `json:"long_description,omitempty"`
	Version      string            `json:"version"`
	Author       MarketplaceAuthor `json:"author"`
	Type         string            `json:"type"`
	Category     string            `json:"category"`
	Tags         []string          `json:"tags,omitempty"`
	IconURL      string            `json:"icon_url,omitempty"`
	RepoURL      string            `json:"repo_url,omitempty"`
	DocsURL      string            `json:"docs_url,omitempty"`
	ConfigSchema json.RawMessage   `json:"config_schema,omitempty"`
	EntryPoint   string            `json:"entry_point"`
	Runtime      string            `json:"runtime"` // javascript, wasm, docker
	Permissions  []string          `json:"permissions,omitempty"`
	Stats        MarketplaceStats  `json:"stats"`
	Status       string            `json:"status"`
	IsOfficial   bool              `json:"is_official"`
	IsVerified   bool              `json:"is_verified"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// MarketplaceAuthor represents a plugin author
type MarketplaceAuthor struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	URL      string `json:"url,omitempty"`
	Verified bool   `json:"verified"`
}

// MarketplaceStats tracks plugin usage metrics
type MarketplaceStats struct {
	InstallCount   int64   `json:"install_count"`
	ActiveInstalls int64   `json:"active_installs"`
	Rating         float64 `json:"rating"`
	ReviewCount    int     `json:"review_count"`
	DownloadCount  int64   `json:"download_count"`
}

// MarketplaceReview represents a user review in the marketplace
type MarketplaceReview struct {
	ID        string    `json:"id"`
	PluginID  string    `json:"plugin_id"`
	TenantID  string    `json:"tenant_id"`
	Rating    int       `json:"rating"` // 1-5
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Helpful   int       `json:"helpful_count"`
	CreatedAt time.Time `json:"created_at"`
}

// MarketplaceInstall represents an installed plugin instance
type MarketplaceInstall struct {
	ID          string          `json:"id"`
	PluginID    string          `json:"plugin_id"`
	TenantID    string          `json:"tenant_id"`
	Version     string          `json:"version"`
	Config      json.RawMessage `json:"config,omitempty"`
	Status      string          `json:"status"` // active, paused, error
	SandboxID   string          `json:"sandbox_id,omitempty"`
	InstalledAt time.Time       `json:"installed_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// PluginExecution represents a sandboxed plugin execution
type PluginExecution struct {
	ID           string          `json:"id"`
	InstallID    string          `json:"install_id"`
	Input        json.RawMessage `json:"input"`
	Output       json.RawMessage `json:"output,omitempty"`
	Status       string          `json:"status"` // success, error, timeout
	DurationMs   int64           `json:"duration_ms"`
	MemoryUsedKB int64           `json:"memory_used_kb"`
	Error        string          `json:"error,omitempty"`
	ExecutedAt   time.Time       `json:"executed_at"`
}

// PluginSDKSpec defines the plugin development SDK specification
type PluginSDKSpec struct {
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Hooks       []PluginHookSpec `json:"hooks"`
	ConfigSpec  json.RawMessage  `json:"config_spec,omitempty"`
	Permissions []string         `json:"permissions"`
	Limits      PluginLimits     `json:"limits"`
}

// PluginHookSpec defines a hook point for plugins
type PluginHookSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputType   string `json:"input_type"`
	OutputType  string `json:"output_type"`
	Required    bool   `json:"required"`
}

// PluginLimits defines sandboxed execution limits
type PluginLimits struct {
	MaxExecutionMs int  `json:"max_execution_ms"`
	MaxMemoryKB    int  `json:"max_memory_kb"`
	MaxPayloadKB   int  `json:"max_payload_kb"`
	MaxNetworkReqs int  `json:"max_network_requests"`
	AllowNetwork   bool `json:"allow_network"`
}

// PluginSubmission represents a community plugin submission
type PluginSubmission struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	RepoURL     string            `json:"repo_url"`
	Author      MarketplaceAuthor `json:"author"`
	PluginType  string            `json:"plugin_type"`
	Status      string            `json:"status"`
	ReviewNotes string            `json:"review_notes,omitempty"`
	SubmittedAt time.Time         `json:"submitted_at"`
	ReviewedAt  *time.Time        `json:"reviewed_at,omitempty"`
}

// SearchPluginsRequest defines search parameters
type SearchPluginsRequest struct {
	Query    string `json:"query,omitempty" form:"q"`
	Type     string `json:"type,omitempty" form:"type"`
	Category string `json:"category,omitempty" form:"category"`
	Tag      string `json:"tag,omitempty" form:"tag"`
	Sort     string `json:"sort,omitempty" form:"sort"`
	Limit    int    `json:"limit,omitempty" form:"limit"`
	Offset   int    `json:"offset,omitempty" form:"offset"`
}

// InstallMarketplacePluginRequest requests plugin installation
type InstallMarketplacePluginRequest struct {
	PluginID string          `json:"plugin_id" binding:"required"`
	Config   json.RawMessage `json:"config,omitempty"`
}

// SubmitPluginRequest submits a community plugin
type SubmitPluginRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description" binding:"required"`
	RepoURL     string   `json:"repo_url" binding:"required"`
	PluginType  string   `json:"plugin_type" binding:"required"`
	Tags        []string `json:"tags,omitempty"`
}

// CreateMarketplaceReviewRequest creates a plugin review
type CreateMarketplaceReviewRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"`
	Title  string `json:"title" binding:"required"`
	Body   string `json:"body"`
}

// FullMarketplaceService manages the extended plugin marketplace
type FullMarketplaceService struct {
	plugins     map[string]*MarketplacePlugin
	installs    map[string][]*MarketplaceInstall
	reviews     map[string][]*MarketplaceReview
	submissions []*PluginSubmission
}

// NewFullMarketplaceService creates a new marketplace service
func NewFullMarketplaceService() *FullMarketplaceService {
	svc := &FullMarketplaceService{
		plugins:  make(map[string]*MarketplacePlugin),
		installs: make(map[string][]*MarketplaceInstall),
		reviews:  make(map[string][]*MarketplaceReview),
	}
	for _, p := range builtinMarketplacePlugins() {
		svc.plugins[p.ID] = p
	}
	return svc
}

// SearchPlugins searches the marketplace
func (s *FullMarketplaceService) SearchPlugins(ctx context.Context, req *SearchPluginsRequest) ([]MarketplacePlugin, int) {
	if req.Limit <= 0 {
		req.Limit = 20
	}
	var results []MarketplacePlugin
	for _, p := range s.plugins {
		if req.Type != "" && p.Type != req.Type {
			continue
		}
		if req.Category != "" && p.Category != req.Category {
			continue
		}
		if req.Query != "" {
			q := strings.ToLower(req.Query)
			if !strings.Contains(strings.ToLower(p.Name), q) && !strings.Contains(strings.ToLower(p.Description), q) {
				continue
			}
		}
		results = append(results, *p)
	}
	switch req.Sort {
	case "popular":
		sort.Slice(results, func(i, j int) bool { return results[i].Stats.InstallCount > results[j].Stats.InstallCount })
	case "rating":
		sort.Slice(results, func(i, j int) bool { return results[i].Stats.Rating > results[j].Stats.Rating })
	default:
		sort.Slice(results, func(i, j int) bool { return results[i].CreatedAt.After(results[j].CreatedAt) })
	}
	total := len(results)
	if req.Offset < len(results) {
		results = results[req.Offset:]
	}
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}
	return results, total
}

// GetPlugin retrieves a plugin by ID
func (s *FullMarketplaceService) GetPlugin(ctx context.Context, pluginID string) (*MarketplacePlugin, error) {
	p, ok := s.plugins[pluginID]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", pluginID)
	}
	return p, nil
}

// InstallPlugin installs a plugin for a tenant
func (s *FullMarketplaceService) InstallPlugin(ctx context.Context, tenantID string, req *InstallMarketplacePluginRequest) (*MarketplaceInstall, error) {
	plugin, ok := s.plugins[req.PluginID]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", req.PluginID)
	}
	install := &MarketplaceInstall{
		ID: uuid.New().String(), PluginID: req.PluginID, TenantID: tenantID,
		Version: plugin.Version, Config: req.Config, Status: "active",
		SandboxID: "sandbox-" + uuid.New().String()[:8], InstalledAt: time.Now(), UpdatedAt: time.Now(),
	}
	s.installs[tenantID] = append(s.installs[tenantID], install)
	plugin.Stats.InstallCount++
	return install, nil
}

// CreateReview adds a review for a plugin
func (s *FullMarketplaceService) CreateReview(ctx context.Context, tenantID, pluginID string, req *CreateMarketplaceReviewRequest) (*MarketplaceReview, error) {
	if _, ok := s.plugins[pluginID]; !ok {
		return nil, fmt.Errorf("plugin %q not found", pluginID)
	}
	review := &MarketplaceReview{ID: uuid.New().String(), PluginID: pluginID, TenantID: tenantID, Rating: req.Rating, Title: req.Title, Body: req.Body, CreatedAt: time.Now()}
	s.reviews[pluginID] = append(s.reviews[pluginID], review)
	plugin := s.plugins[pluginID]
	reviews := s.reviews[pluginID]
	var totalRating int
	for _, r := range reviews {
		totalRating += r.Rating
	}
	plugin.Stats.Rating = float64(totalRating) / float64(len(reviews))
	plugin.Stats.ReviewCount = len(reviews)
	return review, nil
}

// SubmitPlugin handles community plugin submission
func (s *FullMarketplaceService) SubmitPlugin(ctx context.Context, tenantID string, req *SubmitPluginRequest) (*PluginSubmission, error) {
	submission := &PluginSubmission{
		ID: uuid.New().String(), Name: req.Name, Description: req.Description,
		RepoURL: req.RepoURL, PluginType: req.PluginType,
		Author: MarketplaceAuthor{ID: tenantID}, Status: "submitted", SubmittedAt: time.Now(),
	}
	s.submissions = append(s.submissions, submission)
	return submission, nil
}

// GetPluginSDKSpec returns the SDK specification for plugin development
func (s *FullMarketplaceService) GetPluginSDKSpec() *PluginSDKSpec {
	return &PluginSDKSpec{
		Name: "WaaS Plugin SDK", Version: "1.0.0",
		Hooks: []PluginHookSpec{
			{Name: "onWebhookReceived", Description: "Called when a webhook is received", InputType: "WebhookEvent", OutputType: "WebhookEvent"},
			{Name: "onBeforeDelivery", Description: "Called before delivering to endpoint", InputType: "DeliveryRequest", OutputType: "DeliveryRequest"},
			{Name: "onAfterDelivery", Description: "Called after delivery attempt", InputType: "DeliveryResult", OutputType: "void"},
			{Name: "onTransform", Description: "Transform webhook payload", InputType: "Payload", OutputType: "Payload"},
			{Name: "onFilter", Description: "Filter/route webhooks", InputType: "WebhookEvent", OutputType: "FilterDecision"},
			{Name: "onError", Description: "Handle delivery errors", InputType: "DeliveryError", OutputType: "ErrorAction"},
		},
		Permissions: []string{"read:webhooks", "write:webhooks", "read:endpoints", "read:deliveries", "network:outbound"},
		Limits:      PluginLimits{MaxExecutionMs: 5000, MaxMemoryKB: 65536, MaxPayloadKB: 1024, MaxNetworkReqs: 10, AllowNetwork: true},
	}
}

func builtinMarketplacePlugins() []*MarketplacePlugin {
	now := time.Now()
	author := MarketplaceAuthor{ID: "waas-team", Name: "WaaS Team", Verified: true}
	return []*MarketplacePlugin{
		{ID: "plugin-slack-notify", Name: "Slack Notification", Slug: "slack-notify", Description: "Forward webhook events to Slack channels", Version: "1.2.0", Author: author, Type: "notifier", Category: "notifications", Tags: []string{"slack", "notifications"}, Runtime: "javascript", IsOfficial: true, IsVerified: true, Status: "published", Stats: MarketplaceStats{InstallCount: 15000, Rating: 4.8, ReviewCount: 234}, CreatedAt: now},
		{ID: "plugin-json-transform", Name: "JSON Transform", Slug: "json-transform", Description: "Transform webhook payloads using JSONPath expressions", Version: "2.0.0", Author: author, Type: "transformation", Category: "transforms", Tags: []string{"json", "transform"}, Runtime: "javascript", IsOfficial: true, IsVerified: true, Status: "published", Stats: MarketplaceStats{InstallCount: 12000, Rating: 4.7, ReviewCount: 189}, CreatedAt: now},
		{ID: "plugin-payload-validator", Name: "Payload Validator", Slug: "payload-validator", Description: "Validate webhook payloads against JSON schemas", Version: "1.5.0", Author: author, Type: "validator", Category: "validation", Tags: []string{"schema", "validation"}, Runtime: "javascript", IsOfficial: true, IsVerified: true, Status: "published", Stats: MarketplaceStats{InstallCount: 8000, Rating: 4.6, ReviewCount: 145}, CreatedAt: now},
		{ID: "plugin-retry-backoff", Name: "Smart Retry Backoff", Slug: "smart-retry", Description: "Intelligent retry with adaptive backoff", Version: "1.0.0", Author: author, Type: "router", Category: "delivery", Tags: []string{"retry", "backoff"}, Runtime: "javascript", IsOfficial: true, IsVerified: true, Status: "published", Stats: MarketplaceStats{InstallCount: 6000, Rating: 4.5, ReviewCount: 98}, CreatedAt: now},
		{ID: "plugin-pii-redactor", Name: "PII Redactor", Slug: "pii-redactor", Description: "Automatically redact PII from webhook payloads", Version: "1.3.0", Author: author, Type: "filter", Category: "security", Tags: []string{"pii", "security"}, Runtime: "wasm", IsOfficial: true, IsVerified: true, Status: "published", Stats: MarketplaceStats{InstallCount: 5500, Rating: 4.9, ReviewCount: 87}, CreatedAt: now},
		{ID: "plugin-dedup", Name: "Event Deduplicator", Slug: "dedup", Description: "Prevent duplicate webhook delivery", Version: "1.1.0", Author: author, Type: "filter", Category: "reliability", Tags: []string{"dedup", "idempotency"}, Runtime: "javascript", IsOfficial: true, IsVerified: true, Status: "published", Stats: MarketplaceStats{InstallCount: 4500, Rating: 4.4, ReviewCount: 76}, CreatedAt: now},
		{ID: "plugin-rate-smoother", Name: "Rate Smoother", Slug: "rate-smoother", Description: "Smooth bursty webhook traffic", Version: "1.0.0", Author: author, Type: "router", Category: "delivery", Tags: []string{"rate-limit"}, Runtime: "javascript", IsOfficial: true, IsVerified: true, Status: "published", Stats: MarketplaceStats{InstallCount: 3000, Rating: 4.3, ReviewCount: 54}, CreatedAt: now},
		{ID: "plugin-enricher", Name: "Payload Enricher", Slug: "enricher", Description: "Enrich payloads with external API data", Version: "1.0.0", Author: author, Type: "enricher", Category: "transforms", Tags: []string{"enrich"}, Runtime: "javascript", IsOfficial: true, IsVerified: true, Status: "published", Stats: MarketplaceStats{InstallCount: 2500, Rating: 4.2, ReviewCount: 42}, CreatedAt: now},
	}
}

// FullMarketplaceHandler provides HTTP handlers for the full marketplace
type FullMarketplaceHandler struct {
	service *FullMarketplaceService
}

// NewFullMarketplaceHandler creates a new handler
func NewFullMarketplaceHandler(service *FullMarketplaceService) *FullMarketplaceHandler {
	return &FullMarketplaceHandler{service: service}
}

// RegisterFullMarketplaceRoutes registers marketplace routes
func (h *FullMarketplaceHandler) RegisterFullMarketplaceRoutes(router *gin.RouterGroup) {
	m := router.Group("/plugin-marketplace")
	{
		m.GET("/plugins", h.SearchPlugins)
		m.GET("/plugins/:id", h.GetPlugin)
		m.POST("/plugins/:id/install", h.InstallPlugin)
		m.POST("/plugins/:id/reviews", h.CreateReview)
		m.POST("/submit", h.SubmitPlugin)
		m.GET("/sdk-spec", h.GetPluginSDKSpec)
		m.GET("/categories", h.ListCategories)
	}
}

func (h *FullMarketplaceHandler) SearchPlugins(c *gin.Context) {
	var req SearchPluginsRequest
	c.ShouldBindQuery(&req)
	plugins, total := h.service.SearchPlugins(c.Request.Context(), &req)
	c.JSON(http.StatusOK, gin.H{"plugins": plugins, "total": total})
}

func (h *FullMarketplaceHandler) GetPlugin(c *gin.Context) {
	plugin, err := h.service.GetPlugin(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, httputil.APIErrorResponse{Code: "NOT_FOUND", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (h *FullMarketplaceHandler) InstallPlugin(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	req := &InstallMarketplacePluginRequest{PluginID: c.Param("id")}
	c.ShouldBindJSON(req)
	install, err := h.service.InstallPlugin(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INSTALL_FAILED", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, install)
}

func (h *FullMarketplaceHandler) CreateReview(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req CreateMarketplaceReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}
	review, err := h.service.CreateReview(c.Request.Context(), tenantID, c.Param("id"), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "REVIEW_FAILED", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, review)
}

func (h *FullMarketplaceHandler) SubmitPlugin(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	var req SubmitPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}
	submission, err := h.service.SubmitPlugin(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, httputil.APIErrorResponse{Code: "SUBMIT_FAILED", Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, submission)
}

func (h *FullMarketplaceHandler) GetPluginSDKSpec(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetPluginSDKSpec())
}

func (h *FullMarketplaceHandler) ListCategories(c *gin.Context) {
	categories := []map[string]string{
		{"id": "notifications", "name": "Notifications"},
		{"id": "transforms", "name": "Transforms"},
		{"id": "validation", "name": "Validation"},
		{"id": "delivery", "name": "Delivery"},
		{"id": "security", "name": "Security"},
		{"id": "reliability", "name": "Reliability"},
		{"id": "analytics", "name": "Analytics"},
		{"id": "connectors", "name": "Connectors"},
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
