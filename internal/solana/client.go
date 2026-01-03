package solana

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Client wraps the Solana RPC client
type Client struct {
	rpc *rpc.Client
}

// NewClient creates a new Solana client
func NewClient(endpoint string) *Client {
	return &Client{
		rpc: rpc.New(endpoint),
	}
}

// GetBalance returns the balance of an account in lamports
func (c *Client) GetBalance(ctx context.Context, pubkey solana.PublicKey) (uint64, error) {
	result, err := c.rpc.GetBalance(ctx, pubkey, rpc.CommitmentFinalized)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	return result.Value, nil
}

// GetBalanceSOL returns the balance in SOL (not lamports)
func (c *Client) GetBalanceSOL(ctx context.Context, pubkey solana.PublicKey) (float64, error) {
	lamports, err := c.GetBalance(ctx, pubkey)
	if err != nil {
		return 0, err
	}
	return float64(lamports) / float64(solana.LAMPORTS_PER_SOL), nil
}

// GetLatestBlockhash returns the latest blockhash
func (c *Client) GetLatestBlockhash(ctx context.Context) (*rpc.GetLatestBlockhashResult, error) {
	result, err := c.rpc.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest blockhash: %w", err)
	}
	return result, nil
}

// GetAccountInfo returns the account info for a given public key
func (c *Client) GetAccountInfo(ctx context.Context, pubkey solana.PublicKey) (*rpc.GetAccountInfoResult, error) {
	result, err := c.rpc.GetAccountInfo(ctx, pubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}
	return result, nil
}

// RequestAirdrop requests an airdrop of SOL (only works on devnet/testnet)
func (c *Client) RequestAirdrop(ctx context.Context, pubkey solana.PublicKey, lamports uint64) (solana.Signature, error) {
	sig, err := c.rpc.RequestAirdrop(ctx, pubkey, lamports, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to request airdrop: %w", err)
	}
	return sig, nil
}

// GetTransaction returns transaction details
func (c *Client) GetTransaction(ctx context.Context, sig solana.Signature) (*rpc.GetTransactionResult, error) {
	maxVersion := uint64(0)
	result, err := c.rpc.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		MaxSupportedTransactionVersion: &maxVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return result, nil
}

// SendTransaction sends a transaction
func (c *Client) SendTransaction(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	sig, err := c.rpc.SendTransaction(ctx, tx)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send transaction: %w", err)
	}
	return sig, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	// RPC client doesn't have a close method, but we include this for future use
	return nil
}
