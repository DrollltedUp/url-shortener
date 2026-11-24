package store

import (
	"context"
	"errors"
)

var (
	ErrCodeTaken = errors.New("code already taken")
	ErrNotFound  = errors.New("not found")
)

type Store interface {
	Save(ctx context.Context, code, target string) error
	Get(ctx context.Context, code string) (string, error)
}
