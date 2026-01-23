package ai

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Classifier provides error classification capabilities
type Classifier struct {
	patterns       []classificationPattern
	mu             sync.RWMutex
	outcomeHistory map[string]*outcomeStats
}

type classificationPattern struct {
	regex      *regexp.Regexp
	category   ErrorCategory
	subCategory string
	confidence float64
	retryable  bool
	delay      int
}

type outcomeStats struct {
	total    int
	correct  int
	category ErrorCategory
}

// NewClassifier creates a new error classifier with predefined patterns
func NewClassifier() *Classifier {
	c := &Classifier{
		outcomeHistory: make(map[string]*outcomeStats),
	}
	c.initPatterns()
	return c
}

func (c *Classifier) initPatterns() {
	c.patterns = []classificationPattern{
		// Network errors
		{regexp.MustCompile(`(?i)connection refused`), CategoryNetwork, "connection_refused", 0.95, true, 30},
		{regexp.MustCompile(`(?i)connection reset`), CategoryNetwork, "connection_reset", 0.95, true, 15},
		{regexp.MustCompile(`(?i)no route to host`), CategoryNetwork, "no_route", 0.95, true, 60},
		{regexp.MustCompile(`(?i)network is unreachable`), CategoryNetwork, "unreachable", 0.95, true, 60},
		{regexp.MustCompile(`(?i)i/o timeout`), CategoryNetwork, "io_timeout", 0.90, true, 30},
		{regexp.MustCompile(`(?i)broken pipe`), CategoryNetwork, "broken_pipe", 0.90, true, 15},
		{regexp.MustCompile(`(?i)connection closed`), CategoryNetwork, "connection_closed", 0.90, true, 15},
		{regexp.MustCompile(`(?i)host is down`), CategoryNetwork, "host_down", 0.95, true, 60},
		{regexp.MustCompile(`(?i)protocol error`), CategoryNetwork, "protocol_error", 0.85, true, 30},
		
		// Timeout errors
		{regexp.MustCompile(`(?i)context deadline exceeded`), CategoryTimeout, "deadline", 0.95, true, 60},
		{regexp.MustCompile(`(?i)timeout|timed out`), CategoryTimeout, "generic", 0.85, true, 30},
		{regexp.MustCompile(`(?i)request timeout`), CategoryTimeout, "request", 0.90, true, 30},
		{regexp.MustCompile(`(?i)gateway timeout`), CategoryTimeout, "gateway", 0.90, true, 60},
		
		// DNS errors
		{regexp.MustCompile(`(?i)no such host`), CategoryDNS, "resolution_failed", 0.95, true, 120},
		{regexp.MustCompile(`(?i)dns lookup|lookup .* on`), CategoryDNS, "lookup_failed", 0.90, true, 120},
		{regexp.MustCompile(`(?i)temporary failure in name resolution`), CategoryDNS, "temporary", 0.90, true, 60},
		
		// Certificate errors
		{regexp.MustCompile(`(?i)certificate.*(expired|invalid|untrusted)`), CategoryCertificate, "invalid", 0.95, false, 0},
		{regexp.MustCompile(`(?i)certificate signed by unknown authority`), CategoryCertificate, "unknown_ca", 0.95, false, 0},
		{regexp.MustCompile(`(?i)certificate.*revoked`), CategoryCertificate, "revoked", 0.95, false, 0},
		{regexp.MustCompile(`(?i)ssl.*handshake`), CategoryCertificate, "ssl_handshake", 0.90, false, 0},
		{regexp.MustCompile(`(?i)tls.*handshake`), CategoryCertificate, "tls_handshake", 0.90, false, 0},
		{regexp.MustCompile(`(?i)x509|tls|ssl.*error`), CategoryCertificate, "tls_error", 0.85, false, 0},
		
		// Authentication errors
		{regexp.MustCompile(`(?i)401|unauthorized`), CategoryAuth, "unauthorized", 0.90, false, 0},
		{regexp.MustCompile(`(?i)403|forbidden`), CategoryAuth, "forbidden", 0.90, false, 0},
		{regexp.MustCompile(`(?i)invalid.*api.?key`), CategoryAuth, "invalid_key", 0.95, false, 0},
		{regexp.MustCompile(`(?i)authentication.*failed`), CategoryAuth, "auth_failed", 0.90, false, 0},
		{regexp.MustCompile(`(?i)invalid.*token`), CategoryAuth, "invalid_token", 0.90, false, 0},
		
		// Rate limiting
		{regexp.MustCompile(`(?i)429|too many requests`), CategoryRateLimit, "throttled", 0.95, true, 60},
		{regexp.MustCompile(`(?i)rate.?limit`), CategoryRateLimit, "rate_limited", 0.90, true, 60},
		{regexp.MustCompile(`(?i)quota exceeded`), CategoryRateLimit, "quota", 0.90, true, 300},
		
		// Server errors
		{regexp.MustCompile(`(?i)500|internal server error`), CategoryServerError, "internal", 0.85, true, 30},
		{regexp.MustCompile(`(?i)502|bad gateway`), CategoryServerError, "bad_gateway", 0.90, true, 30},
		{regexp.MustCompile(`(?i)503|service unavailable`), CategoryServerError, "unavailable", 0.90, true, 60},
		{regexp.MustCompile(`(?i)504|gateway timeout`), CategoryServerError, "gateway_timeout", 0.90, true, 60},
		{regexp.MustCompile(`(?i)invalid.*response`), CategoryServerError, "invalid_response", 0.80, true, 30},
		{regexp.MustCompile(`(?i)unexpected.*eof`), CategoryServerError, "unexpected_eof", 0.85, true, 15},
		
		// Client errors
		{regexp.MustCompile(`(?i)400|bad request`), CategoryClientError, "bad_request", 0.85, false, 0},
		{regexp.MustCompile(`(?i)404|not found`), CategoryClientError, "not_found", 0.90, false, 0},
		{regexp.MustCompile(`(?i)405|method not allowed`), CategoryClientError, "method_not_allowed", 0.95, false, 0},
		{regexp.MustCompile(`(?i)406|not acceptable`), CategoryClientError, "not_acceptable", 0.90, false, 0},
		{regexp.MustCompile(`(?i)415|unsupported media type`), CategoryClientError, "unsupported_media", 0.95, false, 0},
		{regexp.MustCompile(`(?i)422|unprocessable`), CategoryClientError, "unprocessable", 0.90, false, 0},
		{regexp.MustCompile(`(?i)301|302|303|307|308|moved|redirect`), CategoryClientError, "redirect", 0.85, false, 0},
		{regexp.MustCompile(`(?i)413|payload too large|request entity too large`), CategoryClientError, "payload_too_large", 0.90, false, 0},
		
		// Payload errors
		{regexp.MustCompile(`(?i)invalid.*json`), CategoryPayload, "invalid_json", 0.90, false, 0},
		{regexp.MustCompile(`(?i)malformed.*payload`), CategoryPayload, "malformed", 0.90, false, 0},
		{regexp.MustCompile(`(?i)missing.*field|required.*field`), CategoryPayload, "missing_field", 0.85, false, 0},
		{regexp.MustCompile(`(?i)payload.*too.*large|entity.*too.*large`), CategoryPayload, "too_large", 0.90, false, 0},
		{regexp.MustCompile(`(?i)content.?length.*exceeded`), CategoryPayload, "size_exceeded", 0.90, false, 0},
	}
}

// Classify classifies an error based on the error message and HTTP status
func (c *Classifier) Classify(errorMessage string, httpStatus *int, responseBody string) ErrorClassification {
	combined := errorMessage
	if responseBody != "" {
		combined += " " + responseBody
	}
	
	// Check patterns
	for _, p := range c.patterns {
		if p.regex.MatchString(combined) {
			return ErrorClassification{
				Category:       p.category,
				Confidence:     p.confidence,
				SubCategory:    p.subCategory,
				IsRetryable:    p.retryable,
				SuggestedDelay: p.delay,
			}
		}
	}
	
	// Fallback to HTTP status code classification
	if httpStatus != nil {
		return c.classifyByStatusCode(*httpStatus)
	}
	
	return ErrorClassification{
		Category:    CategoryUnknown,
		Confidence:  0.5,
		IsRetryable: true,
		SuggestedDelay: 60,
	}
}

func (c *Classifier) classifyByStatusCode(status int) ErrorClassification {
	switch {
	case status == 401:
		return ErrorClassification{CategoryAuth, 0.90, "unauthorized", false, 0}
	case status == 403:
		return ErrorClassification{CategoryAuth, 0.90, "forbidden", false, 0}
	case status == 404:
		return ErrorClassification{CategoryClientError, 0.90, "not_found", false, 0}
	case status == 408:
		return ErrorClassification{CategoryTimeout, 0.90, "request_timeout", true, 30}
	case status == 429:
		return ErrorClassification{CategoryRateLimit, 0.95, "throttled", true, 60}
	case status >= 400 && status < 500:
		return ErrorClassification{CategoryClientError, 0.75, "client_error", false, 0}
	case status == 500:
		return ErrorClassification{CategoryServerError, 0.85, "internal", true, 30}
	case status == 502:
		return ErrorClassification{CategoryServerError, 0.90, "bad_gateway", true, 30}
	case status == 503:
		return ErrorClassification{CategoryServerError, 0.90, "unavailable", true, 60}
	case status == 504:
		return ErrorClassification{CategoryServerError, 0.90, "gateway_timeout", true, 60}
	case status >= 500:
		return ErrorClassification{CategoryServerError, 0.75, "server_error", true, 30}
	default:
		return ErrorClassification{CategoryUnknown, 0.5, "", true, 60}
	}
}

// GetSuggestions returns fix suggestions based on classification
func (c *Classifier) GetSuggestions(classification ErrorClassification, ctx *DeliveryContext) []Suggestion {
	var suggestions []Suggestion
	
	switch classification.Category {
	case CategoryNetwork:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Verify Endpoint Connectivity",
			Description: "Check if the endpoint URL is reachable from your network.",
			Action:      ActionCheckDNS,
		})
		if classification.IsRetryable {
			suggestions = append(suggestions, Suggestion{
				Priority:    2,
				Title:       "Automatic Retry Scheduled",
				Description: "The delivery will be automatically retried with exponential backoff.",
				Action:      ActionRetry,
			})
		}
		
	case CategoryTimeout:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Increase Timeout",
			Description: "Consider increasing the endpoint timeout if the target service requires more processing time.",
			Action:      ActionUpdateEndpoint,
			Parameters:  map[string]string{"timeout_ms": "30000"},
		})
		suggestions = append(suggestions, Suggestion{
			Priority:    2,
			Title:       "Optimize Payload Size",
			Description: "Reduce payload size to decrease transfer and processing time.",
			Action:      ActionAddTransform,
		})
		
	case CategoryAuth:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Verify Authentication Credentials",
			Description: "Check that your API key, token, or credentials are valid and have not expired.",
			Action:      ActionUpdateHeaders,
		})
		suggestions = append(suggestions, Suggestion{
			Priority:    2,
			Title:       "Check Endpoint Permissions",
			Description: "Ensure your credentials have the required permissions for this endpoint.",
			Action:      ActionContactSupport,
		})
		
	case CategoryRateLimit:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Wait and Retry",
			Description: "The endpoint is rate-limiting requests. Delivery will be retried after the rate limit window.",
			Action:      ActionRetry,
			Parameters:  map[string]string{"delay_seconds": "60"},
		})
		suggestions = append(suggestions, Suggestion{
			Priority:    2,
			Title:       "Enable Request Batching",
			Description: "Consider batching multiple events into a single webhook to reduce request frequency.",
			Action:      ActionAddTransform,
		})
		
	case CategoryServerError:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Automatic Retry",
			Description: "Server errors are typically transient. The delivery will be automatically retried.",
			Action:      ActionRetry,
		})
		suggestions = append(suggestions, Suggestion{
			Priority:    2,
			Title:       "Contact Endpoint Owner",
			Description: "If errors persist, contact the endpoint owner to investigate server-side issues.",
			Action:      ActionContactSupport,
		})
		
	case CategoryClientError:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Review Request Format",
			Description: "The request was rejected by the server. Check the payload format and required fields.",
			Action:      ActionUpdatePayload,
		})
		if classification.SubCategory == "not_found" {
			suggestions = append(suggestions, Suggestion{
				Priority:    1,
				Title:       "Verify Endpoint URL",
				Description: "The endpoint URL may be incorrect or the resource no longer exists.",
				Action:      ActionUpdateEndpoint,
			})
		}
		
	case CategoryPayload:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Fix Payload Format",
			Description: "The payload is not in the expected format. Add a transformation to fix the structure.",
			Action:      ActionAddTransform,
		})
		if classification.SubCategory == "too_large" {
			suggestions = append(suggestions, Suggestion{
				Priority:    1,
				Title:       "Reduce Payload Size",
				Description: "The payload exceeds size limits. Use a transformation to trim or split the data.",
				Action:      ActionAddTransform,
				CodeSnippet: `return pick(payload, ['id', 'event', 'timestamp', 'data']);`,
			})
		}
		
	case CategoryCertificate:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Check SSL Certificate",
			Description: "The endpoint's SSL certificate is invalid, expired, or not trusted.",
			Action:      ActionCheckCertificate,
		})
		suggestions = append(suggestions, Suggestion{
			Priority:    2,
			Title:       "Contact Endpoint Owner",
			Description: "Notify the endpoint owner about the certificate issue.",
			Action:      ActionContactSupport,
		})
		
	case CategoryDNS:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Verify Domain Name",
			Description: "The domain cannot be resolved. Check if the URL is correct.",
			Action:      ActionCheckDNS,
		})
		suggestions = append(suggestions, Suggestion{
			Priority:    2,
			Title:       "Check DNS Configuration",
			Description: "The domain may not be properly configured. Contact the endpoint owner.",
			Action:      ActionContactSupport,
		})
		
	default:
		suggestions = append(suggestions, Suggestion{
			Priority:    1,
			Title:       "Review Error Details",
			Description: "Examine the error message and response for more information.",
			Action:      ActionContactSupport,
		})
	}
	
	return suggestions
}

// ExtractErrorPatterns extracts patterns from error messages for learning
func (c *Classifier) ExtractErrorPatterns(errors []string) map[string]int {
	patterns := make(map[string]int)
	
	// Normalize and count patterns
	for _, err := range errors {
		normalized := c.normalizeError(err)
		patterns[normalized]++
	}
	
	return patterns
}

func (c *Classifier) normalizeError(err string) string {
	// Remove UUIDs
	err = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).ReplaceAllString(err, "<UUID>")
	// Remove IP addresses
	err = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).ReplaceAllString(err, "<IP>")
	// Remove timestamps
	err = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`).ReplaceAllString(err, "<TIMESTAMP>")
	// Remove numbers
	err = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(err, "<N>")
	// Normalize whitespace
	err = regexp.MustCompile(`\s+`).ReplaceAllString(err, " ")
	
	return strings.TrimSpace(err)
}

// ClassifyFromHTTPResponse classifies an error from HTTP response details
func (c *Classifier) ClassifyFromHTTPResponse(statusCode int, body string, headers map[string]string, latency time.Duration) ErrorClassification {
	// Build combined context for pattern matching
	combined := body
	if statusCode > 0 {
		combined += fmt.Sprintf(" %d", statusCode)
	}

	// Check for latency-based timeout
	if latency > 30*time.Second {
		return ErrorClassification{
			Category:       CategoryTimeout,
			Confidence:     0.85,
			SubCategory:    "slow_response",
			IsRetryable:    true,
			SuggestedDelay: 60,
		}
	}

	// Check for retry-after header
	if retryAfter, ok := headers["Retry-After"]; ok && retryAfter != "" {
		return ErrorClassification{
			Category:       CategoryRateLimit,
			Confidence:     0.95,
			SubCategory:    "retry_after",
			IsRetryable:    true,
			SuggestedDelay: 60,
		}
	}

	// Check for redirect
	if statusCode >= 300 && statusCode < 400 {
		return ErrorClassification{
			Category:       CategoryClientError,
			Confidence:     0.90,
			SubCategory:    "redirect",
			IsRetryable:    false,
			SuggestedDelay: 0,
		}
	}

	// Run through pattern matching
	for _, p := range c.patterns {
		if p.regex.MatchString(combined) {
			return ErrorClassification{
				Category:       p.category,
				Confidence:     p.confidence,
				SubCategory:    p.subCategory,
				IsRetryable:    p.retryable,
				SuggestedDelay: p.delay,
			}
		}
	}

	return c.classifyByStatusCode(statusCode)
}

// LearnFromOutcome records a feedback entry to improve future classification accuracy
func (c *Classifier) LearnFromOutcome(classification *ErrorClassification, actualOutcome string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := string(classification.Category) + ":" + classification.SubCategory
	stats, exists := c.outcomeHistory[key]
	if !exists {
		stats = &outcomeStats{category: classification.Category}
		c.outcomeHistory[key] = stats
	}

	stats.total++
	if actualOutcome == string(classification.Category) {
		stats.correct++
	}
}

// GetRetryabilityScore calculates a retryability score combining error type, history, and context
func (c *Classifier) GetRetryabilityScore(classification ErrorClassification, attemptNumber int, totalFailures int) float64 {
	if !classification.IsRetryable {
		return 0.0
	}

	// Base score from classification confidence
	score := classification.Confidence

	// Decrease score based on attempt number (diminishing returns)
	if attemptNumber > 0 {
		score *= 1.0 / (1.0 + float64(attemptNumber)*0.3)
	}

	// Decrease score if endpoint has high historical failure rate
	if totalFailures > 100 {
		score *= 0.5
	} else if totalFailures > 50 {
		score *= 0.7
	} else if totalFailures > 20 {
		score *= 0.85
	}

	// Category-specific adjustments
	switch classification.Category {
	case CategoryRateLimit:
		score = min64(score*1.2, 1.0) // rate limits usually recover
	case CategoryServerError:
		score *= 0.9
	case CategoryTimeout:
		score *= 0.85
	case CategoryNetwork:
		if classification.SubCategory == "connection_refused" {
			score *= 0.7
		}
	}

	// Clamp to [0, 1]
	if score < 0 {
		return 0.0
	}
	if score > 1.0 {
		return 1.0
	}
	return score
}

// GetAccuracy returns the accuracy of classifications based on recorded outcomes
func (c *Classifier) GetAccuracy() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalCorrect := 0
	totalCount := 0
	for _, stats := range c.outcomeHistory {
		totalCorrect += stats.correct
		totalCount += stats.total
	}
	if totalCount == 0 {
		return 0.0
	}
	return float64(totalCorrect) / float64(totalCount)
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
