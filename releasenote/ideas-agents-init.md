---
title: "Feature Idea: devctl agents init Scaffolding Subcommand"
type: idea-proposal
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
created: 2026-09-22
updated: 2026-09-22
tags: [devctl, idea, roadmap, agents, scaffolding, rules, cli]
---

# 機能拡張アイデア: `devctl agents init` スキャフォールディング機能

## 1. 背景と動機 (Context & Motivation)

`devctl` を利用する各アプリケーションプロジェクトにおいて、AI コーディングエージェント（Antigravity 等）が不用意に直接 `docker compose` を実行せず、常に `devctl` CLI を利用するように制約を適用する必要があります。

現在は、配布用テンプレート（`templates/rules/devctl.md`）を手動で各プロジェクトの `.agents/rules/` にコピー配置するアプローチ（アプローチ 1）で運用していますが、プロジェクト数が増えた際や新規メンバーがセットアップする際に、以下の課題が生じます：

- テンプレートファイルの場所を探して手動でディレクトリ作成・コピーする手間がかかる。
- プロジェクトごとにエージェント規約のバージョンや内容に差異・更新漏れが生じやすい。

これを解決するため、`devctl` CLI 自体に **エージェント規約の自動配置コマンド（`devctl agents init`）** を実装し、1コマンドでスキャフォールディングを完了できるようにします。

---

## 2. コマンド構文とインターフェース設計

### 基本構文
```bash
# カレントディレクトリのプロジェクトに .agents/rules/devctl.md を配置
devctl agents init

# または
devctl agents init [target-path] [flags]
```

### サポートするフラグ一覧
| フラグ | 短縮 | 型 | 説明 |
| :--- | :--- | :--- | :--- |
| `--force` | `-f` | bool | 既に `.agents/rules/devctl.md` が存在する場合に上書き配置 |
| `--dry-run` | `-n` | bool | 実際にファイルを生成せず、配置予定のパスと内容をプレビュー表示 |

---

## 3. 具体的な使用例と動作フロー

### 使用例
```bash
cd ~/workspace/my-new-app

# エージェント規約を自動生成
devctl agents init

# 出力例:
# ✔ .agents/rules/ ディレクトリを確認しました。
# ✔ .agents/rules/devctl.md を正常に配置しました。
# ℹ Antigravity がプロジェクト内のコンテナ操作に devctl を優先利用するよう設定されました。
```

### 既存ファイルが存在する場合の保護動作
```bash
$ devctl agents init
▲ .agents/rules/devctl.md は既に存在します。上書きする場合は -f (--force) を指定してください。

$ devctl agents init -f
✔ .agents/rules/devctl.md を上書き配置しました。
```

---

## 4. 内部実装アプローチ (Implementation Approach)

### 1. テンプレート定数の埋め込み (`internal/config/template.go` 等)
バイナリ単体で動作させるため、`templates/rules/devctl.md` の内容を Go の文字列定数、または `//go:embed` でバイナリ内に埋め込みます。

```go
// DefaultDevctlRuleTemplate は各プロジェクトに配布するエージェント規約テンプレートです。
const DefaultDevctlRuleTemplate = `---
title: devctl-operations
description: devctl プロジェクトにおけるコンテナ・インフラ操作の規約
---
...
`
```

### 2. コマンド実装 (`internal/cmd/agents_cmd.go` を新設)
- `devctl config init` の実装パターンを踏襲。
- 指定された作業ディレクトリ（未指定時はカレントディレクトリ）配下に `.agents/rules/` ディレクトリを作成し、`devctl.md` を書き出す。
- `--dry-run` 時はファイル書き込みを行わずシミュレーション表示を実施。

---

## 5. 期待される効果 (Benefits)

1. **ゼロ手間のセットアップ**:
   新規リポジトリの立ち上げ時に `devctl agents init` を叩くだけで、即座にプロジェクト向けエージェント規約が整う。
2. **規約の一貫性担保**:
   チーム内のすべての開発プロジェクトで、最新かつ統一された `devctl` 運用ルールが適用される。
3. **完全な単一バイナリ完結**:
   外部のスクリプトやリポジトリのクローンが不要で、インストール済みの `devctl` バイナリだけで完結する。
