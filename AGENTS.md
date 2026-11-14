# AGENTS.md

## issue と PR のラベル

種類・優先度・epic の付け方は `labeling-github-issues` skill に従う。機能の area はこの表から選ぶ。

| area | 含むもの |
| ---- | ---- |
| `area: writing-gate` | Stop hook の writing-gate (Go の CLI、phrase rule、gate-off / gate-on) |
| `area: textlint` | textlint-check と同梱の textlint config、prh のルール |
| `area: proofread` | proofread skill、proofreader subagent、sanitize-artifacts |
| `area: tech-writing` | japanese-tech-writing の文章規範 |
| `area: feedback` | writing-feedback、writing-retrospective、crit の approve 後の prompt |
| `area: setup` | setup skill、SessionStart hook による writing-gate のビルドと data ディレクトリの準備 |
| `area: dev` | CI、lint、テスト、Taskfile、flake、保守用の skill と docs |

移設を取りやめた agy-delegation (#19) には area を付けない。

- PR の本文に、閉じる issue を `Closes #N` で 1 行ずつ書く。pr-labels workflow (gh-manage の reusable workflow の呼び出し) が issue の種類と area のラベルを PR へ写す。
- issue を閉じない PR は、`gh pr create --label` で種類と area のラベルを付けて作る。選び方は `labeling-github-issues` skill。
