package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Database  DatabaseConfig   `yaml:"database"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
	Trading   TradingConfig    `yaml:"trading"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port string `yaml:"port" default:":8080"`
	Host string `yaml:"host" default:"localhost"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Driver string `yaml:"driver" default:"sqlite"`
	DSN    string `yaml:"dsn" default:"tv-forward.db"`
}

// EndpointConfig represents a downstream endpoint configuration
type EndpointConfig struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"` // telegram, wechat, dingtalk, webhook
	URL      string `yaml:"url"`
	Token    string `yaml:"token,omitempty"`
	ChatID   string `yaml:"chat_id,omitempty"`
	IsActive bool   `yaml:"is_active" default:"true"`
}

// TradingConfig represents trading platform configuration
type TradingConfig struct {
	Bitget  BitgetConfig  `yaml:"bitget"`
	Binance BinanceConfig `yaml:"binance"`
	OKX     OKXConfig     `yaml:"okx"`
	Derbit  DerbitConfig  `yaml:"derbit"`
}

// BitgetConfig represents Bitget trading platform configuration
type BitgetConfig struct {
	APIKey     string `yaml:"api_key"`
	SecretKey  string `yaml:"secret_key"`
	Passphrase string `yaml:"passphrase"`
	IsActive   bool   `yaml:"is_active" default:"false"`
}

// BinanceConfig represents Binance trading platform configuration
type BinanceConfig struct {
	APIKey    string `yaml:"api_key"`
	SecretKey string `yaml:"secret_key"`
	IsActive  bool   `yaml:"is_active" default:"false"`
}

// OKXConfig represents OKX trading platform configuration
type OKXConfig struct {
	APIKey     string `yaml:"api_key"`
	SecretKey  string `yaml:"secret_key"`
	Passphrase string `yaml:"passphrase"`
	IsActive   bool   `yaml:"is_active" default:"false"`
}

// DerbitConfig represents Derbit trading platform configuration
type DerbitConfig struct {
	APIKey    string `yaml:"api_key"`
	SecretKey string `yaml:"secret_key"`
	IsActive  bool   `yaml:"is_active" default:"false"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
