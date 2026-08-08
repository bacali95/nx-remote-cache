package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Local stores cache artifacts as files on disk. Suited for a single
// long-lived instance with a persistent volume; for multiple replicas or
// ephemeral hosts, use the S3 backend instead.
type Local struct {
	dir string
}

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &Local{dir: dir}, nil
}

func (l *Local) path(hash string) string {
	return filepath.Join(l.dir, hash)
}

func (l *Local) Exists(_ context.Context, hash string) (bool, error) {
	_, err := os.Stat(l.path(hash))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (l *Local) Put(_ context.Context, hash string, r io.Reader, size int64) error {
	dest := l.path(hash)
	if _, err := os.Stat(dest); err == nil {
		return ErrAlreadyExists
	} else if !os.IsNotExist(err) {
		return err
	}

	tmp, err := os.CreateTemp(l.dir, hash+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed

	n, err := io.Copy(tmp, io.LimitReader(r, size))
	closeErr := tmp.Close()
	if err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close temp file: %w", closeErr)
	}
	if n != size {
		return fmt.Errorf("short write: expected %d bytes, wrote %d", size, n)
	}

	// Atomic rename avoids a second writer racing the same hash into a
	// half-written file; the loser's temp file is removed by the defer.
	if err := os.Link(tmpName, dest); err != nil {
		if os.IsExist(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("finalize artifact: %w", err)
	}
	return nil
}

func (l *Local) Get(_ context.Context, hash string) (io.ReadCloser, int64, error) {
	f, err := os.Open(l.path(hash))
	if os.IsNotExist(err) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}
