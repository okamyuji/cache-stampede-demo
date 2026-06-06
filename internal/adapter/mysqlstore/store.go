package mysqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cache-stampede-demo/internal/domain"
	"cache-stampede-demo/internal/port"
)

var _ port.ProductStore = (*Store)(nil)

// Store MySQL ProductStore実装
type Store struct {
	db    *sql.DB
	delay time.Duration
}

// New MySQLストアを生成する。delay重いクエリシミュレーション用。
func New(db *sql.DB, delay time.Duration) *Store {
	return &Store{db: db, delay: delay}
}

func (s *Store) FetchProduct(ctx context.Context, id int64) (*domain.Product, error) {
	if s.db == nil {
		return nil, fmt.Errorf("fetch product id=%d: database not initialized", id)
	}
	if s.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.delay):
		}
	}

	p := &domain.Product{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, description, price, stock, created_at, updated_at FROM products WHERE id = ?",
		id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("fetch product id=%d: %w", id, err)
	}
	return p, nil
}

func (s *Store) SeedProducts(ctx context.Context, count int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx,
		"INSERT IGNORE INTO products (id, name, description, price, stock) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for i := 1; i <= count; i++ {
		_, err = stmt.ExecContext(ctx, i,
			fmt.Sprintf("Product-%d", i),
			fmt.Sprintf("Description for product %d", i),
			float64(i)*1.99,
			i*10,
		)
		if err != nil {
			return fmt.Errorf("insert product %d: %w", i, err)
		}
	}
	return tx.Commit()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
