package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Solana SolanaConfig `mapstructure:"solana"`
	Log    LogConfig    `mapstructure:"log"`
}

// SolanaConfig holds Solana-specific configuration
type SolanaConfig struct {
	RPC     string `mapstructure:"rpc"`
	Network string `mapstructure:"network"`
	Timeout int    `mapstructure:"timeout"` // in seconds
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json or text
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Solana: SolanaConfig{
			RPC:     "https://api.devnet.solana.com",
			Network: "devnet",
			Timeout: 30,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// Load loads configuration from file and environment
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName(".carbon")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
	}

	// Environment variables
	viper.SetEnvPrefix("CARBON")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file (ignore if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// GetRPCEndpoint returns the RPC endpoint for the configured network
func (c *SolanaConfig) GetRPCEndpoint() string {
	if c.RPC != "" {
		return c.RPC
	}

	switch c.Network {
	case "mainnet", "mainnet-beta":
		return "https://api.mainnet-beta.solana.com"
	case "testnet":
		return "https://api.testnet.solana.com"
	case "localnet", "localhost":
		return "http://localhost:8899"
	default:
		return "https://api.devnet.solana.com"
	}
}
