package view

import (
	"encoding/binary"
	"errors"
	"unsafe"

	"github.com/gagliardetto/solana-go"
)

var (
	ErrInvalidBuffer      = errors.New("invalid buffer size")
	ErrInvalidAccountData = errors.New("invalid account data")
)

type AccountView struct {
	buffer []byte
}

func NewAccountView(buffer []byte) *AccountView {
	return &AccountView{
		buffer: buffer,
	}
}

func (v *AccountView) Pubkey() solana.PublicKey {
	if len(v.buffer) < 32 {
		return solana.PublicKey{}
	}
	return *(*solana.PublicKey)(unsafe.Pointer(&v.buffer[0]))
}

func (v *AccountView) Lamports() uint64 {
	if len(v.buffer) < 40 {
		return 0
	}
	return binary.LittleEndian.Uint64(v.buffer[32:40])
}

func (v *AccountView) Data() []byte {
	if len(v.buffer) < 72 {
		return nil
	}
	dataLen := binary.LittleEndian.Uint64(v.buffer[40:48])
	start := 48
	end := start + int(dataLen)
	if end > len(v.buffer) {
		return nil
	}
	return v.buffer[start:end]
}

func (v *AccountView) Owner() solana.PublicKey {
	dataLen := binary.LittleEndian.Uint64(v.buffer[40:48])
	ownerOffset := 48 + int(dataLen)
	if ownerOffset+32 > len(v.buffer) {
		return solana.PublicKey{}
	}
	return *(*solana.PublicKey)(unsafe.Pointer(&v.buffer[ownerOffset]))
}

func (v *AccountView) IsExecutable() bool {
	dataLen := binary.LittleEndian.Uint64(v.buffer[40:48])
	execOffset := 48 + int(dataLen) + 32
	if execOffset >= len(v.buffer) {
		return false
	}
	return v.buffer[execOffset] != 0
}

func (v *AccountView) RentEpoch() uint64 {
	dataLen := binary.LittleEndian.Uint64(v.buffer[40:48])
	rentOffset := 48 + int(dataLen) + 32 + 1
	if rentOffset+8 > len(v.buffer) {
		return 0
	}
	return binary.LittleEndian.Uint64(v.buffer[rentOffset : rentOffset+8])
}

type EventView struct {
	buffer        []byte
	discriminator [8]byte
}

func NewEventView(buffer []byte) (*EventView, error) {
	if len(buffer) < 8 {
		return nil, ErrInvalidBuffer
	}

	var disc [8]byte
	copy(disc[:], buffer[:8])

	return &EventView{
		buffer:        buffer,
		discriminator: disc,
	}, nil
}

func (v *EventView) Discriminator() [8]byte {
	return v.discriminator
}

func (v *EventView) Data() []byte {
	if len(v.buffer) <= 8 {
		return nil
	}
	return v.buffer[8:]
}

func (v *EventView) FullData() []byte {
	return v.buffer
}
