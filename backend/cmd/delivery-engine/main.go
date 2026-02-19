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