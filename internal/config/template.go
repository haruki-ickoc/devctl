package config

// DefaultConfigYAML は 'devctl config init' 実行時に生成される汎用的な設定ファイルのひな形です。
// 各開発者が自身のローカル環境に合わせてパスやサービス名を編集して利用します。
const DefaultConfigYAML = `# devctl 設定ファイル
# 各開発者のローカル環境に合わせて workdir やサービス定義を編集してください。
version: "1"

# 全体デフォルト設定
defaults:
  compose_cmd: "docker compose" # または "docker-compose"
  env_file: ".env"

# 共通 Docker ネットワーク定義
# プロジェクトや Core の起動時に devctl が存在を確認し、なければ自動作成します
network:
  name: "dev-network"
  driver: "bridge"
  attachable: true
  # 固定IP（Static IP）サポート用（省略時はDockerのデフォルトが適用されます）
  # subnet: "172.20.0.0/16"
  # gateway: "172.20.0.1"

# 共通インフラ基盤 (Core Services)
# リバースプロキシ、ログ基盤、共通DB、Redis などの横断サービスを管理します
core:
  # 共通基盤の docker-compose.yml / compose.yml が配置されたディレクトリ
  workdir: "~/workspace/core-infra"
  # compose_files を省略した場合、compose.yaml / compose.yml / docker-compose.yaml / docker-compose.yml が自動検出されます
  compose_files:
    - "compose.yml"
  env_file: ".env"
  services:
    traefik:
      description: "Traefik リバースプロキシ & SSL終端"
    database:
      description: "共通データベース (PostgreSQL / MySQL 等)"
    redis:
      description: "共通 Redis キャッシュサーバー"

# 管理対象プロジェクト定義
projects:
  web-app:
    description: "メインWebアプリケーション"
    workdir: "~/workspace/web-app"
    # compose_files は省略可能（compose.yml 等が存在すれば自動検出）
    env_file: ".env"
    # プロジェクト起動前に自動起動すべき前提依存
    depends_on:
      network: true
      core:
        - "database"
        - "redis"
    # 各プロジェクト固有のタスク（Makefile 代替コマンド）
    tasks:
      migrate:
        description: "データベースマイグレーションの実行"
        command: "docker compose exec app npm run migrate"
      seed:
        description: "初期シードデータの投入"
        command: "docker compose exec app npm run seed"
      logs:
        description: "アプリケーションログのリアルタイム追跡"
        command: "docker compose logs -f app"

  api-service:
    description: "API マイクロサービス"
    workdir: "~/workspace/api-service"
    depends_on:
      network: true
      core:
        - "database"
    tasks:
      test:
        description: "コンテナ内ユニットテスト実行"
        command: "docker compose exec api go test ./..."
`
