package livemigration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Cutover phase constants
const (
	CutoverPhaseValidate    = "validate"
	CutoverPhaseDualWrite   = "dual_write"
	CutoverPhaseGradualShift = "gradual_shift"
	CutoverPhaseComplete    = "complete"
	CutoverPhaseRollback    = "rollback"
)

// Cutover status constants
const (
	CutoverStatusPending    = "pending"
	CutoverStatusInProgress = "in_progress"
	CutoverStatusCompleted  = "completed"
	CutoverStatusFailed     = "failed"
	CutoverStatusRolledBack = "rolled_back"
)

// CutoverPlanDetailed describes a phased cutover plan with validation, dual-write,
// gradual traffic shifting, and rollback support
type CutoverPlanDetailed struct {
	ID               string          `json:"id"`
	JobID            string          `json:"job_id"`
	TenantID         string          `json:"tenant_id"`
	Status           string          `json:"status"`
	CurrentPhase     string          `json:"current_phase"`
	Phases           []CutoverPhase  `json:"phases"`
	TrafficSplitPct  int             `json:"traffic_split_pct"`
	DryRun           bool            `json:"dry_run"`
	ValidationReport *ValidationReport `json:"validation_report,omitempty"`
	RollbackSnapshot *RollbackSnapshot `json:"rollback_snapshot,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// CutoverPhase represents a single phase in the cutover plan
type CutoverPhase struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// ValidationReport captures the result of pre-cutover validation
type ValidationReport struct {
	TotalEndpoints    int      `json:"total_endpoints"`
	ValidEndpoints    int      `json:"valid_endpoints"`
	InvalidEndpoints  int      `json:"invalid_endpoints"`
	Warnings          []string `json:"warnings,omitempty"`
	Errors            []string `json:"errors,omitempty"`
	CanProceed        bool     `json:"can_proceed"`
	DryRunSummary     string   `json:"dry_run_summary,omitempty"`
}

// RollbackSnapshot captures the state before cutover for safe rollback
type RollbackSnapshot struct {
	ID                string               `json:"id"`
	JobID             string               `json:"job_id"`
	TenantID          string               `json:"tenant_id"`
	EndpointStates    []EndpointSnapshot   `json:"endpoint_states"`
	TrafficSplitPct   int                  `json:"traffic_split_pct"`
	PreviousJobStatus string               `json:"previous_job_status"`
	CapturedAt        time.Time            `json:"captured_at"`
}

// EndpointSnapshot captures an endpoint's state at snapshot time
type EndpointSnapshot struct {
	EndpointID    string `json:"endpoint_id"`
	SourceID      string `json:"source_id"`
	SourceURL     string `json:"source_url"`
	DestinationID string `json:"destination_id"`
	Status        string `json:"status"`
}

// --- DualWriteManager ---

// DualWriteManager writes webhook deliveries to both old and new systems
// and compares results for verification
type DualWriteManager struct {
	jobID    string
	tenantID string
	enabled  bool
	results  []DualWriteResult
	mu       sync.RWMutex
}

// DualWriteResult captures the result of a single dual-write operation
type DualWriteResult struct {
	ID              string    `json:"id"`
	EndpointID      string    `json:"endpoint_id"`
	EventID         string    `json:"event_id"`
	OldSystemStatus int       `json:"old_system_status"`
	NewSystemStatus int       `json:"new_system_status"`
	OldLatencyMs    int64     `json:"old_latency_ms"`
	NewLatencyMs    int64     `json:"new_latency_ms"`
	Match           bool      `json:"match"`
	Timestamp       time.Time `json:"timestamp"`
}

// NewDualWriteManager creates a new dual-write manager
func NewDualWriteManager(jobID, tenantID string) *DualWriteManager {
	return &DualWriteManager{
		jobID:    jobID,
		tenantID: tenantID,
		enabled:  true,
	}
}

// IsEnabled returns whether dual-write is active
func (dw *DualWriteManager) IsEnabled() bool {
	dw.mu.RLock()
	defer dw.mu.RUnlock()
	return dw.enabled
}

// Enable turns on dual-write mode
func (dw *DualWriteManager) Enable() {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	dw.enabled = true
}

// Disable turns off dual-write mode
func (dw *DualWriteManager) Disable() {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	dw.enabled = false
}

// RecordResult records a dual-write comparison result
func (dw *DualWriteManager) RecordResult(endpointID, eventID string, oldStatus, newStatus int, oldLatency, newLatency int64) {
	dw.mu.Lock()
	defer dw.mu.Unlock()

	result := DualWriteResult{
		ID:              uuid.New().String(),
		EndpointID:      endpointID,
		EventID:         eventID,
		OldSystemStatus: oldStatus,
		NewSystemStatus: newStatus,
		OldLatencyMs:    oldLatency,
		NewLatencyMs:    newLatency,
		Match:           oldStatus == newStatus,
		Timestamp:       time.Now(),
	}
	dw.results = append(dw.results, result)
}

// GetResults returns all recorded dual-write results
func (dw *DualWriteManager) GetResults() []DualWriteResult {
	dw.mu.RLock()
	defer dw.mu.RUnlock()
	out := make([]DualWriteResult, len(dw.results))
	copy(out, dw.results)
	return out
}

// MatchRate returns the percentage of matching results
func (dw *DualWriteManager) MatchRate() float64 {
	dw.mu.RLock()
	defer dw.mu.RUnlock()
	if len(dw.results) == 0 {
		return 0
	}
	matches := 0
	for _, r := range dw.results {
		if r.Match {
			matches++
		}
	}
	return float64(matches) / float64(len(dw.results))
}

// --- TrafficShifter ---

// TrafficShifter provides gradual traffic shifting from 0% to 100%
// using predefined steps: 0% → 25% → 50% → 75% → 100%
type TrafficShifter struct {
	jobID      string
	tenantID   string
	currentPct int
	steps      []int
	stepIndex  int
	mu         sync.RWMutex
}

// DefaultTrafficSteps defines the standard gradual shift steps
var DefaultTrafficSteps = []int{0, 25, 50, 75, 100}

// NewTrafficShifter creates a new traffic shifter with default steps
func NewTrafficShifter(jobID, tenantID string) *TrafficShifter {
	return &TrafficShifter{
		jobID:      jobID,
		tenantID:   tenantID,
		currentPct: 0,
		steps:      DefaultTrafficSteps,
		stepIndex:  0,
	}
}

// NewTrafficShifterWithSteps creates a traffic shifter with custom steps
func NewTrafficShifterWithSteps(jobID, tenantID string, steps []int) *TrafficShifter {
	if len(steps) == 0 {
		steps = DefaultTrafficSteps
	}
	return &TrafficShifter{
		jobID:      jobID,
		tenantID:   tenantID,
		currentPct: steps[0],
		steps:      steps,
		stepIndex:  0,
	}
}

// CurrentPercentage returns the current traffic percentage to the new system
func (ts *TrafficShifter) CurrentPercentage() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.currentPct
}

// NextStep advances to the next traffic step and returns the new percentage
func (ts *TrafficShifter) NextStep() (int, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.stepIndex >= len(ts.steps)-1 {
		return ts.currentPct, fmt.Errorf("already at maximum traffic percentage: %d%%", ts.currentPct)
	}

	ts.stepIndex++
	ts.currentPct = ts.steps[ts.stepIndex]
	return ts.currentPct, nil
}

// SetPercentage sets an exact traffic percentage
func (ts *TrafficShifter) SetPercentage(pct int) error {
	if pct < 0 || pct > 100 {
		return fmt.Errorf("percentage must be 0-100, got %d", pct)
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.currentPct = pct
	// Find closest step index
	for i, s := range ts.steps {
		if s >= pct {
			ts.stepIndex = i
			break
		}
	}
	return nil
}

// Reset resets traffic to 0%
func (ts *TrafficShifter) Reset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.currentPct = 0
	ts.stepIndex = 0
}

// IsComplete returns true if traffic is at 100%
func (ts *TrafficShifter) IsComplete() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.currentPct == 100
}

// --- CutoverService ---

// CutoverService orchestrates the cutover process with phases
type CutoverService struct {
	repo             Repository
	plans            map[string]*CutoverPlanDetailed
	dualWriters      map[string]*DualWriteManager
	trafficShifters  map[string]*TrafficShifter
	mu               sync.RWMutex
}

// NewCutoverService creates a new cutover service
func NewCutoverService(repo Repository) *CutoverService {
	return &CutoverService{
		repo:            repo,
		plans:           make(map[string]*CutoverPlanDetailed),
		dualWriters:     make(map[string]*DualWriteManager),
		trafficShifters: make(map[string]*TrafficShifter),
	}
}

// StartCutover initiates a phased cutover for a migration job
func (cs *CutoverService) StartCutover(ctx context.Context, tenantID, jobID string, dryRun bool) (*CutoverPlanDetailed, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if existing, ok := cs.plans[jobID]; ok {
		if existing.Status == CutoverStatusInProgress {
			return nil, fmt.Errorf("cutover already in progress for job %s", jobID)
		}
	}

	job, err := cs.repo.GetJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration job: %w", err)
	}

	now := time.Now()
	plan := &CutoverPlanDetailed{
		ID:           uuid.New().String(),
		JobID:        jobID,
		TenantID:     tenantID,
		Status:       CutoverStatusInProgress,
		CurrentPhase: CutoverPhaseValidate,
		DryRun:       dryRun,
		Phases: []CutoverPhase{
			{Name: CutoverPhaseValidate, Status: CutoverStatusPending, Description: "Validate endpoints and configurations"},
			{Name: CutoverPhaseDualWrite, Status: CutoverStatusPending, Description: "Write to both old and new systems"},
			{Name: CutoverPhaseGradualShift, Status: CutoverStatusPending, Description: "Gradually shift traffic 0% → 25% → 50% → 75% → 100%"},
			{Name: CutoverPhaseComplete, Status: CutoverStatusPending, Description: "Finalize cutover and decommission old system"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Run validation phase
	endpoints, err := cs.repo.ListEndpointsByJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	validationReport := cs.validateEndpoints(endpoints, dryRun)
	plan.ValidationReport = validationReport

	// Mark validate phase
	plan.Phases[0].Status = CutoverStatusCompleted
	phaseStart := now
	plan.Phases[0].StartedAt = &phaseStart
	phaseEnd := time.Now()
	plan.Phases[0].CompletedAt = &phaseEnd

	if !validationReport.CanProceed {
		plan.Status = CutoverStatusFailed
		plan.Phases[0].Error = "validation failed: cannot proceed with cutover"
		cs.plans[jobID] = plan
		return plan, nil
	}

	if dryRun {
		plan.Status = CutoverStatusCompleted
		cs.plans[jobID] = plan
		return plan, nil
	}

	// Capture rollback snapshot
	plan.RollbackSnapshot = cs.captureSnapshot(job, endpoints)

	// Initialize dual-write and traffic shifter
	cs.dualWriters[jobID] = NewDualWriteManager(jobID, tenantID)
	cs.trafficShifters[jobID] = NewTrafficShifter(jobID, tenantID)

	// Start dual-write phase
	plan.CurrentPhase = CutoverPhaseDualWrite
	dualStart := time.Now()
	plan.Phases[1].Status = CutoverStatusInProgress
	plan.Phases[1].StartedAt = &dualStart
	plan.UpdatedAt = time.Now()

	cs.plans[jobID] = plan
	return plan, nil
}

// GetStatus returns the current cutover status for a job
func (cs *CutoverService) GetStatus(ctx context.Context, jobID string) (*CutoverPlanDetailed, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	plan, ok := cs.plans[jobID]
	if !ok {
		return nil, fmt.Errorf("no cutover plan found for job %s", jobID)
	}

	if shifter, ok := cs.trafficShifters[jobID]; ok {
		plan.TrafficSplitPct = shifter.CurrentPercentage()
	}

	return plan, nil
}

// AdjustTrafficSplit changes the traffic split for an active cutover
func (cs *CutoverService) AdjustTrafficSplit(ctx context.Context, jobID string, percentage int) (*CutoverPlanDetailed, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	plan, ok := cs.plans[jobID]
	if !ok {
		return nil, fmt.Errorf("no cutover plan found for job %s", jobID)
	}

	if plan.Status != CutoverStatusInProgress {
		return nil, fmt.Errorf("cutover is not in progress for job %s", jobID)
	}

	shifter, ok := cs.trafficShifters[jobID]
	if !ok {
		return nil, fmt.Errorf("no traffic shifter found for job %s", jobID)
	}

	if err := shifter.SetPercentage(percentage); err != nil {
		return nil, err
	}

	plan.TrafficSplitPct = percentage
	plan.UpdatedAt = time.Now()

	// Transition to gradual_shift phase if in dual_write
	if plan.CurrentPhase == CutoverPhaseDualWrite && percentage > 0 {
		now := time.Now()
		plan.Phases[1].Status = CutoverStatusCompleted
		plan.Phases[1].CompletedAt = &now
		plan.CurrentPhase = CutoverPhaseGradualShift
		plan.Phases[2].Status = CutoverStatusInProgress
		plan.Phases[2].StartedAt = &now
	}

	// Transition to complete phase if at 100%
	if percentage == 100 {
		now := time.Now()
		plan.Phases[2].Status = CutoverStatusCompleted
		plan.Phases[2].CompletedAt = &now
		plan.CurrentPhase = CutoverPhaseComplete
		plan.Phases[3].Status = CutoverStatusCompleted
		plan.Phases[3].StartedAt = &now
		plan.Phases[3].CompletedAt = &now
		plan.Status = CutoverStatusCompleted
	}

	return plan, nil
}

// Rollback reverts the cutover to the pre-cutover state using the snapshot
func (cs *CutoverService) Rollback(ctx context.Context, jobID string) (*CutoverPlanDetailed, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	plan, ok := cs.plans[jobID]
	if !ok {
		return nil, fmt.Errorf("no cutover plan found for job %s", jobID)
	}

	if plan.Status == CutoverStatusRolledBack {
		return nil, fmt.Errorf("cutover already rolled back for job %s", jobID)
	}

	// Reset traffic to 0%
	if shifter, ok := cs.trafficShifters[jobID]; ok {
		shifter.Reset()
	}

	// Disable dual-write
	if dw, ok := cs.dualWriters[jobID]; ok {
		dw.Disable()
	}

	now := time.Now()
	plan.Status = CutoverStatusRolledBack
	plan.CurrentPhase = CutoverPhaseRollback
	plan.TrafficSplitPct = 0
	plan.UpdatedAt = now

	// Mark all in-progress phases as rolled back
	for i := range plan.Phases {
		if plan.Phases[i].Status == CutoverStatusInProgress {
			plan.Phases[i].Status = CutoverStatusRolledBack
			plan.Phases[i].CompletedAt = &now
		}
	}

	// Restore endpoint states from snapshot if available
	if plan.RollbackSnapshot != nil {
		for _, epSnap := range plan.RollbackSnapshot.EndpointStates {
			ep, err := cs.repo.GetEndpoint(ctx, plan.TenantID, epSnap.EndpointID)
			if err != nil {
				continue
			}
			ep.Status = epSnap.Status
			ep.DestinationID = epSnap.DestinationID
			_ = cs.repo.UpdateEndpoint(ctx, ep)
		}
	}

	return plan, nil
}

// GetDualWriteManager returns the dual-write manager for a job
func (cs *CutoverService) GetDualWriteManager(jobID string) (*DualWriteManager, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	dw, ok := cs.dualWriters[jobID]
	if !ok {
		return nil, fmt.Errorf("no dual-write manager found for job %s", jobID)
	}
	return dw, nil
}

// GetTrafficShifter returns the traffic shifter for a job
func (cs *CutoverService) GetTrafficShifter(jobID string) (*TrafficShifter, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	ts, ok := cs.trafficShifters[jobID]
	if !ok {
		return nil, fmt.Errorf("no traffic shifter found for job %s", jobID)
	}
	return ts, nil
}

// --- Internal helpers ---

func (cs *CutoverService) validateEndpoints(endpoints []MigrationEndpoint, dryRun bool) *ValidationReport {
	report := &ValidationReport{
		TotalEndpoints: len(endpoints),
		CanProceed:     true,
	}

	for _, ep := range endpoints {
		if ep.Status == EndpointStatusValidated || ep.Status == EndpointStatusActive || ep.Status == EndpointStatusImported {
			report.ValidEndpoints++
		} else if ep.Status == EndpointStatusFailed {
			report.InvalidEndpoints++
			report.Errors = append(report.Errors, fmt.Sprintf("endpoint %s is in failed state: %s", ep.ID, ep.ErrorMessage))
		} else {
			report.Warnings = append(report.Warnings, fmt.Sprintf("endpoint %s has status %s", ep.ID, ep.Status))
		}
	}

	if report.InvalidEndpoints > 0 {
		report.CanProceed = false
	}

	if len(endpoints) == 0 {
		report.CanProceed = false
		report.Errors = append(report.Errors, "no endpoints found for migration")
	}

	if dryRun {
		report.DryRunSummary = fmt.Sprintf("Dry run: %d endpoints would be migrated, %d invalid, %d warnings",
			report.ValidEndpoints, report.InvalidEndpoints, len(report.Warnings))
	}

	return report
}

func (cs *CutoverService) captureSnapshot(job *MigrationJob, endpoints []MigrationEndpoint) *RollbackSnapshot {
	snapshot := &RollbackSnapshot{
		ID:                uuid.New().String(),
		JobID:             job.ID,
		TenantID:          job.TenantID,
		PreviousJobStatus: job.Status,
		CapturedAt:        time.Now(),
	}

	for _, ep := range endpoints {
		snapshot.EndpointStates = append(snapshot.EndpointStates, EndpointSnapshot{
			EndpointID:    ep.ID,
			SourceID:      ep.SourceID,
			SourceURL:     ep.SourceURL,
			DestinationID: ep.DestinationID,
			Status:        ep.Status,
		})
	}

	return snapshot
}
