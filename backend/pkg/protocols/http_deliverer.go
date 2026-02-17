package protocols

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// HTTPDeliverer implements HTTP/HTTPS webhook delivery
type HTTPDeliverer struct {
	client *http.Client
}

// NewHTTPDeliverer creates a new HTTP deliverer
func NewHTTPDeliverer() *HTTPDeliverer {
	return &HTTPDeliverer{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Protocol returns the protocol
func (d *HTTPDeliverer) Protocol() Protocol {
	return ProtocolHTTP
}

// Validate validates the delivery config
func (d *HTTPDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("target URL is required")
	}

	u, err := url.Parse(config.Target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s (expected http or https)", u.Scheme)
	}

	return nil
}

// Deliver performs the HTTP delivery
func (d *HTTPDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	response := &DeliveryResponse{}

	// Get HTTP options
	opts := parseHTTPOptions(config.Options)

	// Build request
	method := opts.Method
	if method == "" {
		method = http.MethodPost
	}

	targetURL := config.Target
	if opts.Path != "" {
		if !strings.HasSuffix(targetURL, "/") && !strings.HasPrefix(opts.Path, "/") {
			targetURL += "/"
		}
		targetURL += opts.Path
	}

	// Add query params
	if len(opts.QueryParams) > 0 {
		u, err := url.Parse(targetURL)
		if err == nil {
			q := u.Query()
			for k, v := range opts.QueryParams {
				q.Set(k, v)
			}
			u.RawQuery = q.Encode()
			targetURL = u.String()
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(request.Payload))
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	// Set headers
	req.Header.Set("Content-Type", request.ContentType)
	if request.ContentType == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set delivery-specific headers
	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}

	// Set config headers
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// Add standard webhook headers
	req.Header.Set("X-Webhook-ID", request.WebhookID)
	req.Header.Set("X-Delivery-ID", request.ID)
	req.Header.Set("X-Delivery-Attempt", fmt.Sprintf("%d", request.AttemptNumber))

	// Apply authentication
	if config.Auth != nil {
		applyHTTPAuth(req, config.Auth)
	}

	// Create client with custom TLS if needed
	client := d.client
	if config.TLS != nil && config.TLS.Enabled {
		transport := &http.Transport{
			TLSClientConfig: buildTLSConfig(config.TLS),
		}
		client = &http.Client{
			Timeout:   time.Duration(config.Timeout) * time.Second,
			Transport: transport,
		}
	}

	if config.Timeout > 0 {
		client.Timeout = time.Duration(config.Timeout) * time.Second
	}

	// Configure redirects
	if !opts.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if opts.MaxRedirects > 0 {
		maxRedirects := opts.MaxRedirects
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		}
	}

	// Perform request
	resp, err := client.Do(req)
	response.Duration = time.Since(start)

	if err != nil {
		response.Error = err.Error()
		response.ErrorType = categorizeHTTPError(err)
		return response, nil
	}
	defer resp.Body.Close()

	response.StatusCode = resp.StatusCode
	response.Headers = make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			response.Headers[k] = v[0]
		}
	}

	// Read body (limited to 1MB)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	response.Body = body

	// Check for retry-after header
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds := parseRetryAfter(retryAfter); seconds > 0 {
			d := time.Duration(seconds) * time.Second
			response.RetryAfter = &d
		}
	}

	// Determine success
	expectedStatuses := opts.ExpectedStatuses
	if len(expectedStatuses) == 0 {
		expectedStatuses = []int{200, 201, 202, 204}
	}

	response.Success = false
	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			response.Success = true
			break
		}
	}

	if !response.Success {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			response.ErrorType = ErrorTypeClientError
		} else if resp.StatusCode >= 500 {
			response.ErrorType = ErrorTypeServer
		}
		if resp.StatusCode == 429 {
			response.ErrorType = ErrorTypeRateLimit
		}
		response.Error = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
	}

	return response, nil
}

// Close closes the deliverer
func (d *HTTPDeliverer) Close() error {
	d.client.CloseIdleConnections()
	return nil
}

func parseHTTPOptions(opts map[string]interface{}) HTTPOptions {
	options := HTTPOptions{
		Method:          "POST",
		FollowRedirects: true,
		MaxRedirects:    10,
	}

	if opts == nil {
		return options
	}

	if method, ok := opts["method"].(string); ok {
		options.Method = method
	}
	if path, ok := opts["path"].(string); ok {
		options.Path = path
	}
	if qp, ok := opts["query_params"].(map[string]interface{}); ok {
		options.QueryParams = make(map[string]string)
		for k, v := range qp {
			options.QueryParams[k] = fmt.Sprintf("%v", v)
		}
	}
	if fr, ok := opts["follow_redirects"].(bool); ok {
		options.FollowRedirects = fr
	}
	if mr, ok := opts["max_redirects"].(float64); ok {
		options.MaxRedirects = int(mr)
	}
	if es, ok := opts["expected_statuses"].([]interface{}); ok {
		options.ExpectedStatuses = make([]int, 0, len(es))
		for _, s := range es {
			if status, ok := s.(float64); ok {
				options.ExpectedStatuses = append(options.ExpectedStatuses, int(status))
			}
		}
	}

	return options
}

func applyHTTPAuth(req *http.Request, auth *AuthConfig) {
	if auth == nil {
		return
	}

	switch auth.Type {
	case AuthBasic:
		username := auth.Credentials["username"]
		password := auth.Credentials["password"]
		req.SetBasicAuth(username, password)
	case AuthBearer:
		token := auth.Credentials["token"]
		req.Header.Set("Authorization", "Bearer "+token)
	case AuthAPIKey:
		header := auth.Credentials["header"]
		if header == "" {
			header = "X-API-Key"
		}
		key := auth.Credentials["key"]
		req.Header.Set(header, key)
	}
}

func buildTLSConfig(config *TLSConfig) *tls.Config {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	if config.InsecureSkipVerify {
		if os.Getenv("ALLOW_INSECURE_TLS") == "true" {
			log.Printf("AUDIT: TLS certificate verification disabled (InsecureSkipVerify=true, ALLOW_INSECURE_TLS=true) — this allows MITM attacks")
			tlsConfig.InsecureSkipVerify = true
		} else {
			log.Printf("WARNING: InsecureSkipVerify requested but ALLOW_INSECURE_TLS env var is not set to 'true' — ignoring, TLS verification remains enabled")
		}
	}

	if config.ServerName != "" {
		tlsConfig.ServerName = config.ServerName
	}

	return tlsConfig
}

func categorizeHTTPError(err error) DeliveryErrorType {
	errStr := err.Error()

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return ErrorTypeTimeout
	}
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		return ErrorTypeConnection
	}
	if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls") {
		return ErrorTypeTLS
	}

	return ErrorTypeConnection
}

func parseRetryAfter(value string) int {
	// Try parsing as seconds
	var seconds int
	if _, err := fmt.Sscanf(value, "%d", &seconds); err == nil {
		return seconds
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(value); err == nil {
		return int(time.Until(t).Seconds())
	}

	return 0
}
