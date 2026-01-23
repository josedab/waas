package webhooktest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunSuiteSkipped(t *testing.T) {
	runner := NewRunner(false)
	suite := &TestSuite{
		Name: "skip-test",
		Tests: []TestCase{
			{Name: "test-1", Skip: true},
		},
	}

	result := runner.RunSuite(suite)
	if result.TotalTests != 1 {
		t.Errorf("expected 1 total test, got %d", result.TotalTests)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
}

func TestRunSuiteWithServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "del-123",
			"status": "pending",
		})
	}))
	defer server.Close()

	runner := NewRunner(true)
	suite := &TestSuite{
		Name:    "integration-test",
		BaseURL: server.URL,
		Tests: []TestCase{
			{
				Name: "send webhook",
				Request: TestRequest{
					Payload: json.RawMessage(`{"event":"test"}`),
				},
				Assertions: []Assertion{
					{Type: AssertStatusCode, Expected: 200},
					{Type: AssertResponseBody, Operator: "not_empty"},
					{Type: AssertPayloadField, Field: "status", Expected: "pending"},
				},
			},
		},
	}

	result := runner.RunSuite(suite)
	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d passed / %d failed", result.Passed, result.Failed)
		for _, r := range result.Results {
			for _, a := range r.Assertions {
				t.Logf("  assertion %s: passed=%v expected=%v actual=%v", a.Type, a.Passed, a.Expected, a.Actual)
			}
		}
	}
}

func TestRunSuiteWithFixtures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	runner := NewRunner(false)
	suite := &TestSuite{
		Name:    "fixture-test",
		BaseURL: server.URL,
		Setup: &SetupConfig{
			Fixtures: []Fixture{
				{Name: "order", Payload: json.RawMessage(`{"type":"order.created","amount":100}`)},
			},
		},
		Tests: []TestCase{
			{
				Name: "use fixture",
				Request: TestRequest{
					UseFixture: "order",
				},
				Assertions: []Assertion{
					{Type: AssertStatusCode, Expected: 200},
				},
			},
		},
	}

	result := runner.RunSuite(suite)
	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Passed)
	}
}

func TestAssertions(t *testing.T) {
	tests := []struct {
		name     string
		assert   Assertion
		status   int
		body     string
		duration float64
		passed   bool
	}{
		{"status ok", Assertion{Type: AssertStatusCode, Expected: 200}, 200, "", 0, true},
		{"status fail", Assertion{Type: AssertStatusCode, Expected: 200}, 404, "", 0, false},
		{"body contains", Assertion{Type: AssertResponseBody, Operator: "contains", Expected: "hello"}, 200, "hello world", 0, true},
		{"body not contains", Assertion{Type: AssertResponseBody, Operator: "contains", Expected: "missing"}, 200, "hello", 0, false},
		{"latency ok", Assertion{Type: AssertLatency, Expected: 1000.0}, 200, "", 500, true},
		{"latency fail", Assertion{Type: AssertLatency, Expected: 100.0}, 200, "", 500, false},
		{"regex match", Assertion{Type: AssertRegex, Expected: `"id":\s*"[a-z]+-\d+"`}, 200, `{"id":"del-123"}`, 0, true},
		{"regex no match", Assertion{Type: AssertRegex, Expected: `^error`}, 200, `{"ok":true}`, 0, false},
		{"payload field", Assertion{Type: AssertPayloadField, Field: "name", Expected: "test"}, 200, `{"name":"test"}`, 0, true},
		{"nested field", Assertion{Type: AssertPayloadField, Field: "data.id", Expected: "abc"}, 200, `{"data":{"id":"abc"}}`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ar := evaluateAssertion(tt.assert, tt.status, tt.body, tt.duration)
			if ar.Passed != tt.passed {
				t.Errorf("expected passed=%v, got passed=%v (actual=%v)", tt.passed, ar.Passed, ar.Actual)
			}
		})
	}
}

func TestFormatResults(t *testing.T) {
	result := &SuiteResult{
		Name:       "test-suite",
		TotalTests: 3,
		Passed:     2,
		Failed:     1,
		Results: []TestResult{
			{Name: "test1", Status: StatusPassed, DurationMs: 100},
			{Name: "test2", Status: StatusFailed, DurationMs: 200},
			{Name: "test3", Status: StatusSkipped},
		},
	}

	output := FormatResults(result)
	if output == "" {
		t.Error("expected non-empty formatted output")
	}
	if !containsStr(output, "test-suite") {
		t.Error("expected suite name in output")
	}
	if !containsStr(output, "PASS") {
		t.Error("expected PASS in output")
	}
}

func TestToJUnit(t *testing.T) {
	result := &SuiteResult{
		Name:       "junit-test",
		TotalTests: 2,
		Passed:     1,
		Failed:     1,
		Results: []TestResult{
			{Name: "pass-test", Status: StatusPassed, DurationMs: 50},
			{Name: "fail-test", Status: StatusFailed, DurationMs: 100, Assertions: []AssertResult{
				{Passed: false, Message: "status mismatch"},
			}},
		},
	}

	xml := ToJUnit(result)
	if !containsStr(xml, "<?xml") {
		t.Error("expected XML header")
	}
	if !containsStr(xml, "junit-test") {
		t.Error("expected suite name")
	}
	if !containsStr(xml, "failure") {
		t.Error("expected failure element")
	}
}

func TestGetNestedField(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "deep",
			},
		},
	}

	if v := getNestedField(data, "a.b.c"); v != "deep" {
		t.Errorf("expected 'deep', got '%v'", v)
	}
	if v := getNestedField(data, "nonexistent"); v != nil {
		t.Errorf("expected nil, got '%v'", v)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
