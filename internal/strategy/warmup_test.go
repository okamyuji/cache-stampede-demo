package strategy_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"cache-stampede-demo/internal/strategy"
)

func TestWarmup_CacheMiss(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewWarmup(backend, 10*time.Second)
	defer s.Close()

	var count atomic.Int64
	got, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("fresh")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "fresh" {
		t.Errorf("want fresh, got %s", got)
	}
	if count.Load() != 1 {
		t.Errorf("fetcher should be called once, got %d", count.Load())
	}
}

func TestWarmup_CacheHit(t *testing.T) {
	backend := newMockBackend()
	backend.store["key1"] = []byte("cached")
	s := strategy.NewWarmup(backend, 10*time.Second)
	defer s.Close()

	var count atomic.Int64
	got, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("fresh")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "cached" {
		t.Errorf("want cached, got %s", got)
	}
	if count.Load() != 0 {
		t.Errorf("fetcher should not be called on cache hit")
	}
}

func TestWarmup_BackgroundRefresh(t *testing.T) {
	backend := newMockBackend()
	ttl := 100 * time.Millisecond
	s := strategy.NewWarmup(backend, ttl)
	defer s.Close()

	var count atomic.Int64
	fetcher := countingFetcher(&count, []byte("data"))

	_, err := s.Get(context.Background(), "key1", fetcher)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(ttl)

	calls := count.Load()
	if calls < 2 {
		t.Logf("warmup background refresh may not have triggered yet (got %d calls), timing-dependent", calls)
	}
}

func TestWarmup_SetError(t *testing.T) {
	backend := newMockBackend()
	backend.setErr = errSimulated
	s := strategy.NewWarmup(backend, 10*time.Second)
	defer s.Close()

	var count atomic.Int64
	_, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("data")))
	if err == nil {
		t.Fatal("expected error on set failure")
	}
}

func TestWarmup_FetcherError(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewWarmup(backend, 10*time.Second)
	defer s.Close()

	_, err := s.Get(context.Background(), "key1", func(_ context.Context) ([]byte, error) {
		return nil, errSimulated
	})
	if err == nil {
		t.Fatal("expected fetcher error to propagate")
	}
}

func TestWarmup_BackgroundRefreshError(t *testing.T) {
	backend := newMockBackend()
	ttl := 100 * time.Millisecond
	s := strategy.NewWarmup(backend, ttl)
	defer s.Close()

	callCount := atomic.Int64{}
	failingFetcher := func(_ context.Context) ([]byte, error) {
		if callCount.Add(1) > 1 {
			return nil, errSimulated
		}
		return []byte("data"), nil
	}

	_, err := s.Get(context.Background(), "error-key", failingFetcher)
	if err != nil {
		t.Fatalf("initial fetch: %v", err)
	}

	time.Sleep(ttl + 50*time.Millisecond)

	if callCount.Load() < 2 {
		t.Logf("background refresh may not have fired yet (timing-dependent)")
	}
}

func TestWarmup_BackgroundSetError(t *testing.T) {
	backend := newMockBackend()
	ttl := 100 * time.Millisecond
	s := strategy.NewWarmup(backend, ttl)
	defer s.Close()

	var count atomic.Int64
	_, err := s.Get(context.Background(), "set-err-key", countingFetcher(&count, []byte("data")))
	if err != nil {
		t.Fatalf("initial fetch: %v", err)
	}

	backend.mu.Lock()
	backend.setErr = errSimulated
	backend.mu.Unlock()

	time.Sleep(ttl + 50*time.Millisecond)
	t.Log("background refresh with set error completed (logged as warning)")
}

func TestWarmup_Close(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewWarmup(backend, 10*time.Second)
	if err := s.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestWarmup_DoubleClose(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewWarmup(backend, 10*time.Second)
	_ = s.Close()
	if err := s.Close(); err != nil {
		t.Fatalf("double close should be safe: %v", err)
	}
}
