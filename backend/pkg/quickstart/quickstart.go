// Package quickstart provides an embedded evaluation mode for WaaS.
// It enables zero-config startup with SQLite fallback and in-memory queue
// for quick evaluation without external dependencies.
package quickstart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/josedab/waas/pkg/queue"
)

// Mode represents the quickstart runtime mode
type Mode string

const (
	ModeQuickstart Mode = "quickstart"
	ModeStandard   Mode = "standard"
)

// Config holds quickstart configuration
type Config struct {
	Mode           Mode   `json:"mode"`
	DataDir        string `json:"data_dir"`
	EnableTutorial bool   `json:"enable_tutorial"`
	EmbeddedDB     bool   `json:"embedded_db"`
	InMemoryQueue  bool   `json:"in_memory_queue"`
	Port           int    `json:"port"`
	AutoSeed       bool   `json:"auto_seed"`
}

// DefaultConfig returns the default quickstart configuration
func DefaultConfig() *Config {
	return &Config{
		Mode:           ModeQuickstart,
		DataDir:        "/tmp/waas-quickstart",
		EnableTutorial: true,
		EmbeddedDB:     true,
		InMemoryQueue:  true,
		Port:           8080,
		AutoSeed:       true,
	}
}

// MemoryQueue implements queue.PublisherInterface with in-memory storage
// for quickstart mode when Redis is not available.
type MemoryQueue struct {
	mu       sync.RWMutex
	delivery []queue.DeliveryMessage
	retry    []queue.DeliveryMessage
	dlq      []queue.DeliveryMessage
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		delivery: make([]queue.DeliveryMessage, 0, 100),
		retry:    make([]queue.DeliveryMessage, 0, 100),
		dlq:      make([]queue.DeliveryMessage, 0, 100),
	}
}

// PublishDelivery adds a message to the in-memory delivery queue
func (q *MemoryQueue) PublishDelivery(ctx context.Context, message *queue.DeliveryMessage) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.delivery = append(q.delivery, *message)
	return nil
}

// PublishDelayedDelivery adds a message to the in-memory retry queue
func (q *MemoryQueue) PublishDelayedDelivery(ctx context.Context, message *queue.DeliveryMessage, delay time.Duration) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.retry = append(q.retry, *message)
	return nil
}

// PublishToDeadLetter adds a message to the in-memory dead letter queue
func (q *MemoryQueue) PublishToDeadLetter(ctx context.Context, message *queue.DeliveryMessage, reason string) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.dlq = append(q.dlq, *message)
	return nil
}

// GetQueueLength returns the length of a named queue
func (q *MemoryQueue) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	switch queueName {
	case "delivery":
		return int64(len(q.delivery)), nil
	case "retry":
		return int64(len(q.retry)), nil
	case "dlq":
		return int64(len(q.dlq)), nil
	default:
		return 0, nil
	}
}

// GetQueueStats returns statistics for all queues
func (q *MemoryQueue) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return map[string]int64{
		"delivery": int64(len(q.delivery)),
		"retry":    int64(len(q.retry)),
		"dlq":      int64(len(q.dlq)),
	}, nil
}

// Dequeue removes and returns the next message from the delivery queue
func (q *MemoryQueue) Dequeue(ctx context.Context) (*queue.DeliveryMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.delivery) == 0 {
		return nil, nil
	}
	msg := q.delivery[0]
	q.delivery = q.delivery[1:]
	return &msg, nil
}

// TutorialStep represents a step in the interactive tutorial
type TutorialStep struct {
	Step        int    `json:"step"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Method      string `json:"method,omitempty"`
	Body        string `json:"body,omitempty"`
	Expected    string `json:"expected,omitempty"`
}

// GetTutorialSteps returns the interactive getting-started tutorial
func GetTutorialSteps() []TutorialStep {
	return []TutorialStep{
		{
			Step:        1,
			Title:       "Health Check",
			Description: "Verify the WaaS service is running and healthy.",
			Command:     "curl -s http://localhost:8080/health | jq .",
			Endpoint:    "/health",
			Method:      "GET",
			Expected:    `{"status": "healthy"}`,
		},
		{
			Step:        2,
			Title:       "Create a Tenant",
			Description: "Create your first tenant to get an API key for authentication.",
			Command:     `curl -s -X POST http://localhost:8080/api/v1/tenants -H "Content-Type: application/json" -d '{"name": "quickstart-tenant", "email": "quickstart@example.com"}'`,
			Endpoint:    "/api/v1/tenants",
			Method:      "POST",
			Body:        `{"name": "quickstart-tenant", "email": "quickstart@example.com"}`,
			Expected:    `{"tenant": {...}, "api_key": "whk_..."}`,
		},
		{
			Step:        3,
			Title:       "Register a Webhook Endpoint",
			Description: "Register a URL where webhooks will be delivered. Use the API key from step 2.",
			Command:     `curl -s -X POST http://localhost:8080/api/v1/endpoints -H "Content-Type: application/json" -H "X-API-Key: YOUR_API_KEY" -d '{"url": "https://httpbin.org/post", "event_types": ["order.created"]}'`,
			Endpoint:    "/api/v1/endpoints",
			Method:      "POST",
			Body:        `{"url": "https://httpbin.org/post", "event_types": ["order.created"]}`,
			Expected:    `{"endpoint": {"id": "...", "url": "https://httpbin.org/post"}}`,
		},
		{
			Step:        4,
			Title:       "Send a Webhook",
			Description: "Send your first webhook event. WaaS will deliver it with retries and signing.",
			Command:     `curl -s -X POST http://localhost:8080/api/v1/webhooks/send -H "Content-Type: application/json" -H "X-API-Key: YOUR_API_KEY" -d '{"event_type": "order.created", "payload": {"order_id": "123", "amount": 99.99}}'`,
			Endpoint:    "/api/v1/webhooks/send",
			Method:      "POST",
			Body:        `{"event_type": "order.created", "payload": {"order_id": "123", "amount": 99.99}}`,
			Expected:    `{"delivery_id": "...", "status": "queued"}`,
		},
		{
			Step:        5,
			Title:       "Check Delivery Status",
			Description: "Monitor the delivery status of your webhook.",
			Command:     `curl -s http://localhost:8080/api/v1/webhooks/deliveries -H "X-API-Key: YOUR_API_KEY"`,
			Endpoint:    "/api/v1/webhooks/deliveries",
			Method:      "GET",
			Expected:    `{"deliveries": [...]}`,
		},
		{
			Step:        6,
			Title:       "View Analytics",
			Description: "Check the analytics dashboard for delivery metrics.",
			Command:     `curl -s http://localhost:8080/api/v1/analytics/overview -H "X-API-Key: YOUR_API_KEY"`,
			Endpoint:    "/api/v1/analytics/overview",
			Method:      "GET",
			Expected:    `{"total_deliveries": ..., "success_rate": ...}`,
		},
		{
			Step:        7,
			Title:       "Explore the API",
			Description: "Browse the full interactive API documentation.",
			Command:     "open http://localhost:8080/docs/",
			Endpoint:    "/docs/",
			Method:      "GET",
			Expected:    "Swagger UI with all API endpoints",
		},
	}
}

// GettingStartedHandler serves the interactive tutorial as JSON API
func GettingStartedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		steps := GetTutorialSteps()
		response := map[string]interface{}{
			"title":       "WaaS Getting Started Guide",
			"description": "Interactive tutorial to get up and running with WaaS in minutes",
			"mode":        "quickstart",
			"version":     "1.0.0",
			"steps":       steps,
			"next_steps": []string{
				"Read the full documentation at /docs/",
				"Set up a production deployment with PostgreSQL and Redis",
				"Configure webhook transformations and filters",
				"Set up real-time monitoring and alerting",
			},
			"links": map[string]string{
				"documentation": "/docs/",
				"health":        "/health",
				"api_base":      "/api/v1",
				"github":        "https://github.com/josedab/waas",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// GettingStartedHTMLHandler serves an HTML version of the tutorial
func GettingStartedHTMLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, gettingStartedHTML)
	}
}

const gettingStartedHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WaaS - Getting Started</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #e2e8f0; line-height: 1.6; }
        .container { max-width: 800px; margin: 0 auto; padding: 2rem; }
        h1 { font-size: 2.5rem; background: linear-gradient(135deg, #3b82f6, #8b5cf6); -webkit-background-clip: text; -webkit-text-fill-color: transparent; margin-bottom: 0.5rem; }
        .subtitle { color: #94a3b8; font-size: 1.1rem; margin-bottom: 2rem; }
        .step { background: #1e293b; border-radius: 12px; padding: 1.5rem; margin-bottom: 1rem; border: 1px solid #334155; }
        .step-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 0.75rem; }
        .step-number { background: #3b82f6; color: white; width: 32px; height: 32px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: bold; font-size: 0.875rem; }
        .step-title { font-size: 1.1rem; font-weight: 600; }
        .step-desc { color: #94a3b8; margin-bottom: 0.75rem; }
        .command { background: #0f172a; border: 1px solid #334155; border-radius: 8px; padding: 0.75rem 1rem; font-family: 'Fira Code', monospace; font-size: 0.85rem; overflow-x: auto; }
        .command:hover { border-color: #3b82f6; }
        .badge { display: inline-block; padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 600; margin-bottom: 1.5rem; }
        .badge-quickstart { background: #059669; color: white; }
        .links { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-top: 2rem; }
        .link-card { background: #1e293b; border: 1px solid #334155; border-radius: 8px; padding: 1rem; text-decoration: none; color: #e2e8f0; transition: border-color 0.2s; }
        .link-card:hover { border-color: #3b82f6; }
        .link-card h3 { font-size: 0.9rem; margin-bottom: 0.25rem; }
        .link-card p { color: #64748b; font-size: 0.8rem; }
    </style>
</head>
<body>
    <div class="container">
        <span class="badge badge-quickstart">⚡ Quickstart Mode</span>
        <h1>WaaS</h1>
        <p class="subtitle">Webhook as a Service — Get started in under 30 seconds</p>
        <div id="steps"></div>
        <div class="links">
            <a href="/docs/" class="link-card"><h3>📖 API Docs</h3><p>Interactive Swagger documentation</p></a>
            <a href="/health" class="link-card"><h3>💚 Health Check</h3><p>Service status and components</p></a>
            <a href="/api/v1" class="link-card"><h3>🔌 API Base</h3><p>REST API v1 endpoint</p></a>
            <a href="https://github.com/josedab/waas" class="link-card"><h3>⭐ GitHub</h3><p>Source code and issues</p></a>
        </div>
    </div>
    <script>
        const steps = [
            {step: 1, title: "Health Check", desc: "Verify WaaS is running", cmd: "curl -s http://localhost:8080/health | jq ."},
            {step: 2, title: "Create a Tenant", desc: "Get your API key", cmd: "curl -s -X POST http://localhost:8080/api/v1/tenants -H 'Content-Type: application/json' -d '{\"name\": \"my-app\", \"email\": \"dev@example.com\"}'"},
            {step: 3, title: "Register Endpoint", desc: "Add a webhook delivery target", cmd: "curl -s -X POST http://localhost:8080/api/v1/endpoints -H 'X-API-Key: YOUR_KEY' -H 'Content-Type: application/json' -d '{\"url\": \"https://httpbin.org/post\"}'"},
            {step: 4, title: "Send Webhook", desc: "Deliver your first event", cmd: "curl -s -X POST http://localhost:8080/api/v1/webhooks/send -H 'X-API-Key: YOUR_KEY' -H 'Content-Type: application/json' -d '{\"event_type\": \"order.created\", \"payload\": {\"id\": \"123\"}}'"},
            {step: 5, title: "Check Status", desc: "Monitor delivery progress", cmd: "curl -s http://localhost:8080/api/v1/webhooks/deliveries -H 'X-API-Key: YOUR_KEY'"},
        ];
        const container = document.getElementById('steps');
        steps.forEach(s => {
            container.innerHTML += '<div class="step"><div class="step-header"><div class="step-number">'+s.step+'</div><div class="step-title">'+s.title+'</div></div><p class="step-desc">'+s.desc+'</p><div class="command">'+s.cmd+'</div></div>';
        });
    </script>
</body>
</html>`
