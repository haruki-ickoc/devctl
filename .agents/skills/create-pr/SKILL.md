---
name: create-pr
description: 実装・検証が完了したブランチから、develop 向けのプルリクエスト（PR）のタイトルおよび本文（説明文）を日本語で生成します。
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
---

# スキル: プルリクエスト (PR) 生成

## 概要

このスキルは、機能開発や修正が完了したブランチ（`feature/*`, `fix/*` 等）から、`develop` ブランチ向けに起票するプルリクエスト（PR）のタイトルおよび本文を標準テンプレートに従って自動生成します。

---

## 前提条件

1. 対象バージョンの計画書（`releasenote/v<バージョン>-plan.md`）のステータスが `completed` であること。
2. リリースノート（`releasenote/v<バージョン>-released.md`）が生成されていること。
3. すべての変更がコミットされ、リモートブランチ（`origin/<ブランチ名>`）へプッシュ済みであること。

---

## 前提リソース

- PR 雛形: `.agents/skills/create-pr/resources/pr-template.md`
- Git 運用規約: `.agents/rules/git-workflow.md`

---

## 実行手順

1. **変更情報の収集**:
   - カレントブランチ名を取得します（例: `feature/build-command`）。
   - 最新のコミットメッセージを取得します（`git log -1 --pretty=%B`）。
   - 対象バージョンの `releasenote/v<バージョン>-plan.md` および `v<バージョン>-released.md` を読み込みます。
   - 解決対象の GitHub Issue 番号を確認します（plan ファイル内の `issue:` フロントマターまたはコミットメッセージ）。

2. **PR タイトルの決定**:
   - 原則「日本語」で記載します。
   - コミットの 1 行目と同一またはそれに準じた形式とします。
   - 例: `feat: コンテナイメージのビルド専用コマンド (build) の追加 (v1.1.0)`

3. **PR 本文の生成**:
   - `.agents/skills/create-pr/resources/pr-template.md` を読み込みます。
   - 各セクションを以下の情報で埋めます：
     - **概要**: released ファイルの「変更概要」および背景。
     - **主な変更点**: released ファイルの「変更詳細」を箇条書きで整理。
     - **関連 Issue**: 対象 Issue がある場合は `Closes #XX` を記載（なければ「特になし」）。
     - **破壊的変更**: released ファイルの「破壊的変更」の内容を転記。
     - **動作確認・検証結果**: released ファイルに記録された `verify.sh` のログや CLI 実行結果を抜粋。

4. **出力および報告**:
   - 生成した PR タイトルおよび Markdown 本文をユーザーに提示します。
   - GitHub Web UI で PR を作成するための URL（`https://github.com/<owner>/<repo>/pull/new/<branch>`）を案内します。
   - （ユーザーの明示的な許可がある場合）GitHub API を利用して直接 PR を起票することも可能です。
