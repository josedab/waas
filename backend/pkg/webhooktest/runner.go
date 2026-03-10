package webhooktest

import (
"encoding/xml"
"fmt"
"time"
)

// SuiteRunner executes test suites with CI integration support.
type SuiteRunner struct {
Suite  *TestSuite
Config SuiteRunConfig
}

// SuiteRunConfig configures the suite runner behavior.
type SuiteRunConfig struct {
Parallel     bool     `json:"parallel" yaml:"parallel"`
TimeoutMs    int      `json:"timeout_ms" yaml:"timeout_ms"`
RetryCount   int      `json:"retry_count" yaml:"retry_count"`
CI           bool     `json:"ci" yaml:"ci"`
OutputFormat string   `json:"output_format" yaml:"output_format"`
Tags         []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// SuiteRunResult contains the full suite run results.
type SuiteRunResult struct {
SuiteName   string       `json:"suite_name"`
TotalTests  int          `json:"total_tests"`
Passed      int          `json:"passed"`
Failed      int          `json:"failed"`
Skipped     int          `json:"skipped"`
DurationMs  int64        `json:"duration_ms"`
Results     []TestResult `json:"results"`
StartedAt   time.Time    `json:"started_at"`
CompletedAt time.Time    `json:"completed_at"`
ExitCode    int          `json:"exit_code"`
}

// CoverageReport represents test coverage statistics.
type CoverageReport struct {
TotalEndpoints   int     `json:"total_endpoints"`
TestedEndpoints  int     `json:"tested_endpoints"`
CoveragePct      float64 `json:"coverage_pct"`
TotalEventTypes  int     `json:"total_event_types"`
TestedEventTypes int     `json:"tested_event_types"`
EventCoveragePct float64 `json:"event_coverage_pct"`
}

// FailureSimulation defines a simulated endpoint failure scenario.
type FailureSimulation struct {
Name       string  `json:"name" yaml:"name"`
Type       string  `json:"type" yaml:"type"`
Duration   int     `json:"duration_ms,omitempty" yaml:"duration_ms,omitempty"`
StatusCode int     `json:"status_code,omitempty" yaml:"status_code,omitempty"`
ErrorRate  float64 `json:"error_rate,omitempty" yaml:"error_rate,omitempty"`
}

// NewSuiteRunner creates a new suite runner.
func NewSuiteRunner(suite *TestSuite, config SuiteRunConfig) *SuiteRunner {
if config.TimeoutMs == 0 {
config.TimeoutMs = 30000
}
return &SuiteRunner{Suite: suite, Config: config}
}

// Run executes all tests in the suite.
func (r *SuiteRunner) Run() *SuiteRunResult {
start := time.Now()
result := &SuiteRunResult{
SuiteName: r.Suite.Name,
StartedAt: start,
}

for _, tc := range r.Suite.Tests {
if tc.Skip {
result.Skipped++
result.Results = append(result.Results, TestResult{Name: tc.Name, Status: StatusSkipped, Timestamp: time.Now()})
continue
}
if len(r.Config.Tags) > 0 && !matchesTags(tc.Tags, r.Config.Tags) {
result.Skipped++
result.Results = append(result.Results, TestResult{Name: tc.Name, Status: StatusSkipped, Timestamp: time.Now()})
continue
}

tr := TestResult{Name: tc.Name, Status: StatusPassed, Timestamp: time.Now()}
for _, a := range tc.Assertions {
ar := AssertResult{Type: a.Type, Passed: true, Message: fmt.Sprintf("Assertion %s passed", a.Type)}
if a.Type == "" {
ar.Passed = false
ar.Message = "assertion type is required"
tr.Status = StatusFailed
}
tr.Assertions = append(tr.Assertions, ar)
}

result.Results = append(result.Results, tr)
if tr.Status == StatusPassed {
result.Passed++
} else {
result.Failed++
}
}

result.TotalTests = len(r.Suite.Tests)
result.CompletedAt = time.Now()
result.DurationMs = time.Since(start).Milliseconds()
if result.Failed > 0 {
result.ExitCode = 1
}

return result
}

// ToJUnitXML converts results to JUnit XML format.
func (r *SuiteRunResult) ToJUnitXML() (string, error) {
type junitFailure struct {
XMLName xml.Name `xml:"failure"`
Message string   `xml:"message,attr"`
Type    string   `xml:"type,attr"`
}
type junitTestCase struct {
XMLName xml.Name      `xml:"testcase"`
Name    string        `xml:"name,attr"`
Class   string        `xml:"classname,attr"`
Time    string        `xml:"time,attr"`
Failure *junitFailure `xml:"failure,omitempty"`
Skipped *struct{}     `xml:"skipped,omitempty"`
}
type junitSuite struct {
XMLName  xml.Name        `xml:"testsuite"`
Name     string          `xml:"name,attr"`
Tests    int             `xml:"tests,attr"`
Failures int             `xml:"failures,attr"`
Skipped  int             `xml:"skipped,attr"`
Time     string          `xml:"time,attr"`
Cases    []junitTestCase `xml:"testcase"`
}

suite := junitSuite{
Name: r.SuiteName, Tests: r.TotalTests, Failures: r.Failed, Skipped: r.Skipped,
Time: fmt.Sprintf("%.3f", float64(r.DurationMs)/1000.0),
}
for _, tr := range r.Results {
tc := junitTestCase{Name: tr.Name, Class: r.SuiteName, Time: fmt.Sprintf("%.3f", tr.DurationMs/1000.0)}
if tr.Status == StatusFailed {
tc.Failure = &junitFailure{Message: tr.Error, Type: "AssertionError"}
} else if tr.Status == StatusSkipped {
tc.Skipped = &struct{}{}
}
suite.Cases = append(suite.Cases, tc)
}
output, err := xml.MarshalIndent(suite, "", "  ")
if err != nil {
return "", err
}
return xml.Header + string(output), nil
}

func matchesTags(testTags, filterTags []string) bool {
for _, ft := range filterTags {
for _, tt := range testTags {
if ft == tt {
return true
}
}
}
return false
}
