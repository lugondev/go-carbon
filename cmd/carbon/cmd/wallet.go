package cmd

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/spf13/cobra"
)

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Wallet management commands",
	Long:  `Commands for managing Solana wallets including generation and balance checks.`,
}

var walletNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Generate a new wallet",
	Long:  `Generate a new Solana wallet keypair.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		account := solana.NewWallet()

		fmt.Println("New wallet generated!")
		fmt.Printf("  Public Key:  %s\n", account.PublicKey().String())
		fmt.Printf("  Private Key: %s\n", account.PrivateKey.String())
		fmt.Println("\n⚠️  WARNING: Save your private key securely. Never share it with anyone!")

		return nil
	},
}

var walletBalanceCmd = &cobra.Command{
	Use:   "balance [address]",
	Short: "Check wallet balance",
	Long:  `Check the SOL balance of a wallet address.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]

		pubKey, err := solana.PublicKeyFromBase58(address)
		if err != nil {
			return fmt.Errorf("invalid address: %w", err)
		}

		fmt.Printf("Address: %s\n", pubKey.String())
		fmt.Println("(Balance check requires RPC connection - implement in solana package)")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(walletCmd)
	walletCmd.AddCommand(walletNewCmd)
	walletCmd.AddCommand(walletBalanceCmd)
}
