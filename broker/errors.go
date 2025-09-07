package broker

import (
	"errors"
	"fmt"
)

// Common broker errors
var (
	ErrBrokerNotFound      = errors.New("broker not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrNotConnected        = errors.New("broker not connected")
	ErrInvalidSymbol       = errors.New("invalid symbol")
	ErrInvalidOrderType    = errors.New("invalid order type")
	ErrInvalidOrderSide    = errors.New("invalid order side")
	ErrInvalidQuantity     = errors.New("invalid quantity")
	ErrInvalidPrice        = errors.New("invalid price")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrOrderNotFound       = errors.New("order not found")
	ErrPositionNotFound    = errors.New("position not found")
	ErrMarketClosed        = errors.New("market is closed")
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrAPIError            = errors.New("API error")
	ErrNetworkError        = errors.New("network error")
	ErrTimeout             = errors.New("request timeout")
	ErrInvalidLeverage     = errors.New("invalid leverage")
	ErrInvalidMarginType   = errors.New("invalid margin type")
)

// BrokerError represents a broker-specific error
type BrokerError struct {
	Broker  string `json:"broker"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *BrokerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s (%v)", e.Broker, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Broker, e.Code, e.Message)
}

func (e *BrokerError) Unwrap() error {
	return e.Err
}

// NewBrokerError creates a new broker error
func NewBrokerError(broker, code, message string, err error) *BrokerError {
	return &BrokerError{
		Broker:  broker,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsTemporaryError checks if an error is temporary (network, rate limit, etc.)
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Check for known temporary errors
	if errors.Is(err, ErrRateLimitExceeded) ||
		errors.Is(err, ErrNetworkError) ||
		errors.Is(err, ErrTimeout) {
		return true
	}

	// Check for broker-specific temporary errors
	var brokerErr *BrokerError
	if errors.As(err, &brokerErr) {
		switch brokerErr.Code {
		case "RATE_LIMIT", "NETWORK_ERROR", "TIMEOUT", "SERVER_ERROR":
			return true
		}
	}

	return false
}

// IsRetryableError checks if an error should be retried
func IsRetryableError(err error) bool {
	return IsTemporaryError(err)
}
