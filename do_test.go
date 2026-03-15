package ok_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/rakunlabs/ok"
)

func TestDo_NilFunc(t *testing.T) {
	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, nil)
	if !errors.Is(err, ok.ErrResponseFuncNil) {
		t.Errorf("expected ErrResponseFuncNil, got %v", err)
	}
}

func TestDo_ResponseBodyDrained(t *testing.T) {
	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"key":"value"}`))
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Don't read the body in the callback — it should still be drained.
	err = client.Do(req, func(resp *http.Response) error {
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		// Intentionally not reading body.
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDo_ErrorFromCallback(t *testing.T) {
	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	callbackErr := errors.New("callback error")
	err = client.Do(req, func(resp *http.Response) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Errorf("expected callback error, got %v", err)
	}
}

func TestDo_PackageLevel(t *testing.T) {
	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`OK`))
	})

	httpClient := &http.Client{Transport: th}

	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = ok.Do(httpClient, req, func(resp *http.Response) error {
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResponseFuncJSON_Success(t *testing.T) {
	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"test","value":42}`))
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	err = client.Do(req, ok.ResponseFuncJSON(&result))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("expected name %q, got %q", "test", result.Name)
	}
	if result.Value != 42 {
		t.Errorf("expected value 42, got %d", result.Value)
	}
}

func TestResponseFuncJSON_NotFound(t *testing.T) {
	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
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
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	var respErr *ok.ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("expected *ResponseError, got %T: %v", err, err)
	}
	if respErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", respErr.StatusCode)
	}
}

func TestResponseFuncJSON_NilData(t *testing.T) {
	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"key":"value"}`))
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithDisableRetry(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nil data — just check status, don't decode.
	err = client.Do(req, ok.ResponseFuncJSON(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
