// Package storage defines the cache artifact backend contract and its
// implementations (local filesystem, S3-compatible object storage).
package storage

import (
	"context"
	"errors"
	"io"
	"regexp"
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
}

// validHash matches the hash format Nx sends in the URL path: an
// alphanumeric task identifier. Rejecting anything else prevents path
// traversal and key-injection into object storage.
var validHash = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

// ValidHash reports whether hash is safe to use as a storage key.
func ValidHash(hash string) bool {
	return validHash.MatchString(hash)
}
