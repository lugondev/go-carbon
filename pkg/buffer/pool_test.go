package buffer

import (
	"fmt"
	"testing"
)

func TestNextPowerOfTwo(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{1023, 1024},
		{1024, 1024},
		{1025, 2048},
	}

	for _, tt := range tests {
		result := nextPowerOfTwo(tt.input)
		if result != tt.expected {
			t.Errorf("nextPowerOfTwo(%d) = %d; want %d", tt.input, result, tt.expected)
		}
	}
}

func TestPoolGetPut(t *testing.T) {
	pool := NewPool()

	sizes := []int{64, 256, 1024, 4096}

	for _, size := range sizes {
		buf := pool.Get(size)
		if len(buf) != size {
			t.Errorf("Get(%d) returned buffer of length %d", size, len(buf))
		}
		if cap(buf) < size {
			t.Errorf("Get(%d) returned buffer with capacity %d", size, cap(buf))
		}

		pool.Put(buf)
	}
}

func TestPoolConcurrency(t *testing.T) {
	pool := NewPool()
	done := make(chan bool)
	workers := 10
	iterations := 1000

	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				buf := pool.Get(1024)
				buf[0] = byte(j)
				pool.Put(buf)
			}
			done <- true
		}()
	}

	for i := 0; i < workers; i++ {
		<-done
	}
}

func BenchmarkPoolGetPut(b *testing.B) {
	pool := NewPool()

	sizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := pool.Get(size)
				pool.Put(buf)
			}
		})
	}
}

func BenchmarkDirectAllocation(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := make([]byte, size)
				_ = buf
			}
		})
	}
}

func BenchmarkPoolVsAllocation(b *testing.B) {
	pool := NewPool()
	size := 1024

	b.Run("Pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := pool.Get(size)
			pool.Put(buf)
		}
	})

	b.Run("DirectAlloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := make([]byte, size)
			_ = buf
		}
	})
}
