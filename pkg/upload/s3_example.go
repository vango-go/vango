//go:build s3example
// +build s3example

// This file provides an example S3Store implementation.
// It is excluded from regular builds because it requires the AWS SDK.
//
// To use this in your project, copy this file and add the AWS SDK:
//   go get github.com/aws/aws-sdk-go-v2
//   go get github.com/aws/aws-sdk-go-v2/config
//   go get github.com/aws/aws-sdk-go-v2/service/s3

package upload

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Store stores uploads in AWS S3.
//
// Example usage:
//
//	cfg, _ := config.LoadDefaultConfig(context.Background())
//	s3Client := s3.NewFromConfig(cfg)
//	store := upload.NewS3Store(s3Client, "my-bucket", "uploads/", 50<<20)
//
//	r.Post("/upload", upload.Handler(store))
type S3Store struct {
	client    *s3.Client
	bucket    string
	prefix    string
	maxSize   int64
	urlExpiry time.Duration
}

// NewS3Store creates a new S3 upload store.
//
// Parameters:
//   - client: AWS S3 client from aws-sdk-go-v2
//   - bucket: S3 bucket name
//   - prefix: Key prefix for uploads (e.g., "uploads/temp/")
//   - maxSize: Maximum file size in bytes (0 = no limit)
func NewS3Store(client *s3.Client, bucket, prefix string, maxSize int64) *S3Store {
	return &S3Store{
		client:    client,
		bucket:    bucket,
		prefix:    prefix,
		maxSize:   maxSize,
		urlExpiry: 24 * time.Hour,
	}
}

// WithURLExpiry sets how long presigned URLs are valid.
func (s *S3Store) WithURLExpiry(d time.Duration) *S3Store {
	s.urlExpiry = d
	return s
}

// Save uploads a file to S3 and returns a temp ID.
func (s *S3Store) Save(filename, contentType string, size int64, r io.Reader) (string, error) {
	// Check size limit
	if s.maxSize > 0 && size > s.maxSize {
		return "", ErrTooLarge
	}

	// Generate temp ID
	tempID := s.generateTempID()
	key := s.prefix + tempID

	// Read file into memory (for small files) or use multipart upload for large files
	// For simplicity, we buffer the file. For production, consider streaming.
	var buf bytes.Buffer
	if s.maxSize > 0 {
		limited := io.LimitReader(r, s.maxSize+1)
		n, err := io.Copy(&buf, limited)
		if err != nil {
			return "", err
		}
		if n > s.maxSize {
			return "", ErrTooLarge
		}
	} else {
		if _, err := io.Copy(&buf, r); err != nil {
			return "", err
		}
	}

	// Upload to S3
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"original-filename": filename,
			"upload-time":       time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return "", fmt.Errorf("s3 upload failed: %w", err)
	}

	return tempID, nil
}

// Claim retrieves a temp file from S3.
func (s *S3Store) Claim(tempID string) (*File, error) {
	key := s.prefix + tempID

	// Get object metadata
	headResult, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, ErrNotFound
	}

	// Get the actual object
	getResult, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, ErrNotFound
	}

	// Extract original filename from metadata
	filename := tempID
	if fn, ok := headResult.Metadata["original-filename"]; ok {
		filename = fn
	}

	contentType := "application/octet-stream"
	if headResult.ContentType != nil {
		contentType = *headResult.ContentType
	}

	size := int64(0)
	if headResult.ContentLength != nil {
		size = *headResult.ContentLength
	}

	// Generate presigned URL for direct access
	presignClient := s3.NewPresignClient(s.client)
	presignResult, err := presignClient.PresignGetObject(context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(s.urlExpiry),
	)

	url := ""
	if err == nil {
		url = presignResult.URL
	}

	// Delete the temp object (claimed = consumed)
	go func() {
		s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		})
	}()

	return &File{
		ID:          tempID,
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		URL:         url,
		Reader:      getResult.Body,
	}, nil
}

// Cleanup removes expired temp files from S3.
func (s *S3Store) Cleanup(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)

	// List objects with prefix
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.prefix),
	})

	var toDelete []string

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return err
		}

		for _, obj := range page.Contents {
			if obj.LastModified != nil && obj.LastModified.Before(cutoff) {
				if obj.Key != nil {
					toDelete = append(toDelete, *obj.Key)
				}
			}
		}
	}

	// Delete expired objects
	for _, key := range toDelete {
		s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		})
	}

	return nil
}

func (s *S3Store) generateTempID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
