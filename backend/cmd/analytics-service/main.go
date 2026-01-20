package main

import (
	"log"
	"webhook-platform/internal/analytics"
)

func main() {
	log.Println("Starting Analytics Service...")
	
	service := analytics.NewService()
	if err := service.Start(":8082"); err != nil {
		log.Fatal("Failed to start analytics service:", err)
	}
}