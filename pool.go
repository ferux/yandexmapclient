package yandexmapclient

import (
	"bytes"
	"sync"
)

type bytesPool struct {
	pool        sync.Pool
	maxDataSize int
	log         ModuleLogger
}

func newCachedPool(maxDataSize int, log Logger) *bytesPool {
	var ml ModuleLogger

	switch tl := log.(type) {
	case nil:
		ml = nopLogger{}
	case ModuleLogger:
		ml = tl
	case Logger:
		ml = ModuleLoggerWrapper(log)
	}

	return &bytesPool{
		maxDataSize: maxDataSize,
		log:         ml.Module("pool"),
		pool: sync.Pool{
			New: func() interface{} {
				data := make([]byte, 0, maxDataSize)

				return bytes.NewBuffer(data)
			},
		},
	}
}

// Get bytes from pool.
func (c *bytesPool) Get() (out *bytes.Buffer) {
	out = c.pool.Get().(*bytes.Buffer)

	return out
}

func (c *bytesPool) Put(data *bytes.Buffer) {
	if cp := data.Cap(); cp > c.maxDataSize {
		c.log.Debugf("capacity %d larger than %d", cp, c.maxDataSize)

		return
	}

	data.Reset()
	// nolint: staticcheck
	c.pool.Put(data)
}

// Apply takes byte slice from the pool and passes to f. After that it puts
// this slice back to the pool.
func (c *bytesPool) Apply(f func(*bytes.Buffer)) {
	data := c.Get()

	f(data)

	c.Put(data)
}
