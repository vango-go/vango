package upload_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/upload"
)

type recordingStore struct {
	tempID string
	saveFn func(filename, contentType string, size int64, r io.Reader) (string, error)
}

func (s *recordingStore) Save(filename, contentType string, size int64, r io.Reader) (string, error) {
	if s.saveFn != nil {
		return s.saveFn(filename, contentType, size, r)
	}
	if s.tempID == "" {
		return "temp123", nil
	}
	return s.tempID, nil
}

func (s *recordingStore) Claim(string) (*upload.File, error) {
	return nil, errors.New("not implemented")
}
func (s *recordingStore) Cleanup(_ time.Duration) error { return errors.New("not implemented") }

func newMultipartUploadRequest(t *testing.T, filename string, contentTypeHeader string, content []byte) (*http.Request, string) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	if contentTypeHeader != "" {
		h.Set("Content-Type", contentTypeHeader)
	}

	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("part.Write: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, filename
}

func TestHandler_RejectsNonPOST(t *testing.T) {
	h := upload.Handler(&recordingStore{})
	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_FailsWhenNotMultipart(t *testing.T) {
	h := upload.Handler(&recordingStore{})
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewBufferString("not multipart"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_FailsWhenNoFileProvided(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("not_file", "x"); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	h := upload.Handler(&recordingStore{})
	req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_EnforcesAllowedTypes(t *testing.T) {
	store := &recordingStore{
		saveFn: func(string, string, int64, io.Reader) (string, error) {
			t.Fatal("Save should not be called when MIME type is rejected")
			return "", nil
		},
	}

	h := upload.HandlerWithConfig(store, &upload.Config{
		MaxFileSize:  1024 * 1024,
		AllowedTypes: []string{"image/png"},
		TempExpiry:   0,
	})

	// Spoof Content-Type header, but upload bytes are JPEG.
	content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01}
	req, _ := newMultipartUploadRequest(t, "test.png", "image/png; charset=binary", content)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnsupportedMediaType)
	}
}

func TestHandler_AllowsAllowedTypes_CaseAndParams(t *testing.T) {
	var got struct {
		filename    string
		contentType string
		size        int64
		data        []byte
	}

	store := &recordingStore{
		tempID: "abc123",
		saveFn: func(filename, contentType string, size int64, r io.Reader) (string, error) {
			got.filename = filename
			got.contentType = contentType
			got.size = size
			data, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			got.data = data
			return "abc123", nil
		},
	}

	h := upload.HandlerWithConfig(store, &upload.Config{
		MaxFileSize:  1024 * 1024,
		AllowedTypes: []string{"IMAGE/PNG"},
	})

	content := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}, []byte("not a real png, but signature is enough")...)
	req, _ := newMultipartUploadRequest(t, "test.png", "Image/PNG; charset=utf-8", content)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", ct, "application/json")
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal response: %v; body=%q", err, rec.Body.String())
	}
	if resp["temp_id"] != "abc123" {
		t.Fatalf("temp_id = %q, want %q", resp["temp_id"], "abc123")
	}

	if got.filename != "test.png" {
		t.Fatalf("Save filename = %q, want %q", got.filename, "test.png")
	}
	if got.contentType != "image/png" {
		t.Fatalf("Save contentType = %q, want %q", got.contentType, "image/png")
	}
	if got.size != int64(len(content)) {
		t.Fatalf("Save size = %d, want %d", got.size, len(content))
	}
	if !bytes.Equal(got.data, content) {
		t.Fatalf("Save reader data mismatch")
	}
}

func TestHandler_EnforcesAllowedExtensions(t *testing.T) {
	store := &recordingStore{
		saveFn: func(string, string, int64, io.Reader) (string, error) {
			t.Fatal("Save should not be called when extension is rejected")
			return "", nil
		},
	}

	h := upload.HandlerWithConfig(store, &upload.Config{
		MaxFileSize:       1024 * 1024,
		AllowedTypes:      []string{"image/png"},
		AllowedExtensions: []string{".png"},
	})

	content := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	req, _ := newMultipartUploadRequest(t, "test.jpg", "image/png", content)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnsupportedMediaType)
	}
}

func TestHandler_RequireExtensionMatchRejectsMismatch(t *testing.T) {
	store := &recordingStore{
		saveFn: func(string, string, int64, io.Reader) (string, error) {
			t.Fatal("Save should not be called when extension does not match detected type")
			return "", nil
		},
	}

	h := upload.HandlerWithConfig(store, &upload.Config{
		MaxFileSize:           1024 * 1024,
		AllowedTypes:          []string{"image/png"},
		RequireExtensionMatch: true,
	})

	content := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	req, _ := newMultipartUploadRequest(t, "test.jpg", "image/png", content)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnsupportedMediaType)
	}
}

func TestHandler_RejectsTooLargeBodyAtHTTPLevel(t *testing.T) {
	h := upload.HandlerWithConfig(&recordingStore{}, &upload.Config{
		MaxFileSize: 16, // intentionally tiny; includes multipart overhead
	})

	req, _ := newMultipartUploadRequest(t, "test.txt", "text/plain", bytes.Repeat([]byte("a"), 256))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}

func TestHandler_MapsStoreErrTooLargeTo413(t *testing.T) {
	store := &recordingStore{
		saveFn: func(string, string, int64, io.Reader) (string, error) {
			return "", upload.ErrTooLarge
		},
	}
	h := upload.HandlerWithConfig(store, &upload.Config{
		MaxFileSize: 1024 * 1024,
	})

	req, _ := newMultipartUploadRequest(t, "test.txt", "text/plain", []byte("x"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHandler_MapsStoreErrorTo500(t *testing.T) {
	store := &recordingStore{
		saveFn: func(string, string, int64, io.Reader) (string, error) {
			return "", errors.New("boom")
		},
	}
	h := upload.HandlerWithConfig(store, &upload.Config{
		MaxFileSize: 1024 * 1024,
	})

	req, _ := newMultipartUploadRequest(t, "test.txt", "text/plain", []byte("x"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
