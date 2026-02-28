package capacityplanner

import (
	"fmt"
	"sync"
	"time"
)

// Repository defines the data access interface for capacity planning.
type Repository interface {
	SaveReport(report *CapacityReport) error
	GetReport(id string) (*CapacityReport, error)
	ListReports(tenantID string) ([]*CapacityReport, error)

	SaveTrafficSnapshot(tenantID string, snapshot *TrafficSnapshot) error
	GetTrafficHistory(tenantID string, start, end time.Time) ([]TrafficSnapshot, error)

	SaveAlert(alert *CapacityAlert) error
	GetAlert(id string) (*CapacityAlert, error)
	ListAlerts(tenantID string) ([]CapacityAlert, error)
	DeleteAlert(id string) error

	GetAlertThreshold(tenantID, resource string) (float64, error)
	SetAlertThreshold(tenantID, resource string, value float64) error
}

// MemoryRepository provides an in-memory implementation.
type MemoryRepository struct {
	mu         sync.RWMutex
	reports    map[string]*CapacityReport
	snapshots  map[string][]TrafficSnapshot
	alerts     map[string]*CapacityAlert
	thresholds map[string]float64
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		reports:    make(map[string]*CapacityReport),
		snapshots:  make(map[string][]TrafficSnapshot),
		alerts:     make(map[string]*CapacityAlert),
		thresholds: make(map[string]float64),
	}
}

func (r *MemoryRepository) SaveReport(report *CapacityReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports[report.ID] = report
	return nil
}

func (r *MemoryRepository) GetReport(id string) (*CapacityReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if report, ok := r.reports[id]; ok {
		return report, nil
	}
	return nil, fmt.Errorf("report not found: %s", id)
}

func (r *MemoryRepository) ListReports(tenantID string) ([]*CapacityReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*CapacityReport
	for _, report := range r.reports {
		if report.TenantID == tenantID {
			result = append(result, report)
		}
	}
	return result, nil
}

func (r *MemoryRepository) SaveTrafficSnapshot(tenantID string, snapshot *TrafficSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots[tenantID] = append(r.snapshots[tenantID], *snapshot)
	return nil
}

func (r *MemoryRepository) GetTrafficHistory(tenantID string, start, end time.Time) ([]TrafficSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []TrafficSnapshot
	for _, s := range r.snapshots[tenantID] {
		if !s.Timestamp.Before(start) && !s.Timestamp.After(end) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *MemoryRepository) SaveAlert(alert *CapacityAlert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alerts[alert.ID] = alert
	return nil
}

func (r *MemoryRepository) GetAlert(id string) (*CapacityAlert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if alert, ok := r.alerts[id]; ok {
		return alert, nil
	}
	return nil, fmt.Errorf("alert not found: %s", id)
}

func (r *MemoryRepository) ListAlerts(tenantID string) ([]CapacityAlert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []CapacityAlert
	for _, alert := range r.alerts {
		if alert.TenantID == tenantID {
			result = append(result, *alert)
		}
	}
	return result, nil
}

func (r *MemoryRepository) DeleteAlert(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.alerts, id)
	return nil
}

func thresholdKey(tenantID, resource string) string {
	return tenantID + ":" + resource
}

func (r *MemoryRepository) GetAlertThreshold(tenantID, resource string) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if v, ok := r.thresholds[thresholdKey(tenantID, resource)]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("threshold not found for %s/%s", tenantID, resource)
}

func (r *MemoryRepository) SetAlertThreshold(tenantID, resource string, value float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.thresholds[thresholdKey(tenantID, resource)] = value
	return nil
}
