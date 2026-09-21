---
title: "Feature Idea: Multi-Repository Orchestration & Dependencies"
type: idea-proposal
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
created: 2026-09-21
updated: 2026-09-21
tags: [devctl, idea, roadmap, multi-repo, depends_on, stack]
---

# 機能拡張アイデア: マルチリポジトリ横断管理とプロジェクト間依存関係

## 1. 背景と動機 (Context & Motivation)

現代のWebアプリケーションやマイクロサービスでは、フロントエンド、バックエンドAPI、認証基盤、MLサービスなどが **複数の独立した Git リポジトリ** に分割されて管理されるケースが一般的です。

現在（v1.0.1）は、親ディレクトリに1つの `compose.yml` を置き、その配下に各リポジトリを配置する「**親ディレクトリ統合型（パターン①）**」で対応可能ですが、各リポジトリがそれぞれ固有の `compose.yml` を持ち完全に分散している「**マルチリポジトリ独立型（パターン②）**」では以下の課題が生じます。

今後、システム規模の拡大やリポジトリの独立運用に伴い、これらの複数リポジトリ環境をより快適にオーケストレーションするための機能拡張アイデアをここに保管します。

---

## 2. 現在の制約と課題 (Current Limitations)

1. **プロジェクト間依存関係の欠如**:
   - 現状の `depends_on` は `network`（共通ネットワーク）と `core`（共通基盤）のみ。
   - 「`frontend` を起動する前に、依存する別リポジトリの `backend` を自動で先行起動する」チェーンが組めない。
2. **単一プロジェクト前提の CLI 引数**:
   - `devctl up`, `devctl down` 等は第1引数をプロジェクト名、第2引数以降をコンテナサービス名として扱うため、`devctl up frontend backend` のような複数指定ができない。
   - 全プロジェクトを一括起動・停止する `--all` フラグがない。
3. **プロジェクトグループ / スタック概念の不在**:
   - 関連する複数のリポジトリ（プロジェクト）を1つの「業務スタック」として束ねて一括操作する仕組みがない。

---

## 3. 具体的な機能拡張アイデア

### アイデア A: プロジェクト間依存関係 (`depends_on.projects`)

`config.yaml` の各プロジェクト定義において、依存する他プロジェクトを指定可能にします。

#### 設定例 (`config.yaml`)
```yaml
projects:
  auth-service:
    workdir: ~/workspace/auth-service
    depends_on:
      network: true

  backend:
    workdir: ~/workspace/backend
    depends_on:
      network: true
      core:
        - dbgate
      projects:
        - auth-service  # backend 起動前に auth-service を先行起動

  frontend:
    workdir: ~/workspace/frontend
    depends_on:
      network: true
      projects:
        - backend       # frontend 起動前に backend (および auth-service) を連鎖起動
```

#### 期待される動作
- `devctl up frontend` を実行した際、依存グラフを解析（トポロジカルソート）し、`auth-service` -> `backend` -> `frontend` の順で自動先行起動。
- 循環依存（A -> B -> A）が定義された場合は設定読み込み時にエラーとして検知。

---

### アイデア B: 複数プロジェクトの一括操作 & `--all` フラグ

CLI コマンドの引数パースを拡張し、複数プロジェクトの同時指定や全プロジェクト一括操作を可能にします。

#### 使用例
```bash
# 複数プロジェクトを指定して一括起動
devctl up frontend backend ml-service

# 登録されている全プロジェクトを一括起動 / 停止
devctl up --all
devctl down --all

# 全プロジェクトのコンテナイメージを一括ビルド
devctl up --all --build
```

#### 実装アプローチ
- コマンドライン引数から「設定ファイルに存在するプロジェクト名」を判定し、複数マッチした場合は各プロジェクトの Compose クライアントを順次実行。

---

### アイデア C: プロジェクトグループ / スタック (`groups`) の導入

複数のプロジェクトを束ねる論理的なグループ（スタック）を `config.yaml` に定義可能にします。

#### 設定例 (`config.yaml`)
```yaml
groups:
  fullstack:
    description: "Webフルスタック環境（Frontend + Backend + Auth）"
    projects:
      - auth-service
      - backend
      - frontend

  worker-stack:
    description: "非同期ジョブ・バッチ処理環境"
    projects:
      - backend
      - worker-service
```

#### 使用例
```bash
# グループ単位で一括起動・停止
devctl group up fullstack
devctl group down fullstack

# グループ全体の稼働ステータス確認
devctl group ps fullstack
```

---

### アイデア D: リポジトリ横断タスク (`run`) の拡張

複数リポジトリを跨ぐ一括テストやリセットタスクを定義できるようにします。

#### 設定例
```yaml
tasks:
  test-all:
    description: "全リポジトリの単体テストを一括実行"
    command: |
      devctl run auth-service test
      devctl run backend test
      devctl run frontend test
```

---

## 4. 互換性および設計方針

- **完全な後方互換性の維持**:
  - パターン①（親ディレクトリ統合型）や単一リポジトリプロジェクトの既存設定は一切変更せずにそのまま動作し続けること。
- **段階的な導入ステップ案**:
  1. **Step 1 (低コスト・高効果)**: `depends_on.projects` のサポート（`internal/cmd/project.go` の `prepareDependencies` を再帰的に拡張）
  2. **Step 2**: `devctl up --all` / `down --all` のサポート
  3. **Step 3**: `groups:` セクションの導入による大規模マイクロサービス対応
