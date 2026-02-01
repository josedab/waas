package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/josedab/waas/internal/delivery"
)

func main() {
	log.Println("Starting Webhook Delivery Engine...")
	
	engine, err := delivery.NewEngine()
	if err != nil {
		log.Fatal("Failed to initialize delivery engine: ", err)
	}
	if err := engine.Start(); err != nil {
		log.Fatal("Failed to start delivery engine:", err)
	}

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down delivery engine...")
	engine.Stop()
	log.Println("Delivery engine stopped")
}