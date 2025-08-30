package services

import (
	"github.com/Cyvadra/tv-forward/internal/database"
	"github.com/Cyvadra/tv-forward/internal/models"
	"gorm.io/gorm"
)

// AlertService handles alert-related operations
type AlertService struct {
	db *gorm.DB
}

// NewAlertService creates a new alert service
func NewAlertService() *AlertService {
	return &AlertService{
		db: database.GetDB(),
	}
}

// SaveAlert saves an alert to the database
func (s *AlertService) SaveAlert(alert *models.Alert) error {
	return s.db.Create(alert).Error
}

// GetAlert retrieves an alert by ID
func (s *AlertService) GetAlert(id uint) (*models.Alert, error) {
	var alert models.Alert
	if err := s.db.First(&alert, id).Error; err != nil {
		return nil, err
	}
	return &alert, nil
}

// GetAlerts retrieves alerts with pagination and optional status filter
func (s *AlertService) GetAlerts(page, limit int, status string) ([]models.Alert, int64, error) {
	var alerts []models.Alert
	var total int64

	query := s.db.Model(&models.Alert{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&alerts).Error; err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

// UpdateAlertStatus updates the status of an alert
func (s *AlertService) UpdateAlertStatus(id uint, status string) error {
	return s.db.Model(&models.Alert{}).Where("id = ?", id).Update("status", status).Error
}

// GetAlertsByStrategy retrieves alerts by strategy name
func (s *AlertService) GetAlertsByStrategy(strategy string, limit int) ([]models.Alert, error) {
	var alerts []models.Alert
	err := s.db.Where("strategy = ?", strategy).
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts).Error
	return alerts, err
}

// GetAlertsBySymbol retrieves alerts by symbol
func (s *AlertService) GetAlertsBySymbol(symbol string, limit int) ([]models.Alert, error) {
	var alerts []models.Alert
	err := s.db.Where("symbol = ?", symbol).
		Order("created_at DESC").
		Limit(limit).
		Find(&alerts).Error
	return alerts, err
}
