package upload_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/upload"
)

func TestDiskStore_SaveAndClaim(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()

	store, err := upload.NewDiskStore(dir, 10*1024*1024)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Save a file
	content := []byte("hello world")
	tempID, err := store.Save("test.txt", "text/plain", int64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	if tempID == "" {
		t.Fatal("expected non-empty temp ID")
	}

	// Claim the file
	file, err := store.Claim(tempID)
	if err != nil {
		t.Fatalf("failed to claim: %v", err)
	}
	defer file.Close()

	if file.Filename != "test.txt" {
		t.Errorf("expected filename test.txt, got %s", file.Filename)
	}
	if file.ContentType != "text/plain" {
		t.Errorf("expected content type text/plain, got %s", file.ContentType)
	}
	if file.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), file.Size)
	}

	// Read content
	data, err := io.ReadAll(file.Reader)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch")
	}
}

func TestDiskStore_ClaimDeletesFile(t *testing.T) {
	dir := t.TempDir()
	store, _ := upload.NewDiskStore(dir, 0)

	content := []byte("data")
	tempID, _ := store.Save("file.txt", "text/plain", int64(len(content)), bytes.NewReader(content))

	// Verify file exists
	path := filepath.Join(dir, tempID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("file should exist before claim")
	}

	// Claim and close
	file, _ := store.Claim(tempID)
	file.Close()

	// File should be deleted
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted after close")
	}
}

func TestDiskStore_ClaimNotFound(t *testing.T) {
	dir := t.TempDir()
	store, _ := upload.NewDiskStore(dir, 0)

	_, err := store.Claim("nonexistent")
	if err != upload.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDiskStore_SizeLimitExceeded(t *testing.T) {
	dir := t.TempDir()
	store, _ := upload.NewDiskStore(dir, 10) // 10 byte limit

	content := []byte("this is more than 10 bytes")
	_, err := store.Save("big.txt", "text/plain", int64(len(content)), bytes.NewReader(content))

	if err != upload.ErrTooLarge {
		t.Errorf("expected ErrTooLarge, got %v", err)
	}
}

func TestDiskStore_Cleanup(t *testing.T) {
	dir := t.TempDir()
	store, _ := upload.NewDiskStore(dir, 0)

	// Save a file
	content := []byte("temp data")
	tempID, _ := store.Save("temp.txt", "text/plain", int64(len(content)), bytes.NewReader(content))

	// Verify file exists
	path := filepath.Join(dir, tempID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("file should exist")
	}

	// Cleanup with very short max age should delete it
	time.Sleep(10 * time.Millisecond)
	store.Cleanup(1 * time.Nanosecond)

	// File should be deleted
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted after cleanup")
	}
}

func TestDiskStore_DoubleClaim(t *testing.T) {
	dir := t.TempDir()
	store, _ := upload.NewDiskStore(dir, 0)

	content := []byte("data")
	tempID, _ := store.Save("file.txt", "text/plain", int64(len(content)), bytes.NewReader(content))

	// First claim
	file, err := store.Claim(tempID)
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	file.Close()

	// Second claim should fail
	_, err = store.Claim(tempID)
	if err != upload.ErrNotFound {
		t.Errorf("expected ErrNotFound on second claim, got %v", err)
	}
}
