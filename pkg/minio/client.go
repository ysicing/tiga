package minio

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/minio/minio-go/v7/pkg/credentials"

	sdk "github.com/minio/minio-go/v7"
)

type ClientManager struct {
	clients *expirable.LRU[string, *sdk.Client]
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: expirable.NewLRU[string, *sdk.Client](
			100,              // Maximum 100 MinIO clients
			nil,              // No eviction callback needed
			30*time.Minute,   // 30 minutes TTL
		),
	}
}

func clientKey(endpoint, accessKey string, secure bool) string {
	if secure {
		return endpoint + "|" + accessKey + "|1"
	}
	return endpoint + "|" + accessKey + "|0"
}

func (m *ClientManager) GetClient(ctx context.Context, endpoint, accessKey, secretKey string, secure bool) (*sdk.Client, error) {
	key := clientKey(endpoint, accessKey, secure)

	// Check cache with automatic TTL handling
	if c, ok := m.clients.Get(key); ok {
		// Verify connection is still healthy
		if err := m.HealthCheck(ctx, c); err == nil {
			return c, nil
		}
		// Remove unhealthy client from cache
		m.clients.Remove(key)
	}

	// Create new client
	c, err := sdk.New(endpoint, &sdk.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("new minio client: %w", err)
	}

	// Store in cache
	m.clients.Add(key, c)
	return c, nil
}

func (m *ClientManager) HealthCheck(ctx context.Context, c *sdk.Client) error {
	_, err := c.ListBuckets(ctx)
	return err
}

// GetServerInfo fetches basic server info (bucket count)
func (m *ClientManager) GetServerInfo(ctx context.Context, c *sdk.Client) (map[string]interface{}, error) {
	buckets, err := c.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	info := map[string]interface{}{
		"bucket_count": len(buckets),
	}
	return info, nil
}
