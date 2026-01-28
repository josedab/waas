package analytics

import (
	"context"
	"fmt"
	"net/http"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/metrics"
	"webhook-platform/pkg/repository"
	"webhook-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Service struct {
	router        *gin.Engine
	db            *database.DB
	logger        *utils.Logger
	config        *utils.Config
	handlers      *Handlers
	wsManager     *WebSocketManager
	aggregator    *Aggregator
	analyticsRepo repository.AnalyticsRepositoryInterface
}

func NewService() (*Service, error) {
	config := utils.LoadConfig()
	logger := utils.NewLogger("analytics-service")

	db, err := database.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Initialize repositories
	analyticsRepo := repository.NewAnalyticsRepository(db.Pool)

	// Initialize components
	handlers := NewHandlers(analyticsRepo)
	wsManager := NewWebSocketManager(analyticsRepo, logger)
	aggregator := NewAggregator(analyticsRepo, logger)

	// Setup router
	router := gin.Default()

	// Add Prometheus middleware
	router.Use(metrics.PrometheusMiddleware())

	// Register routes
	handlers.RegisterRoutes(router)
	wsManager.RegisterWebSocketRoutes(router)

	// Add Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Add health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	return &Service{
		router:        router,
		db:            db,
		logger:        logger,
		config:        config,
		handlers:      handlers,
		wsManager:     wsManager,
		aggregator:    aggregator,
		analyticsRepo: analyticsRepo,
	}, nil
}

// Handler returns the HTTP handler (gin.Engine) for use with a custom http.Server.
func (s *Service) Handler() http.Handler {
	return s.router
}

// StartWorkers starts background workers (WebSocket manager, aggregator).
func (s *Service) StartWorkers() {
	ctx := context.Background()
	s.wsManager.Start(ctx)
	s.aggregator.Start(ctx)
}

func (s *Service) Start(addr string) error {
	s.logger.Info("Starting analytics service", map[string]interface{}{
		"address": addr,
	})

	// Start background workers
	s.StartWorkers()

	return s.router.Run(addr)
}

func (s *Service) Stop() {
	s.logger.Info("Stopping analytics service", nil)

	// Stop background workers
	s.wsManager.Stop()
	s.aggregator.Stop()

	// Close database connection
	s.db.Close()
}
