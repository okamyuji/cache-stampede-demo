package port

import (
	"context"
	"time"
)

// CacheBackend Redis等のキャッシュバックエンドを抽象化する
type CacheBackend interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
}
