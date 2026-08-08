package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalPutGetExists(t *testing.T) {
	ctx := context.Background()
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	exists, err := l.Exists(ctx, "hash1")
	if err != nil || exists {
		t.Fatalf("Exists before put = (%v, %v), want (false, nil)", exists, err)
	}

	payload := []byte("hello cache")
	if err := l.Put(ctx, "hash1", bytes.NewReader(payload), int64(len(payload))); err != nil {
		t.Fatalf("Put: %v", err)
	}

	exists, err = l.Exists(ctx, "hash1")
	if err != nil || !exists {
		t.Fatalf("Exists after put = (%v, %v), want (true, nil)", exists, err)
	}

	r, size, err := l.Get(ctx, "hash1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer r.Close()
	if size != int64(len(payload)) {
		t.Fatalf("size = %d, want %d", size, len(payload))
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %q, want %q", got, payload)
	}
}

func TestLocalPutDuplicateFails(t *testing.T) {
	ctx := context.Background()
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	if err := l.Put(ctx, "hash1", bytes.NewReader([]byte("a")), 1); err != nil {
		t.Fatalf("first Put: %v", err)
	}
	err = l.Put(ctx, "hash1", bytes.NewReader([]byte("b")), 1)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("second Put error = %v, want ErrAlreadyExists", err)
	}
}

func TestLocalGetMissing(t *testing.T) {
	ctx := context.Background()
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	_, _, err = l.Get(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get error = %v, want ErrNotFound", err)
	}
}

func TestValidHash(t *testing.T) {
	valid := []string{"abc123", "ABC_def-123", "a"}
	invalid := []string{"", "abc/def", "abc..def", "abc$def", "abc def"}
	for _, h := range valid {
		if !ValidHash(h) {
			t.Errorf("ValidHash(%q) = false, want true", h)
		}
	}
	for _, h := range invalid {
		if ValidHash(h) {
			t.Errorf("ValidHash(%q) = true, want false", h)
		}
	}
}
