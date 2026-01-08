// Package transaction provides types and traits for handling transaction processing
// in the carbon-core framework. It also provides utilities for matching transactions
// to schemas and executing custom processing logic on matched data.
//
// # Key Components
//
//   - TransactionPipe: Represents a processing pipe for transactions, with functionality
//     to parse and match instructions against a schema and handle matched data with a
//     specified processor.
//   - TransactionMetadata: Metadata associated with a transaction, including slot,
//     signature, and fee payer information.
//   - ParsedTransaction: Represents a transaction with its metadata and parsed instructions.
//
// # Usage
//
// To use this module, create a TransactionPipe with a transaction schema and a processor.
// Then, run the transaction pipe with a set of instructions and metrics to parse, match,
// and process transaction data.
package transaction

import (
	"context"
	"log/slog"
	"sync"

	"github.com/lugondev/go-carbon/internal/datasource"
	cerrors "github.com/lugondev/go-carbon/internal/errors"
	"github.com/lugondev/go-carbon/internal/filter"
	"github.com/lugondev/go-carbon/internal/instruction"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/processor"
	"github.com/lugondev/go-carbon/pkg/types"
)

// TransactionMetadata contains metadata about a transaction, including its slot, signature,
// fee payer, transaction status metadata, and block time.
type TransactionMetadata struct {
	// Slot is the slot number in which this transaction was processed.
	Slot uint64

	// Signature is the unique signature of this transaction.
	Signature types.Signature

	// FeePayer is the public key of the fee payer account that paid for this transaction.
	FeePayer types.Pubkey

	// Meta contains transaction status metadata including execution status, fees, balances.
	Meta *types.TransactionStatusMeta

	// Index is the index of the transaction within the slot (block).
	Index *uint64

	// BlockTime is the Unix timestamp of when the transaction was processed.
	BlockTime *int64

	// BlockHash is the block hash that can be used to detect a fork.
	BlockHash *types.Hash

	// AccountKeys is the list of all account keys used in the transaction.
	AccountKeys []types.Pubkey
}

// GetSlot returns the transaction slot.
func (m *TransactionMetadata) GetSlot() uint64 {
	return m.Slot
}

// GetSignature returns the transaction signature.
func (m *TransactionMetadata) GetSignature() types.Signature {
	return m.Signature
}

// GetFeePayer returns the fee payer's public key.
func (m *TransactionMetadata) GetFeePayer() types.Pubkey {
	return m.FeePayer
}

// ToInstructionMetadataRef converts TransactionMetadata to a reference for instruction metadata.
func (m *TransactionMetadata) ToInstructionMetadataRef() *instruction.TransactionMetadataRef {
	var logMessages []string
	if m.Meta != nil {
		logMessages = m.Meta.LogMessages
	}

	return &instruction.TransactionMetadataRef{
		Slot:        m.Slot,
		Signature:   m.Signature,
		FeePayer:    m.FeePayer,
		LogMessages: logMessages,
		Meta:        m.Meta,
	}
}

// NewTransactionMetadataFromUpdate creates TransactionMetadata from a TransactionUpdate.
func NewTransactionMetadataFromUpdate(update *datasource.TransactionUpdate) (*TransactionMetadata, error) {
	if update.Transaction == nil {
		return nil, cerrors.ErrMissingAccount
	}

	// Get account keys from the transaction message
	var accountKeys []types.Pubkey
	if update.Transaction.Message.AccountKeys != nil {
		for _, key := range update.Transaction.Message.AccountKeys {
			accountKeys = append(accountKeys, key)
		}
	}

	// Get fee payer (first account key)
	var feePayer types.Pubkey
	if len(accountKeys) > 0 {
		feePayer = accountKeys[0]
	} else {
		return nil, cerrors.ErrMissingFeePayer
	}

	return &TransactionMetadata{
		Slot:        update.Slot,
		Signature:   update.Signature,
		FeePayer:    feePayer,
		Meta:        &update.Meta,
		Index:       update.Index,
		BlockTime:   update.BlockTime,
		BlockHash:   update.BlockHash,
		AccountKeys: accountKeys,
	}, nil
}

// DecodedInstructionWithMetadata pairs a decoded instruction with its metadata.
type DecodedInstructionWithMetadata[T any] struct {
	// Metadata contains information about the instruction.
	Metadata *instruction.InstructionMetadata

	// DecodedInstruction contains the decoded instruction data.
	DecodedInstruction *instruction.DecodedInstruction[T]
}

// TransactionProcessorInput is the input type for the transaction processor.
type TransactionProcessorInput[T any, U any] struct {
	// Metadata contains information about the transaction.
	Metadata *TransactionMetadata

	// Instructions contains all decoded instructions with their metadata.
	Instructions []DecodedInstructionWithMetadata[T]

	// MatchedData contains the matched schema data, if schema matching was enabled.
	MatchedData *U
}

// TransactionPipe is a pipe for processing transactions based on a defined schema and processor.
//
// The TransactionPipe parses a transaction's instructions, optionally checks them against
// the schema, and runs the processor if the instructions match the schema.
//
// Type parameters:
//   - T: The instruction type.
//   - U: The output type for the matched data, if schema-matching.
type TransactionPipe[T any, U any] struct {
	// Schema is the schema against which to match transaction instructions.
	Schema *TransactionSchema[T]

	// Processor handles matched transaction data.
	Processor processor.Processor[TransactionProcessorInput[T, U]]

	// Filters determine which transaction updates should be processed.
	Filters []filter.Filter

	// InstructionDecoder decodes instructions into the type T.
	InstructionDecoder instruction.InstructionDecoder[T]

	// Logger is used for logging (optional).
	Logger *slog.Logger
}

// NewTransactionPipe creates a new TransactionPipe.
func NewTransactionPipe[T any, U any](
	schema *TransactionSchema[T],
	decoder instruction.InstructionDecoder[T],
	proc processor.Processor[TransactionProcessorInput[T, U]],
) *TransactionPipe[T, U] {
	return &TransactionPipe[T, U]{
		Schema:             schema,
		InstructionDecoder: decoder,
		Processor:          proc,
		Filters:            make([]filter.Filter, 0),
		Logger:             slog.Default(),
	}
}

// NewTransactionPipeWithFilters creates a new TransactionPipe with filters.
func NewTransactionPipeWithFilters[T any, U any](
	schema *TransactionSchema[T],
	decoder instruction.InstructionDecoder[T],
	proc processor.Processor[TransactionProcessorInput[T, U]],
	filters []filter.Filter,
) *TransactionPipe[T, U] {
	return &TransactionPipe[T, U]{
		Schema:             schema,
		InstructionDecoder: decoder,
		Processor:          proc,
		Filters:            filters,
		Logger:             slog.Default(),
	}
}

// WithLogger sets a custom logger for the TransactionPipe.
func (p *TransactionPipe[T, U]) WithLogger(logger *slog.Logger) *TransactionPipe[T, U] {
	p.Logger = logger
	return p
}

// GetFilters returns the filters associated with this pipe.
func (p *TransactionPipe[T, U]) GetFilters() []filter.Filter {
	return p.Filters
}

// Run processes a transaction, parsing its instructions and matching against the schema.
func (p *TransactionPipe[T, U]) Run(
	ctx context.Context,
	metadata *TransactionMetadata,
	nestedInstructions *instruction.NestedInstructions,
	metricsCollection *metrics.Collection,
) error {
	p.Logger.Debug("TransactionPipe.Run",
		"slot", metadata.Slot,
		"signature", metadata.Signature.String(),
	)

	// Parse instructions
	parsedInstructions := p.parseInstructions(metadata, nestedInstructions)

	// Unnest instructions for the processor
	unnestedInstructions := p.unnestInstructions(metadata, nestedInstructions, 0)

	// Create input for processor
	input := TransactionProcessorInput[T, U]{
		Metadata:     metadata,
		Instructions: unnestedInstructions,
		MatchedData:  nil, // Schema matching is handled separately if needed
	}

	// Check schema match if provided
	if p.Schema != nil && !p.Schema.Matches(parsedInstructions) {
		// Schema doesn't match, skip processing
		return nil
	}

	return p.Processor.Process(ctx, input, metricsCollection)
}

// parseInstructions parses nested instructions into ParsedInstructions.
func (p *TransactionPipe[T, U]) parseInstructions(
	metadata *TransactionMetadata,
	nestedIxs *instruction.NestedInstructions,
) []*ParsedInstruction[T] {
	var parsed []*ParsedInstruction[T]

	for _, nestedIx := range nestedIxs.Instructions {
		// Try to decode the instruction
		decoded := p.InstructionDecoder.DecodeInstruction(nestedIx.Instruction)
		if decoded != nil {
			parsed = append(parsed, &ParsedInstruction[T]{
				ProgramID:         nestedIx.Instruction.ProgramID,
				Instruction:       decoded,
				InnerInstructions: p.parseInstructions(metadata, nestedIx.InnerInstructions),
			})
		} else {
			// If we can't decode this instruction, try to parse its inner instructions
			innerParsed := p.parseInstructions(metadata, nestedIx.InnerInstructions)
			parsed = append(parsed, innerParsed...)
		}
	}

	return parsed
}

// unnestInstructions flattens nested instructions with metadata.
func (p *TransactionPipe[T, U]) unnestInstructions(
	metadata *TransactionMetadata,
	nestedIxs *instruction.NestedInstructions,
	depth int,
) []DecodedInstructionWithMetadata[T] {
	var result []DecodedInstructionWithMetadata[T]

	for _, nestedIx := range nestedIxs.Instructions {
		decoded := p.InstructionDecoder.DecodeInstruction(nestedIx.Instruction)
		if decoded != nil {
			result = append(result, DecodedInstructionWithMetadata[T]{
				Metadata:           nestedIx.Metadata,
				DecodedInstruction: decoded,
			})
		}

		// Recursively process inner instructions
		innerResult := p.unnestInstructions(metadata, nestedIx.InnerInstructions, depth+1)
		result = append(result, innerResult...)
	}

	return result
}

// TransactionPipeRunner is an interface for running transaction pipes.
// This allows for type-erased storage of TransactionPipe instances.
type TransactionPipeRunner interface {
	// RunTransaction processes a transaction.
	RunTransaction(
		ctx context.Context,
		metadata *TransactionMetadata,
		nestedInstructions *instruction.NestedInstructions,
		metricsCollection *metrics.Collection,
	) error

	// GetFilters returns the filters for this pipe.
	GetFilters() []filter.Filter
}

// Ensure TransactionPipe implements TransactionPipeRunner.
var _ TransactionPipeRunner = (*TransactionPipe[any, any])(nil)

// RunTransaction implements TransactionPipeRunner interface.
func (p *TransactionPipe[T, U]) RunTransaction(
	ctx context.Context,
	metadata *TransactionMetadata,
	nestedInstructions *instruction.NestedInstructions,
	metricsCollection *metrics.Collection,
) error {
	return p.Run(ctx, metadata, nestedInstructions, metricsCollection)
}

// MultiTransactionPipe manages multiple transaction pipes and routes updates to all of them.
type MultiTransactionPipe struct {
	pipes  []TransactionPipeRunner
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewMultiTransactionPipe creates a new MultiTransactionPipe.
func NewMultiTransactionPipe() *MultiTransactionPipe {
	return &MultiTransactionPipe{
		pipes:  make([]TransactionPipeRunner, 0),
		logger: slog.Default(),
	}
}

// AddPipe adds a transaction pipe to the multi-pipe.
func (m *MultiTransactionPipe) AddPipe(pipe TransactionPipeRunner) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pipes = append(m.pipes, pipe)
}

// WithLogger sets a custom logger.
func (m *MultiTransactionPipe) WithLogger(logger *slog.Logger) *MultiTransactionPipe {
	m.logger = logger
	return m
}

// Run processes a transaction through all pipes.
func (m *MultiTransactionPipe) Run(
	ctx context.Context,
	datasourceID datasource.DatasourceID,
	metadata *TransactionMetadata,
	nestedInstructions *instruction.NestedInstructions,
	metricsCollection *metrics.Collection,
) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pipe := range m.pipes {
		if !filter.CheckTransactionFilters(datasourceID, pipe.GetFilters(), metadata, nestedInstructions) {
			continue
		}

		if err := pipe.RunTransaction(ctx, metadata, nestedInstructions, metricsCollection); err != nil {
			return err
		}
	}
	return nil
}

// Len returns the number of pipes in the multi-pipe.
func (m *MultiTransactionPipe) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pipes)
}

// ParsedInstruction represents a parsed instruction with its decoded data and inner instructions.
type ParsedInstruction[T any] struct {
	// ProgramID is the program ID associated with this instruction.
	ProgramID types.Pubkey

	// Instruction is the decoded instruction data.
	Instruction *instruction.DecodedInstruction[T]

	// InnerInstructions contains parsed nested instructions.
	InnerInstructions []*ParsedInstruction[T]
}

// TransactionSchema represents the schema for a transaction, defining the structure
// and expected instructions.
//
// TransactionSchema allows you to define the structure of a transaction by specifying
// a list of SchemaNode elements at the root level. These nodes can represent specific
// instruction types or allow for flexibility with Any nodes.
type TransactionSchema[T any] struct {
	// Root contains the root schema nodes.
	Root []SchemaNode[T]

	// Matcher is a custom matching function (optional).
	Matcher func(instructions []*ParsedInstruction[T]) bool
}

// Matches checks if the given instructions match this schema.
func (s *TransactionSchema[T]) Matches(instructions []*ParsedInstruction[T]) bool {
	// Use custom matcher if provided
	if s.Matcher != nil {
		return s.Matcher(instructions)
	}

	// Default matching logic
	return s.matchNodes(s.Root, instructions)
}

// SchemaNode represents a node within a transaction schema, which can be either
// an Instruction node or an Any node to allow for flexible matching.
type SchemaNode[T any] interface {
	isSchemaNode()
}

// InstructionSchemaNode represents an instruction node within a schema, containing
// the instruction type, name, and optional nested instructions.
type InstructionSchemaNode[T any] struct {
	// IxType is the expected instruction type.
	IxType T

	// Name is a unique name identifier for the instruction node.
	Name string

	// InnerInstructions contains nested schema nodes.
	InnerInstructions []SchemaNode[T]

	// Matcher is a custom matching function for this instruction.
	Matcher func(instruction *instruction.DecodedInstruction[T]) bool
}

func (n *InstructionSchemaNode[T]) isSchemaNode() {}

// AnySchemaNode matches any instruction type, providing flexibility within the schema.
type AnySchemaNode[T any] struct{}

func (n *AnySchemaNode[T]) isSchemaNode() {}

// NewTransactionSchema creates a new TransactionSchema with the given root nodes.
func NewTransactionSchema[T any](root ...SchemaNode[T]) *TransactionSchema[T] {
	return &TransactionSchema[T]{
		Root: root,
	}
}

// matchNodes matches instructions against schema nodes.
func (s *TransactionSchema[T]) matchNodes(nodes []SchemaNode[T], instructions []*ParsedInstruction[T]) bool {
	nodeIndex := 0
	instructionIndex := 0
	anyMode := false

	for nodeIndex < len(nodes) {
		node := nodes[nodeIndex]

		// Handle Any node
		if _, isAny := node.(*AnySchemaNode[T]); isAny {
			anyMode = true
			nodeIndex++
			continue
		}

		matched := false

		for instructionIndex < len(instructions) {
			currentInstruction := instructions[instructionIndex]

			instructionNode, isInstruction := node.(*InstructionSchemaNode[T])
			if !isInstruction {
				return false
			}

			// Check if instruction matches
			if instructionNode.Matcher != nil {
				if !instructionNode.Matcher(currentInstruction.Instruction) {
					if !anyMode {
						return false
					}
					instructionIndex++
					continue
				}
			}

			// Match inner instructions if specified
			if len(instructionNode.InnerInstructions) > 0 {
				if !s.matchNodes(instructionNode.InnerInstructions, currentInstruction.InnerInstructions) {
					if !anyMode {
						return false
					}
					instructionIndex++
					continue
				}
			}

			instructionIndex++
			nodeIndex++
			anyMode = false
			matched = true
			break
		}

		if !matched {
			return false
		}
	}

	return true
}

// ParsedTransaction represents a parsed transaction with its metadata and instructions.
type ParsedTransaction[T any] struct {
	// Metadata contains transaction metadata.
	Metadata *TransactionMetadata

	// Instructions contains parsed instructions.
	Instructions []*ParsedInstruction[T]
}
