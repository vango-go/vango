package session

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStore_CopyOnSaveAndLoad(t *testing.T) {
	store := NewMemoryStore(WithCleanupInterval(24 * time.Hour))
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	expiresAt := time.Now().Add(time.Minute)

	original := []byte("abc")
	if err := store.Save(ctx, "s1", original, expiresAt); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	original[0] = 'z'
	loaded, err := store.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if string(loaded) != "abc" {
		t.Fatalf("Load() returned mutated data: got %q", string(loaded))
	}

	loaded[1] = 'y'
	loaded2, err := store.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if string(loaded2) != "abc" {
		t.Fatalf("Load() returned mutated data after caller mutation: got %q", string(loaded2))
	}
}

func TestMemoryStore_SaveAllCopiesInput(t *testing.T) {
	store := NewMemoryStore(WithCleanupInterval(24 * time.Hour))
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	expiresAt := time.Now().Add(time.Minute)

	data := []byte("hello")
	sessions := map[string]SessionData{
		"s1": {Data: data, ExpiresAt: expiresAt},
	}
	if err := store.SaveAll(ctx, sessions); err != nil {
		t.Fatalf("SaveAll() error: %v", err)
	}

	data[0] = 'j'
	loaded, err := store.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if string(loaded) != "hello" {
		t.Fatalf("Load() returned mutated data: got %q", string(loaded))
	}
}

func TestMemoryStore_CloseMakesOperationsFail(t *testing.T) {
	store := NewMemoryStore(WithCleanupInterval(24 * time.Hour))
	ctx := context.Background()

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() second call error: %v", err)
	}

	if err := store.Save(ctx, "s", []byte("x"), time.Now().Add(time.Minute)); err == nil {
		t.Fatal("Save() expected error after Close, got nil")
	}
	if _, err := store.Load(ctx, "s"); err == nil {
		t.Fatal("Load() expected error after Close, got nil")
	}
	if err := store.Touch(ctx, "s", time.Now().Add(time.Minute)); err == nil {
		t.Fatal("Touch() expected error after Close, got nil")
	}
	if err := store.Delete(ctx, "s"); err == nil {
		t.Fatal("Delete() expected error after Close, got nil")
	}
	if err := store.SaveAll(ctx, map[string]SessionData{}); err == nil {
		t.Fatal("SaveAll() expected error after Close, got nil")
	}
}

func TestMemoryStore_CleanupRemovesExpired(t *testing.T) {
	store := NewMemoryStore(WithCleanupInterval(24 * time.Hour))
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	if err := store.Save(ctx, "expired", []byte("x"), time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if err := store.Save(ctx, "alive", []byte("y"), time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	store.cleanup()

	if got := store.Count(); got != 1 {
		t.Fatalf("Count() got %d want 1", got)
	}
	if data, err := store.Load(ctx, "alive"); err != nil || string(data) != "y" {
		t.Fatalf("Load(alive) got %q err=%v", string(data), err)
	}
	if data, err := store.Load(ctx, "expired"); err != nil || data != nil {
		t.Fatalf("Load(expired) got %v err=%v", data, err)
	}
}

