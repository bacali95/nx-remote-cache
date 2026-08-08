package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Session struct {
	IDHash    string
	UserID    int64
	ExpiresAt time.Time
}

func (s *Store) CreateSession(ctx context.Context, idHash string, userID int64, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (id_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		idHash, userID, expiresAt,
	)
	return err
}

// GetSession returns the session only if it exists and has not expired.
func (s *Store) GetSession(ctx context.Context, idHash string) (Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx,
		`SELECT id_hash, user_id, expires_at FROM sessions WHERE id_hash = $1 AND expires_at > now()`,
		idHash,
	).Scan(&sess.IDHash, &sess.UserID, &sess.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	return sess, nil
}

func (s *Store) DeleteSession(ctx context.Context, idHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id_hash = $1`, idHash)
	return err
}

// DeleteExpiredSessions prunes stale rows. Called opportunistically on
// login rather than on a schedule, since this is a low-traffic admin table.
func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	return err
}
