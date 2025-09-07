package routes

import (
	"github.com/Cyvadra/tv-forward/internal/handlers"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for the application
func SetupRoutes(r *gin.Engine) {
	// Get the configured handler
	alertHandler := handlers.GetGlobalHandler()
	if alertHandler == nil {
		// Fallback to creating a new handler if global handler is not set
		alertHandler = handlers.NewAlertHandler()
	}

	// API routes
	api := r.Group("/api/v1")
	{
		// TradingView webhook endpoint
		api.POST("/webhook/tradingview", alertHandler.HandleTradingViewAlert)

		// Alert management endpoints
		alerts := api.Group("/alerts")
		{
			alerts.GET("", alertHandler.GetAlerts)
			alerts.GET("/:id", alertHandler.GetAlert)
			alerts.GET("/:id/signals", alertHandler.GetTradingSignals)
		}

		// User management endpoints
		users := api.Group("/users")
		{
			users.GET("/:api_sec/signals", alertHandler.GetUserSignals)
			users.GET("/:api_sec/positions", alertHandler.GetUserPositions)
		}
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "tv-forward",
		})
	})

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "TradingView Alert Forwarder",
			"version": "1.0.0",
			"endpoints": gin.H{
				"webhook": "/api/v1/webhook/tradingview",
				"alerts":  "/api/v1/alerts",
				"health":  "/health",
			},
		})
	})
}
