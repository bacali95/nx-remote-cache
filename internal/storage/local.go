package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// Path returns the on-disk file path for hash. Exported for operational
// tooling (and tests) that need to touch the underlying file directly.
func (l *Local) Path(hash string) string {
	return filepath.Join(l.dir, hash)
}

func (l *Local) Exists(_ context.Context, hash string) (bool, error) {
	_, err := os.Stat(l.Path(hash))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (l *Local) Put(_ context.Context, hash string, r io.Reader, size int64) error {
	dest := l.Path(hash)
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
	defer func() { _ = os.Remove(tmpName) }() // no-op once renamed

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
	f, err := os.Open(l.Path(hash))
	if os.IsNotExist(err) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}

func (l *Local) Delete(_ context.Context, hash string) error {
	err := os.Remove(l.Path(hash))
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}

const defaultListLimit = 50

func (l *Local) List(_ context.Context, cursor string, limit int) (ListPage, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}

	dirEntries, err := os.ReadDir(l.dir)
	if err != nil {
		return ListPage{}, err
	}

	// os.ReadDir guarantees results sorted by filename, so a simple
	// greater-than scan gives us cursor-based pagination for free.
	var names []string
	for _, e := range dirEntries {
		if e.IsDir() || strings.Contains(e.Name(), ".tmp-") {
			continue
		}
		names = append(names, e.Name())
	}

	start := sort.Search(len(names), func(i int) bool { return names[i] > cursor })
	end := start + limit
	if end > len(names) {
		end = len(names)
	}

	page := ListPage{}
	for _, name := range names[start:end] {
		info, err := os.Stat(filepath.Join(l.dir, name))
		if err != nil {
			continue // removed between ReadDir and Stat; skip rather than fail the whole page
		}
		page.Entries = append(page.Entries, Entry{Hash: name, Size: info.Size(), ModTime: info.ModTime()})
	}
	if end < len(names) {
		page.NextCursor = names[end-1]
	}
	return page, nil
}
