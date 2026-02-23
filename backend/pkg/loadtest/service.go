package loadtest

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// ServiceConfig configures the load test service.
type ServiceConfig struct {
	MaxRPS             int
	MaxDuration        time.Duration
	MaxConcurrentTests int
	DefaultPayload     string
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxRPS:             10000,
		MaxDuration:        1 * time.Hour,
		MaxConcurrentTests: 5,
		DefaultPayload:     `{"type":"loadtest.ping","data":{"timestamp":"{{.Timestamp}}"}}`,
	}
}

// Service implements the load test business logic.
type Service struct {
	repo       Repository
	config     *ServiceConfig
	client     *http.Client
	activeRuns sync.Map
}

// NewService creates a new load test service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{
		repo:   repo,
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateTestRun creates and starts a load test.
func (s *Service) CreateTestRun(tenantID string, cfg *TestConfig) (*TestRun, error) {
	if err := s.validateConfig(cfg); err != nil {
		return nil, err
	}

	if cfg.Pattern == "" {
		cfg.Pattern = PatternConstant
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}
	if cfg.PayloadTemplate == "" {
		cfg.PayloadTemplate = s.config.DefaultPayload
	}

	cfg.ID = uuid.New().String()
	cfg.TenantID = tenantID

	now := time.Now()
	run := &TestRun{
		ID:        cfg.ID,
		TenantID:  tenantID,
		Config:    cfg,
		Status:    "running",
		StartedAt: &now,
	}

	if err := s.repo.CreateTestRun(run); err != nil {
		return nil, fmt.Errorf("failed to create test run: %w", err)
	}

	// Run test in background
	go s.executeTest(run)

	return run, nil
}

// GetTestRun retrieves a test run by ID.
func (s *Service) GetTestRun(id string) (*TestRun, error) {
	return s.repo.GetTestRun(id)
}

// ListTestRuns returns all test runs for a tenant.
func (s *Service) ListTestRuns(tenantID string) ([]*TestRun, error) {
	return s.repo.ListTestRuns(tenantID)
}

// CancelTestRun cancels a running test.
func (s *Service) CancelTestRun(id string) error {
	run, err := s.repo.GetTestRun(id)
	if err != nil {
		return err
	}
	if run.Status != "running" {
		return fmt.Errorf("test is not running")
	}
	// Signal cancellation
	if cancel, ok := s.activeRuns.Load(id); ok {
		cancel.(func())()
	}
	run.Status = "cancelled"
	now := time.Now()
	run.CompletedAt = &now
	return s.repo.UpdateTestRun(run)
}

// GetScenarios returns pre-built test scenarios.
func (s *Service) GetScenarios() []Scenario {
	return DefaultScenarios()
}

func (s *Service) validateConfig(cfg *TestConfig) error {
	if cfg.EndpointURL == "" {
		return fmt.Errorf("endpoint_url is required")
	}
	if cfg.RPS <= 0 {
		return fmt.Errorf("rps must be positive")
	}
	if cfg.RPS > s.config.MaxRPS {
		return fmt.Errorf("rps exceeds maximum of %d", s.config.MaxRPS)
	}
	if cfg.Duration == "" {
		return fmt.Errorf("duration is required")
	}
	duration, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	if duration > s.config.MaxDuration {
		return fmt.Errorf("duration exceeds maximum of %s", s.config.MaxDuration)
	}
	return nil
}

func (s *Service) executeTest(run *TestRun) {
	duration, _ := time.ParseDuration(run.Config.Duration)
	cancelled := make(chan struct{})

	s.activeRuns.Store(run.ID, func() { close(cancelled) })
	defer s.activeRuns.Delete(run.ID)

	var totalReqs, successReqs, failureReqs int64
	var latencies []float64
	var latMu sync.Mutex
	statusCodes := make(map[int]int64)
	var scMu sync.Mutex
	var timeSeries []TimePoint

	startTime := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	endTime := startTime.Add(duration)
	sem := make(chan struct{}, run.Config.Concurrency)

	var wg sync.WaitGroup

	for time.Now().Before(endTime) {
		select {
		case <-cancelled:
			goto done
		case <-ticker.C:
			elapsed := time.Since(startTime).Seconds()
			rps := s.computeRPS(run.Config, elapsed, duration.Seconds())

			intervalStart := time.Now()
			var iSuccess, iFailure int64

			for i := 0; i < rps; i++ {
				sem <- struct{}{}
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-sem }()

					start := time.Now()
					statusCode, err := s.sendRequest(run.Config)
					lat := float64(time.Since(start).Milliseconds())

					atomic.AddInt64(&totalReqs, 1)
					latMu.Lock()
					latencies = append(latencies, lat)
					latMu.Unlock()

					scMu.Lock()
					statusCodes[statusCode]++
					scMu.Unlock()

					if err != nil || statusCode >= 400 {
						atomic.AddInt64(&failureReqs, 1)
						atomic.AddInt64(&iFailure, 1)
					} else {
						atomic.AddInt64(&successReqs, 1)
						atomic.AddInt64(&iSuccess, 1)
					}
				}()
			}

			timeSeries = append(timeSeries, TimePoint{
				Timestamp:    intervalStart,
				RPS:          float64(rps),
				SuccessCount: atomic.LoadInt64(&iSuccess),
				FailureCount: atomic.LoadInt64(&iFailure),
			})
		}
	}

done:
	wg.Wait()

	now := time.Now()
	run.CompletedAt = &now
	run.Status = "completed"

	// Compute report
	sort.Float64s(latencies)
	report := &TestReport{
		TotalRequests: atomic.LoadInt64(&totalReqs),
		SuccessCount:  atomic.LoadInt64(&successReqs),
		FailureCount:  atomic.LoadInt64(&failureReqs),
		Duration:      time.Since(startTime).String(),
		StatusCodes:   statusCodes,
		TimeSeries:    timeSeries,
		GeneratedAt:   time.Now(),
	}

	if report.TotalRequests > 0 {
		report.ErrorRate = float64(report.FailureCount) / float64(report.TotalRequests) * 100
		report.Throughput = float64(report.TotalRequests) / time.Since(startTime).Seconds()
	}

	if len(latencies) > 0 {
		report.LatencyMin = latencies[0]
		report.LatencyMax = latencies[len(latencies)-1]
		report.LatencyP50 = percentile(latencies, 50)
		report.LatencyP95 = percentile(latencies, 95)
		report.LatencyP99 = percentile(latencies, 99)

		var sum float64
		for _, l := range latencies {
			sum += l
		}
		report.LatencyAvg = sum / float64(len(latencies))
	}

	report.Recommendations = s.generateRecommendations(report)
	run.Report = report
	_ = s.repo.UpdateTestRun(run)
}

func (s *Service) computeRPS(cfg *TestConfig, elapsed, total float64) int {
	switch cfg.Pattern {
	case PatternRampUp:
		start := float64(cfg.RampUpStart)
		if start <= 0 {
			start = 1
		}
		return int(start + (float64(cfg.RPS)-start)*(elapsed/total))
	case PatternBurst:
		multiplier := cfg.BurstMultiplier
		if multiplier <= 0 {
			multiplier = 3.0
		}
		burstDur, _ := time.ParseDuration(cfg.BurstDuration)
		if burstDur == 0 {
			burstDur = 30 * time.Second
		}
		// Burst at midpoint
		mid := total / 2
		if elapsed >= mid && elapsed <= mid+burstDur.Seconds() {
			return int(float64(cfg.RPS) * multiplier)
		}
		return cfg.RPS
	case PatternSineWave:
		amplitude := float64(cfg.RPS) * 0.5
		base := float64(cfg.RPS) * 0.5
		return int(base + amplitude*math.Sin(2*math.Pi*elapsed/total))
	default:
		return cfg.RPS
	}
}

func (s *Service) sendRequest(cfg *TestConfig) (int, error) {
	payload := strings.ReplaceAll(cfg.PayloadTemplate, "{{.Timestamp}}", time.Now().Format(time.RFC3339))

	req, err := http.NewRequest("POST", cfg.EndpointURL, strings.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-WaaS-LoadTest", "true")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func (s *Service) generateRecommendations(report *TestReport) []string {
	var recs []string
	if report.ErrorRate > 10 {
		recs = append(recs, "High error rate detected — consider increasing timeout or retry limits")
	}
	if report.LatencyP99 > 5000 {
		recs = append(recs, "P99 latency exceeds 5s — investigate endpoint performance bottlenecks")
	}
	if report.LatencyP95 > 1000 {
		recs = append(recs, "P95 latency exceeds 1s — consider adding caching or async processing")
	}
	if report.Throughput > 0 && report.Throughput < float64(report.TotalRequests)/10 {
		recs = append(recs, "Throughput significantly below target — consider horizontal scaling")
	}
	return recs
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
