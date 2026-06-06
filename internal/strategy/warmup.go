package strategy

import (
	"context"
	"log/slog"
	"maps"
	"sync"
	"time"

	"cache-stampede-demo/internal/port"
)

// Warmup スケジュール事前ウォーム。TTLの80%経過でバックグラウンド更新する。
type Warmup struct {
	backend   port.CacheBackend
	ttl       time.Duration
	interval  time.Duration
	mu        sync.Mutex
	entries   map[string]port.Fetcher
	done      chan struct{}
	closeOnce sync.Once
}

// NewWarmup Warmup戦略を生成する
func NewWarmup(backend port.CacheBackend, ttl time.Duration) *Warmup {
	w := &Warmup{
		backend:  backend,
		ttl:      ttl,
		interval: time.Duration(float64(ttl) * 0.8),
		entries:  make(map[string]port.Fetcher),
		done:     make(chan struct{}),
	}
	go w.loop()
	return w
}

func (w *Warmup) Get(ctx context.Context, key string, fetcher port.Fetcher) ([]byte, error) {
	cached, err := w.backend.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		w.register(key, fetcher)
		return cached, nil
	}

	val, err := fetcher(ctx)
	if err != nil {
		return nil, err
	}
	if setErr := w.backend.Set(ctx, key, val, w.ttl); setErr != nil {
		return nil, setErr
	}
	w.register(key, fetcher)
	return val, nil
}

func (w *Warmup) register(key string, fetcher port.Fetcher) {
	w.mu.Lock()
	w.entries[key] = fetcher
	w.mu.Unlock()
}

func (w *Warmup) loop() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			w.warmAll()
		}
	}
}

func (w *Warmup) warmAll() {
	w.mu.Lock()
	snapshot := make(map[string]port.Fetcher, len(w.entries))
	maps.Copy(snapshot, w.entries)
	w.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for key, fetcher := range snapshot {
		val, err := fetcher(ctx)
		if err != nil {
			slog.Warn("warmup fetch failed", "key", key, "error", err)
			continue
		}
		if err := w.backend.Set(ctx, key, val, w.ttl); err != nil {
			slog.Warn("warmup set failed", "key", key, "error", err)
		}
	}
}

func (w *Warmup) Close() error {
	w.closeOnce.Do(func() { close(w.done) })
	return nil
}
