package exec

import (
	"testing"
)

func TestBufferPool(t *testing.T) {
	t.Run("resets buffer before returning", func(t *testing.T) {
		buf := getBuffer()
		buf.WriteString("test data")
		putBuffer(buf)

		buf2 := getBuffer()
		if buf2.Len() != 0 {
			t.Errorf("expected reset buffer, got length %d", buf2.Len())
		}
		putBuffer(buf2)
	})

	t.Run("copyBuffer copies data correctly", func(t *testing.T) {
		buf := getBuffer()
		buf.WriteString("test data")

		copied := copyBuffer(buf)
		if string(copied) != "test data" {
			t.Errorf("expected 'test data', got %q", string(copied))
		}

		// Original buffer should be unchanged
		if buf.String() != "test data" {
			t.Errorf("original buffer modified")
		}

		putBuffer(buf)
	})

	t.Run("copyBuffer returns nil for empty buffer", func(t *testing.T) {
		buf := getBuffer()
		copied := copyBuffer(buf)
		if copied != nil {
			t.Errorf("expected nil for empty buffer, got %v", copied)
		}
		putBuffer(buf)
	})

	t.Run("does not pool oversized buffers", func(t *testing.T) {
		buf := getBuffer()
		large := make([]byte, 65*1024)
		buf.Write(large)

		if buf.Cap() <= maxBufferCap {
			t.Skip("buffer didn't grow large enough for test")
		}

		putBuffer(buf)

		buf2 := getBuffer()
		if buf2 == buf {
			t.Errorf("oversized buffer was added to the pool")
		}
		putBuffer(buf2)
	})
}

func TestFieldMapPool(t *testing.T) {
	t.Run("clears map before returning", func(t *testing.T) {
		m := getFieldMap()
		m["test"] = &fieldToExec{}
		m["foo"] = &fieldToExec{}
		putFieldMap(m)

		m2 := getFieldMap()
		if len(m2) != 0 {
			t.Errorf("expected cleared map, got length %d", len(m2))
		}
		putFieldMap(m2)
	})

	t.Run("does not pool oversized maps", func(t *testing.T) {
		m := getFieldMap()
		for i := 0; i < 129; i++ {
			m[string(rune(i))] = &fieldToExec{}
		}

		putFieldMap(m) // Should not be added to pool

		m2 := getFieldMap()
		if len(m2) != 0 {
			t.Errorf("got non-empty map from pool, length: %d", len(m2))
		}
		putFieldMap(m2)
	})
}
