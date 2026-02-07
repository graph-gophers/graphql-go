package resolvable

import (
	"bytes"
	"sync"
	"testing"
)

func testBufferPool() *bufferPool {
	return &bufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		maxBufferCap: 1024,
	}
}

func TestBufferPool(t *testing.T) {
	s := testBufferPool()

	t.Run("resets buffer before returning", func(t *testing.T) {
		buf := s.Get()
		buf.WriteString("test data")
		s.Put(buf)

		buf2 := s.Get()
		if buf2.Len() != 0 {
			t.Errorf("expected reset buffer, got length %d", buf2.Len())
		}
		s.Put(buf2)
	})

	t.Run("does not pool oversized buffers", func(t *testing.T) {
		buf := s.Get()
		large := make([]byte, 1025)
		buf.Write(large)

		if buf.Cap() <= s.maxBufferCap {
			t.Skip("buffer didn't grow large enough for test")
		}

		s.Put(buf)

		buf2 := s.Get()
		if buf2 == buf {
			t.Errorf("oversized buffer was added to pool")
		}
		s.Put(buf2)
	})

	t.Run("respects zero max cap to disable pooling", func(t *testing.T) {
		noPool := &bufferPool{
			pool: sync.Pool{
				New: func() interface{} {
					return new(bytes.Buffer)
				},
			},
			maxBufferCap: 0,
		}

		buf := noPool.Get()
		buf.WriteString("test")
		noPool.Put(buf)

		buf2 := noPool.Get()
		if buf2 == buf {
			t.Errorf("buffer was pooled when maxBufferCap is 0")
		}
	})
}
