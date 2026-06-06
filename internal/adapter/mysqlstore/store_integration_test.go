package mysqlstore_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"cache-stampede-demo/internal/adapter/mysqlstore"
)

func dockerAvailable() bool {
	return exec.Command("docker", "info").Run() == nil
}

func setupMySQL(t *testing.T) *sql.DB {
	t.Helper()
	if !dockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	var dsn string
	if env := os.Getenv("INTEGRATION_MYSQL_DSN"); env != "" {
		dsn = env
	} else {
		container, err := tcmysql.Run(ctx, "mysql:8.0",
			tcmysql.WithDatabase("stampede"),
			tcmysql.WithUsername("app"),
			tcmysql.WithPassword("app"),
		)
		if err != nil {
			t.Fatalf("start mysql container: %v", err)
		}
		t.Cleanup(func() { testcontainers.CleanupContainer(t, container) })

		host, _ := container.Host(ctx)
		port, _ := container.MappedPort(ctx, "3306")
		dsn = fmt.Sprintf("app:app@tcp(%s:%s)/stampede?parseTime=true", host, port.Port())
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	deadline := time.Now().Add(60 * time.Second)
	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("mysql ping deadline exceeded")
		}
		time.Sleep(500 * time.Millisecond)
	}

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS products (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price DECIMAL(10,2) NOT NULL,
			stock INT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, _ = db.ExecContext(ctx, "TRUNCATE TABLE products")
	return db
}

func TestIntegration_SeedAndFetch(t *testing.T) {
	db := setupMySQL(t)
	store := mysqlstore.New(db, 0)
	ctx := context.Background()

	if err := store.SeedProducts(ctx, 10); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p, err := store.FetchProduct(ctx, 1)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if p.ID != 1 {
		t.Errorf("want id=1, got %d", p.ID)
	}
	if p.Name != "Product-1" {
		t.Errorf("want Product-1, got %s", p.Name)
	}
}

func TestIntegration_FetchNotFound(t *testing.T) {
	db := setupMySQL(t)
	store := mysqlstore.New(db, 0)

	_, err := store.FetchProduct(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for non-existent product")
	}
}

func TestIntegration_Ping(t *testing.T) {
	db := setupMySQL(t)
	store := mysqlstore.New(db, 0)

	if err := store.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestIntegration_FetchWithDelay(t *testing.T) {
	db := setupMySQL(t)
	delay := 50 * time.Millisecond
	store := mysqlstore.New(db, delay)
	ctx := context.Background()

	store.SeedProducts(ctx, 1)

	start := time.Now()
	_, err := store.FetchProduct(ctx, 1)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if elapsed < delay {
		t.Errorf("expected at least %v delay, got %v", delay, elapsed)
	}
}
