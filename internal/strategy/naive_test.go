package strategy_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cache-stampede-demo/internal/strategy"
)

func TestNaive_CacheHit(t *testing.T) {
	backend := newMockBackend()
	backend.store["key1"] = []byte("cached")
	s := strategy.NewNaive(backend, 10*time.Second)

	var count atomic.Int64
	got, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("fresh")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "cached" {
		t.Errorf("want cached, got %s", got)
	}
	if count.Load() != 0 {
		t.Errorf("fetcher should not be called on cache hit, called %d times", count.Load())
	}
}

func TestNaive_CacheMiss(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewNaive(backend, 10*time.Second)

	var count atomic.Int64
	got, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("fresh")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "fresh" {
		t.Errorf("want fresh, got %s", got)
	}
	if count.Load() != 1 {
		t.Errorf("fetcher should be called once, called %d times", count.Load())
	}
}

func TestNaive_Stampede(t *testing.T) {
	backend := newMockBackend()
	backend.getNil = true
	s := strategy.NewNaive(backend, 10*time.Second)

	var count atomic.Int64
	fetcher := countingFetcher(&count, []byte("data"))

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			_, _ = s.Get(context.Background(), "key1", fetcher)
		})
	}
	wg.Wait()

	if count.Load() != 100 {
		t.Errorf("naive should call fetcher for every miss, got %d calls", count.Load())
	}
}

func TestNaive_SetError(t *testing.T) {
	backend := newMockBackend()
	backend.setErr = errSimulated
	s := strategy.NewNaive(backend, 10*time.Second)

	var count atomic.Int64
	_, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("data")))
	if err == nil {
		t.Fatal("expected error on set failure")
	}
}

func TestNaive_BackendGetError(t *testing.T) {
	backend := &errorBackend{getErr: errSimulated}
	s := strategy.NewNaive(backend, 10*time.Second)

	var count atomic.Int64
	_, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("data")))
	if err == nil {
		t.Fatal("expected backend get error to propagate")
	}
}

func TestNaive_Close(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewNaive(backend, 10*time.Second)
	if err := s.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestNaive_FetcherError(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewNaive(backend, 10*time.Second)

	_, err := s.Get(context.Background(), "key1", func(_ context.Context) ([]byte, error) {
		return nil, errSimulated
	})
	if err == nil {
		t.Fatal("expected fetcher error to propagate")
	}
}
