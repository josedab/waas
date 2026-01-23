package multiregion

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrRegionNotFound         = errors.New("region not found")
	ErrResidencyViolation     = errors.New("data residency policy violation")
	ErrPolicyAlreadyExists    = errors.New("residency policy already exists for this tenant")
	ErrInvalidResidencyRegion = errors.New("invalid residency region")
)

// DataResidencyPolicy defines where tenant data must be stored
type DataResidencyPolicy struct {
	ID                string              `json:"id" db:"id"`
	TenantID          string              `json:"tenant_id" db:"tenant_id"`
	PrimaryRegion     string              `json:"primary_region" db:"primary_region"`
	AllowedRegions    []string            `json:"allowed_regions" db:"allowed_regions"`
	BlockedRegions    []string            `json:"blocked_regions" db:"blocked_regions"`
	Regulation        ComplianceRegulation `json:"regulation" db:"regulation"`
	EnforceStrict     bool                `json:"enforce_strict" db:"enforce_strict"`
	DataCategories    []DataCategory      `json:"data_categories" db:"data_categories"`
	CreatedAt         time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at" db:"updated_at"`
}

// ComplianceRegulation represents the regulatory framework
type ComplianceRegulation string

const (
	RegulationGDPR    ComplianceRegulation = "gdpr"
	RegulationCCPA    ComplianceRegulation = "ccpa"
	RegulationHIPAA   ComplianceRegulation = "hipaa"
	RegulationPCIDSS  ComplianceRegulation = "pci_dss"
	RegulationSOX     ComplianceRegulation = "sox"
	RegulationCustom  ComplianceRegulation = "custom"
)

// DataCategory classifies types of data for residency rules
type DataCategory string

const (
	DataCategoryPII       DataCategory = "pii"
	DataCategoryPHI       DataCategory = "phi"
	DataCategoryFinancial DataCategory = "financial"
	DataCategoryPayload   DataCategory = "payload"
	DataCategoryMetadata  DataCategory = "metadata"
	DataCategoryLogs      DataCategory = "logs"
	DataCategoryAll       DataCategory = "all"
)

// DataFlowLog records cross-region data movements for audit
type DataFlowLog struct {
	ID             string       `json:"id" db:"id"`
	TenantID       string       `json:"tenant_id" db:"tenant_id"`
	SourceRegion   string       `json:"source_region" db:"source_region"`
	TargetRegion   string       `json:"target_region" db:"target_region"`
	DataCategory   DataCategory `json:"data_category" db:"data_category"`
	Operation      string       `json:"operation" db:"operation"`
	BytesTransferred int64      `json:"bytes_transferred" db:"bytes_transferred"`
	Allowed        bool         `json:"allowed" db:"allowed"`
	PolicyID       string       `json:"policy_id" db:"policy_id"`
	Timestamp      time.Time    `json:"timestamp" db:"timestamp"`
}

// RegionComplianceStatus shows compliance state per region
type RegionComplianceStatus struct {
	RegionID           string               `json:"region_id"`
	Certifications     []string             `json:"certifications"`
	SupportedRegulations []ComplianceRegulation `json:"supported_regulations"`
	DataSovereignty    bool                 `json:"data_sovereignty"`
	EncryptionAtRest   bool                 `json:"encryption_at_rest"`
	EncryptionInTransit bool                `json:"encryption_in_transit"`
	AuditLogEnabled    bool                 `json:"audit_log_enabled"`
}

// DataResidencyService manages data residency policies
type DataResidencyService struct {
	regions  map[string]*Region
	policies map[string]*DataResidencyPolicy // keyed by tenant_id
}

// NewDataResidencyService creates a new data residency service
func NewDataResidencyService() *DataResidencyService {
	return &DataResidencyService{
		regions:  DefaultRegions(),
		policies: make(map[string]*DataResidencyPolicy),
	}
}

// DefaultRegions returns the default set of deployment regions
func DefaultRegions() map[string]*Region {
	return map[string]*Region{
		"us-east-1": {
			ID: "us-east-1", Name: "US East (Virginia)", Code: "us-east-1",
			IsActive: true, IsPrimary: true, Priority: 1,
			Metadata: Metadata{Cloud: "aws", Datacenter: "us-east-1", Coordinates: &GeoCoord{Latitude: 38.95, Longitude: -77.45}},
		},
		"us-west-2": {
			ID: "us-west-2", Name: "US West (Oregon)", Code: "us-west-2",
			IsActive: true, Priority: 2,
			Metadata: Metadata{Cloud: "aws", Datacenter: "us-west-2", Coordinates: &GeoCoord{Latitude: 45.59, Longitude: -122.33}},
		},
		"eu-west-1": {
			ID: "eu-west-1", Name: "EU West (Ireland)", Code: "eu-west-1",
			IsActive: true, Priority: 3,
			Metadata: Metadata{Cloud: "aws", Datacenter: "eu-west-1", Coordinates: &GeoCoord{Latitude: 53.35, Longitude: -6.26}},
		},
		"ap-southeast-1": {
			ID: "ap-southeast-1", Name: "Asia Pacific (Singapore)", Code: "ap-southeast-1",
			IsActive: true, Priority: 4,
			Metadata: Metadata{Cloud: "aws", Datacenter: "ap-southeast-1", Coordinates: &GeoCoord{Latitude: 1.35, Longitude: 103.82}},
		},
		"ap-northeast-1": {
			ID: "ap-northeast-1", Name: "Asia Pacific (Tokyo)", Code: "ap-northeast-1",
			IsActive: true, Priority: 5,
			Metadata: Metadata{Cloud: "aws", Datacenter: "ap-northeast-1", Coordinates: &GeoCoord{Latitude: 35.68, Longitude: 139.69}},
		},
	}
}

// SetPolicy creates or updates a data residency policy
func (s *DataResidencyService) SetPolicy(_ context.Context, policy *DataResidencyPolicy) error {
	if policy.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if _, ok := s.regions[policy.PrimaryRegion]; !ok {
		return ErrInvalidResidencyRegion
	}
	for _, r := range policy.AllowedRegions {
		if _, ok := s.regions[r]; !ok {
			return fmt.Errorf("%w: %s", ErrInvalidResidencyRegion, r)
		}
	}

	now := time.Now()
	if policy.ID == "" {
		policy.ID = fmt.Sprintf("drp-%d", now.UnixNano())
	}
	policy.UpdatedAt = now
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}

	s.policies[policy.TenantID] = policy
	return nil
}

// GetPolicy retrieves the data residency policy for a tenant
func (s *DataResidencyService) GetPolicy(_ context.Context, tenantID string) (*DataResidencyPolicy, error) {
	policy, ok := s.policies[tenantID]
	if !ok {
		return nil, ErrRegionNotFound
	}
	return policy, nil
}

// CheckDataFlow validates if a cross-region data transfer is allowed
func (s *DataResidencyService) CheckDataFlow(_ context.Context, tenantID, sourceRegion, targetRegion string) (bool, error) {
	policy, ok := s.policies[tenantID]
	if !ok {
		return true, nil // No policy = all transfers allowed
	}

	// If source and target are the same, always allowed
	if sourceRegion == targetRegion {
		return true, nil
	}

	// Check blocked regions
	for _, blocked := range policy.BlockedRegions {
		if targetRegion == blocked {
			return false, ErrResidencyViolation
		}
	}

	// If allowed regions specified, target must be in the list
	if len(policy.AllowedRegions) > 0 {
		for _, allowed := range policy.AllowedRegions {
			if targetRegion == allowed {
				return true, nil
			}
		}
		return false, ErrResidencyViolation
	}

	return true, nil
}

// ListRegions returns all available regions
func (s *DataResidencyService) ListRegions() []*Region {
	regions := make([]*Region, 0, len(s.regions))
	for _, r := range s.regions {
		regions = append(regions, r)
	}
	return regions
}

// GetRegionCompliance returns compliance status for a region
func (s *DataResidencyService) GetRegionCompliance(regionID string) (*RegionComplianceStatus, error) {
	region, ok := s.regions[regionID]
	if !ok {
		return nil, ErrRegionNotFound
	}

	status := &RegionComplianceStatus{
		RegionID:            region.ID,
		EncryptionAtRest:    true,
		EncryptionInTransit: true,
		AuditLogEnabled:     true,
		DataSovereignty:     true,
	}

	// Assign certifications and regulations based on region
	switch {
	case region.Code == "eu-west-1":
		status.Certifications = []string{"ISO27001", "SOC2", "GDPR"}
		status.SupportedRegulations = []ComplianceRegulation{RegulationGDPR, RegulationPCIDSS}
	case region.Code == "us-east-1" || region.Code == "us-west-2":
		status.Certifications = []string{"ISO27001", "SOC2", "HIPAA", "FedRAMP"}
		status.SupportedRegulations = []ComplianceRegulation{RegulationHIPAA, RegulationCCPA, RegulationSOX, RegulationPCIDSS}
	default:
		status.Certifications = []string{"ISO27001", "SOC2"}
		status.SupportedRegulations = []ComplianceRegulation{RegulationPCIDSS}
	}

	return status, nil
}
