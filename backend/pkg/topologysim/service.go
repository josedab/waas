package topologysim

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ServiceConfig configures the topology simulator.
type ServiceConfig struct {
	MaxEndpoints       int
	MaxTrafficSources  int
	MaxSimDuration     time.Duration
	MaxMonteCarloRuns  int
	DefaultCostPerHour float64
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxEndpoints:       50,
		MaxTrafficSources:  20,
		MaxSimDuration:     1 * time.Hour,
		MaxMonteCarloRuns:  100,
		DefaultCostPerHour: 0.50,
	}
}

// Service implements the topology simulation engine.
type Service struct {
	repo   Repository
	config *ServiceConfig
}

// NewService creates a new topology simulation service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{repo: repo, config: config}
}

// CreateTopology creates a new topology definition.
func (s *Service) CreateTopology(tenantID string, req *CreateTopologyRequest) (*Topology, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one endpoint is required")
	}
	if len(req.Endpoints) > s.config.MaxEndpoints {
		return nil, fmt.Errorf("maximum %d endpoints", s.config.MaxEndpoints)
	}
	if len(req.Traffic) == 0 {
		return nil, fmt.Errorf("at least one traffic source is required")
	}

	topology := &Topology{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Endpoints:   req.Endpoints,
		Traffic:     req.Traffic,
		Constraints: req.Constraints,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateTopology(topology); err != nil {
		return nil, fmt.Errorf("failed to create topology: %w", err)
	}
	return topology, nil
}

// GetTopology retrieves a topology by ID.
func (s *Service) GetTopology(id string) (*Topology, error) {
	return s.repo.GetTopology(id)
}

// ListTopologies returns all topologies for a tenant.
func (s *Service) ListTopologies(tenantID string) ([]*Topology, error) {
	return s.repo.ListTopologies(tenantID)
}

// RunSimulation executes a discrete-event simulation.
func (s *Service) RunSimulation(cfg *SimulationConfig) (*SimulationResult, error) {
	topology, err := s.repo.GetTopology(cfg.TopologyID)
	if err != nil {
		return nil, err
	}

	duration, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}
	if duration > s.config.MaxSimDuration {
		return nil, fmt.Errorf("duration exceeds maximum of %s", s.config.MaxSimDuration)
	}

	if cfg.MonteCarloRuns > 0 {
		return s.runMonteCarlo(topology, cfg, duration)
	}

	result := s.simulateOnce(topology, duration, cfg.Seed)
	if err := s.repo.StoreResult(result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetResult retrieves a simulation result.
func (s *Service) GetResult(id string) (*SimulationResult, error) {
	return s.repo.GetResult(id)
}

// ListResults returns all results for a topology.
func (s *Service) ListResults(topologyID string) ([]*SimulationResult, error) {
	return s.repo.ListResults(topologyID)
}

func (s *Service) simulateOnce(topology *Topology, duration time.Duration, seed int64) *SimulationResult {
	rng := rand.New(rand.NewPCG(uint64(seed), 0))
	if seed == 0 {
		rng = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
	}

	durationMs := duration.Seconds() * 1000
	epMetrics := make(map[string]*EndpointMetric)
	for _, ep := range topology.Endpoints {
		epMetrics[ep.ID] = &EndpointMetric{EndpointID: ep.ID}
	}

	var totalEvents, delivered, failed, retried int64
	var latencies []float64
	var timeline []TimelineEvent
	maxQueueDepth := 0
	queueDepthSum := 0.0
	queueSamples := 0

	// Build endpoint lookup
	epMap := make(map[string]SimEndpoint)
	for _, ep := range topology.Endpoints {
		epMap[ep.ID] = ep
	}

	// Simulate traffic
	for _, traffic := range topology.Traffic {
		eventDuration, _ := time.ParseDuration(traffic.Duration)
		if eventDuration == 0 {
			eventDuration = duration
		}
		eventsTotal := int(traffic.RPS * eventDuration.Seconds())
		intervalMs := durationMs / float64(eventsTotal)
		if eventsTotal <= 0 {
			continue
		}

		currentQueue := 0

		for i := 0; i < eventsTotal; i++ {
			eventTime := intervalMs * float64(i)
			totalEvents++

			for _, targetID := range traffic.TargetIDs {
				ep, ok := epMap[targetID]
				if !ok {
					continue
				}

				// Simulate latency (normal distribution)
				latency := ep.LatencyMean + ep.LatencyStdDev*rng.NormFloat64()
				if latency < 1 {
					latency = 1
				}
				latencies = append(latencies, latency)

				// Simulate failure
				if rng.Float64() < ep.FailureRate {
					failed++
					if m, ok := epMetrics[targetID]; ok {
						m.TotalFailed++
					}

					// Simulate retries
					if ep.RetryPolicy != nil {
						for r := 0; r < ep.RetryPolicy.MaxRetries; r++ {
							retried++
							if m, ok := epMetrics[targetID]; ok {
								m.TotalRetried++
							}
							if rng.Float64() >= ep.FailureRate {
								delivered++
								if m, ok := epMetrics[targetID]; ok {
									m.TotalDelivered++
								}
								break
							}
						}
					}

					timeline = append(timeline, TimelineEvent{
						Time:      eventTime,
						EventType: "failure",
						Endpoint:  targetID,
					})
				} else {
					delivered++
					if m, ok := epMetrics[targetID]; ok {
						m.TotalDelivered++
					}
				}

				// Track queue depth
				currentQueue++
				if ep.MaxConcurrency > 0 && currentQueue > ep.MaxConcurrency {
					if currentQueue > maxQueueDepth {
						maxQueueDepth = currentQueue
					}
				}
				queueDepthSum += float64(currentQueue)
				queueSamples++
				currentQueue--
			}
		}
	}

	// Compute latency percentiles
	sort.Float64s(latencies)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)
	avgLat := 0.0
	if len(latencies) > 0 {
		sum := 0.0
		for _, l := range latencies {
			sum += l
		}
		avgLat = sum / float64(len(latencies))
	}

	avgQueue := 0.0
	if queueSamples > 0 {
		avgQueue = queueDepthSum / float64(queueSamples)
	}

	// Detect bottlenecks
	var bottlenecks []Bottleneck
	if topology.Constraints != nil {
		if maxQueueDepth > topology.Constraints.MaxQueueDepth {
			bottlenecks = append(bottlenecks, Bottleneck{
				Resource:    "queue",
				Utilization: float64(maxQueueDepth) / float64(topology.Constraints.MaxQueueDepth),
				Threshold:   0.9,
				Impact:      "Queue overflow - events will be dropped or delayed",
			})
		}
	}

	// Detect retry storms
	retryStorms := 0
	if retried > totalEvents/2 {
		retryStorms = int(retried - totalEvents/2)
	}

	// Compute per-endpoint avg latency
	for _, ep := range topology.Endpoints {
		if m, ok := epMetrics[ep.ID]; ok {
			m.AvgLatencyMs = ep.LatencyMean
		}
	}

	var epMetricsList []EndpointMetric
	for _, m := range epMetrics {
		epMetricsList = append(epMetricsList, *m)
	}

	// Cost estimate
	var costEstimate *CostEstimate
	hours := duration.Hours()
	if hours > 0 {
		costEstimate = &CostEstimate{
			ComputeCostPerHour:   s.config.DefaultCostPerHour,
			EstimatedMonthlyCost: s.config.DefaultCostPerHour * 730, // avg hours/month
			CostPerMillionEvents: s.config.DefaultCostPerHour * 730 / (float64(totalEvents) / 1e6),
		}
	}

	result := &SimulationResult{
		ID:               uuid.New().String(),
		TopologyID:       topology.ID,
		Status:           "completed",
		Duration:         duration.String(),
		TotalEvents:      totalEvents,
		DeliveredEvents:  delivered,
		FailedEvents:     failed,
		RetriedEvents:    retried,
		AvgLatencyMs:     avgLat,
		P95LatencyMs:     p95,
		P99LatencyMs:     p99,
		MaxQueueDepth:    maxQueueDepth,
		AvgQueueDepth:    avgQueue,
		RetryStormEvents: retryStorms,
		Bottlenecks:      bottlenecks,
		EndpointMetrics:  epMetricsList,
		Timeline:         timeline,
		CostEstimate:     costEstimate,
		Recommendations:  s.generateRecommendations(failed, totalEvents, p99, retryStorms),
		CompletedAt:      time.Now(),
	}

	return result
}

func (s *Service) runMonteCarlo(topology *Topology, cfg *SimulationConfig, duration time.Duration) (*SimulationResult, error) {
	runs := cfg.MonteCarloRuns
	if runs > s.config.MaxMonteCarloRuns {
		runs = s.config.MaxMonteCarloRuns
	}

	var throughputs, failureRates, p99Latencies []float64
	retryStormCount := 0

	for i := 0; i < runs; i++ {
		result := s.simulateOnce(topology, duration, cfg.Seed+int64(i))
		if result.TotalEvents > 0 {
			throughputs = append(throughputs, float64(result.DeliveredEvents)/duration.Seconds())
			failureRates = append(failureRates, float64(result.FailedEvents)/float64(result.TotalEvents)*100)
			p99Latencies = append(p99Latencies, result.P99LatencyMs)
			if result.RetryStormEvents > 0 {
				retryStormCount++
			}
		}
	}

	sort.Float64s(throughputs)
	sort.Float64s(failureRates)
	sort.Float64s(p99Latencies)

	// Aggregate into a single result with Monte Carlo stats
	aggregated := s.simulateOnce(topology, duration, cfg.Seed)
	aggregated.Recommendations = append(aggregated.Recommendations,
		fmt.Sprintf("Monte Carlo analysis (%d runs): %.1f%% chance of retry storm, P5 throughput: %.0f rps",
			runs, float64(retryStormCount)/float64(runs)*100, percentile(throughputs, 5)))

	if err := s.repo.StoreResult(aggregated); err != nil {
		return nil, err
	}
	return aggregated, nil
}

func (s *Service) generateRecommendations(failed, total int64, p99 float64, retryStorms int) []string {
	var recs []string
	if total > 0 {
		failRate := float64(failed) / float64(total) * 100
		if failRate > 10 {
			recs = append(recs, "High failure rate — consider adding circuit breakers")
		}
	}
	if p99 > 5000 {
		recs = append(recs, "P99 latency exceeds 5s — review endpoint performance")
	}
	if retryStorms > 0 {
		recs = append(recs, "Retry storms detected — implement exponential backoff with jitter")
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
