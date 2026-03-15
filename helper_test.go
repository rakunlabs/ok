package ok_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
)

// transportHandler is a thread-safe fake http.RoundTripper backed by
// httptest.ResponseRecorder for use in tests.
type transportHandler struct {
	mu      sync.RWMutex
	handler http.HandlerFunc
}

func (t *transportHandler) SetHandler(handler http.HandlerFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handler = handler
}

func (t *transportHandler) RoundTrip(req *http.Request) (*http.Response, error) {
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
