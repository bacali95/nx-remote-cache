-- Tracks read activity per cache hash (the cache entries themselves live in
-- the storage backend, not here). Used to show "reads" / "last read" in the
-- admin UI and to decide what the background janitor sweep can safely
-- prune as unused.
CREATE TABLE cache_reads (
    hash         TEXT PRIMARY KEY,
    read_count   BIGINT NOT NULL DEFAULT 0,
    last_read_at TIMESTAMPTZ
);
