package port

import "time"

// MetricsCollector 戦略別メトリクス収集インターフェース
type MetricsCollector interface {
	RecordCacheHit(strategy string)
	RecordCacheMiss(strategy string)
	RecordDBQuery(strategy string, duration time.Duration)
	RecordLatency(strategy string, duration time.Duration)
	Snapshot() map[string]*StrategyMetrics
	Reset()
}

// StrategyMetrics 1つの戦略のメトリクス
type StrategyMetrics struct {
	CacheHits   int64   `json:"cache_hits"`
	CacheMisses int64   `json:"cache_misses"`
	DBQueries   int64   `json:"db_queries"`
	DBTimeMs    float64 `json:"db_time_ms"`
	AvgLatMs    float64 `json:"avg_latency_ms"`
	P99LatMs    float64 `json:"p99_latency_ms"`
}
