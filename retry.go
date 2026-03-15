package ok

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

// retryTransport is an http.RoundTripper that automatically retries
// failed requests according to a configurable policy.
type retryTransport struct {
	// Base is the underlying RoundTripper.
	Base http.RoundTripper

	// RetryMax is the maximum number of retries (not counting the initial attempt).
	RetryMax int

	// RetryWaitMin is the minimum time to wait between retries.
	RetryWaitMin time.Duration

	// RetryWaitMax is the maximum time to wait between retries.
	RetryWaitMax time.Duration

	// CheckRetry is called after each attempt to determine if a retry should occur.
	CheckRetry func(ctx context.Context, resp *http.Response, err error) (bool, error)

	// Backoff computes the wait duration for a given attempt.
	// If nil, DefaultBackoff is used.
	Backoff func(min, max time.Duration, attempt int) time.Duration

	// Logger for retry events.
	Logger Logger

	// RetryLog enables logging of retry attempts.
	RetryLog bool
}

// RoundTrip implements http.RoundTripper with retry logic.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp    *http.Response
		err     error
		bodyBuf []byte
		hasBody bool
	)

	// Buffer the request body for re-sending on retry.
	if req.Body != nil {
		bodyBuf, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		hasBody = true
	}

	checkRetry := t.CheckRetry
	if checkRetry == nil {
		checkRetry = DefaultRetryPolicy
	}

	backoff := t.Backoff
	if backoff == nil {
		backoff = DefaultBackoff
	}

	for attempt := 0; ; attempt++ {
		// Clone the request for each attempt.
		reqClone := req.Clone(req.Context())
		if hasBody {
			reqClone.Body = io.NopCloser(bytes.NewReader(bodyBuf))
			reqClone.ContentLength = int64(len(bodyBuf))
			reqClone.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBuf)), nil
			}
		}

		// Execute the request.
		resp, err = t.Base.RoundTrip(reqClone)

		// Check if we should retry.
		shouldRetry, checkErr := checkRetry(req.Context(), resp, err)
		if checkErr != nil {
			// Policy returned an error (e.g., context cancelled).
			if resp != nil {
				DrainBody(resp.Body)
			}
			return nil, checkErr
		}

		if !shouldRetry || attempt >= t.RetryMax {
			break
		}

		// Log retry attempt.
		if t.RetryLog && t.Logger != nil {
			var bodyExcerpt string
			if resp != nil {
				bodyExcerpt = string(LimitedResponse(resp))
			}
			t.Logger.Warn("retrying request",
				"attempt", attempt+1,
				"max", t.RetryMax,
				"method", req.Method,
				"url", req.URL.String(),
				"error", err,
				"response_body", bodyExcerpt,
			)
		}

		// Drain the response body from the failed attempt.
		if resp != nil {
			DrainBody(resp.Body)
		}

		// Calculate wait duration and sleep.
		wait := backoff(t.RetryWaitMin, t.RetryWaitMax, attempt)

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}

	return resp, err
}

// DefaultBackoff implements exponential backoff with full jitter.
// The formula is: sleep = random(min, min(max, min * 2^attempt))
func DefaultBackoff(min, max time.Duration, attempt int) time.Duration {
	mult := math.Pow(2, float64(attempt))
	sleep := time.Duration(float64(min) * mult)
	if sleep > max {
		sleep = max
	}
	if sleep < min {
		sleep = min
	}

	// Add jitter: random value between [sleep/2, sleep)
	jitter := rand.Int64N(int64(sleep) / 2) //nolint:gosec
	return sleep/2 + time.Duration(jitter)
}

// retryTimeoutTransport wraps a RoundTripper with per-attempt timeouts.
// Each individual HTTP attempt is cancelled after the configured duration.
type retryTimeoutTransport struct {
	Base    http.RoundTripper
	Timeout time.Duration
}

// RoundTrip implements http.RoundTripper with per-attempt timeout.
func (t *retryTimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), t.Timeout)
	defer cancel()

	return t.Base.RoundTrip(req.WithContext(ctx))
}
