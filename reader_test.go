package ok

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestMultiReader_SingleReader(t *testing.T) {
	r := NewMultiReader(io.NopCloser(bytes.NewReader([]byte("hello"))))
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected %q, got %q", "hello", string(data))
	}
}

func TestMultiReader_MultipleReaders(t *testing.T) {
	r := NewMultiReader(
		io.NopCloser(bytes.NewReader([]byte("hello"))),
		io.NopCloser(bytes.NewReader([]byte(" "))),
		io.NopCloser(bytes.NewReader([]byte("world"))),
	)
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", string(data))
	}
}

func TestMultiReader_Empty(t *testing.T) {
	r := NewMultiReader()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %q", string(data))
	}
}

func TestMultiReader_Close(t *testing.T) {
	closed := make([]bool, 3)
	readers := make([]io.ReadCloser, 3)
	for i := range readers {
		idx := i
		readers[i] = &trackingCloser{
			Reader: bytes.NewReader([]byte("x")),
			onClose: func() {
				closed[idx] = true
			},
		}
	}

	mr := NewMultiReader(readers...)
	if err := mr.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, c := range closed {
		if !c {
			t.Errorf("reader %d was not closed", i)
		}
	}
}

func TestMultiReader_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := NewMultiReader(io.NopCloser(bytes.NewReader([]byte("hello"))))
	r.SetContext(ctx)

	_, err := r.Read(make([]byte, 5))
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

type trackingCloser struct {
	io.Reader
	onClose func()
}

func (tc *trackingCloser) Close() error {
	tc.onClose()
	return nil
}
