package metrics_test

import (
	"sync"
	"testing"
	"time"

	"cache-stampede-demo/internal/metrics"
)

func TestCollector_RecordAndSnapshot(t *testing.T) {
	c := metrics.New()
	c.RecordCacheHit("naive")
	c.RecordCacheHit("naive")
	c.RecordCacheMiss("naive")
	c.RecordDBQuery("naive", 10*time.Millisecond)
	c.RecordLatency("naive", 15*time.Millisecond)

	snap := c.Snapshot()
	m, ok := snap["naive"]
	if !ok {
		t.Fatal("naive strategy not found in snapshot")
	}
	if m.CacheHits != 2 {
		t.Errorf("want 2 hits, got %d", m.CacheHits)
	}
	if m.CacheMisses != 1 {
		t.Errorf("want 1 miss, got %d", m.CacheMisses)
	}
	if m.DBQueries != 1 {
		t.Errorf("want 1 query, got %d", m.DBQueries)
	}
}

func TestCollector_Reset(t *testing.T) {
	c := metrics.New()
	c.RecordCacheHit("naive")
	c.Reset()

	snap := c.Snapshot()
	if len(snap) != 0 {
		t.Errorf("want empty after reset, got %d strategies", len(snap))
	}
}

func TestCollector_ConcurrentSafety(t *testing.T) {
	c := metrics.New()
	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() {
			c.RecordCacheHit("naive")
			c.RecordCacheMiss("mutex")
			c.RecordDBQuery("probabilistic", time.Millisecond)
			c.RecordLatency("warmup", time.Millisecond)
		})
	}
	wg.Wait()

	snap := c.Snapshot()
	if snap["naive"].CacheHits != 100 {
		t.Errorf("want 100 hits, got %d", snap["naive"].CacheHits)
	}
}

func TestCollector_MultipleStrategies(t *testing.T) {
	c := metrics.New()
	strategies := []string{"naive", "mutex", "probabilistic", "warmup"}
	for _, s := range strategies {
		c.RecordCacheHit(s)
		c.RecordLatency(s, time.Millisecond)
	}

	snap := c.Snapshot()
	if len(snap) != 4 {
		t.Errorf("want 4 strategies, got %d", len(snap))
	}
	for _, s := range strategies {
		if snap[s].CacheHits != 1 {
			t.Errorf("strategy %s: want 1 hit, got %d", s, snap[s].CacheHits)
		}
	}
}

func TestCollector_Percentile(t *testing.T) {
	c := metrics.New()
	for i := range 100 {
		c.RecordLatency("test", time.Duration(i+1)*time.Millisecond)
	}

	snap := c.Snapshot()
	m := snap["test"]
	if m.P99LatMs < 99 || m.P99LatMs > 101 {
		t.Errorf("p99 should be around 99ms, got %.2f", m.P99LatMs)
	}
}
