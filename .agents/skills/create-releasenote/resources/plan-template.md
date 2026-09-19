---
title: v{{TARGET_VERSION}}-plan
target_version: {{TARGET_VERSION}}
status: planned # planned | in-progress | completed
created: {{YYYY-MM-DD}}
updated: {{YYYY-MM-DD}}
type: change-request
tags: [devctl, docker, compose, cli, golang]
---

## 変更の目的・背景

今回のバージョンで解決する課題や、追加・改善を行う背景を1〜2文で簡潔に記載する。

## 要求仕様一覧

- [ ] 要求項目1: 実装すべき機能、追加するコマンド、修正する挙動を具体的に記載する。
- [ ] 要求項目2:
- [ ] 要求項目3:

## 変更対象の想定範囲

作業の影響を最小限に抑えるため、編集を想定するパッケージやファイルを明示する。

- 想定修正ディレクトリ/ファイル:
  - `internal/...`
  - `configs/...`

## 技術的制約および互換性

- 破壊的変更の許容: なし（既存の config.yaml や CLI 引数の互換性を維持すること）
- ランナー層の経由: コマンド実行を追加する場合は必ず `internal/docker/runner.go` 等を経由し、`--dry-run` に対応させること。
- 外部パッケージの追加: 原則不可（標準ライブラリおよび既存の Cobra / YAML パッケージの範囲で実装すること）。

## 完了の受け入れ基準 (Acceptance Criteria)

以下の条件をすべて満たした時点で作業完了とみなす。

- [ ] 定義した要求仕様一覧のすべてのチェックボックスが満たされていること。
- [ ] `.agents/skills/create-releasenote/scripts/verify.sh` が正常終了（エラー0件）すること。
- [ ] `releasenote/v{{TARGET_VERSION}}-released.md` がテンプレートに沿って生成され、動作確認ログが記録されていること。
- [ ] 本ファイルの `status` が `completed` に更新されていること。
