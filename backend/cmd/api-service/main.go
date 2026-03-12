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
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/josedab/waas/docs"
	"github.com/josedab/waas/internal/api"
	"github.com/josedab/waas/pkg/utils"

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

var logger = utils.NewLogger("api-service")

func main() {
	logger.Info("Starting Webhook API Service...", nil)

	server, err := api.NewServer()
	if err != nil {
		logger.Error("Failed to initialize API service", map[string]interface{}{"error": err.Error()})
		logStartupHint(err)
		os.Exit(1)
	}
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	srv := &http.Server{
		Addr:    addr,
		Handler: server.Router(),
	}

	// Start HTTP server in a goroutine so we can listen for shutdown signals
	go func() {
		logger.Info("API server listening", map[string]interface{}{"address": addr})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start API service", map[string]interface{}{"error": err.Error()})
			os.Exit(1)
		}
	}()

	// Wait for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("Received signal, shutting down gracefully...", map[string]interface{}{"signal": sig.String()})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", map[string]interface{}{"error": err.Error()})
	}

	logger.Info("API service stopped", nil)
}

// logStartupHint inspects an initialization error and logs actionable guidance.
func logStartupHint(err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "database connection failed"), strings.Contains(msg, "sqlx database connection"):
		logger.Error("Hint: ensure PostgreSQL is running and DATABASE_URL is set. Try: make docker-up", nil)
	case strings.Contains(msg, "redis connection failed"):
		logger.Error("Hint: ensure Redis is running and REDIS_URL is set. Try: make docker-up", nil)
	case strings.Contains(msg, "migration"):
		logger.Error("Hint: database migrations failed. Try: make migrate-up", nil)
	case strings.Contains(msg, "configuration error"):
		logger.Error("Hint: check your .env file. Try: make ensure-env && make validate-env", nil)
	}
}
