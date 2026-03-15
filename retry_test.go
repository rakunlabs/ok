package ok_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rakunlabs/ok"
)

func TestRetry_Success(t *testing.T) {
	var attempts atomic.Int32

	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithRetryMax(4),
		ok.WithRetryWaitMin(10*time.Millisecond),
		ok.WithRetryWaitMax(50*time.Millisecond),
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

	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetry_MaxExhausted(t *testing.T) {
	var attempts atomic.Int32

	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithRetryMax(2),
		ok.WithRetryWaitMin(10*time.Millisecond),
		ok.WithRetryWaitMax(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, func(resp *http.Response) error {
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", resp.StatusCode)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 initial + 2 retries = 3
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetry_DisabledRetry(t *testing.T) {
	var attempts atomic.Int32

	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
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

	err = client.Do(req, func(resp *http.Response) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retry), got %d", attempts.Load())
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	var attempts atomic.Int32

	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithRetryMax(10),
		ok.WithRetryWaitMin(100*time.Millisecond),
		ok.WithRetryWaitMax(200*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, func(resp *http.Response) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}

	// Should have made at least 1 attempt but not all 10.
	if attempts.Load() >= 10 {
		t.Errorf("expected fewer than 10 attempts due to context timeout, got %d", attempts.Load())
	}
}

func TestRetry_PerRequestDisable(t *testing.T) {
	var attempts atomic.Int32

	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithRetryMax(5),
		ok.WithRetryWaitMin(10*time.Millisecond),
		ok.WithRetryWaitMax(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := ok.CtxWithRetryPolicy(context.Background(),
		ok.OptionRetry.WithRetryDisable(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, func(resp *http.Response) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (per-request disable), got %d", attempts.Load())
	}
}

func TestRetry_TooManyRequests(t *testing.T) {
	var attempts atomic.Int32

	th := &transportHandler{}
	th.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	client, err := ok.New(
		ok.WithBaseTransport(th),
		ok.WithRetryMax(3),
		ok.WithRetryWaitMin(10*time.Millisecond),
		ok.WithRetryWaitMax(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = client.Do(req, func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
}

func TestDefaultBackoff(t *testing.T) {
	min := 100 * time.Millisecond
	max := 10 * time.Second

	for attempt := 0; attempt < 10; attempt++ {
		d := ok.DefaultBackoff(min, max, attempt)
		if d < min/2 {
			t.Errorf("attempt %d: backoff %v is less than min/2 %v", attempt, d, min/2)
		}
		if d > max {
			t.Errorf("attempt %d: backoff %v exceeds max %v", attempt, d, max)
		}
	}
}
