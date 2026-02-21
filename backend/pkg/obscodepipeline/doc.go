// Package obscodepipeline provides an observability-as-code pipeline that allows
// webhook delivery metrics, traces, and logs to be defined and managed as code.
// Tenants define observability configurations declaratively, and the system
// automatically instruments webhooks, collects telemetry, and routes data to
// configured backends (Prometheus, Datadog, OpenTelemetry collectors, etc.).
package obscodepipeline
