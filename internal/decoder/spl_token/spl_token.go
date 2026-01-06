// Package spl_token provides a decoder plugin for SPL Token program events.
//
// This plugin decodes common SPL Token events including:
//   - Transfer events
//   - Mint events
//   - Burn events
//   - Approve events
package spl_token

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/decoder"
	"github.com/lugondev/go-carbon/pkg/log"
	"github.com/lugondev/go-carbon/pkg/plugin"
)

// SPL Token Program ID
var TokenProgramID = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

// Token2022 Program ID
var Token2022ProgramID = solana.MustPublicKeyFromBase58("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")

// TransferEvent represents a token transfer event.
type TransferEvent struct {
	From   solana.PublicKey `json:"from"`
	To     solana.PublicKey `json:"to"`
	Amount uint64           `json:"amount"`
	Mint   solana.PublicKey `json:"mint,omitempty"`
}

// MintToEvent represents a mint-to event.
type MintToEvent struct {
	Mint        solana.PublicKey `json:"mint"`
	Destination solana.PublicKey `json:"destination"`
	Amount      uint64           `json:"amount"`
	Authority   solana.PublicKey `json:"authority"`
}

// BurnEvent represents a token burn event.
type BurnEvent struct {
	Account   solana.PublicKey `json:"account"`
	Mint      solana.PublicKey `json:"mint"`
	Amount    uint64           `json:"amount"`
	Authority solana.PublicKey `json:"authority"`
}

// ApproveEvent represents a token approval event.
type ApproveEvent struct {
	Source   solana.PublicKey `json:"source"`
	Delegate solana.PublicKey `json:"delegate"`
	Owner    solana.PublicKey `json:"owner"`
	Amount   uint64           `json:"amount"`
}

// SPLTokenPlugin is a plugin that decodes SPL Token events.
type SPLTokenPlugin struct {
	*plugin.BasePlugin
	decoders      []decoder.Decoder
	logProcessors []log.LogProcessor
}

// NewSPLTokenPlugin creates a new SPL Token plugin.
func NewSPLTokenPlugin() *SPLTokenPlugin {
	base := plugin.NewBasePlugin(
		"spl-token",
		"1.0.0",
		"SPL Token program event decoder",
	)

	p := &SPLTokenPlugin{
		BasePlugin: base,
	}

	// Create decoders
	p.decoders = []decoder.Decoder{
		NewTransferDecoder(),
		NewMintToDecoder(),
		NewBurnDecoder(),
		NewApproveDecoder(),
	}

	// Create log processors
	p.logProcessors = []log.LogProcessor{
		NewTransferLogProcessor(),
	}

	return p
}

// GetDecoders implements DecoderPlugin interface.
func (p *SPLTokenPlugin) GetDecoders() []decoder.Decoder {
	return p.decoders
}

// GetLogProcessors implements DecoderPlugin interface.
func (p *SPLTokenPlugin) GetLogProcessors() []log.LogProcessor {
	return p.logProcessors
}

// TransferDecoder decodes SPL Token transfer events from logs.
type TransferDecoder struct {
	name      string
	programID solana.PublicKey
}

// NewTransferDecoder creates a new TransferDecoder.
func NewTransferDecoder() *TransferDecoder {
	return &TransferDecoder{
		name:      "spl-token:transfer",
		programID: TokenProgramID,
	}
}

// Decode implements Decoder interface.
func (d *TransferDecoder) Decode(data []byte) (*decoder.Event, error) {
	// SPL Token doesn't emit events via "Program data:", but via logs
	// This is a placeholder for custom parsing logic
	return nil, fmt.Errorf("use log processor for SPL token transfers")
}

// CanDecode implements Decoder interface.
func (d *TransferDecoder) CanDecode(data []byte) bool {
	return false // SPL Token uses logs, not program data
}

// GetName implements Decoder interface.
func (d *TransferDecoder) GetName() string {
	return d.name
}

// GetProgramID implements Decoder interface.
func (d *TransferDecoder) GetProgramID() solana.PublicKey {
	return d.programID
}

// MintToDecoder decodes mint-to events.
type MintToDecoder struct {
	name      string
	programID solana.PublicKey
}

// NewMintToDecoder creates a new MintToDecoder.
func NewMintToDecoder() *MintToDecoder {
	return &MintToDecoder{
		name:      "spl-token:mint-to",
		programID: TokenProgramID,
	}
}

// Decode implements Decoder interface.
func (d *MintToDecoder) Decode(data []byte) (*decoder.Event, error) {
	return nil, fmt.Errorf("use log processor for SPL token mint-to")
}

// CanDecode implements Decoder interface.
func (d *MintToDecoder) CanDecode(data []byte) bool {
	return false
}

// GetName implements Decoder interface.
func (d *MintToDecoder) GetName() string {
	return d.name
}

// GetProgramID implements Decoder interface.
func (d *MintToDecoder) GetProgramID() solana.PublicKey {
	return d.programID
}

// BurnDecoder decodes burn events.
type BurnDecoder struct {
	name      string
	programID solana.PublicKey
}

// NewBurnDecoder creates a new BurnDecoder.
func NewBurnDecoder() *BurnDecoder {
	return &BurnDecoder{
		name:      "spl-token:burn",
		programID: TokenProgramID,
	}
}

// Decode implements Decoder interface.
func (d *BurnDecoder) Decode(data []byte) (*decoder.Event, error) {
	return nil, fmt.Errorf("use log processor for SPL token burn")
}

// CanDecode implements Decoder interface.
func (d *BurnDecoder) CanDecode(data []byte) bool {
	return false
}

// GetName implements Decoder interface.
func (d *BurnDecoder) GetName() string {
	return d.name
}

// GetProgramID implements Decoder interface.
func (d *BurnDecoder) GetProgramID() solana.PublicKey {
	return d.programID
}

// ApproveDecoder decodes approve events.
type ApproveDecoder struct {
	name      string
	programID solana.PublicKey
}

// NewApproveDecoder creates a new ApproveDecoder.
func NewApproveDecoder() *ApproveDecoder {
	return &ApproveDecoder{
		name:      "spl-token:approve",
		programID: TokenProgramID,
	}
}

// Decode implements Decoder interface.
func (d *ApproveDecoder) Decode(data []byte) (*decoder.Event, error) {
	return nil, fmt.Errorf("use log processor for SPL token approve")
}

// CanDecode implements Decoder interface.
func (d *ApproveDecoder) CanDecode(data []byte) bool {
	return false
}

// GetName implements Decoder interface.
func (d *ApproveDecoder) GetName() string {
	return d.name
}

// GetProgramID implements Decoder interface.
func (d *ApproveDecoder) GetProgramID() solana.PublicKey {
	return d.programID
}

// TransferLogProcessor processes transfer logs.
type TransferLogProcessor struct{}

// NewTransferLogProcessor creates a new TransferLogProcessor.
func NewTransferLogProcessor() *TransferLogProcessor {
	return &TransferLogProcessor{}
}

// ProcessLog implements LogProcessor interface.
func (p *TransferLogProcessor) ProcessLog(logEntry *log.ParsedLog) bool {
	if logEntry.Type != log.LogTypeLog {
		return false
	}

	// SPL Token logs don't follow a strict format, but we can detect patterns
	// Example: "Program log: Instruction: Transfer"
	if logEntry.Message == "Instruction: Transfer" {
		// This is a transfer instruction log
		return true
	}

	return false
}

// GetName implements LogProcessor interface.
func (p *TransferLogProcessor) GetName() string {
	return "spl-token:transfer-log"
}

// ParseTransferFromAccounts parses transfer details from instruction accounts.
// This is more reliable than parsing logs for SPL Token.
func ParseTransferFromAccounts(accounts []solana.PublicKey, amount uint64) *TransferEvent {
	if len(accounts) < 3 {
		return nil
	}

	return &TransferEvent{
		From:   accounts[0], // Source account
		To:     accounts[1], // Destination account
		Amount: amount,
		// Owner is accounts[2]
	}
}

// DecodeU64 decodes a uint64 from instruction data.
func DecodeU64(data []byte, offset int) (uint64, error) {
	if len(data) < offset+8 {
		return 0, fmt.Errorf("insufficient data for u64 at offset %d", offset)
	}
	return binary.LittleEndian.Uint64(data[offset : offset+8]), nil
}

// TransferInstructionData represents the data in a transfer instruction.
type TransferInstructionData struct {
	Instruction uint8  // Should be 3 for Transfer
	Amount      uint64 // Amount to transfer
}

// ParseTransferInstruction parses a transfer instruction's data.
func ParseTransferInstruction(data []byte) (*TransferInstructionData, error) {
	if len(data) < 9 {
		return nil, fmt.Errorf("insufficient data for transfer instruction: need 9 bytes, got %d", len(data))
	}

	instruction := data[0]
	if instruction != 3 { // 3 is the Transfer instruction
		return nil, fmt.Errorf("not a transfer instruction: got %d, expected 3", instruction)
	}

	amount := binary.LittleEndian.Uint64(data[1:9])

	return &TransferInstructionData{
		Instruction: instruction,
		Amount:      amount,
	}, nil
}
