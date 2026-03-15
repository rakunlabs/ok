package ok

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"time"
)

// OptionClientFn is a functional option for configuring the Client.
type OptionClientFn func(*optionClientValue)

// RoundTripperFunc is a function that wraps a RoundTripper with additional behavior.
// It receives the context and the current transport, and returns a new transport.
type RoundTripperFunc func(ctx context.Context, rt http.RoundTripper) (http.RoundTripper, error)

// optionClientValue holds all configurable values for the client.
type optionClientValue struct {
	// HTTP client
	HTTPClient    *http.Client
	BaseTransport http.RoundTripper

	// URL and headers
	BaseURL            string
	EnableBaseURLCheck bool
	Header             http.Header
	UserAgent          string

	// Retry
	DisableRetry bool
	RetryMax     int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
	RetryTimeout time.Duration
	RetryPolicy  func(ctx context.Context, resp *http.Response, err error) (bool, error)
	Backoff      func(min, max time.Duration, attempt int) time.Duration
	RetryLog     bool

	// Timeout
	Timeout time.Duration

	// TLS
	InsecureSkipVerify bool
	TLSConfig          *tls.Config

	// Transport
	RoundTripperList []RoundTripperFunc
	Proxy            string
	HTTP2            bool
	Inject           func(ctx context.Context, req *http.Request)

	// Logging
	Logger Logger

	// Environment
	EnableEnvValues bool
}

// defaults returns an optionClientValue with sensible defaults.
func defaults() *optionClientValue {
	return &optionClientValue{
		RetryMax:     4,
		RetryWaitMin: 1 * time.Second,
		RetryWaitMax: 30 * time.Second,
		RetryLog:     true,
		Header:       make(http.Header),
		Logger:       slog.Default(),
	}
}

// --- URL and Headers ---

// WithBaseURL sets the base URL for all requests.
// Relative request URLs will be resolved against this base.
func WithBaseURL(url string) OptionClientFn {
	return func(o *optionClientValue) {
		o.BaseURL = url
	}
}

// WithEnableBaseURLCheck enables validation of the base URL during client construction.
// By default, base URL validation is disabled.
func WithEnableBaseURLCheck(enable bool) OptionClientFn {
	return func(o *optionClientValue) {
		o.EnableBaseURLCheck = enable
	}
}

// WithHeader sets the default headers for all requests.
// These headers are applied only if not already set on the request.
func WithHeader(header http.Header) OptionClientFn {
	return func(o *optionClientValue) {
		o.Header = header.Clone()
	}
}

// WithHeaderAdd adds a header value to the default headers.
// Multiple values can be added for the same key.
func WithHeaderAdd(key, value string) OptionClientFn {
	return func(o *optionClientValue) {
		o.Header.Add(key, value)
	}
}

// WithHeaderSet sets a header value in the default headers,
// replacing any existing values for the key.
func WithHeaderSet(key, value string) OptionClientFn {
	return func(o *optionClientValue) {
		o.Header.Set(key, value)
	}
}

// WithHeaderDel removes a header key from the default headers.
func WithHeaderDel(key string) OptionClientFn {
	return func(o *optionClientValue) {
		o.Header.Del(key)
	}
}

// WithUserAgent sets the User-Agent header for all requests.
func WithUserAgent(ua string) OptionClientFn {
	return func(o *optionClientValue) {
		o.UserAgent = ua
	}
}

// --- HTTP Client ---

// WithHTTPClient sets a custom *http.Client as the base.
// The client's transport will be used as the base transport.
func WithHTTPClient(client *http.Client) OptionClientFn {
	return func(o *optionClientValue) {
		o.HTTPClient = client
	}
}

// WithBaseTransport sets the base http.RoundTripper.
// This is the innermost transport in the chain.
func WithBaseTransport(rt http.RoundTripper) OptionClientFn {
	return func(o *optionClientValue) {
		o.BaseTransport = rt
	}
}

// --- Retry ---

// WithDisableRetry disables automatic retry behavior.
func WithDisableRetry(disable bool) OptionClientFn {
	return func(o *optionClientValue) {
		o.DisableRetry = disable
	}
}

// WithRetryMax sets the maximum number of retry attempts.
// Default is 4.
func WithRetryMax(max int) OptionClientFn {
	return func(o *optionClientValue) {
		o.RetryMax = max
	}
}

// WithRetryWaitMin sets the minimum wait time between retries.
// Default is 1 second.
func WithRetryWaitMin(d time.Duration) OptionClientFn {
	return func(o *optionClientValue) {
		o.RetryWaitMin = d
	}
}

// WithRetryWaitMax sets the maximum wait time between retries.
// Default is 30 seconds.
func WithRetryWaitMax(d time.Duration) OptionClientFn {
	return func(o *optionClientValue) {
		o.RetryWaitMax = d
	}
}

// WithRetryTimeout sets the per-attempt timeout.
// Each individual HTTP attempt will be cancelled after this duration.
// A zero value means no per-attempt timeout (only the overall client timeout applies).
// This is skipped when HTTP/2 is enabled.
func WithRetryTimeout(d time.Duration) OptionClientFn {
	return func(o *optionClientValue) {
		o.RetryTimeout = d
	}
}

// WithRetryPolicy sets a custom retry policy function.
// The function receives the context, response, and error from each attempt
// and returns whether to retry and any error to propagate.
func WithRetryPolicy(policy func(ctx context.Context, resp *http.Response, err error) (bool, error)) OptionClientFn {
	return func(o *optionClientValue) {
		o.RetryPolicy = policy
	}
}

// WithBackoff sets a custom backoff function.
// It receives min wait, max wait, and the current attempt number (0-indexed),
// and returns the duration to wait before the next attempt.
func WithBackoff(backoff func(min, max time.Duration, attempt int) time.Duration) OptionClientFn {
	return func(o *optionClientValue) {
		o.Backoff = backoff
	}
}

// WithRetryLog enables or disables logging of retry attempts.
// Default is true.
func WithRetryLog(enable bool) OptionClientFn {
	return func(o *optionClientValue) {
		o.RetryLog = enable
	}
}

// --- Timeout ---

// WithTimeout sets the overall timeout for the HTTP client.
// This is the total time allowed for all retry attempts combined.
func WithTimeout(d time.Duration) OptionClientFn {
	return func(o *optionClientValue) {
		o.Timeout = d
	}
}

// --- TLS ---

// WithInsecureSkipVerify disables TLS certificate verification.
// This should only be used for testing or development.
func WithInsecureSkipVerify(skip bool) OptionClientFn {
	return func(o *optionClientValue) {
		o.InsecureSkipVerify = skip
	}
}

// WithTLSConfig sets a custom *tls.Config for the transport.
func WithTLSConfig(cfg *tls.Config) OptionClientFn {
	return func(o *optionClientValue) {
		o.TLSConfig = cfg
	}
}

// --- Transport ---

// WithRoundTripper adds a RoundTripper wrapper to the transport chain.
// Wrappers are applied in order, outermost last.
func WithRoundTripper(fn RoundTripperFunc) OptionClientFn {
	return func(o *optionClientValue) {
		o.RoundTripperList = append(o.RoundTripperList, fn)
	}
}

// WithProxy sets the proxy URL for the HTTP transport.
// Ignored when HTTP/2 is enabled.
func WithProxy(proxy string) OptionClientFn {
	return func(o *optionClientValue) {
		o.Proxy = proxy
	}
}

// WithHTTP2 enables HTTP/2 support including h2c (unencrypted HTTP/2).
// When enabled, proxy settings are ignored and per-attempt retry timeout
// is skipped.
func WithHTTP2(enable bool) OptionClientFn {
	return func(o *optionClientValue) {
		o.HTTP2 = enable
	}
}

// WithInject sets a function that is called before each request is sent.
// This can be used for tracing propagation (e.g., OpenTelemetry) or
// other per-request modifications.
func WithInject(fn func(ctx context.Context, req *http.Request)) OptionClientFn {
	return func(o *optionClientValue) {
		o.Inject = fn
	}
}

// --- Logging ---

// WithLogger sets the logger for debug and retry logging.
// Default is a no-op logger (no output).
func WithLogger(logger Logger) OptionClientFn {
	return func(o *optionClientValue) {
		o.Logger = logger
	}
}

// --- Environment ---

// WithEnableEnvValues enables reading configuration from environment variables.
// By default, environment variable reading is disabled.
func WithEnableEnvValues(enable bool) OptionClientFn {
	return func(o *optionClientValue) {
		o.EnableEnvValues = enable
	}
}

// OptionsPre prepends pre-options before the user-supplied options.
// This is useful for setting default options that can be overridden.
func OptionsPre(opts []OptionClientFn, preOpts ...OptionClientFn) []OptionClientFn {
	return append(preOpts, opts...)
}
