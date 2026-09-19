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

```text
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
├── examples/                    # 実動サンプルプロジェクト
│   └── sample/                  # サンプル環境 (Laravel + MySQL, マルチネットワーク構成)
├── scripts/                     # 配布・運用スクリプト (install.sh 等)
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

# 共通ネットワークの存在・Subnet/Gateway 整合性確認
devctl network check

# 共通ネットワークの作成・削除
devctl network create
devctl network rm
```

#### 固定 IP（Static IP）環境の設定ガイドと検証手順

1. **`config.yaml` での共通ネットワーク定義**:
   固定 IP アドレスを配備するために `subnet` と `gateway` を定義します。

   ```yaml
   network:
     name: "dev-network"
     driver: "bridge"
     attachable: true
     subnet: "172.20.0.0/16"
     gateway: "172.20.0.1"
   ```

2. **各プロジェクト側の `compose.yml` での固定 IP 指定**:
   プロジェクトのサービスに固定 IP を割り当てる際は、共通ネットワークを外部ネットワーク（`external: true`）として参照し、`ipv4_address` を指定します。

   ```yaml
   services:
     web:
       image: nginx:alpine
       networks:
         dev-network:
           ipv4_address: 172.20.0.10

   networks:
     dev-network:
       external: true
   ```

3. **事前検証と整合性チェック**:
   `devctl network check` を実行すると、実環境のネットワーク構成と `config.yaml` の設定値（Subnet / Gateway）が一致しているかを自動診断します。

   ```bash
   devctl network check
   ```

   > **設定不一致時のトラブルシューティング**:
   > 既にデフォルトの Docker サブネットでネットワークが作成されている場合など、設定値との不一致が検知された際は警告が表示されます。
   > その場合は `devctl network rm` で既存ネットワークを削除した上で、`devctl network create` または `devctl up` を実行して再作成してください。

4. **コンテナ起動後の動作確認**:
   プロジェクト起動後、コンテナに指定の IP が割り当てられているか確認できます。

   ```bash
   # devctl で起動
   devctl up web-app

   # 割り当てられた IP アドレスを確認
   docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' <コンテナ名>
   ```

#### 共通基盤 DBGate から各プロジェクト DB への接続設定

Core サービスとして起動する DBGate（Web データベース管理ツール: `http://dbgate.localhost`）から、各プロジェクトの DB コンテナへアクセスするための推奨設定手順です。

1. **共通ネットワークへのエイリアス設定 (`compose.yml`)**:
   各プロジェクトの `db` コンテナを共通ネットワーク（`core-network`）に参加させ、コンテナ名または固定エイリアスを付与します。

   ```yaml
   services:
     db:
       image: mysql:8.0
       networks:
         sample-network:
           ipv4_address: 10.11.0.3
         core-network:
           aliases:
             - sample-db  # DBGate から参照するホスト名

   networks:
     sample-network:
       name: sample-network
     core-network:
       name: core-network
       external: true
   ```

2. **DBGate Web UI（<http://dbgate.localhost）での接続設定>**:
   ブラウザから DBGate を開き、「Add connection」より以下の情報を入力します：
   - **Connection type**: `MySQL`（または使用する DB エンジン）
   - **Server (Host)**: `sample-db`（プロジェクト側で指定した alias 名）
   - **Port**: `3306`（Docker ネットワーク内部の標準ポート）
   - **User**: `sample_user`（設定したデータベースユーザー名）
   - **Password**: `sample_pass`
   - **Database**: `sample`

> **メリット**:
>
> - ホスト側のポート競合（`3306`, `3307` 等）を気にする必要がなく、全プロジェクトの DB に内部ポート `3306` のまま接続できます。
> - ホストマシンへポートフォワードを公開（`ports: "3306:3306"`）することなく安全にアクセス可能です。

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

## インストール & クイックスタート

### 1. ワンライナー・インストール（推奨）

Linux および macOS（Intel / Apple Silicon）環境に対応しています。コマンド 1 行で最新の単一バイナリが `~/.local/bin/devctl` に配置されます。

```bash
curl -fsSL https://raw.githubusercontent.com/haruki-ickoc/devctl/main/scripts/install.sh | bash
```

> ※ 特定のバージョンを指定してインストールする場合:
> ```bash
> curl -fsSL https://raw.githubusercontent.com/haruki-ickoc/devctl/main/scripts/install.sh | VERSION=v0.1.0 bash
> ```

### 2. GitHub Releases からの直接ダウンロード

[GitHub Releases](https://github.com/haruki-ickoc/devctl/releases) からご利用環境に合わせた単一バイナリを直接ダウンロードして配置できます。

| プラットフォーム | アーキテクチャ | ダウンロード対象ファイル名 |
| :--- | :--- | :--- |
| **Linux** | x86_64 / amd64 | `devctl-linux-amd64` |
| **Linux** | ARM64 / aarch64 | `devctl-linux-arm64` |
| **macOS** | Intel (amd64) | `devctl-darwin-amd64` |
| **macOS** | Apple Silicon (arm64) | `devctl-darwin-arm64` |
| **Windows** | x86_64 / amd64 | `devctl-windows-amd64.exe` |

```bash
# 例: Linux (amd64) で手動ダウンロードする場合
mkdir -p ~/.local/bin
curl -Lo ~/.local/bin/devctl https://github.com/haruki-ickoc/devctl/releases/latest/download/devctl-linux-amd64
chmod +x ~/.local/bin/devctl
```

### 3. ソースからのビルド（Go 開発環境がある場合）

```bash
git clone https://github.com/haruki-ickoc/devctl.git
cd devctl
make install
```

### 4. 初期セットアップ

```bash
# 設定ファイルのひな形 (~/.config/devctl/config.yaml) を生成
devctl config init

# 現在読み込まれている設定ファイルパスの確認
devctl config path

# 設定ファイルの構文および各ディレクトリ・compose ファイルの存在チェック
devctl config check
```

---

## ライセンス (License)

本プロジェクトは [MIT License](LICENSE) の下で公開されています。

Copyright (c) 2026 [haruki-ickoc](https://github.com/haruki-ickoc)
