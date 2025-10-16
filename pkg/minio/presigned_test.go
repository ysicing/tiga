package minio

import (
	"testing"
	"time"
)

// These tests validate the cache behavior without requiring a live MinIO server.

func TestPresignedCache_GetSet(t *testing.T) {
	c := NewPresignedCache()
	c.set("GET|b|k", "http://example", time.Hour)
	if u, ok := c.get("GET|b|k"); !ok || u != "http://example" {
		t.Fatalf("cache get failed, got (%v,%v)", u, ok)
	}
}

func TestGenerateDownloadURL_UsesCache(t *testing.T) {
	c := NewPresignedCache()
	c.set("GET|bucket|key", "http://cached", time.Hour)
	// client is nil; function should return cache hit before using client
	u, err := GenerateDownloadURL(nil, nil, "bucket", "key", time.Hour, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != "http://cached" {
		t.Fatalf("expected cached url, got %s", u)
	}
}

func TestGenerateUploadURL_UsesCache(t *testing.T) {
	c := NewPresignedCache()
	c.set("PUT|bucket|key", "http://cached-put", time.Hour)
	u, err := GenerateUploadURL(nil, nil, "bucket", "key", time.Hour, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != "http://cached-put" {
		t.Fatalf("expected cached url, got %s", u)
	}
}
