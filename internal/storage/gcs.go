package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

// GCS stores cache artifacts in a Google Cloud Storage bucket. Like the S3
// backend, this suits multi-replica servers or ephemeral CI runners since
// the store lives outside the process. Credentials are resolved via
// Application Default Credentials (a service account key file via
// GOOGLE_APPLICATION_CREDENTIALS, workload identity on GKE/GCE, or
// `gcloud auth application-default login` locally).
type GCS struct {
	bucket *storage.BucketHandle
	prefix string
}

type GCSOptions struct {
	Bucket string
	Prefix string
}

func NewGCS(ctx context.Context, opts GCSOptions) (*GCS, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create GCS client: %w", err)
	}
	return &GCS{bucket: client.Bucket(opts.Bucket), prefix: opts.Prefix}, nil
}

func (g *GCS) key(hash string) string {
	if g.prefix == "" {
		return hash
	}
	return path.Join(g.prefix, hash)
}

func (g *GCS) Exists(ctx context.Context, hash string) (bool, error) {
	_, err := g.bucket.Object(g.key(hash)).Attrs(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g *GCS) Put(ctx context.Context, hash string, r io.Reader, size int64) error {
	// DoesNotExist precondition makes the write fail atomically if another
	// request already stored this hash, mirroring the local backend's
	// hard-link race guard without a read-then-write round trip.
	obj := g.bucket.Object(g.key(hash)).If(storage.Conditions{DoesNotExist: true})
	w := obj.NewWriter(ctx)
	w.ContentType = "application/octet-stream"

	if _, err := io.Copy(w, io.LimitReader(r, size)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write object: %w", err)
	}
	if err := w.Close(); err != nil {
		if isPreconditionFailed(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("finalize object: %w", err)
	}
	return nil
}

func (g *GCS) Get(ctx context.Context, hash string) (io.ReadCloser, int64, error) {
	r, err := g.bucket.Object(g.key(hash)).NewReader(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("read object: %w", err)
	}
	return r, r.Attrs.Size, nil
}

func isPreconditionFailed(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == 412
	}
	return false
}
