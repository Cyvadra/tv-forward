package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// UserConfig represents user-specific configuration
type UserConfig struct {
	Users []UserConfigEntry `yaml:"users"`
}

// UserConfigEntry represents a single user configuration
type UserConfigEntry struct {
	APISec      string                 `yaml:"api_sec"`
	Name        string                 `yaml:"name"`
	IsActive    bool                   `yaml:"is_active" default:"true"`
	Credentials []UserCredentialConfig `yaml:"credentials"`
}

// UserCredentialConfig represents exchange credentials for a user
type UserCredentialConfig struct {
	Exchange   string `yaml:"exchange"` // bitget, binance, okx
	APIKey     string `yaml:"api_key"`
	SecretKey  string `yaml:"secret_key"`
	Passphrase string `yaml:"passphrase,omitempty"` // For Bitget
	IsActive   bool   `yaml:"is_active" default:"true"`
}

// LoadUserConfig loads user configuration from a YAML file
func LoadUserConfig(filename string) (*UserConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read user config file: %w", err)
	}

	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse user config file: %w", err)
	}

	return &config, nil
}

// SaveUserConfig saves user configuration to a YAML file
func SaveUserConfig(config *UserConfig, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal user config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write user config file: %w", err)
	}

	return nil
}

// GetUserByAPISec finds a user configuration by api_sec
func (uc *UserConfig) GetUserByAPISec(apiSec string) *UserConfigEntry {
	for i := range uc.Users {
		if uc.Users[i].APISec == apiSec {
			return &uc.Users[i]
		}
	}
	return nil
}

// GetCredentialsForExchange returns credentials for a specific exchange
func (uce *UserConfigEntry) GetCredentialsForExchange(exchange string) *UserCredentialConfig {
	for i := range uce.Credentials {
		if uce.Credentials[i].Exchange == exchange && uce.Credentials[i].IsActive {
			return &uce.Credentials[i]
		}
	}
	return nil
}
