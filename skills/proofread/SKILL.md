---
name: proofread
description: >-
  md ドラフト (wiki 草稿、 ADR/Design Doc、 探索ノート、 タスク本文の下書き) の総合添削｡
  機械点検・文章規範・制作過程の漏出のチェックを proofreader subagent に集約して
  別コンテキストで実行し、確度別に分類した修正を適用して報告する｡
  修正案が一意なものは問い合わせずに反映し、論証や事実確認が要るものだけ警告に留める｡
  ドメイン固有の用語・事実チェックは domain_check 引数で、書き手の文体は style_check 引数で差し込む｡
  文面を外部へ出力する直前、 .md を書き上げた直後に使う｡
  日誌・議事メモや個人 OSS の README には適用しない｡
---

# proofread

md ドラフトの総合添削スキル｡ proofreader subagent で別コンテキスト実行する｡

[[japanese-tech-writing]] の **Phase 3 self-check の subagent 側 (網羅視点)** に対応する｡
書き手の self-check (主観・気付き、 主コンテキストで実行) と本スキル (網羅・客観、 subagent 別コンテキスト) は重複が健全｡
書き手の主観が見落としたものを subagent が網羅的にスキャンして拾う｡

呼び出しタイミング:

1. 書き手が [[japanese-tech-writing]] の Phase 3 self-check の項目 1〜6 を頭で点検する
2. 通過したら、 self-check 項目 7 として本スキルを呼ぶ ([[japanese-tech-writing]] Phase 1.2 で判定した medium を渡す)
3. 本スキルは proofreader subagent を dispatch し、 Tier 1/2/3 で分類された結果を return する
4. 親 (呼び出し元) が Tier 2 を適用して報告する → ユーザーが必要なら Tier 3 を deep-dive

呼び出し元は、文書を生成・投稿するスキル (wiki ページ作成、タスク起票など) と
[[japanese-tech-writing]] の Phase 3.1 項目 7 である｡

## 入力

| 引数 | 必須 | 型 | 説明 |
|------|------|-----|------|
| `file_path` | 必須 | str | 添削対象 .md の絶対パス |
| `medium` | 必須 | enum | `"wiki"` / `"docs"` / `"note"` / `"short-draft"` |
| `domain_check` | 任意 | obj | ドメイン固有の用語・事実チェックの差し込み口 (下記) |
| `style_check` | 任意 | obj | 書き手の文体スキルの差し込み口 (下記) |
| `config_root` | 任意 | str | `.textlintrc.json` を置いてあるディレクトリ｡docs 系ならリポジトリ root｡未指定なら [[textlint-check]] 同梱の config |
| `tmp_dir` | 任意 | str | 一時ファイル置き場｡未指定なら `${config_root:-.}/tmp` |

### medium

| medium | 対象 |
|--------|------|
| `wiki` | wiki ページ・ADR・Design Doc の本文 |
| `docs` | docs 系リポジトリの `.md` (`config_root` にリポジトリ root を渡す) |
| `note` | 探索メモ・長文ノート (思考のスケッチ段階を尊重して規範を緩める) |
| `short-draft` | タスク本文・チケット説明などの短い下書き |

### domain_check

呼び出し元のドメイン (製品の用語表、組織の体制情報など) に固有のチェックを差し込む｡

```json
{
  "skill": "<ドメイン知識を持つ skill 名>",
  "tiering_ref": "<Tier 分類表のパス>"
}
```

- `skill`: proofreader subagent が `Skill` ツールで読み込む｡用語表・体制表などの正本｡
- `tiering_ref`: そのドメインの検出結果をどの Tier に割り当てるかを定義したファイル｡
  「どんな語が出たらチェックを適用するか」の判定条件も、この参照先が持つ｡
  絶対パス、または `skill` のデプロイ先ディレクトリ (`~/.claude/skills/<skill>/`) からの
  相対パスで指定する｡

未指定なら、ドメインチェックの step を skip し `skipped` に記録する｡汎用の機械点検・
文章規範チェックはそのまま走る｡

### style_check

書き手ごとの文体スキル (文書の種類ごとの語尾・句読点・絵文字の指紋) を差し込む｡

```json
{
  "skill": "<文体の指紋を持つ skill 名>"
}
```

- `skill`: proofreader subagent が `Skill` ツールで読み込む｡文書の種類 (Slack / wiki / blog など) ごとの
  文体ルールと例文を持つ skill を指す｡
- 適用する medium は `references/medium-matrix.md` の `style_check` 列に従う (`wiki` / `note` のみ)｡
- 検出結果の Tier は `references/tier-classification.md` の「style_check 由来」の項に従う｡

未指定なら、文体チェックの step を skip し `skipped` に記録する｡[[japanese-tech-writing]] の
文書の種類別マトリクスにある「個人文体」列も無視され、fofr 強化規範だけが効く｡

## 出力 (subagent からの return)

詳細は `${CLAUDE_PLUGIN_ROOT}/agents/proofreader.md` の「出力 (return value)」 セクションを参照｡
scan モードは `{tier1_applied, tier2_proposals, tier3_warnings, skipped, failures}` を含む JSON、
deep-dive モードは `{warning_id, context, rule_citation, diagnosis, fix_directions, promotable_to_tier2, tier2_proposal}` を含む JSON を返す｡

## ワークフロー

### Step 0: textlint を親コンテキストで実行する

`proofreader` subagent は `Bash` を持たず、textlint を起動できない｡機械点検は親が先に済ませる｡

[[textlint-check]] を `Skill` ツールで `japanese-writer:textlint-check` として invoke し、`file_path` / `config_root` / `tmp_dir` をそのまま渡す｡
返ってきた `lint_result_path` を Step 1 で subagent へ引き渡す｡

この時点で textlint の autofix は `file_path` へ適用済みになる｡以降 subagent が読むのは修正後のファイルで、
`remaining_issues` の行番号もそれに揃っている｡

`textlint-check` が失敗したら、そこで **error 返却して終了する** (F1)｡ md ファイルは無変更のまま残る｡

### Step 1: proofreader subagent dispatch (mode="scan")

Agent ツールで `proofreader` subagent を `subagent_type: japanese-writer:proofreader` として 1 回 dispatch する｡ 引数: 上記 「入力」 セクションのうち
`file_path` / `medium` / `domain_check` / `style_check` と、Step 0 で得た `lint_result_path` を渡す｡
`config_root` と `tmp_dir` は親が textlint 実行に使うもので、subagent へは渡さない｡

### Step 2: Stage 1 で Tier 2 を Edit 適用 + 報告

Tier 2 は問い合わせずに全件適用し、 適用した前後と Tier 3 警告を報告する｡ ユーザーへ apply 対象を
選ばせない｡ 戻したい ID はユーザーが報告後に `revert 2-N` で指示する｡ 詳細は `references/deep-dive-flow.md`｡

修正は語の差し替えで済ませない｡指摘された語を含む文をまるごと書き直す｡語だけを替えると、
同じ問題が形を変えて残る｡

報告まで終えたら `rm "$lint_result_path"` で結果 JSON を片付ける｡消すのは Step 0 で本スキルが
作ったファイルだけで、`tmp_dir` そのものは消さない｡ ここで proofread skill は終了する｡

### Step 3: Stage 1.5 (任意, ユーザー指示時のみ)

ユーザーが `explore` を指示した場合のみ実行｡ proofreader を `mode="deep-dive"` で再 dispatch｡
再 dispatch 時の引数: file_path / medium / mode="deep-dive" / target_warning_id (ユーザー指定の Tier 3 ID) / prior_findings (初回 scan の `tier3_warnings` 配列)｡ 詳細は `references/deep-dive-flow.md#stage-15` 参照｡

### Step 4: 繰り返す指摘をルールへ移す

同じ種類の指摘を 2 回以上受けたら、[[writing-feedback]] でルールとして登録する｡
登録しない限り、次の文書でも同じ指摘が出る｡

## 完了ゲートとの関係

同梱の Stop hook `writing-gate` が、セッションを閉じるときに編集した `.md` を点検する｡
本スキルとは見る対象が違う｡

| | writing-gate (Stop hook) | proofread (本スキル) |
|----|----|----|
| 発火 | 自動 (完了時) | 明示的に呼ぶ |
| 対象 | 文書全体の集計と逐語の漏出 | 表記・語彙・論証・文体 (style_check)・ドメイン用語 (domain_check) |
| 重さ | Go 単体で即座に終わる | textlint + subagent |
| 位置づけ | 素通りを防ぐ最低線 | 外部へ出す前の本検査 |

ゲートを通ったことは本スキルを済ませたことにはならない｡

## medium 別マトリクス

詳細は `references/medium-matrix.md`｡

## Tier 分類

詳細は `references/tier-classification.md`｡

## 失敗時挙動

詳細は `references/failure-modes.md`｡
