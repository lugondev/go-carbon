package buffer

import (
	"math/bits"
	"sync"
)

type Pool struct {
	pools map[int]*sync.Pool
}

var globalPool = NewPool()

func NewPool() *Pool {
	p := &Pool{
		pools: make(map[int]*sync.Pool),
	}

	sizes := []int{
		64,
		256,
		1024,
		4 * 1024,
		16 * 1024,
		64 * 1024,
		256 * 1024,
		1024 * 1024,
	}

	for _, size := range sizes {
		poolSize := size
		p.pools[size] = &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, poolSize)
				return &buf
			},
		}
	}

	return p
}

func (p *Pool) Get(size int) []byte {
	if size <= 0 {
		return nil
	}

	poolSize := nextPowerOfTwo(size)

	if poolSize > 1024*1024 {
		return make([]byte, size)
	}

	pool, ok := p.pools[poolSize]
	if !ok {
		return make([]byte, size)
	}

	bufPtr := pool.Get().(*[]byte)
	buf := *bufPtr
	return buf[:size]
}

func (p *Pool) Put(buf []byte) {
	if buf == nil || cap(buf) == 0 {
		return
	}

	poolSize := cap(buf)

	if poolSize > 1024*1024 {
		return
	}

	pool, ok := p.pools[poolSize]
	if !ok {
		return
	}

	buf = buf[:cap(buf)]
	for i := range buf {
		buf[i] = 0
	}

	pool.Put(&buf)
}

func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 0
	}
	if n&(n-1) == 0 {
		return n
	}
	return 1 << bits.Len(uint(n))
}

func GetBuffer(size int) []byte {
	return globalPool.Get(size)
}

func PutBuffer(buf []byte) {
	globalPool.Put(buf)
}
