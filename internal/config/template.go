package config

// DefaultConfigYAML は 'devctl config init' 実行時に生成される設定ファイルのひな形です。
const DefaultConfigYAML = `# devctl 設定ファイル
version: "1"

# 全体デフォルト設定
defaults:
  compose_cmd: "docker compose" # または "docker-compose"
  env_file: ".env"

# 共通 Docker ネットワーク
network:
  name: "internal-network"
  driver: "bridge"
  attachable: true

# 共通インフラ基盤 (Core Services)
core:
  workdir: "~/workspace/toolbox"
  compose_files:
    - "compose.yml"
  env_file: ".env"
  services:
    traefik:
      description: "Traefik リバースプロキシ & SSL終端 (http://traefik.localhost)"
    dbgate:
      description: "DBGate Webデータベース管理クライアント (http://dbgate.localhost)"
    dozzle:
      description: "Dozzle リアルタイムログビューア (http://dozzle.localhost)"

# 管理対象プロジェクト
projects:
  sample:
    description: "サンプルアプリケーション環境"
    workdir: "~/workspace/sample"
    compose_files:
      - "compose.yml"
    env_file: ".env"
    depends_on:
      network: true
      core:
        - "traefik"

  catchUper:
    description: "catchUper フルスタックWebサービス (Nuxt + Node.js + FastAPI)"
    workdir: "~/workspace/catchUper"
    compose_files:
      - "docker-compose.yml"
    depends_on:
      network: true
    tasks:
      logs:
        description: "全サービスのログをリアルタイム追跡"
        command: "docker compose logs -f"
      backend-sh:
        description: "backend コンテナでシェルを起動"
        command: "docker compose exec backend sh"
      scraper-sh:
        description: "scraper コンテナでシェルを起動"
        command: "docker compose exec scraper sh"
`
