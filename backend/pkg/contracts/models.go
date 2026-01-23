package contracts

import "time"

// StrictnessLevel defines how strictly a contract is validated
type StrictnessLevel string

const (
	StrictnessLoose    StrictnessLevel = "loose"
	StrictnessStandard StrictnessLevel = "standard"
	StrictnessStrict   StrictnessLevel = "strict"
)

// CreateContractRequest is the request body for creating/updating a contract
type CreateContractRequest struct {
	EndpointID string          `json:"endpoint_id" binding:"required"`
	Name       string          `json:"name" binding:"required,min=1,max=255"`
	Version    string          `json:"version" binding:"required"`
	EventType  string          `json:"event_type" binding:"required"`
	Schema     string          `json:"schema" binding:"required"`
	Strictness StrictnessLevel `json:"strictness,omitempty"`
}

// ValidatePayloadRequest is the request body for validating a payload against a contract
type ValidatePayloadRequest struct {
	ContractID string `json:"contract_id" binding:"required"`
	Payload    string `json:"payload" binding:"required"`
}

// ContractTestResult represents the result of validating a payload against a contract
type ContractTestResult struct {
	ID         string      `json:"id"`
	TenantID   string      `json:"tenant_id"`
	ContractID string      `json:"contract_id"`
	EndpointID string      `json:"endpoint_id"`
	Passed     bool        `json:"passed"`
	Violations []Violation `json:"violations,omitempty"`
	TestedAt   time.Time   `json:"tested_at"`
	DurationMs int         `json:"duration_ms"`
}

// Violation represents a single contract violation found during validation
type Violation struct {
	Path     string `json:"path"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// SchemaDiff represents the differences between two contract versions
type SchemaDiff struct {
	ContractID string       `json:"contract_id"`
	OldVersion string       `json:"old_version"`
	NewVersion string       `json:"new_version"`
	Changes    []DiffChange `json:"changes"`
	IsBreaking bool         `json:"is_breaking"`
}

// DiffChange represents a single change between schema versions
type DiffChange struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	Description string `json:"description"`
	IsBreaking  bool   `json:"is_breaking"`
}

// ContractStatus represents the overall health of contracts for a tenant
type ContractStatus struct {
	TotalContracts  int     `json:"total_contracts"`
	ActiveContracts int     `json:"active_contracts"`
	PassRate        float64 `json:"pass_rate"`
	LastTestedAt    string  `json:"last_tested_at,omitempty"`
}
