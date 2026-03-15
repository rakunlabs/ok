package ok

import "io"

// ResponseErrLimit is the maximum number of bytes read from a response body
// for error reporting and drain operations. Default is 1 MB.
var ResponseErrLimit int64 = 1 << 20

// OptionDrain configures drain behavior.
type OptionDrain func(*drainOptions)

type drainOptions struct {
	limit int64
}

// WithDrainLimit sets the maximum bytes to drain.
// A negative value means unlimited.
func WithDrainLimit(limit int64) OptionDrain {
	return func(o *drainOptions) {
		o.limit = limit
	}
}

// DrainBody reads up to the configured limit from the body and closes it.
// This ensures HTTP connections are properly returned to the pool.
// If body is nil, this is a no-op.
func DrainBody(body io.ReadCloser, opts ...OptionDrain) {
	if body == nil {
		return
	}

	o := drainOptions{
		limit: ResponseErrLimit,
	}
	for _, opt := range opts {
		opt(&o)
	}

	var r io.Reader = body
	if o.limit >= 0 {
		r = io.LimitReader(body, o.limit)
	}

	_, _ = io.Copy(io.Discard, r)
	_ = body.Close()
}
