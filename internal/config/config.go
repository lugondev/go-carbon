package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Solana   SolanaConfig   `mapstructure:"solana"`
	Log      LogConfig      `mapstructure:"log"`
	Database DatabaseConfig `mapstructure:"database"`
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

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Enabled  bool           `mapstructure:"enabled"`
	Type     string         `mapstructure:"type"` // mongodb, postgres, or mysql
	MongoDB  MongoDBConfig  `mapstructure:"mongodb"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
}

// MongoDBConfig holds MongoDB-specific configuration
type MongoDBConfig struct {
	URI            string `mapstructure:"uri"`
	Database       string `mapstructure:"database"`
	MaxPoolSize    uint64 `mapstructure:"max_pool_size"`
	MinPoolSize    uint64 `mapstructure:"min_pool_size"`
	ConnectTimeout int    `mapstructure:"connect_timeout"` // in seconds
}

// PostgresConfig holds PostgreSQL-specific configuration
type PostgresConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type MySQLConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
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
		Database: DatabaseConfig{
			Enabled: false,
			Type:    "mongodb",
			MongoDB: MongoDBConfig{
				URI:            "mongodb://localhost:27017",
				Database:       "carbon",
				MaxPoolSize:    100,
				MinPoolSize:    10,
				ConnectTimeout: 10,
			},
			Postgres: PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				User:            "carbon",
				Password:        "",
				Database:        "carbon",
				SSLMode:         "disable",
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 300,
			},
			MySQL: MySQLConfig{
				Host:            "localhost",
				Port:            3306,
				User:            "carbon",
				Password:        "",
				Database:        "carbon",
				SSLMode:         "false",
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 300,
			},
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
