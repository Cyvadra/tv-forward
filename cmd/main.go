package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/database"
	"github.com/Cyvadra/tv-forward/internal/handlers"
	"github.com/Cyvadra/tv-forward/internal/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config from %s, creating default config...", *configFile))
	}

	// Load user configuration
	userConfigFile := "users.yaml"
	userConfig, err := config.LoadUserConfig(userConfigFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to load user config from %s, creating default user config...", userConfigFile))
	}

	// Initialize database
	if err := database.InitDatabase(cfg.Database.DSN); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Set up Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Add middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Set up services with configuration
	setupServices(cfg, userConfig)

	// Set up routes
	routes.SetupRoutes(r)

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	log.Printf("TradingView webhook endpoint: http://%s/api/v1/webhook/tradingview", addr)
	log.Printf("Health check: http://%s/health", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// setupServices configures all services with the application configuration
func setupServices(cfg *config.Config, userConfig *config.UserConfig) {
	// Create alert handler and set its configuration
	alertHandler := handlers.NewAlertHandler()
	alertHandler.SetConfig(cfg)

	// Set user configuration for user service
	alertHandler.SetUserConfig(userConfig)

	// Store the configured handler globally so routes can access it
	handlers.SetGlobalHandler(alertHandler)
}
