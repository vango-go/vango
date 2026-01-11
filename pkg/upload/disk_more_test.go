package upload_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/upload"
)

func TestDiskStore_SaveRejectsWhenReaderExceedsLimitEvenIfDeclaredSizeIsSmaller(t *testing.T) {
	dir := t.TempDir()
	store, err := upload.NewDiskStore(dir, 5)
	if err != nil {
		t.Fatalf("NewDiskStore: %v", err)
	}

	// size says 4, but reader provides 6 bytes.
	_, err = store.Save("x.txt", "text/plain", 4, bytes.NewReader([]byte("123456")))
	if err != upload.ErrTooLarge {
		t.Fatalf("err = %v, want %v", err, upload.ErrTooLarge)
	}
}

func TestDiskStore_ClaimLoadsMetadataFromDiskWhenNotInMemory(t *testing.T) {
	dir := t.TempDir()

	store1, err := upload.NewDiskStore(dir, 0)
	if err != nil {
		t.Fatalf("NewDiskStore(store1): %v", err)
	}

	content := []byte("persist me")
	tempID, err := store1.Save("persist.txt", "text/plain", int64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	metaPath := filepath.Join(dir, tempID+".meta")
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected meta file to exist: %v", err)
	}

	// New store instance simulates a restart (no in-memory map entry).
	store2, err := upload.NewDiskStore(dir, 0)
	if err != nil {
		t.Fatalf("NewDiskStore(store2): %v", err)
	}

	file, err := store2.Claim(tempID)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	data, err := io.ReadAll(file.Reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("content mismatch")
	}

	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, tempID)); !os.IsNotExist(err) {
		t.Fatalf("expected claimed file to be deleted on close; stat err=%v", err)
	}
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Fatalf("expected meta file to be deleted on close; stat err=%v", err)
	}
}

func TestDiskStore_ClaimValidTempIDButMissingFileReturnsNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := upload.NewDiskStore(dir, 0)
	if err != nil {
		t.Fatalf("NewDiskStore: %v", err)
	}

	// Valid 32-hex tempID, but only metadata exists.
	tempID := "0123456789abcdef0123456789abcdef"
	metaPath := filepath.Join(dir, tempID+".meta")

	metaJSON, err := json.Marshal(map[string]any{
		"filename":     "missing.txt",
		"content_type": "text/plain",
		"size":         3,
		"created_at":   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Marshal meta: %v", err)
	}
	if err := os.WriteFile(metaPath, metaJSON, 0644); err != nil {
		t.Fatalf("WriteFile meta: %v", err)
	}

	_, err = store.Claim(tempID)
	if err != upload.ErrNotFound {
		t.Fatalf("err = %v, want %v", err, upload.ErrNotFound)
	}
}

func TestDiskStore_ClaimRejectsTraversalTempID(t *testing.T) {
	dir := t.TempDir()
	store, err := upload.NewDiskStore(dir, 0)
	if err != nil {
		t.Fatalf("NewDiskStore: %v", err)
	}

	_, err = store.Claim("../../etc/passwd")
	if err != upload.ErrNotFound {
		t.Fatalf("err = %v, want %v", err, upload.ErrNotFound)
	}
}

func TestDiskStore_CleanupRemovesOrphanedOldFilesButNotDirectories(t *testing.T) {
	dir := t.TempDir()
	store, err := upload.NewDiskStore(dir, 0)
	if err != nil {
		t.Fatalf("NewDiskStore: %v", err)
	}

	oldFile := filepath.Join(dir, "orphan.bin")
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile oldFile: %v", err)
	}
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes oldFile: %v", err)
	}

	recentFile := filepath.Join(dir, "recent.bin")
	if err := os.WriteFile(recentFile, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile recentFile: %v", err)
	}

	subdir := filepath.Join(dir, "keepdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("MkdirAll subdir: %v", err)
	}
	nestedOld := filepath.Join(subdir, "nested-old.bin")
	if err := os.WriteFile(nestedOld, []byte("nested"), 0644); err != nil {
		t.Fatalf("WriteFile nestedOld: %v", err)
	}
	if err := os.Chtimes(nestedOld, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes nestedOld: %v", err)
	}

	if err := store.Cleanup(1 * time.Hour); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("expected oldFile to be deleted; stat err=%v", err)
	}
	if _, err := os.Stat(recentFile); err != nil {
		t.Fatalf("expected recentFile to remain; stat err=%v", err)
	}
	if _, err := os.Stat(subdir); err != nil {
		t.Fatalf("expected subdir to remain; stat err=%v", err)
	}
	if _, err := os.Stat(nestedOld); err != nil {
		t.Fatalf("expected nested file to remain (cleanup does not recurse); stat err=%v", err)
	}
}

type closeTracker struct {
	closed bool
	err    error
}

func (c *closeTracker) Read([]byte) (int, error) { return 0, io.EOF }
func (c *closeTracker) Close() error {
	c.closed = true
	return c.err
}

func TestFile_CloseClosesReaderWhenPresent(t *testing.T) {
	tracker := &closeTracker{}
	f := &upload.File{Reader: tracker}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !tracker.closed {
		t.Fatalf("expected reader to be closed")
	}
}

func TestFile_CloseNoopWhenNoReader(t *testing.T) {
	f := &upload.File{}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

type claimOnlyStore struct {
	called bool
	gotID  string
	file   *upload.File
	err    error
}

func (s *claimOnlyStore) Save(string, string, int64, io.Reader) (string, error) { panic("unexpected") }
func (s *claimOnlyStore) Claim(tempID string) (*upload.File, error) {
	s.called = true
	s.gotID = tempID
	return s.file, s.err
}
func (s *claimOnlyStore) Cleanup(time.Duration) error { panic("unexpected") }

func TestClaim_DelegatesToStore(t *testing.T) {
	wantFile := &upload.File{ID: "abc"}
	store := &claimOnlyStore{file: wantFile}

	gotFile, err := upload.Claim(store, "temp123")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if !store.called || store.gotID != "temp123" {
		t.Fatalf("store.Claim not called as expected (called=%v gotID=%q)", store.called, store.gotID)
	}
	if gotFile != wantFile {
		t.Fatalf("file pointer mismatch")
	}
}
