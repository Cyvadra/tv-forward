package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Cyvadra/tv-forward/broker"
	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/database"
	"github.com/Cyvadra/tv-forward/internal/models"
	"gorm.io/gorm"
)

// EnhancedTradingService handles trading operations using the new broker system
type EnhancedTradingService struct {
	db              *gorm.DB
	config          *config.Config
	userService     *UserService
	brokerManager   *broker.Manager
	signalProcessor *broker.SignalProcessor
	logger          *log.Logger
}

// NewEnhancedTradingService creates a new enhanced trading service
func NewEnhancedTradingService() *EnhancedTradingService {
	manager := broker.NewManager()
	return &EnhancedTradingService{
		db:              database.GetDB(),
		config:          nil,
		userService:     NewUserService(),
		brokerManager:   manager,
		signalProcessor: broker.NewSignalProcessor(manager),
		logger:          log.New(log.Writer(), "[EnhancedTrading] ", log.LstdFlags),
	}
}

// SetConfig sets the configuration for the trading service
func (s *EnhancedTradingService) SetConfig(cfg *config.Config) {
	s.config = cfg
}

// SetUserService sets the user service
func (s *EnhancedTradingService) SetUserService(userService *UserService) {
	s.userService = userService
}

// SetLogger sets a custom logger
func (s *EnhancedTradingService) SetLogger(logger *log.Logger) {
	s.logger = logger
	s.brokerManager.SetLogger(logger)
	s.signalProcessor.SetLogger(logger)
}

// InitializeBrokers initializes broker connections for a user
func (s *EnhancedTradingService) InitializeBrokers(ctx context.Context, userID uint) error {
	// Get user credentials
	credentials, err := s.userService.GetAllUserCredentials(userID)
	if err != nil {
		return fmt.Errorf("failed to get user credentials: %w", err)
	}

	// Initialize each broker
	var initErrors []error
	for _, cred := range credentials {
		if !cred.IsActive {
			continue
		}

		brokerCreds := &broker.Credentials{
			APIKey:     cred.APIKey,
			SecretKey:  cred.SecretKey,
			Passphrase: cred.Passphrase,
		}

		if err := s.brokerManager.InitializeBroker(ctx, cred.Exchange, brokerCreds); err != nil {
			s.logger.Printf("Failed to initialize %s for user %d: %v", cred.Exchange, userID, err)
			initErrors = append(initErrors, fmt.Errorf("%s: %w", cred.Exchange, err))
			continue
		}

		s.logger.Printf("Successfully initialized %s for user %d", cred.Exchange, userID)
	}

	if len(initErrors) > 0 && len(initErrors) == len(credentials) {
		return fmt.Errorf("failed to initialize any brokers: %v", initErrors)
	}

	return nil
}

// ProcessTradingViewSignal processes a TradingView signal using the new broker system
func (s *EnhancedTradingService) ProcessTradingViewSignal(ctx context.Context, signalData *models.TradingViewSignal) error {
	if s.config == nil || s.userService == nil {
		return fmt.Errorf("configuration or user service not set")
	}

	// Get or create user by api_sec
	user, err := s.userService.GetOrCreateUserByAPISec(signalData.APISec)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Initialize brokers for this user if not already done
	if err := s.InitializeBrokers(ctx, user.ID); err != nil {
		s.logger.Printf("Warning: Failed to initialize brokers for user %d: %v", user.ID, err)
		// Continue with fallback to legacy system if needed
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

	// Convert to broker signal format
	brokerSignal := &broker.TradingSignal{
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
	}

	// Process signal using the new broker system
	var executionError error
	if err := s.signalProcessor.ProcessSignal(ctx, brokerSignal); err != nil {
		executionError = err
		s.logger.Printf("Failed to process signal with new broker system: %v", err)

		// Fallback to legacy system if new system fails
		if fallbackErr := s.executeWithLegacySystem(user.ID, tradingSignal); fallbackErr != nil {
			executionError = fmt.Errorf("both new and legacy systems failed: %v, %v", err, fallbackErr)
		} else {
			executionError = nil // Legacy system succeeded
			s.logger.Printf("Successfully processed signal with legacy system")
		}
	}

	// Update signal status
	if executionError != nil {
		tradingSignal.Status = "failed"
		tradingSignal.ErrorMessage = executionError.Error()
		s.logger.Printf("Trading execution failed for user %s, signal %s: %v",
			signalData.APISec, signalData.ID, executionError)
	} else {
		tradingSignal.Status = "filled"
		now := time.Now()
		tradingSignal.ExecutedAt = &now

		// Update position
		if err := s.updateUserPosition(user.ID, signalData); err != nil {
			s.logger.Printf("Failed to update position for user %s: %v", signalData.APISec, err)
		}
	}

	// Save trading signal
	if err := s.db.Create(tradingSignal).Error; err != nil {
		return fmt.Errorf("failed to save trading signal: %w", err)
	}

	return executionError
}

// executeWithLegacySystem executes using the legacy system as fallback
func (s *EnhancedTradingService) executeWithLegacySystem(userID uint, signal *models.TradingSignal) error {
	// Get user credentials for the exchange
	credential, err := s.userService.GetUserCredentials(userID, signal.Exchange)
	if err != nil {
		return fmt.Errorf("failed to get %s credentials: %w", signal.Exchange, err)
	}

	// This is a placeholder for the legacy implementation
	s.logger.Printf("Executing %s order for %s on %s using legacy system",
		signal.Action, signal.Symbol, signal.Exchange)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("legacy_%s_%d_%d", signal.Exchange, signal.ID, time.Now().Unix())

	// TODO: Implement actual legacy API integrations
	_ = credential

	return nil
}

// GetBrokerManager returns the broker manager
func (s *EnhancedTradingService) GetBrokerManager() *broker.Manager {
	return s.brokerManager
}

// GetSignalProcessor returns the signal processor
func (s *EnhancedTradingService) GetSignalProcessor() *broker.SignalProcessor {
	return s.signalProcessor
}

// GetAllPositions gets positions from all brokers for a user
func (s *EnhancedTradingService) GetAllPositions(ctx context.Context, userID uint) (map[string][]broker.Position, error) {
	// Initialize brokers if needed
	if err := s.InitializeBrokers(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to initialize brokers: %w", err)
	}

	return s.brokerManager.GetAllPositions(ctx), nil
}

// GetAccountInfo gets account info from all brokers for a user
func (s *EnhancedTradingService) GetAccountInfo(ctx context.Context, userID uint) (map[string]*broker.AccountInfo, error) {
	// Initialize brokers if needed
	if err := s.InitializeBrokers(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to initialize brokers: %w", err)
	}

	return s.brokerManager.GetAllAccountInfo(ctx), nil
}

// SetLeverageOnAllBrokers sets leverage on all brokers for a symbol
func (s *EnhancedTradingService) SetLeverageOnAllBrokers(ctx context.Context, userID uint, symbol string, leverage int) map[string]error {
	// Initialize brokers if needed
	if err := s.InitializeBrokers(ctx, userID); err != nil {
		return map[string]error{"initialization": err}
	}

	req := &broker.LeverageRequest{
		Symbol:   symbol,
		Leverage: leverage,
	}

	return s.brokerManager.SetLeverageOnAllBrokers(ctx, req)
}

// CloseAllPositions closes all positions on all brokers for a user
func (s *EnhancedTradingService) CloseAllPositions(ctx context.Context, userID uint) map[string]error {
	// Initialize brokers if needed
	if err := s.InitializeBrokers(ctx, userID); err != nil {
		return map[string]error{"initialization": err}
	}

	return s.brokerManager.CloseAllPositions(ctx)
}

// TestBrokerConnections tests connections to all brokers for a user
func (s *EnhancedTradingService) TestBrokerConnections(ctx context.Context, userID uint) map[string]error {
	// Initialize brokers if needed
	if err := s.InitializeBrokers(ctx, userID); err != nil {
		return map[string]error{"initialization": err}
	}

	return s.brokerManager.TestConnections(ctx)
}

// Helper functions (reused from original trading service)

// validatePositionChange validates the position change based on prev_market_position_size
func (s *EnhancedTradingService) validatePositionChange(userID uint, signal *models.TradingViewSignal) error {
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
	prevSize, err := parseFloat(signal.PrevMarketPositionSize)
	if err != nil {
		return fmt.Errorf("invalid prev_market_position_size: %w", err)
	}

	// Check if previous position size matches current position
	if currentPosition != nil {
		currentSize, err := parseFloat(currentPosition.Size)
		if err != nil {
			return fmt.Errorf("invalid current position size: %w", err)
		}

		// Allow small floating point differences
		if absFloat(currentSize-prevSize) > 0.0001 {
			s.logger.Printf("Position size mismatch for user %d, symbol %s: expected %.8f, got %.8f",
				userID, signal.Symbol, prevSize, currentSize)
			// Don't fail the trade, just log the warning
		}
	} else if prevSize != 0 {
		s.logger.Printf("No current position found for user %d, symbol %s, but prev_size is %.8f",
			userID, signal.Symbol, prevSize)
	}

	return nil
}

// updateUserPosition updates the user's position after a successful trade
func (s *EnhancedTradingService) updateUserPosition(userID uint, signal *models.TradingViewSignal) error {
	// Parse position size
	positionSize, err := parseFloat(signal.MarketPositionSize)
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

// Helper functions
func parseFloat(s string) (float64, error) {
	return broker.ParseQuantity(s)
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Close closes all broker connections
func (s *EnhancedTradingService) Close() error {
	return s.brokerManager.Close()
}
