package upload

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DiskStore stores uploads on the local filesystem.
type DiskStore struct {
	dir     string
	maxSize int64

	mu    sync.RWMutex
	files map[string]*diskMeta
}

type diskMeta struct {
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewDiskStore creates a new DiskStore.
//
// Parameters:
//   - dir: Directory to store temp files
//   - maxSize: Maximum file size in bytes (0 = no limit)
func NewDiskStore(dir string, maxSize int64) (*DiskStore, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &DiskStore{
		dir:     dir,
		maxSize: maxSize,
		files:   make(map[string]*diskMeta),
	}, nil
}

// Save stores the uploaded file and returns a temp ID.
func (s *DiskStore) Save(filename, contentType string, size int64, r io.Reader) (string, error) {
	// Check size limit
	if s.maxSize > 0 && size > s.maxSize {
		return "", ErrTooLarge
	}

	// Generate temp ID
	tempID := generateTempID()
	path := filepath.Join(s.dir, tempID)

	// Create temp file
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Copy with size limit
	var reader io.Reader = r
	if s.maxSize > 0 {
		reader = io.LimitReader(r, s.maxSize+1) // +1 to detect overflow
	}

	written, err := io.Copy(f, reader)
	if err != nil {
		os.Remove(path)
		return "", err
	}

	// Check if we hit the limit
	if s.maxSize > 0 && written > s.maxSize {
		os.Remove(path)
		return "", ErrTooLarge
	}

	// Store metadata
	meta := &diskMeta{
		Filename:    filename,
		ContentType: contentType,
		Size:        written,
		CreatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.files[tempID] = meta
	s.mu.Unlock()

	// Also save metadata to disk for persistence
	s.saveMeta(tempID, meta)

	return tempID, nil
}

// Claim retrieves and removes a temp file.
func (s *DiskStore) Claim(tempID string) (*File, error) {
	s.mu.Lock()
	meta, ok := s.files[tempID]
	if ok {
		delete(s.files, tempID)
	}
	s.mu.Unlock()

	// Try loading from disk if not in memory
	if !ok {
		var err error
		meta, err = s.loadMeta(tempID)
		if err != nil {
			return nil, ErrNotFound
		}
	}

	path := filepath.Join(s.dir, tempID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrNotFound
	}

	// Open file for reading
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// Delete the temp file after it's been read
	// We return a wrapped reader that deletes on close
	return &File{
		ID:          tempID,
		Filename:    meta.Filename,
		ContentType: meta.ContentType,
		Size:        meta.Size,
		Path:        path,
		Reader:      &deleteOnCloseReader{File: f, path: path, metaPath: s.metaPath(tempID)},
	}, nil
}

// Cleanup removes expired temp files.
func (s *DiskStore) Cleanup(maxAge time.Duration) error {
	now := time.Now()
	cutoff := now.Add(-maxAge)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clean up in-memory entries
	for tempID, meta := range s.files {
		if meta.CreatedAt.Before(cutoff) {
			delete(s.files, tempID)
			os.Remove(filepath.Join(s.dir, tempID))
			os.Remove(s.metaPath(tempID))
		}
	}

	// Also scan directory for orphaned files
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(s.dir, entry.Name()))
		}
	}

	return nil
}

func (s *DiskStore) metaPath(tempID string) string {
	return filepath.Join(s.dir, tempID+".meta")
}

func (s *DiskStore) saveMeta(tempID string, meta *diskMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(tempID), data, 0644)
}

func (s *DiskStore) loadMeta(tempID string) (*diskMeta, error) {
	data, err := os.ReadFile(s.metaPath(tempID))
	if err != nil {
		return nil, err
	}
	var meta diskMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// generateTempID generates a cryptographically random temp ID.
func generateTempID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// deleteOnCloseReader wraps a file and deletes it when closed.
type deleteOnCloseReader struct {
	*os.File
	path     string
	metaPath string
}

func (r *deleteOnCloseReader) Close() error {
	err := r.File.Close()
	os.Remove(r.path)
	os.Remove(r.metaPath)
	return err
}
