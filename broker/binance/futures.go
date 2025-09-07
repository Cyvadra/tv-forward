package binance

import (
	"context"
	"fmt"
	"time"

	"github.com/Cyvadra/tv-forward/broker"
	"github.com/adshao/go-binance/v2/futures"
)

// GetFuturesAccountInfo retrieves futures account information
func (c *Client) GetFuturesAccountInfo(ctx context.Context) (*broker.AccountInfo, error) {
	// For Binance futures, this is the same as GetAccountInfo
	return c.GetAccountInfo(ctx)
}

// GetFuturesPositions retrieves all futures positions
func (c *Client) GetFuturesPositions(ctx context.Context) ([]broker.Position, error) {
	// For Binance futures, this is the same as GetPositions
	return c.GetPositions(ctx)
}

// PlaceFuturesOrder places a futures order
func (c *Client) PlaceFuturesOrder(ctx context.Context, req *broker.OrderRequest) (*broker.Order, error) {
	// For Binance futures, this is the same as PlaceOrder
	return c.PlaceOrder(ctx, req)
}

// ClosePosition closes a specific position
func (c *Client) ClosePosition(ctx context.Context, symbol string, positionSide broker.PositionSide) error {
	if !c.connected {
		return broker.ErrNotConnected
	}

	// Get current position to determine quantity to close
	position, err := c.GetPosition(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get position: %w", err)
	}

	// Parse position size
	positionSize, err := broker.ParseQuantity(position.Size)
	if err != nil {
		return fmt.Errorf("invalid position size: %w", err)
	}

	if positionSize == 0 {
		return nil // Position is already closed
	}

	// Determine order side to close position
	var orderSide broker.OrderSide
	if positionSize > 0 {
		orderSide = broker.OrderSideSell // Close long position
	} else {
		orderSide = broker.OrderSideBuy // Close short position
		positionSize = -positionSize    // Make quantity positive
	}

	// Create close order request
	closeReq := &broker.OrderRequest{
		Symbol:       symbol,
		Side:         orderSide,
		Type:         broker.OrderTypeMarket,
		Quantity:     broker.FormatQuantity(positionSize, 8),
		PositionSide: positionSide,
		ReduceOnly:   true,
	}

	_, err = c.PlaceOrder(ctx, closeReq)
	if err != nil {
		return broker.NewBrokerError(c.name, "CLOSE_POSITION_FAILED", "Failed to close position", err)
	}

	return nil
}

// CloseAllPositions closes all open positions
func (c *Client) CloseAllPositions(ctx context.Context) error {
	if !c.connected {
		return broker.ErrNotConnected
	}

	positions, err := c.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	var errors []error
	for _, position := range positions {
		if err := c.ClosePosition(ctx, position.Symbol, position.PositionSide); err != nil {
			errors = append(errors, fmt.Errorf("failed to close position %s: %w", position.Symbol, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to close some positions: %v", errors)
	}

	return nil
}

// SetPositionMode sets the position mode (hedge or one-way)
func (c *Client) SetPositionMode(ctx context.Context, dualSidePosition bool) error {
	if !c.connected {
		return broker.ErrNotConnected
	}

	err := c.client.NewChangePositionModeService().
		DualSide(dualSidePosition).
		Do(ctx)

	if err != nil {
		return broker.NewBrokerError(c.name, "POSITION_MODE_FAILED", "Failed to set position mode", err)
	}

	return nil
}

// GetPositionMode gets the current position mode
func (c *Client) GetPositionMode(ctx context.Context) (bool, error) {
	if !c.connected {
		return false, broker.ErrNotConnected
	}

	result, err := c.client.NewGetPositionModeService().Do(ctx)
	if err != nil {
		return false, broker.NewBrokerError(c.name, "GET_POSITION_MODE_FAILED", "Failed to get position mode", err)
	}

	return result.DualSidePosition, nil
}

// Additional Binance-specific futures methods

// ChangeInitialLeverage changes the initial leverage for a symbol
func (c *Client) ChangeInitialLeverage(ctx context.Context, symbol string, leverage int) error {
	if !c.connected {
		return broker.ErrNotConnected
	}

	if !broker.IsValidLeverage(leverage) {
		return broker.ErrInvalidLeverage
	}

	_, err := c.client.NewChangeLeverageService().
		Symbol(symbol).
		Leverage(leverage).
		Do(ctx)

	if err != nil {
		return broker.NewBrokerError(c.name, "LEVERAGE_CHANGE_FAILED", "Failed to change leverage", err)
	}

	return nil
}

// GetPositionRisk gets position risk information
func (c *Client) GetPositionRisk(ctx context.Context, symbol string) ([]broker.Position, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	service := c.client.NewGetPositionRiskService()
	if symbol != "" {
		service = service.Symbol(symbol)
	}

	positions, err := service.Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "POSITION_RISK_FAILED", "Failed to get position risk", err)
	}

	var result []broker.Position
	for _, pos := range positions {
		result = append(result, broker.Position{
			Symbol:       pos.Symbol,
			PositionSide: convertPositionSideFromString(string(pos.PositionSide)),
			Size:         pos.PositionAmt,
			EntryPrice:   pos.EntryPrice,
			Leverage:     int(parseFloatOrZero(pos.Leverage)),
			MarginType:   convertMarginTypeFromString(string(pos.MarginType)),
			UpdatedAt:    time.Now(), // UpdateTime field may not be available
		})
	}

	return result, nil
}

// GetIncomeHistory gets income history (funding fees, realized PnL, etc.)
func (c *Client) GetIncomeHistory(ctx context.Context, symbol string, incomeType string, limit int) ([]*futures.IncomeHistory, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	service := c.client.NewGetIncomeHistoryService()
	if symbol != "" {
		service = service.Symbol(symbol)
	}
	if incomeType != "" {
		service = service.IncomeType(incomeType)
	}
	if limit > 0 {
		service = service.Limit(int64(limit))
	}

	income, err := service.Do(ctx)
	if err != nil {
		return nil, broker.NewBrokerError(c.name, "INCOME_HISTORY_FAILED", "Failed to get income history", err)
	}

	return income, nil
}

// PlaceStopLossOrder places a stop loss order
func (c *Client) PlaceStopLossOrder(ctx context.Context, symbol string, side broker.OrderSide, quantity string, stopPrice string, positionSide broker.PositionSide) (*broker.Order, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	order, err := c.client.NewCreateOrderService().
		Symbol(symbol).
		Side(convertToBinanceSide(side)).
		Type(futures.OrderTypeStopMarket).
		Quantity(quantity).
		StopPrice(stopPrice).
		PositionSide(convertToBinancePositionSide(positionSide)).
		ReduceOnly(true).
		TimeInForce(futures.TimeInForceTypeGTC).
		Do(ctx)

	if err != nil {
		return nil, broker.NewBrokerError(c.name, "STOP_LOSS_FAILED", "Failed to place stop loss order", err)
	}

	return convertBinanceOrder(order), nil
}

// PlaceTakeProfitOrder places a take profit order
func (c *Client) PlaceTakeProfitOrder(ctx context.Context, symbol string, side broker.OrderSide, quantity string, stopPrice string, positionSide broker.PositionSide) (*broker.Order, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	order, err := c.client.NewCreateOrderService().
		Symbol(symbol).
		Side(convertToBinanceSide(side)).
		Type(futures.OrderTypeTakeProfitMarket).
		Quantity(quantity).
		StopPrice(stopPrice).
		PositionSide(convertToBinancePositionSide(positionSide)).
		ReduceOnly(true).
		TimeInForce(futures.TimeInForceTypeGTC).
		Do(ctx)

	if err != nil {
		return nil, broker.NewBrokerError(c.name, "TAKE_PROFIT_FAILED", "Failed to place take profit order", err)
	}

	return convertBinanceOrder(order), nil
}

// GetTradingStatus gets current trading status
func (c *Client) GetTradingStatus(ctx context.Context) (map[string]interface{}, error) {
	if !c.connected {
		return nil, broker.ErrNotConnected
	}

	// Note: This would need to be implemented based on actual Binance API
	// For now, return a placeholder, TODO
	return map[string]interface{}{
		"isLocked":           false,
		"plannedRecoverTime": 0,
		"triggerCondition": map[string]interface{}{
			"GCR":  150,
			"IFER": 150,
			"UFR":  300,
		},
		"updateTime": time.Now().Unix(),
	}, nil
}
