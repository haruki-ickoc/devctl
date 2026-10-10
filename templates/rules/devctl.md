---
title: devctl-operations
description: devctl プロジェクトにおけるコンテナ・インフラ操作の規約
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
---

# コンテナ・インフラ操作規約 (devctl)

本プロジェクトは `devctl` CLI によってインフラ、ネットワーク、および依存関係が統合管理されています。
エージェントがコンテナの起動・停止・ログ確認・ビルド・タスク実行を行う際は、直接 `docker` や `docker compose` コマンドを実行せず、**必ず `devctl` CLI を利用してください**。

## 推奨コマンド対応表

| 行いたい操作 | 実行すべきコマンド (devctl) | 非推奨（直接実行禁止） |
| :--- | :--- | :--- |
| **起動** | `devctl up`（または `devctl up --build`） | `docker compose up -d` |
| **停止** | `devctl down` | `docker compose down` |
| **再起動** | `devctl restart` | `docker compose restart` |
| **ログ確認** | `devctl logs -f [service]` | `docker compose logs -f` |
| **ステータス確認** | `devctl ps` | `docker compose ps` |
| **タスク実行** | `devctl run <task>` (例: `devctl run migrate`) | `docker compose exec app ...` |
| **コンテナ内シェル接続** | `devctl exec <service> sh` (または `devctl run bash`) | `docker exec -it ...` |
| **本番構成での実行** | `devctl up -f compose.prod.yml` | `docker compose -f ... -f ... up` |

## `devctl` を使用する理由

1. **共通ネットワークの自動担保**:
   - `core-network` の存在確認および未作成時の自動作成を透過的に行います。
2. **共通基盤サービス（Core）の依存解決**:
   - `depends_on.core` に定義された Traefik や DBGate などの前提サービスを自動で先行起動します。
3. **Docker Compose Override の自動マージ**:
   - `compose.override.yml`（開発用ステージ `target: dev`、バインドマウント、ローカルポート等）を安全かつ確実にマージします。
4. **ホスト環境の汚染防止**:
   - 各プロジェクト固有の環境変数（`.env`）および作業ディレクトリが正しく適用されます。
