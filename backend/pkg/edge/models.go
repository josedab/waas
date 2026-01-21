// Package edge provides edge function runtime for low-latency webhook processing
package edge

import (
	"encoding/json"
	"time"
)

// RuntimeType represents supported edge runtime platforms
type RuntimeType string

const (
	RuntimeCloudflareWorkers RuntimeType = "cloudflare_workers"
	RuntimeCloudflare        RuntimeType = "cloudflare_workers" // Alias
	RuntimeDenoDeloy         RuntimeType = "deno_deploy"
	RuntimeDeno              RuntimeType = "deno_deploy" // Alias
	RuntimeFastlyCompute     RuntimeType = "fastly_compute"
	RuntimeFastly            RuntimeType = "fastly_compute" // Alias
	RuntimeVercelEdge        RuntimeType = "vercel_edge"
	RuntimeVercel            RuntimeType = "vercel_edge" // Alias
	RuntimeAWSLambdaEdge     RuntimeType = "aws_lambda_edge"
	RuntimeLambdaEdge        RuntimeType = "aws_lambda_edge" // Alias
	RuntimeLocal             RuntimeType = "local"           // For testing
)

// FunctionStatus represents edge function lifecycle status
type FunctionStatus string

const (
	FunctionStatusDraft     FunctionStatus = "draft"
	FunctionStatusDeploying FunctionStatus = "deploying"
	FunctionStatusActive    FunctionStatus = "active"
	FunctionStatusFailed    FunctionStatus = "failed"
	FunctionStatusDisabled  FunctionStatus = "disabled"
)

// EdgeRegion represents edge deployment regions
type EdgeRegion string

const (
	RegionGlobal      EdgeRegion = "global"
	RegionUSEast      EdgeRegion = "us-east"
	RegionUSWest      EdgeRegion = "us-west"
	RegionEUWest      EdgeRegion = "eu-west"
	RegionEUCentral   EdgeRegion = "eu-central"
	RegionAPACEast    EdgeRegion = "apac-east"
	RegionAPACSouth   EdgeRegion = "apac-south"
	RegionAPSoutheast EdgeRegion = "apac-south" // Alias
	RegionAPNortheast EdgeRegion = "apac-east"  // Alias
	RegionSAEast      EdgeRegion = "sa-east"
	RegionMECentral   EdgeRegion = "me-central"
	RegionAFSouth     EdgeRegion = "af-south"
	RegionOCEast      EdgeRegion = "oc-east"
)

// AllEdgeRegions returns all available edge regions
func AllEdgeRegions() []EdgeRegion {
	return []EdgeRegion{
		RegionUSEast, RegionUSWest, RegionEUWest, RegionEUCentral,
		RegionAPACEast, RegionAPACSouth, RegionSAEast, RegionMECentral,
		RegionAFSouth, RegionOCEast,
	}
}

// EdgeFunction represents a deployed edge function
type EdgeFunction struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Runtime         RuntimeType       `json:"runtime"`
	Status          FunctionStatus    `json:"status"`
	Version         int               `json:"version"`
	Code            string            `json:"code"`
	CompiledCode    string            `json:"compiled_code,omitempty"`
	EntryPoint      string            `json:"entry_point"`
	Regions         []EdgeRegion      `json:"regions"`
	Config          *FunctionConfig   `json:"config"`
	Triggers        []FunctionTrigger `json:"triggers,omitempty"`
	Environment     map[string]string `json:"environment,omitempty"`
	Secrets         []string          `json:"secrets,omitempty"` // Secret names, not values
	DeploymentURLs  map[EdgeRegion]string `json:"deployment_urls,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	LastDeployedAt  *time.Time        `json:"last_deployed_at,omitempty"`
	LastInvokedAt   *time.Time        `json:"last_invoked_at,omitempty"`
	InvocationCount int64             `json:"invocation_count"`
	ErrorCount      int64             `json:"error_count"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// FunctionConfig contains edge function configuration
type FunctionConfig struct {
	MemoryMB        int           `json:"memory_mb,omitempty"`
	TimeoutMs       int           `json:"timeout_ms"`
	MaxConcurrency  int           `json:"max_concurrency,omitempty"`
	CPUTimeMs       int           `json:"cpu_time_ms,omitempty"`
	EnableCaching   bool          `json:"enable_caching"`
	CacheTTLSec     int           `json:"cache_ttl_sec,omitempty"`
	WarmInstances   int           `json:"warm_instances,omitempty"`
	EnableLogging   bool          `json:"enable_logging"`
	LogLevel        string        `json:"log_level,omitempty"`
	KVBindings      []KVBinding   `json:"kv_bindings,omitempty"`
	DurableObjects  []DOBinding   `json:"durable_objects,omitempty"`
}

// KVBinding represents a key-value store binding
type KVBinding struct {
	Name        string `json:"name"`
	NamespaceID string `json:"namespace_id"`
}

// DOBinding represents a durable object binding
type DOBinding struct {
	Name      string `json:"name"`
	ClassName string `json:"class_name"`
}

// FunctionTrigger defines what triggers the edge function
type FunctionTrigger struct {
	Type        TriggerType `json:"type"`
	EndpointID  string      `json:"endpoint_id,omitempty"`
	Route       string      `json:"route,omitempty"`
	EventType   string      `json:"event_type,omitempty"`
	Conditions  []Condition `json:"conditions,omitempty"`
}

// TriggerType for edge functions
type TriggerType string

const (
	TriggerTypeWebhookInbound  TriggerType = "webhook_inbound"
	TriggerTypeWebhookOutbound TriggerType = "webhook_outbound"
	TriggerTypeHTTPRoute       TriggerType = "http_route"
	TriggerTypeSchedule        TriggerType = "schedule"
	TriggerTypeEvent           TriggerType = "event"
)

// Condition for trigger evaluation
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// FunctionDeployment represents a deployment record
type FunctionDeployment struct {
	ID           string         `json:"id"`
	FunctionID   string         `json:"function_id"`
	TenantID     string         `json:"tenant_id"`
	Version      int            `json:"version"`
	Status       FunctionStatus `json:"status"`
	Runtime      RuntimeType    `json:"runtime"`
	Regions      []EdgeRegion   `json:"regions"`
	DeploymentURLs map[EdgeRegion]string `json:"deployment_urls,omitempty"`
	BuildLogs    string         `json:"build_logs,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	Duration     int64          `json:"duration_ms"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}

// FunctionInvocation represents a function execution
type FunctionInvocation struct {
	ID            string          `json:"id"`
	FunctionID    string          `json:"function_id"`
	TenantID      string          `json:"tenant_id"`
	Region        EdgeRegion      `json:"region"`
	TriggerType   TriggerType     `json:"trigger_type"`
	Input         json.RawMessage `json:"input,omitempty"`
	Output        json.RawMessage `json:"output,omitempty"`
	Status        string          `json:"status"` // success, error
	ErrorMessage  string          `json:"error_message,omitempty"`
	DurationMs    int64           `json:"duration_ms"`
	CPUTimeMs     int64           `json:"cpu_time_ms"`
	MemoryUsedMB  float64         `json:"memory_used_mb"`
	CacheHit      bool            `json:"cache_hit"`
	InvokedAt     time.Time       `json:"invoked_at"`
}

// FunctionMetrics contains aggregated metrics
type FunctionMetrics struct {
	FunctionID       string            `json:"function_id"`
	TenantID         string            `json:"tenant_id"`
	Period           string            `json:"period"` // hourly, daily
	TotalInvocations int64             `json:"total_invocations"`
	SuccessCount     int64             `json:"success_count"`
	ErrorCount       int64             `json:"error_count"`
	AvgDurationMs    float64           `json:"avg_duration_ms"`
	P50DurationMs    float64           `json:"p50_duration_ms"`
	P95DurationMs    float64           `json:"p95_duration_ms"`
	P99DurationMs    float64           `json:"p99_duration_ms"`
	AvgMemoryMB      float64           `json:"avg_memory_mb"`
	CacheHitRate     float64           `json:"cache_hit_rate"`
	ByRegion         map[EdgeRegion]RegionMetrics `json:"by_region,omitempty"`
	CollectedAt      time.Time         `json:"collected_at"`
}

// RegionMetrics contains per-region metrics
type RegionMetrics struct {
	Invocations   int64   `json:"invocations"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	ErrorRate     float64 `json:"error_rate"`
}

// FunctionTemplate provides reusable function templates
type FunctionTemplate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags,omitempty"`
	Code        string   `json:"code"`
	EntryPoint  string   `json:"entry_point"`
	Runtime     RuntimeType `json:"runtime"`
	Config      *FunctionConfig `json:"config"`
}

// CreateFunctionRequest represents a request to create an edge function
type CreateFunctionRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description,omitempty"`
	Runtime     RuntimeType       `json:"runtime" binding:"required"`
	Code        string            `json:"code" binding:"required"`
	EntryPoint  string            `json:"entry_point"`
	Regions     []EdgeRegion      `json:"regions,omitempty"`
	Config      *FunctionConfig   `json:"config,omitempty"`
	Triggers    []FunctionTrigger `json:"triggers,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	AutoDeploy  bool              `json:"auto_deploy"`
}

// UpdateFunctionRequest represents a request to update an edge function
type UpdateFunctionRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Code        *string           `json:"code,omitempty"`
	EntryPoint  *string           `json:"entry_point,omitempty"`
	Regions     []EdgeRegion      `json:"regions,omitempty"`
	Config      *FunctionConfig   `json:"config,omitempty"`
	Triggers    []FunctionTrigger `json:"triggers,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Status      *FunctionStatus   `json:"status,omitempty"`
}

// InvokeFunctionRequest represents a request to invoke an edge function
type InvokeFunctionRequest struct {
	Input   json.RawMessage `json:"input"`
	Region  EdgeRegion      `json:"region,omitempty"`
	Async   bool            `json:"async,omitempty"`
}

// InvokeFunctionResponse represents the response from function invocation
type InvokeFunctionResponse struct {
	InvocationID string          `json:"invocation_id"`
	Output       json.RawMessage `json:"output,omitempty"`
	Status       string          `json:"status"`
	DurationMs   int64           `json:"duration_ms"`
	Region       EdgeRegion      `json:"region"`
	CacheHit     bool            `json:"cache_hit"`
}

// FunctionFilters for listing functions
type FunctionFilters struct {
	Runtime  *RuntimeType    `json:"runtime,omitempty"`
	Status   *FunctionStatus `json:"status,omitempty"`
	Region   *EdgeRegion     `json:"region,omitempty"`
	Search   string          `json:"search,omitempty"`
	Page     int             `json:"page,omitempty"`
	PageSize int             `json:"page_size,omitempty"`
}

// ListFunctionsResponse represents paginated function list
type ListFunctionsResponse struct {
	Functions  []EdgeFunction `json:"functions"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// DefaultFunctionConfig returns default configuration
func DefaultFunctionConfig() *FunctionConfig {
	return &FunctionConfig{
		MemoryMB:       128,
		TimeoutMs:      10000,
		MaxConcurrency: 100,
		CPUTimeMs:      50,
		EnableCaching:  true,
		CacheTTLSec:    60,
		EnableLogging:  true,
		LogLevel:       "info",
	}
}
