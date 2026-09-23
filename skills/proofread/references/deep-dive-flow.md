# 親側対応フロー (Stage 1 → 1.5 → 2 → 3)

proofreader subagent から return を受け取ったら、 以下の 4 stage を順に処理する｡

## Stage 1: dry-run 提示

return JSON を以下フォーマットで表示｡

```
✓ proofreader 完了 ({file_path})

[Tier 1 自動適用: {tier1_applied 件数}]
  - {rule}: {count} 件 (line {lines})
  ...

[Tier 2 提案: {tier2_proposals 件数}]
  [2-1] line {line} {category}
        現: {current}
        案: {suggestion}
  ...

[Tier 3 警告: {tier3_warnings 件数}]
  [3-1] line {line} ⚠ {category}
        抜粋: {excerpt}
  ...

[skipped/failures] (あれば)
  - {skill}: {reason}

→ 次のどれかで指示する:
   - "all"             Tier 2 全件適用
   - "2-1 2-3"         ID 指定で複数適用
   - "skip"            今回は何も apply しない
   - "detail 2-1"      Tier 2 の周辺 5 行を親が Read で表示 (subagent 不要)
   - "explore 3-1"     Tier 3 を deep-dive (Stage 1.5 へ)
```

## Stage 1.5: Tier 3 deep-dive (任意, ループ可)

ユーザーが `explore 3-N` 指定時、 proofreader subagent を `mode="deep-dive"` で再 dispatch する｡

引数:
- file_path: 同じ
- medium: 同じ
- mode: "deep-dive"
- target_warning_id: "3-N"
- prior_findings: 初回 scan の `tier3_warnings` 配列

return を以下フォーマットで表示:

```
[3-N deep-dive]

該当箇所 (line {line} ± 20):
{context}

該当規範:
  skill: {rule_citation.skill}
  section: {rule_citation.section}
  rule: {rule_citation.rule}
  example: {rule_citation.example}

診断: {diagnosis}

修正方向:
  1. {fix_directions[0]}
  2. {fix_directions[1]}
  ...

{promotable_to_tier2 ? "→ promote {warning_id}: Tier 2 に格上げして適用候補に追加" : "→ promote 不可: 修正方向が複数で絞れない"}

→ 次のどれかで指示する:
   - "promote 3-N"     Tier 2 に格上げして Stage 1 に戻る (promotable_to_tier2=true のみ)
   - "ignore 3-N"      警告を対応済みマークして Stage 1 に戻る (セッション内のみ保持)
   - "explore 3-M"     別の警告を deep-dive (ループ)
   - "next"            deep-dive 終了、 Stage 1 に戻る
```

`promote 3-N` の場合:
- `tier2_proposals` に新規 ID (例: `2-N+`) を追加
- ユーザーには「promote 完了｡ Stage 1 に戻ります」 と表示｡ Stage 1 リストを再表示

`ignore 3-N` の場合:
- 親の内部状態で「ignore 済み」フラグを立てる｡ Stage 3 報告で「ignored」 として扱う
- Stage 1 リストでは ignore 済み警告に `[ignored]` マーク付与

## Stage 2: ユーザー応答受領 (apply 対象指定)

ユーザーから受け取れる応答:

| 応答 | 解釈 |
|------|------|
| `all` | Tier 2 全件を apply 対象に |
| `2-1 2-3` | 指定 ID のみ apply 対象に |
| `skip` | 何も apply しない (Stage 3 で 0 件報告) |

パース失敗 (範囲外 ID 等) は `Error: ID "2-99" は範囲外｡ もう一度指示してください` と返して再入力を求める｡

## Stage 3: apply 実行 + 結果報告

apply 対象 ID を順番に処理｡ 各 ID に対し `Edit` ツールで suggestion を current と置換｡

報告フォーマット:

```
✓ Tier 2 適用完了 ({apply 件数} 件)
  - line {line} {category} → apply
  ...

skipped:
  - line {line} {category} → ユーザー指定でスキップ
  ...

[Tier 3 警告: {未対応件数} 件 (未対応)]
  ⚠ line {line} {category}
  ...
  (ignore マーク済み: N 件)
```

Stage 3 完了後、 親はユーザー応答待ちに戻る (proofread skill 終了)｡
