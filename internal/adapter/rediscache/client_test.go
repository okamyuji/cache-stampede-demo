package rediscache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"cache-stampede-demo/internal/adapter/rediscache"
)

func setupMiniRedis(t *testing.T) (*miniredis.Miniredis, *rediscache.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := rediscache.New(mr.Addr())
	t.Cleanup(func() { client.Close() })
	return mr, client
}

func TestClient_SetAndGet(t *testing.T) {
	_, client := setupMiniRedis(t)
	ctx := context.Background()

	if err := client.Set(ctx, "k1", []byte("v1"), 10*time.Second); err != nil {
		t.Fatalf("set: %v", err)
	}

	got, err := client.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("want v1, got %s", got)
	}
}

func TestClient_GetMiss(t *testing.T) {
	_, client := setupMiniRedis(t)
	ctx := context.Background()

	got, err := client.Get(ctx, "missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

func TestClient_Delete(t *testing.T) {
	_, client := setupMiniRedis(t)
	ctx := context.Background()

	client.Set(ctx, "k1", []byte("v1"), 10*time.Second)
	if err := client.Delete(ctx, "k1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, _ := client.Get(ctx, "k1")
	if got != nil {
		t.Error("key should be deleted")
	}
}

func TestClient_SetNX(t *testing.T) {
	_, client := setupMiniRedis(t)
	ctx := context.Background()

	ok, err := client.SetNX(ctx, "k1", []byte("v1"), 10*time.Second)
	if err != nil {
		t.Fatalf("setnx: %v", err)
	}
	if !ok {
		t.Error("first SetNX should succeed")
	}

	ok, err = client.SetNX(ctx, "k1", []byte("v2"), 10*time.Second)
	if err != nil {
		t.Fatalf("setnx2: %v", err)
	}
	if ok {
		t.Error("second SetNX should fail (key exists)")
	}
}

func TestClient_TTL(t *testing.T) {
	mr, client := setupMiniRedis(t)
	ctx := context.Background()

	client.Set(ctx, "k1", []byte("v1"), 30*time.Second)
	ttl, err := client.TTL(ctx, "k1")
	if err != nil {
		t.Fatalf("ttl: %v", err)
	}
	_ = mr
	if ttl <= 0 {
		t.Errorf("want positive TTL, got %v", ttl)
	}
}

func TestClient_Ping(t *testing.T) {
	_, client := setupMiniRedis(t)
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestClient_FlushAll(t *testing.T) {
	_, client := setupMiniRedis(t)
	ctx := context.Background()

	client.Set(ctx, "k1", []byte("v1"), 10*time.Second)
	if err := client.FlushAll(ctx); err != nil {
		t.Fatalf("flushall: %v", err)
	}

	got, _ := client.Get(ctx, "k1")
	if got != nil {
		t.Error("key should be flushed")
	}
}
