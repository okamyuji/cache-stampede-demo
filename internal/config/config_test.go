package config_test

import (
	"testing"
	"time"

	"cache-stampede-demo/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := config.Load()

	if cfg.Addr != ":8080" {
		t.Errorf("want :8080, got %s", cfg.Addr)
	}
	if cfg.CacheTTL != 10*time.Second {
		t.Errorf("want 10s, got %v", cfg.CacheTTL)
	}
	if cfg.QueryDelay != 50*time.Millisecond {
		t.Errorf("want 50ms, got %v", cfg.QueryDelay)
	}
	if cfg.XFetchBeta != 1.0 {
		t.Errorf("want 1.0, got %f", cfg.XFetchBeta)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("ADDR", ":9090")
	t.Setenv("CACHE_TTL", "30s")
	t.Setenv("QUERY_DELAY", "100ms")
	t.Setenv("XFETCH_BETA", "2.5")

	cfg := config.Load()

	if cfg.Addr != ":9090" {
		t.Errorf("want :9090, got %s", cfg.Addr)
	}
	if cfg.CacheTTL != 30*time.Second {
		t.Errorf("want 30s, got %v", cfg.CacheTTL)
	}
	if cfg.QueryDelay != 100*time.Millisecond {
		t.Errorf("want 100ms, got %v", cfg.QueryDelay)
	}
	if cfg.XFetchBeta != 2.5 {
		t.Errorf("want 2.5, got %f", cfg.XFetchBeta)
	}
}

func TestLoad_InvalidEnvFallback(t *testing.T) {
	t.Setenv("CACHE_TTL", "invalid")
	t.Setenv("XFETCH_BETA", "not-a-number")

	cfg := config.Load()

	if cfg.CacheTTL != 10*time.Second {
		t.Errorf("invalid duration should fall back to default, got %v", cfg.CacheTTL)
	}
	if cfg.XFetchBeta != 1.0 {
		t.Errorf("invalid float should fall back to default, got %f", cfg.XFetchBeta)
	}
}
