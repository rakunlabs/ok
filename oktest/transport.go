// Package oktest provides test utilities for ok HTTP client testing.
//
// TransportHandler is a fake http.RoundTripper that dispatches requests
// to a configurable http.HandlerFunc without starting a real HTTP server.
// This enables fast, deterministic unit tests for API wrappers.
//
// Usage:
//
//	th := &oktest.TransportHandler{}
//	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
//	    w.WriteHeader(http.StatusOK)
//	    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
//	})
//
//	client, err := ok.New(
//	    ok.WithBaseTransport(th),
//	    ok.WithDisableRetry(true),
//	    ok.WithDisableBaseURLCheck(true),
//	)
package oktest

import (
	"net/http"
	"net/http/httptest"
	"sync"
)

// TransportHandler is a thread-safe fake http.RoundTripper backed by
// httptest.ResponseRecorder. It dispatches requests to a configurable
// http.HandlerFunc without requiring a running HTTP server.
type TransportHandler struct {
	mu      sync.RWMutex
	handler http.HandlerFunc
}

// SetHandler sets the handler function that will process requests.
// This is thread-safe and can be changed between test cases.
func (t *TransportHandler) SetHandler(handler http.HandlerFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handler = handler
}

// Handler returns the currently configured handler function.
func (t *TransportHandler) Handler() http.HandlerFunc {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.handler
}

// RoundTrip implements http.RoundTripper.
// It creates an httptest.ResponseRecorder, calls the handler, and converts
// the recorded response into an *http.Response.
func (t *TransportHandler) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.RLock()
	handler := t.handler
	t.mu.RUnlock()

	if handler == nil {
		handler = func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		}
	}

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec.Result(), nil
}
