package storage

import (
	"context"
	"errors"
	"time"
)

var ErrNotConfigured = errors.New("storage not configured")

type PresignedUpload struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Fields    map[string]string `json:"fields"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type PresignedURL struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ChallengeFileStore interface {
	PresignUpload(ctx context.Context, key, contentType string) (PresignedUpload, error)
	PresignDownload(ctx context.Context, key, filename string) (PresignedURL, error)
	Delete(ctx context.Context, key string) error
}

type ProfileImageStore interface {
	PresignUpload(ctx context.Context, key, contentType string, maxSizeBytes int64) (PresignedUpload, error)
	Delete(ctx context.Context, key string) error
}
