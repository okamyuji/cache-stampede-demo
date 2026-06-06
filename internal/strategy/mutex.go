package strategy

import (
	"context"
	"time"

	"golang.org/x/sync/singleflight"

	"cache-stampede-demo/internal/port"
)

// Mutex singleflight方式。同一キーの同時リクエストを1つに集約する。
type Mutex struct {
	backend port.CacheBackend
	ttl     time.Duration
	group   singleflight.Group
}

// NewMutex Mutex戦略を生成する
func NewMutex(backend port.CacheBackend, ttl time.Duration) *Mutex {
	return &Mutex{backend: backend, ttl: ttl}
}

func (m *Mutex) Get(ctx context.Context, key string, fetcher port.Fetcher) ([]byte, error) {
	cached, err := m.backend.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return cached, nil
	}

	val, err, _ := m.group.Do(key, func() (any, error) {
		cached2, err2 := m.backend.Get(ctx, key)
		if err2 != nil {
			return nil, err2
		}
		if cached2 != nil {
			return cached2, nil
		}

		fetched, fetchErr := fetcher(ctx)
		if fetchErr != nil {
			return nil, fetchErr
		}
		if setErr := m.backend.Set(ctx, key, fetched, m.ttl); setErr != nil {
			return nil, setErr
		}
		return fetched, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]byte), nil
}

func (m *Mutex) Close() error { return nil }
