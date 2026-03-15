package ok

import (
	"bytes"
	"io"
	"testing"
)

func TestDrainBody_Nil(t *testing.T) {
	// Should not panic.
	DrainBody(nil)
}

func TestDrainBody_Normal(t *testing.T) {
	body := io.NopCloser(bytes.NewReader([]byte("hello world")))
	DrainBody(body)
}

func TestDrainBody_WithLimit(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'a'
	}

	closed := false
	body := &trackingReadCloser{
		Reader: bytes.NewReader(data),
		onClose: func() {
			closed = true
		},
	}

	DrainBody(body, WithDrainLimit(100))

	if !closed {
		t.Error("body was not closed")
	}
}

func TestDrainBody_UnlimitedDrain(t *testing.T) {
	data := make([]byte, 2*1024*1024) // 2MB
	for i := range data {
		data[i] = 'b'
	}

	closed := false
	body := &trackingReadCloser{
		Reader: bytes.NewReader(data),
		onClose: func() {
			closed = true
		},
	}

	DrainBody(body, WithDrainLimit(-1))

	if !closed {
		t.Error("body was not closed")
	}
}

type trackingReadCloser struct {
	io.Reader
	onClose func()
}

func (t *trackingReadCloser) Close() error {
	if t.onClose != nil {
		t.onClose()
	}
	return nil
}
