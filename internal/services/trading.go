package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Cyvadra/tv-forward/broker"
	"github.com/Cyvadra/tv-forward/broker/binance"
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
	log.Printf("Starting Binance legacy execution for alert %d", alert.ID)

	if s.config == nil || !s.config.Trading.Binance.IsActive {
		log.Printf("Binance trading not configured or not active for alert %d", alert.ID)
		return fmt.Errorf("binance trading is not configured or not active")
	}

	// Validate credentials
	if s.config.Trading.Binance.APIKey == "" || s.config.Trading.Binance.SecretKey == "" {
		log.Printf("Binance credentials are empty for alert %d", alert.ID)
		return fmt.Errorf("binance credentials are not configured")
	}

	// Create Binance client using config credentials
	client := binance.NewClient()
	brokerCreds := &broker.Credentials{
		APIKey:    s.config.Trading.Binance.APIKey,
		SecretKey: s.config.Trading.Binance.SecretKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Initialize(ctx, brokerCreds); err != nil {
		log.Printf("Failed to initialize Binance client for alert %d: %v", alert.ID, err)
		return fmt.Errorf("failed to initialize binance client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close Binance client: %v", closeErr)
		}
	}()

	// Convert alert to order request
	orderReq, err := s.convertAlertToOrderRequest(alert)
	if err != nil {
		log.Printf("Failed to convert alert to order request for alert %d: %v", alert.ID, err)
		return fmt.Errorf("failed to convert alert to order request: %w", err)
	}

	log.Printf("Executing %s order for %s on Binance at price %.8f (alert %d)",
		alert.Action, alert.Symbol, alert.Price, alert.ID)

	// Place order with retry logic
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	order, err := client.PlaceOrder(ctx, orderReq)
	if err != nil {
		log.Printf("Binance legacy order failed for alert %d: %v", alert.ID, err)

		// Check if this is a retryable error
		if broker.IsRetryableError(err) {
			log.Printf("Retryable error detected, attempting retry for alert %d", alert.ID)
			// Wait a bit and retry once
			time.Sleep(1 * time.Second)

			ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel2()

			order, err = client.PlaceOrder(ctx2, orderReq)
			if err != nil {
				log.Printf("Binance legacy order retry failed for alert %d: %v", alert.ID, err)
				return fmt.Errorf("failed to place binance order after retry: %w", err)
			}
			log.Printf("Binance legacy order succeeded on retry for alert %d", alert.ID)
		} else {
			return fmt.Errorf("failed to place binance order: %w", err)
		}
	}

	// Validate response
	if order == nil {
		log.Printf("Warning: Received nil order response from Binance for alert %d", alert.ID)
		return fmt.Errorf("received nil order response from binance")
	}

	// Store order ID
	signal.OrderID = order.ID
	log.Printf("Binance legacy order placed successfully for alert %d: ID=%s, Status=%s, Symbol=%s",
		alert.ID, order.ID, order.Status, order.Symbol)

	// Log additional order details for audit trail
	log.Printf("Binance legacy order details - Alert: %d, OrderID: %s, Symbol: %s, Side: %s, Quantity: %s, Price: %s, Status: %s",
		alert.ID, order.ID, order.Symbol, order.Side, order.Quantity, order.Price, order.Status)

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

// executeOnBinance executes a trade on Binance using the broker system
func (s *TradingService) executeOnBinance(userID uint, signal *models.TradingSignal) error {
	log.Printf("Starting Binance execution for user %d, signal %s", userID, signal.SignalID)

	// Get user credentials
	credential, err := s.userService.GetUserCredentials(userID, "binance")
	if err != nil {
		log.Printf("Failed to get Binance credentials for user %d: %v", userID, err)
		return fmt.Errorf("failed to get binance credentials: %w", err)
	}

	// Create Binance client
	binanceClient, err := createBinanceClient(credential)
	if err != nil {
		log.Printf("Failed to create Binance client for user %d: %v", userID, err)
		return fmt.Errorf("failed to create binance client: %w", err)
	}
	defer func() {
		if closeErr := binanceClient.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close Binance client: %v", closeErr)
		}
	}()

	// Convert signal to order request
	orderReq, err := s.convertSignalToOrderRequest(signal)
	if err != nil {
		log.Printf("Failed to convert signal to order request: %v", err)
		return fmt.Errorf("failed to convert signal to order request: %w", err)
	}

	if orderReq == nil {
		log.Printf("No order needed for signal: %s", signal.Symbol)
		return nil
	}

	log.Printf("Executing %s order for %s on Binance: side=%s, quantity=%s, price=%s, user=%d",
		signal.Action, signal.Symbol, orderReq.Side, orderReq.Quantity, orderReq.Price, userID)

	// Place order with retry logic for temporary errors
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	order, err := binanceClient.PlaceOrder(ctx, orderReq)
	if err != nil {
		log.Printf("Binance order failed for user %d: %v", userID, err)

		// Check if this is a retryable error
		if broker.IsRetryableError(err) {
			log.Printf("Retryable error detected, attempting retry for user %d", userID)
			// Wait a bit and retry once
			time.Sleep(1 * time.Second)

			ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel2()

			order, err = binanceClient.PlaceOrder(ctx2, orderReq)
			if err != nil {
				log.Printf("Binance order retry failed for user %d: %v", userID, err)
				return fmt.Errorf("failed to place binance order after retry: %w", err)
			}
			log.Printf("Binance order succeeded on retry for user %d", userID)
		} else {
			return fmt.Errorf("failed to place binance order: %w", err)
		}
	}

	// Store order ID and validate response
	if order == nil {
		log.Printf("Warning: Received nil order response from Binance for user %d", userID)
		return fmt.Errorf("received nil order response from binance")
	}

	signal.OrderID = order.ID
	log.Printf("Binance order placed successfully for user %d: ID=%s, Status=%s, Symbol=%s",
		userID, order.ID, order.Status, order.Symbol)

	// Log additional order details for audit trail
	log.Printf("Binance order details - User: %d, OrderID: %s, Symbol: %s, Side: %s, Quantity: %s, Price: %s, Status: %s",
		userID, order.ID, order.Symbol, order.Side, order.Quantity, order.Price, order.Status)

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

// Helper functions for Binance integration

// createBinanceClient creates a Binance client using user credentials
func createBinanceClient(credential *models.UserCredential) (broker.Broker, error) {
	// Create Binance client
	client := binance.NewClient()

	// Initialize with credentials
	brokerCreds := &broker.Credentials{
		APIKey:    credential.APIKey,
		SecretKey: credential.SecretKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Initialize(ctx, brokerCreds); err != nil {
		return nil, fmt.Errorf("failed to initialize binance client: %w", err)
	}

	return client, nil
}

// convertSignalToOrderRequest converts a trading signal to a broker order request
func (s *TradingService) convertSignalToOrderRequest(signal *models.TradingSignal) (*broker.OrderRequest, error) {
	// Parse position sizes
	prevSize, err := strconv.ParseFloat(signal.PrevMarketPositionSize, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid prev_market_position_size: %w", err)
	}

	targetSize, err := strconv.ParseFloat(signal.MarketPositionSize, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid market_position_size: %w", err)
	}

	// Calculate order quantity and side
	quantity, side, err := broker.CalculateOrderQuantity(prevSize, targetSize)
	if err != nil {
		if err.Error() == "no position change required" {
			return nil, nil // No order needed
		}
		return nil, err
	}

	// Determine position side for futures
	var positionSide broker.PositionSide
	if targetSize > 0 {
		positionSide = broker.PositionSideLong
	} else if targetSize < 0 {
		positionSide = broker.PositionSideShort
	} else {
		positionSide = broker.PositionSideBoth
	}

	// Determine order type
	var orderType broker.OrderType
	var price string
	if signal.OrderType == "limit" && signal.Price != "" {
		orderType = broker.OrderTypeLimit
		price = signal.Price
	} else {
		orderType = broker.OrderTypeMarket
	}

	// Format symbol for Binance
	symbol := broker.FormatSymbol(signal.Symbol, "binance")

	// Create order request
	orderReq := &broker.OrderRequest{
		Symbol:       symbol,
		Side:         side,
		Type:         orderType,
		Quantity:     broker.FormatQuantity(quantity, 8),
		Price:        price,
		PositionSide: positionSide,
		TimeInForce:  "GTC",
	}

	// Set reduce only for closing positions
	if (prevSize > 0 && targetSize < prevSize) || (prevSize < 0 && targetSize > prevSize) {
		orderReq.ReduceOnly = true
	}

	return orderReq, nil
}

// convertAlertToOrderRequest converts a legacy alert to a broker order request
func (s *TradingService) convertAlertToOrderRequest(alert *models.Alert) (*broker.OrderRequest, error) {
	// Determine order side based on action
	var side broker.OrderSide
	switch alert.Action {
	case "buy":
		side = broker.OrderSideBuy
	case "sell":
		side = broker.OrderSideSell
	default:
		return nil, fmt.Errorf("unsupported action: %s", alert.Action)
	}

	// Format symbol for Binance
	symbol := broker.FormatSymbol(alert.Symbol, "binance")

	// Create order request (using market orders for alerts)
	orderReq := &broker.OrderRequest{
		Symbol:       symbol,
		Side:         side,
		Type:         broker.OrderTypeMarket,
		Quantity:     fmt.Sprintf("%.8f", alert.Quantity),
		PositionSide: broker.PositionSideBoth, // Default for one-way mode
		TimeInForce:  "GTC",
	}

	// Use limit order if price is specified
	if alert.Price > 0 {
		orderReq.Type = broker.OrderTypeLimit
		orderReq.Price = fmt.Sprintf("%.8f", alert.Price)
	}

	return orderReq, nil
}

// trackBinanceOrderStatus tracks the status of a Binance order and updates the signal
func (s *TradingService) trackBinanceOrderStatus(ctx context.Context, client broker.Broker, signal *models.TradingSignal) error {
	if signal.OrderID == "" {
		return fmt.Errorf("no order ID to track")
	}

	// Get order status from Binance
	order, err := client.GetOrder(ctx, signal.Symbol, signal.OrderID)
	if err != nil {
		log.Printf("Failed to get order status for %s: %v", signal.OrderID, err)
		return fmt.Errorf("failed to get order status: %w", err)
	}

	// Update signal with order status
	prevStatus := signal.Status
	switch order.Status {
	case broker.OrderStatusFilled:
		signal.Status = "filled"
		if signal.ExecutedAt == nil {
			now := time.Now()
			signal.ExecutedAt = &now
		}
	case broker.OrderStatusCanceled, broker.OrderStatusRejected, broker.OrderStatusExpired:
		signal.Status = "failed"
		signal.ErrorMessage = fmt.Sprintf("Order %s: %s", string(order.Status), order.ID)
	case broker.OrderStatusPartiallyFilled:
		signal.Status = "partially_filled"
	default:
		signal.Status = "pending"
	}

	// Log status change
	if prevStatus != signal.Status {
		log.Printf("Order status changed for %s: %s -> %s (OrderID: %s)",
			signal.Symbol, prevStatus, signal.Status, signal.OrderID)
	}

	// Save updated signal
	if err := s.db.Save(signal).Error; err != nil {
		log.Printf("Failed to update signal status: %v", err)
		return fmt.Errorf("failed to update signal status: %w", err)
	}

	return nil
}

// getBinanceOrderStatus retrieves the current status of a Binance order
func (s *TradingService) getBinanceOrderStatus(userID uint, symbol, orderID string) (*broker.Order, error) {
	// Get user credentials
	credential, err := s.userService.GetUserCredentials(userID, "binance")
	if err != nil {
		return nil, fmt.Errorf("failed to get binance credentials: %w", err)
	}

	// Create Binance client
	client, err := createBinanceClient(credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create binance client: %w", err)
	}
	defer client.Close()

	// Get order status
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	order, err := client.GetOrder(ctx, symbol, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	return order, nil
}
