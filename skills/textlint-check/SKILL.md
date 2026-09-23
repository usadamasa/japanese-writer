---
name: textlint-check
description: >-
  textlint CLI による markdown の機械点検｡ ai っぽい表現と日本語悪文を検出・自動修正し、
  半角句読点や英文略号といった表記ルールも prh で強制する｡
  文書の種類ごとにルールを切り替えられるよう、呼び出し元が config_root で `.textlintrc.json` を指定する｡
  docs 系リポジトリやノートの .md を編集した後、および文面を出力する直前に使う｡
  Slack 短文には適用しない｡
---

# textlint-check

markdown を textlint で機械点検し、表記ゆれだけ autofix を適用したうえで残った指摘を返すスキル｡

[[japanese-tech-writing]] の規範のうち、 機械検出可能な項目を textlint preset と prh ルールで検出する｡

実行には `Bash` が要る｡ `Bash` を持たない subagent からは呼べない｡
[[proofread]] では親コンテキストが本スキルを実行し、結果 JSON のパスを proofreader subagent へ渡す｡

## 入力

呼び出し側から以下を受け取る｡

| 入力形態 | 必須 | 説明 |
|---------|------|------|
| `file_path` | A or B | 点検対象の `.md` ファイル絶対パス｡autofix も同ファイルに対して行う |
| `text` | A or B | 点検対象のテキスト本体｡ファイル化はスキル内で行う |
| `config_root` | 任意 | `.textlintrc.json` を置いてあるディレクトリ｡未指定なら同梱の `configs/base.textlintrc.json` |
| `tmp_dir` | 任意 | 結果 JSON と一時ファイルの置き場｡未指定なら `${config_root:-.}/tmp` |

文書の種類ごとにルールを切り替えたい場合、どの `.textlintrc.json` を効かせるかは呼び出し元が
`config_root` で指定する｡本スキルは文書の種類 (medium) の語彙を持たない｡

## 出力

呼び出し元に以下を返す｡

```json
{
  "lint_result_path": "/abs/path/to/tmp/textlint-result.json",
  "applied_fixes": 3,
  "remaining_issues": 2,
  "revised_text": "(text 入力の場合のみ)"
}
```

`lint_result_path` が指す JSON の中身:

```json
{
  "applied_fixes": [
    {
      "filePath": "/abs/path/to/doc.md",
      "ruleId": "prh",
      "line": 3,
      "column": 11,
      "message": "e.g.  => 例えば"
    }
  ],
  "remaining_issues": [
    {
      "filePath": "/abs/path/to/doc.md",
      "ruleId": "ja-technical-writing/sentence-length",
      "severity": 2,
      "line": 12,
      "column": 5,
      "message": "Line 12 sentence length(135) exceeds the maximum sentence length of 120."
    }
  ]
}
```

`applied_fixes` は既にファイルへ書き込み済みの修正｡`remaining_issues` は autofix で消えなかった指摘で、
**修正後のファイルに対する行番号**を持つ｡呼び出し元はそのまま該当行を参照できる｡

呼び出し元へは件数だけを返し、明細は `lint_result_path` から読ませる｡指摘が多い md でも
呼び出し元のコンテキストに全文が載らない｡

## config の解決

`config_root` を指定したときは `${config_root}/.textlintrc.json` を使う｡未指定なら
本スキル同梱の `${CLAUDE_SKILL_DIR}/configs/base.textlintrc.json` を使う｡

どちらの場合も、textlint CLI の `--config` に絶対パスで渡す｡cwd からの暗黙解決には頼らない｡
prh の `rulePaths` は `.textlintrc.json` の置き場所からの相対パスで解決されるため、
`config_root` を持ち出すときは prh ルールセットの相対パスが通ることを確認する｡

### autofix と検出で config を分ける

lint config の隣に `fix.textlintrc.json` があれば、`textlint-run.sh` は autofix pass だけを
そちらで走らせる｡同梱の config はこの形になっており、autofix が触るのは表記ゆれだけになる｡

語だけを置き換えると、同じ問題が形を変えて残る｡英文略号を和訳語に差し替えても読点と
語順は英語のままで、「あなた」を「読者」に替えても主語の立て方は変わらない｡そのため意味に
関わる指摘は autofix せず、検出だけして文ごと書き直させる｡

`config_root` に `fix.textlintrc.json` を置いていないリポジトリでは、従来どおり lint と
同じ config で autofix する｡分けたいリポジトリだけが 2 枚目を置けばよい｡

## ワークフロー

### Step 1: 一時ファイルの配置 (text 入力時のみ)

`tmp_dir` (未指定なら `${config_root:-.}/tmp`) に一時ファイルを作る｡

```bash
mkdir -p "$tmp_dir"
TMP_MD=$(mktemp "${tmp_dir}/textlint-XXXXXX.md")
```

テキスト本体は `Write` ツールで `$TMP_MD` へ書く｡シェル引数に通すと、markdown 中の
バッククォートや `$` が展開されて壊れる｡

### Step 2: textlint を実行する

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/textlint-run.sh" \
  --config "$config" \
  --output "${tmp_dir}/textlint-result.json" \
  "$target_file"
```

`$config` は「config の解決」節のとおり決める｡`$target_file` は `file_path` 入力ならそのパス、
`text` 入力なら Step 1 の `$TMP_MD`｡いずれも絶対パスで渡す｡

スクリプトは autofix を適用してから再 lint し、両方の結果を `--output` の JSON にまとめる｡
stdout には `applied_fixes=N remaining_issues=M output=PATH` のサマリ 1 行だけが出る｡

このスクリプトは textlint を `npx` 経由で起動し、ルールパッケージも `--package` で明示的に渡す｡
対象リポジトリに textlint が入っていなくても同じルールで点検できる｡

### Step 3: 結果を呼び出し元に返す

stdout のサマリから件数を読み、`lint_result_path` とあわせて返す｡

`text` 入力の場合は `Read` で `$TMP_MD` を読んで `revised_text` に載せ、読み終えたら
`rm "$TMP_MD"` で削除する (Step 1 で本スキルが作ったファイルだけを消す)｡`lint_result_path` の JSON は
呼び出し元が読み終えるまで残す｡

## 失敗時

`textlint-run.sh` が非 0 で終了したら、呼び出し元はそこで**エラー終了する**｡
機械点検を欠いたまま添削を完了扱いにしない｡

スクリプトが非 0 を返すのは次のときで、いずれも textlint の指摘とは無関係な前提条件の不足である｡
指摘が残っただけなら終了コードは 0 になる｡

| 終了理由 | 直し方 |
|---------|--------|
| `npx` / `jq` が無い | Node.js と jq を入れる |
| config が無い / 絶対パスでない | `config_root` を見直す |
| 点検対象が無い / 絶対パスでない | `file_path` を見直す |
| 出力先ディレクトリが無い | `tmp_dir` を作ってから呼ぶ |
| textlint が JSON を返さない | stderr に textlint の出力がそのまま出る｡多くは config の rule 解決失敗 |

## 同梱する config

| ファイル | 用途 |
|---------|------|
| `configs/base.textlintrc.json` | lint pass の config｡ai-writing preset + ja-technical-writing preset + prh 両方 |
| `configs/fix.textlintrc.json` | autofix pass の config｡表記ゆれのルールと `prh-notation.yml` だけ |
| `configs/prh-notation.yml` | 表記ゆれ ([[japanese-tech-writing]] Phase 2.7)｡autofix する |
| `configs/prh-prose.yml` | 語・句レベルの NG 表現 (Phase 2.4 / 2.9 / 2.10)｡検出のみ |

`base.textlintrc.json` の prh `rulePaths` は同じディレクトリの 2 つの yml を指す｡
テンプレートをリポジトリへ持ち出すときは、prh ルールセットも一緒に置くか、`rulePaths` を
実体への相対パスへ書き換える｡

`no-ai-list-formatting` は `disableBoldListItems: true` で太字リストアイテムの検査だけを切って
ある｡このルールはリストアイテムの「強調 + コロン」(`- **用語**: 説明`) を機械的な印象として
指摘するが、この形は [[japanese-tech-writing]] Phase 2.1 と 2.8 が用語定義の書き方として
指示している｡絵文字リストアイテムなど同ルールの他の検査は残す｡

`prh-prose.yml` へのルール追加は [[writing-feedback]] が行う｡NG パターンと良い書き換え例を
対で登録する｡禁止だけを並べると、回りくどい言い換えを生む｡

`textlint-run.sh` が `npx --package` で渡すルールパッケージと、config が参照する rule は
対応している必要がある｡config に別の rule を足すときはスクリプトの `TEXTLINT_PACKAGES` にも足す｡

## 半角句読点

句読点は常に半角の「｡ ､」を使う｡全角の「。」「、」は `prh-notation.yml` が半角へ autofix する｡
`ja-no-mixed-period` の `periodMark` も `"｡"` にしてあり、文末が句点以外で終わる文を検出する｡

新しい prh ルールは、置換先が文脈によらず一意に決まるなら `prh-notation.yml`、そうでなければ
`prh-prose.yml` へ足す｡

## 注意事項

- `config_root` の `.textlintrc.json` を編集すると、その配下の全 `.md` の lint に影響する｡変更前に影響範囲を確認する｡
- autofix は対象ファイルをその場で書き換える｡未コミットの変更を失いたくないなら、呼ぶ前に commit するか `git stash` する｡
- Slack 短文は textlint の対象外｡呼び出し自体を行わない｡
