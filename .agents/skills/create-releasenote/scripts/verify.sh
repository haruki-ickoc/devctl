#!/usr/bin/env bash
set -e

echo "==> 1. 静的解析の実行 (go vet)"
go vet ./...

echo "==> 2. 単体テストの実行 (go test)"
go test ./...

echo "==> 3. バイナリビルド (make build)"
make build

echo "==> 4. 設定構文チェック (--dry-run)"
./bin/devctl -c configs/config.example.yaml config check

echo "==> 全ての検証が正常に完了しました。"