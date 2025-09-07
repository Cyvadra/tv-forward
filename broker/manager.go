package broker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager manages multiple broker connections and provides unified trading operations
type Manager struct {
	brokers map[string]Broker
	mutex   sync.RWMutex
	logger  *log.Logger
}

// NewManager creates a new broker manager
func NewManager() *Manager {
	return &Manager{
		brokers: make(map[string]Broker),
		logger:  log.New(log.Writer(), "[BrokerManager] ", log.LstdFlags),
	}
}

// SetLogger sets a custom logger
func (m *Manager) SetLogger(logger *log.Logger) {
	m.logger = logger
}

// AddBroker adds a broker to the manager
func (m *Manager) AddBroker(name string, broker Broker) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if broker == nil {
		return fmt.Errorf("broker cannot be nil")
	}

	m.brokers[name] = broker
	m.logger.Printf("Added broker: %s", name)
	return nil
}

// RemoveBroker removes a broker from the manager
func (m *Manager) RemoveBroker(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	broker, exists := m.brokers[name]
	if !exists {
		return fmt.Errorf("broker %s not found", name)
	}

	// Close the broker connection
	if err := broker.Close(); err != nil {
		m.logger.Printf("Error closing broker %s: %v", name, err)
	}

	delete(m.brokers, name)
	m.logger.Printf("Removed broker: %s", name)
	return nil
}

// GetBroker retrieves a broker by name
func (m *Manager) GetBroker(name string) (Broker, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	broker, exists := m.brokers[name]
	if !exists {
		return nil, fmt.Errorf("broker %s not found", name)
	}

	return broker, nil
}

// GetBrokers returns all registered brokers
func (m *Manager) GetBrokers() map[string]Broker {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	brokers := make(map[string]Broker)
	for name, broker := range m.brokers {
		brokers[name] = broker
	}

	return brokers
}

// InitializeBroker initializes a broker with credentials
func (m *Manager) InitializeBroker(ctx context.Context, name string, credentials *Credentials) error {
	// Create broker instance
	broker, err := Create(name)
	if err != nil {
		return fmt.Errorf("failed to create broker %s: %w", name, err)
	}

	// Initialize with credentials
	if err := broker.Initialize(ctx, credentials); err != nil {
		return fmt.Errorf("failed to initialize broker %s: %w", name, err)
	}

	// Add to manager
	if err := m.AddBroker(name, broker); err != nil {
		broker.Close() // Clean up on failure
		return fmt.Errorf("failed to add broker %s to manager: %w", name, err)
	}

	return nil
}

// TestConnections tests all broker connections
func (m *Manager) TestConnections(ctx context.Context) map[string]error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	results := make(map[string]error)
	for name, broker := range m.brokers {
		if err := broker.TestConnection(ctx); err != nil {
			results[name] = err
			m.logger.Printf("Connection test failed for %s: %v", name, err)
		} else {
			m.logger.Printf("Connection test passed for %s", name)
		}
	}

	return results
}

// ExecuteOnBroker executes an operation on a specific broker
func (m *Manager) ExecuteOnBroker(ctx context.Context, brokerName string, operation func(Broker) error) error {
	broker, err := m.GetBroker(brokerName)
	if err != nil {
		return err
	}

	if !broker.IsConnected() {
		return fmt.Errorf("broker %s is not connected", brokerName)
	}

	return operation(broker)
}

// ExecuteOnAllBrokers executes an operation on all brokers
func (m *Manager) ExecuteOnAllBrokers(ctx context.Context, operation func(string, Broker) error) map[string]error {
	m.mutex.RLock()
	brokers := make(map[string]Broker)
	for name, broker := range m.brokers {
		brokers[name] = broker
	}
	m.mutex.RUnlock()

	results := make(map[string]error)
	for name, broker := range brokers {
		if !broker.IsConnected() {
			results[name] = fmt.Errorf("broker %s is not connected", name)
			continue
		}

		if err := operation(name, broker); err != nil {
			results[name] = err
		}
	}

	return results
}

// PlaceOrderOnBroker places an order on a specific broker
func (m *Manager) PlaceOrderOnBroker(ctx context.Context, brokerName string, req *OrderRequest) (*Order, error) {
	var result *Order
	err := m.ExecuteOnBroker(ctx, brokerName, func(broker Broker) error {
		order, err := broker.PlaceOrder(ctx, req)
		if err != nil {
			return err
		}
		result = order
		return nil
	})

	return result, err
}

// GetPositionsFromBroker gets positions from a specific broker
func (m *Manager) GetPositionsFromBroker(ctx context.Context, brokerName string) ([]Position, error) {
	var result []Position
	err := m.ExecuteOnBroker(ctx, brokerName, func(broker Broker) error {
		positions, err := broker.GetPositions(ctx)
		if err != nil {
			return err
		}
		result = positions
		return nil
	})

	return result, err
}

// GetAllPositions gets positions from all brokers
func (m *Manager) GetAllPositions(ctx context.Context) map[string][]Position {
	results := make(map[string][]Position)
	errors := m.ExecuteOnAllBrokers(ctx, func(name string, broker Broker) error {
		positions, err := broker.GetPositions(ctx)
		if err != nil {
			return err
		}
		results[name] = positions
		return nil
	})

	// Log any errors
	for name, err := range errors {
		m.logger.Printf("Failed to get positions from %s: %v", name, err)
	}

	return results
}

// CloseAllPositions closes all positions on all brokers
func (m *Manager) CloseAllPositions(ctx context.Context) map[string]error {
	return m.ExecuteOnAllBrokers(ctx, func(name string, broker Broker) error {
		if futuresBroker, ok := broker.(FuturesBroker); ok {
			return futuresBroker.CloseAllPositions(ctx)
		}
		return fmt.Errorf("broker %s does not support futures operations", name)
	})
}

// SetLeverageOnBroker sets leverage on a specific broker
func (m *Manager) SetLeverageOnBroker(ctx context.Context, brokerName string, req *LeverageRequest) error {
	return m.ExecuteOnBroker(ctx, brokerName, func(broker Broker) error {
		return broker.SetLeverage(ctx, req)
	})
}

// SetLeverageOnAllBrokers sets leverage on all brokers
func (m *Manager) SetLeverageOnAllBrokers(ctx context.Context, req *LeverageRequest) map[string]error {
	return m.ExecuteOnAllBrokers(ctx, func(name string, broker Broker) error {
		return broker.SetLeverage(ctx, req)
	})
}

// GetAccountInfoFromBroker gets account info from a specific broker
func (m *Manager) GetAccountInfoFromBroker(ctx context.Context, brokerName string) (*AccountInfo, error) {
	var result *AccountInfo
	err := m.ExecuteOnBroker(ctx, brokerName, func(broker Broker) error {
		accountInfo, err := broker.GetAccountInfo(ctx)
		if err != nil {
			return err
		}
		result = accountInfo
		return nil
	})

	return result, err
}

// GetAllAccountInfo gets account info from all brokers
func (m *Manager) GetAllAccountInfo(ctx context.Context) map[string]*AccountInfo {
	results := make(map[string]*AccountInfo)
	errors := m.ExecuteOnAllBrokers(ctx, func(name string, broker Broker) error {
		accountInfo, err := broker.GetAccountInfo(ctx)
		if err != nil {
			return err
		}
		results[name] = accountInfo
		return nil
	})

	// Log any errors
	for name, err := range errors {
		m.logger.Printf("Failed to get account info from %s: %v", name, err)
	}

	return results
}

// HealthCheck performs health checks on all brokers
func (m *Manager) HealthCheck(ctx context.Context) map[string]bool {
	results := make(map[string]bool)
	errors := m.TestConnections(ctx)

	m.mutex.RLock()
	for name := range m.brokers {
		results[name] = errors[name] == nil
	}
	m.mutex.RUnlock()

	return results
}

// Close closes all broker connections
func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var errors []error
	for name, broker := range m.brokers {
		if err := broker.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close broker %s: %w", name, err))
		}
	}

	// Clear brokers map
	m.brokers = make(map[string]Broker)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing brokers: %v", errors)
	}

	return nil
}

// GetConnectedBrokers returns names of all connected brokers
func (m *Manager) GetConnectedBrokers() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var connected []string
	for name, broker := range m.brokers {
		if broker.IsConnected() {
			connected = append(connected, name)
		}
	}

	return connected
}

// RetryOperation retries an operation with exponential backoff
func (m *Manager) RetryOperation(ctx context.Context, maxRetries int, operation func() error) error {
	return RetryWithBackoff(ctx, maxRetries, time.Second, operation)
}
