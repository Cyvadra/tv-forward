package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/database"
	"github.com/Cyvadra/tv-forward/internal/models"
	"gorm.io/gorm"
)

// TradingService handles trading operations
type TradingService struct {
	db          *gorm.DB
	config      *config.Config
	userService *UserService
}

// NewTradingService creates a new trading service
func NewTradingService() *TradingService {
	return &TradingService{
		db:          database.GetDB(),
		config:      nil, // Will be set later
		userService: NewUserService(),
	}
}

// SetConfig sets the configuration for the trading service
func (s *TradingService) SetConfig(cfg *config.Config) {
	s.config = cfg
}

// SetUserService sets the user service
func (s *TradingService) SetUserService(userService *UserService) {
	s.userService = userService
}

// ProcessTradingViewSignal processes a TradingView signal and executes orders
func (s *TradingService) ProcessTradingViewSignal(signalData *models.TradingViewSignal) error {
	if s.config == nil || s.userService == nil {
		return fmt.Errorf("configuration or user service not set")
	}

	// Get or create user by api_sec
	user, err := s.userService.GetOrCreateUserByAPISec(signalData.APISec)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Validate position change
	if err := s.validatePositionChange(user.ID, signalData); err != nil {
		return fmt.Errorf("position validation failed: %w", err)
	}

	// Create trading signal record
	rawPayload, _ := json.Marshal(signalData)
	tradingSignal := &models.TradingSignal{
		UserID:                 user.ID,
		SignalID:               signalData.ID,
		Symbol:                 signalData.Symbol,
		Exchange:               signalData.ExchangeName,
		Action:                 signalData.Action,
		PositionSize:           signalData.PositionSize,
		Price:                  signalData.Price,
		MarketPosition:         signalData.MarketPosition,
		MarketPositionSize:     signalData.MarketPositionSize,
		PrevMarketPosition:     signalData.PrevMarketPosition,
		PrevMarketPositionSize: signalData.PrevMarketPositionSize,
		Leverage:               signalData.Leverage,
		TradingMode:            signalData.TradingMode,
		OrderType:              signalData.OrderType,
		Status:                 "pending",
		RawPayload:             string(rawPayload),
		CreatedAt:              time.Now(),
	}

	// Try to execute on the specified exchange
	var executionError error
	switch signalData.ExchangeName {
	case "bitget":
		executionError = s.executeOnBitget(user.ID, tradingSignal)
	case "binance":
		executionError = s.executeOnBinance(user.ID, tradingSignal)
	case "okx":
		executionError = s.executeOnOKX(user.ID, tradingSignal)
	default:
		executionError = fmt.Errorf("unsupported exchange: %s", signalData.ExchangeName)
	}

	if executionError != nil {
		tradingSignal.Status = "failed"
		tradingSignal.ErrorMessage = executionError.Error()
		log.Printf("Trading execution failed for user %s, signal %s: %v",
			signalData.APISec, signalData.ID, executionError)
	} else {
		tradingSignal.Status = "filled"
		now := time.Now()
		tradingSignal.ExecutedAt = &now

		// Update position
		if err := s.updateUserPosition(user.ID, signalData); err != nil {
			log.Printf("Failed to update position for user %s: %v", signalData.APISec, err)
		}
	}

	// Save trading signal
	if err := s.db.Create(tradingSignal).Error; err != nil {
		return fmt.Errorf("failed to save trading signal: %w", err)
	}

	return nil
}

// ProcessTradingSignal processes a trading signal and executes orders (legacy method)
func (s *TradingService) ProcessTradingSignal(alert *models.Alert) error {
	if s.config == nil {
		return fmt.Errorf("configuration not set")
	}

	// Create trading signal record
	signal := &models.TradingSignal{
		AlertID:   alert.ID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Try to execute on different platforms
	var executionErrors []string

	// Try Bitget
	if s.config.Trading.Bitget.IsActive {
		if err := s.executeOnBitgetLegacy(alert, signal); err != nil {
			executionErrors = append(executionErrors, fmt.Sprintf("Bitget: %v", err))
		} else {
			signal.Exchange = "bitget"
			signal.Status = "filled"
			now := time.Now()
			signal.ExecutedAt = &now
		}
	}

	// Try Binance
	if signal.Status == "pending" && s.config.Trading.Binance.IsActive {
		if err := s.executeOnBinanceLegacy(alert, signal); err != nil {
			executionErrors = append(executionErrors, fmt.Sprintf("Binance: %v", err))
		} else {
			signal.Exchange = "binance"
			signal.Status = "filled"
			now := time.Now()
			signal.ExecutedAt = &now
		}
	}

	// Try OKX
	if signal.Status == "pending" && s.config.Trading.OKX.IsActive {
		if err := s.executeOnOKXLegacy(alert, signal); err != nil {
			executionErrors = append(executionErrors, fmt.Sprintf("OKX: %v", err))
		} else {
			signal.Exchange = "okx"
			signal.Status = "filled"
			now := time.Now()
			signal.ExecutedAt = &now
		}
	}

	// If all platforms failed, mark as failed
	if signal.Status == "pending" {
		signal.Status = "failed"
		log.Printf("All trading platforms failed for alert %d: %v", alert.ID, executionErrors)
	}

	// Save trading signal
	if err := s.db.Create(signal).Error; err != nil {
		return fmt.Errorf("failed to save trading signal: %w", err)
	}

	// Update alert status
	alertStatus := "processed"
	if signal.Status == "failed" {
		alertStatus = "failed"
	}

	if err := s.db.Model(alert).Update("status", alertStatus).Error; err != nil {
		log.Printf("Failed to update alert status: %v", err)
	}

	return nil
}

// validatePositionChange validates the position change based on prev_market_position_size
func (s *TradingService) validatePositionChange(userID uint, signal *models.TradingViewSignal) error {
	// Get current position
	positions, err := s.userService.GetUserPositions(userID)
	if err != nil {
		return fmt.Errorf("failed to get user positions: %w", err)
	}

	// Find position for this symbol and exchange
	var currentPosition *models.Position
	for i := range positions {
		if positions[i].Symbol == signal.Symbol && positions[i].Exchange == signal.ExchangeName {
			currentPosition = &positions[i]
			break
		}
	}

	// Parse previous position size from signal
	prevSize, err := strconv.ParseFloat(signal.PrevMarketPositionSize, 64)
	if err != nil {
		return fmt.Errorf("invalid prev_market_position_size: %w", err)
	}

	// Check if previous position size matches current position
	if currentPosition != nil {
		currentSize, err := strconv.ParseFloat(currentPosition.Size, 64)
		if err != nil {
			return fmt.Errorf("invalid current position size: %w", err)
		}

		// Allow small floating point differences
		if abs(currentSize-prevSize) > 0.0001 {
			log.Printf("Position size mismatch for user %d, symbol %s: expected %.8f, got %.8f",
				userID, signal.Symbol, prevSize, currentSize)
			// Don't fail the trade, just log the warning
		}
	} else if prevSize != 0 {
		log.Printf("No current position found for user %d, symbol %s, but prev_size is %.8f",
			userID, signal.Symbol, prevSize)
	}

	return nil
}

// updateUserPosition updates the user's position after a successful trade
func (s *TradingService) updateUserPosition(userID uint, signal *models.TradingViewSignal) error {
	// Parse position size
	positionSize, err := strconv.ParseFloat(signal.MarketPositionSize, 64)
	if err != nil {
		return fmt.Errorf("invalid market_position_size: %w", err)
	}

	// If position size is 0, close the position
	if positionSize == 0 {
		return s.userService.ClosePosition(userID, signal.Symbol, signal.ExchangeName)
	}

	// Update or create position
	return s.userService.UpdatePosition(
		userID,
		signal.Symbol,
		signal.ExchangeName,
		signal.MarketPosition,
		signal.MarketPositionSize,
		signal.Price,
		signal.Price, // Use price as mark price for now
		"0",          // Unrealized PnL not available in signal
		signal.Leverage,
		signal.TradingMode,
	)
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// executeOnBitget executes a trade on Bitget
func (s *TradingService) executeOnBitget(userID uint, signal *models.TradingSignal) error {
	// Get user credentials
	credential, err := s.userService.GetUserCredentials(userID, "bitget")
	if err != nil {
		return fmt.Errorf("failed to get bitget credentials: %w", err)
	}

	// This is a placeholder implementation
	// In a real implementation, you would integrate with Bitget's API
	log.Printf("Executing %s order for %s on Bitget at price %s for user %d",
		signal.Action, signal.Symbol, signal.Price, userID)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("bitget_%d_%d", signal.ID, time.Now().Unix())

	// TODO: Implement actual Bitget API integration using credential
	_ = credential

	return nil
}

// executeOnBitgetLegacy executes a trade on Bitget (legacy method for alerts)
func (s *TradingService) executeOnBitgetLegacy(alert *models.Alert, signal *models.TradingSignal) error {
	// This is a placeholder implementation
	// In a real implementation, you would integrate with Bitget's API
	log.Printf("Executing %s order for %s on Bitget at price %.8f",
		alert.Action, alert.Symbol, alert.Price)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("bitget_%d_%d", alert.ID, time.Now().Unix())

	return nil
}

// executeOnBinanceLegacy executes a trade on Binance (legacy method for alerts)
func (s *TradingService) executeOnBinanceLegacy(alert *models.Alert, signal *models.TradingSignal) error {
	// This is a placeholder implementation
	// In a real implementation, you would integrate with Binance's API
	log.Printf("Executing %s order for %s on Binance at price %.8f",
		alert.Action, alert.Symbol, alert.Price)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("binance_%d_%d", alert.ID, time.Now().Unix())

	return nil
}

// executeOnOKXLegacy executes a trade on OKX (legacy method for alerts)
func (s *TradingService) executeOnOKXLegacy(alert *models.Alert, signal *models.TradingSignal) error {
	// This is a placeholder implementation
	// In a real implementation, you would integrate with OKX's API
	log.Printf("Executing %s order for %s on OKX at price %.8f",
		alert.Action, alert.Symbol, alert.Price)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("okx_%d_%d", alert.ID, time.Now().Unix())

	return nil
}

// executeOnBinance executes a trade on Binance
func (s *TradingService) executeOnBinance(userID uint, signal *models.TradingSignal) error {
	// Get user credentials
	credential, err := s.userService.GetUserCredentials(userID, "binance")
	if err != nil {
		return fmt.Errorf("failed to get binance credentials: %w", err)
	}

	// This is a placeholder implementation
	// In a real implementation, you would integrate with Binance's API
	log.Printf("Executing %s order for %s on Binance at price %s for user %d",
		signal.Action, signal.Symbol, signal.Price, userID)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("binance_%d_%d", signal.ID, time.Now().Unix())

	// TODO: Implement actual Binance API integration using credential
	_ = credential

	return nil
}

// executeOnOKX executes a trade on OKX
func (s *TradingService) executeOnOKX(userID uint, signal *models.TradingSignal) error {
	// Get user credentials
	credential, err := s.userService.GetUserCredentials(userID, "okx")
	if err != nil {
		return fmt.Errorf("failed to get okx credentials: %w", err)
	}

	// This is a placeholder implementation
	// In a real implementation, you would integrate with OKX's API
	log.Printf("Executing %s order for %s on OKX at price %s for user %d",
		signal.Action, signal.Symbol, signal.Price, userID)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("okx_%d_%d", signal.ID, time.Now().Unix())

	// TODO: Implement actual OKX API integration using credential
	_ = credential

	return nil
}

// GetTradingSignals retrieves trading signals for an alert
func (s *TradingService) GetTradingSignals(alertID uint) ([]models.TradingSignal, error) {
	var signals []models.TradingSignal
	err := s.db.Where("alert_id = ?", alertID).Find(&signals).Error
	return signals, err
}

// GetTradingSignalsByPlatform retrieves trading signals by platform
func (s *TradingService) GetTradingSignalsByPlatform(platform string, limit int) ([]models.TradingSignal, error) {
	var signals []models.TradingSignal
	err := s.db.Where("platform = ?", platform).
		Order("created_at DESC").
		Limit(limit).
		Find(&signals).Error
	return signals, err
}

// GetTradingSignalsByStatus retrieves trading signals by status
func (s *TradingService) GetTradingSignalsByStatus(status string, limit int) ([]models.TradingSignal, error) {
	var signals []models.TradingSignal
	err := s.db.Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Find(&signals).Error
	return signals, err
}
