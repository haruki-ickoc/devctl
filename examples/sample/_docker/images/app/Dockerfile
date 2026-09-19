FROM php:8.2-apache

# システム依存パッケージのインストール
RUN apt-get update && apt-get install -y \
    git \
    curl \
    libpng-dev \
    libonig-dev \
    libxml2-dev \
    libzip-dev \
    libsqlite3-dev \
    zip \
    unzip \
    default-mysql-client \
    && docker-php-ext-install pdo_mysql pdo_sqlite mbstring bcmath zip gd opcache \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Apache 設定の調整 (Laravel の public ディレクトリをドキュメントルートに設定)
ENV APACHE_DOCUMENT_ROOT /var/www/html/public
RUN sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf \
    && sed -ri -e 's!/var/www/!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/apache2.conf /etc/apache2/conf-available/*.conf \
    && a2enmod rewrite

# Composer のインストール
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

# エントリポイントスクリプトの配置
COPY _docker/entrypoints/app.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /var/www/html

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["apache2-foreground"]
