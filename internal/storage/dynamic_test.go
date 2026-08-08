package storage

import (
	"bytes"
	"context"
	"sync"
	"testing"
)

func TestDynamicSwap(t *testing.T) {
	ctx := context.Background()
	a, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal a: %v", err)
	}
	b, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal b: %v", err)
	}

	d := NewDynamic(a)
	if err := d.Put(ctx, "hash1", bytes.NewReader([]byte("in-a")), 4); err != nil {
		t.Fatalf("Put via a: %v", err)
	}
	if exists, _ := a.Exists(ctx, "hash1"); !exists {
		t.Fatalf("hash1 should exist directly in backend a")
	}

	d.Swap(b)
	if exists, _ := d.Exists(ctx, "hash1"); exists {
		t.Fatalf("after swap, Dynamic should see backend b (empty), not a's data")
	}
	if err := d.Put(ctx, "hash2", bytes.NewReader([]byte("in-b")), 4); err != nil {
		t.Fatalf("Put via b: %v", err)
	}
	if exists, _ := b.Exists(ctx, "hash2"); !exists {
		t.Fatalf("hash2 should have landed directly in backend b")
	}
	if exists, _ := a.Exists(ctx, "hash2"); exists {
		t.Fatalf("hash2 should not exist in backend a")
	}
}

func TestDynamicConcurrentSwapIsRaceFree(t *testing.T) {
	ctx := context.Background()
	backend, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	d := NewDynamic(backend)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_, _ = d.Exists(ctx, "whatever")
		}(i)
		go func() {
			defer wg.Done()
			other, err := NewLocal(t.TempDir())
			if err != nil {
				return
			}
			d.Swap(other)
		}()
	}
	wg.Wait()
}
