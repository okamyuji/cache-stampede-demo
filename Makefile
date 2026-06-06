SHELL := /bin/bash
GO ?= go
GOFLAGS := -mod=readonly
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
PKG := ./...
DB_HOST ?= 127.0.0.1
DB_PORT ?= 3307
DB_USER ?= app
DB_PASS ?= app
DB_NAME ?= stampede
MIGRATE_DSN ?= mysql://$(DB_USER):$(DB_PASS)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?parseTime=true&multiStatements=true

.PHONY: help build test test-integration lint quality fmt run clean \
	compose-up compose-down migrate-up migrate-down seed loadtest \
	precommit-install

help: ## ヘルプ表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## bin/stampede-serverをビルド
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/stampede-server ./cmd/server

test: ## ユニットテスト実行
	$(GO) test --count=1 --shuffle=on -race -coverprofile=coverage.out -covermode=atomic $(PKG)
	@$(GO) tool cover -func=coverage.out | tail -n 1

test-integration: ## 統合テスト実行(testcontainers使用)
	$(GO) test --count=1 --shuffle=on -race -tags=integration -timeout=10m $(PKG)

lint: ## 全lint実行
	$(GO) vet $(PKG)
	staticcheck $(PKG)
	golangci-lint run --timeout 5m $(PKG)

quality: ## quality-gate.sh実行(pre-commitと同一)
	./scripts/quality-gate.sh

fmt: ## gofmt実行
	$(GO) fmt $(PKG)

run: build ## サーバー起動
	./bin/stampede-server

clean: ## ビルド成果物削除
	rm -rf bin coverage.out

compose-up: ## Redis+MySQL起動
	docker compose up -d --wait

compose-down: ## Redis+MySQL停止
	docker compose down

migrate-up: ## マイグレーション適用
	@for f in migrations/*.up.sql; do \
		echo "==> applying $$f"; \
		mysql -h $(DB_HOST) -P $(DB_PORT) -u $(DB_USER) -p$(DB_PASS) $(DB_NAME) < "$$f"; \
	done

migrate-down: ## マイグレーション巻き戻し
	@for f in $$(ls -r migrations/*.down.sql); do \
		echo "==> reverting $$f"; \
		mysql -h $(DB_HOST) -P $(DB_PORT) -u $(DB_USER) -p$(DB_PASS) $(DB_NAME) < "$$f"; \
	done

seed: ## テストデータ投入
	@echo "==> seeding 100 products"
	@mysql -h $(DB_HOST) -P $(DB_PORT) -u $(DB_USER) -p$(DB_PASS) $(DB_NAME) -e "\
		INSERT IGNORE INTO products (id, name, description, price, stock) \
		SELECT seq, CONCAT('Product-', seq), CONCAT('Description for product ', seq), \
		ROUND(RAND()*100+1, 2), FLOOR(RAND()*1000) \
		FROM (SELECT @rownum := @rownum + 1 AS seq FROM information_schema.columns a, \
		information_schema.columns b, (SELECT @rownum := 0) r LIMIT 100) t;"

loadtest: ## k6全戦略比較実行
	bash loadtest/compare.sh

precommit-install: ## pre-commitフック設定
	pre-commit install
