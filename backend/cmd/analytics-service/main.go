package main

import (
	"log"
	"webhook-platform/internal/analytics"
)

func main() {
	log.Println("Starting Analytics Service...")
	
	service, err := analytics.NewService()
	if err != nil {
		log.Fatal("Failed to initialize analytics service: ", err)
	}
	if err := service.Start(":8082"); err != nil {
		log.Fatal("Failed to start analytics service:", err)
	}
}