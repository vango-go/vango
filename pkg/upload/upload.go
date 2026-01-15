package upload

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotFound is returned when a temp file doesn't exist.
var ErrNotFound = errors.New("upload: file not found")

// ErrExpired is returned when a temp file has expired.
var ErrExpired = errors.New("upload: file expired")

// ErrTooLarge is returned when a file exceeds the size limit.
var ErrTooLarge = errors.New("upload: file too large")

// ErrTypeNotAllowed is returned when a file's MIME type is not in AllowedTypes.
var ErrTypeNotAllowed = errors.New("upload: file type not allowed")

// Store is the interface for upload storage backends.
// Implement this interface to use S3, GCS, or other storage.
type Store interface {
	// Save stores the uploaded file and returns a temp ID.
	// The file is stored temporarily until Claim is called.
	Save(filename string, contentType string, size int64, r io.Reader) (tempID string, err error)

	// Claim retrieves and removes a temp file, returning a file handle.
	// After claiming, the temp file is deleted (or marked for deletion).
	Claim(tempID string) (*File, error)

	// Cleanup removes expired temp files.
	// Call this periodically (e.g., every 5 minutes).
	Cleanup(maxAge time.Duration) error
}

// File represents an uploaded file.
type File struct {
	// ID is the unique identifier for this upload.
	ID string

	// Filename is the original filename from the client.
	Filename string

	// ContentType is the MIME type of the file.
	ContentType string

	// Size is the file size in bytes.
	Size int64

	// Path is the local filesystem path (for DiskStore).
	Path string

	// URL is the remote URL (for S3/CDN storage).
	URL string

	// Reader provides access to the file contents.
	// May be nil if the file is stored on disk (use Path instead).
	Reader io.ReadCloser
}

// Close closes the file reader if open.
func (f *File) Close() error {
	if f.Reader != nil {
		return f.Reader.Close()
	}
	return nil
}

// Handler returns an http.Handler for file uploads.
// Mount this on your router: r.Post("/upload", upload.Handler(store))
//
// The handler expects a multipart form with a "file" field.
// It returns JSON with the temp_id:
//
//	{"temp_id": "abc123"}
func Handler(store Store) http.Handler {
	return HandlerWithConfig(store, DefaultConfig())
}

// HandlerWithConfig returns an upload handler with custom configuration.
func HandlerWithConfig(store Store, config *Config) http.Handler {
	maxSize := config.MaxFileSize
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // 10MB default
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// SECURITY: Limit request body size BEFORE parsing to prevent DoS
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)

		// Parse multipart form (32MB max in memory, but body already limited)
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			// Check if it was a size limit error (MaxBytesReader can be wrapped).
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) ||
				errors.Is(err, multipart.ErrMessageTooLarge) ||
				strings.Contains(err.Error(), "http: request body too large") {
				http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file provided", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// SECURITY: Never trust client-provided part headers (e.g. Content-Type). Detect from bytes.
		sniff := make([]byte, 512)
		n, err := io.ReadFull(file, sniff)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			http.Error(w, "Failed to read upload", http.StatusBadRequest)
			return
		}
		sniff = sniff[:n]
		contentType := http.DetectContentType(sniff)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			http.Error(w, "Failed to read upload", http.StatusBadRequest)
			return
		}

		// Check MIME type if AllowedTypes is configured
		if len(config.AllowedTypes) > 0 && !isTypeAllowed(contentType, config.AllowedTypes) {
			http.Error(w, "File type not allowed", http.StatusUnsupportedMediaType)
			return
		}

		// Optional extension allowlist.
		if len(config.AllowedExtensions) > 0 && !isExtensionAllowed(header.Filename, config.AllowedExtensions) {
			http.Error(w, "File type not allowed", http.StatusUnsupportedMediaType)
			return
		}

		// Optional extension-to-mime match enforcement.
		if config.RequireExtensionMatch && !extensionMatchesType(header.Filename, contentType) {
			http.Error(w, "File type not allowed", http.StatusUnsupportedMediaType)
			return
		}

		// Store the file
		tempID, err := store.Save(
			header.Filename,
			contentType,
			header.Size,
			file,
		)
		if err != nil {
			if errors.Is(err, ErrTooLarge) {
				http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "Upload failed", http.StatusInternalServerError)
			return
		}

		// Return temp ID as JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"temp_id": tempID,
		})
	})
}

// Claim retrieves a temp file by ID.
// Call this in your Vango handler after receiving the temp_id.
//
// Example:
//
//	file, err := upload.Claim(store, tempID)
//	if err != nil {
//	    return err
//	}
//	defer file.Close()
//	// Use file.Path or file.Reader
func Claim(store Store, tempID string) (*File, error) {
	return store.Claim(tempID)
}

// Config holds configuration for the upload handler.
type Config struct {
	// MaxFileSize is the maximum allowed file size in bytes.
	// Default: 10MB.
	MaxFileSize int64

	// AllowedTypes is a list of allowed MIME types.
	// If empty, all types are allowed.
	AllowedTypes []string

	// AllowedExtensions is a list of allowed filename extensions (e.g. ".png", "jpg").
	// If empty, all extensions are allowed.
	AllowedExtensions []string

	// RequireExtensionMatch rejects uploads whose filename extension does not match the detected MIME type.
	// This is an optional defense-in-depth check to avoid content-type/extension confusion.
	RequireExtensionMatch bool

	// TempExpiry is how long temp files live before cleanup.
	// Default: 1 hour.
	TempExpiry time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		TempExpiry:  time.Hour,
	}
}

func normalizeMIMEType(contentType string) string {
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

// isTypeAllowed checks if contentType is in the allowed list (case-insensitive).
func isTypeAllowed(contentType string, allowed []string) bool {
	contentType = normalizeMIMEType(contentType)

	for _, a := range allowed {
		if normalizeMIMEType(a) == contentType {
			return true
		}
	}
	return false
}

func normalizeExtension(ext string) string {
	ext = strings.TrimSpace(ext)
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return strings.ToLower(ext)
}

func isExtensionAllowed(filename string, allowed []string) bool {
	ext := normalizeExtension(filepath.Ext(filename))
	if ext == "" {
		return false
	}
	for _, a := range allowed {
		if normalizeExtension(a) == ext {
			return true
		}
	}
	return false
}

func extensionMatchesType(filename, contentType string) bool {
	ext := normalizeExtension(filepath.Ext(filename))
	if ext == "" {
		return false
	}

	baseType := normalizeMIMEType(contentType)
	if baseType == "" {
		return false
	}

	// Start with a small deterministic mapping for common types.
	switch baseType {
	case "image/jpeg":
		if ext == ".jpg" || ext == ".jpeg" || ext == ".jpe" {
			return true
		}
	case "image/png":
		if ext == ".png" {
			return true
		}
	case "image/gif":
		if ext == ".gif" {
			return true
		}
	case "image/webp":
		if ext == ".webp" {
			return true
		}
	case "application/pdf":
		if ext == ".pdf" {
			return true
		}
	case "text/plain":
		if ext == ".txt" {
			return true
		}
	case "text/html":
		if ext == ".html" || ext == ".htm" {
			return true
		}
	case "application/json":
		if ext == ".json" {
			return true
		}
	case "application/zip":
		if ext == ".zip" {
			return true
		}
	}

	// Fall back to the Go MIME database.
	exts, err := mime.ExtensionsByType(baseType)
	if err != nil {
		return false
	}
	for _, e := range exts {
		if normalizeExtension(e) == ext {
			return true
		}
	}
	return false
}
