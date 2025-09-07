package models

import (
	"time"

	"gorm.io/gorm"
)

// TradingViewSignal represents the structure of a TradingView trading signal
type TradingViewSignal struct {
	Ticker                 string `json:"ticker"`
	Exchange               string `json:"ex"`
	Close                  string `json:"close"`
	Open                   string `json:"open"`
	High                   string `json:"high"`
	Low                    string `json:"low"`
	Time                   string `json:"time"`
	Volume                 string `json:"volume"`
	TimeNow                string `json:"timenow"`
	Interval               string `json:"interval"`
	PositionSize           string `json:"position_size"`
	Action                 string `json:"action"`
	Contracts              string `json:"contracts"`
	Price                  string `json:"price"`
	ID                     string `json:"id"`
	MarketPosition         string `json:"market_position"`
	MarketPositionSize     string `json:"market_position_size"`
	PrevMarketPosition     string `json:"prev_market_position"`
	PrevMarketPositionSize string `json:"prev_market_position_size"`
	ExchangeName           string `json:"exchange"`
	Leverage               int    `json:"lever"`
	TradingMode            string `json:"td_mode"`
	Symbol                 string `json:"symbol"`
	OrderType              string `json:"ord_type"`
	OrderBase              string `json:"ord_base"`
	Amount                 string `json:"amount"`
	StrategyMethod         string `json:"strategy_method"`
	Delay                  int    `json:"delay"`
	SLTPType               string `json:"sltp_type"`
	AID                    string `json:"aid"`
	APISec                 string `json:"api_sec"`
}

// User represents a user account identified by api_sec
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	APISec    string         `json:"api_sec" gorm:"uniqueIndex;not null"`
	Name      string         `json:"name"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	Credentials []UserCredential `json:"credentials" gorm:"foreignKey:UserID"`
	Signals     []TradingSignal  `json:"signals" gorm:"foreignKey:UserID"`
	Positions   []Position       `json:"positions" gorm:"foreignKey:UserID"`
}

// UserCredential represents exchange credentials for a user
type UserCredential struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	UserID     uint           `json:"user_id" gorm:"not null"`
	Exchange   string         `json:"exchange" gorm:"not null"` // bitget, binance, okx
	APIKey     string         `json:"api_key" gorm:"not null"`
	SecretKey  string         `json:"secret_key" gorm:"not null"`
	Passphrase string         `json:"passphrase,omitempty"` // For Bitget
	IsActive   bool           `json:"is_active" gorm:"default:true"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// Position represents a trading position for a user
type Position struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        uint           `json:"user_id" gorm:"not null"`
	Symbol        string         `json:"symbol" gorm:"not null"`
	Exchange      string         `json:"exchange" gorm:"not null"`
	Side          string         `json:"side"` // long, short, flat
	Size          string         `json:"size"`
	EntryPrice    string         `json:"entry_price"`
	MarkPrice     string         `json:"mark_price"`
	UnrealizedPnL string         `json:"unrealized_pnl"`
	Leverage      int            `json:"leverage"`
	TradingMode   string         `json:"trading_mode"` // isolated, cross
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	LastUpdated   time.Time      `json:"last_updated"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// TradingSignal represents a processed trading signal (updated from original)
type TradingSignal struct {
	ID                     uint           `json:"id" gorm:"primaryKey"`
	UserID                 uint           `json:"user_id" gorm:"not null"`
	AlertID                uint           `json:"alert_id,omitempty"`
	Alert                  Alert          `json:"alert,omitempty" gorm:"foreignKey:AlertID"`
	SignalID               string         `json:"signal_id"` // From TradingView signal
	Symbol                 string         `json:"symbol"`
	Exchange               string         `json:"exchange"`
	Action                 string         `json:"action"`
	PositionSize           string         `json:"position_size"`
	Price                  string         `json:"price"`
	MarketPosition         string         `json:"market_position"`
	MarketPositionSize     string         `json:"market_position_size"`
	PrevMarketPosition     string         `json:"prev_market_position"`
	PrevMarketPositionSize string         `json:"prev_market_position_size"`
	Leverage               int            `json:"leverage"`
	TradingMode            string         `json:"trading_mode"`
	OrderType              string         `json:"order_type"`
	OrderID                string         `json:"order_id"`
	Status                 string         `json:"status"` // pending, filled, cancelled, failed
	ErrorMessage           string         `json:"error_message,omitempty"`
	ExecutedAt             *time.Time     `json:"executed_at"`
	RawPayload             string         `json:"raw_payload" gorm:"type:text"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}
