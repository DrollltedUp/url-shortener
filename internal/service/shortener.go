package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/url"
	"regexp"
	"strings"

	"github.com/drollllted/url-shortener/internal/store"
)

type CodeGenerator interface {
	Generate() string
}

type RandomGeneration struct{}

type Shortener struct {
	store store.Store
	gen   CodeGenerator
}

var (
	ErrInvalidURL     = errors.New("invalid URL")
	ErrCodeGeneration = errors.New("could not generate unique code")
	ErrNotFound       = errors.New("link not found")

	ErrInvalidAlias  = errors.New("invalid alias")
	ErrReservedAlias = errors.New("alias is reserved")
	ErrAliasTaken    = errors.New("alias is taken")
)

var aliasRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

var reserved = map[string]struct{}{
	"health": {}, "healthz": {}, "readyz": {}, "shorten": {},
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func NewShortener(s store.Store, gen CodeGenerator) *Shortener {
	return &Shortener{
		store: s,
		gen:   gen,
	}
}

func (s *Shortener) Shorten(ctx context.Context, rawURL, alias string) (string, error) {
	if !isValidURL(rawURL) {
		return "", ErrInvalidURL
	}

	if alias != "" {
		return s.saveAlias(ctx, rawURL, alias)
	}

	for i := 0; i < 5; i++ {
		code := s.gen.Generate()
		err := s.store.Save(ctx, code, rawURL)
		if err == nil {
			return code, nil
		}

		if !errors.Is(err, store.ErrCodeTaken) {
			return "", err
		}
	}

	return "", ErrCodeGeneration
}

func isValidURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}

	return true
}

func (RandomGeneration) Generate() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = alphabet[rand.IntN(len(alphabet))]
	}

	return string(b)
}

// Resolve(get code for search)

func (s *Shortener) Resolve(ctx context.Context, code string) (string, error) {
	target, err := s.store.Get(ctx, code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}

	return target, nil
}

// For Alias

func (s *Shortener) saveAlias(ctx context.Context, rawURL, alias string) (string, error) {
	if !aliasRe.MatchString(alias) {
		return "", ErrInvalidAlias
	}

	if _, ok := reserved[strings.ToLower(alias)]; ok {
		return "", ErrReservedAlias
	}

	err := s.store.Save(ctx, alias, rawURL)
	if errors.Is(err, store.ErrCodeTaken) {
		return "", ErrAliasTaken
	}
	if err != nil {
		return "", err
	}

	return alias, nil
}
