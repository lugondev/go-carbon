package view

import (
	"encoding/binary"
	"testing"

	"github.com/gagliardetto/solana-go"
)

func createTestAccountBuffer() []byte {
	// Layout: pubkey(32) + lamports(8) + data_len(8) + data(32) + owner(32) + executable(1) + rent_epoch(8)
	// Total: 32 + 8 + 8 + 32 + 32 + 1 + 8 = 121 bytes
	buf := make([]byte, 121)

	// Pubkey (offset 0-31)
	pubkey := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	copy(buf[0:32], pubkey[:])

	// Lamports (offset 32-39)
	binary.LittleEndian.PutUint64(buf[32:40], 1000000000)

	// Data length (offset 40-47)
	dataLen := uint64(32)
	binary.LittleEndian.PutUint64(buf[40:48], dataLen)

	// Data (offset 48-79)
	for i := 0; i < 32; i++ {
		buf[48+i] = byte(i)
	}

	// Owner (offset 80-111)
	owner := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	copy(buf[80:112], owner[:])

	// Executable (offset 112)
	buf[112] = 1

	// Rent epoch (offset 113-120)
	binary.LittleEndian.PutUint64(buf[113:121], 12345)

	return buf
}

func TestAccountView(t *testing.T) {
	buf := createTestAccountBuffer()
	view := NewAccountView(buf)

	pubkey := view.Pubkey()
	if pubkey.IsZero() {
		t.Error("Expected non-zero pubkey")
	}

	lamports := view.Lamports()
	if lamports != 1000000000 {
		t.Errorf("Expected lamports 1000000000, got %d", lamports)
	}

	data := view.Data()
	if len(data) != 32 {
		t.Errorf("Expected data length 32, got %d", len(data))
	}

	owner := view.Owner()
	expectedOwner := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	if !owner.Equals(expectedOwner) {
		t.Errorf("Expected owner %s, got %s", expectedOwner, owner)
	}

	if !view.IsExecutable() {
		t.Error("Expected executable to be true")
	}

	rentEpoch := view.RentEpoch()
	if rentEpoch != 12345 {
		t.Errorf("Expected rent epoch 12345, got %d", rentEpoch)
	}
}

func TestEventView(t *testing.T) {
	buf := make([]byte, 32)
	for i := range buf {
		buf[i] = byte(i)
	}

	view, err := NewEventView(buf)
	if err != nil {
		t.Fatalf("Failed to create event view: %v", err)
	}

	disc := view.Discriminator()
	for i := 0; i < 8; i++ {
		if disc[i] != byte(i) {
			t.Errorf("Discriminator byte %d: expected %d, got %d", i, i, disc[i])
		}
	}

	data := view.Data()
	if len(data) != 24 {
		t.Errorf("Expected data length 24, got %d", len(data))
	}

	fullData := view.FullData()
	if len(fullData) != 32 {
		t.Errorf("Expected full data length 32, got %d", len(fullData))
	}
}

func BenchmarkAccountView(b *testing.B) {
	buf := createTestAccountBuffer()

	b.Run("ZeroCopyView", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			view := NewAccountView(buf)
			_ = view.Pubkey()
			_ = view.Lamports()
			_ = view.Data()
			_ = view.Owner()
		}
	})

	b.Run("TraditionalParsing", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			pubkey := make([]byte, 32)
			copy(pubkey, buf[0:32])

			lamports := binary.LittleEndian.Uint64(buf[32:40])
			dataLen := binary.LittleEndian.Uint64(buf[40:48])

			data := make([]byte, dataLen)
			copy(data, buf[48:48+dataLen])

			owner := make([]byte, 32)
			ownerOffset := 48 + int(dataLen)
			copy(owner, buf[ownerOffset:ownerOffset+32])

			_ = pubkey
			_ = lamports
			_ = data
			_ = owner
		}
	})
}

func BenchmarkEventView(b *testing.B) {
	buf := make([]byte, 64)
	for i := range buf {
		buf[i] = byte(i)
	}

	b.Run("ZeroCopyView", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			view, _ := NewEventView(buf)
			_ = view.Discriminator()
			_ = view.Data()
		}
	})

	b.Run("TraditionalParsing", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			disc := make([]byte, 8)
			copy(disc, buf[:8])

			data := make([]byte, len(buf)-8)
			copy(data, buf[8:])

			_ = disc
			_ = data
		}
	})
}
