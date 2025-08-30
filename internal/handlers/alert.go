package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/models"
	"github.com/Cyvadra/tv-forward/internal/services"
	"github.com/gin-gonic/gin"
)

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
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler() *AlertHandler {
	return &AlertHandler{
		alertService:   services.NewAlertService(),
		forwardService: services.NewForwardService(),
		tradingService: services.NewTradingService(),
	}
}

// SetConfig sets the configuration for all services
func (h *AlertHandler) SetConfig(cfg *config.Config) {
	h.forwardService.SetConfig(cfg)
	h.tradingService.SetConfig(cfg)
}

// HandleTradingViewAlert handles incoming TradingView alerts
func (h *AlertHandler) HandleTradingViewAlert(c *gin.Context) {
	var alert TradingViewAlert
	if err := c.ShouldBindJSON(&alert); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
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
		if err := h.tradingService.ProcessTradingSignal(alertRecord); err != nil {
			log.Printf("Failed to process trading signal: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Alert received and processed",
		"alert_id": alertRecord.ID,
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
