// Command server runs an Nx self-hosted remote cache server:
// https://nx.dev/docs/kb/self-hosted-caching#build-your-own-caching-server
// alongside an embedded admin UI for browsing/pruning the cache and
// managing users and access tokens.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nx-remote-cache/internal/adminapi"
	"nx-remote-cache/internal/auth"
	"nx-remote-cache/internal/config"
	"nx-remote-cache/internal/db"
	"nx-remote-cache/internal/server"
	"nx-remote-cache/internal/session"
	"nx-remote-cache/internal/storage"
	"nx-remote-cache/internal/store"
	webui "nx-remote-cache/web"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := waitForDB(ctx, cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	st := store.New(pool)

	if err := bootstrapAdmin(ctx, st, cfg, log); err != nil {
		return err
	}

	backend, err := newBackend(ctx, cfg)
	if err != nil {
		return err
	}

	uiFS, err := webui.DistFS()
	if err != nil {
		return fmt.Errorf("load embedded admin UI: %w", err)
	}

	dataSrv := server.New(backend, auth.NewCacheTokenAuth(st), log, cfg.MaxEntryBytes)
	adminSrv := adminapi.New(st, session.NewManager(st, cfg.SessionTTL), backend, log, cfg.CookieSecure, cfg.SessionTTL, uiFS)

	mux := http.NewServeMux()
	mux.Handle("/v1/", dataSrv.Handler())
	mux.Handle("/health", dataSrv.Handler())
	mux.Handle("/admin/", adminSrv.Handler())

	httpServer := &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr, "storage", cfg.Storage)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownGrace)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

// waitForDB retries migrations until they succeed or ctx is cancelled.
// Postgres containers can report healthy (accepting connections) while
// still mid-restart from their own init sequence, during which auth briefly
// fails — a real race in orchestrators generally (compose, k8s, ECS), not
// just a local quirk, so this retry belongs here rather than in a
// docker-compose healthcheck tweak.
func waitForDB(ctx context.Context, url string) error {
	const (
		maxAttempts = 15
		backoff     = 1 * time.Second
	)
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = db.Migrate(url)
		if lastErr == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
	return fmt.Errorf("database not ready after %d attempts: %w", maxAttempts, lastErr)
}

// bootstrapAdmin creates the first admin user from env vars if the users
// table is empty. This is the only way to get a first login on a fresh
// deployment, since there is no public self-registration route.
func bootstrapAdmin(ctx context.Context, st *store.Store, cfg *config.Config, log *slog.Logger) error {
	if cfg.AdminBootstrapEmail == "" || cfg.AdminBootstrapPassword == "" {
		return nil
	}
	n, err := st.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if n > 0 {
		return nil
	}

	hash, err := session.HashPassword(cfg.AdminBootstrapPassword)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	if _, err := st.CreateUser(ctx, cfg.AdminBootstrapEmail, hash); err != nil {
		return fmt.Errorf("create bootstrap admin: %w", err)
	}
	log.Info("created bootstrap admin user", "email", cfg.AdminBootstrapEmail)
	return nil
}

func newBackend(ctx context.Context, cfg *config.Config) (storage.Backend, error) {
	switch cfg.Storage {
	case config.StorageS3:
		return storage.NewS3(ctx, storage.S3Options{
			Bucket:       cfg.S3Bucket,
			Region:       cfg.S3Region,
			Prefix:       cfg.S3Prefix,
			Endpoint:     cfg.S3Endpoint,
			UsePathStyle: cfg.S3UsePathStyle,
		})
	case config.StorageGCS:
		return storage.NewGCS(ctx, storage.GCSOptions{
			Bucket: cfg.GCSBucket,
			Prefix: cfg.GCSPrefix,
		})
	default:
		return storage.NewLocal(cfg.LocalDir)
	}
}
