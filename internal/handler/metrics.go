package handler

import (
	"net/http"

	"cache-stampede-demo/internal/port"
)

// MetricsHandler メトリクスAPIハンドラ
type MetricsHandler struct {
	collector port.MetricsCollector
}

// NewMetricsHandler MetricsHandlerを生成する
func NewMetricsHandler(collector port.MetricsCollector) *MetricsHandler {
	return &MetricsHandler{collector: collector}
}

// Get 全戦略のメトリクスを返す
func (h *MetricsHandler) Get(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.collector.Snapshot())
}

// Reset メトリクスをリセットする(メソッド制約はmuxのルーティングパターンで担保)
func (h *MetricsHandler) Reset(w http.ResponseWriter, _ *http.Request) {
	h.collector.Reset()
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}
