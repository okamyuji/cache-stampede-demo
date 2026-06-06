package mysqlstore_test

import (
	"context"
	"testing"
	"time"

	"cache-stampede-demo/internal/adapter/mysqlstore"
)

func TestNew(t *testing.T) {
	store := mysqlstore.New(nil, 0)
	if store == nil {
		t.Fatal("New should return non-nil store")
	}
}

func TestNew_WithDelay(t *testing.T) {
	store := mysqlstore.New(nil, 100*time.Millisecond)
	if store == nil {
		t.Fatal("New should return non-nil store with delay")
	}
}

func TestFetchProduct_NilDB(t *testing.T) {
	store := mysqlstore.New(nil, 0)
	_, err := store.FetchProduct(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error with nil db")
	}
}
