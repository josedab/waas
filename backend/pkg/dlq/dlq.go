package dlq

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---------- Models ----------

// DLQEntry represents a failed delivery that has been moved to the dead letter queue.
type DLQEntry struct {
	ID                 string          `json:"id"`
	TenantID           string          `json:"tenant_id"`
	EndpointID         string          `json:"endpoint_id"`
	OriginalDeliveryID string          `json:"original_delivery_id"`
	Payload            json.RawMessage `json:"payload"`
	Headers            json.RawMessage `json:"headers"`
	AllAttempts        []AttemptDetail `json:"all_attempts"`
	ErrorType          string          `json:"error_type"`
	FinalStatus        string          `json:"final_status"`
	FinalHTTPStatus    *int            `json:"final_http_status,omitempty"`
	FinalResponseBody  *string         `json:"final_response_body,omitempty"`
	RetryCount         int             `json:"retry_count"`
	MaxRetries         int             `json:"max_retries"`
	CreatedAt          time.Time       `json:"created_at"`
	ExpiresAt          time.Time       `json:"expires_at"`
	Replayed           bool            `json:"replayed"`
	ReplayedAt         *time.Time      `json:"replayed_at,omitempty"`
}

// AttemptDetail captures information about a single delivery attempt.
type AttemptDetail struct {
	AttemptNumber int       `json:"attempt_number"`
	HTTPStatus    *int      `json:"http_status,omitempty"`
	ResponseBody  *string   `json:"response_body,omitempty"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
	AttemptedAt   time.Time `json:"attempted_at"`
	DurationMs    int       `json:"duration_ms"`
}

// DLQFilter defines criteria for querying DLQ entries.
type DLQFilter struct {
	TenantID    string     `json:"tenant_id,omitempty" form:"tenant_id"`
	EndpointID  string     `json:"endpoint_id,omitempty" form:"endpoint_id"`
	Status      string     `json:"status,omitempty" form:"status"`
	ErrorType   string     `json:"error_type,omitempty" form:"error_type"`
	DateFrom    *time.Time `json:"date_from,omitempty" form:"date_from"`
	DateTo      *time.Time `json:"date_to,omitempty" form:"date_to"`
	SearchQuery string     `json:"search_query,omitempty" form:"search_query"`
	Limit       int        `json:"limit,omitempty" form:"limit"`
	Offset      int        `json:"offset,omitempty" form:"offset"`
}

// DLQStats provides aggregate statistics about the dead letter queue.
type DLQStats struct {
	TotalEntries   int64   `json:"total_entries"`
	PendingCount   int64   `json:"pending_count"`
	ReplayedCount  int64   `json:"replayed_count"`
	GrowthRate     float64 `json:"growth_rate"`
	OldestEntryAge string  `json:"oldest_entry_age"`
}

// AlertRule defines a rule for generating alerts based on DLQ metrics.
type AlertRule struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	Name      string         `json:"name"`
	Condition AlertCondition `json:"condition"`
	Action    string         `json:"action"`
	Enabled   bool           `json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
}

// AlertCondition specifies when an alert should fire.
type AlertCondition struct {
	MetricType    string  `json:"metric_type"`
	Threshold     float64 `json:"threshold"`
	WindowMinutes int     `json:"window_minutes"`
	Operator      string  `json:"operator"`
}

// RetentionPolicy controls how long DLQ entries are kept.
type RetentionPolicy struct {
	TenantID          string `json:"tenant_id"`
	RetentionDays     int    `json:"retention_days"`
	MaxEntries        int64  `json:"max_entries"`
	CompressAfterDays int    `json:"compress_after_days"`
	AutoPurge         bool   `json:"auto_purge"`
}

// BulkRetryRequest defines parameters for a bulk retry operation.
type BulkRetryRequest struct {
	Filter   DLQFilter `json:"filter"`
	MaxCount int       `json:"max_count"`
	DryRun   bool      `json:"dry_run"`
}

// BulkRetryResult summarises the outcome of a bulk retry.
type BulkRetryResult struct {
	Requested int      `json:"requested"`
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
	DryRun    bool     `json:"dry_run"`
}

// ExportRequest defines parameters for exporting DLQ entries.
type ExportRequest struct {
	Filter DLQFilter `json:"filter"`
	Format string    `json:"format"`
}

// ---------- Service ----------

// Service implements DLQ business logic backed by in-memory storage.
type Service struct {
	mu                sync.RWMutex
	entries           map[string]*DLQEntry
	alertRules        map[string]*AlertRule
	retentionPolicies map[string]*RetentionPolicy
}

// NewService creates a new DLQ service with in-memory storage.
func NewService() *Service {
	return &Service{
		entries:           make(map[string]*DLQEntry),
		alertRules:        make(map[string]*AlertRule),
		retentionPolicies: make(map[string]*RetentionPolicy),
	}
}

// GetEntries returns DLQ entries matching the supplied filter and a total count.
func (s *Service) GetEntries(_ context.Context, filter *DLQFilter) ([]DLQEntry, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []DLQEntry
	for _, e := range s.entries {
		if !s.matchFilter(e, filter) {
			continue
		}
		results = append(results, *e)
	}

	total := len(results)

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	if offset >= len(results) {
		return []DLQEntry{}, total, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end], total, nil
}

// GetEntry returns a single DLQ entry by tenant and entry ID.
func (s *Service) GetEntry(_ context.Context, tenantID, entryID string) (*DLQEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[entryID]
	if !ok || entry.TenantID != tenantID {
		return nil, fmt.Errorf("DLQ entry %s not found", entryID)
	}
	return entry, nil
}

// ReplayEntry marks an entry as replayed and returns it.
func (s *Service) ReplayEntry(_ context.Context, tenantID, entryID string) (*DLQEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[entryID]
	if !ok || entry.TenantID != tenantID {
		return nil, fmt.Errorf("DLQ entry %s not found", entryID)
	}
	if entry.Replayed {
		return nil, fmt.Errorf("DLQ entry %s has already been replayed", entryID)
	}

	now := time.Now()
	entry.Replayed = true
	entry.ReplayedAt = &now
	entry.FinalStatus = "replayed"
	return entry, nil
}

// BulkRetry retries entries matching the filter within the request constraints.
func (s *Service) BulkRetry(_ context.Context, tenantID string, req *BulkRetryRequest) (*BulkRetryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req.Filter.TenantID = tenantID
	maxCount := req.MaxCount
	if maxCount <= 0 {
		maxCount = 100
	}

	result := &BulkRetryResult{DryRun: req.DryRun}

	for _, entry := range s.entries {
		if result.Requested >= maxCount {
			break
		}
		if !s.matchFilter(entry, &req.Filter) || entry.Replayed {
			continue
		}
		result.Requested++
		if req.DryRun {
			result.Succeeded++
			continue
		}
		now := time.Now()
		entry.Replayed = true
		entry.ReplayedAt = &now
		entry.FinalStatus = "replayed"
		result.Succeeded++
	}

	return result, nil
}

// GetStats returns aggregate statistics for a tenant's DLQ.
func (s *Service) GetStats(_ context.Context, tenantID string) (*DLQStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &DLQStats{}
	var oldest time.Time

	for _, e := range s.entries {
		if e.TenantID != tenantID {
			continue
		}
		stats.TotalEntries++
		if e.Replayed {
			stats.ReplayedCount++
		} else {
			stats.PendingCount++
		}
		if oldest.IsZero() || e.CreatedAt.Before(oldest) {
			oldest = e.CreatedAt
		}
	}

	if !oldest.IsZero() {
		stats.OldestEntryAge = time.Since(oldest).Round(time.Second).String()
	}

	return stats, nil
}

// GetAlertRules returns all alert rules for a tenant.
func (s *Service) GetAlertRules(_ context.Context, tenantID string) ([]AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rules []AlertRule
	for _, r := range s.alertRules {
		if r.TenantID == tenantID {
			rules = append(rules, *r)
		}
	}
	return rules, nil
}

// CreateAlertRule creates a new alert rule for a tenant.
func (s *Service) CreateAlertRule(_ context.Context, tenantID string, rule *AlertRule) (*AlertRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule.ID = uuid.New().String()
	rule.TenantID = tenantID
	rule.CreatedAt = time.Now()
	s.alertRules[rule.ID] = rule
	return rule, nil
}

// GetRetentionPolicy returns the retention policy for a tenant.
func (s *Service) GetRetentionPolicy(_ context.Context, tenantID string) (*RetentionPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, ok := s.retentionPolicies[tenantID]
	if !ok {
		return &RetentionPolicy{
			TenantID:          tenantID,
			RetentionDays:     30,
			MaxEntries:        10000,
			CompressAfterDays: 7,
			AutoPurge:         true,
		}, nil
	}
	return policy, nil
}

// UpdateRetentionPolicy creates or updates a retention policy for a tenant.
func (s *Service) UpdateRetentionPolicy(_ context.Context, tenantID string, policy *RetentionPolicy) (*RetentionPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy.TenantID = tenantID
	s.retentionPolicies[tenantID] = policy
	return policy, nil
}

// ExportEntries exports DLQ entries matching the filter in the requested format.
func (s *Service) ExportEntries(ctx context.Context, tenantID string, req *ExportRequest) ([]byte, error) {
	req.Filter.TenantID = tenantID
	req.Filter.Limit = 10000
	entries, _, err := s.GetEntries(ctx, &req.Filter)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(req.Format) {
	case "csv":
		return s.exportCSV(entries)
	default:
		return json.MarshalIndent(entries, "", "  ")
	}
}

func (s *Service) exportCSV(entries []DLQEntry) ([]byte, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	header := []string{"id", "tenant_id", "endpoint_id", "error_type", "final_status", "retry_count", "created_at", "replayed"}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	for _, e := range entries {
		row := []string{
			e.ID,
			e.TenantID,
			e.EndpointID,
			e.ErrorType,
			e.FinalStatus,
			strconv.Itoa(e.RetryCount),
			e.CreatedAt.Format(time.RFC3339),
			strconv.FormatBool(e.Replayed),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return []byte(buf.String()), w.Error()
}

// RouteToDeadLetter creates a new DLQ entry from a failed delivery.
// This is intended to be called by the delivery engine when all retries are exhausted.
func (s *Service) RouteToDeadLetter(_ context.Context, tenantID, endpointID, deliveryID string, payload json.RawMessage, headers json.RawMessage, attempts []AttemptDetail, finalError string) (*DLQEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	errType := "unknown"
	var finalHTTPStatus *int
	var finalResponseBody *string
	if len(attempts) > 0 {
		last := attempts[len(attempts)-1]
		finalHTTPStatus = last.HTTPStatus
		finalResponseBody = last.ResponseBody
		if last.HTTPStatus != nil {
			code := *last.HTTPStatus
			switch {
			case code >= 500:
				errType = "server_error"
			case code == 429:
				errType = "rate_limit"
			case code >= 400:
				errType = "client_error"
			}
		} else if last.ErrorMessage != nil && strings.Contains(strings.ToLower(*last.ErrorMessage), "timeout") {
			errType = "timeout"
		}
	}

	entry := &DLQEntry{
		ID:                 uuid.New().String(),
		TenantID:           tenantID,
		EndpointID:         endpointID,
		OriginalDeliveryID: deliveryID,
		Payload:            payload,
		Headers:            headers,
		AllAttempts:        attempts,
		ErrorType:          errType,
		FinalStatus:        "dead_letter",
		FinalHTTPStatus:    finalHTTPStatus,
		FinalResponseBody:  finalResponseBody,
		RetryCount:         len(attempts),
		MaxRetries:         len(attempts),
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(30 * 24 * time.Hour),
	}

	s.entries[entry.ID] = entry
	return entry, nil
}

func (s *Service) matchFilter(entry *DLQEntry, filter *DLQFilter) bool {
	if filter == nil {
		return true
	}
	if filter.TenantID != "" && entry.TenantID != filter.TenantID {
		return false
	}
	if filter.EndpointID != "" && entry.EndpointID != filter.EndpointID {
		return false
	}
	if filter.Status != "" && entry.FinalStatus != filter.Status {
		return false
	}
	if filter.ErrorType != "" && entry.ErrorType != filter.ErrorType {
		return false
	}
	if filter.DateFrom != nil && entry.CreatedAt.Before(*filter.DateFrom) {
		return false
	}
	if filter.DateTo != nil && entry.CreatedAt.After(*filter.DateTo) {
		return false
	}
	if filter.SearchQuery != "" {
		q := strings.ToLower(filter.SearchQuery)
		payloadStr := strings.ToLower(string(entry.Payload))
		errMsg := ""
		for _, a := range entry.AllAttempts {
			if a.ErrorMessage != nil {
				errMsg += " " + *a.ErrorMessage
			}
		}
		if !strings.Contains(payloadStr, q) && !strings.Contains(strings.ToLower(errMsg), q) {
			return false
		}
	}
	return true
}

// ---------- Handler ----------

// Handler exposes DLQ operations over HTTP.
type Handler struct {
	service *Service
}

// NewHandler creates a new DLQ handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all DLQ routes on the supplied router group.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	dlq := r.Group("/dlq")
	{
		dlq.GET("/entries", h.ListEntries)
		dlq.GET("/entries/:id", h.GetEntry)
		dlq.POST("/entries/:id/replay", h.ReplayEntry)
		dlq.POST("/bulk-retry", h.BulkRetry)
		dlq.GET("/stats", h.GetStats)
		dlq.GET("/alerts", h.ListAlertRules)
		dlq.POST("/alerts", h.CreateAlertRule)
		dlq.GET("/retention", h.GetRetentionPolicy)
		dlq.PUT("/retention", h.UpdateRetentionPolicy)
		dlq.POST("/export", h.ExportEntries)
		dlq.GET("/search", h.SearchEntries)
	}
}

func (h *Handler) getTenantID(c *gin.Context) string {
	if tid, exists := c.Get("tenant_id"); exists {
		switch v := tid.(type) {
		case string:
			return v
		case uuid.UUID:
			return v.String()
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// ListEntries handles GET /dlq/entries
func (h *Handler) ListEntries(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	var filter DLQFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}
	filter.TenantID = tenantID

	entries, total, err := h.service.GetEntries(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "DLQ_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"pagination": gin.H{
			"total":    total,
			"limit":    filter.Limit,
			"offset":   filter.Offset,
			"has_more": filter.Offset+filter.Limit < total,
		},
	})
}

// GetEntry handles GET /dlq/entries/:id
func (h *Handler) GetEntry(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	entry, err := h.service.GetEntry(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, entry)
}

// ReplayEntry handles POST /dlq/entries/:id/replay
func (h *Handler) ReplayEntry(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	entry, err := h.service.ReplayEntry(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "REPLAY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, entry)
}

// BulkRetry handles POST /dlq/bulk-retry
func (h *Handler) BulkRetry(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	var req BulkRetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	result, err := h.service.BulkRetry(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "BULK_RETRY_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetStats handles GET /dlq/stats
func (h *Handler) GetStats(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "STATS_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ListAlertRules handles GET /dlq/alerts
func (h *Handler) ListAlertRules(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	rules, err := h.service.GetAlertRules(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "ALERT_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alert_rules": rules, "count": len(rules)})
}

// CreateAlertRule handles POST /dlq/alerts
func (h *Handler) CreateAlertRule(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	created, err := h.service.CreateAlertRule(c.Request.Context(), tenantID, &rule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CREATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetRetentionPolicy handles GET /dlq/retention
func (h *Handler) GetRetentionPolicy(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	policy, err := h.service.GetRetentionPolicy(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "RETENTION_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// UpdateRetentionPolicy handles PUT /dlq/retention
func (h *Handler) UpdateRetentionPolicy(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	var policy RetentionPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	updated, err := h.service.UpdateRetentionPolicy(c.Request.Context(), tenantID, &policy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "UPDATE_FAILED", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// ExportEntries handles POST /dlq/export
func (h *Handler) ExportEntries(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	data, err := h.service.ExportEntries(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "EXPORT_FAILED", "message": err.Error()}})
		return
	}

	contentType := "application/json"
	if strings.ToLower(req.Format) == "csv" {
		contentType = "text/csv"
	}

	c.Data(http.StatusOK, contentType, data)
}

// SearchEntries handles GET /dlq/search?q=...
func (h *Handler) SearchEntries(c *gin.Context) {
	tenantID := h.getTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Tenant not found in context"}})
		return
	}

	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "query parameter 'q' is required"}})
		return
	}

	filter := DLQFilter{
		TenantID:    tenantID,
		SearchQuery: q,
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = v
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = v
		}
	}

	entries, total, err := h.service.GetEntries(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SEARCH_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"query":   q,
		"pagination": gin.H{
			"total":    total,
			"limit":    filter.Limit,
			"offset":   filter.Offset,
			"has_more": filter.Offset+filter.Limit < total,
		},
	})
}
