# devctl

複数プロジェクトの Docker Compose 環境および共通基盤（リバースプロキシ、共通DB、ログ監視、共通ネットワークなど）を単一のコマンドから統合管理するための Go 製 CLI ツールです。

## 主な特徴

- **単一バイナリ (Single Binary)**: CGO 不要、外部ランタイム不要で各環境の `bin` に配置して即実行可能。
- **Compose ファイル自動検出**: `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml` を自動認識。
- **共通基盤の統合制御 (`devctl core`)**: リバースプロキシや共通 DB などの共通コンテナをどこからでも一発で起動・停止。
- **共通ネットワークの自動担保 (`devctl network`)**: プロジェクトやコアサービスの起動時に、必要な Docker ネットワークの存在確認と自動作成を実施。
- **プロジェクト横断の Docker Compose 操作 (`devctl project` / `devctl up`)**: 任意のカレントディレクトリから対象プロジェクト名を指定して一元管理。
- **Makefile の共通化・吸収 (`devctl run`)**: 各リポジトリに散乱しがちなログ確認やシェル起動などのタスクを設定ファイル (`config.yaml`) に集約。
- **コンテキスト自動判定**: プロジェクトのリポジトリディレクトリに移動している場合は、プロジェクト名の指定を自動で省略可能。

---

## ディレクトリ構成

```
devctl/
├── cmd/
│   └── devctl/
│       └── main.go              # CLI エントリポイント
├── internal/
│   ├── cmd/                     # Cobra コマンド群
│   │   ├── root.go              # ルートコマンド、共通フラグ (--config, --dry-run, --verbose)
│   │   ├── core.go              # devctl core (共通基盤の起動・停止・ログ・ステータス)
│   │   ├── project.go           # devctl project / top-level aliases (各プロジェクト操作・タスク実行)
│   │   ├── network.go           # devctl network (共通ネットワークの確認・作成・削除)
│   │   ├── ps.go                # devctl ps (コア・全プロジェクト横断ステータス一覧)
│   │   ├── config_cmd.go        # devctl config (設定確認・初期化・構文/パスチェック)
│   │   └── version.go           # devctl version (バージョン表示)
│   ├── config/                  # 設定ローダー、YAML パース、チルダ/環境変数展開、Compose 自動検出
│   │   ├── config.go
│   │   ├── config_test.go
│   │   └── template.go
│   ├── docker/                  # Docker & Compose コマンド実行ラッパー
│   │   ├── compose.go
│   │   ├── network.go
│   │   └── runner.go
│   └── ui/                      # ターミナル出力装飾
│       └── output.go
├── configs/
│   └── config.example.yaml      # 設定ファイル仕様サンプル
├── Makefile                     # ビルド・クロスコンパイル用
├── go.mod
└── go.sum
```

---

## コマンド体系

### 1. 共通基盤 (Core Infrastructure)

```bash
# 共通ネットワークを自動確認・作成した上で共通基盤を起動
devctl core up

# 共通基盤を停止
devctl core down

# 共通基盤を再起動
devctl core restart

# 共通基盤の稼働ステータスを確認
devctl core ps

# 共通基盤のログを確認
devctl core logs -f
```

### 2. プロジェクト操作 (Project Management)

```bash
# 登録プロジェクト一覧を表示
devctl project list    # または devctl list

# 依存するネットワークや Core サービスを準備してプロジェクトを起動
devctl project up sample    # または devctl up sample

# プロジェクトを停止
devctl project down sample  # または devctl down sample

# プロジェクトのログを確認
devctl project logs -f catchUper

# コンテナ内でコマンドを実行
devctl project exec catchUper backend sh

# 設定ファイルに定義されたカスタムタスク（旧 Makefile 処理）を実行
devctl run catchUper logs
devctl run catchUper backend-sh
```

> **カレントディレクトリの自動判定**:
> プロジェクトの作業ディレクトリに移動している場合、`devctl up` や `devctl run logs` のようにプロジェクト名の引数を省略して実行できます。

### 3. 全体ステータス & ネットワーク管理

```bash
# ネットワーク、Core、全プロジェクトの稼働状況をまとめて表示
devctl ps

# 共通ネットワークの個別操作
devctl network check
devctl network create
devctl network rm
```

### 4. 設定ファイルの管理

```bash
# 汎用サンプル設定ファイル (~/.config/devctl/config.yaml) を生成
devctl config init

# チーム等で配布された既存設定ファイルをインポート配置（構文検証付き）
devctl config init ./shared-config.yaml
devctl config init --from ./configs/team.yaml

# 既存の設定ファイルを強制上書き
devctl config init ./shared-config.yaml -f

# 現在読み込まれている設定ファイルパスを表示
devctl config path

# 設定内容をダンプ表示
devctl config view

# 設定ファイルの構文および各ディレクトリ・compose ファイルの存在チェック
devctl config check
```

---

## クイックスタート & ビルド

```bash
# ビルド (bin/devctl が生成されます)
make build

# ~/.local/bin/devctl にインストール
make install

# 設定ファイルのひな形を作成
devctl config init

# 設定ファイルのパスを確認・編集
devctl config path
```
