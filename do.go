package ok

import (
	"fmt"
	"net/http"
)

// Do executes the request using the client's HTTP client and calls fn
// with the response. The response body is always drained and closed
// after fn returns, ensuring connections are returned to the pool.
//
// If fn is nil, ErrResponseFuncNil is returned.
// If the request fails with no response, the error is wrapped with ErrRequest.
func (c *Client) Do(req *http.Request, fn func(*http.Response) error) error {
	return Do(c.HTTP, req, fn)
}

// Do is a package-level function that executes a request and calls fn
// with the response. The response body is always drained and closed
// after fn returns.
//
// This is useful when you have an *http.Client directly rather than a *Client.
func Do(c *http.Client, req *http.Request, fn func(*http.Response) error) error {
	if fn == nil {
		return ErrResponseFuncNil
	}

	resp, err := c.Do(req)
	if err != nil {
		if resp != nil {
			DrainBody(resp.Body)
		}
		return fmt.Errorf("%w: %w", ErrRequest, err)
	}

	// Always drain and close the body after fn returns.
	defer DrainBody(resp.Body)

	return fn(resp)
}
