package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cache-stampede-demo/internal/domain"
	"cache-stampede-demo/internal/handler"
	"cache-stampede-demo/internal/metrics"
	"cache-stampede-demo/internal/port"
)

type mockStore struct{}

func (m *mockStore) FetchProduct(_ context.Context, id int64) (*domain.Product, error) {
	return &domain.Product{
		ID:    id,
		Name:  "Test Product",
		Price: 9.99,
		Stock: 10,
	}, nil
}

func (m *mockStore) SeedProducts(_ context.Context, _ int) error { return nil }
func (m *mockStore) Ping(_ context.Context) error               { return nil }

type mockStrategy struct{}

func (m *mockStrategy) Get(ctx context.Context, _ string, fetcher port.Fetcher) ([]byte, error) {
	return fetcher(ctx)
}
func (m *mockStrategy) Close() error { return nil }

func newTestHandler() *handler.ProductHandler {
	store := &mockStore{}
	strategies := map[string]port.CacheStrategy{
		"naive":         &mockStrategy{},
		"mutex":         &mockStrategy{},
		"probabilistic": &mockStrategy{},
		"warmup":        &mockStrategy{},
	}
	collector := metrics.New()
	return handler.NewProductHandler(store, strategies, collector)
}

func TestGetProduct_Success(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/products/1?strategy=naive", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.GetProduct(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var p domain.Product
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p.Name != "Test Product" {
		t.Errorf("want Test Product, got %s", p.Name)
	}
}

func TestGetProduct_InvalidID(t *testing.T) {
	h := newTestHandler()

	tests := []struct {
		name string
		id   string
	}{
		{"non-numeric", "abc"},
		{"zero", "0"},
		{"negative", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/"+tt.id, nil)
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.GetProduct(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("want 400, got %d", w.Code)
			}
		})
	}
}

func TestGetProduct_UnknownStrategy(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/products/1?strategy=unknown", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.GetProduct(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

type cacheHitStrategy struct{}

func (c *cacheHitStrategy) Get(_ context.Context, _ string, _ port.Fetcher) ([]byte, error) {
	return []byte(`{"id":1,"name":"cached"}`), nil
}
func (c *cacheHitStrategy) Close() error { return nil }

type errorStrategy struct{}

func (e *errorStrategy) Get(_ context.Context, _ string, _ port.Fetcher) ([]byte, error) {
	return nil, errors.New("strategy error")
}
func (e *errorStrategy) Close() error { return nil }

func TestGetProduct_StrategyError(t *testing.T) {
	store := &mockStore{}
	strategies := map[string]port.CacheStrategy{
		"naive": &errorStrategy{},
	}
	collector := metrics.New()
	h := handler.NewProductHandler(store, strategies, collector)

	req := httptest.NewRequest(http.MethodGet, "/products/1?strategy=naive", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.GetProduct(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestGetProduct_CacheHitRecorded(t *testing.T) {
	store := &mockStore{}
	strategies := map[string]port.CacheStrategy{
		"naive": &cacheHitStrategy{},
	}
	collector := metrics.New()
	h := handler.NewProductHandler(store, strategies, collector)

	req := httptest.NewRequest(http.MethodGet, "/products/1?strategy=naive", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.GetProduct(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	snap := collector.Snapshot()
	if snap["naive"].CacheHits != 1 {
		t.Errorf("cache hit should be recorded, got %d", snap["naive"].CacheHits)
	}
}

func TestGetProduct_AllStrategies(t *testing.T) {
	h := newTestHandler()
	for _, s := range []string{"naive", "mutex", "probabilistic", "warmup"} {
		t.Run(s, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/1?strategy="+s, nil)
			req.SetPathValue("id", "1")
			w := httptest.NewRecorder()
			h.GetProduct(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("strategy %s: want 200, got %d", s, w.Code)
			}
		})
	}
}

func TestGetProduct_DefaultStrategy(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.GetProduct(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (defaults to naive)", w.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := handler.SecurityHeaders(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"Referrer-Policy":       "strict-origin-when-cross-origin",
	}
	for key, want := range headers {
		got := w.Header().Get(key)
		if got != want {
			t.Errorf("%s: want %q, got %q", key, want, got)
		}
	}
}

func TestAccessLog(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := handler.AccessLog(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestRequestTimeout(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	h := handler.RequestTimeout(50 * time.Millisecond)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 on timeout, got %d", w.Code)
	}
}
