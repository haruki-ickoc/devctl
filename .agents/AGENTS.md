# プロジェクト概要

このリポジトリは、複数の Docker Compose 環境および共通基盤を横断管理する Go 言語製 CLI ツール「devctl」です。
Cobra フレームワークをベースに、依存関係を持たない単一バイナリとして動作します。

# 基本開発規約

- コード変更時は `.agents/rules/` 配下の規約を順守してください。
- 各作業が完了した際は、`.agents/skills/create-releasenote.md` のスキル手順に従って検証を実施し、`releasenote/` 配下に記録を作成した上で報告してください。
