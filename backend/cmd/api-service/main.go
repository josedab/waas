// Webhook Service Platform API
//
// A comprehensive webhook-as-a-service platform that enables companies to reliably send, receive, and manage webhooks without building their own infrastructure.
//
//	@title			Webhook Service Platform API
//	@version		1.0.0
//	@description	A comprehensive webhook-as-a-service platform that enables companies to reliably send, receive, and manage webhooks without building their own infrastructure.
//	@termsOfService	http://swagger.io/terms/
//
//	@contact.name	Webhook Platform Team
//	@contact.url	http://www.webhook-platform.com
//	@contact.email	support@webhook-platform.com
//
//	@license.name	MIT
//	@license.url	http://opensource.org/licenses/MIT
//
//	@host		localhost:8080
//	@BasePath	/api/v1
//	@schemes	http https
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				API key for authentication. Get your API key by creating a tenant account.
package main

import (
	"log"
	"webhook-platform/internal/api"
	_ "webhook-platform/docs"
	
	// Import feature packages for swagger doc generation
	_ "webhook-platform/pkg/costing"
	_ "webhook-platform/pkg/embed"
	_ "webhook-platform/pkg/flow"
	_ "webhook-platform/pkg/georouting"
	_ "webhook-platform/pkg/metaevents"
	_ "webhook-platform/pkg/mocking"
	_ "webhook-platform/pkg/otel"
	_ "webhook-platform/pkg/protocols"
)

func main() {
	log.Println("Starting Webhook API Service...")
	
	server, err := api.NewServer()
	if err != nil {
		log.Fatal("Failed to initialize API service: ", err)
	}
	if err := server.Start(":8080"); err != nil {
		log.Fatal("Failed to start API service:", err)
	}
}