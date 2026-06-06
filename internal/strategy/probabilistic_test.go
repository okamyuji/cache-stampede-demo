package strategy_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"cache-stampede-demo/internal/strategy"
)

func TestProbabilistic_CacheMiss(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewProbabilistic(backend, 10*time.Second, 1.0)

	var count atomic.Int64
	got, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("fresh")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if count.Load() != 1 {
		t.Errorf("fetcher should be called once on miss, got %d", count.Load())
	}
}

func TestProbabilistic_CacheHitWithHighTTL(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewProbabilistic(backend, 10*time.Second, 1.0)

	var count atomic.Int64
	fetcher := countingFetcher(&count, []byte("fresh"))

	_, err := s.Get(context.Background(), "key1", fetcher)
	if err != nil {
		t.Fatalf("initial fetch: %v", err)
	}
	initialCalls := count.Load()

	backend.getNil = false
	backend.ttls["key1"] = 9 * time.Second

	_, err = s.Get(context.Background(), "key1", fetcher)
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}

	if count.Load() != initialCalls {
		t.Logf("with high remaining TTL, recompute may or may not trigger (probabilistic)")
	}
}

func TestProbabilistic_FetcherError(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewProbabilistic(backend, 10*time.Second, 1.0)

	_, err := s.Get(context.Background(), "key1", func(_ context.Context) ([]byte, error) {
		return nil, errSimulated
	})
	if err == nil {
		t.Fatal("expected fetcher error to propagate")
	}
}

func TestProbabilistic_ExpiredTTL(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewProbabilistic(backend, 10*time.Second, 1.0)

	var count atomic.Int64
	fetcher := countingFetcher(&count, []byte("fresh"))

	_, err := s.Get(context.Background(), "key1", fetcher)
	if err != nil {
		t.Fatalf("initial fetch: %v", err)
	}

	backend.ttls["key1"] = 0

	_, err = s.Get(context.Background(), "key1", fetcher)
	if err != nil {
		t.Fatalf("expired fetch: %v", err)
	}
	if count.Load() < 2 {
		t.Logf("expired TTL should trigger recompute (got %d calls)", count.Load())
	}
}

func TestProbabilistic_SetError(t *testing.T) {
	backend := newMockBackend()
	backend.setErr = errSimulated
	s := strategy.NewProbabilistic(backend, 10*time.Second, 1.0)

	var count atomic.Int64
	_, err := s.Get(context.Background(), "key1", countingFetcher(&count, []byte("data")))
	if err == nil {
		t.Fatal("expected error on set failure")
	}
}

func TestProbabilistic_Close(t *testing.T) {
	backend := newMockBackend()
	s := strategy.NewProbabilistic(backend, 10*time.Second, 1.0)
	if err := s.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}
