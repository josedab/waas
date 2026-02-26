package delivery

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/josedab/waas/pkg/catalog"
	"github.com/josedab/waas/pkg/database"
	apperrors "github.com/josedab/waas/pkg/errors"
	"github.com/josedab/waas/pkg/httputil"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/transform"
	"github.com/josedab/waas/pkg/utils"

	"github.com/google/uuid"
)

// Delivery engine configuration defaults
const (
	DefaultMaxWorkers            = 10
	DefaultRequestTimeout        = 30 * time.Second
	DefaultMaxIdleConns          = 100
	DefaultMaxIdleConnsPerHost   = 10
	DefaultIdleConnTimeout       = 90 * time.Second
	DefaultTLSHandshakeTimeout   = 10 * time.Second
	DefaultResponseHeaderTimeout = 10 * time.Second
	DefaultDialTimeout           = 30 * time.Second
	DefaultDialKeepAlive         = 30 * time.Second
	DefaultExpectContinueTimeout = 1 * time.Second
	MaxResponseBodySize          = 64 * 1024 // 64KB
)

// DeliveryEngine handles webhook delivery processing
type DeliveryEngine struct {
	db              *database.DB
	redis           *database.RedisClient
	logger          *utils.Logger
	config          *utils.Config
	httpClient      *http.Client
	consumer        *queue.Consumer
	deliveryRepo    repository.DeliveryAttemptRepository
	webhookRepo     repository.WebhookEndpointRepository
	transformRepo   repository.TransformationRepository
	transformEngine *transform.Engine
	healthMonitor   *EndpointHealthMonitor
	healthScorer    *HealthScorer
	catalogService  *catalog.Service
	dlqHook         DLQHook
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// DeliveryConfig holds configuration for webhook delivery
type DeliveryConfig struct {
	MaxWorkers            int           `json:"max_workers"`
	RequestTimeout        time.Duration `json:"request_timeout"`
	MaxIdleConns          int           `json:"max_idle_conns"`
	MaxIdleConnsPerHost   int           `json:"max_idle_conns_per_host"`
	IdleConnTimeout       time.Duration `json:"idle_conn_timeout"`
	TLSHandshakeTimeout   time.Duration `json:"tls_handshake_timeout"`
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout"`
}

// NewEngine creates a new delivery engine instance
func NewEngine() (*DeliveryEngine, error) {
	config, err := utils.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}
	logger := utils.NewLogger("delivery-engine")

	db, err := database.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	redis, err := database.NewRedisConnection(config.RedisURL)
	if err != nil {
		logger.Error("Failed to connect to Redis", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	// Create repositories
	deliveryRepo := repository.NewDeliveryAttemptRepository(db)
	webhookRepo := repository.NewWebhookEndpointRepository(db)
	transformRepo := repository.NewTransformationRepository(db)

	// Create transformation engine
	transformEngine := transform.NewEngine(transform.DefaultEngineConfig())

	// Create HTTP client with optimized settings
	httpClient := createHTTPClient(getDeliveryConfig())

	// Create health monitor
	healthMonitor := NewEndpointHealthMonitor(webhookRepo, logger)

	// Create health scorer for 0-100 endpoint health scoring
	healthScorer := NewHealthScorer(DefaultHealthScoringConfig())
	healthScorer.SetPauseCallback(func(endpointID uuid.UUID, reason string) {
		logger.Warn("Auto-pausing unhealthy endpoint", map[string]interface{}{
			"endpoint_id": endpointID,
			"reason":      reason,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := webhookRepo.UpdateStatus(ctx, endpointID, false); err != nil {
			logger.Error("Failed to pause unhealthy endpoint", map[string]interface{}{
				"endpoint_id": endpointID,
				"error":       err.Error(),
			})
		}
	})
	healthScorer.SetResumeCallback(func(endpointID uuid.UUID) {
		logger.Info("Auto-resuming recovered endpoint", map[string]interface{}{
			"endpoint_id": endpointID,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := webhookRepo.UpdateStatus(ctx, endpointID, true); err != nil {
			logger.Error("Failed to resume recovered endpoint", map[string]interface{}{
				"endpoint_id": endpointID,
				"error":       err.Error(),
			})
		}
	})

	ctx, cancel := context.WithCancel(context.Background())

	engine := &DeliveryEngine{
		db:              db,
		redis:           redis,
		logger:          logger,
		config:          config,
		httpClient:      httpClient,
		deliveryRepo:    deliveryRepo,
		webhookRepo:     webhookRepo,
		transformRepo:   transformRepo,
		transformEngine: transformEngine,
		healthMonitor:   healthMonitor,
		healthScorer:    healthScorer,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Create consumer with this engine as the message handler
	engine.consumer = queue.NewConsumer(redis, engine, getDeliveryConfig().MaxWorkers)

	return engine, nil
}

// Start begins the delivery engine processing
func (e *DeliveryEngine) Start() error {
	e.logger.Info("Starting delivery engine", nil)

	// Start health monitor
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.healthMonitor.Start(e.ctx)
	}()

	// Start queue consumer
	if err := e.consumer.Start(e.ctx); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	e.logger.Info("Delivery engine started successfully", nil)
	return nil
}

// Stop gracefully stops the delivery engine
func (e *DeliveryEngine) Stop() {
	e.logger.Info("Stopping delivery engine", nil)

	e.consumer.Stop()
	e.cancel()
	e.wg.Wait()

	e.logger.Info("Delivery engine stopped", nil)
}

// HandleDelivery implements the MessageHandler interface.
//
// Pipeline stages (executed in order):
//  1. Endpoint lookup   — fetch endpoint config from DB; fail fast if missing
//  2. Record attempt    — persist a DeliveryAttempt row (status: processing)
//  3. Active check      — skip delivery if the endpoint has been deactivated
//  4. Health gate       — skip (with retry) if the health scorer has auto-paused
//     the endpoint due to sustained failures
//  5. Transformation    — apply any configured payload transformations; fall back
//     to the original payload on error
//  6. Schema validation — validate payload against the event catalog (if enabled);
//     in strict mode, reject delivery on schema mismatch
//  7. HTTP delivery     — perform the actual POST to the endpoint URL
//  8. Record result     — update the attempt row with status, HTTP code, latency
//  9. Health feedback   — feed success/failure + latency into the health scorer
//     and the legacy health monitor
//  10. DLQ routing       — if permanently failed (retries exhausted), move to
//     the dead-letter queue for manual inspection
func (e *DeliveryEngine) HandleDelivery(ctx context.Context, message *queue.DeliveryMessage) (*queue.DeliveryResult, error) {
	startTime := time.Now()

	e.logger.Info("Processing webhook delivery", map[string]interface{}{
		"delivery_id":    message.DeliveryID,
		"endpoint_id":    message.EndpointID,
		"attempt_number": message.AttemptNumber,
	})

	// Get webhook endpoint details
	endpoint, err := e.webhookRepo.GetByID(ctx, message.EndpointID)
	if err != nil {
		e.logger.Error("Failed to get webhook endpoint", map[string]interface{}{
			"delivery_id": message.DeliveryID,
			"endpoint_id": message.EndpointID,
			"error":       err.Error(),
		})
		errMsg := fmt.Sprintf("endpoint not found: %v", err)
		return &queue.DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        queue.StatusFailed,
			ErrorMessage:  &errMsg,
			AttemptNumber: message.AttemptNumber,
		}, nil
	}

	// Create delivery attempt record
	attempt := &models.DeliveryAttempt{
		ID:            uuid.New(),
		EndpointID:    message.EndpointID,
		PayloadHash:   utils.HashPayload(message.Payload),
		PayloadSize:   len(message.Payload),
		Status:        queue.StatusProcessing,
		AttemptNumber: message.AttemptNumber,
		ScheduledAt:   message.ScheduledAt,
		CreatedAt:     time.Now(),
	}

	if err := e.deliveryRepo.Create(ctx, attempt); err != nil {
		e.logger.Error("Failed to create delivery attempt record", map[string]interface{}{
			"delivery_id": message.DeliveryID,
			"error":       err.Error(),
		})
	}

	// Check if endpoint is active
	if !endpoint.IsActive {
		e.logger.Warn("Skipping delivery to inactive endpoint", map[string]interface{}{
			"delivery_id": message.DeliveryID,
			"endpoint_id": message.EndpointID,
		})
		errMsg := "endpoint is inactive"
		result := &queue.DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        queue.StatusFailed,
			ErrorMessage:  &errMsg,
			AttemptNumber: message.AttemptNumber,
		}

		// Update delivery attempt with result
		attempt.Status = result.Status
		attempt.ErrorMessage = result.ErrorMessage
		if err := e.deliveryRepo.Update(ctx, attempt); err != nil {
			e.logger.Error("Failed to update delivery attempt", map[string]interface{}{
				"delivery_id": message.DeliveryID,
				"error":       err.Error(),
			})
		}

		return result, nil
	}

	// Check endpoint health score — skip delivery if endpoint is auto-paused
	if e.healthScorer != nil {
		score := e.healthScorer.GetScore(message.EndpointID)
		if score.IsPaused {
			e.logger.Warn("Skipping delivery to health-paused endpoint", map[string]interface{}{
				"delivery_id":  message.DeliveryID,
				"endpoint_id":  message.EndpointID,
				"health_score": score.Score,
				"pause_reason": score.PauseReason,
			})
			errMsg := fmt.Sprintf("endpoint paused: health score %d/100 (%s)", score.Score, score.PauseReason)
			result := &queue.DeliveryResult{
				DeliveryID:    message.DeliveryID,
				Status:        queue.StatusRetrying,
				ErrorMessage:  &errMsg,
				AttemptNumber: message.AttemptNumber,
			}
			attempt.Status = result.Status
			attempt.ErrorMessage = result.ErrorMessage
			if err := e.deliveryRepo.Update(ctx, attempt); err != nil {
				e.logger.Error("Failed to update delivery attempt", map[string]interface{}{
					"delivery_id": message.DeliveryID,
					"error":       err.Error(),
				})
			}
			return result, nil
		}
	}

	// Apply transformations if configured for this endpoint
	transformedPayload, err := e.applyTransformations(ctx, message.EndpointID, message.Payload)
	if err != nil {
		e.logger.Warn("Transformation failed, using original payload", map[string]interface{}{
			"delivery_id": message.DeliveryID,
			"endpoint_id": message.EndpointID,
			"error":       err.Error(),
		})
		// Continue with original payload on transformation failure
	} else if transformedPayload != nil {
		message.Payload = transformedPayload
		e.logger.Info("Payload transformed successfully", map[string]interface{}{
			"delivery_id": message.DeliveryID,
			"endpoint_id": message.EndpointID,
		})
	}

	// Publish-time schema validation via event catalog
	if e.catalogService != nil && message.EventType != "" {
		valResult := e.catalogService.ValidateForDelivery(ctx, message.TenantID, message.EventType, message.Payload)
		if valResult != nil && !valResult.Valid {
			e.logger.Warn("Payload failed publish-time schema validation", map[string]interface{}{
				"delivery_id": message.DeliveryID,
				"endpoint_id": message.EndpointID,
				"event_type":  message.EventType,
				"mode":        valResult.Mode,
				"issues":      valResult.Issues,
			})
			if valResult.Mode == string(catalog.ValidationModeStrict) {
				errMsg := fmt.Sprintf("schema validation failed: %v", valResult.Issues)
				result := &queue.DeliveryResult{
					DeliveryID:    message.DeliveryID,
					Status:        queue.StatusFailed,
					ErrorMessage:  &errMsg,
					AttemptNumber: message.AttemptNumber,
				}
				attempt.Status = result.Status
				attempt.ErrorMessage = result.ErrorMessage
				if err := e.deliveryRepo.Update(ctx, attempt); err != nil {
					e.logger.Error("Failed to update delivery attempt", map[string]interface{}{
						"delivery_id": message.DeliveryID,
						"error":       err.Error(),
					})
				}
				return result, nil
			}
		}
	}

	// Perform the HTTP delivery
	result := e.performDelivery(ctx, endpoint, message)

	// Update delivery attempt with result
	attempt.Status = result.Status
	attempt.HTTPStatus = result.HTTPStatus
	attempt.ResponseBody = result.ResponseBody
	attempt.ErrorMessage = result.ErrorMessage
	if result.Status == queue.StatusSuccess {
		now := time.Now()
		attempt.DeliveredAt = &now
	}

	if err := e.deliveryRepo.Update(ctx, attempt); err != nil {
		e.logger.Error("Failed to update delivery attempt", map[string]interface{}{
			"delivery_id": message.DeliveryID,
			"error":       err.Error(),
		})
	}

	// Update endpoint health status
	e.healthMonitor.RecordDeliveryResult(message.EndpointID, result.Status == queue.StatusSuccess, result.HTTPStatus)

	// Update health scorer with delivery metrics for 0-100 scoring
	if e.healthScorer != nil {
		latencyMs := float64(time.Since(startTime).Milliseconds())
		httpStatus := 0
		if result.HTTPStatus != nil {
			httpStatus = *result.HTTPStatus
		}
		e.healthScorer.RecordDelivery(message.EndpointID, result.Status == queue.StatusSuccess, latencyMs, httpStatus)
	}

	// Route to DLQ if permanently failed (retries exhausted)
	if result.Status == queue.StatusFailed {
		e.routeToDeadLetter(ctx, message, result)
	}

	duration := time.Since(startTime)
	e.logger.Info("Delivery processing completed", map[string]interface{}{
		"delivery_id":    message.DeliveryID,
		"status":         result.Status,
		"duration_ms":    duration.Milliseconds(),
		"attempt_number": message.AttemptNumber,
	})

	return result, nil
}

// performDelivery executes the actual HTTP request to deliver the webhook
func (e *DeliveryEngine) performDelivery(ctx context.Context, endpoint *models.WebhookEndpoint, message *queue.DeliveryMessage) *queue.DeliveryResult {
	// Create request context with timeout
	deliveryCtx, cancel := context.WithTimeout(ctx, getDeliveryConfig().RequestTimeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(deliveryCtx, "POST", endpoint.URL, strings.NewReader(string(message.Payload)))
	if err != nil {
		errMsg := fmt.Sprintf("failed to create request: %v", err)
		return &queue.DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        queue.StatusFailed,
			ErrorMessage:  &errMsg,
			AttemptNumber: message.AttemptNumber,
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WebhookPlatform/1.0")

	// Add signature header
	if message.Signature != "" {
		req.Header.Set("X-Webhook-Signature", message.Signature)
	}

	// Add custom headers from endpoint configuration
	for key, value := range endpoint.CustomHeaders {
		req.Header.Set(key, value)
	}

	// Add headers from message
	for key, value := range message.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("request failed: %v", err)

		// Determine if this is a retryable error
		status := queue.StatusFailed
		if isRetryableError(err) && message.AttemptNumber < message.MaxAttempts {
			status = queue.StatusRetrying
		}

		return &queue.DeliveryResult{
			DeliveryID:    message.DeliveryID,
			Status:        status,
			ErrorMessage:  &errMsg,
			AttemptNumber: message.AttemptNumber,
		}
	}
	defer resp.Body.Close()

	// Read response body (limit to MaxResponseBodySize)
	bodyReader := io.LimitReader(resp.Body, MaxResponseBodySize)
	responseBody, err := io.ReadAll(bodyReader)
	if err != nil {
		e.logger.Warn("Failed to read response body", map[string]interface{}{
			"delivery_id": message.DeliveryID,
			"error":       err.Error(),
		})
	}

	responseBodyStr := string(responseBody)

	// Determine delivery status based on HTTP status code
	status := queue.StatusSuccess
	var errorMessage *string

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		status = queue.StatusFailed
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, responseBodyStr)
		errorMessage = &errMsg

		// Check if this is retryable based on status code
		if isRetryableStatusCode(resp.StatusCode) && message.AttemptNumber < message.MaxAttempts {
			status = queue.StatusRetrying
		}
	}

	now := time.Now()
	result := &queue.DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        status,
		HTTPStatus:    &resp.StatusCode,
		ResponseBody:  &responseBodyStr,
		ErrorMessage:  errorMessage,
		AttemptNumber: message.AttemptNumber,
	}

	if status == queue.StatusSuccess {
		result.DeliveredAt = &now
	}

	return result
}

// isPrivateIP delegates to the shared httputil package.
func isPrivateIP(ip net.IP) bool {
	return httputil.IsPrivateIP(ip)
}

// ssrfSafeDialContext delegates to the shared httputil package.
func ssrfSafeDialContext(dialer *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return httputil.SSRFSafeDialContext(dialer)
}

// createHTTPClient creates an optimized HTTP client for webhook delivery
func createHTTPClient(config DeliveryConfig) *http.Client {
	dialer := &net.Dialer{
		Timeout:   DefaultDialTimeout,
		KeepAlive: DefaultDialKeepAlive,
	}

	transport := &http.Transport{
		DialContext:           ssrfSafeDialContext(dialer),
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: DefaultExpectContinueTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.RequestTimeout,
	}
}

// getDeliveryConfig returns the delivery configuration
func getDeliveryConfig() DeliveryConfig {
	return DeliveryConfig{
		MaxWorkers:            DefaultMaxWorkers,
		RequestTimeout:        DefaultRequestTimeout,
		MaxIdleConns:          DefaultMaxIdleConns,
		MaxIdleConnsPerHost:   DefaultMaxIdleConnsPerHost,
		IdleConnTimeout:       DefaultIdleConnTimeout,
		TLSHandshakeTimeout:   DefaultTLSHandshakeTimeout,
		ResponseHeaderTimeout: DefaultResponseHeaderTimeout,
	}
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are generally retryable
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// Context timeout errors are retryable
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// DNS errors are retryable
	if errors.Is(err, apperrors.ErrNoSuchHost) || strings.Contains(err.Error(), "no such host") {
		return true
	}

	// Connection refused is retryable
	if errors.Is(err, apperrors.ErrConnectionRefused) || strings.Contains(err.Error(), "connection refused") {
		return true
	}

	return false
}

// isRetryableStatusCode determines if an HTTP status code should trigger a retry
func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case 408, // Request Timeout
		429, // Too Many Requests
		500, // Internal Server Error
		502, // Bad Gateway
		503, // Service Unavailable
		504: // Gateway Timeout
		return true
	default:
		return false
	}
}

// applyTransformations applies all transformations configured for an endpoint
func (e *DeliveryEngine) applyTransformations(ctx context.Context, endpointID uuid.UUID, payload json.RawMessage) (json.RawMessage, error) {
	// Get transformations for this endpoint
	transformations, err := e.transformRepo.GetByEndpointID(ctx, endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transformations: %w", err)
	}

	if len(transformations) == 0 {
		return nil, nil // No transformations to apply
	}

	// Parse payload
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Apply each transformation in order
	currentData := data
	for _, t := range transformations {
		if !t.Enabled {
			continue
		}

		result, err := e.transformEngine.Transform(ctx, t.Script, currentData)
		if err != nil {
			e.logger.Warn("Transformation execution failed", map[string]interface{}{
				"transformation_id": t.ID,
				"endpoint_id":       endpointID,
				"error":             err.Error(),
			})
			// Skip this transformation but continue with others
			continue
		}

		// Log transformation execution
		if t.Config.EnableLogging {
			e.logTransformationExecution(ctx, t.ID, endpointID, result)
		}

		// Use transformed data for next transformation
		if transformedData, ok := result.Output.(map[string]interface{}); ok {
			currentData = transformedData
		} else {
			e.logger.Warn("Transformation output is not an object, skipping", map[string]interface{}{
				"transformation_id": t.ID,
				"endpoint_id":       endpointID,
			})
		}
	}

	// Serialize transformed data back to JSON
	transformedPayload, err := json.Marshal(currentData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transformed payload: %w", err)
	}

	return transformedPayload, nil
}

// logTransformationExecution logs the result of a transformation execution
func (e *DeliveryEngine) logTransformationExecution(ctx context.Context, transformationID, endpointID uuid.UUID, result *transform.TransformResult) {
	log := &models.TransformationLog{
		ID:               uuid.New(),
		TransformationID: transformationID,
		EndpointID:       &endpointID,
		ExecutionTimeMs:  result.ExecutionTimeMs,
		Success:          true,
		CreatedAt:        time.Now(),
	}

	if len(result.Logs) > 0 {
		logsJSON, _ := json.Marshal(result.Logs)
		logsStr := string(logsJSON)
		log.OutputPreview = &logsStr
	}

	if err := e.transformRepo.CreateLog(ctx, log); err != nil {
		e.logger.Warn("Failed to create transformation log", map[string]interface{}{
			"transformation_id": transformationID,
			"error":             err.Error(),
		})
	}
}

// DLQHook is a function called when a delivery permanently fails (retries exhausted).
type DLQHook func(ctx context.Context, tenantID, endpointID, deliveryID string, payload json.RawMessage, headers json.RawMessage, attempts []DLQAttemptDetail, finalError string)

// DLQAttemptDetail captures one delivery attempt for DLQ context.
type DLQAttemptDetail struct {
	AttemptNumber int       `json:"attempt_number"`
	HTTPStatus    *int      `json:"http_status,omitempty"`
	ResponseBody  *string   `json:"response_body,omitempty"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
	AttemptedAt   time.Time `json:"attempted_at"`
}

// SetCatalogService sets the catalog service for publish-time schema validation
func (e *DeliveryEngine) SetCatalogService(svc *catalog.Service) {
	e.catalogService = svc
}

// SetDLQHook registers a hook that is called for permanently failed deliveries.
func (e *DeliveryEngine) SetDLQHook(hook DLQHook) {
	e.dlqHook = hook
}

// routeToDeadLetter invokes the DLQ hook if configured.
func (e *DeliveryEngine) routeToDeadLetter(ctx context.Context, message *queue.DeliveryMessage, result *queue.DeliveryResult) {
	if e.dlqHook == nil {
		return
	}

	payloadJSON, _ := json.Marshal(message.Payload)
	headersJSON, _ := json.Marshal(message.Headers)

	errMsg := ""
	if result.ErrorMessage != nil {
		errMsg = *result.ErrorMessage
	}

	attempt := DLQAttemptDetail{
		AttemptNumber: result.AttemptNumber,
		HTTPStatus:    result.HTTPStatus,
		ResponseBody:  result.ResponseBody,
		ErrorMessage:  result.ErrorMessage,
		AttemptedAt:   time.Now(),
	}

	e.dlqHook(ctx, message.TenantID.String(), message.EndpointID.String(), message.DeliveryID.String(), payloadJSON, headersJSON, []DLQAttemptDetail{attempt}, errMsg)
}
