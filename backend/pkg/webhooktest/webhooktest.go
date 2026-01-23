// Package webhooktest provides a testing framework for webhook configurations
package webhooktest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// TestSuite represents a collection of webhook test cases
type TestSuite struct {
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	BaseURL     string       `json:"base_url" yaml:"base_url"`
	APIKey      string       `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Setup       *SetupConfig `json:"setup,omitempty" yaml:"setup,omitempty"`
	Tests       []TestCase   `json:"tests" yaml:"tests"`
}

// SetupConfig defines setup/teardown operations
type SetupConfig struct {
	Fixtures []Fixture `json:"fixtures,omitempty" yaml:"fixtures,omitempty"`
}

// Fixture represents test fixture data
type Fixture struct {
	Name    string          `json:"name" yaml:"name"`
	Payload json.RawMessage `json:"payload" yaml:"payload"`
}

// TestCase represents a single webhook test scenario
type TestCase struct {
	Name       string      `json:"name" yaml:"name"`
	Skip       bool        `json:"skip,omitempty" yaml:"skip,omitempty"`
	Request    TestRequest `json:"request" yaml:"request"`
	Assertions []Assertion `json:"assertions" yaml:"assertions"`
	Tags       []string    `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// TestRequest defines the webhook request to send
type TestRequest struct {
	EndpointID string            `json:"endpoint_id,omitempty" yaml:"endpoint_id,omitempty"`
	Payload    json.RawMessage   `json:"payload" yaml:"payload"`
	Headers    map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	EventType  string            `json:"event_type,omitempty" yaml:"event_type,omitempty"`
	UseFixture string            `json:"use_fixture,omitempty" yaml:"use_fixture,omitempty"`
}

// Assertion defines an expected condition
type Assertion struct {
	Type     AssertionType `json:"type" yaml:"type"`
	Field    string        `json:"field,omitempty" yaml:"field,omitempty"`
	Operator string        `json:"operator,omitempty" yaml:"operator,omitempty"`
	Expected interface{}   `json:"expected,omitempty" yaml:"expected,omitempty"`
	Message  string        `json:"message,omitempty" yaml:"message,omitempty"`
}

// AssertionType defines the type of assertion
type AssertionType string

const (
	AssertStatusCode   AssertionType = "status_code"
	AssertResponseBody AssertionType = "response_body"
	AssertLatency      AssertionType = "latency"
	AssertPayloadField AssertionType = "payload_field"
	AssertRegex        AssertionType = "regex"
)

// TestResult represents the outcome of a test case
type TestResult struct {
	Name       string         `json:"name"`
	Status     ResultStatus   `json:"status"`
	DurationMs float64        `json:"duration_ms"`
	Assertions []AssertResult `json:"assertions"`
	Error      string         `json:"error,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

// AssertResult represents the outcome of a single assertion
type AssertResult struct {
	Type     AssertionType `json:"type"`
	Passed   bool          `json:"passed"`
	Expected interface{}   `json:"expected"`
	Actual   interface{}   `json:"actual"`
	Message  string        `json:"message"`
}

// ResultStatus represents the result status
type ResultStatus string

const (
	StatusPassed  ResultStatus = "passed"
	StatusFailed  ResultStatus = "failed"
	StatusSkipped ResultStatus = "skipped"
	StatusError   ResultStatus = "error"
)

// SuiteResult represents the outcome of an entire test suite
type SuiteResult struct {
	Name       string       `json:"name"`
	TotalTests int          `json:"total_tests"`
	Passed     int          `json:"passed"`
	Failed     int          `json:"failed"`
	Skipped    int          `json:"skipped"`
	Errors     int          `json:"errors"`
	DurationMs float64      `json:"duration_ms"`
	Results    []TestResult `json:"results"`
	Timestamp  time.Time    `json:"timestamp"`
}

// Runner executes test suites
type Runner struct {
	client  *http.Client
	verbose bool
}

// NewRunner creates a new test runner
func NewRunner(verbose bool) *Runner {
	return &Runner{
		client:  &http.Client{Timeout: 30 * time.Second},
		verbose: verbose,
	}
}

// RunSuite executes all tests in a suite
func (r *Runner) RunSuite(suite *TestSuite) *SuiteResult {
	start := time.Now()
	result := &SuiteResult{Name: suite.Name, Timestamp: start}

	for _, tc := range suite.Tests {
		result.TotalTests++
		if tc.Skip {
			result.Skipped++
			result.Results = append(result.Results, TestResult{Name: tc.Name, Status: StatusSkipped, Timestamp: time.Now()})
			continue
		}

		tr := r.runTest(suite, tc)
		result.Results = append(result.Results, tr)
		switch tr.Status {
		case StatusPassed:
			result.Passed++
		case StatusFailed:
			result.Failed++
		case StatusError:
			result.Errors++
		}
	}

	result.DurationMs = float64(time.Since(start).Milliseconds())
	return result
}

func (r *Runner) runTest(suite *TestSuite, tc TestCase) TestResult {
	start := time.Now()
	tr := TestResult{Name: tc.Name, Timestamp: start}

	payload := tc.Request.Payload
	if tc.Request.UseFixture != "" && suite.Setup != nil {
		for _, f := range suite.Setup.Fixtures {
			if f.Name == tc.Request.UseFixture {
				payload = f.Payload
				break
			}
		}
	}

	url := suite.BaseURL + "/api/v1/webhooks/send"
	body := map[string]interface{}{"payload": json.RawMessage(payload)}
	if tc.Request.EndpointID != "" {
		body["endpoint_id"] = tc.Request.EndpointID
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		tr.Status = StatusError
		tr.Error = fmt.Sprintf("marshal error: %v", err)
		return tr
	}

	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	if suite.APIKey != "" {
		req.Header.Set("X-API-Key", suite.APIKey)
	}
	for k, v := range tc.Request.Headers {
		req.Header.Set(k, v)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		tr.Status = StatusError
		tr.Error = fmt.Sprintf("request failed: %v", err)
		tr.DurationMs = float64(time.Since(start).Milliseconds())
		return tr
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	tr.DurationMs = float64(time.Since(start).Milliseconds())

	allPassed := true
	for _, assertion := range tc.Assertions {
		ar := evaluateAssertion(assertion, resp.StatusCode, string(respBody), tr.DurationMs)
		tr.Assertions = append(tr.Assertions, ar)
		if !ar.Passed {
			allPassed = false
		}
	}

	if allPassed {
		tr.Status = StatusPassed
	} else {
		tr.Status = StatusFailed
	}
	return tr
}

func evaluateAssertion(a Assertion, statusCode int, body string, durationMs float64) AssertResult {
	ar := AssertResult{Type: a.Type, Expected: a.Expected, Message: a.Message}

	switch a.Type {
	case AssertStatusCode:
		expected, ok := toInt(a.Expected)
		if !ok {
			ar.Passed = false
			ar.Message = "invalid expected status code"
			return ar
		}
		ar.Actual = statusCode
		ar.Passed = statusCode == expected
		if ar.Message == "" {
			ar.Message = fmt.Sprintf("status code: expected %d, got %d", expected, statusCode)
		}

	case AssertResponseBody:
		ar.Actual = body
		switch a.Operator {
		case "contains":
			ar.Passed = strings.Contains(body, fmt.Sprintf("%v", a.Expected))
		case "not_empty":
			ar.Passed = len(body) > 0
		default:
			ar.Passed = len(body) > 0
		}

	case AssertLatency:
		maxMs, ok := toFloat(a.Expected)
		if !ok {
			ar.Passed = false
			ar.Message = "invalid latency threshold"
			return ar
		}
		ar.Actual = durationMs
		ar.Passed = durationMs <= maxMs
		if ar.Message == "" {
			ar.Message = fmt.Sprintf("latency: %.0fms (max: %.0fms)", durationMs, maxMs)
		}

	case AssertRegex:
		pattern := fmt.Sprintf("%v", a.Expected)
		re, err := regexp.Compile(pattern)
		if err != nil {
			ar.Passed = false
			ar.Message = "invalid regex: " + err.Error()
			return ar
		}
		ar.Actual = body
		ar.Passed = re.MatchString(body)

	case AssertPayloadField:
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(body), &parsed); err != nil {
			ar.Passed = false
			ar.Actual = "parse error"
			return ar
		}
		value := getNestedField(parsed, a.Field)
		ar.Actual = value
		ar.Passed = fmt.Sprintf("%v", value) == fmt.Sprintf("%v", a.Expected)
	}

	return ar
}

func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	default:
		return 0, false
	}
}

func getNestedField(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}

// FormatResults formats suite results as a human-readable string
func FormatResults(r *SuiteResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n=== Test Suite: %s ===\n", r.Name))
	b.WriteString(fmt.Sprintf("Duration: %.0fms\n\n", r.DurationMs))

	for _, tr := range r.Results {
		icon := "PASS"
		switch tr.Status {
		case StatusFailed:
			icon = "FAIL"
		case StatusSkipped:
			icon = "SKIP"
		case StatusError:
			icon = "ERR "
		}
		b.WriteString(fmt.Sprintf("[%s] %s (%.0fms)\n", icon, tr.Name, tr.DurationMs))
		if tr.Error != "" {
			b.WriteString(fmt.Sprintf("       Error: %s\n", tr.Error))
		}
	}

	b.WriteString(fmt.Sprintf("\nTotal: %d | Passed: %d | Failed: %d | Skipped: %d | Errors: %d\n",
		r.TotalTests, r.Passed, r.Failed, r.Skipped, r.Errors))
	return b.String()
}

// ToJUnit converts results to JUnit XML format for CI integration
func ToJUnit(r *SuiteResult) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(fmt.Sprintf(`<testsuite name="%s" tests="%d" failures="%d" errors="%d" skipped="%d" time="%.3f">`+"\n",
		r.Name, r.TotalTests, r.Failed, r.Errors, r.Skipped, r.DurationMs/1000))

	for _, tr := range r.Results {
		b.WriteString(fmt.Sprintf(`  <testcase name="%s" time="%.3f"`, tr.Name, tr.DurationMs/1000))
		switch tr.Status {
		case StatusSkipped:
			b.WriteString(">\n    <skipped/>\n  </testcase>\n")
		case StatusFailed:
			b.WriteString(">\n")
			for _, ar := range tr.Assertions {
				if !ar.Passed {
					b.WriteString(fmt.Sprintf("    <failure message=\"%s\"/>\n", ar.Message))
				}
			}
			b.WriteString("  </testcase>\n")
		case StatusError:
			b.WriteString(fmt.Sprintf(">\n    <error message=\"%s\"/>\n  </testcase>\n", tr.Error))
		default:
			b.WriteString("/>\n")
		}
	}

	b.WriteString("</testsuite>\n")
	return b.String()
}
