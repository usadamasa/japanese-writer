# 親側対応フロー (Stage 1 → 1.5)

proofreader subagent から return を受け取ったら､ 以下の stage を順に処理する｡
Stage 1 でユーザーへの問い合わせは挟まない｡ Tier 2 は親の判断で適用し､ 結果を報告する｡

## Stage 1: Tier 2 適用 + 報告

`tier2_proposals` を ID 順に処理し､ 各 ID に対し `Edit` ツールで `suggestion` を `current` と置換する｡
`suggestion` が空､ または `current` が本文に見つからない ID は適用せず､ 報告の「未適用」に理由付きで載せる｡
Edit の失敗は F4 と同じ扱いにする (以降を止めず､ その ID だけ未適用に回す)｡

適用が終わったら､ 以下フォーマットで報告する｡

```
✓ proofreader 完了 ({file_path})

[Tier 1 自動適用: {tier1_applied 件数}]
  - {rule}: {count} 件 (line {lines})
  ...

[Tier 2 適用: {適用件数} 件]
  [2-1] line {line} {category}
        前: {current}
        後: {suggestion}
  ...

[Tier 2 未適用: {件数} 件] (あれば)
  [2-4] line {line} {category} → {理由}

[Tier 3 警告: {tier3_warnings 件数}]
  [3-1] line {line} ⚠ {category}
        抜粋: {excerpt}
  ...

[skipped/failures] (あれば)
  - {skill}: {reason}
```

報告はここで終える｡ 選択肢の一覧や「どれを適用するか」の問いを末尾に付けず､ proofread skill を終了する｡

ユーザーが報告を読んだ後に出せる指示は次の 3 つ｡ 親から促さず､ 出てきたときだけ受ける｡

| 指示 | 動作 |
|------|------|
| `revert 2-N` | 該当 ID の `suggestion` を `current` へ `Edit` で戻し､ `✓ 2-N を戻した` と報告する |
| `detail 2-N` | Tier 2 の周辺 5 行を親が Read で表示する (subagent 不要) |
| `explore 3-N` | Tier 3 を deep-dive する (Stage 1.5 へ) |

同じ種類の revert を 2 回以上受けたら､ [[writing-feedback]] で規範側を直す候補として扱う｡

## Stage 1.5: Tier 3 deep-dive (任意, ループ可)

ユーザーが `explore 3-N` 指定時､ proofreader subagent を `mode="deep-dive"` で再 dispatch する｡

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

{promotable_to_tier2 ? "→ promote {warning_id}: tier2_proposal をそのまま適用できる" : "→ promote 不可: 修正方向が複数で絞れない"}

必要なら次で指示する:
   - "promote 3-N"     tier2_proposal を Edit で適用する (promotable_to_tier2=true のみ)
   - "ignore 3-N"      警告を対応済みマークする (セッション内のみ保持)
   - "explore 3-M"     別の警告を deep-dive (ループ)
```

`promote 3-N` の場合:
- `tier2_proposal` の `suggestion` を `current` と `Edit` で置換し､ 新規 ID (例: `2-N+`) を振って
  `✓ 2-N+ (3-N から昇格) line {line} を適用した` と報告する
- 以降の `revert` はこの新規 ID で受ける

`ignore 3-N` の場合:
- 親の内部状態で「ignore 済み」フラグを立てる｡ 以降の報告で `[ignored]` マークを付ける
