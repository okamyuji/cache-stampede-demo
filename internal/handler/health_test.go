package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"cache-stampede-demo/internal/handler"
)

type mockRedisClient struct {
	pingErr error
}

func (m *mockRedisClient) Ping(_ context.Context) error { return m.pingErr }

type mockStoreWithPing struct {
	mockStore
	pingErr error
}

func (m *mockStoreWithPing) Ping(_ context.Context) error { return m.pingErr }

func TestHealth_OK(t *testing.T) {
	h := handler.NewHealthHandler(&mockStoreWithPing{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestReady_AllHealthy(t *testing.T) {
	store := &mockStoreWithPing{}
	redis := &mockRedisClient{}
	h := handler.NewHealthHandler(store, redis)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	h.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestReady_MySQLDown(t *testing.T) {
	store := &mockStoreWithPing{pingErr: errors.New("mysql down")}
	redis := &mockRedisClient{}
	h := handler.NewHealthHandler(store, redis)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	h.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
}

func TestReady_RedisDown(t *testing.T) {
	store := &mockStoreWithPing{}
	redis := &mockRedisClient{pingErr: errors.New("redis down")}
	h := handler.NewHealthHandler(store, redis)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	h.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
}
