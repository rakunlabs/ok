package ok_test

import (
	"net/http"
	"testing"

	"github.com/rakunlabs/ok"
	"github.com/rakunlabs/ok/oktest"
)

func TestEnv_BaseURL(t *testing.T) {
	t.Setenv("OK_BASE_URL", "https://env.example.com")

	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "env.example.com" {
			t.Errorf("expected host env.example.com, got %s", r.Host)
		}
		w.WriteHeader(http.StatusOK)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
		ok.WithEnableEnvValues(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, func(resp *http.Response) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnv_BaseURL_OptionTakesPrecedence(t *testing.T) {
	t.Setenv("OK_BASE_URL", "https://env.example.com")

	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "option.example.com" {
			t.Errorf("expected host option.example.com, got %s", r.Host)
		}
		w.WriteHeader(http.StatusOK)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
		ok.WithEnableEnvValues(true),
		ok.WithBaseURL("https://option.example.com"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, func(resp *http.Response) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnv_DisabledByDefault(t *testing.T) {
	t.Setenv("OK_BASE_URL", "https://env.example.com")

	th := &oktest.TransportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		// Env values are disabled by default, so the env base URL should NOT be used.
		w.WriteHeader(http.StatusOK)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "http://direct.example.com/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, func(resp *http.Response) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
