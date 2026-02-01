package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/josedab/waas/internal/analytics"
)

func main() {
	log.Println("Starting Analytics Service...")

	service, err := analytics.NewService()
	if err != nil {
		log.Fatal("Failed to initialize analytics service: ", err)
	}

	srv := &http.Server{
		Addr:    ":8082",
		Handler: service.Handler(),
	}

	// Start background workers
	service.StartWorkers()

	// Start server in a goroutine
	go func() {
		log.Printf("Analytics service listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %s, shutting down gracefully...", sig)

	// Give outstanding requests 15 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	service.Stop()
	log.Println("Analytics service stopped")
}