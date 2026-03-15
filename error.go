package ok

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for common failure conditions.
var (
	ErrRequest         = errors.New("request error")
	ErrResponseFuncNil = errors.New("response function is nil")
)

// ResponseError represents an HTTP response error with status code,
// body excerpt, and optional request ID.
type ResponseError struct {
	StatusCode int
	Body       string
	RequestID  string
}

func (e *ResponseError) Error() string {
	msg := fmt.Sprintf("unexpected status code: %d", e.StatusCode)
	if e.RequestID != "" {
		msg += fmt.Sprintf(", request_id: %s", e.RequestID)
	}
	if e.Body != "" {
		msg += fmt.Sprintf(", body: %s", e.Body)
	}
	return msg
}

// UnexpectedResponse returns a *ResponseError if the status code is not 2xx.
// Returns nil if the response indicates success.
func UnexpectedResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return &ResponseError{
		StatusCode: resp.StatusCode,
	}
}

// ErrResponse reads a limited portion of the response body and returns
// a *ResponseError with status code, body text, and X-Request-Id header.
func ErrResponse(resp *http.Response) error {
	body := LimitedResponse(resp)
	return &ResponseError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		RequestID:  resp.Header.Get("X-Request-Id"),
	}
}
