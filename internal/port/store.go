package port

import (
	"context"

	"cache-stampede-demo/internal/domain"
)

// ProductStore オリジンDBからの商品データ取得を抽象化する
type ProductStore interface {
	FetchProduct(ctx context.Context, id int64) (*domain.Product, error)
	SeedProducts(ctx context.Context, count int) error
	Ping(ctx context.Context) error
}
