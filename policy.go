package ok

import (
	"context"
	"net"
	"net/http"
)

const ctxKeyRetryPolicy contextKey = "ok-retry-policy"

// OptionRetryFn is a functional option for per-request retry configuration.
type OptionRetryFn func(*optionRetryValue)

// optionRetryValue holds per-request retry overrides.
type optionRetryValue struct {
	DisableRetry        bool
	DisabledStatusCodes []int
	EnabledStatusCodes  []int
}

// OptionRetryHolder provides a namespace for retry option constructors.
type OptionRetryHolder struct{}

// OptionRetry is the singleton for accessing retry option constructors.
var OptionRetry OptionRetryHolder

// WithRetryDisable disables retry for this request.
func (OptionRetryHolder) WithRetryDisable() OptionRetryFn {
	return func(o *optionRetryValue) {
		o.DisableRetry = true
	}
}

// WithRetryDisabledStatusCodes sets status codes that should NOT trigger a retry.
func (OptionRetryHolder) WithRetryDisabledStatusCodes(codes ...int) OptionRetryFn {
	return func(o *optionRetryValue) {
		o.DisabledStatusCodes = codes
	}
}

// WithRetryEnabledStatusCodes sets status codes that should FORCE a retry.
func (OptionRetryHolder) WithRetryEnabledStatusCodes(codes ...int) OptionRetryFn {
	return func(o *optionRetryValue) {
		o.EnabledStatusCodes = codes
	}
}

// NewRetryValue creates an optionRetryValue with the given options.
func NewRetryValue(opts ...OptionRetryFn) *optionRetryValue {
	v := &optionRetryValue{}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// CtxWithRetryPolicy attaches retry options to a context for per-request
// retry configuration. These options are checked by the retry policy
// on each attempt.
func CtxWithRetryPolicy(ctx context.Context, opts ...OptionRetryFn) context.Context {
	return context.WithValue(ctx, ctxKeyRetryPolicy, NewRetryValue(opts...))
}

// DefaultRetryPolicy is the standard retry policy.
// It retries on:
//   - 5xx status codes (server errors)
//   - 429 status code (too many requests)
//   - Timeout errors (context.DeadlineExceeded, net.Error timeout)
//   - Connection errors
//
// It does not retry on:
//   - Context cancellation
//   - 4xx status codes (except 429)
//   - Successful responses (2xx, 3xx)
//
// Per-request overrides are supported via CtxWithRetryPolicy.
func DefaultRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// Never retry if the context was cancelled.
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// Check for per-request retry overrides.
	if v, ok := ctx.Value(ctxKeyRetryPolicy).(*optionRetryValue); ok && v != nil {
		if v.DisableRetry {
			return false, nil
		}

		if resp != nil {
			// Check disabled status codes.
			for _, code := range v.DisabledStatusCodes {
				if resp.StatusCode == code {
					return false, nil
				}
			}
			// Check enabled status codes (force retry).
			for _, code := range v.EnabledStatusCodes {
				if resp.StatusCode == code {
					return true, nil
				}
			}
		}
	}

	// If there is an error, check if it's retryable.
	if err != nil {
		if isTimeoutError(err) {
			return true, nil
		}
		// Connection refused, reset, etc.
		if isConnectionError(err) {
			return true, nil
		}
		return false, nil
	}

	// Check response status codes.
	if resp != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			return true, nil
		}
		if resp.StatusCode == 0 || resp.StatusCode >= 500 {
			return true, nil
		}
	}

	return false, nil
}

// isTimeoutError checks if the error is a timeout error.
func isTimeoutError(err error) bool {
	if err == context.DeadlineExceeded {
		return true
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return false
}

// isConnectionError checks if the error indicates a connection failure.
func isConnectionError(err error) bool {
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	return false
}
