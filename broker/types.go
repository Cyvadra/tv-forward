package broker

import (
	"time"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents the type of an order
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
)

// PositionSide represents the side of a position for futures trading
type PositionSide string

const (
	PositionSideLong  PositionSide = "LONG"
	PositionSideShort PositionSide = "SHORT"
	PositionSideBoth  PositionSide = "BOTH"
)

// MarginType represents the margin type for futures trading
type MarginType string

const (
	MarginTypeIsolated MarginType = "ISOLATED"
	MarginTypeCross    MarginType = "CROSSED"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "NEW"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCanceled        OrderStatus = "CANCELED"
	OrderStatusPendingCancel   OrderStatus = "PENDING_CANCEL"
	OrderStatusRejected        OrderStatus = "REJECTED"
	OrderStatusExpired         OrderStatus = "EXPIRED"
)

// Credentials represents the API credentials for a broker
type Credentials struct {
	APIKey     string `json:"api_key"`
	SecretKey  string `json:"secret_key"`
	Passphrase string `json:"passphrase,omitempty"` // For some exchanges like OKX
}

// OrderRequest represents a request to place an order
type OrderRequest struct {
	Symbol       string       `json:"symbol"`
	Side         OrderSide    `json:"side"`
	Type         OrderType    `json:"type"`
	Quantity     string       `json:"quantity"`
	Price        string       `json:"price,omitempty"`         // Required for limit orders
	PositionSide PositionSide `json:"position_side,omitempty"` // For futures trading
	TimeInForce  string       `json:"time_in_force,omitempty"` // GTC, IOC, FOK
	ReduceOnly   bool         `json:"reduce_only,omitempty"`   // For futures trading
}

// Order represents an order response
type Order struct {
	ID               string       `json:"id"`
	ClientOrderID    string       `json:"client_order_id"`
	Symbol           string       `json:"symbol"`
	Side             OrderSide    `json:"side"`
	Type             OrderType    `json:"type"`
	Quantity         string       `json:"quantity"`
	Price            string       `json:"price"`
	ExecutedQuantity string       `json:"executed_quantity"`
	CumulativeQuote  string       `json:"cumulative_quote"`
	Status           OrderStatus  `json:"status"`
	TimeInForce      string       `json:"time_in_force"`
	PositionSide     PositionSide `json:"position_side"`
	ReduceOnly       bool         `json:"reduce_only"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// Position represents a futures position
type Position struct {
	Symbol            string       `json:"symbol"`
	PositionSide      PositionSide `json:"position_side"`
	Size              string       `json:"size"`
	EntryPrice        string       `json:"entry_price"`
	MarkPrice         string       `json:"mark_price"`
	UnrealizedPnL     string       `json:"unrealized_pnl"`
	Leverage          int          `json:"leverage"`
	MarginType        MarginType   `json:"margin_type"`
	IsolatedMargin    string       `json:"isolated_margin,omitempty"`
	MaintenanceMargin string       `json:"maintenance_margin"`
	InitialMargin     string       `json:"initial_margin"`
	OpenOrderMargin   string       `json:"open_order_margin"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

// Balance represents account balance
type Balance struct {
	Asset                  string `json:"asset"`
	WalletBalance          string `json:"wallet_balance"`
	UnrealizedPnL          string `json:"unrealized_pnl"`
	MarginBalance          string `json:"margin_balance"`
	MaintMargin            string `json:"maint_margin"`
	InitialMargin          string `json:"initial_margin"`
	PositionInitialMargin  string `json:"position_initial_margin"`
	OpenOrderInitialMargin string `json:"open_order_initial_margin"`
	CrossWalletBalance     string `json:"cross_wallet_balance"`
	CrossUnPnl             string `json:"cross_un_pnl"`
	AvailableBalance       string `json:"available_balance"`
	MaxWithdrawAmount      string `json:"max_withdraw_amount"`
}

// AccountInfo represents account information
type AccountInfo struct {
	TotalWalletBalance          string     `json:"total_wallet_balance"`
	TotalUnrealizedPnL          string     `json:"total_unrealized_pnl"`
	TotalMarginBalance          string     `json:"total_margin_balance"`
	TotalPositionInitialMargin  string     `json:"total_position_initial_margin"`
	TotalOpenOrderInitialMargin string     `json:"total_open_order_initial_margin"`
	TotalCrossWalletBalance     string     `json:"total_cross_wallet_balance"`
	TotalCrossUnPnl             string     `json:"total_cross_un_pnl"`
	AvailableBalance            string     `json:"available_balance"`
	MaxWithdrawAmount           string     `json:"max_withdraw_amount"`
	Assets                      []Balance  `json:"assets"`
	Positions                   []Position `json:"positions"`
	CanTrade                    bool       `json:"can_trade"`
	CanWithdraw                 bool       `json:"can_withdraw"`
	FeeTier                     int        `json:"fee_tier"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}

// SymbolInfo represents trading symbol information
type SymbolInfo struct {
	Symbol                     string      `json:"symbol"`
	BaseAsset                  string      `json:"base_asset"`
	QuoteAsset                 string      `json:"quote_asset"`
	Status                     string      `json:"status"`
	BaseAssetPrecision         int         `json:"base_asset_precision"`
	QuoteAssetPrecision        int         `json:"quote_asset_precision"`
	OrderTypes                 []OrderType `json:"order_types"`
	IcebergAllowed             bool        `json:"iceberg_allowed"`
	OcoAllowed                 bool        `json:"oco_allowed"`
	QuoteOrderQtyMarketAllowed bool        `json:"quote_order_qty_market_allowed"`
	AllowTrailingStop          bool        `json:"allow_trailing_stop"`
	CancelReplaceAllowed       bool        `json:"cancel_replace_allowed"`
	IsSpotTradingAllowed       bool        `json:"is_spot_trading_allowed"`
	IsMarginTradingAllowed     bool        `json:"is_margin_trading_allowed"`
	MinQty                     string      `json:"min_qty"`
	MaxQty                     string      `json:"max_qty"`
	StepSize                   string      `json:"step_size"`
	MinPrice                   string      `json:"min_price"`
	MaxPrice                   string      `json:"max_price"`
	TickSize                   string      `json:"tick_size"`
	MinNotional                string      `json:"min_notional"`
	MaxNotional                string      `json:"max_notional"`
}

// LeverageRequest represents a request to change leverage
type LeverageRequest struct {
	Symbol   string `json:"symbol"`
	Leverage int    `json:"leverage"`
}

// MarginTypeRequest represents a request to change margin type
type MarginTypeRequest struct {
	Symbol     string     `json:"symbol"`
	MarginType MarginType `json:"margin_type"`
}
