<!-- writing-gate-disable -->

<!--
  このファイルは検出パターンそのものを例示するため、全ルールを無効化している｡
  japanese-tech-writing が textlint に対して同じことをしているのと同型｡
-->

# writing-gate

markdown の文章品質を機械点検する CLI｡Stop hook から起動して、そのセッションで
書いた `.md` に重大な指摘が残っていれば完了をブロックする｡

設計・ルール・運用はこのファイルにある｡

## なぜ要るか

文章規範を書き手への指示だけで守らせると、長い会話のなかで薄まる｡グローバル
`CLAUDE.md` にも skill にも規範は書いてあるが、20 ターン先で効いている保証は無い｡
モデルが変われば癖も変わる｡

そこで生成物そのものを見る層を執筆の後段に置く｡指示を強めるのではなく、
出来上がったものを機械で点検する｡

## 何を見るか

textlint が拾えないものだけを見る｡語彙・表記の点検は `proofread`
(内側で textlint) の担当で、ここでは扱わない｡

### 文書全体の集計

| ルール ID | 検出するもの | 既定の severity |
|----|----|----|
| `sentence-ending-repeat` | 同じ語尾の文が段落内で 3 文以上続く | error |
| `style-mix` | である体とですます体が混ざっている | error |
| `metaphor-repeat` | 比喩の目印が 3 箇所以上ある | warn |

判定の範囲を絞ってある｡

- `sentence-ending-repeat` は本文の段落だけを見る｡箇条書きの語尾が揃うのは
  規範上むしろ推奨されるため (`japanese-tech-writing` Phase 2.8)｡
- 段落 (空行と非本文行で区切る) をまたいだ連続は数えない｡見出しを挟んだ「連続」は
  読み手に連続として届かない｡
- 語尾がひらがなで終わらない行は数えない｡語彙の列挙や記号の並びを拾ってしまうため｡
- `style-mix` は引用・見出し・表を除く｡少数派の文を最大 5 件まで挙げる｡
- 丁寧語の「ます」は連用形 (い段・え段) に付く｡直前がそれ以外なら本動詞なので
  ですます体として数えない (「済ます」「励ます」)｡

### 漏出フレーズ

| ルール ID | 検出するもの | 既定の severity |
|----|----|----|
| `process-leak` | 制作過程の痕跡 | error |
| `pink-elephant` | 却下・削除した案の痕跡 | error |
| `inline-enumeration` | 括弧内の「A / B / C」や「A + B + C」で構成要素を並べた説明文 | warn |

パターンの正本は `sanitize-artifacts` skill の「検出対象」表｡こちらはそのうち
逐語で拾えるものを機械化したもの｡意味・注意配分・視覚の層は機械では拾えないため、
同 skill の点検が別に要る｡

引用行 (`>`) は他人の文なので対象外｡コードブロック・インラインコード・
フロントマター・HTML コメントは解析前に落とす｡

## 使い方

```sh
writing-gate scan [--format json|text] [--rules PATH] FILE...
writing-gate gate --transcript PATH [--config PATH] [--rules PATH] [--max-files N]
```

`scan` は指定したファイルを点検して結果を出す｡`gate` は transcript から
編集された `.md` を拾って点検し、Stop hook が解釈する JSON を出す｡

`gate` が block を返すのは severity=error があるときだけ｡warn は報告に載るが
完了は止めない｡

## ルールの置き場

境界は「prh が `expected` を必須とするか」で決まる｡置換先を書けない漏出フレーズは
prh に載せられないため、ここが分担の線になる｡

| 置き場 | 扱うもの | 実行経路 |
|----|----|----|
| textlint-check skill の `configs/prh-notation.yml` | 表記ゆれ｡autofix する | textlint (fix pass) |
| textlint-check skill の `configs/prh-prose.yml` | 語・句レベルの NG 表現｡検出のみ | textlint (lint pass) |
| writing-gate の `internal/rules/rules.json` | 置換先を持たない漏出フレーズと集計の閾値 | Stop hook |

ルールの追加は `writing-feedback` skill が行う｡追加先の判定も同 skill が持つ｡

### rules.json の形

```json
{
  "version": 1,
  "phrase_rules": [
    {
      "id": "process-leak",
      "severity": "error",
      "message": "何が問題か",
      "guidance": "どう直すか",
      "patterns": ["部分一致で拾う文字列"],
      "regexps": ["Go の正規表現"]
    }
  ],
  "metaphor_markers": ["いわば"],
  "aggregates": {
    "sentence_ending_repeat": {
      "enabled": true, "severity": "error",
      "min_run": 3, "suffix_runes": 3, "min_sentence_runes": 6
    },
    "style_mix": { "enabled": true, "severity": "error", "min_sentences": 5, "max_examples": 5 },
    "metaphor_repeat": { "enabled": true, "severity": "warn", "min_count": 3 }
  }
}
```

このファイルは `go:embed` でバイナリへ焼き込む｡変更したら再ビルド (「検証」節) が要る｡

### 端末固有の追加ルール

`~/.claude/writing-gate-rules.json` を置くと、埋め込みルールへマージする｡
リポジトリには入れない｡端末で試して定着したら `rules.json` へ移す｡

マージの規則:

- 同じ `id` の `phrase_rules` は `patterns` と `regexps` を追記する｡
  `message` / `guidance` / `severity` は非空なら差し替える｡
- 未知の `id` は追加する｡
- `metaphor_markers` は追記する｡
- `aggregates` は現れたブロックだけ差し替える｡

## 除外と無効化

### ファイル単位の無効化

md の先頭に HTML コメントを置く｡`writing-gate-disable` に続けてルール ID を
空白区切りで書くと、そのルールだけ無効になる｡ID を書かなければ全ルールが無効になる｡

規範そのものを説明する md は違反例を本文に持つため、この無効化が要る｡
`japanese-tech-writing` が textlint に対して同じことをしている｡

コメントの検索はコードブロックの中も含めて行う｡書式を例示するだけで無効化が
効いてしまうので、説明用の md はその前提で書く｡

### パス単位の除外

既定で外すもの:

```text
**/tmp/**  **/node_modules/**  **/.git/**
**/.claude/projects/**  **/.claude/worktrees/**
**/obsidian/**  **/daily/**  **/diary/**  **/minutes/**
**/PLAN.md  **/CHANGELOG.md
```

会話の記録そのものが成果物である面 (議事録・日誌・対話ログ) と、生成物・作業用
ファイルを外してある｡`sanitize-artifacts` の「適用しない面」に揃えた｡

`~/.claude/writing-gate.json` で変えられる｡リポジトリには入れない｡

```json
{ "exclude_extra": ["**/scratch/**"] }
```

`exclude_extra` は既定へ追記する｡`exclude` を書くと既定を置き換える｡

glob は `**/dir/**` (パスに `/dir/` を含む) と `**/name.md` (basename 一致) の
2 つの形だけを自前で解釈し、それ以外は `filepath.Match` へ渡す｡Go の標準ライブラリは
`**` を扱えないため｡

## hook との配線

```text
Stop hook (hooks/stop-writing-gate.sh)
  └─ <hooks の隣の bin>/writing-gate gate --transcript <path>
       ├─ transcript から Write/Edit/MultiEdit/NotebookEdit の .md を拾う
       ├─ 除外と上限 (既定 20 件) を当てる
       ├─ 点検する
       └─ error があれば {"decision":"block","reason":...} を出す
```

hook はバイナリを自分のディレクトリからの相対 (`../bin/writing-gate`) で起動し、PATH は見ない｡
hook が継承する PATH は起動元の環境で変わるため｡

hook 側の判断:

- `stop_hook_active` が true なら判定しない｡同じ指摘で往復させないため｡
- バイナリが無ければ block して、その事実を伝える｡黙って通すとゲートが丸ごと
  無効なまま素通りする｡
- 点検の実行そのものが失敗したら stderr へ出して通す｡文章の不備ではないため｡

npx を経由しない｡textlint を通すと完了のたびに数秒待たされる｡

## 検証

```sh
go test ./...                      # 単体テスト
go build -o <hooks の隣の bin>/ ./... # 埋め込みルールを反映する

# リポジトリ全体へ当てて誤検出を見る
git ls-files '*.md' > ./tmp/l.txt
xargs writing-gate scan --format json < ./tmp/l.txt | jq '{errors,warnings}'
```

ルールを足したら、直したかった文を拾うことと、既存の md で誤検出が増えないことの
両方を見る｡手順は `writing-feedback` skill の Step 3 と Step 4｡

## 限界

- 語彙・表記は見ない｡外部へ出す文書なら `proofread` を続けて呼ぶ｡
  ゲートを通ったことは添削を済ませたことにはならない｡
- 漏出の検出は逐語だけ｡言い換え・上位語・図やラベルへの漏れは拾えない｡
- 文体の判定は文末の形だけを見る｡体言止めはどちらにも数えない｡
- 判定は日本語の文にしか当てない｡英文は語尾と文体の判定から外れる｡
