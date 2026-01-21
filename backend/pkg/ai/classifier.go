package ai

import (
	"regexp"
	"strings"
)

// Classifier provides error classification capabilities
type Classifier struct {
	patterns []classificationPattern
}

type classificationPattern struct {
	regex      *regexp.Regexp
	category   ErrorCategory
	subCategory string
	confidence float64
	retryable  bool
	delay      int
}

// NewClassifier creates a new error classifier with predefined patterns
func NewClassifier() *Classifier {
	c := &Classifier{}
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
		{regexp.MustCompile(`(?i)x509|tls|ssl.*error`), CategoryCertificate, "tls_error", 0.85, false, 0},
		{regexp.MustCompile(`(?i)certificate signed by unknown authority`), CategoryCertificate, "unknown_ca", 0.95, false, 0},
		
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
		
		// Client errors
		{regexp.MustCompile(`(?i)400|bad request`), CategoryClientError, "bad_request", 0.85, false, 0},
		{regexp.MustCompile(`(?i)404|not found`), CategoryClientError, "not_found", 0.90, false, 0},
		{regexp.MustCompile(`(?i)405|method not allowed`), CategoryClientError, "method_not_allowed", 0.95, false, 0},
		{regexp.MustCompile(`(?i)406|not acceptable`), CategoryClientError, "not_acceptable", 0.90, false, 0},
		{regexp.MustCompile(`(?i)415|unsupported media type`), CategoryClientError, "unsupported_media", 0.95, false, 0},
		{regexp.MustCompile(`(?i)422|unprocessable`), CategoryClientError, "unprocessable", 0.90, false, 0},
		
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
