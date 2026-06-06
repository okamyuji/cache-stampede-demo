package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cache-stampede-demo/internal/port"
)

// ProductHandler 商品APIハンドラ
type ProductHandler struct {
	store      port.ProductStore
	strategies map[string]port.CacheStrategy
	collector  port.MetricsCollector
}

// NewProductHandler ProductHandlerを生成する
func NewProductHandler(
	store port.ProductStore,
	strategies map[string]port.CacheStrategy,
	collector port.MetricsCollector,
) *ProductHandler {
	return &ProductHandler{
		store:      store,
		strategies: strategies,
		collector:  collector,
	}
}

var validStrategies = map[string]bool{
	"naive":         true,
	"mutex":         true,
	"probabilistic": true,
	"warmup":        true,
}

// GetProduct GET /products/{id}?strategy=naive|mutex|probabilistic|warmup
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		WriteErrorPublic(w, http.StatusBadRequest, "invalid product id")
		return
	}

	strategyName := r.URL.Query().Get("strategy")
	if strategyName == "" {
		strategyName = "naive"
	}
	if !validStrategies[strategyName] {
		WriteErrorPublic(w, http.StatusBadRequest, "unknown strategy")
		return
	}

	strategy, ok := h.strategies[strategyName]
	if !ok {
		WriteErrorPublic(w, http.StatusInternalServerError, "strategy not configured")
		return
	}

	key := fmt.Sprintf("product:%d", id)
	var fetched bool
	fetcher := h.makeFetcher(id, strategyName, &fetched)

	data, err := strategy.Get(r.Context(), key, fetcher)
	if err != nil {
		slog.Error("strategy.Get failed", "strategy", strategyName, "id", id, "error", err)
		WriteErrorPublic(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !fetched {
		h.collector.RecordCacheHit(strategyName)
	}

	h.collector.RecordLatency(strategyName, time.Since(start))

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (h *ProductHandler) makeFetcher(id int64, strategyName string, fetched *bool) port.Fetcher {
	return func(ctx context.Context) ([]byte, error) {
		*fetched = true
		h.collector.RecordCacheMiss(strategyName)

		dbStart := time.Now()
		product, err := h.store.FetchProduct(ctx, id)
		h.collector.RecordDBQuery(strategyName, time.Since(dbStart))
		if err != nil {
			return nil, err
		}
		return json.Marshal(product)
	}
}

// ClearCache POST /cache/clear (メソッド制約はmuxのルーティングパターンで担保)
func (h *ProductHandler) ClearCache(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// WriteErrorPublic JSONエラーレスポンスを返す
func WriteErrorPublic(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
