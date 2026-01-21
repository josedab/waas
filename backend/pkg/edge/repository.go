package edge

import (
	"context"
	"errors"
	"time"
)

var (
	ErrFunctionNotFound    = errors.New("edge function not found")
	ErrNotFound            = errors.New("edge function not found") // Alias
	ErrDeploymentFailed    = errors.New("function deployment failed")
	ErrInvocationFailed    = errors.New("function invocation failed")
	ErrInvalidCode         = errors.New("invalid function code")
	ErrRuntimeNotSupported = errors.New("runtime not supported")
	ErrRegionNotAvailable  = errors.New("region not available")
)

// Repository defines the interface for edge function storage
type Repository interface {
	// Functions
	CreateFunction(ctx context.Context, fn *EdgeFunction) error
	GetFunction(ctx context.Context, tenantID, functionID string) (*EdgeFunction, error)
	GetFunctionByName(ctx context.Context, tenantID, name string) (*EdgeFunction, error)
	UpdateFunction(ctx context.Context, fn *EdgeFunction) error
	DeleteFunction(ctx context.Context, tenantID, functionID string) error
	ListFunctions(ctx context.Context, tenantID string, filters *FunctionFilters) ([]EdgeFunction, int, error)

	// Deployments
	CreateDeployment(ctx context.Context, deployment *FunctionDeployment) error
	GetDeployment(ctx context.Context, deploymentID string) (*FunctionDeployment, error)
	UpdateDeployment(ctx context.Context, deployment *FunctionDeployment) error
	ListDeployments(ctx context.Context, functionID string, limit int) ([]FunctionDeployment, error)

	// Invocations
	CreateInvocation(ctx context.Context, invocation *FunctionInvocation) error
	GetInvocation(ctx context.Context, invocationID string) (*FunctionInvocation, error)
	ListInvocations(ctx context.Context, functionID string, filters *InvocationFilters) ([]FunctionInvocation, int, error)

	// Metrics
	SaveMetrics(ctx context.Context, metrics *FunctionMetrics) error
	GetMetrics(ctx context.Context, functionID, period string) (*FunctionMetrics, error)
	IncrementCounters(ctx context.Context, functionID string, invocations, errors int64) error

	// Secrets
	SaveSecret(ctx context.Context, tenantID, name, value string) error
	GetSecret(ctx context.Context, tenantID, name string) (string, error)
	DeleteSecret(ctx context.Context, tenantID, name string) error
}

// InvocationFilters for listing invocations
type InvocationFilters struct {
	Status    string     `json:"status,omitempty"`
	Region    *EdgeRegion `json:"region,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Page      int        `json:"page,omitempty"`
	PageSize  int        `json:"page_size,omitempty"`
}

// RuntimeProvider defines the interface for edge runtime platforms
type RuntimeProvider interface {
	// GetType returns the runtime type
	GetType() RuntimeType

	// Deploy deploys a function to the edge
	Deploy(ctx context.Context, fn *EdgeFunction) (*FunctionDeployment, error)

	// Undeploy removes a function from the edge
	Undeploy(ctx context.Context, fn *EdgeFunction) error

	// Invoke executes the function
	Invoke(ctx context.Context, fn *EdgeFunction, input []byte, region EdgeRegion) (*InvokeFunctionResponse, error)

	// GetLogs retrieves function logs
	GetLogs(ctx context.Context, fn *EdgeFunction, since time.Time, limit int) ([]LogEntry, error)

	// ValidateCode validates function code
	ValidateCode(ctx context.Context, code string) error

	// GetSupportedRegions returns supported regions
	GetSupportedRegions() []EdgeRegion

	// HealthCheck checks if the provider is healthy
	HealthCheck(ctx context.Context) error
}

// LogEntry represents a function log entry
type LogEntry struct {
	Timestamp time.Time       `json:"timestamp"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	Region    EdgeRegion      `json:"region,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// CodeCompiler defines the interface for code compilation
type CodeCompiler interface {
	// Compile compiles source code for deployment
	Compile(ctx context.Context, code string, runtime RuntimeType) (string, error)

	// Bundle bundles code with dependencies
	Bundle(ctx context.Context, code string, dependencies []string) (string, error)

	// Minify minifies code for deployment
	Minify(ctx context.Context, code string) (string, error)
}

// EdgeRouter defines the interface for routing requests to edge functions
type EdgeRouter interface {
	// Route routes a request to the appropriate edge function
	Route(ctx context.Context, tenantID string, request *RouteRequest) (*EdgeFunction, EdgeRegion, error)

	// GetNearestRegion returns the nearest region for a client
	GetNearestRegion(ctx context.Context, clientIP string) (EdgeRegion, error)

	// UpdateRoutes updates routing configuration
	UpdateRoutes(ctx context.Context, tenantID string) error
}

// RouteRequest represents a routing request
type RouteRequest struct {
	Path      string            `json:"path"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ClientIP  string            `json:"client_ip"`
	GeoData   *GeoData          `json:"geo_data,omitempty"`
}

// GeoData contains geographic information
type GeoData struct {
	Country     string  `json:"country"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
}
