#!/usr/bin/env bash
set -e

# ==============================================================================
# devctl Installer Script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/skyou/devctl/main/scripts/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/skyou/devctl/main/scripts/install.sh | VERSION=v0.1.0 bash
# ==============================================================================

REPO="skyou/devctl"
BIN_NAME="devctl"

# 1. OS & アーキテクチャの判別
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${OS}" in
    linux)
        TARGET_OS="linux"
        ;;
    darwin)
        TARGET_OS="darwin"
        ;;
    *)
        echo "❌ エラー: サポートされていない OS です: ${OS}"
        exit 1
        ;;
esac

case "${ARCH}" in
    x86_64|amd64)
        TARGET_ARCH="amd64"
        ;;
    aarch64|arm64)
        TARGET_ARCH="arm64"
        ;;
    *)
        echo "❌ エラー: サポートされていないアーキテクチャです: ${ARCH}"
        exit 1
        ;;
esac

BINARY_FILENAME="${BIN_NAME}-${TARGET_OS}-${TARGET_ARCH}"

# 2. ダウンロード URL の構築
if [ -n "${VERSION}" ]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_FILENAME}"
    echo "==> devctl バージョン ${VERSION} (${TARGET_OS}/${TARGET_ARCH}) をダウンロード中..."
else
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_FILENAME}"
    echo "==> devctl 最新リリース (${TARGET_OS}/${TARGET_ARCH}) をダウンロード中..."
fi

# 3. インストール先ディレクトリの決定
if [ -z "${INSTALL_DIR}" ]; then
    if [ "$(id -u)" -eq 0 ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="${HOME}/.local/bin"
    fi
fi

mkdir -p "${INSTALL_DIR}"
TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}"

# 4. 一時ディレクトリでダウンロードと配置
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

TMP_BIN="${TMP_DIR}/${BIN_NAME}"

if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_BIN}"
elif command -v wget >/dev/null 2>&1; then
    wget -qO "${TMP_BIN}" "${DOWNLOAD_URL}"
else
    echo "❌ エラー: curl または wget が必要です。"
    exit 1
fi

chmod +x "${TMP_BIN}"
mv "${TMP_BIN}" "${TARGET_PATH}"

echo "✔ devctl を正常にインストールしました: ${TARGET_PATH}"

# 5. PATH の確認と案内
if ! command -v "${BIN_NAME}" >/dev/null 2>&1; then
    echo ""
    echo "⚠️  注意: ${INSTALL_DIR} に PATH が通っていません。"
    echo "お使いのシェル設定ファイル（~/.bashrc, ~/.zshrc 等）に以下を追加してください:"
    echo ""
    echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    echo "設定後、以下のコマンドで反映できます:"
    echo "    source ~/.bashrc  # または source ~/.zshrc"
else
    echo ""
    "${TARGET_PATH}" version 2>/dev/null || true
fi

echo ""
echo "🎉 インストールが完了しました！ 'devctl --help' で利用可能なコマンドを確認できます。"
