package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "carbon",
	Short: "Carbon CLI - A Solana blockchain tool",
	Long: `Carbon is a CLI application for interacting with the Solana blockchain.

It provides commands for:
- Wallet management
- Transaction operations
- Account queries
- And more...`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.carbon.yaml)")
	rootCmd.PersistentFlags().String("rpc", "https://api.devnet.solana.com", "Solana RPC endpoint")
	rootCmd.PersistentFlags().String("network", "devnet", "Solana network (mainnet, devnet, testnet)")

	if err := viper.BindPFlag("rpc", rootCmd.PersistentFlags().Lookup("rpc")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %v\n", err)
	}
	if err := viper.BindPFlag("network", rootCmd.PersistentFlags().Lookup("network")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %v\n", err)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".carbon")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
