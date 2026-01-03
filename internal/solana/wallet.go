package solana

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
)

// Wallet represents a Solana wallet
type Wallet struct {
	privateKey solana.PrivateKey
}

// NewWallet generates a new random wallet
func NewWallet() *Wallet {
	account := solana.NewWallet()
	return &Wallet{
		privateKey: account.PrivateKey,
	}
}

// WalletFromPrivateKey creates a wallet from an existing private key
func WalletFromPrivateKey(pk solana.PrivateKey) *Wallet {
	return &Wallet{
		privateKey: pk,
	}
}

// WalletFromBase58 creates a wallet from a base58-encoded private key
func WalletFromBase58(key string) (*Wallet, error) {
	pk, err := solana.PrivateKeyFromBase58(key)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Wallet{privateKey: pk}, nil
}

// WalletFromFile loads a wallet from a JSON keypair file (Solana CLI format)
func WalletFromFile(path string) (*Wallet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read keypair file: %w", err)
	}

	var keypair []byte
	if err := json.Unmarshal(data, &keypair); err != nil {
		return nil, fmt.Errorf("failed to parse keypair: %w", err)
	}

	if len(keypair) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid keypair size: expected %d, got %d", ed25519.PrivateKeySize, len(keypair))
	}

	return &Wallet{
		privateKey: solana.PrivateKey(keypair),
	}, nil
}

// PublicKey returns the wallet's public key
func (w *Wallet) PublicKey() solana.PublicKey {
	return w.privateKey.PublicKey()
}

// PrivateKey returns the wallet's private key
func (w *Wallet) PrivateKey() solana.PrivateKey {
	return w.privateKey
}

// Sign signs a message with the wallet's private key
func (w *Wallet) Sign(message []byte) ([]byte, error) {
	sig, err := w.privateKey.Sign(message)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}
	return sig[:], nil
}

// SaveToFile saves the keypair to a JSON file (Solana CLI format)
func (w *Wallet) SaveToFile(path string) error {
	keypair := []byte(w.privateKey)
	data, err := json.Marshal(keypair)
	if err != nil {
		return fmt.Errorf("failed to marshal keypair: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write keypair file: %w", err)
	}

	return nil
}

// String returns the public key as a string
func (w *Wallet) String() string {
	return w.PublicKey().String()
}
