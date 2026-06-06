package port

import "context"

// Fetcher キャッシュミス時にオリジンから値を取得するコールバック
type Fetcher func(ctx context.Context) ([]byte, error)

// CacheStrategy キャッシュ戦略の共通インターフェース
type CacheStrategy interface {
	Get(ctx context.Context, key string, fetcher Fetcher) ([]byte, error)
	Close() error
}
