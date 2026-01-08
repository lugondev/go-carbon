// Package account provides structures and interfaces for processing and decoding
// Solana accounts within the pipeline.
//
// This package includes the necessary components for handling account data updates
// in the carbon pipeline. It provides abstractions for decoding accounts,
// processing account metadata, and implementing account-specific pipes.
//
// # Overview
//
// The account package supports various tasks related to Solana account processing:
//   - Account Metadata: Metadata about accounts, including slot and public key.
//   - Decoded Account: Holds detailed account data after decoding.
//   - Account Decoders: Interface-based mechanism to decode raw Solana account data.
//   - Account Pipes: Encapsulates account processing logic for the pipeline.
package account

import (
	"context"
	"log/slog"

	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/filter"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/processor"
	"github.com/lugondev/go-carbon/pkg/types"
)

// AccountMetadata holds metadata for an account update, including the slot and public key.
//
// AccountMetadata provides essential information about an account update, such as
// the slot number where the account was updated and the account's public key.
// This metadata is used within the pipeline to identify and process account updates.
type AccountMetadata struct {
	// Slot is the Solana slot number where the account was updated.
	Slot uint64

	// Pubkey is the public key of the account.
	Pubkey types.Pubkey

	// TransactionSignature is the signature of the transaction that caused the update.
	TransactionSignature *types.Signature
}

// NewAccountMetadata creates AccountMetadata from an AccountUpdate.
func NewAccountMetadata(update *datasource.AccountUpdate) *AccountMetadata {
	return &AccountMetadata{
		Slot:                 update.Slot,
		Pubkey:               update.Pubkey,
		TransactionSignature: update.TransactionSignature,
	}
}

// DecodedAccount represents the decoded data of a Solana account, including
// account-specific details.
//
// DecodedAccount holds the detailed data of a Solana account after it has been
// decoded. It includes the account's lamports, owner, executable status, and
// rent epoch, as well as any decoded data specific to the account.
//
// Type parameter T is the type of data specific to the account, which is
// determined by the decoder used.
type DecodedAccount[T any] struct {
	// Lamports is the number of lamports in the account.
	Lamports uint64

	// Data is the decoded data specific to the account.
	Data T

	// Owner is the public key of the account's owner.
	Owner types.Pubkey

	// Executable indicates whether the account is executable.
	Executable bool

	// RentEpoch is the rent epoch of the account.
	RentEpoch uint64
}

// AccountDecoder defines an interface for decoding Solana accounts into structured data types.
//
// AccountDecoder provides a way to convert raw Solana Account data into structured
// DecodedAccount instances. By implementing this interface, you can define custom
// decoding logic to interpret account data.
//
// Type parameter T is the data type resulting from decoding the account.
type AccountDecoder[T any] interface {
	// DecodeAccount decodes a raw Solana account into a DecodedAccount.
	// Returns nil if the account cannot be decoded by this decoder.
	DecodeAccount(account *types.Account) *DecodedAccount[T]
}

// AccountDecoderFunc is a function type that implements AccountDecoder.
type AccountDecoderFunc[T any] func(account *types.Account) *DecodedAccount[T]

// DecodeAccount implements AccountDecoder interface.
func (f AccountDecoderFunc[T]) DecodeAccount(account *types.Account) *DecodedAccount[T] {
	return f(account)
}

// AccountProcessorInput is the input type for the account processor.
// It contains account metadata, the decoded account, and the raw account.
type AccountProcessorInput[T any] struct {
	// Metadata contains information about the account update.
	Metadata *AccountMetadata

	// DecodedAccount contains the decoded account data.
	DecodedAccount *DecodedAccount[T]

	// RawAccount contains the original raw account data.
	RawAccount *types.Account
}

// AccountPipe is a processing pipe that decodes and processes Solana account updates.
//
// AccountPipe combines an AccountDecoder and a Processor to manage account updates
// in the pipeline. This struct decodes the raw account data and then processes the
// resulting DecodedAccount with the specified processing logic.
//
// Type parameter T is the data type of the decoded account information.
type AccountPipe[T any] struct {
	// Decoder is an AccountDecoder that decodes raw account data.
	Decoder AccountDecoder[T]

	// Processor handles the processing logic for decoded accounts.
	Processor processor.Processor[AccountProcessorInput[T]]

	// Filters determine which account updates should be processed.
	// Only updates that pass all filters will be processed.
	Filters []filter.Filter

	// Logger is used for logging (optional).
	Logger *slog.Logger
}

// NewAccountPipe creates a new AccountPipe with the given decoder and processor.
func NewAccountPipe[T any](
	decoder AccountDecoder[T],
	proc processor.Processor[AccountProcessorInput[T]],
) *AccountPipe[T] {
	return &AccountPipe[T]{
		Decoder:   decoder,
		Processor: proc,
		Filters:   make([]filter.Filter, 0),
		Logger:    slog.Default(),
	}
}

// NewAccountPipeWithFilters creates a new AccountPipe with filters.
func NewAccountPipeWithFilters[T any](
	decoder AccountDecoder[T],
	proc processor.Processor[AccountProcessorInput[T]],
	filters []filter.Filter,
) *AccountPipe[T] {
	return &AccountPipe[T]{
		Decoder:   decoder,
		Processor: proc,
		Filters:   filters,
		Logger:    slog.Default(),
	}
}

// WithLogger sets a custom logger for the AccountPipe.
func (p *AccountPipe[T]) WithLogger(logger *slog.Logger) *AccountPipe[T] {
	p.Logger = logger
	return p
}

// AddFilter adds a filter to the AccountPipe.
func (p *AccountPipe[T]) AddFilter(f filter.Filter) {
	p.Filters = append(p.Filters, f)
}

// GetFilters returns the filters associated with this pipe.
func (p *AccountPipe[T]) GetFilters() []filter.Filter {
	return p.Filters
}

// Run processes an account update through the pipe.
//
// It first attempts to decode the account using the decoder. If decoding succeeds,
// it passes the decoded account to the processor.
func (p *AccountPipe[T]) Run(
	ctx context.Context,
	metadata *AccountMetadata,
	account *types.Account,
	metricsCollection *metrics.Collection,
) error {
	p.Logger.Debug("AccountPipe.Run",
		"slot", metadata.Slot,
		"pubkey", metadata.Pubkey.String(),
	)

	// Attempt to decode the account
	decodedAccount := p.Decoder.DecodeAccount(account)
	if decodedAccount == nil {
		// Account doesn't match this decoder, skip processing
		return nil
	}

	// Process the decoded account
	input := AccountProcessorInput[T]{
		Metadata:       metadata,
		DecodedAccount: decodedAccount,
		RawAccount:     account,
	}

	return p.Processor.Process(ctx, input, metricsCollection)
}

// AccountPipeRunner is an interface for running account pipes.
// This allows for type-erased storage of AccountPipe instances with different type parameters.
type AccountPipeRunner interface {
	// RunAccount processes an account update.
	RunAccount(
		ctx context.Context,
		metadata *AccountMetadata,
		account *types.Account,
		metricsCollection *metrics.Collection,
	) error

	// GetFilters returns the filters for this pipe.
	GetFilters() []filter.Filter
}

// Ensure AccountPipe implements AccountPipeRunner.
var _ AccountPipeRunner = (*AccountPipe[any])(nil)

// RunAccount implements AccountPipeRunner interface.
func (p *AccountPipe[T]) RunAccount(
	ctx context.Context,
	metadata *AccountMetadata,
	account *types.Account,
	metricsCollection *metrics.Collection,
) error {
	return p.Run(ctx, metadata, account, metricsCollection)
}

// MultiAccountPipe manages multiple account pipes and routes updates to all of them.
type MultiAccountPipe struct {
	pipes  []AccountPipeRunner
	logger *slog.Logger
}

// NewMultiAccountPipe creates a new MultiAccountPipe.
func NewMultiAccountPipe() *MultiAccountPipe {
	return &MultiAccountPipe{
		pipes:  make([]AccountPipeRunner, 0),
		logger: slog.Default(),
	}
}

// AddPipe adds an account pipe to the multi-pipe.
func (m *MultiAccountPipe) AddPipe(pipe AccountPipeRunner) {
	m.pipes = append(m.pipes, pipe)
}

// WithLogger sets a custom logger.
func (m *MultiAccountPipe) WithLogger(logger *slog.Logger) *MultiAccountPipe {
	m.logger = logger
	return m
}

// Run processes an account update through all pipes.
func (m *MultiAccountPipe) Run(
	ctx context.Context,
	datasourceID datasource.DatasourceID,
	metadata *AccountMetadata,
	account *types.Account,
	metricsCollection *metrics.Collection,
) error {
	for _, pipe := range m.pipes {
		accountMetadata := &filter.AccountMetadata{
			Slot:                 metadata.Slot,
			Pubkey:               metadata.Pubkey,
			TransactionSignature: metadata.TransactionSignature,
		}

		if !filter.CheckAccountFilters(datasourceID, pipe.GetFilters(), accountMetadata, account) {
			continue
		}

		if err := pipe.RunAccount(ctx, metadata, account, metricsCollection); err != nil {
			return err
		}
	}
	return nil
}

// Len returns the number of pipes in the multi-pipe.
func (m *MultiAccountPipe) Len() int {
	return len(m.pipes)
}

// ProgramAccountDecoder is a helper for creating account decoders that filter by program ID.
type ProgramAccountDecoder[T any] struct {
	// ProgramID is the expected program ID for accounts this decoder handles.
	ProgramID types.Pubkey

	// DecodeFunc is the function that decodes the account data.
	DecodeFunc func(data []byte) (T, error)
}

// NewProgramAccountDecoder creates a new ProgramAccountDecoder.
func NewProgramAccountDecoder[T any](
	programID types.Pubkey,
	decodeFunc func(data []byte) (T, error),
) *ProgramAccountDecoder[T] {
	return &ProgramAccountDecoder[T]{
		ProgramID:  programID,
		DecodeFunc: decodeFunc,
	}
}

// DecodeAccount implements AccountDecoder interface.
// It only decodes accounts owned by the specified program.
func (d *ProgramAccountDecoder[T]) DecodeAccount(account *types.Account) *DecodedAccount[T] {
	// Check if account is owned by the expected program
	if account.Owner != d.ProgramID {
		return nil
	}

	// Attempt to decode the account data
	data, err := d.DecodeFunc(account.Data)
	if err != nil {
		return nil
	}

	return &DecodedAccount[T]{
		Lamports:   account.Lamports,
		Data:       data,
		Owner:      account.Owner,
		Executable: account.Executable,
		RentEpoch:  account.RentEpoch,
	}
}

// CompositeAccountDecoder tries multiple decoders in sequence.
// It returns the result from the first decoder that succeeds.
type CompositeAccountDecoder[T any] struct {
	decoders []AccountDecoder[T]
}

// NewCompositeAccountDecoder creates a new CompositeAccountDecoder.
func NewCompositeAccountDecoder[T any](decoders ...AccountDecoder[T]) *CompositeAccountDecoder[T] {
	return &CompositeAccountDecoder[T]{
		decoders: decoders,
	}
}

// AddDecoder adds a decoder to the composite.
func (c *CompositeAccountDecoder[T]) AddDecoder(decoder AccountDecoder[T]) {
	c.decoders = append(c.decoders, decoder)
}

// DecodeAccount implements AccountDecoder interface.
func (c *CompositeAccountDecoder[T]) DecodeAccount(account *types.Account) *DecodedAccount[T] {
	for _, decoder := range c.decoders {
		if result := decoder.DecodeAccount(account); result != nil {
			return result
		}
	}
	return nil
}
