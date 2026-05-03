package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"wargame/internal/config"
)

func TestNewS3MediaProfileImageStoreDisabled(t *testing.T) {
	if _, err := NewS3MediaProfileImageStore(context.Background(), config.S3Config{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMemoryProfileImageStorePresignUpload(t *testing.T) {
	store := NewMemoryProfileImageStore(5 * time.Minute)
	upload, err := store.PresignUpload(context.Background(), "profiles/3.jpg", "image/jpeg", 100*1024)
	if err != nil {
		t.Fatalf("presign upload: %v", err)
	}

	if upload.Method != "POST" || upload.URL == "" {
		t.Fatalf("unexpected upload: %+v", upload)
	}

	if upload.Fields["key"] != "profiles/3.jpg" {
		t.Fatalf("expected profile key, got %+v", upload.Fields)
	}

	if err := store.Delete(context.Background(), "profiles/3.jpg"); err != nil {
		t.Fatalf("delete should succeed: %v", err)
	}
}

func TestS3MediaProfileImageStorePresignUploadPolicyIncludesSizeLimit(t *testing.T) {
	store, err := NewS3MediaProfileImageStore(context.Background(), config.S3Config{
		Enabled:         true,
		Region:          "auto",
		Bucket:          "media-bucket",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Endpoint:        "https://example.com",
		PresignTTL:      10 * time.Minute,
		UploadMethod:    "post",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	upload, err := store.PresignUpload(context.Background(), "profiles/7.png", "image/png", 100*1024)
	if err != nil {
		t.Fatalf("presign upload: %v", err)
	}

	if upload.Method != "POST" {
		t.Fatalf("expected POST method, got %s", upload.Method)
	}

	policyEncoded, ok := upload.Fields["policy"]
	if !ok || policyEncoded == "" {
		t.Fatalf("expected policy field in presigned POST fields")
	}

	policyBytes, err := base64.StdEncoding.DecodeString(policyEncoded)
	if err != nil {
		t.Fatalf("decode policy: %v", err)
	}

	var policy struct {
		Conditions []any `json:"conditions"`
	}
	if err := json.Unmarshal(policyBytes, &policy); err != nil {
		t.Fatalf("unmarshal policy: %v", err)
	}

	foundRange := false
	for _, condition := range policy.Conditions {
		arr, ok := condition.([]any)
		if !ok || len(arr) != 3 {
			continue
		}

		key, ok := arr[0].(string)
		if !ok || key != "content-length-range" {
			continue
		}

		max, ok := arr[2].(float64)
		if !ok {
			t.Fatalf("expected numeric max in content-length-range, got %#v", arr[2])
		}

		if int64(max) != 100*1024 {
			t.Fatalf("expected max size 102400, got %v", max)
		}

		foundRange = true
		break
	}

	if !foundRange {
		t.Fatalf("content-length-range condition not found in policy: %s", string(policyBytes))
	}
}
