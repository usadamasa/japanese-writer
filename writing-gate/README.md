<!-- writing-gate-disable -->

<!--
  このファイルは検出パターンそのものを例示するため、全ルールを無効化している。
  japanese-tech-writing が textlint に対して同じことをしているのと同型。
-->

# writing-gate

markdown の文章品質を機械点検する CLI。Stop hook から起動して、そのセッションで
書いた `.md` に重大な指摘が残っていれば完了をブロックする。

設計と運用はこのファイルに、ルールごとの詳細は `docs/rules/` にある。

## なぜ要るか

文章規範を書き手への指示だけで守らせると、長い会話のなかで薄まる。グローバル
`CLAUDE.md` にも skill にも規範は書いてあるが、20 ターン先で効いている保証は無い。
モデルが変われば癖も変わる。

そこで生成物そのものを見る層を執筆の後段に置く。指示を強めるのではなく、
出来上がったものを機械で点検する。

## 何を見るか

textlint が拾えないものだけを見る。語彙・表記の点検は `proofread`
(内側で textlint) の担当で、ここでは扱わない。

集計と表示のルールは判定の範囲と設定を `docs/rules/<ルール ID>.md` に置く。
漏出フレーズは `internal/rules/rules.json` が正本で、共通の判定の範囲を [`docs/rules/phrase-rules.md`](docs/rules/phrase-rules.md) に置く。

### 文書全体の集計

| ルール ID | 検出するもの | 既定の severity |
|----|----|----|
| [`sentence-ending-repeat`](docs/rules/sentence-ending-repeat.md) | 同じ語尾の文が段落内で 3 文以上続く | error |
| [`style-mix`](docs/rules/style-mix.md) | である体とですます体が混ざっている | error |
| [`metaphor-repeat`](docs/rules/metaphor-repeat.md) | 比喩の目印が 3 箇所以上ある | warn |

### 漏出フレーズ

| ルール ID | 検出するもの | 既定の severity |
|----|----|----|
| `process-leak` | 制作過程の痕跡 | error |
| `pink-elephant` | 却下・削除した案の痕跡 | error |
| `inline-enumeration` | 括弧内の「A / B / C」や読点で 4 つ以上並べた列挙、「A + B + C」で構成要素を並べた説明文 | warn |

判定の範囲と設定のキーは [`docs/rules/phrase-rules.md`](docs/rules/phrase-rules.md) にある。

### Markdown の表示

| ルール ID | 検出するもの | 既定の severity |
|----|----|----|
| [`bold-not-rendered`](docs/rules/bold-not-rendered.md) | GitHub などで太字として表示されず、`**` がそのまま見える書き方 | error |

## 使い方

```sh
writing-gate scan [--format json|text] [--rules PATH] FILE...
writing-gate gate --transcript PATH [--config PATH] [--rules PATH] [--max-files N]
```

`scan` は指定したファイルを点検して結果を出す。`gate` は transcript から
編集された `.md` を拾って点検し、Stop hook が解釈する JSON を出す。

`gate` が block を返すのは severity=error があるときだけ。warn は報告に載るが
完了は止めない。

### gate が指摘を出す範囲

`gate` はそのセッションで書いた行にだけ指摘を出す。既存の文書に手を入れたとき、
過去の行の指摘を今回の不備として出さないため。ファイル全体を見たいときは `scan` を使う。

- `Write` はファイル全体を書いたものとして扱う
- `Edit` / `MultiEdit` は `new_string` が今の本文に現れる行を書いた行とする。
  後の編集で消えた断片は数えない。同じ断片が複数箇所にあれば全部を数える
- `new_string` のうち `old_string` にそのまま含まれる行は書いた行に数えない。一意にするために添えた
  前後の行で、その行に元からある指摘を今回の不備として出さないため。空行も数えない
- 書いた本文を取り出せないツール (`NotebookEdit` など) はファイル全体を書いたものとして扱う

集計ルールは数える範囲と指摘を出す範囲が違う。数えるのは文書全体で、指摘は書いた行に
かかるものだけを出す。

- `sentence-ending-repeat` は連続のどれかの文を書いていれば出す。行番号は連続の先頭を指すので、
  書いていない行を指すことがある
- `style-mix` は多数派を文書全体で決め、書いた行の少数派の文だけを例に挙げる
- `metaphor-repeat` は文書全体で数え、書いた行に目印があれば出す

block の reason には、原因になったルール ID を入れた無効化コメントの書式と、セッションの間だけ
止める `/japanese-writer:gate-off` の案内を載せる。

### 同じ指摘は 1 回だけ出す

`gate` は同じ指摘でセッションを 2 度止めない。指摘を受けてもその文書では直さないと決めたとき、
ユーザーがメッセージを送るたびに同じ指摘で止まるのを避けるため。

block の reason の各指摘には ID (`(wg:3f2a9c1e)` の形) を付ける。ID はファイル・ルール ID・該当文から
作るので、上の行を編集して行番号がずれても同じ指摘は同じ ID になる。block は transcript に
残るので、次の `gate` は transcript にある ID の指摘を判定から外す。外した指摘は reason にも載らない。

`sentence-ending-repeat` の該当文は語尾だけなので、同じ語尾の連続は同じファイルの中で 1 度しか出ない。

### セッションの間だけ止める

ユーザーが `/japanese-writer:gate-off` を打つと、そのセッションの `gate` は点検せずに通す。
`/japanese-writer:gate-on` で戻る。後に打った方が効く。

状態はファイルにも設定にも持たず、その Stop が渡してきた transcript だけから読む。だから効くのは
同じセッション (同じ transcript) の間だけで、セッションが変われば初期状態 (点検する) に戻る。

無効化コメントと `exclude_extra` はファイル単位で恒久的に効くので、他人のファイルや共有のルールファイル
には入れにくい。gate-off はファイルにも設定にも触れない。

ユーザーが打った slash command は transcript の user エントリに `<command-name>` として残り、`gate` は
それだけを読む。Claude が Skill ツールで同じ skill を呼んでも `tool_use` にしか残らないので効かない。
止める判断をユーザーだけが持つようにするため。skill 側も `disable-model-invocation: true` で
Claude からは呼べなくしてある。

## ルールの置き場

境界は「prh が `expected` を必須とするか」で決まる。置換先を書けない漏出フレーズは
prh に載せられないため、ここが分担の線になる。

| 置き場 | 扱うもの | 実行経路 |
|----|----|----|
| textlint-check skill の `configs/prh-notation.yml` | 表記ゆれ。autofix する | textlint (fix pass) |
| textlint-check skill の `configs/prh-prose.yml` | 語・句レベルの NG 表現。検出のみ | textlint (lint pass) |
| writing-gate の `internal/rules/rules.json` | 置換先を持たない漏出フレーズと集計の閾値 | Stop hook |

ルールの追加は `writing-feedback` skill が行う。追加先の判定も同 skill が持つ。

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
      "regexps": ["Go の正規表現"],
      "kinds": ["heading"]
    }
  ],
  "metaphor_markers": ["いわば"],
  "aggregates": {
    "sentence_ending_repeat": {
      "enabled": true, "severity": "error",
      "min_run": 3, "suffix_runes": 3, "min_sentence_runes": 6
    },
    "style_mix": { "enabled": true, "severity": "error", "min_sentences": 5, "max_examples": 5 },
    "metaphor_repeat": { "enabled": true, "severity": "warn", "min_count": 3 },
    "bold_not_rendered": { "enabled": true, "severity": "error" }
  }
}
```

このファイルは `go:embed` でバイナリへ焼き込む。変更したら再ビルド (「検証」節) が要る。

### 端末固有の追加ルール

`~/.claude/writing-gate-rules.json` を置くと、埋め込みルールへマージする。
リポジトリには入れない。端末で試して定着したら `rules.json` へ移す。

マージの規則:

- 同じ `id` の `phrase_rules` は `patterns` と `regexps` を追記する。
  `message` / `guidance` / `severity` は非空なら差し替える。
- 未知の `id` は追加する。
- `kinds` は非空なら差し替える。省略時は quote 以外の全行 (`body` / `heading` / `list` / `table`) に当たる。
- `metaphor_markers` は追記する。
- `aggregates` は現れたブロックだけ差し替える。

## 除外と無効化

セッションの間だけ止めるなら、ファイルにも設定にも触れない `/japanese-writer:gate-off`
(「セッションの間だけ止める」節) を使う。

### ファイル単位の無効化

md の先頭に HTML コメントを置く。`writing-gate-disable` に続けてルール ID を
空白区切りで書くと、そのルールだけ無効になる。ID を書かなければ全ルールが無効になる。

規範そのものを説明する md は違反例を本文に持つため、この無効化が要る。
`japanese-tech-writing` が textlint に対して同じことをしている。

コメントの検索はコードブロックの中も含めて行う。書式を例示するだけで無効化が
効いてしまうので、説明用の md はその前提で書く。

### パス単位の除外

既定で外すもの:

```text
**/tmp/**  **/node_modules/**  **/.git/**
**/.claude/projects/**  **/.claude/worktrees/**
**/obsidian/**  **/daily/**  **/diary/**  **/minutes/**
**/PLAN.md  **/CHANGELOG.md
```

会話の記録そのものが成果物である面 (議事録・日誌・対話ログ) と、生成物・作業用
ファイルを外してある。`sanitize-artifacts` を適用しない面に揃えた。

`~/.claude/writing-gate.json` で変えられる。リポジトリには入れない。

```json
{ "exclude_extra": ["**/scratch/**"] }
```

`exclude_extra` は既定へ追記する。`exclude` を書くと既定を置き換える。

glob は `**/dir/**` (パスに `/dir/` を含む) と `**/name.md` (basename 一致) の
2 つの形だけを自前で解釈し、それ以外は `filepath.Match` へ渡す。Go の標準ライブラリは
`**` を扱えないため。

## hook との配線

```text
SessionStart hook (hooks/session-start-prepare.sh)
  └─ scripts/prepare-data.sh --if-stale
       └─ Go のソースと rules.json の cksum が前回と違えば ${CLAUDE_PLUGIN_DATA}/bin/ へビルドし直す

Stop hook (hooks/stop-writing-gate.sh)
  └─ ${CLAUDE_PLUGIN_DATA}/bin/writing-gate gate --transcript <path>
       ├─ transcript から Write/Edit/MultiEdit/NotebookEdit の .md、前回の block の指摘 ID、
       │  ユーザーが打った gate-off / gate-on を拾う
       ├─ gate-off が効いていれば点検せずに通す
       ├─ 除外と上限 (既定 20 件) を当てる
       ├─ 点検して、書いた行にかかる指摘だけを残し、前回に載せた指摘を外す
       └─ error があれば {"decision":"block","reason":...} を出す
```

hook はバイナリを plugin の data ディレクトリから絶対パスで起動し、PATH は見ない。
hook が継承する PATH は起動元の環境で変わるため。plugin root (`${CLAUDE_PLUGIN_ROOT}`) は
版ごとに別のディレクトリになり update で入れ替わるので、ビルドの置き場には使わない。

hook 側の判断:

- `stop_hook_active` が true なら判定しない。同じ指摘で往復させないため。
- バイナリが無ければ block して、その事実を伝える。黙って通すとゲートが丸ごと
  無効なまま素通りする。
- 点検の実行そのものが失敗したら stderr へ出して通す。文章の不備ではないため。

npx を経由しない。textlint を通すと完了のたびに数秒待たされる。

## 検証

```sh
go test ./...  # 単体テスト
task build     # 埋め込みルールを反映した bin/writing-gate を作る

# リポジトリ全体へ当てて誤検出を見る
git ls-files '*.md' > ./tmp/l.txt
xargs ./bin/writing-gate scan --format json < ./tmp/l.txt | jq '{errors,warnings}'
```

ルールを足したら、直したかった文を拾うことと、既存の md で誤検出が増えないことの
両方を見る。手順は `writing-feedback` skill の Step 3 と Step 4。

## 限界

- 語彙・表記は見ない。外部へ出す文書なら `proofread` を続けて呼ぶ。
  ゲートを通ったことは添削を済ませたことにはならない。
- 漏出の検出は逐語だけ。言い換え・上位語・図やラベルへの漏れは拾えない。
- 文体の判定は文末の形だけを見る。体言止めはどちらにも数えない。
- 判定は日本語の文にしか当てない。英文は語尾と文体の判定から外れる。
## 出典

- 執筆後の別の層で機械点検し、検出語の置換ではなく文ごと書き直させる設計は
  <https://x.com/yugen_matuni/status/2088251220452679951> を翻案した。
- 却下案の痕跡を漏出として拾う考え方は <https://x.com/Kashiko_AIart/status/2091137586991645101> を翻案した。
- `bold-not-rendered` の判定と直し方の案は、nanaism の yomiyasu <https://github.com/nanaism/yomiyasu> (MIT) の
  `scripts/yomiyasu_lint.py` にある `bold_not_rendered` を Go へ移植した。著作権表示とライセンス文は
  `internal/prose/bold.go` の冒頭に残した。`internal/prose/bold_test.go` の回帰ケースの表は原典の
  `tests/fixtures/bold_regressions.json` を table test へ移したもので、同じ表示を冒頭に置いた。
