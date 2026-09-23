# medium × 統合 skill 適用マトリクス

| medium | textlint-check | japanese-tech-writing | style_check (指定時) | sanitize-artifacts |
|--------|:--:|:--:|:--:|:--:|
| `wiki` | ✓ | ✓ | ✓ | ✓ |
| `note` | ✓ | △ (緩め)<sup>1</sup> | ✓ | ✓ |
| `docs` | ✓ | ✓ | ✗ | ✓ |
| `short-draft` | ✓ | ✗ <sup>2</sup> | ✗ | ✓ <sup>3</sup> |

`style_check` 列は `proofread` の `style_check` 引数で文体 skill が差し込まれたときだけ効く｡未指定なら
全 medium で skip し、 `skipped` に記録する｡

<sup>1</sup> note では japanese-tech-writing の Tier 2 提案 (段落分割等の構造変更提案) を抑制し、 Tier 3 警告のみ出力｡ 「思考のスケッチ段階」 を尊重するため｡

<sup>2</sup> short-draft は短文中心 (タスク本文・チケット説明) ｡ パラグラフライティング規範は適用対象外｡

<sup>3</sup> sanitize-artifacts は全 medium で適用する｡ 制作過程の漏出は文書の長さや格式ではなく制作の経緯で決まり、 短い下書きにも起こるため｡ 議事録・対話ログは会話の記録そのものが成果物なので、 本文がそれと判断できるときは skip して `skipped` に記録する｡

## domain_check の適用判定

`domain_check` が指定されていれば適用し、未指定なら skip して `skipped` に記録する｡

「本文にどの語が出たらそのドメインのチェックを適用するか」 という内容ベースの判定条件は
`domain_check.tiering_ref` が持つ｡本マトリクスは medium では on/off しない｡

## textlint-check への引き渡し

textlint-check を呼ぶのは親コンテキストで、proofreader subagent ではない (subagent は `Bash` を持たない)｡
親は `config_root` / `tmp_dir` をそのまま textlint-check へ渡し、返ってきた `lint_result_path` だけを
subagent へ引き渡す｡

textlint-check は medium の語彙を持たず、どの `.textlintrc.json` を効かせるかは `config_root` だけで決まる｡
`config_root` 未指定のときは textlint-check 同梱の `configs/base.textlintrc.json` が使われる｡

上の表の textlint-check 列は、親が Step 0 で機械点検を走らせるかどうかを示す｡どの medium でも走らせる｡
