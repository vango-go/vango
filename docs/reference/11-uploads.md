# File Uploads

Handle file uploads with Vango's hybrid HTTP + WebSocket approach.

## Why Hybrid?

Large file uploads over WebSocket can block heartbeats. Vango uses HTTP for upload, WebSocket for claiming:

1. User selects file → JavaScript POSTs to `/upload`
2. Server stores file, returns `temp_id`
3. Form submission includes `temp_id` via WebSocket
4. Handler calls `upload.Claim(tempID)` to get the file

## Server Setup

```go
import "github.com/vango-dev/vango/v2/pkg/upload"

// Create a disk store
store, _ := upload.NewDiskStore("./uploads", 50<<20)  // 50MB max

// Mount handler on your router
r.Post("/upload", upload.Handler(store))
```

## Claiming Files

```go
func CreatePost(ctx vango.Ctx, form FormData) error {
    tempID := form.Get("attachment_id")
    
    if tempID != "" {
        file, err := upload.Claim(store, tempID)
        if err != nil {
            toast.Error(ctx, "File not found or expired")
            return err
        }
        defer file.Close()
        
        // Access file properties
        file.Filename    // Original filename
        file.ContentType // MIME type
        file.Size        // Bytes
        file.Reader      // io.Reader for streaming
        file.Path        // Disk path (for DiskStore)
    }
    
    return nil
}
```

## Client-Side Upload

```javascript
// Simple upload function
async function uploadFile(file) {
    const formData = new FormData();
    formData.append("file", file);
    
    const res = await fetch("/upload", {
        method: "POST",
        body: formData,
    });
    
    const { temp_id } = await res.json();
    return temp_id;
}

// Set hidden input with temp_id before form submit
input.addEventListener("change", async (e) => {
    const tempId = await uploadFile(e.target.files[0]);
    document.getElementById("attachment_id").value = tempId;
});
```

## Cleanup

Unclaimed files are automatically cleaned up:

```go
// Run periodically (e.g., every hour)
store.Cleanup(1 * time.Hour)  // Remove files older than 1 hour
```

## Size Limits

```go
// Set max size when creating store
store, _ := upload.NewDiskStore("./uploads", 10<<20)  // 10MB

// Oversized uploads return upload.ErrTooLarge
```

## Allowed File Types

Restrict uploads to specific MIME types:

```go
config := &upload.Config{
    MaxFileSize:  10 << 20,  // 10MB
    AllowedTypes: []string{
        "image/jpeg",
        "image/png",
        "image/gif",
        "application/pdf",
    },
}

http.Handle("/upload", upload.HandlerWithConfig(store, config))
```

Uploads with disallowed types return `415 Unsupported Media Type`.

> **Note**: MIME type matching is case-insensitive.

## Security

Vango upload handler includes several protections:

- **DoS prevention**: `http.MaxBytesReader` limits request body *before* parsing
- **Path traversal**: Temp IDs are hex-validated, paths are sanitized
- **Type validation**: Optional MIME type whitelist
- **Cryptographic IDs**: Temp IDs use `crypto/rand`

## S3 Storage

For production, implement the `upload.Store` interface for S3:

```go
type Store interface {
    Save(filename, contentType string, size int64, r io.Reader) (tempID string, err error)
    Claim(tempID string) (*File, error)
    Cleanup(maxAge time.Duration) error
}
```

See `pkg/upload/s3_example.go` for a complete S3 implementation.
