package service

import (
	"errors"
	"testing"

	"github.com/drollllted/url-shortener/internal/store"
)

func TestShorten_InvalidURL(t *testing.T) {
	svc := NewShortener(store.NewStore())

	for _, raw := range []string{"https://", "http//", "abc", ""} {
		code, err := svc.Shorten(raw)
		if !errors.Is(err, ErrInvalidURL) {
			t.Errorf("Shorten(%q): err = %v, want ErrInvalidURL", raw, err)
		}
		if code != "" {
			t.Errorf("Shorten(%q): code = %q, want пустой", raw, code)
		}
	}
}

func TestShorten_OK(t *testing.T) {
	st := store.NewStore()
	svc := NewShortener(st)

	code, err := svc.Shorten("https://go.dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("len(code) = %d, want 6", len(code))
	}
	if got, ok := st.Get(code); !ok || got != "https://go.dev" {
		t.Errorf("в хранилище: %q, %v", got, ok)
	}
}
