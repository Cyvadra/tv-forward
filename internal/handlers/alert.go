package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/models"
	"github.com/Cyvadra/tv-forward/internal/services"
	"github.com/gin-gonic/gin"
)

// Global handler instance
var globalHandler *AlertHandler

// TradingViewAlert represents the structure of a TradingView webhook alert
type TradingViewAlert struct {
	Strategy string  `json:"strategy"`
	Symbol   string  `json:"symbol"`
	Action   string  `json:"action"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Message  string  `json:"message"`
	// Additional fields that might be present
	Exchange string `json:"exchange,omitempty"`
	Time     string `json:"time,omitempty"`
}

// AlertHandler handles TradingView alert webhooks
type AlertHandler struct {
	alertService   *services.AlertService
	forwardService *services.ForwardService
	tradingService *services.TradingService
	userService    *services.UserService
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler() *AlertHandler {
	userService := services.NewUserService()
	tradingService := services.NewTradingService()
	tradingService.SetUserService(userService)

	return &AlertHandler{
		alertService:   services.NewAlertService(),
		forwardService: services.NewForwardService(),
		tradingService: tradingService,
		userService:    userService,
	}
}

// SetGlobalHandler sets the global handler instance
func SetGlobalHandler(handler *AlertHandler) {
	globalHandler = handler
}

// GetGlobalHandler returns the global handler instance
func GetGlobalHandler() *AlertHandler {
	return globalHandler
}

// SetConfig sets the configuration for all services
func (h *AlertHandler) SetConfig(cfg *config.Config) {
	h.forwardService.SetConfig(cfg)
	h.tradingService.SetConfig(cfg)
}

// SetUserConfig sets the user configuration for all services
func (h *AlertHandler) SetUserConfig(userConfig *config.UserConfig) {
	h.userService.SetUserConfig(userConfig)
}

// HandleTradingViewAlert handles incoming TradingView alerts
func (h *AlertHandler) HandleTradingViewAlert(c *gin.Context) {
	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Try to parse as TradingView signal first
	var tvSignal models.TradingViewSignal
	if err := json.Unmarshal(body, &tvSignal); err == nil && tvSignal.APISec != "" {
		// This is a TradingView trading signal
		h.handleTradingViewSignal(c, &tvSignal, body)
		return
	}

	// Try to parse as legacy alert format
	var alert TradingViewAlert
	flagIsTradingAlert := true

	if err := json.Unmarshal(body, &alert); err != nil {
		flagIsTradingAlert = false
		// Log the raw request body for debugging
		log.Printf("Raw request body: %s", string(body))
		alert = TradingViewAlert{
			Strategy: "alert",
			Symbol:   "",
			Action:   "",
			Price:    0.0,
			Quantity: 0.0,
			Message:  string(body),
			Exchange: "",
			Time:     time.Now().Format(time.RFC3339),
		}
	}

	// Get raw payload for logging
	rawPayload, _ := json.Marshal(alert)

	// Create alert record
	alertRecord := &models.Alert{
		Strategy:   alert.Strategy,
		Symbol:     alert.Symbol,
		Action:     alert.Action,
		Price:      alert.Price,
		Quantity:   alert.Quantity,
		Message:    alert.Message,
		RawPayload: string(rawPayload),
		Status:     "received",
		CreatedAt:  time.Now(),
	}

	// Save alert to database
	if err := h.alertService.SaveAlert(alertRecord); err != nil {
		log.Printf("Failed to save alert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save alert"})
		return
	}

	// Forward alert to downstream endpoints
	go func() {
		if err := h.forwardService.ForwardAlert(alertRecord); err != nil {
			log.Printf("Failed to forward alert: %v", err)
		}
	}()

	// Process trading signal if applicable
	go func() {
		if flagIsTradingAlert {
			if err := h.tradingService.ProcessTradingSignal(alertRecord); err != nil {
				log.Printf("Failed to process trading signal: %v", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Alert received and processed",
		"alert_id": alertRecord.ID,
	})
}

// handleTradingViewSignal handles TradingView trading signals
func (h *AlertHandler) handleTradingViewSignal(c *gin.Context, signal *models.TradingViewSignal, body []byte) {
	// Process the trading signal
	if err := h.tradingService.ProcessTradingViewSignal(signal); err != nil {
		log.Printf("Failed to process trading signal: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process trading signal",
			"details": err.Error(),
		})
		return
	}

	// Also create an alert record for tracking
	alertRecord := &models.Alert{
		Strategy:   "trading_signal",
		Symbol:     signal.Symbol,
		Action:     signal.Action,
		Price:      0, // Will be parsed from string if needed
		Quantity:   0, // Will be parsed from string if needed
		Message:    fmt.Sprintf("Trading signal: %s %s %s", signal.Action, signal.Symbol, signal.PositionSize),
		RawPayload: string(body),
		Status:     "processed",
		CreatedAt:  time.Now(),
	}

	// Save alert to database
	if err := h.alertService.SaveAlert(alertRecord); err != nil {
		log.Printf("Failed to save alert: %v", err)
	}

	// Forward alert to downstream endpoints
	go func() {
		if err := h.forwardService.ForwardAlert(alertRecord); err != nil {
			log.Printf("Failed to forward alert: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":   "Trading signal received and processed",
		"signal_id": signal.ID,
		"api_sec":   signal.APISec,
		"symbol":    signal.Symbol,
		"action":    signal.Action,
		"alert_id":  alertRecord.ID,
	})
}

// GetAlerts retrieves all alerts with pagination
func (h *AlertHandler) GetAlerts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

	alerts, total, err := h.alertService.GetAlerts(page, limit, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GetAlert retrieves a specific alert by ID
func (h *AlertHandler) GetAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	alert, err := h.alertService.GetAlert(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// GetTradingSignals retrieves trading signals for an alert
func (h *AlertHandler) GetTradingSignals(c *gin.Context) {
	alertID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	signals, err := h.tradingService.GetTradingSignals(uint(alertID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve trading signals"})
		return
	}

	c.JSON(http.StatusOK, signals)
}

// GetUserSignals retrieves trading signals for a specific user by api_sec
func (h *AlertHandler) GetUserSignals(c *gin.Context) {
	apiSec := c.Param("api_sec")
	if apiSec == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_sec parameter is required"})
		return
	}

	// Get user by api_sec
	user, err := h.userService.GetOrCreateUserByAPISec(apiSec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	signals, err := h.userService.GetUserTradingSignals(user.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve trading signals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"api_sec": user.APISec,
		"signals": signals,
	})
}

// GetUserPositions retrieves current positions for a specific user by api_sec
func (h *AlertHandler) GetUserPositions(c *gin.Context) {
	apiSec := c.Param("api_sec")
	if apiSec == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_sec parameter is required"})
		return
	}

	// Get user by api_sec
	user, err := h.userService.GetOrCreateUserByAPISec(apiSec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	positions, err := h.userService.GetUserPositions(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve positions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":   user.ID,
		"api_sec":   user.APISec,
		"positions": positions,
	})
}
