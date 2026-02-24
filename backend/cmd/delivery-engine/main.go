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
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down delivery engine...", nil)
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