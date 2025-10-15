package minio

import (
	"context"
	"sync"
	"time"

	sdk "github.com/minio/minio-go/v7"
)

type cacheEntry struct {
	url     string
	expires time.Time
}

type PresignedCache struct {
	mu sync.Mutex
	m  map[string]cacheEntry
}

func NewPresignedCache() *PresignedCache { return &PresignedCache{m: make(map[string]cacheEntry)} }

func (c *PresignedCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.m[key]; ok && time.Now().Before(e.expires) {
		return e.url, true
	}
	return "", false
}

func (c *PresignedCache) set(key, url string, d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = cacheEntry{url: url, expires: time.Now().Add(d / 2)} // refresh halfway
}

func GenerateDownloadURL(ctx context.Context, client *sdk.Client, bucket, key string, expiry time.Duration, cache *PresignedCache) (string, error) {
	ck := "GET|" + bucket + "|" + key
	if cache != nil {
		if u, ok := cache.get(ck); ok {
			return u, nil
		}
	}
	u, err := client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	if cache != nil {
		cache.set(ck, u.String(), expiry)
	}
	return u.String(), nil
}

func GenerateUploadURL(ctx context.Context, client *sdk.Client, bucket, key string, expiry time.Duration, cache *PresignedCache) (string, error) {
	ck := "PUT|" + bucket + "|" + key
	if cache != nil {
		if u, ok := cache.get(ck); ok {
			return u, nil
		}
	}
	u, err := client.PresignedPutObject(ctx, bucket, key, expiry)
	if err != nil {
		return "", err
	}
	if cache != nil {
		cache.set(ck, u.String(), expiry)
	}
	return u.String(), nil
}

func GeneratePreviewURL(ctx context.Context, client *sdk.Client, bucket, key string, expiry time.Duration, cache *PresignedCache) (string, error) {
	return GenerateDownloadURL(ctx, client, bucket, key, expiry, cache)
}
