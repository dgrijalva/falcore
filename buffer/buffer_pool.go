package buffer

import (
	"bufio"
	"io"
	"io/ioutil"
)

// uses a chan as a leaky bucket buffer pool
type Pool struct {
	// size of buffer when creating new ones
	bufSize int
	// the actual pool of buffers ready for reuse
	pool chan *PoolEntry
}

// This is what's stored in the buffer.  It allows
// for the underlying io.Reader to be changed out
// inside a bufio.Reader.  This is required for reuse.
type PoolEntry struct {
	Br     *bufio.Reader
	source io.Reader
}

// make bufferPoolEntry a passthrough io.Reader
func (bpe *PoolEntry) Read(p []byte) (n int, err error) {
	return bpe.source.Read(p)
}

func NewPool(poolSize, bufferSize int) *Pool {
	return &Pool{
		bufSize: bufferSize,
		pool:    make(chan *PoolEntry, poolSize),
	}
}

// Take a buffer from the pool and set 
// it up to read from r
func (p *Pool) Take(r io.Reader) (bpe *PoolEntry) {
	select {
	case bpe = <-p.pool:
		// prepare for reuse
		if a := bpe.Br.Buffered(); a > 0 {
			// drain the internal buffer
			io.CopyN(ioutil.Discard, bpe.Br, int64(a))
		}
		// swap out the underlying reader
		bpe.source = r
	default:
		// none available.  create a new one
		bpe = &PoolEntry{nil, r}
		bpe.Br = bufio.NewReaderSize(bpe, p.bufSize)
	}
	return
}

// Return a buffer to the pool
func (p *Pool) Give(bpe *PoolEntry) {
	select {
	case p.pool <- bpe: // return to pool
	default: // discard
	}
}
