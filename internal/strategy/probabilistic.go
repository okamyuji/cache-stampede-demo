package strategy

import (
	"context"
	"encoding/json"
	"math"
	"math/rand/v2"
	"time"

	"cache-stampede-demo/internal/port"
)

// Probabilistic XFetch確率的早期再計算。TTLの残りが少ないほど高確率で再計算を発火する。
type Probabilistic struct {
	backend port.CacheBackend
	ttl     time.Duration
	beta    float64
}

type xfetchEntry struct {
	Value     []byte    `json:"v"`
	FetchedAt time.Time `json:"t"`
	Delta     float64   `json:"d"`
}

// NewProbabilistic XFetch戦略を生成する。beta再計算の積極度(通常1.0)。
func NewProbabilistic(backend port.CacheBackend, ttl time.Duration, beta float64) *Probabilistic {
	return &Probabilistic{backend: backend, ttl: ttl, beta: beta}
}

func (p *Probabilistic) Get(ctx context.Context, key string, fetcher port.Fetcher) ([]byte, error) {
	cached, err := p.backend.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if cached != nil {
		var entry xfetchEntry
		if err := json.Unmarshal(cached, &entry); err == nil {
			remaining, ttlErr := p.backend.TTL(ctx, key)
			if ttlErr != nil {
				return entry.Value, nil
			}
			if !p.shouldRecompute(entry.Delta, remaining) {
				return entry.Value, nil
			}
		}
	}

	start := time.Now()
	val, err := fetcher(ctx)
	if err != nil {
		return nil, err
	}
	delta := time.Since(start).Seconds()

	entry := xfetchEntry{
		Value:     val,
		FetchedAt: time.Now(),
		Delta:     delta,
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	if setErr := p.backend.Set(ctx, key, encoded, p.ttl); setErr != nil {
		return nil, setErr
	}
	return val, nil
}

// shouldRecompute XFetchアルゴリズムによる再計算判定。
// rand.Float64 crypto/randでなくセキュリティ用途でない確率計算のため許容する。
func (p *Probabilistic) shouldRecompute(delta float64, remaining time.Duration) bool {
	if remaining <= 0 {
		return true
	}
	threshold := delta * p.beta * math.Log(rand.Float64()) // #nosec G404 -- 暗号用途でない確率計算
	return -threshold >= remaining.Seconds()
}

func (p *Probabilistic) Close() error { return nil }
