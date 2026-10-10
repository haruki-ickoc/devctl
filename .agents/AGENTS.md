---
title: AGENTS
description: devctl プロジェクトのエージェント開発規約と概要
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
---

# プロジェクト概要

このリポジトリは、複数の Docker Compose 環境および共通基盤を横断管理する Go 言語製 CLI ツール「devctl」です。
Cobra フレームワークをベースに、依存関係を持たない単一バイナリとして動作します。

# 基本開発規約

- **コーディング・設計**: `.agents/rules/coding-style.md` および `.agents/rules/safety-guards.md` を順守してください。
- **計画ワークフロー**: すべての機能追加・改修は `.agents/rules/plan-workflow.md` に基づき、`releasenote/v<バージョン>-plan.md` を作成して進めてください。
- **Git 運用・コミット**: ブランチ作成、コミットメッセージ、PR 作成時は `.agents/rules/git-workflow.md` を順守してください。
- **ドキュメント保守**: 機能・設定の変更時は `.agents/rules/documentation.md` に従い、README や設定例を完全同期してください。
- **リリース検証**: 各作業が完了した際は、`.agents/skills/create-releasenote/SKILL.md` のスキル手順に従って検証（`verify.sh`）を実施し、`releasenote/v<バージョン>-released.md` を作成した上で報告してください。
- **PR 作成**: `develop` ブランチ向けのプルリクエストを作成・提示する際は、`.agents/skills/create-pr/SKILL.md` のスキル手順に従ってください。
