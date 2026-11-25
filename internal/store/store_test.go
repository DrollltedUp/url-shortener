package store

import (
	"context"
	"testing"
)

func TestDataStore_SaveAndGet(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	if err := s.Save(ctx, "abc", "https://example.com"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, ok := s.Get(ctx, "abc")
	if !ok {
		t.Fatal("Get: not found, want found")
	}
	if got != "https://example.com" {
		t.Fatalf("Get = %q, want %q", got, "https://example.com")
	}
}

func TestDataStore_SaveDuplicate(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	if err := s.Save(ctx, "abc", "https://a.com"); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	err := s.Save(ctx, "abc", "https://b.com")
	if err != ErrCodeTaken {
		t.Fatalf("second Save err = %v, want ErrCodeTaken", err)
	}
}

func TestDataStore_GetMissing(t *testing.T) {
	s := NewStore()
	_, ok := s.Get(context.Background(), "nope")
	if ok {
		t.Fatal("Get missing: ok=true, want false")
	}
}

func TestDataStore_Concurrent(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	const workers = 50
	done := make(chan struct{}, workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			_ = s.Save(ctx, string(rune('a'+i%26)), "https://example.com")
			_, _ = s.Get(ctx, "a")
		}(i)
	}
	for i := 0; i < workers; i++ {
		<-done
	}
}
