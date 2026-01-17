package resolvable

import (
	"bytes"
	"sync"
)

type Pool[T any] interface {
	Get() T
	Put(T)
}

// bufferPool is a pool of bytes.Buffers
// Avoids allocating new buffers for each resolver or field execution.
type bufferPool struct {
	pool         sync.Pool
	maxBufferCap int
}

func (p *bufferPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (p *bufferPool) Put(buf *bytes.Buffer) {
	if buf.Cap() > p.maxBufferCap {
		return
	}
	p.pool.Put(buf)
}

func newBufferPool(maxBufferCap int) *bufferPool {
	return &bufferPool{
		pool: sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
		maxBufferCap: maxBufferCap,
	}
}
