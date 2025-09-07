package broker

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Config represents broker configuration
type Config struct {
	Brokers map[string]BrokerConfig `yaml:"brokers" json:"brokers"`
	Default DefaultConfig           `yaml:"default" json:"default"`
}

// BrokerConfig represents configuration for a specific broker
type BrokerConfig struct {
	Enabled     bool        `yaml:"enabled" json:"enabled"`
	Credentials Credentials `yaml:"credentials" json:"credentials"`
	Settings    Settings    `yaml:"settings" json:"settings"`
}

// Settings represents broker-specific settings
type Settings struct {
	Leverage       int           `yaml:"leverage" json:"leverage"`
	MarginType     string        `yaml:"margin_type" json:"margin_type"`
	PositionMode   string        `yaml:"position_mode" json:"position_mode"` // hedge or one-way
	TestMode       bool          `yaml:"test_mode" json:"test_mode"`
	RetryAttempts  int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay     time.Duration `yaml:"retry_delay" json:"retry_delay"`
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`
	RateLimitDelay time.Duration `yaml:"rate_limit_delay" json:"rate_limit_delay"`
}

// DefaultConfig represents default settings
type DefaultConfig struct {
	Leverage       int           `yaml:"leverage" json:"leverage"`
	MarginType     string        `yaml:"margin_type" json:"margin_type"`
	PositionMode   string        `yaml:"position_mode" json:"position_mode"`
	RetryAttempts  int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay     time.Duration `yaml:"retry_delay" json:"retry_delay"`
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`
}

// ConfigManager manages broker configurations
type ConfigManager struct {
	config  *Config
	manager *Manager
	logger  *log.Logger
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(config *Config) *ConfigManager {
	return &ConfigManager{
		config:  config,
		manager: NewManager(),
		logger:  log.New(log.Writer(), "[ConfigManager] ", log.LstdFlags),
	}
}

// SetLogger sets a custom logger
func (cm *ConfigManager) SetLogger(logger *log.Logger) {
	cm.logger = logger
	cm.manager.SetLogger(logger)
}

// GetManager returns the broker manager
func (cm *ConfigManager) GetManager() *Manager {
	return cm.manager
}

// InitializeBrokers initializes all enabled brokers from configuration
func (cm *ConfigManager) InitializeBrokers(ctx context.Context) error {
	if cm.config == nil {
		return fmt.Errorf("configuration is nil")
	}

	var errors []error
	for name, brokerConfig := range cm.config.Brokers {
		if !brokerConfig.Enabled {
			cm.logger.Printf("Skipping disabled broker: %s", name)
			continue
		}

		cm.logger.Printf("Initializing broker: %s", name)
		if err := cm.initializeBroker(ctx, name, &brokerConfig); err != nil {
			cm.logger.Printf("Failed to initialize broker %s: %v", name, err)
			errors = append(errors, fmt.Errorf("failed to initialize %s: %w", name, err))
			continue
		}

		cm.logger.Printf("Successfully initialized broker: %s", name)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to initialize some brokers: %v", errors)
	}

	return nil
}

// initializeBroker initializes a single broker
func (cm *ConfigManager) initializeBroker(ctx context.Context, name string, config *BrokerConfig) error {
	// Validate credentials
	if err := cm.validateCredentials(&config.Credentials); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	// Initialize broker with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, cm.getRequestTimeout(config))
	defer cancel()

	if err := cm.manager.InitializeBroker(timeoutCtx, name, &config.Credentials); err != nil {
		return err
	}

	// Apply broker-specific settings
	if err := cm.applyBrokerSettings(ctx, name, config); err != nil {
		cm.logger.Printf("Warning: Failed to apply settings for %s: %v", name, err)
		// Don't fail initialization for settings errors
	}

	return nil
}

// validateCredentials validates broker credentials
func (cm *ConfigManager) validateCredentials(creds *Credentials) error {
	if creds.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if creds.SecretKey == "" {
		return fmt.Errorf("secret key is required")
	}

	return nil
}

// applyBrokerSettings applies broker-specific settings
func (cm *ConfigManager) applyBrokerSettings(ctx context.Context, name string, config *BrokerConfig) error {
	broker, err := cm.manager.GetBroker(name)
	if err != nil {
		return err
	}

	// Apply leverage settings if specified
	if config.Settings.Leverage > 0 {
		// This would need to be applied per symbol, so we'll skip for now
		cm.logger.Printf("Leverage setting for %s will be applied per symbol", name)
	}

	// Apply position mode for futures brokers
	if futuresBroker, ok := broker.(FuturesBroker); ok {
		if config.Settings.PositionMode != "" {
			dualSide := config.Settings.PositionMode == "hedge"
			if err := futuresBroker.SetPositionMode(ctx, dualSide); err != nil {
				return fmt.Errorf("failed to set position mode: %w", err)
			}
			cm.logger.Printf("Set position mode for %s: %s", name, config.Settings.PositionMode)
		}
	}

	return nil
}

// getRequestTimeout returns the request timeout for a broker
func (cm *ConfigManager) getRequestTimeout(config *BrokerConfig) time.Duration {
	if config.Settings.RequestTimeout > 0 {
		return config.Settings.RequestTimeout
	}
	if cm.config.Default.RequestTimeout > 0 {
		return cm.config.Default.RequestTimeout
	}
	return 30 * time.Second // Default timeout
}

// GetBrokerConfig returns configuration for a specific broker
func (cm *ConfigManager) GetBrokerConfig(name string) (*BrokerConfig, error) {
	config, exists := cm.config.Brokers[name]
	if !exists {
		return nil, fmt.Errorf("broker %s not found in configuration", name)
	}

	return &config, nil
}

// UpdateBrokerConfig updates configuration for a specific broker
func (cm *ConfigManager) UpdateBrokerConfig(name string, config *BrokerConfig) error {
	if cm.config.Brokers == nil {
		cm.config.Brokers = make(map[string]BrokerConfig)
	}

	cm.config.Brokers[name] = *config
	cm.logger.Printf("Updated configuration for broker: %s", name)
	return nil
}

// TestAllConnections tests connections to all configured brokers
func (cm *ConfigManager) TestAllConnections(ctx context.Context) map[string]error {
	return cm.manager.TestConnections(ctx)
}

// GetHealthStatus returns health status of all brokers
func (cm *ConfigManager) GetHealthStatus(ctx context.Context) map[string]BrokerHealth {
	results := make(map[string]BrokerHealth)

	for name := range cm.config.Brokers {
		health := BrokerHealth{
			Name:      name,
			Connected: false,
			Error:     nil,
		}

		broker, err := cm.manager.GetBroker(name)
		if err != nil {
			health.Error = err
		} else {
			health.Connected = broker.IsConnected()
			if health.Connected {
				// Test connection
				if err := broker.TestConnection(ctx); err != nil {
					health.Connected = false
					health.Error = err
				}
			}
		}

		results[name] = health
	}

	return results
}

// BrokerHealth represents the health status of a broker
type BrokerHealth struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	Error     error  `json:"error,omitempty"`
}

// ReconnectBroker attempts to reconnect a specific broker
func (cm *ConfigManager) ReconnectBroker(ctx context.Context, name string) error {
	config, err := cm.GetBrokerConfig(name)
	if err != nil {
		return err
	}

	if !config.Enabled {
		return fmt.Errorf("broker %s is disabled", name)
	}

	// Remove existing broker
	if err := cm.manager.RemoveBroker(name); err != nil {
		cm.logger.Printf("Warning: Failed to remove existing broker %s: %v", name, err)
	}

	// Reinitialize
	return cm.initializeBroker(ctx, name, config)
}

// Close closes all broker connections
func (cm *ConfigManager) Close() error {
	return cm.manager.Close()
}

// GetEnabledBrokers returns a list of enabled broker names
func (cm *ConfigManager) GetEnabledBrokers() []string {
	var enabled []string
	for name, config := range cm.config.Brokers {
		if config.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// ValidateConfig validates the broker configuration
func (cm *ConfigManager) ValidateConfig() error {
	if cm.config == nil {
		return fmt.Errorf("configuration is nil")
	}

	if len(cm.config.Brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}

	for name, brokerConfig := range cm.config.Brokers {
		if brokerConfig.Enabled {
			if err := cm.validateCredentials(&brokerConfig.Credentials); err != nil {
				return fmt.Errorf("invalid credentials for broker %s: %w", name, err)
			}

			// Validate settings
			if brokerConfig.Settings.Leverage < 0 || brokerConfig.Settings.Leverage > 125 {
				return fmt.Errorf("invalid leverage for broker %s: must be between 1 and 125", name)
			}

			if brokerConfig.Settings.MarginType != "" &&
				brokerConfig.Settings.MarginType != "isolated" &&
				brokerConfig.Settings.MarginType != "cross" {
				return fmt.Errorf("invalid margin type for broker %s: must be 'isolated' or 'cross'", name)
			}

			if brokerConfig.Settings.PositionMode != "" &&
				brokerConfig.Settings.PositionMode != "hedge" &&
				brokerConfig.Settings.PositionMode != "one-way" {
				return fmt.Errorf("invalid position mode for broker %s: must be 'hedge' or 'one-way'", name)
			}
		}
	}

	return nil
}
