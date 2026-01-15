// Package upload provides file upload handling for Vango.
//
// Since WebSocket connections are poor at handling large binary uploads
// (blocking heartbeats), this package uses a hybrid HTTP+WebSocket approach.
//
// # The Problem
//
// Large file uploads over WebSocket block the heartbeat and event loop,
// causing connection timeouts and poor user experience.
//
// # The Solution
//
// Hybrid approach: HTTP POST for upload, WebSocket for processing.
//
//  1. User selects file in <input type="file">
//  2. Client performs HTTP POST to /upload endpoint (traditional)
//  3. Server streams to temp storage (disk/S3), returns temp_id
//  4. Client includes temp_id in form submission via WebSocket
//  5. Vango handler calls upload.Claim(temp_id) to finalize
//
// # Usage
//
// Mount the upload handler in your router:
//
//	r.Post("/upload", upload.Handler(uploadStore))
//
// # Security
//
// The upload handler enforces Config.AllowedTypes against a server-side detected MIME type
// (http.DetectContentType). Client-provided part headers like Content-Type are not trusted.
//
// For defense-in-depth, also consider:
//   - Restricting filename extensions via Config.AllowedExtensions
//   - Enforcing extension-to-type match via Config.RequireExtensionMatch
//   - Virus/malware scanning before making uploads available to end users
//
// Handle the uploaded file in your Vango component:
//
//	func CreatePost(ctx vango.Ctx, formData server.FormData) error {
//	    tempID := formData.Get("attachment_temp_id")
//
//	    var attachment *upload.File
//	    if tempID != "" {
//	        file, err := upload.Claim(uploadStore, tempID)
//	        if err != nil {
//	            return err
//	        }
//	        attachment = file
//	    }
//
//	    // Use attachment...
//	    return nil
//	}
package upload
