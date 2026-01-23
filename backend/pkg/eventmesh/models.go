package eventmesh

import "time"

// Route defines a declarative routing rule for event fan-out
type Route struct {
	ID          string       `json:"id" db:"id"`
	TenantID    string       `json:"tenant_id" db:"tenant_id"`
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description,omitempty" db:"description"`
	SourceFilter *Filter     `json:"source_filter,omitempty"`
	FilterJSON  string       `json:"-" db:"source_filter"`
	Targets     []RouteTarget `json:"targets"`
	TargetsJSON string       `json:"-" db:"targets"`
	Priority    int          `json:"priority" db:"priority"`
	IsActive    bool         `json:"is_active" db:"is_active"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

// Filter defines content-based filtering rules
type Filter struct {
	EventTypes []string          `json:"event_types,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	JSONPath   []JSONPathRule    `json:"jsonpath,omitempty"`
}

// JSONPathRule defines a JSONPath-based filter condition
type JSONPathRule struct {
	Path     string `json:"path"`
	Operator string `json:"operator"` // eq, neq, contains, gt, lt, exists
	Value    string `json:"value,omitempty"`
}

// RouteTarget defines a destination for routed events
type RouteTarget struct {
	EndpointID  string `json:"endpoint_id"`
	Weight      int    `json:"weight,omitempty"`
	Transform   string `json:"transform,omitempty"`
	Condition   string `json:"condition,omitempty"`
}

// DeadLetterConfig configures the dead-letter queue for a route
type DeadLetterConfig struct {
	ID            string    `json:"id" db:"id"`
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	RouteID       string    `json:"route_id" db:"route_id"`
	MaxRetries    int       `json:"max_retries" db:"max_retries"`
	RetentionDays int       `json:"retention_days" db:"retention_days"`
	AlertOnEntry  bool      `json:"alert_on_entry" db:"alert_on_entry"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// DeadLetterEntry represents an event that failed all delivery targets
type DeadLetterEntry struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	RouteID    string    `json:"route_id" db:"route_id"`
	Payload    string    `json:"payload" db:"payload"`
	Reason     string    `json:"reason" db:"reason"`
	AttemptCount int     `json:"attempt_count" db:"attempt_count"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
}

// RouteExecution records a single route execution
type RouteExecution struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	RouteID      string    `json:"route_id" db:"route_id"`
	SourceEvent  string    `json:"source_event" db:"source_event"`
	TargetsHit   int       `json:"targets_hit" db:"targets_hit"`
	TargetsFailed int      `json:"targets_failed" db:"targets_failed"`
	DurationMs   int       `json:"duration_ms" db:"duration_ms"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// CreateRouteRequest is the request DTO for creating a route
type CreateRouteRequest struct {
	Name         string         `json:"name" binding:"required,min=1,max=255"`
	Description  string         `json:"description,omitempty"`
	SourceFilter *Filter        `json:"source_filter,omitempty"`
	Targets      []RouteTarget  `json:"targets" binding:"required,min=1"`
	Priority     int            `json:"priority"`
}

// RouteEventRequest is the request DTO for routing an event
type RouteEventRequest struct {
	EventType string            `json:"event_type" binding:"required"`
	Payload   string            `json:"payload" binding:"required"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// ConfigureDeadLetterRequest is the request DTO for configuring dead letter
type ConfigureDeadLetterRequest struct {
	MaxRetries    int  `json:"max_retries" binding:"min=0"`
	RetentionDays int  `json:"retention_days" binding:"min=1"`
	AlertOnEntry  bool `json:"alert_on_entry"`
}

// RouteStats provides routing statistics
type RouteStats struct {
	TotalRoutes      int     `json:"total_routes"`
	ActiveRoutes     int     `json:"active_routes"`
	TotalExecutions  int     `json:"total_executions"`
	SuccessRate      float64 `json:"success_rate_pct"`
	AvgFanOutFactor  float64 `json:"avg_fan_out_factor"`
	DeadLetterCount  int     `json:"dead_letter_count"`
}
