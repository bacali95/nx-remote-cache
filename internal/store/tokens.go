package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type TokenScope string

const (
	ScopeRead  TokenScope = "read"
	ScopeWrite TokenScope = "write"
)

type Token struct {
	ID         int64
	Name       string
	Scope      TokenScope
	CreatedBy  *int64
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

func (s *Store) CreateToken(ctx context.Context, name string, scope TokenScope, tokenHash string, createdBy int64) (Token, error) {
	var t Token
	err := s.pool.QueryRow(ctx,
		`INSERT INTO tokens (name, scope, token_hash, created_by) VALUES ($1, $2, $3, $4)
		 RETURNING id, name, scope, created_by, created_at, last_used_at, revoked_at`,
		name, scope, tokenHash, createdBy,
	).Scan(&t.ID, &t.Name, &t.Scope, &t.CreatedBy, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt)
	if isUniqueViolation(err) {
		return Token{}, ErrConflict
	}
	if err != nil {
		return Token{}, err
	}
	return t, nil
}

func (s *Store) ListTokens(ctx context.Context) ([]Token, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, scope, created_by, created_at, last_used_at, revoked_at
		 FROM tokens ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []Token
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.Name, &t.Scope, &t.CreatedBy, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (s *Store) RevokeToken(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Authenticate looks up an active (non-revoked) token by its hash and, on a
// hit, stamps last_used_at in the same round trip.
func (s *Store) Authenticate(ctx context.Context, tokenHash string) (Token, error) {
	var t Token
	err := s.pool.QueryRow(ctx,
		`UPDATE tokens SET last_used_at = now()
		 WHERE token_hash = $1 AND revoked_at IS NULL
		 RETURNING id, name, scope, created_by, created_at, last_used_at, revoked_at`,
		tokenHash,
	).Scan(&t.ID, &t.Name, &t.Scope, &t.CreatedBy, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Token{}, ErrNotFound
	}
	if err != nil {
		return Token{}, err
	}
	return t, nil
}
