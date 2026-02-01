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
	"github.com/josedab/waas/internal/api"
	_ "github.com/josedab/waas/docs"
	
	// Import feature packages for swagger doc generation
	_ "github.com/josedab/waas/pkg/costing"
	_ "github.com/josedab/waas/pkg/embed"
	_ "github.com/josedab/waas/pkg/flow"
	_ "github.com/josedab/waas/pkg/georouting"
	_ "github.com/josedab/waas/pkg/metaevents"
	_ "github.com/josedab/waas/pkg/mocking"
	_ "github.com/josedab/waas/pkg/otel"
	_ "github.com/josedab/waas/pkg/protocols"
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