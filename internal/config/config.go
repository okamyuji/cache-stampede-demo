package config

import (
	"os"
	"strconv"
	"time"
)

// Config アプリケーション設定
type Config struct {
	Addr       string
	MySQLDSN   string
	RedisAddr  string
	CacheTTL   time.Duration
	QueryDelay time.Duration
	XFetchBeta float64
	AdminKey   string
}

// Load 環境変数から設定を読み込む
func Load() Config {
	return Config{
		Addr:       envOr("ADDR", ":8080"),
		MySQLDSN:   envOr("MYSQL_DSN", "app:app@tcp(127.0.0.1:3307)/stampede?parseTime=true"),
		RedisAddr:  envOr("REDIS_ADDR", "127.0.0.1:6379"),
		CacheTTL:   envDuration("CACHE_TTL", 10*time.Second),
		QueryDelay: envDuration("QUERY_DELAY", 50*time.Millisecond),
		XFetchBeta: envFloat("XFETCH_BETA", 1.0),
		AdminKey:   os.Getenv("ADMIN_KEY"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
	}
	return fallback
}
