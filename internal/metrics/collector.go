package metrics

import (
	"math"
	"sort"
	"sync"
	"time"

	"cache-stampede-demo/internal/port"
)

var _ port.MetricsCollector = (*Collector)(nil)

// Collector スレッドセーフなメトリクス収集
type Collector struct {
	mu   sync.Mutex
	data map[string]*strategyData
}

type strategyData struct {
	cacheHits   int64
	cacheMisses int64
	dbQueries   int64
	dbTimeNs    int64
	latencies   []float64
}

// New コレクタを生成する
func New() *Collector {
	return &Collector{data: make(map[string]*strategyData)}
}

func (c *Collector) get(strategy string) *strategyData {
	d, ok := c.data[strategy]
	if !ok {
		d = &strategyData{}
		c.data[strategy] = d
	}
	return d
}

func (c *Collector) RecordCacheHit(strategy string) {
	c.mu.Lock()
	c.get(strategy).cacheHits++
	c.mu.Unlock()
}

func (c *Collector) RecordCacheMiss(strategy string) {
	c.mu.Lock()
	c.get(strategy).cacheMisses++
	c.mu.Unlock()
}

func (c *Collector) RecordDBQuery(strategy string, duration time.Duration) {
	c.mu.Lock()
	d := c.get(strategy)
	d.dbQueries++
	d.dbTimeNs += duration.Nanoseconds()
	c.mu.Unlock()
}

func (c *Collector) RecordLatency(strategy string, duration time.Duration) {
	c.mu.Lock()
	c.get(strategy).latencies = append(c.get(strategy).latencies, float64(duration.Nanoseconds())/1e6)
	c.mu.Unlock()
}

func (c *Collector) Snapshot() map[string]*port.StrategyMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make(map[string]*port.StrategyMetrics, len(c.data))
	for name, d := range c.data {
		m := &port.StrategyMetrics{
			CacheHits:   d.cacheHits,
			CacheMisses: d.cacheMisses,
			DBQueries:   d.dbQueries,
			DBTimeMs:    float64(d.dbTimeNs) / 1e6,
		}
		if len(d.latencies) > 0 {
			m.AvgLatMs = avg(d.latencies)
			m.P99LatMs = percentile(d.latencies, 99)
		}
		result[name] = m
	}
	return result
}

func (c *Collector) Reset() {
	c.mu.Lock()
	c.data = make(map[string]*strategyData)
	c.mu.Unlock()
}

func avg(vals []float64) float64 {
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return math.Round(sum/float64(len(vals))*100) / 100
}

func percentile(vals []float64, pct float64) float64 {
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	idx := max(int(math.Ceil(pct/100*float64(len(sorted))))-1, 0)
	return math.Round(sorted[idx]*100) / 100
}
