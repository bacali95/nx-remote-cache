// Package config loads server configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type StorageBackend string

const (
	StorageLocal StorageBackend = "local"
	StorageS3    StorageBackend = "s3"
	StorageGCS   StorageBackend = "gcs"
)

type Config struct {
	Addr string // e.g. ":3000"

	Storage StorageBackend

	// local backend
	LocalDir string

	// s3 backend
	S3Bucket       string
	S3Region       string
	S3Prefix       string
	S3Endpoint     string // optional: for R2/MinIO/other S3-compatible stores
	S3UsePathStyle bool

	// gcs backend
	GCSBucket string
	GCSPrefix string

	// admin database: users, sessions, and cache access tokens all live in
	// Postgres (see internal/store) rather than static env-var tokens.
	DatabaseURL string

	// admin sessions
	SessionTTL   time.Duration
	CookieSecure bool // set false only for local http:// development

	// bootstrap: if the users table is empty on startup and both of these
	// are set, a first admin account is created so there's a way to log in.
	AdminBootstrapEmail    string
	AdminBootstrapPassword string

	MaxEntryBytes int64

	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	ShutdownGrace time.Duration
}

func FromEnv() (*Config, error) {
	cfg := &Config{
		Addr:                   getEnv("PORT_ADDR", ":"+getEnv("PORT", "3000")),
		Storage:                StorageBackend(getEnv("STORAGE_BACKEND", "local")),
		LocalDir:               getEnv("CACHE_DIR", "/var/lib/nx-remote-cache"),
		S3Bucket:               os.Getenv("S3_BUCKET"),
		S3Region:               os.Getenv("S3_REGION"),
		S3Prefix:               os.Getenv("S3_PREFIX"),
		S3Endpoint:             os.Getenv("S3_ENDPOINT"),
		S3UsePathStyle:         getEnvBool("S3_USE_PATH_STYLE", false),
		GCSBucket:              os.Getenv("GCS_BUCKET"),
		GCSPrefix:              os.Getenv("GCS_PREFIX"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		SessionTTL:             getEnvDuration("SESSION_TTL", 24*time.Hour),
		CookieSecure:           getEnvBool("COOKIE_SECURE", true),
		AdminBootstrapEmail:    os.Getenv("ADMIN_BOOTSTRAP_EMAIL"),
		AdminBootstrapPassword: os.Getenv("ADMIN_BOOTSTRAP_PASSWORD"),
		MaxEntryBytes:          getEnvInt64("MAX_CACHE_ENTRY_BYTES", 500*1024*1024), // 500MB default
		ReadTimeout:            getEnvDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:           getEnvDuration("WRITE_TIMEOUT", 60*time.Second),
		ShutdownGrace:          getEnvDuration("SHUTDOWN_GRACE", 15*time.Second),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required (users, sessions, and cache tokens are stored in Postgres)")
	}

	switch cfg.Storage {
	case StorageLocal:
		// nothing else required
	case StorageS3:
		if cfg.S3Bucket == "" {
			return nil, fmt.Errorf("S3_BUCKET is required when STORAGE_BACKEND=s3")
		}
	case StorageGCS:
		if cfg.GCSBucket == "" {
			return nil, fmt.Errorf("GCS_BUCKET is required when STORAGE_BACKEND=gcs")
		}
	default:
		return nil, fmt.Errorf("unknown STORAGE_BACKEND %q (want %q, %q or %q)", cfg.Storage, StorageLocal, StorageS3, StorageGCS)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
