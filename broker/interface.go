package broker

import (
	"context"
)

// Broker represents a cryptocurrency exchange broker interface
// This interface defines all methods that brokers must implement for futures trading
type Broker interface {
	// Name returns the name of the broker
	Name() string

	// Initialize sets up the broker with credentials
	Initialize(ctx context.Context, credentials *Credentials) error

	// Test connection to the broker
	TestConnection(ctx context.Context) error

	// Account related methods
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
	GetBalance(ctx context.Context, asset string) (*Balance, error)

	// Position related methods
	GetPositions(ctx context.Context) ([]Position, error)
	GetPosition(ctx context.Context, symbol string) (*Position, error)
	SetLeverage(ctx context.Context, req *LeverageRequest) error
	SetMarginType(ctx context.Context, req *MarginTypeRequest) error

	// Order related methods
	PlaceOrder(ctx context.Context, req *OrderRequest) (*Order, error)
	GetOrder(ctx context.Context, symbol string, orderID string) (*Order, error)
	CancelOrder(ctx context.Context, symbol string, orderID string) error
	GetOpenOrders(ctx context.Context, symbol string) ([]Order, error)
	GetOrderHistory(ctx context.Context, symbol string, limit int) ([]Order, error)

	// Market data methods
	GetSymbolInfo(ctx context.Context, symbol string) (*SymbolInfo, error)
	GetExchangeInfo(ctx context.Context) ([]SymbolInfo, error)

	// Utility methods
	IsConnected() bool
	Close() error
}

// FuturesBroker extends the base Broker interface with futures-specific methods
type FuturesBroker interface {
	Broker

	// Futures specific methods
	GetFuturesAccountInfo(ctx context.Context) (*AccountInfo, error)
	GetFuturesPositions(ctx context.Context) ([]Position, error)
	PlaceFuturesOrder(ctx context.Context, req *OrderRequest) (*Order, error)

	// Position management
	ClosePosition(ctx context.Context, symbol string, positionSide PositionSide) error
	CloseAllPositions(ctx context.Context) error

	// Risk management
	SetPositionMode(ctx context.Context, dualSidePosition bool) error
	GetPositionMode(ctx context.Context) (bool, error)
}

// BrokerFactory is a factory function type for creating brokers
type BrokerFactory func() Broker

// Registry holds all registered broker factories
var Registry = make(map[string]BrokerFactory)

// Register registers a broker factory
func Register(name string, factory BrokerFactory) {
	Registry[name] = factory
}

// Create creates a new broker instance by name
func Create(name string) (Broker, error) {
	factory, exists := Registry[name]
	if !exists {
		return nil, ErrBrokerNotFound
	}
	return factory(), nil
}

// GetRegisteredBrokers returns a list of all registered broker names
func GetRegisteredBrokers() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}
