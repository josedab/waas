package contracts

import (
"context"
"encoding/json"
"fmt"
"strings"
"time"

"github.com/google/uuid"
)

// Service provides contract testing functionality
type Service struct {
repo Repository
}

// NewService creates a new contract testing service
func NewService(repo Repository) *Service {
return &Service{repo: repo}
}

// CreateContract creates a new webhook contract
func (s *Service) CreateContract(ctx context.Context, tenantID string, req *CreateContractRequest) (*Contract, error) {
var schemaCheck interface{}
if err := json.Unmarshal([]byte(req.Schema), &schemaCheck); err != nil {
return nil, fmt.Errorf("schema must be valid JSON: %w", err)
}

if req.Strictness == "" {
req.Strictness = StrictnessStandard
}

now := time.Now()
contract := &Contract{
ID:            uuid.New().String(),
TenantID:      tenantID,
Name:          req.Name,
Version:       req.Version,
EventType:     req.EventType,
SchemaFormat:  string(req.Strictness),
RequestSchema: json.RawMessage(req.Schema),
Status:        "active",
CreatedAt:     now,
UpdatedAt:     now,
}

if err := s.repo.CreateContract(ctx, contract); err != nil {
return nil, fmt.Errorf("failed to create contract: %w", err)
}

return contract, nil
}

// GetContract retrieves a contract by ID
func (s *Service) GetContract(ctx context.Context, tenantID, contractID string) (*Contract, error) {
return s.repo.GetContract(ctx, tenantID, contractID)
}

// ListContracts lists all contracts for a tenant
func (s *Service) ListContracts(ctx context.Context, tenantID string, limit, offset int) ([]Contract, int, error) {
if limit <= 0 {
limit = 50
}
return s.repo.ListContracts(ctx, tenantID, limit, offset)
}

// UpdateContract updates an existing contract
func (s *Service) UpdateContract(ctx context.Context, tenantID, contractID string, req *CreateContractRequest) (*Contract, error) {
contract, err := s.repo.GetContract(ctx, tenantID, contractID)
if err != nil {
return nil, err
}

contract.Name = req.Name
contract.Version = req.Version
contract.EventType = req.EventType
contract.RequestSchema = json.RawMessage(req.Schema)
contract.SchemaFormat = string(req.Strictness)
contract.UpdatedAt = time.Now()

if err := s.repo.UpdateContract(ctx, contract); err != nil {
return nil, fmt.Errorf("failed to update contract: %w", err)
}

return contract, nil
}

// DeleteContract deletes a contract
func (s *Service) DeleteContract(ctx context.Context, tenantID, contractID string) error {
return s.repo.DeleteContract(ctx, tenantID, contractID)
}

// ValidatePayload validates a JSON payload against a contract's schema
func (s *Service) ValidatePayload(ctx context.Context, tenantID string, req *ValidatePayloadRequest) (*ContractTestResult, error) {
start := time.Now()

contract, err := s.repo.GetContract(ctx, tenantID, req.ContractID)
if err != nil {
return nil, fmt.Errorf("contract not found: %w", err)
}

var payload map[string]interface{}
if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
return nil, fmt.Errorf("payload must be valid JSON: %w", err)
}

var schema map[string]interface{}
if err := json.Unmarshal(contract.RequestSchema, &schema); err != nil {
return nil, fmt.Errorf("contract schema is invalid: %w", err)
}

strictness := StrictnessLevel(contract.SchemaFormat)
if strictness == "" {
strictness = StrictnessStandard
}
violations := validateAgainstSchema(payload, schema, strictness, "")

result := &ContractTestResult{
ID:         uuid.New().String(),
TenantID:   tenantID,
ContractID: contract.ID,
Passed:     len(violations) == 0,
Violations: violations,
TestedAt:   time.Now(),
DurationMs: int(time.Since(start).Milliseconds()),
}

if err := s.repo.SaveTestResult(ctx, result); err != nil {
return nil, fmt.Errorf("failed to save test result: %w", err)
}

return result, nil
}

// DiffContracts compares two versions of a contract and identifies breaking changes
func (s *Service) DiffContracts(ctx context.Context, tenantID, contractID, oldVersion, newVersion string) (*SchemaDiff, error) {
contract, err := s.repo.GetContract(ctx, tenantID, contractID)
if err != nil {
return nil, err
}

var oldSchema, newSchema map[string]interface{}
json.Unmarshal(contract.RequestSchema, &newSchema)

if oldVersion == contract.Version {
oldSchema = newSchema
} else {
oldSchema = make(map[string]interface{})
}

diff := &SchemaDiff{
ContractID: contractID,
OldVersion: oldVersion,
NewVersion: newVersion,
Changes:    computeChanges(oldSchema, newSchema, ""),
}

for _, change := range diff.Changes {
if change.IsBreaking {
diff.IsBreaking = true
break
}
}

return diff, nil
}

// GetContractStatus returns an overview of contract health for a tenant
func (s *Service) GetContractStatus(ctx context.Context, tenantID string) (*ContractStatus, error) {
return s.repo.GetContractStatus(ctx, tenantID)
}

// GetTestResults lists test results for a contract
func (s *Service) GetTestResults(ctx context.Context, tenantID, contractID string, limit, offset int) ([]ContractTestResult, error) {
if limit <= 0 {
limit = 50
}
return s.repo.ListTestResults(ctx, tenantID, contractID, limit, offset)
}

func validateAgainstSchema(payload, schema map[string]interface{}, strictness StrictnessLevel, path string) []Violation {
var violations []Violation

if props, ok := schema["properties"].(map[string]interface{}); ok {
required, _ := schema["required"].([]interface{})
requiredSet := make(map[string]bool)
for _, r := range required {
if s, ok := r.(string); ok {
requiredSet[s] = true
}
}

for name, propSchema := range props {
fieldPath := path + "." + name
if path == "" {
fieldPath = name
}

value, exists := payload[name]
if !exists && requiredSet[name] {
violations = append(violations, Violation{
Path:     fieldPath,
Expected: "required field",
Actual:   "missing",
Message:  fmt.Sprintf("required field '%s' is missing", name),
Severity: "error",
})
continue
}

if exists && propSchema != nil {
if propMap, ok := propSchema.(map[string]interface{}); ok {
expectedType, _ := propMap["type"].(string)
if expectedType != "" && !matchesType(value, expectedType) {
violations = append(violations, Violation{
Path:     fieldPath,
Expected: expectedType,
Actual:   fmt.Sprintf("%T", value),
Message:  fmt.Sprintf("field '%s' expected type '%s'", name, expectedType),
Severity: "error",
})
}
}
}
}

if strictness == StrictnessStrict {
for name := range payload {
if _, exists := props[name]; !exists {
fieldPath := path + "." + name
if path == "" {
fieldPath = name
}
violations = append(violations, Violation{
Path:     fieldPath,
Expected: "not present",
Actual:   "additional property",
Message:  fmt.Sprintf("unexpected field '%s' in strict mode", name),
Severity: "warning",
})
}
}
}
}

return violations
}

func matchesType(value interface{}, expectedType string) bool {
switch expectedType {
case "string":
_, ok := value.(string)
return ok
case "number", "integer":
_, ok := value.(float64)
return ok
case "boolean":
_, ok := value.(bool)
return ok
case "object":
_, ok := value.(map[string]interface{})
return ok
case "array":
_, ok := value.([]interface{})
return ok
}
return true
}

func computeChanges(oldSchema, newSchema map[string]interface{}, path string) []DiffChange {
var changes []DiffChange

oldProps, _ := oldSchema["properties"].(map[string]interface{})
newProps, _ := newSchema["properties"].(map[string]interface{})

for name := range oldProps {
fieldPath := joinPath(path, name)
if _, exists := newProps[name]; !exists {
changes = append(changes, DiffChange{
Type:        "removed",
Path:        fieldPath,
Description: fmt.Sprintf("field '%s' was removed", name),
IsBreaking:  true,
})
}
}

for name := range newProps {
fieldPath := joinPath(path, name)
if _, exists := oldProps[name]; !exists {
changes = append(changes, DiffChange{
Type:        "added",
Path:        fieldPath,
Description: fmt.Sprintf("field '%s' was added", name),
IsBreaking:  false,
})
}
}

return changes
}

func joinPath(base, name string) string {
if base == "" {
return name
}
return strings.Join([]string{base, name}, ".")
}
