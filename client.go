package ok

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client is the main HTTP client wrapper.
// It provides retryable requests, base URL resolution, default headers,
// and automatic response body draining.
type Client struct {
	// HTTP is the underlying *http.Client.
	// It can be used directly for advanced use cases.
	HTTP *http.Client
}

// New creates a new Client with the given options.
// The transport chain is built as follows (innermost to outermost):
//  1. Base transport (*http.Transport from stdlib or user-provided)
//  2. HTTP/2, proxy, TLS settings applied to base transport
//  3. Per-attempt timeout transport (if RetryTimeout > 0 and not HTTP/2)
//  4. Retry transport (if retry not disabled)
//  5. TransportOK (base URL, default headers, inject)
//  6. User RoundTripper wrappers (via WithRoundTripper)
func New(opts ...OptionClientFn) (*Client, error) {
	o := defaults()
	for _, opt := range opts {
		opt(o)
	}

	// Apply environment variables (lower precedence than explicit options).
	applyEnvValues(o)

	// Apply UserAgent as a header if set.
	if o.UserAgent != "" {
		if o.Header.Get("User-Agent") == "" {
			o.Header.Set("User-Agent", o.UserAgent)
		}
	}

	// Determine base transport.
	var baseTransport http.RoundTripper
	if o.HTTPClient != nil && o.HTTPClient.Transport != nil {
		baseTransport = o.HTTPClient.Transport
	} else if o.BaseTransport != nil {
		baseTransport = o.BaseTransport
	} else {
		baseTransport = defaultTransport()
	}

	// Apply settings to the transport if it's an *http.Transport.
	if transport, ok := baseTransport.(*http.Transport); ok {
		// HTTP/2 support.
		if o.HTTP2 {
			transport.ForceAttemptHTTP2 = true
			var protocols http.Protocols
			protocols.SetUnencryptedHTTP2(true)
			transport.Protocols = &protocols

			o.Logger.Debug("http2 enabled", "h2c", true)
		}

		// Proxy configuration (ignored for HTTP/2).
		if o.Proxy != "" && !o.HTTP2 {
			proxyURL, err := url.Parse(o.Proxy)
			if err != nil {
				return nil, fmt.Errorf("invalid proxy URL %q: %w", o.Proxy, err)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
			o.Logger.Debug("proxy configured", "url", o.Proxy)
		}

		// TLS configuration.
		if o.TLSConfig != nil {
			transport.TLSClientConfig = o.TLSConfig
		} else if o.InsecureSkipVerify {
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = &tls.Config{}
			}
			transport.TLSClientConfig.InsecureSkipVerify = true
			o.Logger.Warn("TLS certificate verification disabled")
		}

		baseTransport = transport
	}

	// Layer 1: Per-attempt timeout (skipped for HTTP/2).
	var currentTransport http.RoundTripper = baseTransport
	if o.RetryTimeout > 0 && !o.HTTP2 {
		currentTransport = &retryTimeoutTransport{
			Base:    currentTransport,
			Timeout: o.RetryTimeout,
		}
		o.Logger.Debug("per-attempt timeout configured", "timeout", o.RetryTimeout)
	}

	// Layer 2: Retry transport.
	if !o.DisableRetry {
		retryPolicy := o.RetryPolicy
		if retryPolicy == nil {
			retryPolicy = DefaultRetryPolicy
		}
		currentTransport = &retryTransport{
			Base:         currentTransport,
			RetryMax:     o.RetryMax,
			RetryWaitMin: o.RetryWaitMin,
			RetryWaitMax: o.RetryWaitMax,
			CheckRetry:   retryPolicy,
			Backoff:      o.Backoff,
			Logger:       o.Logger,
			RetryLog:     o.RetryLog,
		}
		o.Logger.Debug("retry enabled",
			"max", o.RetryMax,
			"wait_min", o.RetryWaitMin,
			"wait_max", o.RetryWaitMax,
		)
	}

	// Parse base URL.
	var baseURL *url.URL
	if o.BaseURL != "" {
		var err error
		baseURL, err = url.Parse(o.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid base URL %q: %w", o.BaseURL, err)
		}
		if o.EnableBaseURLCheck {
			if baseURL.Scheme == "" || baseURL.Host == "" {
				return nil, fmt.Errorf("base URL %q must have scheme and host", o.BaseURL)
			}
		}
		o.Logger.Debug("base URL configured", "url", o.BaseURL)
	}

	// Layer 3: TransportOK (base URL, headers, inject).
	currentTransport = &TransportOK{
		Base:    currentTransport,
		Header:  o.Header.Clone(),
		BaseURL: baseURL,
		Inject:  o.Inject,
	}

	// Layer 4: User RoundTripper wrappers.
	for _, fn := range o.RoundTripperList {
		var err error
		currentTransport, err = fn(context.Background(), currentTransport)
		if err != nil {
			return nil, fmt.Errorf("round tripper wrapper error: %w", err)
		}
	}

	// Build the final HTTP client.
	httpClient := &http.Client{
		Transport: currentTransport,
	}
	if o.Timeout > 0 {
		httpClient.Timeout = o.Timeout
	}

	return &Client{HTTP: httpClient}, nil
}

// defaultTransport creates a default http.Transport with sensible settings.
func defaultTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
