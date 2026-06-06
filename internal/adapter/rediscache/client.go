package rediscache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"cache-stampede-demo/internal/port"
)

var _ port.CacheBackend = (*Client)(nil)

// Client Redis CacheBackend実装
type Client struct {
	rdb *redis.Client
}

// New Redisクライアントを生成する
func New(addr string) *Client {
	return &Client{
		rdb: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	return val, err
}

func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

func (c *Client) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, value, ttl).Result()
}

func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.rdb.TTL(ctx, key).Result()
}

// Ping Redisへの接続確認
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// FlushAll 全キャッシュ削除
func (c *Client) FlushAll(ctx context.Context) error {
	return c.rdb.FlushAll(ctx).Err()
}

// Close 接続を閉じる
func (c *Client) Close() error {
	return c.rdb.Close()
}
