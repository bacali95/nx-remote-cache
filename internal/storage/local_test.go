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
	defer func() { _ = r.Close() }()
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

func TestLocalDelete(t *testing.T) {
	ctx := context.Background()
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	if err := l.Put(ctx, "hash1", bytes.NewReader([]byte("a")), 1); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := l.Delete(ctx, "hash1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if exists, _ := l.Exists(ctx, "hash1"); exists {
		t.Fatalf("entry still exists after Delete")
	}
	if err := l.Delete(ctx, "hash1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete missing: err = %v, want ErrNotFound", err)
	}
}

func TestLocalListPagination(t *testing.T) {
	ctx := context.Background()
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	hashes := []string{"a1", "a2", "a3", "a4", "a5"}
	for _, h := range hashes {
		if err := l.Put(ctx, h, bytes.NewReader([]byte("x")), 1); err != nil {
			t.Fatalf("Put(%s): %v", h, err)
		}
	}

	page1, err := l.List(ctx, "", 2)
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if len(page1.Entries) != 2 || page1.Entries[0].Hash != "a1" || page1.Entries[1].Hash != "a2" {
		t.Fatalf("page1 = %+v, want [a1 a2]", page1.Entries)
	}
	if page1.NextCursor != "a2" {
		t.Fatalf("page1.NextCursor = %q, want a2", page1.NextCursor)
	}

	page2, err := l.List(ctx, page1.NextCursor, 2)
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2.Entries) != 2 || page2.Entries[0].Hash != "a3" || page2.Entries[1].Hash != "a4" {
		t.Fatalf("page2 = %+v, want [a3 a4]", page2.Entries)
	}

	page3, err := l.List(ctx, page2.NextCursor, 2)
	if err != nil {
		t.Fatalf("List page3: %v", err)
	}
	if len(page3.Entries) != 1 || page3.Entries[0].Hash != "a5" {
		t.Fatalf("page3 = %+v, want [a5]", page3.Entries)
	}
	if page3.NextCursor != "" {
		t.Fatalf("page3.NextCursor = %q, want empty (last page)", page3.NextCursor)
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
