package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestShortenConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/shorten",
				strings.NewReader(`{"url":"https://go.dev"}`))
			shorten(httptest.NewRecorder(), req)
		}()
	}
	wg.Wait()
}

func TestGetCodeUnknown(t *testing.T) {
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	req.SetPathValue("code", "nonexistent") // см. ниже, зачем
	rec := httptest.NewRecorder()

	getCode(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestShortenThenRedirect(t *testing.T) {
	// ШАГ 1: как «curl -X POST», только без запущенного сервера.
	// rec («рекордер») — это блокнот, в который хендлер пишет ответ вместо сети.
	req := httptest.NewRequest("POST", "/shorten", strings.NewReader(`{"url":"https://go.dev"}`))
	rec := httptest.NewRecorder()
	shorten(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("shorten: got %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body)
	}

	// ШАГ 2: «скопировать код из ответа».
	// В блокноте лежит текст {"short_url":"http://localhost:9091/AbC123"}.
	// Unmarshal превращает его в map, и resp["short_url"] даёт строку-ссылку.
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("ответ не является JSON: %v\nтело: %s", err, rec.Body)
	}
	// TrimPrefix отрезает "http://localhost:9091/", остаётся только "AbC123"
	code := strings.TrimPrefix(resp["short_url"], baseURL+"/")

	// ШАГ 3: как «curl -i localhost:9091/КОД», но код подставлен автоматически.
	req2 := httptest.NewRequest("GET", "/"+code, nil)
	req2.SetPathValue("code", code) // мы вызываем getCode напрямую, без маршрутизатора,
	rec2 := httptest.NewRecorder()  // поэтому «вырезать» code из пути некому, говорим вручную
	getCode(rec2, req2)
	if rec2.Code != http.StatusFound {
		t.Fatalf("getCode: got %d, want %d", rec2.Code, http.StatusFound)
	}
	if got := rec2.Header().Get("Location"); got != "https://go.dev" {
		t.Errorf("Location = %q, want %q", got, "https://go.dev")
	}
}
