package otel

import "encoding/json"

// GrafanaDashboard represents a pre-built Grafana dashboard configuration
type GrafanaDashboard struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	UID         string         `json:"uid"`
	Tags        []string       `json:"tags"`
	Panels      []GrafanaPanel `json:"panels"`
}

// GrafanaPanel represents a dashboard panel
type GrafanaPanel struct {
	Title       string `json:"title"`
	Type        string `json:"type"` // graph, stat, gauge, table, heatmap
	Description string `json:"description,omitempty"`
	GridPos     struct {
		H int `json:"h"`
		W int `json:"w"`
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"gridPos"`
	Targets []GrafanaTarget `json:"targets"`
}

// GrafanaTarget represents a panel data source query
type GrafanaTarget struct {
	Expr      string `json:"expr"`
	LegendFmt string `json:"legendFormat,omitempty"`
	Interval  string `json:"interval,omitempty"`
}

// GetDeliveryDashboard returns the pre-built delivery metrics Grafana dashboard
func GetDeliveryDashboard() *GrafanaDashboard {
	return &GrafanaDashboard{
		Title:       "WaaS - Delivery Metrics",
		Description: "Webhook delivery latency, error rates, queue depth, and tenant breakdown",
		UID:         "waas-delivery",
		Tags:        []string{"waas", "delivery", "webhooks"},
		Panels: []GrafanaPanel{
			{
				Title: "Delivery Rate",
				Type:  "graph",
				Targets: []GrafanaTarget{
					{Expr: `rate(waas_deliveries_total[5m])`, LegendFmt: "{{status}}"},
				},
			},
			{
				Title: "Delivery Latency (P99)",
				Type:  "graph",
				Targets: []GrafanaTarget{
					{Expr: `histogram_quantile(0.99, rate(waas_delivery_duration_seconds_bucket[5m]))`, LegendFmt: "p99"},
					{Expr: `histogram_quantile(0.95, rate(waas_delivery_duration_seconds_bucket[5m]))`, LegendFmt: "p95"},
					{Expr: `histogram_quantile(0.50, rate(waas_delivery_duration_seconds_bucket[5m]))`, LegendFmt: "p50"},
				},
			},
			{
				Title: "Error Rate",
				Type:  "stat",
				Targets: []GrafanaTarget{
					{Expr: `rate(waas_deliveries_total{status="failed"}[5m]) / rate(waas_deliveries_total[5m]) * 100`, LegendFmt: "Error %"},
				},
			},
			{
				Title: "Queue Depth",
				Type:  "graph",
				Targets: []GrafanaTarget{
					{Expr: `waas_queue_depth{queue="delivery"}`, LegendFmt: "Delivery"},
					{Expr: `waas_queue_depth{queue="retry"}`, LegendFmt: "Retry"},
					{Expr: `waas_queue_depth{queue="dlq"}`, LegendFmt: "DLQ"},
				},
			},
			{
				Title: "Deliveries by Tenant",
				Type:  "table",
				Targets: []GrafanaTarget{
					{Expr: `topk(10, sum by (tenant_id) (rate(waas_deliveries_total[1h])))`, LegendFmt: "{{tenant_id}}"},
				},
			},
			{
				Title: "Active Endpoints",
				Type:  "stat",
				Targets: []GrafanaTarget{
					{Expr: `waas_active_endpoints`, LegendFmt: "Endpoints"},
				},
			},
			{
				Title: "Retry Rate",
				Type:  "graph",
				Targets: []GrafanaTarget{
					{Expr: `rate(waas_retries_total[5m])`, LegendFmt: "Retries/s"},
				},
			},
			{
				Title: "HTTP Status Distribution",
				Type:  "graph",
				Targets: []GrafanaTarget{
					{Expr: `sum by (http_status) (rate(waas_delivery_http_status_total[5m]))`, LegendFmt: "{{http_status}}"},
				},
			},
		},
	}
}

// GetDeliveryDashboardJSON returns the dashboard as a JSON string for Grafana import
func GetDeliveryDashboardJSON() (string, error) {
	dashboard := GetDeliveryDashboard()
	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
