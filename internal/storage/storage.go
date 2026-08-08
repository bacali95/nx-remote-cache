// Package storage defines the cache artifact backend contract and its
// implementations (local filesystem, S3-compatible object storage).
package storage

import (
	"context"
	"errors"
	"io"
	"regexp"
	"time"
)

// ErrNotFound is returned by Get/Stat when the hash has no stored artifact.
var ErrNotFound = errors.New("storage: object not found")

// ErrAlreadyExists is returned by Put when an artifact for the hash was
// already uploaded. The Nx cache contract is content-addressed and
// immutable: a hash is written once and never overwritten.
var ErrAlreadyExists = errors.New("storage: object already exists")

// Backend persists and retrieves cache artifacts keyed by Nx task hash.
type Backend interface {
	// Exists reports whether an artifact for hash has been stored.
	Exists(ctx context.Context, hash string) (bool, error)

	// Put stores size bytes read from r under hash. It returns
	// ErrAlreadyExists if the hash is already present.
	Put(ctx context.Context, hash string, r io.Reader, size int64) error

	// Get returns a reader for the artifact stored under hash along with
	// its size. Callers must close the reader. Returns ErrNotFound if
	// absent.
	Get(ctx context.Context, hash string) (io.ReadCloser, int64, error)

	// Delete removes the artifact stored under hash. Returns ErrNotFound
	// if absent.
	Delete(ctx context.Context, hash string) error

	// List returns up to limit entries starting after cursor (empty cursor
	// starts from the beginning). NextCursor in the returned page is empty
	// once there are no more entries.
	List(ctx context.Context, cursor string, limit int) (ListPage, error)
}

// Entry describes one stored cache artifact, as surfaced by List.
type Entry struct {
	Hash    string
	Size    int64
	ModTime time.Time
}

type ListPage struct {
	Entries    []Entry
	NextCursor string
}

// validHash matches the hash format Nx sends in the URL path: an
// alphanumeric task identifier. Rejecting anything else prevents path
// traversal and key-injection into object storage.
var validHash = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

// ValidHash reports whether hash is safe to use as a storage key.
func ValidHash(hash string) bool {
	return validHash.MatchString(hash)
}
