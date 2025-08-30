package models

import (
	"time"

	"gorm.io/gorm"
)

// Alert represents a TradingView alert
type Alert struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	Strategy   string         `json:"strategy"`
	Symbol     string         `json:"symbol"`
	Action     string         `json:"action"` // buy, sell, close
	Price      float64        `json:"price"`
	Quantity   float64        `json:"quantity"`
	Message    string         `json:"message"`
	RawPayload string         `json:"raw_payload" gorm:"type:text"`
	Status     string         `json:"status" gorm:"default:'received'"` // received, processed, failed
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TradingSignal represents a processed trading signal
type TradingSignal struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	AlertID    uint           `json:"alert_id"`
	Alert      Alert          `json:"alert" gorm:"foreignKey:AlertID"`
	Platform   string         `json:"platform"` // bitget, binance, derbit
	OrderID    string         `json:"order_id"`
	Status     string         `json:"status"` // pending, filled, cancelled, failed
	ExecutedAt *time.Time     `json:"executed_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// DownstreamEndpoint represents a webhook endpoint configuration
type DownstreamEndpoint struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name"`
	Type      string         `json:"type"` // telegram, wechat, dingtalk, webhook
	URL       string         `json:"url"`
	Token     string         `json:"token"`
	ChatID    string         `json:"chat_id"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
