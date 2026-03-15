package ok

import (
	"context"
	"net/http"
	"net/url"
)

type contextKey string

// TransportHeaderKey is the context key used to pass per-request headers
// via context. Headers stored under this key are merged into the request
// before sending.
const TransportHeaderKey contextKey = "ok-transport-header"

// CtxWithHeader returns a new context with per-request headers attached.
// These headers are applied by TransportOK before each request.
func CtxWithHeader(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, TransportHeaderKey, header)
}

// TransportOK is the outermost transport layer that handles
// base URL resolution, default headers, context headers, and inject functions.
type TransportOK struct {
	// Base is the underlying RoundTripper to delegate to.
	Base http.RoundTripper
	// Header contains default headers applied to requests if not already set.
	Header http.Header
	// BaseURL is used to resolve relative request URLs.
	BaseURL *url.URL
	// Inject is called before each request for custom modifications
	// (e.g., tracing propagation).
	Inject func(ctx context.Context, req *http.Request)
}

// RoundTrip implements http.RoundTripper.
// It clones the request, resolves the URL, applies headers, and delegates.
func (t *TransportOK) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request per RoundTripper contract.
	reqClone := req.Clone(req.Context())

	// Resolve URL against base URL.
	if t.BaseURL != nil {
		reqClone.URL = t.BaseURL.ResolveReference(reqClone.URL)
		reqClone.Host = reqClone.URL.Host
	}

	// Apply context headers.
	if ctxHeader, ok := req.Context().Value(TransportHeaderKey).(http.Header); ok {
		for k, vs := range ctxHeader {
			for _, v := range vs {
				reqClone.Header.Add(k, v)
			}
		}
	}

	// Call inject function.
	if t.Inject != nil {
		t.Inject(reqClone.Context(), reqClone)
	}

	// Apply default headers (only if not already set on the request).
	t.setDefaultHeaders(reqClone)

	return t.Base.RoundTrip(reqClone)
}

// setDefaultHeaders applies default headers to the request only if
// the request does not already have a value for the given key.
func (t *TransportOK) setDefaultHeaders(req *http.Request) {
	for k, vs := range t.Header {
		if req.Header.Get(k) == "" {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
}
