package strategy_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"cache-stampede-demo/internal/port"
)

var errSimulated = errors.New("simulated error")

// mockBackend port.CacheBackendのテスト用モック
type mockBackend struct {
	mu      sync.Mutex
	store   map[string][]byte
	ttls    map[string]time.Duration
	setErr  error
	getNil  bool
	setNXOK bool
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		store:   make(map[string][]byte),
		ttls:    make(map[string]time.Duration),
		setNXOK: true,
	}
}

func (m *mockBackend) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getNil {
		return nil, nil
	}
	return m.store[key], nil
}

func (m *mockBackend) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setErr != nil {
		return m.setErr
	}
	m.store[key] = value
	m.ttls[key] = ttl
	return nil
}

func (m *mockBackend) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
	return nil
}

func (m *mockBackend) SetNX(_ context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.store[key]; exists {
		return false, nil
	}
	if m.setNXOK {
		m.store[key] = value
		m.ttls[key] = ttl
	}
	return m.setNXOK, nil
}

func (m *mockBackend) TTL(_ context.Context, key string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ttl, ok := m.ttls[key]
	if !ok {
		return -2, nil
	}
	return ttl, nil
}

// errorBackend Getでエラーを返すモック
type errorBackend struct {
	getErr error
}

func (e *errorBackend) Get(_ context.Context, _ string) ([]byte, error) { return nil, e.getErr }
func (e *errorBackend) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (e *errorBackend) Delete(_ context.Context, _ string) error               { return nil }
func (e *errorBackend) SetNX(_ context.Context, _ string, _ []byte, _ time.Duration) (bool, error) {
	return true, nil
}
func (e *errorBackend) TTL(_ context.Context, _ string) (time.Duration, error) {
	return 0, nil
}

// countingFetcher 呼び出し回数を記録するFetcher
func countingFetcher(count *atomic.Int64, data []byte) port.Fetcher {
	return func(_ context.Context) ([]byte, error) {
		count.Add(1)
		return data, nil
	}
}
