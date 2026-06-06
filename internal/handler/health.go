package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"cache-stampede-demo/internal/port"
)

// Pinger 接続確認インターフェース
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler ヘルスチェックとレディネスチェック
type HealthHandler struct {
	store port.ProductStore
	redis Pinger
}

// NewHealthHandler HealthHandlerを生成する
func NewHealthHandler(store port.ProductStore, redis Pinger) *HealthHandler {
	return &HealthHandler{store: store, redis: redis}
}


// Health ヘルスチェック(プロセス生存確認のみ)
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready レディネスチェック(MySQL+Redis接続確認)
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.store.Ping(ctx); err != nil {
		slog.Error("mysql ping failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready", "detail": "mysql unavailable"})
		return
	}
	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			slog.Error("redis ping failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready", "detail": "redis unavailable"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
