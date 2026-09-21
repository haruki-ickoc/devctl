---
title: "Feature Idea: Dedicated build Subcommand"
type: idea-proposal
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
created: 2026-09-21
updated: 2026-09-21
tags: [devctl, idea, roadmap, build, cli, docker, compose]
---

# 機能拡張アイデア: `devctl build` サブコマンドの新設

## 1. 背景と動機 (Context & Motivation)

現状（v1.0.1）の `devctl` では、コンテナ起動と同時にイメージをビルドする `devctl up --build`（`-b`）はサポートされていますが、**コンテナを起動（デタッチド起動）せずにイメージのビルドのみを実行する専用コマンド**が存在しません。

以下のようなユースケースにおいて、`tasks`（カスタムタスク）を自前で定義したり直接 `docker compose build` を叩いたりすることなく、`devctl` から統一されたインターフェースでビルドを行いたいという需要があります：

- **ローカル開発時**: ソースコードや Dockerfile を変更した後、起動前にイメージ作成のみをテストしたい。
- **特定サービスの個別ビルド**: マルチリポジトリ構成（例: `my-system`）で、フロントエンド（`front`）やバックエンド（`app`）のイメージのみを単体で再ビルドしたい。
- **環境別ターゲットビルド**: `-f compose.prod.yml` を指定して、本番ステージ（`target: prod`）のイメージを検証・作成したい。
- **CI/CD 環境**: コンテナを立ち上げずにイメージのビルドおよび構文チェックのみを自動実行したい。

---

## 2. コマンド構文とインターフェース設計

### 基本構文
```bash
devctl build [project-name] [services...] [flags]

# または
devctl project build [project-name] [services...] [flags]
```

### サポートするフラグ一覧
| フラグ | 短縮 | 型 | 説明 |
| :--- | :--- | :--- | :--- |
| `--file` | `-f` | stringArray | 追加または上書きで適用する Compose ファイル (例: `compose.prod.yml`) |
| `--no-cache` | | bool | キャッシュを使用せずに最初からイメージをビルド |
| `--pull` | | bool | 常に最新のベースイメージをプルしてビルド |
| `--dry-run` | `-n` | bool | (グローバル) コマンドを実行せずプレビュー表示 |
| `--config` | `-c` | string | (グローバル) 設定ファイルパスの明示指定 |

---

## 3. 具体的な使用例

### ① プロジェクト内の全サービスを一括ビルド
```bash
devctl build my-system
```

### ② 対象リポジトリ（サービス）を個別指定してビルド
```bash
# フロントエンド (sample-front) のみビルド
devctl build my-system front

# バックエンド (sample) のみビルド
devctl build my-system app

# 複数サービスを指定してビルド
devctl build my-system front app
```

### ③ 本番ターゲット（target: prod）でビルド
既存の `-f` フラグと連携し、開発用 override を自動除外して `compose.prod.yml` を重ねてビルドします。
```bash
devctl build my-system front -f compose.prod.yml
```

### ④ カレントディレクトリ自動判定での実行
プロジェクトの作業ディレクトリ（`workdir`）配下にいる場合は、プロジェクト名を省略可能とします。
```bash
cd ~/workspace/my-system

# プロジェクト名不要でフロントエンドをビルド
devctl build front

# キャッシュ無効化でビルド
devctl build app --no-cache
```

---

## 4. 内部実装アプローチ (Implementation Approach)

### 1. `internal/docker/compose.go` への `Build` メソッド追加
```go
// Build は指定されたサービスのコンテナイメージをビルドします（コンテナは起動しません）。
func (c *ComposeClient) Build(opts ComposeOptions, noCache, pull bool) error {
    bin, baseArgs := c.buildBaseArgs(opts)
    args := append(baseArgs, "build")
    if noCache {
        args = append(args, "--no-cache")
    }
    if pull {
        args = append(args, "--pull")
    }
    args = append(args, opts.Services...)
    args = append(args, opts.ExtraArgs...)

    ui.Step("%s でイメージをビルド中...", opts.WorkDir)
    return c.runner.RunCommand(opts.WorkDir, bin, args...)
}
```

### 2. `internal/cmd/project.go` へのコマンド追加
- `projectBuildCmd` を定義（`projectUpCmd` と同様に `resolveTargetProject` で引数を解析）。
- トップレベル短縮コマンド `topBuildCmd` を定義し、`RootCmd.AddCommand(topBuildCmd)` で登録。
- `-f / --file`, `--no-cache`, `--pull` フラグをバインド。

---

## 5. 期待される効果 (Benefits)

1. **`tasks` 定義の削減**:
   各プロジェクトの `config.yaml` に `command: "docker compose build ..."` を個別に書く必要がなくなります。
2. **直感的な CLI 体験**:
   `devctl up`, `devctl down`, `devctl restart`, `devctl logs`, `devctl ps` に加え、Docker Compose 標準の主要操作である `build` が第一級コマンドとして揃います。
3. **CI/CD 自動化の容易化**:
   GitHub Actions 等で `devctl build <project> -f compose.prod.yml` の1行で確実なイメージ作成・検証を行えるようになります。
