---
title: documentation
description: ドキュメント作成・保守規約（README、設定例、アイデア起票）
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
---

# ドキュメント作成・保守規約 (documentation.md)

このリポジトリにおけるドキュメント（`README.md`、設定例、アイデア構想書など）の作成および保守ルールを定めます。エージェントは機能追加や仕様変更を行う際、本規約に従ってドキュメントの完全な同期を担保してください。

---

## 1. ドキュメント同期の基本原則

- **コードとドキュメントの同時更新**:
  - 新機能の追加、CLI フラグの変更、設定項目の改修を行う際は、必ず該当するドキュメントも同一の PR / コミット内で更新してください。ドキュメントの更新が後回しになることを防ぎます。
- **正確性と整合性**:
  - `configs/config.example.yaml` の記述内容と `internal/config/` の構造体定義に不整合を生じさせないでください。

---

## 2. `README.md` の更新基準

新しいサブコマンドやフラグの追加、既存コマンドの挙動変更を行う場合は、`README.md` の以下のセクションを更新します。

1. **「主な特徴」 (Key Features)**:
   - ユーザー体験を大きく向上させる機能（例: Compose Override 自動マージ、独立 build コマンド等）を追加した場合は、特徴一覧に項目を追記します。
2. **「使い方」 (Usage)**:
   - 該当する分類（プロジェクト操作、共通基盤操作、ネットワーク操作等）のコードブロックに使用例を追加します。
   - サブコマンド形式（`devctl project <cmd>`）とトップレベル短縮形式（`devctl <cmd>`）の両方を併記します。
   - 主要なフラグ（`-f`, `--no-cache` 等）を組み合わせた実用的なコマンド例を記載します。
3. **引用ブロック（Note / Tip）**:
   - カレントディレクトリ自動検出の挙動や、環境切り替えに関する重要な補足がある場合は、引用ブロック（`> **タイトル**:`）で分かりやすく説明します。

---

## 3. 設定ファイル例 (`configs/config.example.yaml`) の更新基準

設定スキーマの変更や機能の拡張（例: 新フラグ、新規タスク連携等）に伴い、設定例も最新状態に保ちます。

1. **実用的なコメントの記載**:
   - 設定フィールドには、省略時のデフォルト挙動や推奨設定をコメントで明記します。
2. **CLI との関連ガイダンス**:
   - CLI 側の標準機能（例: `devctl build`）によって不要になった設定（例: `tasks` への `docker compose build` 自前定義）がある場合は、その旨をコメントで案内します。
3. **`internal/config/template.go` との同期**:
   - `devctl config init` で生成されるデフォルト設定テンプレート（`DefaultConfigYAML`）と設計思想の整合性を維持してください。

---

## 4. 将来アイデア・構想ドキュメント (`releasenote/ideas-*.md`) の起票

未確定の大型機能やロードマップ候補を記録する際は、`releasenote/ideas-<機能名>.md` として作成します。

### 必須フロントマター
```yaml
---
title: "Feature Idea: <機能名>"
type: idea-proposal
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
created: YYYY-MM-DD
updated: YYYY-MM-DD
tags: [devctl, idea, roadmap, ...]
---
```

### 記載構成
1. **背景と動機 (Context & Motivation)**: 解決したい課題や現場のニーズ。
2. **コマンド構文とインターフェース設計**: 基本構文、サポートするフラグ一覧。
3. **具体的な使用例**: 代表的なユースケースごとの実行例。
4. **内部実装アプローチ (Implementation Approach)**: 変更想定パッケージと実装コード骨子。
5. **期待される効果 (Benefits)**: 導入によるメリット。
