package broker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatSymbol formats a symbol according to exchange requirements
func FormatSymbol(symbol string, exchange string) string {
	symbol = strings.ToUpper(symbol)

	switch exchange {
	case "binance":
		// Binance uses BTCUSDT format
		return symbol
	case "bitget":
		// Bitget uses BTCUSDT format for futures
		return symbol
	case "okx":
		// OKX uses BTC-USDT format for futures
		if !strings.Contains(symbol, "-") {
			// Convert BTCUSDT to BTC-USDT
			if strings.HasSuffix(symbol, "USDT") {
				base := symbol[:len(symbol)-4]
				return base + "-USDT"
			}
		}
		return symbol
	default:
		return symbol
	}
}

// ParseQuantity parses a quantity string to float64
func ParseQuantity(quantity string) (float64, error) {
	if quantity == "" {
		return 0, ErrInvalidQuantity
	}

	qty, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidQuantity, quantity)
	}

	if qty <= 0 {
		return 0, fmt.Errorf("%w: quantity must be positive", ErrInvalidQuantity)
	}

	return qty, nil
}

// ParsePrice parses a price string to float64
func ParsePrice(price string) (float64, error) {
	if price == "" {
		return 0, ErrInvalidPrice
	}

	p, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidPrice, price)
	}

	if p <= 0 {
		return 0, fmt.Errorf("%w: price must be positive", ErrInvalidPrice)
	}

	return p, nil
}

// FormatQuantity formats a quantity for API requests
func FormatQuantity(quantity float64, precision int) string {
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, quantity)
}

// FormatPrice formats a price for API requests
func FormatPrice(price float64, precision int) string {
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, price)
}

// ValidateOrderRequest validates an order request
func ValidateOrderRequest(req *OrderRequest) error {
	if req == nil {
		return fmt.Errorf("order request is nil")
	}

	if req.Symbol == "" {
		return ErrInvalidSymbol
	}

	if req.Side != OrderSideBuy && req.Side != OrderSideSell {
		return ErrInvalidOrderSide
	}

	if req.Type != OrderTypeMarket && req.Type != OrderTypeLimit {
		return ErrInvalidOrderType
	}

	if _, err := ParseQuantity(req.Quantity); err != nil {
		return err
	}

	if req.Type == OrderTypeLimit {
		if req.Price == "" {
			return fmt.Errorf("%w: price required for limit orders", ErrInvalidPrice)
		}
		if _, err := ParsePrice(req.Price); err != nil {
			return err
		}
	}

	return nil
}

// ConvertOrderSideToPositionSide converts order side to position side for futures
func ConvertOrderSideToPositionSide(orderSide OrderSide, positionMode string) PositionSide {
	if positionMode == "hedge" {
		if orderSide == OrderSideBuy {
			return PositionSideLong
		}
		return PositionSideShort
	}
	return PositionSideBoth
}

// CalculateOrderQuantity calculates order quantity based on position change
func CalculateOrderQuantity(currentSize, targetSize float64) (float64, OrderSide, error) {
	diff := targetSize - currentSize

	if diff == 0 {
		return 0, "", fmt.Errorf("no position change required")
	}

	quantity := diff
	side := OrderSideBuy

	if diff < 0 {
		quantity = -diff
		side = OrderSideSell
	}

	return quantity, side, nil
}

// RetryWithBackoff executes a function with exponential backoff
func RetryWithBackoff(ctx context.Context, maxRetries int, baseDelay time.Duration, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1)) // Exponential backoff
			if delay > time.Minute {
				delay = time.Minute // Cap at 1 minute
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if it's not a retryable error
		if !IsRetryableError(err) {
			break
		}

		// Don't retry on the last attempt
		if attempt == maxRetries {
			break
		}
	}

	return lastErr
}

// IsValidLeverage checks if leverage value is valid
func IsValidLeverage(leverage int) bool {
	return leverage >= 1 && leverage <= 125
}

// NormalizeSymbol normalizes symbol format (removes common variations)
func NormalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(symbol)
	symbol = strings.ReplaceAll(symbol, "-", "")
	symbol = strings.ReplaceAll(symbol, "_", "")
	symbol = strings.ReplaceAll(symbol, "/", "")
	return symbol
}

// GetOppositeOrderSide returns the opposite order side
func GetOppositeOrderSide(side OrderSide) OrderSide {
	if side == OrderSideBuy {
		return OrderSideSell
	}
	return OrderSideBuy
}

// GetOppositePositionSide returns the opposite position side
func GetOppositePositionSide(side PositionSide) PositionSide {
	switch side {
	case PositionSideLong:
		return PositionSideShort
	case PositionSideShort:
		return PositionSideLong
	default:
		return PositionSideBoth
	}
}
