package services

import (
	"fmt"
	"log"
	"time"

	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/database"
	"github.com/Cyvadra/tv-forward/internal/models"
	"gorm.io/gorm"
)

// TradingService handles trading operations
type TradingService struct {
	db     *gorm.DB
	config *config.Config
}

// NewTradingService creates a new trading service
func NewTradingService() *TradingService {
	return &TradingService{
		db:     database.GetDB(),
		config: nil, // Will be set later
	}
}

// SetConfig sets the configuration for the trading service
func (s *TradingService) SetConfig(cfg *config.Config) {
	s.config = cfg
}

// ProcessTradingSignal processes a trading signal and executes orders
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
		if err := s.executeOnBitget(alert, signal); err != nil {
			executionErrors = append(executionErrors, fmt.Sprintf("Bitget: %v", err))
		} else {
			signal.Platform = "bitget"
			signal.Status = "filled"
			now := time.Now()
			signal.ExecutedAt = &now
		}
	}

	// Try Binance
	if signal.Status == "pending" && s.config.Trading.Binance.IsActive {
		if err := s.executeOnBinance(alert, signal); err != nil {
			executionErrors = append(executionErrors, fmt.Sprintf("Binance: %v", err))
		} else {
			signal.Platform = "binance"
			signal.Status = "filled"
			now := time.Now()
			signal.ExecutedAt = &now
		}
	}

	// Try Derbit
	if signal.Status == "pending" && s.config.Trading.Derbit.IsActive {
		if err := s.executeOnDerbit(alert, signal); err != nil {
			executionErrors = append(executionErrors, fmt.Sprintf("Derbit: %v", err))
		} else {
			signal.Platform = "derbit"
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

// executeOnBitget executes a trade on Bitget
func (s *TradingService) executeOnBitget(alert *models.Alert, signal *models.TradingSignal) error {
	// This is a placeholder implementation
	// In a real implementation, you would integrate with Bitget's API
	log.Printf("Executing %s order for %s on Bitget at price %.8f",
		alert.Action, alert.Symbol, alert.Price)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("bitget_%d_%d", alert.ID, time.Now().Unix())

	return nil
}

// executeOnBinance executes a trade on Binance
func (s *TradingService) executeOnBinance(alert *models.Alert, signal *models.TradingSignal) error {
	// This is a placeholder implementation
	// In a real implementation, you would integrate with Binance's API
	log.Printf("Executing %s order for %s on Binance at price %.8f",
		alert.Action, alert.Symbol, alert.Price)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("binance_%d_%d", alert.ID, time.Now().Unix())

	return nil
}

// executeOnDerbit executes a trade on Derbit
func (s *TradingService) executeOnDerbit(alert *models.Alert, signal *models.TradingSignal) error {
	// This is a placeholder implementation
	// In a real implementation, you would integrate with Derbit's API
	log.Printf("Executing %s order for %s on Derbit at price %.8f",
		alert.Action, alert.Symbol, alert.Price)

	// Simulate order execution
	signal.OrderID = fmt.Sprintf("derbit_%d_%d", alert.ID, time.Now().Unix())

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
