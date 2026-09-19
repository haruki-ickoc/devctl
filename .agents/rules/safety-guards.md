---
title: safety-guards
description: devctl プロジェクトの安全制約ルール
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
---

# 安全制約ルール

- 外部コマンド（Docker操作など）を実行するコードを追加する場合は、必ず `internal/docker/runner.go` 等のランナー層を経由し、`--dry-run` 時にシミュレーション表示ができる状態を維持してください。
- ユーザー環境を破壊するリスクを防ぐため、コンテナの強制削除やネットワーク削除を伴う処理を、確認なしにローカル実環境で直接実行しないでください。
- `configs/config.example.yaml` の仕様と `internal/config/` の構造体定義に不整合を生じさせないでください。
