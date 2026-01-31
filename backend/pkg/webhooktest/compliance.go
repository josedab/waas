package webhooktest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ComplianceLevel indicates how well an endpoint adheres to webhook best practices
type ComplianceLevel string

const (
	CompliancePlatinum ComplianceLevel = "platinum"
	ComplianceGold     ComplianceLevel = "gold"
	ComplianceSilver   ComplianceLevel = "silver"
	ComplianceBronze   ComplianceLevel = "bronze"
	ComplianceFailing  ComplianceLevel = "failing"
)

// ComplianceCheck represents an individual compliance check
type ComplianceCheck struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Category    string           `json:"category"`
	Description string           `json:"description"`
	Severity    string           `json:"severity"` // critical, warning, info
	Status      ComplianceStatus `json:"status"`
	Details     string           `json:"details,omitempty"`
	DurationMs  float64          `json:"duration_ms"`
}

// ComplianceStatus is the result of a single check
type ComplianceStatus string

const (
	CheckPassed  ComplianceStatus = "passed"
	CheckFailed  ComplianceStatus = "failed"
	CheckWarning ComplianceStatus = "warning"
	CheckSkipped ComplianceStatus = "skipped"
)

// ComplianceReport is the full compliance report for an endpoint
type ComplianceReport struct {
	EndpointURL     string            `json:"endpoint_url"`
	Level           ComplianceLevel   `json:"level"`
	Score           int               `json:"score"`
	MaxScore        int               `json:"max_score"`
	Checks          []ComplianceCheck `json:"checks"`
	Passed          int               `json:"passed"`
	Failed          int               `json:"failed"`
	Warnings        int               `json:"warnings"`
	Skipped         int               `json:"skipped"`
	DurationMs      float64           `json:"duration_ms"`
	Timestamp       time.Time         `json:"timestamp"`
	BadgeURL        string            `json:"badge_url,omitempty"`
	RecommendActions []string          `json:"recommended_actions,omitempty"`
}

// ComplianceRunner runs the full compliance suite against a webhook endpoint
type ComplianceRunner struct {
	client     *http.Client
	signingKey string
}

// NewComplianceRunner creates a new compliance runner
func NewComplianceRunner(signingKey string) *ComplianceRunner {
	return &ComplianceRunner{
		client: &http.Client{Timeout: 30 * time.Second},
		signingKey: signingKey,
	}
}

// RunCompliance runs all compliance checks against the target endpoint
func (cr *ComplianceRunner) RunCompliance(endpointURL string) *ComplianceReport {
	start := time.Now()
	report := &ComplianceReport{
		EndpointURL: endpointURL,
		Timestamp:   start,
	}

	checks := cr.allChecks(endpointURL)
	report.Checks = checks
	report.MaxScore = len(checks) * 10

	score := 0
	for _, c := range checks {
		switch c.Status {
		case CheckPassed:
			report.Passed++
			score += 10
		case CheckFailed:
			report.Failed++
		case CheckWarning:
			report.Warnings++
			score += 5
		case CheckSkipped:
			report.Skipped++
		}
	}

	report.Score = score
	report.DurationMs = float64(time.Since(start).Milliseconds())
	report.Level = calculateLevel(score, report.MaxScore)
	report.BadgeURL = fmt.Sprintf("https://img.shields.io/badge/WaaS%%20Compliance-%s-%s",
		string(report.Level), levelColor(report.Level))
	report.RecommendActions = cr.recommendations(checks)

	return report
}

func (cr *ComplianceRunner) allChecks(url string) []ComplianceCheck {
	var checks []ComplianceCheck

	// Category: Connectivity
	checks = append(checks, cr.checkEndpointReachable(url))
	checks = append(checks, cr.checkHTTPS(url))
	checks = append(checks, cr.checkResponseTime(url))
	checks = append(checks, cr.checkResponseTimeP99(url))

	// Category: HTTP Behavior
	checks = append(checks, cr.checkReturns2xx(url))
	checks = append(checks, cr.checkContentTypeHandling(url))
	checks = append(checks, cr.checkMethodValidation(url))
	checks = append(checks, cr.checkLargePayload(url))

	// Category: Signature Verification
	checks = append(checks, cr.checkAcceptsValidSignature(url))
	checks = append(checks, cr.checkRejectsInvalidSignature(url))
	checks = append(checks, cr.checkRejectsExpiredTimestamp(url))
	checks = append(checks, cr.checkRejectsMissingSignature(url))

	// Category: Idempotency
	checks = append(checks, cr.checkIdempotency(url))
	checks = append(checks, cr.checkDuplicateHandling(url))

	// Category: Timeouts & Resilience
	checks = append(checks, cr.checkQuickResponse(url))
	checks = append(checks, cr.checkRetryAfterHeader(url))
	checks = append(checks, cr.checkGracefulErrors(url))

	// Category: Security
	checks = append(checks, cr.checkNoSensitiveDataInResponse(url))
	checks = append(checks, cr.checkSecurityHeaders(url))

	// Category: Reliability
	checks = append(checks, cr.checkConcurrentDelivery(url))
	checks = append(checks, cr.checkEmptyPayload(url))
	checks = append(checks, cr.checkMalformedJSON(url))

	return checks
}

func (cr *ComplianceRunner) checkEndpointReachable(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "conn-001", Name: "Endpoint Reachable", Category: "connectivity", Severity: "critical"}
	start := time.Now()
	resp, err := cr.sendWebhook(url, `{"event":"ping"}`, nil)
	c.DurationMs = float64(time.Since(start).Milliseconds())
	if err != nil {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Connection failed: %v", err)
		return c
	}
	resp.Body.Close()
	c.Status = CheckPassed
	c.Details = fmt.Sprintf("Connected in %.0fms", c.DurationMs)
	return c
}

func (cr *ComplianceRunner) checkHTTPS(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "conn-002", Name: "HTTPS Enabled", Category: "connectivity", Severity: "critical"}
	if strings.HasPrefix(url, "https://") {
		c.Status = CheckPassed
		c.Details = "Endpoint uses HTTPS"
	} else {
		c.Status = CheckWarning
		c.Details = "Endpoint does not use HTTPS — recommended for production"
	}
	return c
}

func (cr *ComplianceRunner) checkResponseTime(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "conn-003", Name: "Response Time < 5s", Category: "connectivity", Severity: "warning"}
	start := time.Now()
	resp, err := cr.sendWebhook(url, `{"event":"latency_check"}`, nil)
	c.DurationMs = float64(time.Since(start).Milliseconds())
	if err != nil {
		c.Status = CheckSkipped
		c.Details = "Could not connect"
		return c
	}
	resp.Body.Close()
	if c.DurationMs < 5000 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Response in %.0fms", c.DurationMs)
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Response took %.0fms (max: 5000ms)", c.DurationMs)
	}
	return c
}

func (cr *ComplianceRunner) checkResponseTimeP99(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "conn-004", Name: "P99 Latency < 10s", Category: "connectivity", Severity: "info"}
	var durations []float64
	for i := 0; i < 3; i++ {
		start := time.Now()
		resp, err := cr.sendWebhook(url, `{"event":"p99_check","iteration":`+fmt.Sprintf("%d", i)+`}`, nil)
		d := float64(time.Since(start).Milliseconds())
		durations = append(durations, d)
		if err == nil {
			resp.Body.Close()
		}
	}
	maxDuration := 0.0
	for _, d := range durations {
		if d > maxDuration {
			maxDuration = d
		}
	}
	c.DurationMs = maxDuration
	if maxDuration < 10000 {
		c.Status = CheckPassed
	} else {
		c.Status = CheckWarning
	}
	c.Details = fmt.Sprintf("Max latency: %.0fms over %d requests", maxDuration, len(durations))
	return c
}

func (cr *ComplianceRunner) checkReturns2xx(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "http-001", Name: "Returns 2xx on Success", Category: "http", Severity: "critical"}
	resp, err := cr.sendWebhook(url, `{"event":"test"}`, nil)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Returned HTTP %d", resp.StatusCode)
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Returned HTTP %d instead of 2xx", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) checkContentTypeHandling(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "http-002", Name: "Handles application/json", Category: "http", Severity: "warning"}
	headers := map[string]string{"Content-Type": "application/json"}
	resp, err := cr.sendWebhook(url, `{"event":"content_type_test"}`, headers)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.Status = CheckPassed
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Returned %d for application/json content", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) checkMethodValidation(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "http-003", Name: "Rejects non-POST methods", Category: "http", Severity: "info"}
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := cr.client.Do(req)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode == 405 || resp.StatusCode == 404 || resp.StatusCode >= 400 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("GET returned HTTP %d", resp.StatusCode)
	} else {
		c.Status = CheckWarning
		c.Details = fmt.Sprintf("GET returned HTTP %d — should reject non-POST", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) checkLargePayload(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "http-004", Name: "Handles Large Payloads (100KB)", Category: "http", Severity: "warning"}
	largeData := strings.Repeat("x", 100*1024)
	payload := fmt.Sprintf(`{"event":"large_payload","data":"%s"}`, largeData)
	resp, err := cr.sendWebhook(url, payload, nil)
	if err != nil {
		c.Status = CheckWarning
		c.Details = "Failed to send large payload"
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Accepted large payload (HTTP %d)", resp.StatusCode)
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Server error on large payload (HTTP %d)", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) checkAcceptsValidSignature(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "sig-001", Name: "Accepts Valid Signature", Category: "signatures", Severity: "critical"}
	if cr.signingKey == "" {
		c.Status = CheckSkipped
		c.Details = "No signing key configured"
		return c
	}
	payload := `{"event":"sig_valid_test"}`
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := cr.sign(payload, ts)
	headers := map[string]string{
		"X-WaaS-Signature": sig,
		"X-WaaS-Timestamp": ts,
	}
	resp, err := cr.sendWebhook(url, payload, headers)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.Status = CheckPassed
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Valid signature rejected (HTTP %d)", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) checkRejectsInvalidSignature(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "sig-002", Name: "Rejects Invalid Signature", Category: "signatures", Severity: "critical"}
	if cr.signingKey == "" {
		c.Status = CheckSkipped
		c.Details = "No signing key configured"
		return c
	}
	headers := map[string]string{
		"X-WaaS-Signature": "v1=0000000000000000000000000000000000000000000000000000000000000000",
		"X-WaaS-Timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	}
	resp, err := cr.sendWebhook(url, `{"event":"sig_invalid_test"}`, headers)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Correctly rejected with HTTP %d", resp.StatusCode)
	} else {
		c.Status = CheckWarning
		c.Details = fmt.Sprintf("Returned HTTP %d for invalid signature — should return 401/403", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) checkRejectsExpiredTimestamp(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "sig-003", Name: "Rejects Expired Timestamp", Category: "signatures", Severity: "warning"}
	if cr.signingKey == "" {
		c.Status = CheckSkipped
		c.Details = "No signing key configured"
		return c
	}
	oldTs := fmt.Sprintf("%d", time.Now().Add(-1*time.Hour).Unix())
	payload := `{"event":"sig_expired_test"}`
	sig := cr.sign(payload, oldTs)
	headers := map[string]string{
		"X-WaaS-Signature": sig,
		"X-WaaS-Timestamp": oldTs,
	}
	resp, err := cr.sendWebhook(url, payload, headers)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		c.Status = CheckPassed
	} else {
		c.Status = CheckWarning
		c.Details = "Accepted expired timestamp — consider implementing replay protection"
	}
	return c
}

func (cr *ComplianceRunner) checkRejectsMissingSignature(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "sig-004", Name: "Rejects Missing Signature", Category: "signatures", Severity: "warning"}
	if cr.signingKey == "" {
		c.Status = CheckSkipped
		c.Details = "No signing key configured"
		return c
	}
	resp, err := cr.sendWebhook(url, `{"event":"sig_missing_test"}`, nil)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		c.Status = CheckPassed
		c.Details = "Correctly requires signature"
	} else {
		c.Status = CheckWarning
		c.Details = "Accepted request without signature — consider requiring signatures"
	}
	return c
}

func (cr *ComplianceRunner) checkIdempotency(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "idem-001", Name: "Idempotent Processing", Category: "idempotency", Severity: "warning"}
	idempotencyKey := "test-idem-" + fmt.Sprintf("%d", time.Now().UnixNano())
	headers := map[string]string{"X-WaaS-Idempotency-Key": idempotencyKey}
	payload := `{"event":"idempotency_test","id":"` + idempotencyKey + `"}`

	resp1, err := cr.sendWebhook(url, payload, headers)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	resp2, err := cr.sendWebhook(url, payload, headers)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if resp1.StatusCode == resp2.StatusCode {
		c.Status = CheckPassed
		c.Details = "Same response for duplicate delivery"
	} else {
		c.Status = CheckWarning
		c.Details = fmt.Sprintf("Different responses: %d vs %d (bodies: %s vs %s)",
			resp1.StatusCode, resp2.StatusCode, truncateStr(string(body1), 50), truncateStr(string(body2), 50))
	}
	return c
}

func (cr *ComplianceRunner) checkDuplicateHandling(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "idem-002", Name: "Duplicate Delivery Tolerance", Category: "idempotency", Severity: "info"}
	payload := `{"event":"dup_test","id":"dup-` + fmt.Sprintf("%d", time.Now().UnixNano()) + `"}`
	var statuses []int
	for i := 0; i < 3; i++ {
		resp, err := cr.sendWebhook(url, payload, nil)
		if err != nil {
			c.Status = CheckSkipped
			return c
		}
		statuses = append(statuses, resp.StatusCode)
		resp.Body.Close()
	}
	allSame := true
	for _, s := range statuses {
		if s != statuses[0] {
			allSame = false
		}
	}
	if allSame && statuses[0] >= 200 && statuses[0] < 300 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Consistent HTTP %d across 3 duplicate deliveries", statuses[0])
	} else {
		c.Status = CheckWarning
		c.Details = fmt.Sprintf("Inconsistent responses for duplicates: %v", statuses)
	}
	return c
}

func (cr *ComplianceRunner) checkQuickResponse(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "time-001", Name: "Response Within 30s", Category: "timeouts", Severity: "critical"}
	start := time.Now()
	resp, err := cr.sendWebhook(url, `{"event":"timeout_test"}`, nil)
	c.DurationMs = float64(time.Since(start).Milliseconds())
	if err != nil {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Request failed or timed out: %v", err)
		return c
	}
	resp.Body.Close()
	if c.DurationMs < 30000 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Responded in %.0fms", c.DurationMs)
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Response took %.0fms — should respond within 30s", c.DurationMs)
	}
	return c
}

func (cr *ComplianceRunner) checkRetryAfterHeader(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "time-002", Name: "Retry-After Header Support", Category: "timeouts", Severity: "info"}
	// Send many requests quickly to potentially trigger rate limiting
	for i := 0; i < 5; i++ {
		resp, err := cr.sendWebhook(url, `{"event":"retry_after_test"}`, nil)
		if err != nil {
			continue
		}
		if resp.Header.Get("Retry-After") != "" {
			c.Status = CheckPassed
			c.Details = "Provides Retry-After header when rate limited"
			resp.Body.Close()
			return c
		}
		if resp.StatusCode == 429 {
			c.Status = CheckWarning
			c.Details = "Returns 429 but without Retry-After header"
			resp.Body.Close()
			return c
		}
		resp.Body.Close()
	}
	c.Status = CheckPassed
	c.Details = "No rate limiting triggered (acceptable for test traffic)"
	return c
}

func (cr *ComplianceRunner) checkGracefulErrors(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "time-003", Name: "Graceful Error Responses", Category: "timeouts", Severity: "warning"}
	resp, err := cr.sendWebhook(url, `INVALID JSON }{`, nil)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Returns HTTP %d for invalid JSON", resp.StatusCode)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.Status = CheckWarning
		c.Details = "Accepted invalid JSON — consider validating payload"
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Server error (HTTP %d) for invalid JSON", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) checkNoSensitiveDataInResponse(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "sec-001", Name: "No Sensitive Data in Response", Category: "security", Severity: "critical"}
	resp, err := cr.sendWebhook(url, `{"event":"security_test"}`, nil)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := strings.ToLower(string(body))
	sensitivePatterns := []string{"password", "secret", "api_key", "private_key", "access_token"}
	for _, p := range sensitivePatterns {
		if strings.Contains(bodyStr, p) {
			c.Status = CheckFailed
			c.Details = fmt.Sprintf("Response body contains sensitive pattern: %q", p)
			return c
		}
	}
	c.Status = CheckPassed
	c.Details = "No sensitive data patterns found in response"
	return c
}

func (cr *ComplianceRunner) checkSecurityHeaders(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "sec-002", Name: "Security Response Headers", Category: "security", Severity: "info"}
	resp, err := cr.sendWebhook(url, `{"event":"headers_test"}`, nil)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()

	score := 0
	total := 3
	if resp.Header.Get("X-Content-Type-Options") != "" {
		score++
	}
	if resp.Header.Get("X-Frame-Options") != "" {
		score++
	}
	if resp.Header.Get("Strict-Transport-Security") != "" {
		score++
	}

	if score == total {
		c.Status = CheckPassed
	} else if score > 0 {
		c.Status = CheckWarning
	} else {
		c.Status = CheckWarning
	}
	c.Details = fmt.Sprintf("%d/%d security headers present", score, total)
	return c
}

func (cr *ComplianceRunner) checkConcurrentDelivery(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "rel-001", Name: "Handles Concurrent Deliveries", Category: "reliability", Severity: "warning"}
	var wg sync.WaitGroup
	results := make([]int, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := cr.sendWebhook(url, fmt.Sprintf(`{"event":"concurrent_test","idx":%d}`, idx), nil)
			if err != nil {
				results[idx] = -1
				return
			}
			results[idx] = resp.StatusCode
			resp.Body.Close()
		}(i)
	}
	wg.Wait()

	success := 0
	for _, code := range results {
		if code >= 200 && code < 300 {
			success++
		}
	}
	if success == 5 {
		c.Status = CheckPassed
		c.Details = "All 5 concurrent requests succeeded"
	} else if success >= 3 {
		c.Status = CheckWarning
		c.Details = fmt.Sprintf("%d/5 concurrent requests succeeded", success)
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Only %d/5 concurrent requests succeeded", success)
	}
	return c
}

func (cr *ComplianceRunner) checkEmptyPayload(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "rel-002", Name: "Handles Empty Payload", Category: "reliability", Severity: "warning"}
	resp, err := cr.sendWebhook(url, `{}`, nil)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode < 500 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Handled empty payload (HTTP %d)", resp.StatusCode)
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Server error on empty payload (HTTP %d)", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) checkMalformedJSON(url string) ComplianceCheck {
	c := ComplianceCheck{ID: "rel-003", Name: "Handles Malformed JSON", Category: "reliability", Severity: "warning"}
	resp, err := cr.sendWebhook(url, `{"broken":`, nil)
	if err != nil {
		c.Status = CheckSkipped
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode < 500 {
		c.Status = CheckPassed
		c.Details = fmt.Sprintf("Gracefully handled malformed JSON (HTTP %d)", resp.StatusCode)
	} else {
		c.Status = CheckFailed
		c.Details = fmt.Sprintf("Server error on malformed JSON (HTTP %d)", resp.StatusCode)
	}
	return c
}

func (cr *ComplianceRunner) sendWebhook(url, payload string, extraHeaders map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WaaS-Compliance-Runner/1.0")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	return cr.client.Do(req)
}

func (cr *ComplianceRunner) sign(payload, timestamp string) string {
	signedPayload := timestamp + "." + payload
	mac := hmac.New(sha256.New, []byte(cr.signingKey))
	mac.Write([]byte(signedPayload))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func calculateLevel(score, maxScore int) ComplianceLevel {
	if maxScore == 0 {
		return ComplianceFailing
	}
	pct := float64(score) / float64(maxScore) * 100
	switch {
	case pct >= 95:
		return CompliancePlatinum
	case pct >= 80:
		return ComplianceGold
	case pct >= 60:
		return ComplianceSilver
	case pct >= 40:
		return ComplianceBronze
	default:
		return ComplianceFailing
	}
}

func levelColor(level ComplianceLevel) string {
	switch level {
	case CompliancePlatinum:
		return "brightgreen"
	case ComplianceGold:
		return "green"
	case ComplianceSilver:
		return "yellow"
	case ComplianceBronze:
		return "orange"
	default:
		return "red"
	}
}

func (cr *ComplianceRunner) recommendations(checks []ComplianceCheck) []string {
	var recs []string
	for _, c := range checks {
		if c.Status == CheckFailed || c.Status == CheckWarning {
			switch c.ID {
			case "conn-002":
				recs = append(recs, "Enable HTTPS for production webhook endpoints")
			case "sig-002":
				recs = append(recs, "Implement signature verification to prevent spoofed webhooks")
			case "sig-003":
				recs = append(recs, "Add timestamp validation to prevent replay attacks")
			case "idem-001":
				recs = append(recs, "Implement idempotency keys to safely handle duplicate deliveries")
			case "time-002":
				recs = append(recs, "Include Retry-After header in rate limit responses")
			case "rel-001":
				recs = append(recs, "Improve concurrent request handling for burst traffic")
			case "sec-002":
				recs = append(recs, "Add security response headers (X-Content-Type-Options, X-Frame-Options)")
			}
		}
	}
	return recs
}

// FormatComplianceReport formats a compliance report as a human-readable string
func FormatComplianceReport(r *ComplianceReport) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n═══ WaaS Webhook Compliance Report ═══\n"))
	b.WriteString(fmt.Sprintf("Endpoint: %s\n", r.EndpointURL))
	b.WriteString(fmt.Sprintf("Level:    %s (%d/%d points)\n", strings.ToUpper(string(r.Level)), r.Score, r.MaxScore))
	b.WriteString(fmt.Sprintf("Duration: %.0fms\n\n", r.DurationMs))

	categories := map[string][]ComplianceCheck{}
	for _, c := range r.Checks {
		categories[c.Category] = append(categories[c.Category], c)
	}

	for cat, checks := range categories {
		b.WriteString(fmt.Sprintf("── %s ──\n", strings.ToUpper(cat)))
		for _, c := range checks {
			icon := "✓"
			switch c.Status {
			case CheckFailed:
				icon = "✗"
			case CheckWarning:
				icon = "⚠"
			case CheckSkipped:
				icon = "○"
			}
			b.WriteString(fmt.Sprintf("  %s %s", icon, c.Name))
			if c.Details != "" {
				b.WriteString(fmt.Sprintf(" — %s", c.Details))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("Summary: %d passed, %d failed, %d warnings, %d skipped\n", r.Passed, r.Failed, r.Warnings, r.Skipped))
	b.WriteString(fmt.Sprintf("Badge: %s\n", r.BadgeURL))

	if len(r.RecommendActions) > 0 {
		b.WriteString("\nRecommended Actions:\n")
		for i, rec := range r.RecommendActions {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, rec))
		}
	}

	return b.String()
}

// ToComplianceJSON outputs the report as JSON for CI integration
func ToComplianceJSON(r *ComplianceReport) (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
