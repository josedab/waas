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
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    ":8082",
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