// Package instruction provides structures and traits for decoding and processing
// instructions within transactions.
//
// The package includes the following main components:
//   - InstructionMetadata: Metadata associated with an instruction, capturing transaction context.
//   - DecodedInstruction: Represents an instruction that has been decoded, with associated
//     program ID, data, and accounts.
//   - InstructionDecoder: An interface for decoding instructions into specific types.
//   - InstructionPipe: A structure that processes instructions using a decoder and a processor.
//   - NestedInstruction: Represents instructions with potential nested inner instructions,
//     allowing for recursive processing.
//
// These components enable the carbon-core framework to handle Solana transaction instructions
// efficiently, decoding them into structured types and facilitating hierarchical processing.
package instruction

import (
	"context"
	"encoding/base64"
	"log/slog"
	"strings"
	"sync"

	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/filter"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/processor"
	"github.com/lugondev/go-carbon/pkg/types"
)

// MaxInstructionStackDepth is the maximum depth of instruction nesting.
// See: https://github.com/anza-xyz/agave/blob/master/program-runtime/src/execution_budget.rs#L7
const MaxInstructionStackDepth = 5

// InstructionMetadata contains metadata associated with a specific instruction,
// including transaction-level details.
//
// InstructionMetadata is utilized within the pipeline to associate each instruction
// with the broader context of its transaction, as well as its position within the
// instruction stack.
type InstructionMetadata struct {
	// TransactionMetadata provides details of the entire transaction.
	TransactionMetadata *TransactionMetadataRef

	// StackHeight represents the instruction's depth within the stack, where 1 is the root level.
	StackHeight uint32

	// Index is the index of the instruction in the transaction.
	// The index is relative within stack height and is 1-based.
	Index uint32

	// AbsolutePath represents the instruction's position in the nested structure.
	AbsolutePath []uint8
}

// TransactionMetadataRef is a reference to transaction metadata.
// This avoids circular imports with the transaction package.
type TransactionMetadataRef struct {
	Slot        uint64
	Signature   types.Signature
	FeePayer    types.Pubkey
	LogMessages []string
	Meta        *types.TransactionStatusMeta
}

// GetSlot returns the transaction slot.
func (t *TransactionMetadataRef) GetSlot() uint64 {
	return t.Slot
}

// GetSignature returns the transaction signature.
func (t *TransactionMetadataRef) GetSignature() types.Signature {
	return t.Signature
}

// GetFeePayer returns the fee payer's public key.
func (t *TransactionMetadataRef) GetFeePayer() types.Pubkey {
	return t.FeePayer
}

// logType represents the type of a log message.
type logType int

const (
	logTypeStart logType = iota
	logTypeData
	logTypeCU
	logTypeFinish
)

// DecodeLogEvents decodes log events of type T thrown by this instruction.
// Returns all successful events of the type T decoded from the logs of the instruction.
func (m *InstructionMetadata) DecodeLogEvents(deserializer func([]byte) (any, error)) []any {
	logData := m.extractEventLogData()
	var events []any

	for _, data := range logData {
		if len(data) < 8 {
			continue
		}
		if event, err := deserializer(data); err == nil && event != nil {
			events = append(events, event)
		}
	}

	return events
}

// extractEventLogData extracts the data from log messages associated with this instruction.
// This method filters the transaction's log messages to return only those
// that correspond to the current instruction, based on its stack height and
// absolute path within the instruction stack.
func (m *InstructionMetadata) extractEventLogData() [][]byte {
	if m.TransactionMetadata == nil || len(m.TransactionMetadata.LogMessages) == 0 {
		return nil
	}

	var extractedLogs [][]byte
	currentStackHeight := 0
	lastStackHeight := 0
	positionAtLevel := make(map[int]uint8)

	for _, log := range m.TransactionMetadata.LogMessages {
		parsedLog := m.parseLog(log)

		switch parsedLog.logType {
		case logTypeStart:
			currentStackHeight = parsedLog.stackHeight

			var currentPos uint8
			if currentStackHeight > lastStackHeight {
				currentPos = 0
			} else if pos, ok := positionAtLevel[currentStackHeight]; ok {
				currentPos = pos + 1
			} else {
				currentPos = 0
			}

			positionAtLevel[currentStackHeight] = currentPos
			lastStackHeight = currentStackHeight

		case logTypeFinish:
			if currentStackHeight > 0 {
				currentStackHeight--
			}
		}

		// Build current path
		currentPath := make([]uint8, 0, currentStackHeight)
		for level := 1; level <= currentStackHeight; level++ {
			if pos, ok := positionAtLevel[level]; ok {
				currentPath = append(currentPath, pos)
			} else {
				currentPath = append(currentPath, 0)
			}
		}

		// Check if current path matches absolute path and log is data
		if parsedLog.logType == logTypeData && pathsEqual(currentPath, m.AbsolutePath) {
			// Extract the base64 data from the log
			parts := strings.Fields(log)
			if len(parts) > 0 {
				dataStr := parts[len(parts)-1]
				if decoded, err := base64.StdEncoding.DecodeString(dataStr); err == nil {
					extractedLogs = append(extractedLogs, decoded)
				}
			}
		}
	}

	return extractedLogs
}

// parsedLogResult holds the result of parsing a log line.
type parsedLogResult struct {
	logType     logType
	stackHeight int
}

// parseLog parses a log line to determine its type.
func (m *InstructionMetadata) parseLog(log string) parsedLogResult {
	// Check for invoke log
	if strings.HasPrefix(log, "Program ") && strings.Contains(log, " invoke [") {
		parts := strings.Fields(log)
		if len(parts) >= 4 && parts[0] == "Program" && parts[2] == "invoke" {
			levelStr := strings.TrimPrefix(strings.TrimSuffix(parts[3], "]"), "[")
			var level int
			if err := parseUint(levelStr, &level); err == nil {
				return parsedLogResult{logType: logTypeStart, stackHeight: level}
			}
		}
	}

	// Check for success/failure log
	if strings.HasPrefix(log, "Program ") && (strings.HasSuffix(log, " success") || strings.Contains(log, " failed")) {
		parts := strings.Fields(log)
		if len(parts) >= 3 && parts[0] == "Program" {
			return parsedLogResult{logType: logTypeFinish}
		}
	}

	// Check for compute units log
	if strings.Contains(log, "consumed") && strings.Contains(log, "compute units") {
		return parsedLogResult{logType: logTypeCU}
	}

	return parsedLogResult{logType: logTypeData}
}

// parseUint parses a string to an int.
func parseUint(s string, result *int) error {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return &parseError{s}
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return nil
}

type parseError struct {
	s string
}

func (e *parseError) Error() string {
	return "invalid number: " + e.s
}

// pathsEqual compares two byte slices for equality.
func pathsEqual(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// InstructionsWithMetadata is a list of instructions with their metadata.
type InstructionsWithMetadata []InstructionWithMetadata

// InstructionWithMetadata pairs an instruction with its metadata.
type InstructionWithMetadata struct {
	Metadata    *InstructionMetadata
	Instruction *types.Instruction
}

// DecodedInstruction represents a decoded instruction containing program ID, data, and associated accounts.
//
// Type parameter T is the type representing the decoded data for the instruction.
type DecodedInstruction[T any] struct {
	// ProgramID is the program ID that owns the instruction.
	ProgramID types.Pubkey

	// Data is the decoded data payload for the instruction.
	Data T

	// Accounts is a list of accounts involved in the instruction.
	Accounts []types.AccountMeta
}

// InstructionDecoder defines an interface for decoding Solana instructions into structured types.
//
// Type parameter T is the type into which the instruction data will be decoded.
type InstructionDecoder[T any] interface {
	// DecodeInstruction decodes a raw Solana Instruction into a DecodedInstruction.
	// Returns nil if the instruction cannot be decoded by this decoder.
	DecodeInstruction(instruction *types.Instruction) *DecodedInstruction[T]
}

// InstructionDecoderFunc is a function type that implements InstructionDecoder.
type InstructionDecoderFunc[T any] func(instruction *types.Instruction) *DecodedInstruction[T]

// DecodeInstruction implements InstructionDecoder interface.
func (f InstructionDecoderFunc[T]) DecodeInstruction(instruction *types.Instruction) *DecodedInstruction[T] {
	return f(instruction)
}

// InstructionProcessorInput is the input type for the instruction processor.
type InstructionProcessorInput[T any] struct {
	// Metadata contains information about the instruction.
	Metadata *InstructionMetadata

	// DecodedInstruction contains the decoded instruction data.
	DecodedInstruction *DecodedInstruction[T]

	// InnerInstructions contains nested instructions.
	InnerInstructions *NestedInstructions

	// RawInstruction is the original instruction.
	RawInstruction *types.Instruction
}

// InstructionPipe is a processing pipeline for instructions, using a decoder and processor.
//
// Type parameter T is the type representing the decoded instruction data.
type InstructionPipe[T any] struct {
	// Decoder is used for parsing instructions.
	Decoder InstructionDecoder[T]

	// Processor handles decoded instructions.
	Processor processor.Processor[InstructionProcessorInput[T]]

	// Filters determine which instruction updates should be processed.
	Filters []filter.Filter

	// Logger is used for logging (optional).
	Logger *slog.Logger
}

// NewInstructionPipe creates a new InstructionPipe.
func NewInstructionPipe[T any](
	decoder InstructionDecoder[T],
	proc processor.Processor[InstructionProcessorInput[T]],
) *InstructionPipe[T] {
	return &InstructionPipe[T]{
		Decoder:   decoder,
		Processor: proc,
		Filters:   make([]filter.Filter, 0),
		Logger:    slog.Default(),
	}
}

// NewInstructionPipeWithFilters creates a new InstructionPipe with filters.
func NewInstructionPipeWithFilters[T any](
	decoder InstructionDecoder[T],
	proc processor.Processor[InstructionProcessorInput[T]],
	filters []filter.Filter,
) *InstructionPipe[T] {
	return &InstructionPipe[T]{
		Decoder:   decoder,
		Processor: proc,
		Filters:   filters,
		Logger:    slog.Default(),
	}
}

// WithLogger sets a custom logger for the InstructionPipe.
func (p *InstructionPipe[T]) WithLogger(logger *slog.Logger) *InstructionPipe[T] {
	p.Logger = logger
	return p
}

// GetFilters returns the filters associated with this pipe.
func (p *InstructionPipe[T]) GetFilters() []filter.Filter {
	return p.Filters
}

// Run processes a NestedInstruction, recursively processing any inner instructions.
func (p *InstructionPipe[T]) Run(
	ctx context.Context,
	nestedInstruction *NestedInstruction,
	metricsCollection *metrics.Collection,
) error {
	p.Logger.Debug("InstructionPipe.Run",
		"program_id", nestedInstruction.Instruction.ProgramID.String(),
		"stack_height", nestedInstruction.Metadata.StackHeight,
	)

	// Try to decode the instruction
	decoded := p.Decoder.DecodeInstruction(nestedInstruction.Instruction)
	if decoded != nil {
		input := InstructionProcessorInput[T]{
			Metadata:           nestedInstruction.Metadata,
			DecodedInstruction: decoded,
			InnerInstructions:  nestedInstruction.InnerInstructions,
			RawInstruction:     nestedInstruction.Instruction,
		}

		if err := p.Processor.Process(ctx, input, metricsCollection); err != nil {
			return err
		}
	}

	// Recursively process inner instructions
	for _, innerInstruction := range nestedInstruction.InnerInstructions.Instructions {
		if err := p.Run(ctx, innerInstruction, metricsCollection); err != nil {
			return err
		}
	}

	return nil
}

// InstructionPipeRunner is an interface for running instruction pipes.
// This allows for type-erased storage of InstructionPipe instances.
type InstructionPipeRunner interface {
	// RunInstruction processes a nested instruction.
	RunInstruction(
		ctx context.Context,
		nestedInstruction *NestedInstruction,
		metricsCollection *metrics.Collection,
	) error

	// GetFilters returns the filters for this pipe.
	GetFilters() []filter.Filter
}

// Ensure InstructionPipe implements InstructionPipeRunner.
var _ InstructionPipeRunner = (*InstructionPipe[any])(nil)

// RunInstruction implements InstructionPipeRunner interface.
func (p *InstructionPipe[T]) RunInstruction(
	ctx context.Context,
	nestedInstruction *NestedInstruction,
	metricsCollection *metrics.Collection,
) error {
	return p.Run(ctx, nestedInstruction, metricsCollection)
}

// NestedInstruction represents a nested instruction with metadata, including potential inner instructions.
//
// The NestedInstruction struct allows for recursive instruction handling, where each instruction
// may have associated metadata and a list of nested instructions.
type NestedInstruction struct {
	// Metadata is the metadata associated with the instruction.
	Metadata *InstructionMetadata

	// Instruction is the Solana instruction being processed.
	Instruction *types.Instruction

	// InnerInstructions is a list of nested instructions.
	InnerInstructions *NestedInstructions
}

// GetProgramID returns the program ID of the instruction.
func (n *NestedInstruction) GetProgramID() types.Pubkey {
	return n.Instruction.ProgramID
}

// GetData returns the data of the instruction.
func (n *NestedInstruction) GetData() []byte {
	return n.Instruction.Data
}

// NestedInstructions is a collection of nested instructions.
type NestedInstructions struct {
	Instructions []*NestedInstruction
}

// NewNestedInstructions creates a new empty NestedInstructions.
func NewNestedInstructions() *NestedInstructions {
	return &NestedInstructions{
		Instructions: make([]*NestedInstruction, 0),
	}
}

// Len returns the number of instructions.
func (n *NestedInstructions) Len() int {
	return len(n.Instructions)
}

// IsEmpty returns true if there are no instructions.
func (n *NestedInstructions) IsEmpty() bool {
	return n.Len() == 0
}

// Push adds a nested instruction.
func (n *NestedInstructions) Push(instruction *NestedInstruction) {
	n.Instructions = append(n.Instructions, instruction)
}

// Get returns the instruction at the given index.
func (n *NestedInstructions) Get(index int) filter.NestedInstruction {
	if index < 0 || index >= len(n.Instructions) {
		return nil
	}
	return n.Instructions[index]
}

// NestInstructions organizes instructions into a nested structure based on stack height.
//
// This function organizes instructions into a nested structure, enabling hierarchical
// transaction analysis. Instructions are nested according to their stack height,
// forming a tree-like structure.
func NestInstructions(instructions InstructionsWithMetadata) *NestedInstructions {
	if len(instructions) == 0 {
		return NewNestedInstructions()
	}

	builder := newNestedBuilder(instructions)
	return builder.build(instructions)
}

// nestedBuilder builds a nested instruction structure.
type nestedBuilder struct {
	nestedIxs *NestedInstructions
	levelPtrs [MaxInstructionStackDepth]*NestedInstruction
	mu        sync.Mutex
}

// newNestedBuilder creates a new nested builder.
func newNestedBuilder(instructions InstructionsWithMetadata) *nestedBuilder {
	// Count root level instructions for capacity estimation
	capacity := 0
	for _, instr := range instructions {
		if instr.Metadata.StackHeight == 1 {
			capacity++
		}
	}

	return &nestedBuilder{
		nestedIxs: &NestedInstructions{
			Instructions: make([]*NestedInstruction, 0, capacity),
		},
	}
}

// build builds the nested instruction structure.
func (b *nestedBuilder) build(instructions InstructionsWithMetadata) *NestedInstructions {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, item := range instructions {
		metadata := item.Metadata
		instruction := item.Instruction
		stackHeight := int(metadata.StackHeight)

		if stackHeight <= 0 || stackHeight > MaxInstructionStackDepth {
			continue
		}

		// Clear pointers for levels at or above current stack height
		for i := stackHeight - 1; i < MaxInstructionStackDepth; i++ {
			b.levelPtrs[i] = nil
		}

		newInstruction := &NestedInstruction{
			Metadata:          metadata,
			Instruction:       instruction,
			InnerInstructions: NewNestedInstructions(),
		}

		if stackHeight == 1 {
			// Root level instruction
			b.nestedIxs.Push(newInstruction)
			b.levelPtrs[0] = newInstruction
		} else if parentPtr := b.levelPtrs[stackHeight-2]; parentPtr != nil {
			// Nested instruction - add to parent's inner instructions
			parentPtr.InnerInstructions.Push(newInstruction)
			b.levelPtrs[stackHeight-1] = newInstruction
		}
	}

	return b.nestedIxs
}

// MultiInstructionPipe manages multiple instruction pipes and routes updates to all of them.
type MultiInstructionPipe struct {
	pipes  []InstructionPipeRunner
	logger *slog.Logger
}

// NewMultiInstructionPipe creates a new MultiInstructionPipe.
func NewMultiInstructionPipe() *MultiInstructionPipe {
	return &MultiInstructionPipe{
		pipes:  make([]InstructionPipeRunner, 0),
		logger: slog.Default(),
	}
}

// AddPipe adds an instruction pipe to the multi-pipe.
func (m *MultiInstructionPipe) AddPipe(pipe InstructionPipeRunner) {
	m.pipes = append(m.pipes, pipe)
}

// WithLogger sets a custom logger.
func (m *MultiInstructionPipe) WithLogger(logger *slog.Logger) *MultiInstructionPipe {
	m.logger = logger
	return m
}

// Run processes a nested instruction through all pipes.
func (m *MultiInstructionPipe) Run(
	ctx context.Context,
	datasourceID datasource.DatasourceID,
	nestedInstruction *NestedInstruction,
	metricsCollection *metrics.Collection,
) error {
	for _, pipe := range m.pipes {
		if !filter.CheckInstructionFilters(datasourceID, pipe.GetFilters(), nestedInstruction) {
			continue
		}

		if err := pipe.RunInstruction(ctx, nestedInstruction, metricsCollection); err != nil {
			return err
		}
	}
	return nil
}

// Len returns the number of pipes in the multi-pipe.
func (m *MultiInstructionPipe) Len() int {
	return len(m.pipes)
}

// ProgramInstructionDecoder is a helper for creating instruction decoders that filter by program ID.
type ProgramInstructionDecoder[T any] struct {
	// ProgramID is the expected program ID for instructions this decoder handles.
	ProgramID types.Pubkey

	// DecodeFunc is the function that decodes the instruction data.
	DecodeFunc func(data []byte) (T, error)
}

// NewProgramInstructionDecoder creates a new ProgramInstructionDecoder.
func NewProgramInstructionDecoder[T any](
	programID types.Pubkey,
	decodeFunc func(data []byte) (T, error),
) *ProgramInstructionDecoder[T] {
	return &ProgramInstructionDecoder[T]{
		ProgramID:  programID,
		DecodeFunc: decodeFunc,
	}
}

// DecodeInstruction implements InstructionDecoder interface.
// It only decodes instructions for the specified program.
func (d *ProgramInstructionDecoder[T]) DecodeInstruction(instruction *types.Instruction) *DecodedInstruction[T] {
	// Check if instruction is for the expected program
	if instruction.ProgramID != d.ProgramID {
		return nil
	}

	// Attempt to decode the instruction data
	data, err := d.DecodeFunc(instruction.Data)
	if err != nil {
		return nil
	}

	return &DecodedInstruction[T]{
		ProgramID: instruction.ProgramID,
		Data:      data,
		Accounts:  instruction.Accounts,
	}
}

// CompositeInstructionDecoder tries multiple decoders in sequence.
// It returns the result from the first decoder that succeeds.
type CompositeInstructionDecoder[T any] struct {
	decoders []InstructionDecoder[T]
}

// NewCompositeInstructionDecoder creates a new CompositeInstructionDecoder.
func NewCompositeInstructionDecoder[T any](decoders ...InstructionDecoder[T]) *CompositeInstructionDecoder[T] {
	return &CompositeInstructionDecoder[T]{
		decoders: decoders,
	}
}

// AddDecoder adds a decoder to the composite.
func (c *CompositeInstructionDecoder[T]) AddDecoder(decoder InstructionDecoder[T]) {
	c.decoders = append(c.decoders, decoder)
}

// DecodeInstruction implements InstructionDecoder interface.
func (c *CompositeInstructionDecoder[T]) DecodeInstruction(instruction *types.Instruction) *DecodedInstruction[T] {
	for _, decoder := range c.decoders {
		if result := decoder.DecodeInstruction(instruction); result != nil {
			return result
		}
	}
	return nil
}
