# Broker System

This package provides a unified interface for interacting with cryptocurrency exchange brokers for futures trading.

## Features

- **Unified Interface**: Common interface for all brokers
- **Futures Trading Focus**: Designed specifically for futures trading operations
- **Binance Implementation**: Complete implementation using `github.com/adshao/go-binance/v2`
- **Modular Design**: Separate modules for different broker utilities
- **Error Handling**: Comprehensive error handling with broker-specific errors
- **Configuration Management**: Built-in configuration and credential management
- **Signal Processing**: Integration with TradingView signals

## Architecture

### Core Components

1. **Broker Interface** (`interface.go`): Defines the standard interface all brokers must implement
2. **Types** (`types.go`): Common data structures for orders, positions, accounts, etc.
3. **Errors** (`errors.go`): Standardized error handling
4. **Utils** (`utils.go`): Utility functions for validation, formatting, and conversions

### Broker Implementations

- **Binance** (`binance/`): Complete Binance futures trading implementation

### Management Components

1. **Manager** (`manager.go`): Manages multiple broker connections
2. **Config Manager** (`config.go`): Configuration and credential management
3. **Signal Processor** (`integration.go`): Processes TradingView signals
4. **Enhanced Trading Service** (`../internal/services/enhanced_trading.go`): Integration with existing services

## Usage

### Basic Broker Usage

```go
// Create a broker instance
broker, err := broker.Create("binance")
if err != nil {
    log.Fatal(err)
}

// Initialize with credentials
credentials := &broker.Credentials{
    APIKey:    "your-api-key",
    SecretKey: "your-secret-key",
}

err = broker.Initialize(context.Background(), credentials)
if err != nil {
    log.Fatal(err)
}

// Place an order
orderReq := &broker.OrderRequest{
    Symbol:       "BTCUSDT",
    Side:         broker.OrderSideBuy,
    Type:         broker.OrderTypeMarket,
    Quantity:     "0.001",
    PositionSide: broker.PositionSideLong,
}

order, err := broker.PlaceOrder(context.Background(), orderReq)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Order placed: %+v\n", order)
```

### Using the Broker Manager

```go
// Create a manager
manager := broker.NewManager()

// Add multiple brokers
binanceCredentials := &broker.Credentials{
    APIKey:    "binance-api-key",
    SecretKey: "binance-secret-key",
}

err := manager.InitializeBroker(ctx, "binance", binanceCredentials)
if err != nil {
    log.Fatal(err)
}

// Get all positions from all brokers
allPositions := manager.GetAllPositions(ctx)
for exchange, positions := range allPositions {
    fmt.Printf("%s positions: %+v\n", exchange, positions)
}
```

### Processing TradingView Signals

```go
// Create signal processor
manager := broker.NewManager()
processor := broker.NewSignalProcessor(manager)

// Process a signal
signal := &broker.TradingSignal{
    Symbol:                 "BTCUSDT",
    Exchange:               "binance",
    Action:                 "buy",
    MarketPositionSize:     "0.001",
    PrevMarketPositionSize: "0",
    Leverage:               10,
    OrderType:              "market",
}

err := processor.ProcessSignal(ctx, signal)
if err != nil {
    log.Fatal(err)
}
```

### Configuration Management

```go
// Create config
config := &broker.Config{
    Brokers: map[string]broker.BrokerConfig{
        "binance": {
            Enabled: true,
            Credentials: broker.Credentials{
                APIKey:    "your-api-key",
                SecretKey: "your-secret-key",
            },
            Settings: broker.Settings{
                Leverage:     10,
                MarginType:   "cross",
                PositionMode: "hedge",
            },
        },
    },
}

// Create config manager
configManager := broker.NewConfigManager(config)

// Initialize all brokers from config
err := configManager.InitializeBrokers(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Supported Operations

### Account Operations
- Get account information
- Get balance for specific assets
- Test connection

### Position Operations
- Get all positions
- Get specific position
- Set leverage
- Set margin type
- Close position(s)

### Order Operations
- Place orders (market, limit)
- Get order status
- Cancel orders
- Get open orders
- Get order history

### Futures-Specific Operations
- Set position mode (hedge/one-way)
- Place stop loss orders
- Place take profit orders
- Get position risk information

## Error Handling

The broker system provides comprehensive error handling:

```go
// Check for specific error types
if errors.Is(err, broker.ErrNotConnected) {
    // Handle connection error
}

// Check for temporary errors that can be retried
if broker.IsRetryableError(err) {
    // Retry the operation
}

// Check for broker-specific errors
var brokerErr *broker.BrokerError
if errors.As(err, &brokerErr) {
    fmt.Printf("Broker: %s, Code: %s, Message: %s\n", 
        brokerErr.Broker, brokerErr.Code, brokerErr.Message)
}
```

## Testing

Run the tests:

```bash
go test ./broker/... -v
```

For integration tests with real API credentials:

```bash
go test ./broker/... -v -tags=integration
```

## Adding New Brokers

To add a new broker implementation:

1. Create a new directory under `broker/` (e.g., `broker/okx/`)
2. Implement the `Broker` interface
3. Register the broker in the init function:

```go
func init() {
    broker.Register("okx", NewOKXClient)
}
```

4. Add tests for the new implementation

## Security Notes

- Never commit API keys or secrets to version control
- Use environment variables or secure configuration management
- Implement proper credential rotation
- Use testnet/sandbox environments for development

## Dependencies

- `github.com/adshao/go-binance/v2`: Binance API client
- `github.com/stretchr/testify`: Testing framework

## Contributing

1. Follow the existing code structure
2. Implement comprehensive tests
3. Document all public functions
4. Handle errors appropriately
5. Follow Go best practices
