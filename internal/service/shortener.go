package service

import (
	"errors"
	"math/rand/v2"
	"net/url"

	"github.com/drollllted/url-shortener/internal/store"
)

type CodeGenerator interface {
	Generate() string
}

type RandomGeneration struct{}

type Shortener struct {
	store *store.Store
	gen   CodeGenerator
}

var (
	ErrInvalidURL     = errors.New("invalid URL")
	ErrCodeGeneration = errors.New("could not generate unique code")
	ErrNotFound       = errors.New("link not found")
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func NewShortener(s *store.Store, gen CodeGenerator) *Shortener {
	return &Shortener{
		store: s,
		gen:   gen,
	}
}

func (s *Shortener) Shorten(rawURL string) (string, error) {
	if !isValidURL(rawURL) {
		return "", ErrInvalidURL
	}

	for i := 0; i < 5; i++ {
		code := s.gen.Generate()
		err := s.store.Save(code, rawURL)
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

func (s *Shortener) Resolve(code string) (string, error) {
	target, ok := s.store.Get(code)
	if !ok {
		return "", ErrNotFound
	}

	return target, nil
}
