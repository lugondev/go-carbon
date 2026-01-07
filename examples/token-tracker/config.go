package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gagliardetto/solana-go"
	"gopkg.in/yaml.v3"
)

type Config struct {
	RPC         RPCConfig         `yaml:"rpc"`
	Accounts    AccountsConfig    `yaml:"accounts"`
	TrackByMint TrackByMintConfig `yaml:"track_by_mint"`
	Mints       MintsConfig       `yaml:"target_mints"`
	Alerts      AlertsConfig      `yaml:"alerts"`
	Metrics     MetricsConfig     `yaml:"metrics"`
	Pipeline    PipelineConfig    `yaml:"pipeline"`
	Logging     LoggingConfig     `yaml:"logging"`
	Advanced    AdvancedConfig    `yaml:"advanced"`
}

type RPCConfig struct {
	Endpoint     string `yaml:"endpoint"`
	PollInterval int    `yaml:"poll_interval"`
	Timeout      int    `yaml:"timeout"`
}

func (c *RPCConfig) GetPollInterval() time.Duration {
	return time.Duration(c.PollInterval) * time.Second
}

func (c *RPCConfig) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

type AccountsConfig []string

func (c AccountsConfig) GetPublicKeys() ([]solana.PublicKey, error) {
	var pubkeys []solana.PublicKey
	for _, addr := range c {
		if addr == "" {
			continue
		}
		pk, err := solana.PublicKeyFromBase58(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid account address %s: %w", addr, err)
		}
		pubkeys = append(pubkeys, pk)
	}
	return pubkeys, nil
}

type MintsConfig struct {
	Enabled bool     `yaml:"enabled"`
	Mints   []string `yaml:"mints"`
}

func (c *MintsConfig) GetMintMap() (map[string]bool, error) {
	if !c.Enabled {
		return nil, nil
	}

	mintMap := make(map[string]bool)
	for _, mint := range c.Mints {
		if mint == "" {
			continue
		}
		pk, err := solana.PublicKeyFromBase58(mint)
		if err != nil {
			return nil, fmt.Errorf("invalid mint address %s: %w", mint, err)
		}
		mintMap[pk.String()] = true
	}
	return mintMap, nil
}

type TrackByMintConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Mints           []string `yaml:"mints"`
	MaxAccounts     int      `yaml:"max_accounts"`
	RefreshInterval int      `yaml:"refresh_interval"`
}

func (c *TrackByMintConfig) GetPollInterval() time.Duration {
	return time.Duration(c.RefreshInterval) * time.Second
}

func (c *TrackByMintConfig) GetMints() ([]solana.PublicKey, error) {
	var mints []solana.PublicKey
	for _, mint := range c.Mints {
		if mint == "" {
			continue
		}
		pk, err := solana.PublicKeyFromBase58(mint)
		if err != nil {
			return nil, fmt.Errorf("invalid mint address %s: %w", mint, err)
		}
		mints = append(mints, pk)
	}
	return mints, nil
}

type AlertsConfig struct {
	Enabled           bool  `yaml:"enabled"`
	Threshold         int64 `yaml:"threshold"`
	AlertNewAccounts  bool  `yaml:"alert_new_accounts"`
	AlertStateChanges bool  `yaml:"alert_state_changes"`
}

type MetricsConfig struct {
	FlushInterval int          `yaml:"flush_interval"`
	Backend       string       `yaml:"backend"`
	Track         MetricsTrack `yaml:"track"`
}

func (c *MetricsConfig) GetFlushInterval() time.Duration {
	return time.Duration(c.FlushInterval) * time.Second
}

type MetricsTrack struct {
	TokenUpdates         bool `yaml:"token_updates"`
	MintUpdates          bool `yaml:"mint_updates"`
	SignificantTransfers bool `yaml:"significant_transfers"`
	AccountBalances      bool `yaml:"account_balances"`
}

type PipelineConfig struct {
	BufferSize       int  `yaml:"buffer_size"`
	GracefulShutdown bool `yaml:"graceful_shutdown"`
	ShutdownTimeout  int  `yaml:"shutdown_timeout"`
}

func (c *PipelineConfig) GetShutdownTimeout() time.Duration {
	return time.Duration(c.ShutdownTimeout) * time.Second
}

type LoggingConfig struct {
	Level      string        `yaml:"level"`
	Format     string        `yaml:"format"`
	Structured bool          `yaml:"structured"`
	Verbose    VerboseConfig `yaml:"verbose"`
}

type VerboseConfig struct {
	IncludeSlot      bool `yaml:"include_slot"`
	IncludeTimestamp bool `yaml:"include_timestamp"`
	IncludePubkey    bool `yaml:"include_pubkey"`
}

type AdvancedConfig struct {
	WorkerCount  int                `yaml:"worker_count"`
	Experimental ExperimentalConfig `yaml:"experimental"`
}

type ExperimentalConfig struct {
	TrackMintSupply       bool `yaml:"track_mint_supply"`
	DecodeTokenExtensions bool `yaml:"decode_token_extensions"`
	EnableCaching         bool `yaml:"enable_caching"`
	CacheSize             int  `yaml:"cache_size"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.RPC.Endpoint == "" {
		return fmt.Errorf("rpc.endpoint is required")
	}

	if c.RPC.PollInterval <= 0 {
		return fmt.Errorf("rpc.poll_interval must be positive")
	}

	if c.RPC.Timeout <= 0 {
		return fmt.Errorf("rpc.timeout must be positive")
	}

	if c.Metrics.FlushInterval <= 0 {
		return fmt.Errorf("metrics.flush_interval must be positive")
	}

	if c.Pipeline.BufferSize <= 0 {
		return fmt.Errorf("pipeline.buffer_size must be positive")
	}

	if c.Pipeline.ShutdownTimeout <= 0 {
		return fmt.Errorf("pipeline.shutdown_timeout must be positive")
	}

	if c.Advanced.WorkerCount <= 0 {
		return fmt.Errorf("advanced.worker_count must be positive")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging.level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid logging.format: %s (must be text or json)", c.Logging.Format)
	}

	validBackends := map[string]bool{
		"log":        true,
		"prometheus": true,
		"none":       true,
	}
	if !validBackends[c.Metrics.Backend] {
		return fmt.Errorf("invalid metrics.backend: %s (must be log, prometheus, or none)", c.Metrics.Backend)
	}

	return nil
}

func DefaultConfig() *Config {
	return &Config{
		RPC: RPCConfig{
			Endpoint:     "https://api.devnet.solana.com",
			PollInterval: 3,
			Timeout:      30,
		},
		Accounts: AccountsConfig{},
		TrackByMint: TrackByMintConfig{
			Enabled:         false,
			Mints:           []string{},
			MaxAccounts:     100,
			RefreshInterval: 60,
		},
		Mints: MintsConfig{
			Enabled: false,
			Mints:   []string{},
		},
		Alerts: AlertsConfig{
			Enabled:           true,
			Threshold:         1_000_000,
			AlertNewAccounts:  true,
			AlertStateChanges: true,
		},
		Metrics: MetricsConfig{
			FlushInterval: 15,
			Backend:       "log",
			Track: MetricsTrack{
				TokenUpdates:         true,
				MintUpdates:          true,
				SignificantTransfers: true,
				AccountBalances:      true,
			},
		},
		Pipeline: PipelineConfig{
			BufferSize:       500,
			GracefulShutdown: true,
			ShutdownTimeout:  30,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Structured: true,
			Verbose: VerboseConfig{
				IncludeSlot:      true,
				IncludeTimestamp: true,
				IncludePubkey:    true,
			},
		},
		Advanced: AdvancedConfig{
			WorkerCount: 4,
			Experimental: ExperimentalConfig{
				TrackMintSupply:       true,
				DecodeTokenExtensions: false,
				EnableCaching:         false,
				CacheSize:             1000,
			},
		},
	}
}
