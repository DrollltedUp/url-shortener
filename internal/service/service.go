package service

import (
	"errors"
	"math/rand/v2"
	"net/url"

	"github.com/drollllted/url-shortener/internal/store"
)

type Shortener struct {
	store *store.Store
}

var ErrInvalidURL = errors.New("Invalid URL")
var ErrCodeTaken = errors.New("code already taken")

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func NewShortener(s *store.Store) *Shortener {
	return &Shortener{
		store: s,
	}
}

func (s *Shortener) Shorten(rawURL string) (string, error) {
	//var store store.Store
	if !IsValidURL(rawURL) {
		return "", ErrInvalidURL
	}

	for i := 0; i < 5; i++ {
		code := generateCode()
		if err := s.store.Save(code, rawURL); err != nil {
			if errors.Is(err, ErrCodeTaken) {
				return "", err
			} else {
				return "", err
			}
		}

		return code, nil
	}

	return "", nil
}

func IsValidURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}

	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}

	return true
}

func generateCode() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = alphabet[rand.IntN(len(alphabet))]
	}

	return string(b)
}
