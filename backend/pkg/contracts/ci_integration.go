package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CIValidationResult holds CI/CD pipeline validation results in a SARIF-compatible structure.
type CIValidationResult struct {
	ID            string          `json:"id"`
	ContractID    string          `json:"contract_id"`
	RunID         string          `json:"run_id"`
	Environment   string          `json:"environment,omitempty"`
	Branch        string          `json:"branch,omitempty"`
	CommitSHA     string          `json:"commit_sha,omitempty"`
	Passed        bool            `json:"passed"`
	TotalChecks   int             `json:"total_checks"`
	FailedChecks  int             `json:"failed_checks"`
	Warnings      int             `json:"warnings"`
	Violations    []CIViolation   `json:"violations,omitempty"`
	DurationMs    int             `json:"duration_ms"`
	ValidatedAt   time.Time       `json:"validated_at"`
}

// CIViolation represents a single violation found during CI validation.
type CIViolation struct {
	RuleID      string `json:"rule_id"`
	Level       string `json:"level"` // error, warning, note
	Message     string `json:"message"`
	Path        string `json:"path,omitempty"`
	ContractID  string `json:"contract_id,omitempty"`
}

// SARIFReport implements the SARIF 2.1.0 static analysis format.
type SARIFReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single analysis run within a SARIF report.
type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

// SARIFTool describes the tool that produced the analysis.
type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

// SARIFDriver contains metadata and rules for the analysis tool.
type SARIFDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []SARIFRule `json:"rules,omitempty"`
}

// SARIFRule describes a rule used during analysis.
type SARIFRule struct {
	ID               string          `json:"id"`
	Name             string          `json:"name,omitempty"`
	ShortDescription SARIFMessage    `json:"shortDescription"`
	DefaultConfig    SARIFRuleConfig `json:"defaultConfiguration,omitempty"`
}

// SARIFRuleConfig holds the default severity for a rule.
type SARIFRuleConfig struct {
	Level string `json:"level"`
}

// SARIFResult represents a single finding in a SARIF report.
type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"` // error, warning, note
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations,omitempty"`
}

// SARIFMessage is a SARIF text message.
type SARIFMessage struct {
	Text string `json:"text"`
}

// SARIFLocation points to where a finding was detected.
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation,omitempty"`
	LogicalLocations []SARIFLogicalLocation `json:"logicalLocations,omitempty"`
}

// SARIFPhysicalLocation describes an artifact location.
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
}

// SARIFArtifactLocation is a URI reference to an artifact.
type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// SARIFLogicalLocation identifies a logical code element.
type SARIFLogicalLocation struct {
	Name string `json:"name"`
	Kind string `json:"kind,omitempty"`
}

// CIValidator validates contracts and generates CI/CD-compatible reports.
type CIValidator struct {
	validator *Validator
	detector  *BreakingChangeDetector
}

// NewCIValidator creates a new CI validator.
func NewCIValidator() *CIValidator {
	return &CIValidator{
		validator: NewValidator(),
		detector:  NewBreakingChangeDetector(),
	}
}

// ValidateForCI validates a set of contracts and returns a CI-compatible result.
func (c *CIValidator) ValidateForCI(ctx context.Context, contracts []Contract, payloads map[string]json.RawMessage) *CIValidationResult {
	start := time.Now()
	result := &CIValidationResult{
		ID:          uuid.New().String(),
		RunID:       fmt.Sprintf("ci-run-%d", time.Now().UnixNano()),
		Passed:      true,
		ValidatedAt: time.Now(),
	}

	for _, contract := range contracts {
		result.TotalChecks++

		payload, exists := payloads[contract.ID]
		if !exists {
			result.Violations = append(result.Violations, CIViolation{
				RuleID:     "CONTRACT_MISSING_PAYLOAD",
				Level:      "warning",
				Message:    fmt.Sprintf("no payload provided for contract '%s'", contract.Name),
				ContractID: contract.ID,
			})
			result.Warnings++
			continue
		}

		vr := c.validator.ValidatePayload(&contract, payload, nil)
		if !vr.IsValid {
			result.Passed = false
			result.FailedChecks++
			for _, e := range vr.Errors {
				result.Violations = append(result.Violations, CIViolation{
					RuleID:     e.Code,
					Level:      "error",
					Message:    e.Message,
					Path:       e.Path,
					ContractID: contract.ID,
				})
			}
		}
	}

	result.DurationMs = int(time.Since(start).Milliseconds())
	return result
}

// GenerateSARIF converts a CI validation result into a SARIF 2.1.0 report.
func (c *CIValidator) GenerateSARIF(result *CIValidationResult) *SARIFReport {
	rulesMap := make(map[string]SARIFRule)
	var results []SARIFResult

	for _, v := range result.Violations {
		if _, exists := rulesMap[v.RuleID]; !exists {
			rulesMap[v.RuleID] = SARIFRule{
				ID:               v.RuleID,
				ShortDescription: SARIFMessage{Text: v.RuleID},
				DefaultConfig:    SARIFRuleConfig{Level: v.Level},
			}
		}

		sr := SARIFResult{
			RuleID:  v.RuleID,
			Level:   v.Level,
			Message: SARIFMessage{Text: v.Message},
		}
		if v.Path != "" {
			sr.Locations = []SARIFLocation{
				{
					LogicalLocations: []SARIFLogicalLocation{
						{Name: v.Path, Kind: "jsonPath"},
					},
				},
			}
		}
		results = append(results, sr)
	}

	rules := make([]SARIFRule, 0, len(rulesMap))
	for _, r := range rulesMap {
		rules = append(rules, r)
	}

	return &SARIFReport{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:           "waas-contract-validator",
						Version:        "1.0.0",
						InformationURI: "https://github.com/josedab/waas",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}
}

// BreakingChangeChecker compares contract versions and identifies breaking changes.
type BreakingChangeChecker struct {
	detector *BreakingChangeDetector
}

// NewBreakingChangeChecker creates a new breaking change checker.
func NewBreakingChangeChecker() *BreakingChangeChecker {
	return &BreakingChangeChecker{
		detector: NewBreakingChangeDetector(),
	}
}

// CheckBreakingChanges compares old and new contract schemas and returns a list of breaking changes.
func (b *BreakingChangeChecker) CheckBreakingChanges(oldContract, newContract *Contract) (*BreakingChange, error) {
	result, err := b.detector.DetectChanges(oldContract.RequestSchema, newContract.RequestSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to check breaking changes: %w", err)
	}

	result.ID = uuid.New().String()
	result.ContractID = newContract.ID
	result.OldVersion = oldContract.Version
	result.NewVersion = newContract.Version

	return result, nil
}

// SchemaRegistryClient manages schema auto-registration and version tracking.
type SchemaRegistryClient struct {
	schemas map[string][]SchemaRegistryEntry
}

// SchemaRegistryEntry represents a registered schema version.
type SchemaRegistryEntry struct {
	ID           string          `json:"id"`
	ContractID   string          `json:"contract_id"`
	Version      string          `json:"version"`
	Schema       json.RawMessage `json:"schema"`
	IsBreaking   bool            `json:"is_breaking"`
	RegisteredAt time.Time       `json:"registered_at"`
}

// NewSchemaRegistryClient creates a new schema registry client.
func NewSchemaRegistryClient() *SchemaRegistryClient {
	return &SchemaRegistryClient{
		schemas: make(map[string][]SchemaRegistryEntry),
	}
}

// RegisterSchema registers a contract's schema in the registry.
func (r *SchemaRegistryClient) RegisterSchema(contract *Contract) (*SchemaRegistryEntry, error) {
	if contract == nil {
		return nil, fmt.Errorf("contract must not be nil")
	}

	var schemaCheck interface{}
	if err := json.Unmarshal(contract.RequestSchema, &schemaCheck); err != nil {
		return nil, fmt.Errorf("schema must be valid JSON: %w", err)
	}

	isBreaking := false
	entries := r.schemas[contract.ID]
	if len(entries) > 0 {
		latest := entries[len(entries)-1]
		detector := NewBreakingChangeDetector()
		bc, err := detector.DetectChanges(latest.Schema, contract.RequestSchema)
		if err == nil && bc.HasBreakingChanges {
			isBreaking = true
		}
	}

	entry := &SchemaRegistryEntry{
		ID:           uuid.New().String(),
		ContractID:   contract.ID,
		Version:      contract.Version,
		Schema:       contract.RequestSchema,
		IsBreaking:   isBreaking,
		RegisteredAt: time.Now(),
	}

	r.schemas[contract.ID] = append(r.schemas[contract.ID], *entry)
	return entry, nil
}

// GetVersions returns all registered schema versions for a contract.
func (r *SchemaRegistryClient) GetVersions(contractID string) []SchemaRegistryEntry {
	return r.schemas[contractID]
}

// GetLatestVersion returns the latest registered schema for a contract.
func (r *SchemaRegistryClient) GetLatestVersion(contractID string) *SchemaRegistryEntry {
	entries := r.schemas[contractID]
	if len(entries) == 0 {
		return nil
	}
	latest := entries[len(entries)-1]
	return &latest
}

// ChangelogEntry represents a single entry in a generated changelog.
type ChangelogEntry struct {
	Version     string    `json:"version"`
	Date        time.Time `json:"date"`
	IsBreaking  bool      `json:"is_breaking"`
	Summary     string    `json:"summary"`
	Changes     []Change  `json:"changes"`
}

// GenerateChangelog creates a changelog from a series of schema versions.
func GenerateChangelog(entries []SchemaRegistryEntry) []ChangelogEntry {
	if len(entries) < 2 {
		return nil
	}

	detector := NewBreakingChangeDetector()
	var changelog []ChangelogEntry

	for i := 1; i < len(entries); i++ {
		prev := entries[i-1]
		curr := entries[i]

		bc, err := detector.DetectChanges(prev.Schema, curr.Schema)
		if err != nil {
			continue
		}

		summary := fmt.Sprintf("Updated from %s to %s", prev.Version, curr.Version)
		if bc.HasBreakingChanges {
			summary += " (BREAKING)"
		}

		changelog = append(changelog, ChangelogEntry{
			Version:    curr.Version,
			Date:       curr.RegisteredAt,
			IsBreaking: bc.HasBreakingChanges,
			Summary:    summary,
			Changes:    bc.Changes,
		})
	}

	return changelog
}
