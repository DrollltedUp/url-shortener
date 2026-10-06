package service

import (
	"errors"
	"testing"

	"github.com/drollllted/url-shortener/internal/store"
)

type fakeGen struct {
	codes []string
	i     int
}

func (g *fakeGen) Generate() string {
	c := g.codes[g.i%len(g.codes)]
	g.i++
	return c
}

func TestShorten_InvalidURL(t *testing.T) {
	svc := NewShortener(store.NewStore(), RandomGeneration{})

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
	svc := NewShortener(st, RandomGeneration{})

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

func TestShorten_RetriesOnCollision(t *testing.T) {
	st := store.NewStore()

	occupiedCode := "aaaaaa"
	occupiedURL := "https://occupied.example"
	err := st.Save(occupiedCode, occupiedURL)
	if err != nil {
		t.Fatalf("failed to prepare test data: %v", err)
	}

	fg := &fakeGen{
		codes: []string{"aaaaaa", "bbbbbb"},
	}

	svc := NewShortener(st, fg)

	code, err := svc.Shorten("https://go.dev")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if code != "bbbbbb" {
		t.Errorf("expected code to be 'bbbbbb', got '%s'", code)
	}

	savedURL, ok := st.Get(occupiedCode)
	if !ok {
		t.Errorf("expected to find occupied code, got error: %v", occupiedCode)
	}
	if savedURL != occupiedURL {
		t.Errorf("expected occupied URL to remain '%s', got '%s'", occupiedURL, savedURL)
	}
}
