package binance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Cyvadra/tv-forward/broker"
	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

const FLAG_USE_TESTNET = true

// Client represents a Binance futures broker client
type Client struct {
	name        string
	client      *futures.Client
	credentials *broker.Credentials
	connected   bool
}

// NewClient creates a new Binance futures client
func NewClient() broker.Broker {
	return &Client{
		name:      "binance",
		connected: false,
	}
}

// Name returns the broker name
func (c *Client) Name() string {
	return c.name
}

// Initialize sets up the client with credentials
func (c *Client) Initialize(ctx context.Context, credentials *broker.Credentials) error {
	if credentials == nil {
		return broker.ErrInvalidCredentials
	}

	if credentials.APIKey == "" || credentials.SecretKey == "" {
		return broker.NewBrokerError(c.name, "INVALID_CREDENTIALS", "API key and secret key are required", broker.ErrInvalidCredentials)
	}

	c.credentials = credentials
	binance.UseTestnet = FLAG_USE_TESTNET
	c.client = binance.NewFuturesClient(credentials.APIKey, credentials.SecretKey)

	// Test connection
	if err := c.TestConnection(ctx); err != nil {
		return fmt.Errorf("failed to initialize Binance client: %w", err)
	}

	c.connected = true
	return nil
}

// TestConnection tests the connection to Binance
func (c *Client) TestConnection(ctx context.Context) error {
	if c.client == nil {
		return broker.ErrNotConnected
	}

	// Test connectivity by getting server time
	_, err := c.client.NewServerTimeService().Do(ctx)
	if err != nil {
		return broker.NewBrokerError(c.name, "CONNECTION_FAILED", "Failed to connect to Binance", err)
	}

	return nil
}

// GetAccountInfo retrieves account information
func (c *Client) GetAccountInfo(ctx context.Context) (*broker.AccountInfo, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	account, err := c.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "ACCOUNT_INFO_FAILED", "Failed to get account info", err)
	}

	// Convert Binance account to broker AccountInfo
	accountInfo := &broker.AccountInfo{
		TotalWalletBalance:          account.TotalWalletBalance,
		TotalMarginBalance:          account.TotalMarginBalance,
		TotalPositionInitialMargin:  account.TotalPositionInitialMargin,
		TotalOpenOrderInitialMargin: account.TotalOpenOrderInitialMargin,
		TotalCrossWalletBalance:     account.TotalCrossWalletBalance,
		AvailableBalance:            account.AvailableBalance,
		MaxWithdrawAmount:           account.MaxWithdrawAmount,
		CanTrade:                    account.CanTrade,
		CanWithdraw:                 account.CanWithdraw,
		FeeTier:                     int(account.FeeTier),
		UpdatedAt:                   time.Unix(account.UpdateTime/1000, 0),
	}

	// Convert assets
	for _, asset := range account.Assets {
		accountInfo.Assets = append(accountInfo.Assets, broker.Balance{
			Asset:                  asset.Asset,
			WalletBalance:          asset.WalletBalance,
			MarginBalance:          asset.MarginBalance,
			MaintMargin:            asset.MaintMargin,
			InitialMargin:          asset.InitialMargin,
			PositionInitialMargin:  asset.PositionInitialMargin,
			OpenOrderInitialMargin: asset.OpenOrderInitialMargin,
			CrossWalletBalance:     asset.CrossWalletBalance,
			AvailableBalance:       asset.AvailableBalance,
			MaxWithdrawAmount:      asset.MaxWithdrawAmount,
		})
	}

	// Convert positions
	for _, pos := range account.Positions {
		if pos.PositionAmt != "0" { // Only include non-zero positions
			accountInfo.Positions = append(accountInfo.Positions, broker.Position{
				Symbol:       pos.Symbol,
				PositionSide: convertPositionSideFromString(string(pos.PositionSide)),
				Size:         pos.PositionAmt,
				EntryPrice:   pos.EntryPrice,
				Leverage:     int(parseFloatOrZero(pos.Leverage)),
				MarginType:   broker.MarginTypeCross, // Default since MarginType field may not exist
				UpdatedAt:    time.Unix(pos.UpdateTime/1000, 0),
			})
		}
	}

	return accountInfo, nil
}

// GetBalance retrieves balance for a specific asset
func (c *Client) GetBalance(ctx context.Context, asset string) (*broker.Balance, error) {
	accountInfo, err := c.GetAccountInfo(ctx)
	if err != nil {
		return nil, err
	}

	for _, balance := range accountInfo.Assets {
		if balance.Asset == asset {
			return &balance, nil
		}
	}

	return nil, broker.NewBrokerError(c.name, "ASSET_NOT_FOUND", fmt.Sprintf("Asset %s not found", asset), nil)
}

// GetPositions retrieves all positions
func (c *Client) GetPositions(ctx context.Context) ([]broker.Position, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	positions, err := c.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "POSITIONS_FAILED", "Failed to get positions", err)
	}

	var result []broker.Position
	for _, pos := range positions {
		if pos.PositionAmt != "0" { // Only include non-zero positions
			result = append(result, broker.Position{
				Symbol:       pos.Symbol,
				PositionSide: convertPositionSideFromString(string(pos.PositionSide)),
				Size:         pos.PositionAmt,
				EntryPrice:   pos.EntryPrice,
				Leverage:     int(parseFloatOrZero(pos.Leverage)),
				MarginType:   convertMarginTypeFromString(string(pos.MarginType)),
				UpdatedAt:    time.Now(),
			})
		}
	}

	return result, nil
}

// GetPosition retrieves a specific position
func (c *Client) GetPosition(ctx context.Context, symbol string) (*broker.Position, error) {
	positions, err := c.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	for _, pos := range positions {
		if pos.Symbol == symbol {
			return &pos, nil
		}
	}

	return nil, broker.ErrPositionNotFound
}

// SetLeverage sets leverage for a symbol
func (c *Client) SetLeverage(ctx context.Context, req *broker.LeverageRequest) error {
	if !c.connected {
		return broker.ErrNotConnected
	}

	if !broker.IsValidLeverage(req.Leverage) {
		return broker.ErrInvalidLeverage
	}

	_, err := c.client.NewChangeLeverageService().
		Symbol(req.Symbol).
		Leverage(req.Leverage).
		Do(ctx)

	if err != nil {
		return broker.NewBrokerError(c.name, "LEVERAGE_FAILED", "Failed to set leverage", err)
	}

	return nil
}

// SetMarginType sets margin type for a symbol
func (c *Client) SetMarginType(ctx context.Context, req *broker.MarginTypeRequest) error {
	if !c.connected {
		return broker.ErrNotConnected
	}

	var marginType futures.MarginType
	switch req.MarginType {
	case broker.MarginTypeIsolated:
		marginType = futures.MarginTypeIsolated
	case broker.MarginTypeCross:
		marginType = futures.MarginTypeCrossed
	default:
		return broker.ErrInvalidMarginType
	}

	err := c.client.NewChangeMarginTypeService().
		Symbol(req.Symbol).
		MarginType(marginType).
		Do(ctx)

	if err != nil {
		return broker.NewBrokerError(c.name, "MARGIN_TYPE_FAILED", "Failed to set margin type", err)
	}

	return nil
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(ctx context.Context, req *broker.OrderRequest) (*broker.Order, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	if err := broker.ValidateOrderRequest(req); err != nil {
		return nil, err
	}

	service := c.client.NewCreateOrderService().
		Symbol(req.Symbol).
		Side(convertToBinanceSide(req.Side)).
		Type(convertToBinanceOrderType(req.Type)).
		Quantity(req.Quantity)

	// Set position side if specified
	if req.PositionSide != "" {
		service = service.PositionSide(convertToBinancePositionSide(req.PositionSide))
	}

	// Set price for limit orders
	if req.Type == broker.OrderTypeLimit && req.Price != "" {
		service = service.Price(req.Price)
	}

	// Set time in force
	if req.TimeInForce != "" {
		service = service.TimeInForce(futures.TimeInForceType(req.TimeInForce))
	} else if req.Type == broker.OrderTypeLimit {
		service = service.TimeInForce(futures.TimeInForceTypeGTC) // Default to GTC for limit orders
	}

	// Set reduce only
	if req.ReduceOnly {
		service = service.ReduceOnly(req.ReduceOnly)
	}

	order, err := service.Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "ORDER_FAILED", "Failed to place order", err)
	}

	return convertBinanceOrder(order), nil
}

// GetOrder retrieves an order by ID
func (c *Client) GetOrder(ctx context.Context, symbol string, orderID string) (*broker.Order, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "INVALID_ORDER_ID", "Invalid order ID", err)
	}

	order, err := c.client.NewGetOrderService().
		Symbol(symbol).
		OrderID(id).
		Do(ctx)

	if err != nil {
		return nil, broker.NewBrokerError(c.name, "ORDER_NOT_FOUND", "Failed to get order", err)
	}

	return convertBinanceOrderFromGet(order), nil
}

// CancelOrder cancels an order
func (c *Client) CancelOrder(ctx context.Context, symbol string, orderID string) error {
	if !c.connected {
		return broker.ErrNotConnected
	}

	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return broker.NewBrokerError(c.name, "INVALID_ORDER_ID", "Invalid order ID", err)
	}

	_, err = c.client.NewCancelOrderService().
		Symbol(symbol).
		OrderID(id).
		Do(ctx)

	if err != nil {
		return broker.NewBrokerError(c.name, "CANCEL_FAILED", "Failed to cancel order", err)
	}

	return nil
}

// GetOpenOrders retrieves open orders for a symbol
func (c *Client) GetOpenOrders(ctx context.Context, symbol string) ([]broker.Order, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	service := c.client.NewListOpenOrdersService()
	if symbol != "" {
		service = service.Symbol(symbol)
	}

	orders, err := service.Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "OPEN_ORDERS_FAILED", "Failed to get open orders", err)
	}

	var result []broker.Order
	for _, order := range orders {
		result = append(result, *convertBinanceOrderFromGet(order))
	}

	return result, nil
}

// GetOrderHistory retrieves order history for a symbol
func (c *Client) GetOrderHistory(ctx context.Context, symbol string, limit int) ([]broker.Order, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	service := c.client.NewListOrdersService().Symbol(symbol)
	if limit > 0 {
		service = service.Limit(limit)
	}

	orders, err := service.Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "ORDER_HISTORY_FAILED", "Failed to get order history", err)
	}

	var result []broker.Order
	for _, order := range orders {
		result = append(result, *convertBinanceOrderFromGet(order))
	}

	return result, nil
}

// GetSymbolInfo retrieves symbol information
func (c *Client) GetSymbolInfo(ctx context.Context, symbol string) (*broker.SymbolInfo, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	exchangeInfo, err := c.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "EXCHANGE_INFO_FAILED", "Failed to get exchange info", err)
	}

	for _, s := range exchangeInfo.Symbols {
		if s.Symbol == symbol {
			return convertBinanceSymbolInfo(&s), nil
		}
	}

	return nil, broker.ErrInvalidSymbol
}

// GetExchangeInfo retrieves exchange information
func (c *Client) GetExchangeInfo(ctx context.Context) ([]broker.SymbolInfo, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	exchangeInfo, err := c.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "EXCHANGE_INFO_FAILED", "Failed to get exchange info", err)
	}

	var result []broker.SymbolInfo
	for _, s := range exchangeInfo.Symbols {
		result = append(result, *convertBinanceSymbolInfo(&s))
	}

	return result, nil
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	return c.connected
}

// Close closes the client connection
func (c *Client) Close() error {
	c.connected = false
	c.client = nil
	return nil
}

// Helper functions

func parseFloatOrZero(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func convertPositionSideFromString(side string) broker.PositionSide {
	switch strings.ToUpper(side) {
	case "LONG":
		return broker.PositionSideLong
	case "SHORT":
		return broker.PositionSideShort
	case "BOTH":
		return broker.PositionSideBoth
	default:
		return broker.PositionSideBoth
	}
}

func convertToBinancePositionSide(side broker.PositionSide) futures.PositionSideType {
	switch side {
	case broker.PositionSideLong:
		return futures.PositionSideTypeLong
	case broker.PositionSideShort:
		return futures.PositionSideTypeShort
	case broker.PositionSideBoth:
		return futures.PositionSideTypeBoth
	default:
		return futures.PositionSideTypeBoth
	}
}

func convertMarginTypeFromString(marginType string) broker.MarginType {
	switch strings.ToUpper(marginType) {
	case "ISOLATED":
		return broker.MarginTypeIsolated
	case "CROSS", "CROSSED":
		return broker.MarginTypeCross
	default:
		return broker.MarginTypeCross
	}
}

func convertToBinanceSide(side broker.OrderSide) futures.SideType {
	switch side {
	case broker.OrderSideBuy:
		return futures.SideTypeBuy
	case broker.OrderSideSell:
		return futures.SideTypeSell
	default:
		return futures.SideTypeBuy
	}
}

func convertToBinanceOrderType(orderType broker.OrderType) futures.OrderType {
	switch orderType {
	case broker.OrderTypeMarket:
		return futures.OrderTypeMarket
	case broker.OrderTypeLimit:
		return futures.OrderTypeLimit
	default:
		return futures.OrderTypeMarket
	}
}

func convertBinanceOrder(order *futures.CreateOrderResponse) *broker.Order {
	return &broker.Order{
		ID:               strconv.FormatInt(order.OrderID, 10),
		ClientOrderID:    order.ClientOrderID,
		Symbol:           order.Symbol,
		Side:             convertFromBinanceSide(order.Side),
		Type:             convertFromBinanceOrderType(order.Type),
		Quantity:         "0", // Will be filled from actual API response
		Price:            order.Price,
		ExecutedQuantity: "0", // Will be filled from actual API response
		CumulativeQuote:  "0", // Will be filled from actual API response
		Status:           convertBinanceOrderStatus(order.Status),
		TimeInForce:      string(order.TimeInForce),
		PositionSide:     convertPositionSideFromString(string(order.PositionSide)),
		ReduceOnly:       order.ReduceOnly,
		CreatedAt:        time.Unix(order.UpdateTime/1000, 0),
		UpdatedAt:        time.Unix(order.UpdateTime/1000, 0),
	}
}

func convertBinanceOrderFromGet(order *futures.Order) *broker.Order {
	return &broker.Order{
		ID:               strconv.FormatInt(order.OrderID, 10),
		ClientOrderID:    order.ClientOrderID,
		Symbol:           order.Symbol,
		Side:             convertFromBinanceSide(order.Side),
		Type:             convertFromBinanceOrderType(order.Type),
		Quantity:         "0", // Will be filled from actual API response
		Price:            order.Price,
		ExecutedQuantity: "0", // Will be filled from actual API response
		CumulativeQuote:  "0", // Will be filled from actual API response
		Status:           convertBinanceOrderStatus(order.Status),
		TimeInForce:      string(order.TimeInForce),
		PositionSide:     convertPositionSideFromString(string(order.PositionSide)),
		ReduceOnly:       order.ReduceOnly,
		CreatedAt:        time.Unix(order.Time/1000, 0),
		UpdatedAt:        time.Unix(order.UpdateTime/1000, 0),
	}
}

func convertFromBinanceSide(side futures.SideType) broker.OrderSide {
	switch side {
	case futures.SideTypeBuy:
		return broker.OrderSideBuy
	case futures.SideTypeSell:
		return broker.OrderSideSell
	default:
		return broker.OrderSideBuy
	}
}

func convertFromBinanceOrderType(orderType futures.OrderType) broker.OrderType {
	switch orderType {
	case futures.OrderTypeMarket:
		return broker.OrderTypeMarket
	case futures.OrderTypeLimit:
		return broker.OrderTypeLimit
	default:
		return broker.OrderTypeMarket
	}
}

func convertBinanceOrderStatus(status futures.OrderStatusType) broker.OrderStatus {
	switch status {
	case futures.OrderStatusTypeNew:
		return broker.OrderStatusNew
	case futures.OrderStatusTypePartiallyFilled:
		return broker.OrderStatusPartiallyFilled
	case futures.OrderStatusTypeFilled:
		return broker.OrderStatusFilled
	case futures.OrderStatusTypeCanceled:
		return broker.OrderStatusCanceled
	case futures.OrderStatusTypeRejected:
		return broker.OrderStatusRejected
	case futures.OrderStatusTypeExpired:
		return broker.OrderStatusExpired
	default:
		return broker.OrderStatusNew
	}
}

func convertBinanceSymbolInfo(s *futures.Symbol) *broker.SymbolInfo {
	symbolInfo := &broker.SymbolInfo{
		Symbol:     s.Symbol,
		BaseAsset:  s.BaseAsset,
		QuoteAsset: s.QuoteAsset,
		Status:     string(s.Status),
	}

	// Convert order types
	for _, ot := range s.OrderType {
		switch ot {
		case futures.OrderTypeLimit:
			symbolInfo.OrderTypes = append(symbolInfo.OrderTypes, broker.OrderTypeLimit)
		case futures.OrderTypeMarket:
			symbolInfo.OrderTypes = append(symbolInfo.OrderTypes, broker.OrderTypeMarket)
		}
	}

	// Parse filters for min/max values
	for _, filter := range s.Filters {
		switch filter["filterType"] {
		case "LOT_SIZE":
			if minQty, ok := filter["minQty"].(string); ok {
				symbolInfo.MinQty = minQty
			}
			if maxQty, ok := filter["maxQty"].(string); ok {
				symbolInfo.MaxQty = maxQty
			}
			if stepSize, ok := filter["stepSize"].(string); ok {
				symbolInfo.StepSize = stepSize
			}
		case "PRICE_FILTER":
			if minPrice, ok := filter["minPrice"].(string); ok {
				symbolInfo.MinPrice = minPrice
			}
			if maxPrice, ok := filter["maxPrice"].(string); ok {
				symbolInfo.MaxPrice = maxPrice
			}
			if tickSize, ok := filter["tickSize"].(string); ok {
				symbolInfo.TickSize = tickSize
			}
		case "MIN_NOTIONAL":
			if notional, ok := filter["notional"].(string); ok {
				symbolInfo.MinNotional = notional
			}
		}
	}

	return symbolInfo
}

// Register the Binance broker
func init() {
	broker.Register("binance", NewClient)
}
