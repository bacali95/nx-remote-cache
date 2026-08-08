package store

import (
	"context"
	"time"
)

type StorageBackendType string

const (
	StorageLocal StorageBackendType = "local"
	StorageS3    StorageBackendType = "s3"
	StorageGCS   StorageBackendType = "gcs"
)

// AppSettings mirrors the app_settings row exactly: secret fields hold
// ciphertext (or nil if unset), never plaintext. Encryption/decryption is
// the caller's job (see internal/settings) — this package only persists
// bytes.
type AppSettings struct {
	StorageBackend StorageBackendType
	LocalDir       string

	S3Bucket             string
	S3Region             string
	S3Prefix             string
	S3Endpoint           string
	S3UsePathStyle       bool
	S3AccessKeyIDEnc     *string
	S3SecretAccessKeyEnc *string

	GCSBucket             string
	GCSPrefix             string
	GCSCredentialsJSONEnc *string

	SessionTTLSeconds  int
	MaxCacheEntryBytes int64

	UpdatedAt time.Time
	UpdatedBy *int64
}

// GetSettings returns the single app_settings row, seeded by migration.
func (s *Store) GetSettings(ctx context.Context) (AppSettings, error) {
	var st AppSettings
	err := s.pool.QueryRow(ctx, `
		SELECT storage_backend, local_dir,
		       s3_bucket, s3_region, s3_prefix, s3_endpoint, s3_use_path_style,
		       s3_access_key_id_enc, s3_secret_access_key_enc,
		       gcs_bucket, gcs_prefix, gcs_credentials_json_enc,
		       session_ttl_seconds, max_cache_entry_bytes,
		       updated_at, updated_by
		FROM app_settings
	`).Scan(
		&st.StorageBackend, &st.LocalDir,
		&st.S3Bucket, &st.S3Region, &st.S3Prefix, &st.S3Endpoint, &st.S3UsePathStyle,
		&st.S3AccessKeyIDEnc, &st.S3SecretAccessKeyEnc,
		&st.GCSBucket, &st.GCSPrefix, &st.GCSCredentialsJSONEnc,
		&st.SessionTTLSeconds, &st.MaxCacheEntryBytes,
		&st.UpdatedAt, &st.UpdatedBy,
	)
	if err != nil {
		return AppSettings{}, err
	}
	return st, nil
}

func (s *Store) UpdateSettings(ctx context.Context, st AppSettings, updatedBy int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE app_settings SET
			storage_backend = $1,
			local_dir = $2,
			s3_bucket = $3,
			s3_region = $4,
			s3_prefix = $5,
			s3_endpoint = $6,
			s3_use_path_style = $7,
			s3_access_key_id_enc = $8,
			s3_secret_access_key_enc = $9,
			gcs_bucket = $10,
			gcs_prefix = $11,
			gcs_credentials_json_enc = $12,
			session_ttl_seconds = $13,
			max_cache_entry_bytes = $14,
			updated_at = now(),
			updated_by = $15
	`,
		st.StorageBackend, st.LocalDir,
		st.S3Bucket, st.S3Region, st.S3Prefix, st.S3Endpoint, st.S3UsePathStyle,
		st.S3AccessKeyIDEnc, st.S3SecretAccessKeyEnc,
		st.GCSBucket, st.GCSPrefix, st.GCSCredentialsJSONEnc,
		st.SessionTTLSeconds, st.MaxCacheEntryBytes,
		updatedBy,
	)
	return err
}
