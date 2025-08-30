package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/database"
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
		log.Printf("Failed to load config from %s, creating default config...", *configFile)
		cfg = createDefaultConfig()
		if err := config.SaveConfig(cfg, *configFile); err != nil {
			log.Printf("Failed to save default config: %v", err)
		}
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
	setupServices(cfg)

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
func setupServices(cfg *config.Config) {
	// Services are configured within the handlers when they are created
	// The configuration is passed through the handler initialization
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: "8080",
		},
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "tv-forward.db",
		},
		Endpoints: []config.EndpointConfig{
			{
				Name:     "Telegram Bot",
				Type:     "telegram",
				URL:      "",
				Token:    "YOUR_TELEGRAM_BOT_TOKEN",
				ChatID:   "YOUR_CHAT_ID",
				IsActive: false,
			},
			{
				Name:     "WeChat Bot",
				Type:     "wechat",
				URL:      "YOUR_WECHAT_WEBHOOK_URL",
				Token:    "",
				ChatID:   "",
				IsActive: false,
			},
			{
				Name:     "DingTalk Bot",
				Type:     "dingtalk",
				URL:      "YOUR_DINGTALK_WEBHOOK_URL",
				Token:    "",
				ChatID:   "",
				IsActive: false,
			},
		},
		Trading: config.TradingConfig{
			Bitget: config.BitgetConfig{
				APIKey:     "YOUR_BITGET_API_KEY",
				SecretKey:  "YOUR_BITGET_SECRET_KEY",
				Passphrase: "YOUR_BITGET_PASSPHRASE",
				IsActive:   false,
			},
			Binance: config.BinanceConfig{
				APIKey:    "YOUR_BINANCE_API_KEY",
				SecretKey: "YOUR_BINANCE_SECRET_KEY",
				IsActive:  false,
			},
			Derbit: config.DerbitConfig{
				APIKey:    "YOUR_DERBIT_API_KEY",
				SecretKey: "YOUR_DERBIT_SECRET_KEY",
				IsActive:  false,
			},
		},
	}
}
