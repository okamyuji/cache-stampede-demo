package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cache-stampede-demo/internal/handler"
	"cache-stampede-demo/internal/metrics"
	"cache-stampede-demo/internal/port"
)

func TestMetricsGet(t *testing.T) {
	c := metrics.New()
	c.RecordCacheHit("naive")
	h := handler.NewMetricsHandler(c)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var snap map[string]*port.StrategyMetrics
	if err := json.NewDecoder(w.Body).Decode(&snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if snap["naive"].CacheHits != 1 {
		t.Errorf("want 1 hit, got %d", snap["naive"].CacheHits)
	}
}

func TestMetricsReset(t *testing.T) {
	c := metrics.New()
	c.RecordCacheHit("naive")
	h := handler.NewMetricsHandler(c)

	req := httptest.NewRequest(http.MethodPost, "/metrics/reset", nil)
	w := httptest.NewRecorder()
	h.Reset(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	snap := c.Snapshot()
	if len(snap) != 0 {
		t.Errorf("want empty after reset, got %d", len(snap))
	}
}


func TestRequireAdminKey_Enforced(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := handler.RequireAdminKey("secret", inner)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without key, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/test", nil)
	req2.Header.Set("X-Admin-Key", "secret")
	w2 := httptest.NewRecorder()
	h(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("want 200 with correct key, got %d", w2.Code)
	}
}

func TestRequireAdminKey_EmptyKeySkips(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := handler.RequireAdminKey("", inner)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 when key is empty (dev mode), got %d", w.Code)
	}
}

func TestClearCache_POST(t *testing.T) {
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/cache/clear", nil)
	w := httptest.NewRecorder()
	h.ClearCache(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}
