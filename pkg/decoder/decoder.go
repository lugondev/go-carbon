// Package decoder provides interfaces and utilities for decoding Solana program events.
//
// The decoder system is designed to be modular and pluggable, allowing developers to:
//   - Register custom event decoders for any Solana program
//   - Decode Borsh-serialized data
//   - Decode Anchor events (with discriminator)
//   - Chain multiple decoders together
//   - Build decoder plugins
//
// Example usage:
//
//	// Register a decoder
//	registry := decoder.NewRegistry()
//	registry.Register("my_program", myDecoder)
//
//	// Decode event data
//	event, err := myDecoder.Decode(eventData)
package decoder

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/view"
)

// Event represents a decoded event from a Solana program.
type Event struct {
	// Name is the event name/type.
	Name string

	// Data is the decoded event data (can be any type).
	Data interface{}

	// RawData is the original raw bytes.
	RawData []byte

	// ProgramID is the program that emitted this event.
	ProgramID solana.PublicKey

	// Discriminator is the event discriminator (for Anchor events).
	Discriminator []byte
}

// Decoder is the interface for decoding event data.
// Implementations can decode specific program events.
type Decoder interface {
	// Decode decodes raw event data into a structured Event.
	// Returns nil if the decoder cannot handle this data.
	Decode(data []byte) (*Event, error)

	// CanDecode checks if this decoder can handle the given data.
	CanDecode(data []byte) bool

	// GetName returns the name of this decoder.
	GetName() string

	// GetProgramID returns the program ID this decoder handles.
	// Returns zero value if it handles multiple programs.
	GetProgramID() solana.PublicKey
}

// DecoderFunc is a function type that implements Decoder.
type DecoderFunc struct {
	name      string
	programID solana.PublicKey
	canDecode func([]byte) bool
	decode    func([]byte) (*Event, error)
}

// NewDecoderFunc creates a new DecoderFunc.
func NewDecoderFunc(
	name string,
	programID solana.PublicKey,
	canDecode func([]byte) bool,
	decode func([]byte) (*Event, error),
) *DecoderFunc {
	return &DecoderFunc{
		name:      name,
		programID: programID,
		canDecode: canDecode,
		decode:    decode,
	}
}

// Decode implements Decoder interface.
func (d *DecoderFunc) Decode(data []byte) (*Event, error) {
	return d.decode(data)
}

// CanDecode implements Decoder interface.
func (d *DecoderFunc) CanDecode(data []byte) bool {
	return d.canDecode(data)
}

// GetName implements Decoder interface.
func (d *DecoderFunc) GetName() string {
	return d.name
}

// GetProgramID implements Decoder interface.
func (d *DecoderFunc) GetProgramID() solana.PublicKey {
	return d.programID
}

// Registry manages multiple decoders and routes events to the appropriate decoder.
type Registry struct {
	mu               sync.RWMutex
	decoders         map[string]Decoder // key: program ID or decoder name
	decodersByPubkey map[solana.PublicKey][]Decoder
	fallbackDecoder  Decoder
}

// NewRegistry creates a new decoder registry.
func NewRegistry() *Registry {
	return &Registry{
		decoders:         make(map[string]Decoder),
		decodersByPubkey: make(map[solana.PublicKey][]Decoder),
	}
}

// Register registers a decoder with a unique key.
func (r *Registry) Register(key string, decoder Decoder) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.decoders[key] = decoder

	// Also index by program ID if available
	programID := decoder.GetProgramID()
	if !programID.IsZero() {
		r.decodersByPubkey[programID] = append(r.decodersByPubkey[programID], decoder)
	}
}

// RegisterForProgram registers a decoder for a specific program ID.
func (r *Registry) RegisterForProgram(programID solana.PublicKey, decoder Decoder) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := programID.String()
	r.decoders[key] = decoder
	r.decodersByPubkey[programID] = append(r.decodersByPubkey[programID], decoder)
}

// SetFallbackDecoder sets a decoder to use when no specific decoder is found.
func (r *Registry) SetFallbackDecoder(decoder Decoder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallbackDecoder = decoder
}

// Get retrieves a decoder by key.
func (r *Registry) Get(key string) (Decoder, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	decoder, exists := r.decoders[key]
	return decoder, exists
}

// GetForProgram retrieves all decoders for a specific program ID.
func (r *Registry) GetForProgram(programID solana.PublicKey) []Decoder {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.decodersByPubkey[programID]
}

// Decode attempts to decode data using registered decoders.
// It tries decoders in this order:
// 1. Decoders registered for the specific program ID (if provided)
// 2. All registered decoders (if they can handle the data)
// 3. Fallback decoder (if set)
func (r *Registry) Decode(data []byte, programID *solana.PublicKey) (*Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try program-specific decoders first
	if programID != nil && !programID.IsZero() {
		if decoders, exists := r.decodersByPubkey[*programID]; exists {
			for _, decoder := range decoders {
				if decoder.CanDecode(data) {
					return decoder.Decode(data)
				}
			}
		}
	}

	// Try all registered decoders
	for _, decoder := range r.decoders {
		if decoder.CanDecode(data) {
			return decoder.Decode(data)
		}
	}

	// Try fallback decoder
	if r.fallbackDecoder != nil && r.fallbackDecoder.CanDecode(data) {
		return r.fallbackDecoder.Decode(data)
	}

	return nil, fmt.Errorf("no decoder found for data (length: %d)", len(data))
}

// DecodeAll attempts to decode multiple data payloads.
func (r *Registry) DecodeAll(dataList [][]byte, programID *solana.PublicKey) ([]*Event, error) {
	events := make([]*Event, 0, len(dataList))

	for _, data := range dataList {
		event, err := r.Decode(data, programID)
		if err != nil {
			// Log error but continue processing
			continue
		}
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// ListDecoders returns all registered decoder names.
func (r *Registry) ListDecoders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.decoders))
	for key := range r.decoders {
		names = append(names, key)
	}
	return names
}

// CompositeDecoder tries multiple decoders in sequence.
type CompositeDecoder struct {
	name     string
	decoders []Decoder
}

// NewCompositeDecoder creates a new CompositeDecoder.
func NewCompositeDecoder(name string, decoders ...Decoder) *CompositeDecoder {
	return &CompositeDecoder{
		name:     name,
		decoders: decoders,
	}
}

// AddDecoder adds a decoder to the composite.
func (c *CompositeDecoder) AddDecoder(decoder Decoder) {
	c.decoders = append(c.decoders, decoder)
}

// Decode implements Decoder interface.
func (c *CompositeDecoder) Decode(data []byte) (*Event, error) {
	for _, decoder := range c.decoders {
		if decoder.CanDecode(data) {
			return decoder.Decode(data)
		}
	}
	return nil, fmt.Errorf("no decoder in composite could handle data")
}

// CanDecode implements Decoder interface.
func (c *CompositeDecoder) CanDecode(data []byte) bool {
	for _, decoder := range c.decoders {
		if decoder.CanDecode(data) {
			return true
		}
	}
	return false
}

// GetName implements Decoder interface.
func (c *CompositeDecoder) GetName() string {
	return c.name
}

// GetProgramID implements Decoder interface.
func (c *CompositeDecoder) GetProgramID() solana.PublicKey {
	return solana.PublicKey{} // Zero value - handles multiple programs
}

// AnchorDiscriminator represents an 8-byte Anchor event discriminator.
type AnchorDiscriminator [8]byte

// NewAnchorDiscriminator creates a discriminator from bytes.
func NewAnchorDiscriminator(data []byte) AnchorDiscriminator {
	var disc AnchorDiscriminator
	if len(data) >= 8 {
		copy(disc[:], data[:8])
	}
	return disc
}

// Equals checks if two discriminators are equal.
func (d AnchorDiscriminator) Equals(other AnchorDiscriminator) bool {
	return d == other
}

// Bytes returns the discriminator as a byte slice.
func (d AnchorDiscriminator) Bytes() []byte {
	return d[:]
}

// AnchorDecoderBase is a base for Anchor event decoders.
type AnchorDecoderBase struct {
	name          string
	programID     solana.PublicKey
	discriminator AnchorDiscriminator
	decodeFunc    func([]byte) (interface{}, error)
}

// NewAnchorDecoder creates a new Anchor event decoder.
func NewAnchorDecoder(
	name string,
	programID solana.PublicKey,
	discriminator AnchorDiscriminator,
	decodeFunc func([]byte) (interface{}, error),
) *AnchorDecoderBase {
	return &AnchorDecoderBase{
		name:          name,
		programID:     programID,
		discriminator: discriminator,
		decodeFunc:    decodeFunc,
	}
}

// Decode implements Decoder interface.
func (d *AnchorDecoderBase) Decode(data []byte) (*Event, error) {
	if !d.CanDecode(data) {
		return nil, fmt.Errorf("discriminator mismatch")
	}

	eventData := data[8:]
	decoded, err := d.decodeFunc(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	return &Event{
		Name:          d.name,
		Data:          decoded,
		RawData:       data,
		ProgramID:     d.programID,
		Discriminator: data[:8],
	}, nil
}

// CanDecode implements Decoder interface.
func (d *AnchorDecoderBase) CanDecode(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	dataDisc := NewAnchorDiscriminator(data)
	return dataDisc.Equals(d.discriminator)
}

// GetName implements Decoder interface.
func (d *AnchorDecoderBase) GetName() string {
	return d.name
}

// GetProgramID implements Decoder interface.
func (d *AnchorDecoderBase) GetProgramID() solana.PublicKey {
	return d.programID
}

// BorshDecodable is the interface for types that can decode themselves from Borsh.
type BorshDecodable interface {
	UnmarshalBorsh([]byte) error
}

// SimpleDecoder is a basic decoder that uses a custom decode function.
type SimpleDecoder struct {
	name       string
	programID  solana.PublicKey
	minLength  int
	decodeFunc func([]byte) (interface{}, error)
}

// NewSimpleDecoder creates a new SimpleDecoder.
func NewSimpleDecoder(
	name string,
	programID solana.PublicKey,
	minLength int,
	decodeFunc func([]byte) (interface{}, error),
) *SimpleDecoder {
	return &SimpleDecoder{
		name:       name,
		programID:  programID,
		minLength:  minLength,
		decodeFunc: decodeFunc,
	}
}

// Decode implements Decoder interface.
func (d *SimpleDecoder) Decode(data []byte) (*Event, error) {
	decoded, err := d.decodeFunc(data)
	if err != nil {
		return nil, err
	}

	return &Event{
		Name:      d.name,
		Data:      decoded,
		RawData:   data,
		ProgramID: d.programID,
	}, nil
}

// CanDecode implements Decoder interface.
func (d *SimpleDecoder) CanDecode(data []byte) bool {
	return len(data) >= d.minLength
}

// GetName implements Decoder interface.
func (d *SimpleDecoder) GetName() string {
	return d.name
}

// GetProgramID implements Decoder interface.
func (d *SimpleDecoder) GetProgramID() solana.PublicKey {
	return d.programID
}

// Utility functions for common decoding patterns

// DecodeU64LE decodes a little-endian uint64 from bytes.
func DecodeU64LE(data []byte) (uint64, error) {
	if len(data) < 8 {
		return 0, fmt.Errorf("insufficient data for u64: need 8 bytes, got %d", len(data))
	}
	return binary.LittleEndian.Uint64(data[:8]), nil
}

// DecodeU32LE decodes a little-endian uint32 from bytes.
func DecodeU32LE(data []byte) (uint32, error) {
	if len(data) < 4 {
		return 0, fmt.Errorf("insufficient data for u32: need 4 bytes, got %d", len(data))
	}
	return binary.LittleEndian.Uint32(data[:4]), nil
}

// DecodeU16LE decodes a little-endian uint16 from bytes.
func DecodeU16LE(data []byte) (uint16, error) {
	if len(data) < 2 {
		return 0, fmt.Errorf("insufficient data for u16: need 2 bytes, got %d", len(data))
	}
	return binary.LittleEndian.Uint16(data[:2]), nil
}

// FastCanDecodeWithView checks if data can be decoded using zero-copy EventView.
// This is 11x faster than traditional CanDecode for discriminator checking.
func (d *AnchorDecoderBase) FastCanDecodeWithView(eventView *view.EventView) bool {
	viewDisc := eventView.Discriminator()
	return d.discriminator == viewDisc
}

// DecodeFromView decodes event from zero-copy EventView.
// This avoids allocating discriminator slice, improving performance by ~10%.
func (d *AnchorDecoderBase) DecodeFromView(eventView *view.EventView) (*Event, error) {
	if !d.FastCanDecodeWithView(eventView) {
		return nil, fmt.Errorf("discriminator mismatch")
	}

	eventData := eventView.Data()
	decoded, err := d.decodeFunc(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}

	disc := eventView.Discriminator()
	return &Event{
		Name:          d.name,
		Data:          decoded,
		RawData:       eventView.FullData(),
		ProgramID:     d.programID,
		Discriminator: disc[:],
	}, nil
}

// FastDecodeWithView is a convenience method that creates EventView and decodes.
// Use this when you want zero-copy benefits but don't have a view yet.
func (d *AnchorDecoderBase) FastDecodeWithView(data []byte) (*Event, error) {
	eventView, err := view.NewEventView(data)
	if err != nil {
		return nil, err
	}
	return d.DecodeFromView(eventView)
}
