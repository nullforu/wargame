package storage

import (
	"context"
	"sync"
	"time"
)

type MemoryChallengeFileStore struct {
	presignTTL time.Duration
	mu         sync.Mutex // concurrent access to keys map
	keys       map[string]struct{}
}

type MemoryProfileImageStore struct {
	presignTTL time.Duration
}

// for testing purposes
func NewMemoryChallengeFileStore(presignTTL time.Duration) *MemoryChallengeFileStore {
	if presignTTL <= 0 {
		presignTTL = defaultPresignTTL
	}

	return &MemoryChallengeFileStore{
		presignTTL: presignTTL,
		keys:       make(map[string]struct{}),
	}
}

func NewMemoryProfileImageStore(presignTTL time.Duration) *MemoryProfileImageStore {
	if presignTTL <= 0 {
		presignTTL = defaultPresignTTL
	}

	return &MemoryProfileImageStore{presignTTL: presignTTL}
}

func (m *MemoryChallengeFileStore) PresignUpload(ctx context.Context, key, contentType string) (PresignedUpload, error) {
	_ = ctx
	m.mu.Lock()
	m.keys[key] = struct{}{}
	m.mu.Unlock()

	return PresignedUpload{
		URL:    "https://example.com/upload",
		Method: "POST",
		Fields: map[string]string{
			"key":          key,
			"Content-Type": contentType,
		},
		ExpiresAt: time.Now().UTC().Add(m.presignTTL),
	}, nil
}

func (m *MemoryChallengeFileStore) PresignDownload(ctx context.Context, key, filename string) (PresignedURL, error) {
	_ = ctx
	_ = filename

	return PresignedURL{
		URL:       "https://example.com/download/" + key,
		ExpiresAt: time.Now().UTC().Add(m.presignTTL),
	}, nil
}

func (m *MemoryChallengeFileStore) Delete(ctx context.Context, key string) error {
	_ = ctx
	m.mu.Lock()
	delete(m.keys, key)
	m.mu.Unlock()

	return nil
}

func (m *MemoryProfileImageStore) PresignUpload(ctx context.Context, key, contentType string, maxSizeBytes int64) (PresignedUpload, error) {
	_ = ctx
	_ = maxSizeBytes

	return PresignedUpload{
		URL:    "https://example.com/upload-profile",
		Method: "POST",
		Fields: map[string]string{
			"key":          key,
			"Content-Type": contentType,
		},
		ExpiresAt: time.Now().UTC().Add(m.presignTTL),
	}, nil
}

func (m *MemoryProfileImageStore) Delete(ctx context.Context, key string) error {
	_ = ctx
	_ = key
	return nil
}
