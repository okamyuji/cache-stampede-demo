package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"cache-stampede-demo/internal/adapter/mysqlstore"
	"cache-stampede-demo/internal/adapter/rediscache"
	"cache-stampede-demo/internal/config"
	"cache-stampede-demo/internal/handler"
	"cache-stampede-demo/internal/metrics"
	"cache-stampede-demo/internal/port"
	"cache-stampede-demo/internal/strategy"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	slog.Info("starting stampede-server", "version", version, "addr", cfg.Addr)

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	redis := rediscache.New(cfg.RedisAddr)
	defer func() { _ = redis.Close() }()

	store := mysqlstore.New(db, cfg.QueryDelay)
	collector := metrics.New()

	strategies := map[string]port.CacheStrategy{
		"naive":         strategy.NewNaive(redis, cfg.CacheTTL),
		"mutex":         strategy.NewMutex(redis, cfg.CacheTTL),
		"probabilistic": strategy.NewProbabilistic(redis, cfg.CacheTTL, cfg.XFetchBeta),
		"warmup":        strategy.NewWarmup(redis, cfg.CacheTTL),
	}
	defer func() {
		for _, s := range strategies {
			_ = s.Close()
		}
	}()

	productHandler := handler.NewProductHandler(store, strategies, collector)
	healthHandler := handler.NewHealthHandler(store, redis)
	metricsHandler := handler.NewMetricsHandler(collector)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{id}", productHandler.GetProduct)
	mux.HandleFunc("GET /health", healthHandler.Health)
	mux.HandleFunc("GET /ready", healthHandler.Ready)
	mux.HandleFunc("GET /metrics", metricsHandler.Get)
	mux.HandleFunc("POST /metrics/reset", handler.RequireAdminKey(cfg.AdminKey, metricsHandler.Reset))
	mux.HandleFunc("POST /cache/clear", handler.RequireAdminKey(cfg.AdminKey, func(w http.ResponseWriter, r *http.Request) {
		if err := redis.FlushAll(r.Context()); err != nil {
			handler.WriteErrorPublic(w, http.StatusInternalServerError, "cache clear failed")
			return
		}
		productHandler.ClearCache(w, r)
	}))

	var h http.Handler = mux
	h = handler.AccessLog(h)
	h = handler.RequestTimeout(30 * time.Second)(h)
	h = handler.SecurityHeaders(h)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig)
	case err := <-errCh:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
