#!/bin/bash
set -e

# 作業ディレクトリに移動
cd /var/www/html

# Composer 依存関係のインストール（初回起動時など vendor がない場合）
if [ ! -d "vendor" ] && [ -f "composer.json" ]; then
    echo "📦 Installing composer dependencies..."
    composer config -g policy.advisories.block false 2>/dev/null || true
    composer install --no-interaction --prefer-dist --optimize-autoloader --no-audit
fi

# storage および bootstrap/cache のパーミッション確保 (Laravel 環境用)
if [ -d "storage" ]; then
    mkdir -p storage/framework/cache/data storage/framework/sessions storage/framework/views storage/logs bootstrap/cache
    chmod -R 777 storage bootstrap/cache 2>/dev/null || true
fi

# APP_KEY が未設定の場合は生成 (Laravel 環境用)
if [ -f "artisan" ] && ! grep -q "^APP_KEY=base64:" .env 2>/dev/null; then
    echo "🔑 Generating application key..."
    php artisan key:generate --force || true
fi

# データベース接続待ち
if [ -n "$DB_HOST" ]; then
    echo "⏳ Waiting for MySQL ($DB_HOST:$DB_PORT) to be ready..."
    php -r "
    for (\$i = 0; \$i < 30; \$i++) {
        try {
            new PDO('mysql:host=' . (getenv('DB_HOST') ?: 'db') . ';port=' . (getenv('DB_PORT') ?: '3306') . ';dbname=' . getenv('DB_DATABASE'), getenv('DB_USERNAME'), getenv('DB_PASSWORD'));
            echo \"✅ MySQL is ready!\n\";
            exit(0);
        } catch (Exception \$e) {
            sleep(2);
        }
    }
    echo \"⚠️ Could not connect to MySQL within timeout.\n\";
    exit(1);
    " || true

    # 自動マイグレーション実行
    if [ -f "artisan" ]; then
        echo "🔄 Running database migrations..."
        php artisan migrate --force || true
    fi
fi

# コマンドを実行
exec "$@"
