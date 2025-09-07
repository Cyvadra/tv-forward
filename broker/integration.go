package broker

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// TradingSignal represents a trading signal from TradingView
type TradingSignal struct {
	Symbol                 string `json:"symbol"`
	Exchange               string `json:"exchange"`
	Action                 string `json:"action"`
	PositionSize           string `json:"position_size"`
	Price                  string `json:"price"`
	MarketPosition         string `json:"market_position"`
	MarketPositionSize     string `json:"market_position_size"`
	PrevMarketPosition     string `json:"prev_market_position"`
	PrevMarketPositionSize string `json:"prev_market_position_size"`
	Leverage               int    `json:"leverage"`
	TradingMode            string `json:"trading_mode"`
	OrderType              string `json:"order_type"`
}

// SignalProcessor processes trading signals and executes orders
type SignalProcessor struct {
	manager *Manager
	logger  *log.Logger
}

// NewSignalProcessor creates a new signal processor
func NewSignalProcessor(manager *Manager) *SignalProcessor {
	return &SignalProcessor{
		manager: manager,
		logger:  log.New(log.Writer(), "[SignalProcessor] ", log.LstdFlags),
	}
}

// SetLogger sets a custom logger
func (sp *SignalProcessor) SetLogger(logger *log.Logger) {
	sp.logger = logger
}

// ProcessSignal processes a trading signal and executes the corresponding order
func (sp *SignalProcessor) ProcessSignal(ctx context.Context, signal *TradingSignal) error {
	sp.logger.Printf("Processing signal: %s %s on %s", signal.Action, signal.Symbol, signal.Exchange)

	// Validate signal
	if err := sp.validateSignal(signal); err != nil {
		return fmt.Errorf("signal validation failed: %w", err)
	}

	// Get broker
	broker, err := sp.manager.GetBroker(strings.ToLower(signal.Exchange))
	if err != nil {
		return fmt.Errorf("failed to get broker %s: %w", signal.Exchange, err)
	}

	// Check if broker is connected
	if !broker.IsConnected() {
		return fmt.Errorf("broker %s is not connected", signal.Exchange)
	}

	// Set leverage if needed
	if signal.Leverage > 0 {
		if err := sp.setLeverage(ctx, broker, signal); err != nil {
			sp.logger.Printf("Warning: Failed to set leverage: %v", err)
			// Don't fail the entire operation for leverage setting
		}
	}

	// Calculate position change
	orderReq, err := sp.calculateOrderFromSignal(signal)
	if err != nil {
		return fmt.Errorf("failed to calculate order: %w", err)
	}

	if orderReq == nil {
		sp.logger.Printf("No order needed for signal: %s", signal.Symbol)
		return nil
	}

	// Execute order
	order, err := broker.PlaceOrder(ctx, orderReq)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	sp.logger.Printf("Order placed successfully: ID=%s, Symbol=%s, Side=%s, Quantity=%s",
		order.ID, order.Symbol, order.Side, order.Quantity)

	return nil
}

// validateSignal validates a trading signal
func (sp *SignalProcessor) validateSignal(signal *TradingSignal) error {
	if signal == nil {
		return fmt.Errorf("signal is nil")
	}

	if signal.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if signal.Exchange == "" {
		return fmt.Errorf("exchange is required")
	}

	if signal.Action == "" {
		return fmt.Errorf("action is required")
	}

	if signal.MarketPositionSize == "" {
		return fmt.Errorf("market position size is required")
	}

	return nil
}

// setLeverage sets leverage for the symbol
func (sp *SignalProcessor) setLeverage(ctx context.Context, broker Broker, signal *TradingSignal) error {
	if signal.Leverage <= 0 {
		return nil
	}

	leverageReq := &LeverageRequest{
		Symbol:   FormatSymbol(signal.Symbol, strings.ToLower(signal.Exchange)),
		Leverage: signal.Leverage,
	}

	return broker.SetLeverage(ctx, leverageReq)
}

// calculateOrderFromSignal calculates the order needed based on the signal
func (sp *SignalProcessor) calculateOrderFromSignal(signal *TradingSignal) (*OrderRequest, error) {
	// Parse current and target position sizes
	currentSize, err := strconv.ParseFloat(signal.PrevMarketPositionSize, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid prev_market_position_size: %w", err)
	}

	targetSize, err := strconv.ParseFloat(signal.MarketPositionSize, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid market_position_size: %w", err)
	}

	// Calculate order quantity and side
	quantity, side, err := CalculateOrderQuantity(currentSize, targetSize)
	if err != nil {
		return nil, err
	}

	// Determine position side for futures
	var positionSide PositionSide
	if targetSize > 0 {
		positionSide = PositionSideLong
	} else if targetSize < 0 {
		positionSide = PositionSideShort
	} else {
		positionSide = PositionSideBoth
	}

	// Determine order type
	var orderType OrderType
	var price string
	if signal.OrderType == "limit" && signal.Price != "" {
		orderType = OrderTypeLimit
		price = signal.Price
	} else {
		orderType = OrderTypeMarket
	}

	// Create order request
	orderReq := &OrderRequest{
		Symbol:       FormatSymbol(signal.Symbol, strings.ToLower(signal.Exchange)),
		Side:         side,
		Type:         orderType,
		Quantity:     FormatQuantity(quantity, 8),
		Price:        price,
		PositionSide: positionSide,
		TimeInForce:  "GTC",
	}

	// Set reduce only for closing positions
	if (currentSize > 0 && targetSize < currentSize) || (currentSize < 0 && targetSize > currentSize) {
		orderReq.ReduceOnly = true
	}

	return orderReq, nil
}

// ProcessMultipleSignals processes multiple signals concurrently
func (sp *SignalProcessor) ProcessMultipleSignals(ctx context.Context, signals []*TradingSignal) []error {
	results := make([]error, len(signals))

	// Process signals concurrently
	type result struct {
		index int
		err   error
	}

	resultChan := make(chan result, len(signals))

	for i, signal := range signals {
		go func(index int, sig *TradingSignal) {
			err := sp.ProcessSignal(ctx, sig)
			resultChan <- result{index: index, err: err}
		}(i, signal)
	}

	// Collect results
	for i := 0; i < len(signals); i++ {
		res := <-resultChan
		results[res.index] = res.err
	}

	return results
}

// GetPositionSummary gets a summary of all positions across all brokers
func (sp *SignalProcessor) GetPositionSummary(ctx context.Context) (map[string]map[string]Position, error) {
	allPositions := sp.manager.GetAllPositions(ctx)

	// Organize by symbol then by exchange
	summary := make(map[string]map[string]Position)

	for exchange, positions := range allPositions {
		for _, position := range positions {
			if _, exists := summary[position.Symbol]; !exists {
				summary[position.Symbol] = make(map[string]Position)
			}
			summary[position.Symbol][exchange] = position
		}
	}

	return summary, nil
}

// SyncPositions synchronizes positions across all brokers for a symbol
func (sp *SignalProcessor) SyncPositions(ctx context.Context, symbol string, targetPosition Position) map[string]error {
	brokers := sp.manager.GetBrokers()
	results := make(map[string]error)

	for name, broker := range brokers {
		if !broker.IsConnected() {
			results[name] = fmt.Errorf("broker not connected")
			continue
		}

		// Get current position
		currentPos, err := broker.GetPosition(ctx, symbol)
		if err != nil && err != ErrPositionNotFound {
			results[name] = fmt.Errorf("failed to get current position: %w", err)
			continue
		}

		var currentSize float64
		if currentPos != nil {
			currentSize, err = ParseQuantity(currentPos.Size)
			if err != nil {
				results[name] = fmt.Errorf("invalid current position size: %w", err)
				continue
			}
		}

		targetSize, err := ParseQuantity(targetPosition.Size)
		if err != nil {
			results[name] = fmt.Errorf("invalid target position size: %w", err)
			continue
		}

		// Calculate order needed
		quantity, side, err := CalculateOrderQuantity(currentSize, targetSize)
		if err != nil {
			if err.Error() == "no position change required" {
				continue // Skip if no change needed
			}
			results[name] = fmt.Errorf("failed to calculate order: %w", err)
			continue
		}

		// Place order
		orderReq := &OrderRequest{
			Symbol:       symbol,
			Side:         side,
			Type:         OrderTypeMarket,
			Quantity:     FormatQuantity(quantity, 8),
			PositionSide: targetPosition.PositionSide,
		}

		_, err = broker.PlaceOrder(ctx, orderReq)
		if err != nil {
			results[name] = fmt.Errorf("failed to place sync order: %w", err)
		}
	}

	return results
}
