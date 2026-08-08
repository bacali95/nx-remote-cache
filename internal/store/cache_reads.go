package store

import (
	"context"
	"time"
)

type CacheReadStats struct {
	ReadCount  int64
	LastReadAt *time.Time
}

// RecordCacheRead increments the read counter and stamps last_read_at for
// hash, creating the row on first read.
func (s *Store) RecordCacheRead(ctx context.Context, hash string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cache_reads (hash, read_count, last_read_at)
		VALUES ($1, 1, now())
		ON CONFLICT (hash) DO UPDATE SET
			read_count = cache_reads.read_count + 1,
			last_read_at = now()
	`, hash)
	return err
}

// DeleteCacheReadStats removes hash's tracking row. Call this whenever the
// underlying cache entry is deleted (single delete, bulk delete, prune, or
// the janitor sweep) so tracking rows don't accumulate for hashes that no
// longer exist.
func (s *Store) DeleteCacheReadStats(ctx context.Context, hash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM cache_reads WHERE hash = $1`, hash)
	return err
}

// GetCacheReadStatsBatch returns stats for exactly the given hashes (missing
// hashes are simply absent from the result, meaning "never read"). Sized
// for enriching one page of the admin cache list.
func (s *Store) GetCacheReadStatsBatch(ctx context.Context, hashes []string) (map[string]CacheReadStats, error) {
	out := make(map[string]CacheReadStats, len(hashes))
	if len(hashes) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT hash, read_count, last_read_at FROM cache_reads WHERE hash = ANY($1)`,
		hashes,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		var stats CacheReadStats
		if err := rows.Scan(&hash, &stats.ReadCount, &stats.LastReadAt); err != nil {
			return nil, err
		}
		out[hash] = stats
	}
	return out, rows.Err()
}

// ListAllCacheReadStats returns every tracked hash's stats in one query, for
// the janitor's full-table sweep (avoids one query per stored entry).
func (s *Store) ListAllCacheReadStats(ctx context.Context) (map[string]CacheReadStats, error) {
	rows, err := s.pool.Query(ctx, `SELECT hash, read_count, last_read_at FROM cache_reads`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]CacheReadStats)
	for rows.Next() {
		var hash string
		var stats CacheReadStats
		if err := rows.Scan(&hash, &stats.ReadCount, &stats.LastReadAt); err != nil {
			return nil, err
		}
		out[hash] = stats
	}
	return out, rows.Err()
}
