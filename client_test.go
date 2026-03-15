package ok_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakunlabs/ok"
	"github.com/rakunlabs/ok/oktest"
)

func TestNew_Default(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client, err := ok.New(
		ok.WithBaseURL(server.URL),
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

	if result["status"] != "ok" {
		t.Errorf("expected status ok, got %v", result["status"])
	}
}

func TestNew_WithTransportHandler(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
		})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/api/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	err = client.Do(req, ok.ResponseFuncJSON(&result))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["method"] != "GET" {
		t.Errorf("expected method GET, got %v", result["method"])
	}
	if result["path"] != "/api/test" {
		t.Errorf("expected path /api/test, got %v", result["path"])
	}
}

func TestNew_WithDefaultHeaders(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"x-custom": r.Header.Get("X-Custom"),
			"x-api":    r.Header.Get("X-Api-Key"),
		})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
		ok.WithHeaderSet("X-Custom", "custom-value"),
		ok.WithHeaderSet("X-Api-Key", "secret"),
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

	if result["x-custom"] != "custom-value" {
		t.Errorf("expected x-custom header %q, got %q", "custom-value", result["x-custom"])
	}
	if result["x-api"] != "secret" {
		t.Errorf("expected x-api header %q, got %q", "secret", result["x-api"])
	}
}

func TestNew_WithBaseURL(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"host": r.Host,
			"path": r.URL.Path,
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

	req, err := http.NewRequest(http.MethodGet, "/users", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	err = client.Do(req, ok.ResponseFuncJSON(&result))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["host"] != "api.example.com" {
		t.Errorf("expected host %q, got %q", "api.example.com", result["host"])
	}
}

func TestNew_InvalidBaseURL(t *testing.T) {
	_, err := ok.New(
		ok.WithBaseURL("not-a-valid-url-missing-scheme"),
		ok.WithEnableBaseURLCheck(true),
	)
	if err == nil {
		t.Fatal("expected error for invalid base URL with check enabled")
	}
}

func TestNew_WithUserAgent(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"user-agent": r.Header.Get("User-Agent"),
		})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
		ok.WithUserAgent("ok/1.0"),
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

	if result["user-agent"] != "ok/1.0" {
		t.Errorf("expected user-agent %q, got %q", "ok/1.0", result["user-agent"])
	}
}

func TestNew_ConfigNew(t *testing.T) {
	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	disableRetry := true
	cfg := &ok.Config{
		DisableRetry: &disableRetry,
	}

	client, err := cfg.New(ok.WithBaseTransport(th))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, func(resp *http.Response) error {
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
