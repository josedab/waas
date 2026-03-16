// Delivery Engine entry point for the WaaS platform.
//
// The delivery engine consumes webhook delivery jobs from the queue,
// dispatches HTTP requests to configured endpoints, and manages retry
// logic with exponential backoff. It runs as a standalone worker process
// alongside the API service and analytics service.
//
// Usage:
//
//	go run ./cmd/delivery-engine
//	# or via Makefile:
//	make run-delivery
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/josedab/waas/internal/delivery"
	"github.com/josedab/waas/pkg/utils"
)

var logger = utils.NewLogger("delivery-engine")

func main() {
	logger.Info("Starting Webhook Delivery Engine...", nil)

	engine, err := delivery.NewEngine()
	if err != nil {
		logger.Error("Failed to initialize delivery engine", map[string]interface{}{"error": err.Error()})
		logStartupHint(err)
		os.Exit(1)
	}
	if err := engine.Start(); err != nil {
		logger.Error("Failed to start delivery engine", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}

	// Start a minimal health check HTTP server for Kubernetes probes
	healthPort := os.Getenv("DELIVERY_HEALTH_PORT")
	if healthPort == "" {
		healthPort = "8081"
	}
	startTime := time.Now()
	// draining is set to true after a shutdown signal is received, causing
	// the /ready endpoint to return 503 so load balancers stop sending traffic.
	draining := false

	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"service":   "delivery-engine",
			"timestamp": time.Now(),
			"uptime":    time.Since(startTime).String(),
		})
	})
	healthMux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})
	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if draining {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "draining"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	healthSrv := &http.Server{Addr: ":" + healthPort, Handler: healthMux}
	go func() {
		logger.Info("Health endpoint listening", map[string]interface{}{"port": healthPort})
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health server failed", map[string]interface{}{"error": err.Error()})
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Received signal, shutting down...", map[string]interface{}{"signal": sig.String()})

	// Mark as draining so /ready returns 503 and load balancers stop routing.
	draining = true

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	healthSrv.Shutdown(ctx)

	engine.Stop()
	logger.Info("Delivery engine stopped", nil)
}

// logStartupHint inspects an initialization error and logs actionable guidance.
func logStartupHint(err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "database connection"), strings.Contains(msg, "database"):
		logger.Error("Hint: ensure PostgreSQL is running and DATABASE_URL is set. Try: make docker-up", nil)
	case strings.Contains(msg, "redis"):
		logger.Error("Hint: ensure Redis is running and REDIS_URL is set. Try: make docker-up", nil)
	case strings.Contains(msg, "configuration error"):
		logger.Error("Hint: check your .env file. Try: make ensure-env && make validate-env", nil)
	}
}
