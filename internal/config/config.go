// Package config loads server configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

	// auth
	ReadTokens  []string
	WriteTokens []string

	MaxEntryBytes int64

	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	ShutdownGrace time.Duration
}

func FromEnv() (*Config, error) {
	cfg := &Config{
		Addr:           getEnv("PORT_ADDR", ":"+getEnv("PORT", "3000")),
		Storage:        StorageBackend(getEnv("STORAGE_BACKEND", "local")),
		LocalDir:       getEnv("CACHE_DIR", "/var/lib/nx-remote-cache"),
		S3Bucket:       os.Getenv("S3_BUCKET"),
		S3Region:       os.Getenv("S3_REGION"),
		S3Prefix:       os.Getenv("S3_PREFIX"),
		S3Endpoint:     os.Getenv("S3_ENDPOINT"),
		S3UsePathStyle: getEnvBool("S3_USE_PATH_STYLE", false),
		GCSBucket:      os.Getenv("GCS_BUCKET"),
		GCSPrefix:      os.Getenv("GCS_PREFIX"),
		ReadTokens:     splitCSV(os.Getenv("CACHE_READ_TOKENS")),
		WriteTokens:    splitCSV(os.Getenv("CACHE_WRITE_TOKENS")),
		MaxEntryBytes:  getEnvInt64("MAX_CACHE_ENTRY_BYTES", 500*1024*1024), // 500MB default
		ReadTimeout:    getEnvDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:   getEnvDuration("WRITE_TIMEOUT", 60*time.Second),
		ShutdownGrace:  getEnvDuration("SHUTDOWN_GRACE", 15*time.Second),
	}

	if len(cfg.WriteTokens) == 0 {
		return nil, fmt.Errorf("CACHE_WRITE_TOKENS must contain at least one token")
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

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
