package ok

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// LimitedResponse reads up to ResponseErrLimit bytes from the response body
// for error reporting purposes. It reassembles the body using MultiReader
// so the body can still be read again by downstream consumers.
func LimitedResponse(resp *http.Response) []byte {
	if resp == nil || resp.Body == nil {
		return nil
	}

	lr := io.LimitReader(resp.Body, ResponseErrLimit)
	body, err := io.ReadAll(lr)
	if err != nil {
		return nil
	}

	// Reassemble the body so it can still be read.
	resp.Body = NewMultiReader(
		io.NopCloser(bytes.NewReader(body)),
		resp.Body,
	)

	return body
}

// ResponseFuncJSON returns a response callback that checks for a successful
// status code (2xx) and then JSON-decodes the body into the provided data.
//
// Usage:
//
//	var result MyStruct
//	err := client.Do(req, ok.ResponseFuncJSON(&result))
func ResponseFuncJSON(data any) func(*http.Response) error {
	return func(resp *http.Response) error {
		if err := UnexpectedResponse(resp); err != nil {
			// Enrich the error with body and request ID.
			return ErrResponse(resp)
		}

		// No content.
		if resp.ContentLength == 0 {
			return nil
		}

		// If data is nil, caller just wants status check.
		if data == nil {
			return nil
		}

		return json.NewDecoder(resp.Body).Decode(data)
	}
}
