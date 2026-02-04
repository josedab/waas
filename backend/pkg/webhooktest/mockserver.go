package webhooktest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// --- Mock Server ---

// MockServer captures incoming webhooks and returns configurable responses
type MockServer struct {
	mu            sync.RWMutex
	requests      []CapturedRequest
	responseCode  int
	responseBody  string
	responseDelay time.Duration
	maxCaptures   int
	handlers      map[string]ResponseConfig
}

// CapturedRequest represents a captured webhook request
type CapturedRequest struct {
	ID          string            `json:"id"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	Body        json.RawMessage   `json:"body"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	RemoteAddr  string            `json:"remote_addr"`
	ReceivedAt  time.Time         `json:"received_at"`
	BodySize    int               `json:"body_size"`
}

// ResponseConfig defines how the mock server should respond
type ResponseConfig struct {
	StatusCode int               `json:"status_code"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers,omitempty"`
	Delay      time.Duration     `json:"delay"`
}

// NewMockServer creates a new mock webhook server
func NewMockServer(opts ...MockOption) *MockServer {
	s := &MockServer{
		requests:     make([]CapturedRequest, 0, 100),
		responseCode: http.StatusOK,
		responseBody: `{"received": true}`,
		maxCaptures:  10000,
		handlers:     make(map[string]ResponseConfig),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// MockOption configures the mock server
type MockOption func(*MockServer)

// WithResponseCode sets the default response code
func WithResponseCode(code int) MockOption {
	return func(s *MockServer) { s.responseCode = code }
}

// WithResponseDelay sets the default response delay
func WithResponseDelay(d time.Duration) MockOption {
	return func(s *MockServer) { s.responseDelay = d }
}

// WithMaxCaptures sets the maximum number of requests to capture
func WithMaxCaptures(n int) MockOption {
	return func(s *MockServer) { s.maxCaptures = n }
}

// SetResponse configures the default response
func (s *MockServer) SetResponse(code int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseCode = code
	s.responseBody = body
}

// SetPathResponse configures a response for a specific path
func (s *MockServer) SetPathResponse(path string, config ResponseConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[path] = config
}

// ServeHTTP implements http.Handler
func (s *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, 0)
	if r.Body != nil {
		buf := make([]byte, 1024*1024)
		n, _ := r.Body.Read(buf)
		body = buf[:n]
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	captured := CapturedRequest{
		ID:         uuid.New().String(),
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    headers,
		Body:       json.RawMessage(body),
		RemoteAddr: r.RemoteAddr,
		ReceivedAt: time.Now(),
		BodySize:   len(body),
	}

	s.mu.Lock()
	if len(s.requests) < s.maxCaptures {
		s.requests = append(s.requests, captured)
	}
	s.mu.Unlock()

	s.mu.RLock()
	config, hasPathHandler := s.handlers[r.URL.Path]
	code := s.responseCode
	respBody := s.responseBody
	delay := s.responseDelay
	s.mu.RUnlock()

	if hasPathHandler {
		code = config.StatusCode
		respBody = config.Body
		delay = config.Delay
		for k, v := range config.Headers {
			w.Header().Set(k, v)
		}
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprint(w, respBody)
}

// GetRequests returns all captured requests
func (s *MockServer) GetRequests() []CapturedRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]CapturedRequest, len(s.requests))
	copy(result, s.requests)
	return result
}

// GetRequestCount returns the number of captured requests
func (s *MockServer) GetRequestCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.requests)
}

// Clear removes all captured requests
func (s *MockServer) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = s.requests[:0]
}

// --- Traffic Recording & Replay ---

// TrafficRecording stores a sequence of webhook requests for replay
type TrafficRecording struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Requests    []CapturedRequest `json:"requests"`
	TenantID    string            `json:"tenant_id"`
	Environment string            `json:"environment"`
	CreatedAt   time.Time         `json:"created_at"`
	Duration    time.Duration     `json:"duration"`
	TotalSize   int64             `json:"total_size_bytes"`
}

// PIIRedactionConfig defines which fields to redact
type PIIRedactionConfig struct {
	Fields   []string     `json:"fields"`
	Patterns []PIIPattern `json:"patterns"`
	Headers  []string     `json:"headers"`
}

// PIIPattern defines a regex pattern for PII detection
type PIIPattern struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Replace string `json:"replace"`
}

// DefaultPIIConfig returns default PII redaction patterns
func DefaultPIIConfig() *PIIRedactionConfig {
	return &PIIRedactionConfig{
		Fields: []string{"email", "phone", "ssn", "credit_card", "password", "secret", "token", "api_key"},
		Patterns: []PIIPattern{
			{Name: "email", Pattern: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, Replace: "[REDACTED_EMAIL]"},
			{Name: "credit_card", Pattern: `\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`, Replace: "[REDACTED_CC]"},
			{Name: "ssn", Pattern: `\b\d{3}-\d{2}-\d{4}\b`, Replace: "[REDACTED_SSN]"},
		},
		Headers: []string{"Authorization", "X-API-Key", "Cookie", "Set-Cookie"},
	}
}

// RedactPII applies PII redaction to a recording
func RedactPII(recording *TrafficRecording, config *PIIRedactionConfig) *TrafficRecording {
	if config == nil {
		config = DefaultPIIConfig()
	}

	redacted := *recording
	redacted.Requests = make([]CapturedRequest, len(recording.Requests))

	compiledPatterns := make([]*regexp.Regexp, 0, len(config.Patterns))
	replacements := make([]string, 0, len(config.Patterns))
	for _, p := range config.Patterns {
		if re, err := regexp.Compile(p.Pattern); err == nil {
			compiledPatterns = append(compiledPatterns, re)
			replacements = append(replacements, p.Replace)
		}
	}

	headerSet := make(map[string]bool, len(config.Headers))
	for _, h := range config.Headers {
		headerSet[strings.ToLower(h)] = true
	}

	for i, req := range recording.Requests {
		req := req

		// Redact headers
		redactedHeaders := make(map[string]string, len(req.Headers))
		for k, v := range req.Headers {
			if headerSet[strings.ToLower(k)] {
				redactedHeaders[k] = "[REDACTED]"
			} else {
				redactedHeaders[k] = v
			}
		}
		req.Headers = redactedHeaders

		// Redact body using patterns
		bodyStr := string(req.Body)
		for j, re := range compiledPatterns {
			bodyStr = re.ReplaceAllString(bodyStr, replacements[j])
		}

		// Redact known field names in JSON
		var bodyMap map[string]interface{}
		if err := json.Unmarshal([]byte(bodyStr), &bodyMap); err == nil {
			redactFieldsInMap(bodyMap, config.Fields)
			if redactedBody, err := json.Marshal(bodyMap); err == nil {
				req.Body = redactedBody
			}
		} else {
			req.Body = json.RawMessage(bodyStr)
		}

		req.RemoteAddr = "[REDACTED]"
		redacted.Requests[i] = req
	}

	return &redacted
}

func redactFieldsInMap(data map[string]interface{}, fields []string) {
	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[strings.ToLower(f)] = true
	}
	for k, v := range data {
		if fieldSet[strings.ToLower(k)] {
			data[k] = "[REDACTED]"
			continue
		}
		if nested, ok := v.(map[string]interface{}); ok {
			redactFieldsInMap(nested, fields)
		}
	}
}

// ReplayRecording replays a traffic recording against a target URL
func ReplayRecording(ctx context.Context, recording *TrafficRecording, targetURL string) ([]ReplayResult, error) {
	if recording == nil || len(recording.Requests) == 0 {
		return nil, fmt.Errorf("empty recording")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	results := make([]ReplayResult, 0, len(recording.Requests))

	for _, req := range recording.Requests {
		start := time.Now()
		httpReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL+req.Path, strings.NewReader(string(req.Body)))
		if err != nil {
			results = append(results, ReplayResult{RequestID: req.ID, Error: err.Error()})
			continue
		}
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}

		resp, err := client.Do(httpReq)
		duration := time.Since(start)

		result := ReplayResult{RequestID: req.ID, DurationMs: duration.Milliseconds()}
		if err != nil {
			result.Error = err.Error()
		} else {
			result.StatusCode = resp.StatusCode
			result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
			resp.Body.Close()
		}
		results = append(results, result)
	}

	return results, nil
}

// ReplayResult contains the result of replaying a single request
type ReplayResult struct {
	RequestID  string `json:"request_id"`
	StatusCode int    `json:"status_code,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// --- k6 Load Test Script Generation ---

// GenerateK6Script generates a k6 load test script from recorded traffic
func GenerateK6Script(recording *TrafficRecording, targetURL string, vus int, duration string) string {
	if vus <= 0 {
		vus = 10
	}
	if duration == "" {
		duration = "30s"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Generated from WaaS traffic recording: %s
// Recording: %s (%d requests)

const errorRate = new Rate('errors');
const deliveryTime = new Trend('delivery_time', true);

export const options = {
  vus: %d,
  duration: '%s',
  thresholds: {
    errors: ['rate<0.01'],
    delivery_time: ['p(95)<500', 'p(99)<1000'],
    http_req_duration: ['p(95)<500'],
  },
};

const TARGET_URL = '%s';

const requests = [
`, recording.ID, recording.Name, len(recording.Requests), vus, duration, targetURL))

	for i, req := range recording.Requests {
		bodyStr := strings.ReplaceAll(string(req.Body), "`", "\\`")
		bodyStr = strings.ReplaceAll(bodyStr, "${", "\\${")
		sb.WriteString(fmt.Sprintf("  { method: '%s', path: '%s', body: `%s` },\n", req.Method, req.Path, bodyStr))
		if i >= 49 {
			sb.WriteString("  // ... truncated\n")
			break
		}
	}

	sb.WriteString(`];

export default function () {
  const req = requests[Math.floor(Math.random() * requests.length)];
  const url = TARGET_URL + req.path;

  const start = Date.now();
  const res = http.request(req.method, url, req.body, {
    headers: { 'Content-Type': 'application/json' },
  });
  deliveryTime.add(Date.now() - start);

  const success = check(res, {
    'status is 2xx': (r) => r.status >= 200 && r.status < 300,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });

  errorRate.add(!success);
  sleep(0.1);
}
`)

	return sb.String()
}
