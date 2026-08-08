package store

import (
	"context"
	"testing"
)

func TestRecordCacheReadIncrementsAndStampsLastRead(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.RecordCacheRead(ctx, "hash1"); err != nil {
		t.Fatalf("first RecordCacheRead: %v", err)
	}
	stats := mustBatch(t, s, "hash1")
	if stats.ReadCount != 1 || stats.LastReadAt == nil {
		t.Fatalf("after first read: %+v", stats)
	}
	firstRead := *stats.LastReadAt

	if err := s.RecordCacheRead(ctx, "hash1"); err != nil {
		t.Fatalf("second RecordCacheRead: %v", err)
	}
	stats = mustBatch(t, s, "hash1")
	if stats.ReadCount != 2 {
		t.Fatalf("after second read: ReadCount = %d, want 2", stats.ReadCount)
	}
	if stats.LastReadAt.Before(firstRead) {
		t.Fatalf("last_read_at went backwards")
	}
}

func TestGetCacheReadStatsBatchOmitsUnknownHashes(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.RecordCacheRead(ctx, "known"); err != nil {
		t.Fatalf("RecordCacheRead: %v", err)
	}

	batch, err := s.GetCacheReadStatsBatch(ctx, []string{"known", "never-read"})
	if err != nil {
		t.Fatalf("GetCacheReadStatsBatch: %v", err)
	}
	if _, ok := batch["known"]; !ok {
		t.Fatalf("expected stats for 'known'")
	}
	if _, ok := batch["never-read"]; ok {
		t.Fatalf("expected no entry for a hash that was never read")
	}
}

func TestDeleteCacheReadStats(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.RecordCacheRead(ctx, "hash1"); err != nil {
		t.Fatalf("RecordCacheRead: %v", err)
	}
	if err := s.DeleteCacheReadStats(ctx, "hash1"); err != nil {
		t.Fatalf("DeleteCacheReadStats: %v", err)
	}
	batch, err := s.GetCacheReadStatsBatch(ctx, []string{"hash1"})
	if err != nil {
		t.Fatalf("GetCacheReadStatsBatch: %v", err)
	}
	if _, ok := batch["hash1"]; ok {
		t.Fatalf("expected stats to be gone after delete")
	}
}

func TestListAllCacheReadStats(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	for _, h := range []string{"a", "b", "c"} {
		if err := s.RecordCacheRead(ctx, h); err != nil {
			t.Fatalf("RecordCacheRead(%s): %v", h, err)
		}
	}

	all, err := s.ListAllCacheReadStats(ctx)
	if err != nil {
		t.Fatalf("ListAllCacheReadStats: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("ListAllCacheReadStats returned %d entries, want 3", len(all))
	}
}

func mustBatch(t *testing.T, s *Store, hash string) CacheReadStats {
	t.Helper()
	batch, err := s.GetCacheReadStatsBatch(context.Background(), []string{hash})
	if err != nil {
		t.Fatalf("GetCacheReadStatsBatch: %v", err)
	}
	stats, ok := batch[hash]
	if !ok {
		t.Fatalf("no stats found for %q", hash)
	}
	return stats
}
