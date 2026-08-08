package store

import (
	"context"
	"testing"
)

func TestSettingsSeededAndUpdatable(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	got, err := s.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.StorageBackend != StorageLocal {
		t.Fatalf("seeded StorageBackend = %q, want %q", got.StorageBackend, StorageLocal)
	}
	if got.SessionTTLSeconds != 86400 {
		t.Fatalf("seeded SessionTTLSeconds = %d, want 86400", got.SessionTTLSeconds)
	}
	if got.MaxCacheEntryBytes != 524288000 {
		t.Fatalf("seeded MaxCacheEntryBytes = %d, want 524288000", got.MaxCacheEntryBytes)
	}
	if got.S3AccessKeyIDEnc != nil {
		t.Fatalf("seeded S3AccessKeyIDEnc should be nil, got %v", *got.S3AccessKeyIDEnc)
	}

	u, err := s.CreateUser(ctx, "admin@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	secret := "ciphertext-blob"
	got.StorageBackend = StorageS3
	got.S3Bucket = "my-bucket"
	got.S3Region = "us-east-1"
	got.S3SecretAccessKeyEnc = &secret
	got.SessionTTLSeconds = 3600
	got.MaxCacheEntryBytes = 1024

	if err := s.UpdateSettings(ctx, got, u.ID); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	reloaded, err := s.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings after update: %v", err)
	}
	if reloaded.StorageBackend != StorageS3 || reloaded.S3Bucket != "my-bucket" || reloaded.S3Region != "us-east-1" {
		t.Fatalf("reloaded settings = %+v, want s3/my-bucket/us-east-1", reloaded)
	}
	if reloaded.S3SecretAccessKeyEnc == nil || *reloaded.S3SecretAccessKeyEnc != secret {
		t.Fatalf("reloaded S3SecretAccessKeyEnc = %v, want %q", reloaded.S3SecretAccessKeyEnc, secret)
	}
	if reloaded.SessionTTLSeconds != 3600 || reloaded.MaxCacheEntryBytes != 1024 {
		t.Fatalf("reloaded ttl/maxbytes = %d/%d, want 3600/1024", reloaded.SessionTTLSeconds, reloaded.MaxCacheEntryBytes)
	}
	if reloaded.UpdatedBy == nil || *reloaded.UpdatedBy != u.ID {
		t.Fatalf("reloaded UpdatedBy = %v, want %d", reloaded.UpdatedBy, u.ID)
	}
}
