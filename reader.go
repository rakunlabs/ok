package ok

import (
	"context"
	"io"
)

// MultiReader concatenates multiple io.ReadCloser streams into a single
// io.ReadCloser. Readers are consumed in order; when one is exhausted,
// reading continues with the next. Close closes all underlying readers.
type MultiReader struct {
	readers []io.ReadCloser
	current int
	ctx     context.Context
}

// NewMultiReader creates a MultiReader that reads from the provided
// readers in sequence.
func NewMultiReader(rs ...io.ReadCloser) *MultiReader {
	return &MultiReader{
		readers: rs,
	}
}

// SetContext sets a context for cancellation support.
// If the context is cancelled, Read returns the context error.
func (mr *MultiReader) SetContext(ctx context.Context) {
	mr.ctx = ctx
}

// Read implements io.Reader. It reads from the current reader,
// advancing to the next when one is exhausted.
func (mr *MultiReader) Read(p []byte) (int, error) {
	if mr.ctx != nil {
		if err := mr.ctx.Err(); err != nil {
			return 0, err
		}
	}

	for mr.current < len(mr.readers) {
		n, err := mr.readers[mr.current].Read(p)
		if n > 0 {
			return n, nil
		}
		if err != io.EOF {
			return 0, err
		}
		// Current reader exhausted, move to next.
		mr.current++
	}

	return 0, io.EOF
}

// Close closes all underlying readers and returns the first error encountered.
func (mr *MultiReader) Close() error {
	var firstErr error
	for _, r := range mr.readers {
		if err := r.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
