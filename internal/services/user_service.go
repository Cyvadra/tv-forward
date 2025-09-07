package services

import (
	"fmt"
	"log"

	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/database"
	"github.com/Cyvadra/tv-forward/internal/models"
	"gorm.io/gorm"
)

// UserService handles user-related operations
type UserService struct {
	db         *gorm.DB
	userConfig *config.UserConfig
}

// NewUserService creates a new user service
func NewUserService() *UserService {
	return &UserService{
		db: database.GetDB(),
	}
}

// SetUserConfig sets the user configuration
func (s *UserService) SetUserConfig(cfg *config.UserConfig) {
	s.userConfig = cfg
}

// GetOrCreateUserByAPISec gets or creates a user by api_sec
func (s *UserService) GetOrCreateUserByAPISec(apiSec string) (*models.User, error) {
	var user models.User

	// Try to find existing user
	err := s.db.Where("api_sec = ?", apiSec).First(&user).Error
	if err == nil {
		return &user, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// User not found, create new one
	user = models.User{
		APISec:   apiSec,
		Name:     fmt.Sprintf("User_%s", apiSec[:8]), // Default name
		IsActive: true,
	}

	// Check if we have config for this user
	if s.userConfig != nil {
		if userConfig := s.userConfig.GetUserByAPISec(apiSec); userConfig != nil {
			user.Name = userConfig.Name
			user.IsActive = userConfig.IsActive
		}
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create credentials if available in config
	if s.userConfig != nil {
		if userConfig := s.userConfig.GetUserByAPISec(apiSec); userConfig != nil {
			for _, credConfig := range userConfig.Credentials {
				credential := models.UserCredential{
					UserID:     user.ID,
					Exchange:   credConfig.Exchange,
					APIKey:     credConfig.APIKey,
					SecretKey:  credConfig.SecretKey,
					Passphrase: credConfig.Passphrase,
					IsActive:   credConfig.IsActive,
				}

				if err := s.db.Create(&credential).Error; err != nil {
					log.Printf("Failed to create credential for user %s, exchange %s: %v",
						apiSec, credConfig.Exchange, err)
				}
			}
		}
	}

	return &user, nil
}

// GetUserCredentials returns active credentials for a user and exchange
func (s *UserService) GetUserCredentials(userID uint, exchange string) (*models.UserCredential, error) {
	var credential models.UserCredential
	err := s.db.Where("user_id = ? AND exchange = ? AND is_active = ?",
		userID, exchange, true).First(&credential).Error
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

// GetUserPositions returns current positions for a user
func (s *UserService) GetUserPositions(userID uint) ([]models.Position, error) {
	var positions []models.Position
	err := s.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&positions).Error
	return positions, err
}

// UpdatePosition updates or creates a position for a user
func (s *UserService) UpdatePosition(userID uint, symbol, exchange, side, size, entryPrice, markPrice, unrealizedPnL string, leverage int, tradingMode string) error {
	var position models.Position

	// Try to find existing position
	err := s.db.Where("user_id = ? AND symbol = ? AND exchange = ? AND is_active = ?",
		userID, symbol, exchange, true).First(&position).Error

	if err == gorm.ErrRecordNotFound {
		// Create new position
		position = models.Position{
			UserID:        userID,
			Symbol:        symbol,
			Exchange:      exchange,
			Side:          side,
			Size:          size,
			EntryPrice:    entryPrice,
			MarkPrice:     markPrice,
			UnrealizedPnL: unrealizedPnL,
			Leverage:      leverage,
			TradingMode:   tradingMode,
			IsActive:      true,
		}
		return s.db.Create(&position).Error
	} else if err != nil {
		return fmt.Errorf("failed to query position: %w", err)
	}

	// Update existing position
	position.Side = side
	position.Size = size
	position.EntryPrice = entryPrice
	position.MarkPrice = markPrice
	position.UnrealizedPnL = unrealizedPnL
	position.Leverage = leverage
	position.TradingMode = tradingMode
	position.LastUpdated = s.db.NowFunc()

	return s.db.Save(&position).Error
}

// ClosePosition closes a position by setting is_active to false
func (s *UserService) ClosePosition(userID uint, symbol, exchange string) error {
	return s.db.Model(&models.Position{}).
		Where("user_id = ? AND symbol = ? AND exchange = ? AND is_active = ?",
			userID, symbol, exchange, true).
		Update("is_active", false).Error
}

// GetUserTradingSignals returns trading signals for a user
func (s *UserService) GetUserTradingSignals(userID uint, limit int) ([]models.TradingSignal, error) {
	var signals []models.TradingSignal
	query := s.db.Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&signals).Error
	return signals, err
}
