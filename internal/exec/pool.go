package exec

import (
	"bytes"
	"sync"
)

const (
	maxBufferCap    = 64 * 1024
	maxFieldMapSize = 128
	newFieldMapSize = 16
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *bytes.Buffer) {
	if buf.Cap() > maxBufferCap {
		return
	}
	bufferPool.Put(buf)
}

func copyBuffer(buf *bytes.Buffer) []byte {
	if buf.Len() == 0 {
		return nil
	}
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result
}

var fieldMapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]*fieldToExec, newFieldMapSize)
	},
}

func getFieldMap() map[string]*fieldToExec {
	return fieldMapPool.Get().(map[string]*fieldToExec)
}

func putFieldMap(m map[string]*fieldToExec) {
	if len(m) > maxFieldMapSize {
		return
	}
	for k := range m {
		delete(m, k)
	}
	fieldMapPool.Put(m)
}
