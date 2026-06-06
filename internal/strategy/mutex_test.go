package strategy_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cache-stampede-demo/internal/strategy"
)

func TestMutex_CacheHit(t *testing.T) {
	backend := newMockBackend()
	backend.store["key1"] = []byte("cached")
	s := strategy.NewMutex(backend, 10*time.Second)

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

func TestMutex_CoalescesConcurrent(t *testing.T) {
	backend := newMockBackend()
	backend.getNil = true
	s := strategy.NewMutex(backend, 10*time.Second)

	var count atomic.Int64
	fetcher := func(_ context.Context) ([]byte, error) {
		count.Add(1)
		time.Sleep(50 * time.Millisecond)
		return []byte("data"), nil
	}

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			_, _ = s.Get(context.Background(), "key1", fetcher)
		})
	}
	wg.Wait()

	calls := count.Load()
	if calls >= 100 {
		t.Errorf("mutex should coalesce concurrent fetches, got %d calls", calls)
	}
	t.Logf("mutex coalesced 100 requests into %d fetcher calls", calls)
}

func TestMutex_Close(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewMutex(backend, 10*time.Second)
	if err := s.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestMutex_FetcherError(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewMutex(backend, 10*time.Second)

	_, err := s.Get(context.Background(), "key1", func(_ context.Context) ([]byte, error) {
		return nil, errSimulated
	})
	if err == nil {
		t.Fatal("expected fetcher error to propagate")
	}
}
