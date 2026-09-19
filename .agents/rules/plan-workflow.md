---
title: plan-workflow
description: 計画書（plan）に基づく開発ワークフロー規約
author: "haruki-ickoc <https://github.com/haruki-ickoc>"
---

# 計画書（plan）に基づく開発ワークフロー規約

このリポジトリでの機能追加・改修は、すべて `releasenote/v<バージョン>-plan.md` に基づいて実施してください。

1. **計画書の作成**
   - 新しい改修を計画する際は、`.agents/skills/create-releasenote/resources/plan-template.md` を複製して `releasenote/v<バージョン>-plan.md` を作成すること。

2. **作業開始前**
   - 対象バージョンの `releasenote/v<バージョン>-plan.md` を読み込み、要求仕様・対象範囲・制約条件を確認すること。
   - 作業開始時に、plan ファイルのフロントマターにある `status` を `in-progress` に更新すること。

3. **作業中**
   - plan に記載された「変更対象の想定範囲」を逸脱する過度なリファクタリングやファイル変更を行わないこと。
   - 要求仕様の項目を1つ実装完了するごとに、該当チェックボックスを `- [x]` に更新すること。

4. **作業完了時**
   - plan の「完了の受け入れ基準」をすべて満たしていることを確認すること。
   - スキル（`create-releasenote`）を実行して `releasenote/v<バージョン>-released.md` を生成すること。
   - 最終的に plan ファイルの `status` を `completed` に更新すること。
