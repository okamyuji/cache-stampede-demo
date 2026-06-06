package strategy

import (
	"context"
	"time"

	"cache-stampede-demo/internal/port"
)

// Naive 素朴なキャッシュ戦略。ミス時に全リクエストがDBへ直行する。
type Naive struct {
	backend port.CacheBackend
	ttl     time.Duration
}

// NewNaive Naive戦略を生成する
func NewNaive(backend port.CacheBackend, ttl time.Duration) *Naive {
	return &Naive{backend: backend, ttl: ttl}
}

func (n *Naive) Get(ctx context.Context, key string, fetcher port.Fetcher) ([]byte, error) {
	cached, err := n.backend.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return cached, nil
	}

	val, err := fetcher(ctx)
	if err != nil {
		return nil, err
	}

	if setErr := n.backend.Set(ctx, key, val, n.ttl); setErr != nil {
		return nil, setErr
	}
	return val, nil
}

func (n *Naive) Close() error { return nil }
