package minio

import (
	"context"
	"fmt"
	"sync"

	"github.com/minio/minio-go/v7/pkg/credentials"

	sdk "github.com/minio/minio-go/v7"
)

type ClientManager struct {
	clients sync.Map // key: endpoint|accessKey|secure
}

func NewClientManager() *ClientManager { return &ClientManager{} }

func clientKey(endpoint, accessKey string, secure bool) string {
	if secure {
		return endpoint + "|" + accessKey + "|1"
	}
	return endpoint + "|" + accessKey + "|0"
}

func (m *ClientManager) GetClient(ctx context.Context, endpoint, accessKey, secretKey string, secure bool) (*sdk.Client, error) {
	key := clientKey(endpoint, accessKey, secure)
	if v, ok := m.clients.Load(key); ok {
		if c, ok2 := v.(*sdk.Client); ok2 {
			return c, nil
		}
	}
	c, err := sdk.New(endpoint, &sdk.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: secure})
	if err != nil {
		return nil, fmt.Errorf("new minio client: %w", err)
	}
	m.clients.Store(key, c)
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
