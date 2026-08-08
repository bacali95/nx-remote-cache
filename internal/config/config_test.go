package config

import (
	"testing"
	"time"
)

func clearRequiredEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"PORT_ADDR", "PORT", "DATABASE_URL", "SETTINGS_ENCRYPTION_KEY",
		"COOKIE_SECURE", "ADMIN_BOOTSTRAP_EMAIL", "ADMIN_BOOTSTRAP_PASSWORD",
		"READ_TIMEOUT", "WRITE_TIMEOUT", "SHUTDOWN_GRACE",
	} {
		t.Setenv(k, "")
	}
}

func TestFromEnvRequiresDatabaseURL(t *testing.T) {
	clearRequiredEnv(t)
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "key")

	_, err := FromEnv()
	if err == nil {
		t.Fatal("expected an error when DATABASE_URL is unset")
	}
}

func TestFromEnvRequiresSettingsEncryptionKey(t *testing.T) {
	clearRequiredEnv(t)
	t.Setenv("DATABASE_URL", "postgres://x")

	_, err := FromEnv()
	if err == nil {
		t.Fatal("expected an error when SETTINGS_ENCRYPTION_KEY is unset")
	}
}

func TestFromEnvDefaults(t *testing.T) {
	clearRequiredEnv(t)
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "key")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.Addr != ":3000" {
		t.Errorf("Addr = %q, want :3000", cfg.Addr)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure default should be true")
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want 30s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 60*time.Second {
		t.Errorf("WriteTimeout = %v, want 60s", cfg.WriteTimeout)
	}
	if cfg.ShutdownGrace != 15*time.Second {
		t.Errorf("ShutdownGrace = %v, want 15s", cfg.ShutdownGrace)
	}
	if cfg.AdminBootstrapEmail != "" || cfg.AdminBootstrapPassword != "" {
		t.Error("bootstrap admin fields should be empty by default")
	}
}

func TestFromEnvOverrides(t *testing.T) {
	clearRequiredEnv(t)
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "key")
	t.Setenv("PORT", "4000")
	t.Setenv("PORT_ADDR", "127.0.0.1:4000")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("ADMIN_BOOTSTRAP_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "hunter2")
	t.Setenv("READ_TIMEOUT", "5s")
	t.Setenv("WRITE_TIMEOUT", "10s")
	t.Setenv("SHUTDOWN_GRACE", "1s")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.Addr != "127.0.0.1:4000" {
		t.Errorf("Addr = %q, want PORT_ADDR to take precedence over PORT", cfg.Addr)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure should be false")
	}
	if cfg.AdminBootstrapEmail != "admin@example.com" || cfg.AdminBootstrapPassword != "hunter2" {
		t.Error("bootstrap admin fields should be read from env")
	}
	if cfg.ReadTimeout != 5*time.Second || cfg.WriteTimeout != 10*time.Second || cfg.ShutdownGrace != 1*time.Second {
		t.Error("timeouts should be read from env")
	}
}

func TestFromEnvPortFallsBackWithoutPortAddr(t *testing.T) {
	clearRequiredEnv(t)
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "key")
	t.Setenv("PORT", "8080")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}
}

func TestGetEnvBoolFallsBackOnInvalidValue(t *testing.T) {
	clearRequiredEnv(t)
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "key")
	t.Setenv("COOKIE_SECURE", "not-a-bool")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if !cfg.CookieSecure {
		t.Error("invalid COOKIE_SECURE should fall back to the default (true)")
	}
}

func TestGetEnvDurationFallsBackOnInvalidValue(t *testing.T) {
	clearRequiredEnv(t)
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "key")
	t.Setenv("READ_TIMEOUT", "not-a-duration")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("invalid READ_TIMEOUT should fall back to 30s, got %v", cfg.ReadTimeout)
	}
}
