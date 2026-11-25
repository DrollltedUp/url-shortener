package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/drollllted/url-shortener/internal/store"
)

type fakeStore struct {
	mu     sync.Mutex
	saves  []saveCall
	getFn  func(code string) (string, error)
	saveFn func(code, target string) error
}

type saveCall struct {
	code, target string
}

func (f *fakeStore) Save(_ context.Context, code, target string) error {
	f.mu.Lock()
	f.saves = append(f.saves, saveCall{code, target})
	f.mu.Unlock()
	if f.saveFn != nil {
		return f.saveFn(code, target)
	}
	return nil
}

func (f *fakeStore) Get(_ context.Context, code string) (string, error) {
	if f.getFn != nil {
		return f.getFn(code)
	}
	return "", store.ErrNotFound
}

type stubGen struct {
	mu    sync.Mutex
	codes []string
	i     int
}

func (g *stubGen) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	c := g.codes[g.i%len(g.codes)]
	g.i++
	return c
}

func TestShorten_InvalidURL(t *testing.T) {
	cases := []string{
		"",
		"not a url",
		"ftp://example.com",
		"javascript:alert(1)",
		"http://", // нет host
		"/relative",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			svc := NewShortener(&fakeStore{}, &stubGen{codes: []string{"abc123"}})
			_, err := svc.Shorten(context.Background(), in)
			if !errors.Is(err, ErrInvalidURL) {
				t.Fatalf("err = %v, want ErrInvalidURL", err)
			}
		})
	}
}

func TestShorten_OK(t *testing.T) {
	fs := &fakeStore{}
	svc := NewShortener(fs, &stubGen{codes: []string{"abc123"}})

	code, err := svc.Shorten(context.Background(), "https://example.com/path?q=1")
	if err != nil {
		t.Fatalf("Shorten: %v", err)
	}
	if code != "abc123" {
		t.Fatalf("code = %q, want abc123", code)
	}
	if len(fs.saves) != 1 || fs.saves[0].target != "https://example.com/path?q=1" {
		t.Fatalf("saves = %+v", fs.saves)
	}
}

func TestShorten_RetriesOnCodeTaken(t *testing.T) {
	attempts := 0
	fs := &fakeStore{
		saveFn: func(code, _ string) error {
			attempts++
			if attempts < 3 {
				return store.ErrCodeTaken
			}
			return nil
		},
	}
	svc := NewShortener(fs, &stubGen{codes: []string{"a", "b", "c", "d", "e"}})

	code, err := svc.Shorten(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("Shorten: %v", err)
	}
	if code != "c" {
		t.Fatalf("code = %q, want c (3rd attempt)", code)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestShorten_ExhaustsRetries(t *testing.T) {
	fs := &fakeStore{saveFn: func(_, _ string) error { return store.ErrCodeTaken }}
	svc := NewShortener(fs, &stubGen{codes: []string{"a"}})

	_, err := svc.Shorten(context.Background(), "https://example.com")
	if !errors.Is(err, ErrCodeGeneration) {
		t.Fatalf("err = %v, want ErrCodeGeneration", err)
	}
	if len(fs.saves) != 5 {
		t.Fatalf("saves = %d, want 5", len(fs.saves))
	}
}

func TestShorten_StoreErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	fs := &fakeStore{saveFn: func(_, _ string) error { return boom }}
	svc := NewShortener(fs, &stubGen{codes: []string{"a"}})

	_, err := svc.Shorten(context.Background(), "https://example.com")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestResolve_OK(t *testing.T) {
	fs := &fakeStore{getFn: func(code string) (string, error) {
		if code == "abc123" {
			return "https://example.com", nil
		}
		return "", store.ErrNotFound
	}}
	svc := NewShortener(fs, &stubGen{codes: []string{"x"}})

	got, err := svc.Resolve(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "https://example.com" {
		t.Fatalf("got = %q", got)
	}
}

func TestResolve_NotFound(t *testing.T) {
	fs := &fakeStore{getFn: func(string) (string, error) { return "", store.ErrNotFound }}
	svc := NewShortener(fs, &stubGen{codes: []string{"x"}})

	_, err := svc.Resolve(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestResolve_StoreErrorIsNotNotFound(t *testing.T) {
	boom := errors.New("connection refused")
	fs := &fakeStore{getFn: func(string) (string, error) { return "", boom }}
	svc := NewShortener(fs, &stubGen{codes: []string{"x"}})

	_, err := svc.Resolve(context.Background(), "abc")
	if errors.Is(err, ErrNotFound) {
		t.Fatal("store error must NOT become ErrNotFound")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestIsValidURL(t *testing.T) {
	ok := []string{"http://a", "https://example.com", "https://example.com/path?q=1#frag"}
	for _, s := range ok {
		if !isValidURL(s) {
			t.Errorf("isValidURL(%q) = false, want true", s)
		}
	}
	bad := []string{"", "ftp://a", "https://", "mailto:a@b.c", "example.com"}
	for _, s := range bad {
		if isValidURL(s) {
			t.Errorf("isValidURL(%q) = true, want false", s)
		}
	}
}

func TestRandomGeneration_Format(t *testing.T) {
	g := RandomGeneration{}
	for i := 0; i < 1000; i++ {
		c := g.Generate()
		if len(c) != 6 {
			t.Fatalf("len = %d, want 6", len(c))
		}
		for _, r := range c {
			if !strings.ContainsRune(alphabet, r) {
				t.Fatalf("bad rune %q in %q", r, c)
			}
		}
	}
}
