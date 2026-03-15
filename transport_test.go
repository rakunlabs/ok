package ok_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rakunlabs/ok"
	"github.com/rakunlabs/ok/oktest"
)

func TestTransport_ContextHeaders(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"x-trace": r.Header.Get("X-Trace-Id"),
		})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	header := http.Header{}
	header.Set("X-Trace-Id", "trace-abc-123")
	ctx := ok.CtxWithHeader(context.Background(), header)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	err = client.Do(req, ok.ResponseFuncJSON(&result))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["x-trace"] != "trace-abc-123" {
		t.Errorf("expected x-trace %q, got %q", "trace-abc-123", result["x-trace"])
	}
}

func TestTransport_InjectFunction(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"x-injected": r.Header.Get("X-Injected"),
		})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
		ok.WithInject(func(ctx context.Context, req *http.Request) {
			req.Header.Set("X-Injected", "injected-value")
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	err = client.Do(req, ok.ResponseFuncJSON(&result))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["x-injected"] != "injected-value" {
		t.Errorf("expected x-injected %q, got %q", "injected-value", result["x-injected"])
	}
}

func TestTransport_DefaultHeaderNotOverridden(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"x-custom": r.Header.Get("X-Custom"),
		})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
		ok.WithHeaderSet("X-Custom", "default-value"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Request explicitly sets the header — default should NOT override it.
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req.Header.Set("X-Custom", "request-value")

	var result map[string]string
	err = client.Do(req, ok.ResponseFuncJSON(&result))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["x-custom"] != "request-value" {
		t.Errorf("expected request-set header %q, got %q", "request-value", result["x-custom"])
	}
}

func TestTransport_BaseURLResolution(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"url": r.URL.String(),
		})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
		ok.WithBaseURL("https://api.example.com/v1"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/users?page=1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	err = client.Do(req, ok.ResponseFuncJSON(&result))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// URL resolution: base "https://api.example.com/v1" + "/users?page=1"
	// ResolveReference with an absolute path replaces the path.
	expected := "https://api.example.com/users?page=1"
	if result["url"] != expected {
		t.Errorf("expected URL %q, got %q", expected, result["url"])
	}
}

func TestTransport_RoundTripperWrapper(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"x-wrapper": r.Header.Get("X-Wrapper"),
		})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
		ok.WithRoundTripper(func(ctx context.Context, rt http.RoundTripper) (http.RoundTripper, error) {
			return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				req = req.Clone(req.Context())
				req.Header.Set("X-Wrapper", "wrapped")
				return rt.RoundTrip(req)
			}), nil
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	err = client.Do(req, ok.ResponseFuncJSON(&result))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["x-wrapper"] != "wrapped" {
		t.Errorf("expected x-wrapper %q, got %q", "wrapped", result["x-wrapper"])
	}
}

// roundTripperFunc is a helper to create an http.RoundTripper from a function.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
