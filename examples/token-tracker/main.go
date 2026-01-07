package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/account"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/datasource/rpc"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/pipeline"
	"github.com/lugondev/go-carbon/internal/processor"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	config, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogger(config.Logging)
	slog.SetDefault(logger)

	logger.Info("🚀 Starting go-carbon token tracker",
		"config_file", *configPath,
		"rpc_endpoint", config.RPC.Endpoint,
	)

	rpcConfig := rpc.DefaultConfig(config.RPC.Endpoint)
	rpcConfig.PollInterval = config.RPC.GetPollInterval()

	var accountsToMonitor []solana.PublicKey

	if config.TrackByMint.Enabled && len(config.TrackByMint.Mints) > 0 {
		logger.Info("🌐 Track by Mint mode enabled")

		mints, mintErr := config.TrackByMint.GetMints()
		if mintErr != nil {
			logger.Error("Failed to parse track_by_mint addresses", "error", mintErr)
			os.Exit(1)
		}

		logger.Info("🔍 Scanning blockchain for token accounts...",
			"mint_count", len(mints),
			"max_per_mint", config.TrackByMint.MaxAccounts,
		)

		scanner := NewMintScanner(
			config.RPC.Endpoint,
			logger,
			config.TrackByMint.MaxAccounts,
		)

		ctx := context.Background()
		accounts, scanErr := scanner.GetTokenAccountsByMints(ctx, mints)
		if scanErr != nil {
			logger.Error("Failed to scan token accounts", "error", scanErr)
			os.Exit(1)
		}
		accountsToMonitor = accounts

		if len(accountsToMonitor) == 0 {
			logger.Warn("⚠️  No token accounts found for specified mints")
			os.Exit(0)
		}

		logger.Info("📊 Total accounts discovered", "count", len(accountsToMonitor))
	} else {
		accounts, accErr := config.Accounts.GetPublicKeys()
		if accErr != nil {
			logger.Error("Failed to parse account addresses", "error", accErr)
			os.Exit(1)
		}
		accountsToMonitor = accounts
	}

	if len(accountsToMonitor) == 0 {
		logger.Warn("⚠️  No accounts configured to monitor. Please add accounts to config.yaml")
	} else {
		logger.Info("👀 Monitoring accounts", "count", len(accountsToMonitor))
		for i, acc := range accountsToMonitor {
			accShort := acc.String()
			if len(accShort) > 12 {
				accShort = accShort[:8] + "..." + accShort[len(accShort)-4:]
			}
			logger.Info(fmt.Sprintf("   [%d] %s", i+1, accShort))
		}
	}

	rpcDatasource := rpc.NewAccountMonitorDatasource(rpcConfig, accountsToMonitor)
	rpcDatasource.WithLogger(logger)

	tokenDecoder := NewTokenAccountDecoder(logger)
	tokenTracker := NewTokenTracker(logger, &config.Alerts)

	var tokenProcessor processor.Processor[account.AccountProcessorInput[TokenAccount]]

	mintMap, err := config.Mints.GetMintMap()
	if err != nil {
		logger.Error("Failed to parse mint addresses", "error", err)
		os.Exit(1)
	}

	if config.Mints.Enabled && len(mintMap) > 0 {
		logger.Info("🎯 Filtering by mint addresses", "count", len(mintMap))
		for mint := range mintMap {
			mintShort := mint
			if len(mintShort) > 12 {
				mintShort = mintShort[:8] + "..." + mintShort[len(mintShort)-4:]
			}
			logger.Info(fmt.Sprintf("   Target mint: %s", mintShort))
		}

		tokenProcessor = processor.NewConditionalProcessor(
			tokenTracker,
			func(input account.AccountProcessorInput[TokenAccount]) bool {
				mintStr := input.DecodedAccount.Data.Mint.String()
				isMatch := mintMap[mintStr]
				if !isMatch {
					logger.Debug("❌ Account filtered out (mint mismatch)",
						"account", input.Metadata.Pubkey.String()[:8]+"...",
						"mint", mintStr[:8]+"...",
					)
				}
				return isMatch
			},
		)
	} else {
		tokenProcessor = tokenTracker
	}

	tokenPipe := account.NewAccountPipe(tokenDecoder, tokenProcessor)
	tokenPipe.WithLogger(logger)

	metricsCollection := createMetricsCollection(logger, config.Metrics)

	builder := pipeline.Builder().
		Datasource(datasource.NewNamedDatasourceID("rpc-monitor"), rpcDatasource).
		AccountPipe(tokenPipe).
		Metrics(metricsCollection).
		MetricsFlushInterval(config.Metrics.GetFlushInterval()).
		ChannelBufferSize(config.Pipeline.BufferSize).
		Logger(logger)

	if config.Pipeline.GracefulShutdown {
		builder = builder.WithGracefulShutdown()
	}

	if config.Advanced.Experimental.TrackMintSupply {
		logger.Info("🪙 Mint supply tracking enabled")
		mintDecoder := NewTokenMintDecoder()
		mintTracker := NewMintTracker(logger, tokenTracker)
		mintPipe := account.NewAccountPipe(mintDecoder, mintTracker)
		mintPipe.WithLogger(logger)
		builder = builder.AccountPipe(mintPipe)
	}

	p := builder.Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("✅ Pipeline started, monitoring for token movements...")
	logger.Info("💡 Press Ctrl+C to stop")

	errChan := make(chan error, 1)
	go func() {
		errChan <- p.Run(ctx)
	}()

	select {
	case sig := <-sigChan:
		logger.Info("🛑 Received shutdown signal", "signal", sig)
		cancel()
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			logger.Error("❌ Pipeline error", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("👋 Token tracker stopped gracefully")
}

func setupLogger(config LoggingConfig) *slog.Logger {
	var level slog.Level
	switch config.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func createMetricsCollection(logger *slog.Logger, config MetricsConfig) *metrics.Collection {
	if config.Backend == "none" {
		return metrics.NewCollection()
	}

	return metrics.NewCollection(
		metrics.NewLogMetrics(logger),
	)
}
