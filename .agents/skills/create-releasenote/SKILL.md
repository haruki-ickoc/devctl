---
name: create-releasenote
description: planファイルに定義された要求仕様の完了検証を行い、検証ログを含めたリリースノート（v<バージョン>-released.md）を生成してplanのステータスを更新します。
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
---

# スキル: リリースノート生成と計画完了処理

## 前提リソース

- リリースノート雛形: `.agents/skills/create-releasenote/resources/release-template.md`
- 要求書雛形: `.agents/skills/create-releasenote/resources/plan-template.md`
- 検証スクリプト: `.agents/skills/create-releasenote/scripts/verify.sh`

## 実行手順

1. **計画書の検証**
    - 対象の `releasenote/v<バージョン>-plan.md` を確認し、要求仕様のチェックボックスがすべて完了（`- [x]`）していることを確認します。未完了項目がある場合は中断してください。

2. **検証スクリプトの実行**
    - `bash .agents/skills/create-releasenote/scripts/verify.sh` を実行します。
    - エラーが発生した場合は処理を中断し、修正に戻ってください。
    - 実行時の出力ログを取得・保持します。

3. **リリースノートの生成**
    - 雛形: `.agents/skills/create-releasenote/resources/release-template.md` を読み込みます。
    - 出力先: `releasenote/v<バージョン>-released.md`
    - plan ファイルの「変更の目的・背景」や実装した差分から変更概要・詳細を埋めます。
    - 取得した検証ログを「動作確認・検証結果」欄にコードブロックとして貼り付けます。

4. **計画書の完了マーク**
    - `releasenote/v<バージョン>-plan.md` の `status` を `completed` に、`updated` を当日の日付に更新します。

5. **報告**
    - 生成した `releasenote/v<バージョン>-released.md` のパスと、更新した plan ファイルのステータスのみを端的に報告します。
