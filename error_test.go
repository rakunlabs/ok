package ok

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponseError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ResponseError
		contains []string
	}{
		{
			name: "status only",
			err:  &ResponseError{StatusCode: 500},
			contains: []string{
				"unexpected status code: 500",
			},
		},
		{
			name: "with body",
			err:  &ResponseError{StatusCode: 404, Body: "not found"},
			contains: []string{
				"unexpected status code: 404",
				"body: not found",
			},
		},
		{
			name: "with request id",
			err:  &ResponseError{StatusCode: 503, RequestID: "abc-123"},
			contains: []string{
				"unexpected status code: 503",
				"request_id: abc-123",
			},
		},
		{
			name: "all fields",
			err:  &ResponseError{StatusCode: 502, Body: "bad gateway", RequestID: "xyz"},
			contains: []string{
				"502",
				"bad gateway",
				"xyz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, c := range tt.contains {
				if !strings.Contains(msg, c) {
					t.Errorf("error message %q does not contain %q", msg, c)
				}
			}
		})
	}
}

func TestUnexpectedResponse_Success(t *testing.T) {
	for _, code := range []int{200, 201, 204, 299} {
		rec := httptest.NewRecorder()
		rec.WriteHeader(code)
		resp := rec.Result()

		if err := UnexpectedResponse(resp); err != nil {
			t.Errorf("expected nil error for status %d, got %v", code, err)
		}
	}
}

func TestUnexpectedResponse_Error(t *testing.T) {
	for _, code := range []int{400, 401, 404, 500, 503} {
		rec := httptest.NewRecorder()
		rec.WriteHeader(code)
		resp := rec.Result()

		err := UnexpectedResponse(resp)
		if err == nil {
			t.Errorf("expected error for status %d, got nil", code)
			continue
		}

		var respErr *ResponseError
		if !errors.As(err, &respErr) {
			t.Errorf("expected *ResponseError for status %d, got %T", code, err)
			continue
		}

		if respErr.StatusCode != code {
			t.Errorf("expected status code %d, got %d", code, respErr.StatusCode)
		}
	}
}

func TestErrResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusInternalServerError)
	rec.Header().Set("X-Request-Id", "test-req-123")
	_, _ = rec.WriteString("internal server error details")
	resp := rec.Result()
	resp.Header.Set("X-Request-Id", "test-req-123")

	err := ErrResponse(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("expected *ResponseError, got %T", err)
	}

	if respErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", respErr.StatusCode)
	}

	if !strings.Contains(respErr.Body, "internal server error details") {
		t.Errorf("expected body to contain error details, got %q", respErr.Body)
	}

	if respErr.RequestID != "test-req-123" {
		t.Errorf("expected request ID %q, got %q", "test-req-123", respErr.RequestID)
	}
}
