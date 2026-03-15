package oktest

import (
	"io"
	"net/http"
	"testing"
)

func TestTransportHandler_DefaultHandler(t *testing.T) {
	th := &TransportHandler{}

	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := th.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusNotImplemented {
		t.Errorf("expected %d, got %d", http.StatusNotImplemented, resp.StatusCode)
	}
}

func TestTransportHandler_SetHandler(t *testing.T) {
	th := &TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := th.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if string(body) != "hello" {
		t.Errorf("expected %q, got %q", "hello", string(body))
	}
}

func TestTransportHandler_Handler(t *testing.T) {
	th := &TransportHandler{}

	if th.Handler() != nil {
		t.Error("expected nil handler initially")
	}

	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {})

	if th.Handler() == nil {
		t.Error("expected non-nil handler after SetHandler")
	}
}

func TestTransportHandler_SwapHandler(t *testing.T) {
	th := &TransportHandler{}

	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := th.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Swap handler.
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	resp, err = th.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
