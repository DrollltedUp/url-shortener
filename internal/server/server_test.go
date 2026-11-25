// internal/server/server_test.go
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drollllted/url-shortener/internal/service"
	"github.com/drollllted/url-shortener/internal/store"
)

type fakeStore struct {
	saveFn func(code, target string) error
	getFn  func(code string) (string, error)
}

func (f *fakeStore) Save(_ context.Context, code, target string) error {
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
	codes []string
	i     int
}

func (g *stubGen) Generate() string { c := g.codes[g.i%len(g.codes)]; g.i++; return c }

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func newTestServer(t *testing.T, fs store.Store, p Pinger) (*Server, *slog.Logger) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewShortener(fs, &stubGen{codes: []string{"abc123"}})
	return NewServer(svc, log, "http://test", p), log
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodGet, "/health", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v", body)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id missing")
	}
}

func TestReadyz_OK(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodGet, "/readyz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestReadyz_DBDown(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{err: errors.New("conn refused")})
	rec := do(t, srv.Routes(), http.MethodGet, "/readyz", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body errorDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if !strings.Contains(body.Message, "database unavailable") {
		t.Fatalf("message = %q", body.Message)
	}
}

func TestReadyz_NilPinger(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, nil)
	rec := do(t, srv.Routes(), http.MethodGet, "/readyz", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestShorten_Created(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodPost, "/shorten", `{"url":"https://example.com"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body["short_url"] != "http://test/abc123" {
		t.Fatalf("short_url = %q", body["short_url"])
	}
}

func TestShorten_BadJSON(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodPost, "/shorten", `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestShorten_TooLarge(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{})
	big := `{"url":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	rec := do(t, srv.Routes(), http.MethodPost, "/shorten", big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
}

func TestShorten_InvalidURL(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodPost, "/shorten", `{"url":"ftp://x"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestShorten_StoreError(t *testing.T) {
	fs := &fakeStore{saveFn: func(_, _ string) error { return errors.New("boom") }}
	srv, _ := newTestServer(t, fs, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodPost, "/shorten", `{"url":"https://example.com"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestGetCode_Redirect(t *testing.T) {
	fs := &fakeStore{getFn: func(code string) (string, error) {
		if code == "abc123" {
			return "https://example.com", nil
		}
		return "", store.ErrNotFound
	}}
	srv, _ := newTestServer(t, fs, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodGet, "/abc123", "")

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://example.com" {
		t.Fatalf("Location = %q", loc)
	}
}

func TestGetCode_NotFound(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodGet, "/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetCode_StoreError(t *testing.T) {
	fs := &fakeStore{getFn: func(string) (string, error) { return "", errors.New("conn refused") }}
	srv, _ := newTestServer(t, fs, fakePinger{})
	rec := do(t, srv.Routes(), http.MethodGet, "/abc", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestRequestIDHeaderUnique(t *testing.T) {
	srv, _ := newTestServer(t, &fakeStore{}, fakePinger{})
	h := srv.Routes()
	r1 := do(t, h, http.MethodGet, "/health", "")
	r2 := do(t, h, http.MethodGet, "/health", "")
	id1, id2 := r1.Header().Get("X-Request-Id"), r2.Header().Get("X-Request-Id")
	if id1 == "" || id2 == "" || id1 == id2 {
		t.Fatalf("request ids: %q, %q", id1, id2)
	}
}

func TestRecovery_Panic(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(nil, log, "http://test", fakePinger{})
	rec := do(t, srv.Routes(), http.MethodGet, "/abc", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
