// Package config loads server configuration from environment variables.
//
// Storage backend, session TTL, and max cache entry size used to live here
// too, but are now runtime settings managed from the admin UI and stored
// in Postgres (see internal/settings) — what's left here is genuinely
// infra-level: where the database is, how the process listens, and one
// encryption key that must not itself live in the database it protects.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr string // e.g. ":3000"

	// users, sessions, cache access tokens, and runtime settings all live
	// in Postgres (see internal/store).
	DatabaseURL string

	// SettingsEncryptionKey (base64, 32 bytes) encrypts secret settings
	// fields (cloud credentials) at rest in Postgres. Generate with:
	// openssl rand -base64 32
	SettingsEncryptionKey string

	CookieSecure bool // set false only for local http:// development

	// bootstrap: if the users table is empty on startup and both of these
	// are set, a first admin account is created so there's a way to log in.
	AdminBootstrapEmail    string
	AdminBootstrapPassword string

	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	ShutdownGrace time.Duration
}

func FromEnv() (*Config, error) {
	cfg := &Config{
		Addr:                   getEnv("PORT_ADDR", ":"+getEnv("PORT", "3000")),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		SettingsEncryptionKey:  os.Getenv("SETTINGS_ENCRYPTION_KEY"),
		CookieSecure:           getEnvBool("COOKIE_SECURE", true),
		AdminBootstrapEmail:    os.Getenv("ADMIN_BOOTSTRAP_EMAIL"),
		AdminBootstrapPassword: os.Getenv("ADMIN_BOOTSTRAP_PASSWORD"),
		ReadTimeout:            getEnvDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:           getEnvDuration("WRITE_TIMEOUT", 60*time.Second),
		ShutdownGrace:          getEnvDuration("SHUTDOWN_GRACE", 15*time.Second),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required (users, sessions, tokens, and settings are stored in Postgres)")
	}
	if cfg.SettingsEncryptionKey == "" {
		return nil, fmt.Errorf("SETTINGS_ENCRYPTION_KEY is required (encrypts cloud credentials at rest; generate with: openssl rand -base64 32)")
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
