// Analytics Service entry point for the WaaS platform.
//
// The analytics service collects delivery metrics, aggregates statistics,
// and exposes a dashboard API on port 8082. It also provides real-time
// monitoring via WebSocket and runs background workers for periodic
// metric computation.
//
// Usage:
//
//	go run ./cmd/analytics-service
//	# or via Makefile:
//	make run-analytics
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"github.com/josedab/waas/internal/analytics"
	"github.com/josedab/waas/pkg/utils"
)

var logger = utils.NewLogger("analytics-service")

func main() {
	logger.Info("Starting Analytics Service...", nil)

	service, err := analytics.NewService()
	if err != nil {
		logger.Error("Failed to initialize analytics service", map[string]interface{}{"error": err.Error()})
		logStartupHint(err)
		os.Exit(1)
	}

	port := os.Getenv("ANALYTICS_PORT")
	if port == "" {
		port = "8082"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: service.Handler(),
	}

	// Start background workers
	service.StartWorkers()

	// Start server in a goroutine
	go func() {
		logger.Info("Analytics service listening", map[string]interface{}{"addr": srv.Addr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("Received signal, shutting down gracefully...", map[string]interface{}{"signal": sig.String()})

	// Give outstanding requests 15 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", map[string]interface{}{"error": err.Error()})
	}

	service.Stop()
	logger.Info("Analytics service stopped", nil)
}

// logStartupHint inspects an initialization error and logs actionable guidance.
func logStartupHint(err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "database connection"):
		logger.Error("Hint: ensure PostgreSQL is running and DATABASE_URL is set. Try: make docker-up", nil)
	case strings.Contains(msg, "redis"):
		logger.Error("Hint: ensure Redis is running and REDIS_URL is set. Try: make docker-up", nil)
	case strings.Contains(msg, "configuration error"):
		logger.Error("Hint: check your .env file. Try: make ensure-env && make validate-env", nil)
	}
}