// Command server runs an Nx self-hosted remote cache server:
// https://nx.dev/docs/kb/self-hosted-caching#build-your-own-caching-server
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"nx-remote-cache/internal/auth"
	"nx-remote-cache/internal/config"
	"nx-remote-cache/internal/server"
	"nx-remote-cache/internal/storage"
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

	backend, err := newBackend(ctx, cfg)
	if err != nil {
		return err
	}

	tokens := auth.NewTokenStore(cfg.ReadTokens, cfg.WriteTokens)
	srv := server.New(backend, tokens, log, cfg.MaxEntryBytes)

	httpServer := &http.Server{
		Addr:         cfg.Addr,
		Handler:      srv.Handler(),
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
