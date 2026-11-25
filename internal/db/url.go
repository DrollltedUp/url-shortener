package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/drollllted/url-shortener/internal/store"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ store.Store = (*URLMemory)(nil)

type URLMemory struct {
	pool *pgxpool.Pool
}

func NewURLMemory(pool *pgxpool.Pool) *URLMemory {
	return &URLMemory{
		pool: pool,
	}
}

func (u *URLMemory) Save(ctx context.Context, code, target string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const q = `INSERT INTO url (url_original, url_short) VALUES ($1, $2)`
	if _, err := u.pool.Exec(ctx, q, target, code); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return store.ErrCodeTaken
		}
		return fmt.Errorf("insert url: %w", err)
	}

	return nil

}

func (u *URLMemory) Get(ctx context.Context, code string) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const q = `SELECT url_original FROM url WHERE url_short = $1`
	var target string
	err := u.pool.QueryRow(ctx, q, code).Scan(&target)
	// if errors.As(err, pgx.ErrNoRows) {
	// 	return "", store.ErrNotFound
	// }
	if err != nil {
		return "", fmt.Errorf("select url: %w", err)
	}

	return target, nil
}
