---
name: writing-retrospective
description: >-
  過去の crit レビューや会話で受けた文章への指摘のうち、正規表現では拾えない構成・論証・文脈の指摘を
  棚卸しして、文章規範へ反映するときに使う。
  「指摘を棚卸しして」「レビューの傾向を規範に入れて」と言われたとき、記憶の棚卸し (consolidation) を回すときに使う。
  語・句の言い回しを 1 件ずつルールにする作業は writing-feedback の担当。
---

# writing-retrospective

[[writing-feedback]] はレビューのたびに言い回しを prh と writing-gate へ登録する。
構成・論証・文脈への指摘は置換先と検出パターンを書けないので、そこから漏れて残らない。
本スキルはその漏れた指摘をまとめて読み、繰り返し現れるものを [[japanese-tech-writing]] の規範へ畳む。
レビューのたびには回さず、思い出したときにまとめて回す。

## いつ回すか

- ユーザーが棚卸しを頼んだ
- 記憶の棚卸し (consolidation) を回すとき (セッションの蓄積が同じ周期になる)
- inbox のファイルが 10 件を超えた
- 前回の `last_run` から 30 日が近い。crit のコメントと会話での訂正はセッションログ (transcript) から採るが、
  セッションログは Claude Code の保持期間 (`cleanupPeriodDays`、既定 30 日) を過ぎると消える。
  inbox に書かれていない指摘はそこで失われるので、保持期間の内に回す

## 状態の置き場

inbox・持ち越しの候補・前回の実行時刻は、端末ローカルの `$XDG_STATE_HOME/japanese-writer/retrospective/`
に置く (以下 `<state>`)。`XDG_STATE_HOME` が未設定なら `~/.local/state` を使う。中身は他の文書の指摘を
原文で持つので、リポジトリには入れない。

| パス | 中身 |
|----|----|
| `<state>/inbox/*.md` | writing-feedback が見送った構造・文脈の指摘 |
| `<state>/carryover.jsonl` | 前回までに 1 回きりだった候補の一般形 |
| `<state>/state.json` | 前回の棚卸しの開始時刻 (`last_run`) |

書き込みは Write ツールで行う。Bash の sandbox はこの置き場への書き込みを許していないことがある。

## 入力

次の表の入力を、前回の `last_run` 以降に絞って読む。`<state>/state.json` が無ければ初回として全期間を読む。

| 入力 | 場所 | 読み方 |
|----|----|----|
| inbox | `<state>/inbox/*.md` | writing-feedback が見送った構造・文脈の指摘。1 ファイル 1 レビュー分。形式は [[writing-feedback]] が決める |
| crit のコメント | セッションログ (`~/.claude/projects/*/*.jsonl`) と crit の記録 (`~/.crit` 配下の `review.json`) | 同梱のフィルタで機械的に取り出す (Step 1)。crit の記録は approve 後に消えるものが多く、コメントの大半はセッションログに残った approve 後 prompt から取れる |
| 会話での訂正 | セッションログ | crit を通さずに文章を直させた発言。同梱のフィルタで文章を扱ったセッションの発言を取り出す (Step 1) |

## 手順

### Step 1: 採取する

crit のコメントは判断なしで取り出せるので、subagent を使わずにフィルタを当てる。
フィルタ (`${CLAUDE_SKILL_DIR}/extract-crit-comments.jq`) は transcript の 1 行か `review.json` を受ける。
出力はコメント 1 件 1 行の JSON で、本文・返信・指摘された文・ファイルパス・作成時刻を持つ。
同じコメントは approve のたびに繰り返し埋め込まれるので、`id` で重複を除く。

```bash
mkdir -p ./tmp/retrospective
rg -l -0 --glob '*.jsonl' -F '"replies' ~/.claude/projects \
  | xargs -0 jq -c -f "${CLAUDE_SKILL_DIR}/extract-crit-comments.jq" > ./tmp/retrospective/crit-all.jsonl
rg --files --hidden -0 -g 'review.json' ~/.crit \
  | xargs -0 jq -c -f "${CLAUDE_SKILL_DIR}/extract-crit-comments.jq" >> ./tmp/retrospective/crit-all.jsonl
jq -s -c --arg since "<last_run>" '
  group_by(.id) | map(last) | .[]
  | select($since == "" or (.created_at // "") > $since)
  | select((.author // "") | test("Claude") | not)
  | select(.path == null or (.path | test("\\.md$")))
  | {id, doc: (.path // "review"), body, target: ((.quote // .anchor // "")[0:200]),
     replies: [.replies[].body[0:300]]}' \
  ./tmp/retrospective/crit-all.jsonl > ./tmp/retrospective/crit.jsonl
```

`~/.crit` は `find` で辿らず `rg --files` で列挙する。書き手 (Claude) のコメントと、md 以外へのコメント
(コードやその設計への意見が大半) は落とす。

会話での訂正は、セッションログから利用者の発言を取り出して集める。
フィルタ (`${CLAUDE_SKILL_DIR}/extract-user-utterances.jq`) は transcript 1 本を `-s` で受け、
.md を編集したか japanese-writer の skill を呼んだセッションに絞って、発言 1 件 1 行の JSON
(`session_id`、`timestamp`、`text`) を出す。訂正かどうかはここでは決めず、Step 2 の subagent が判定する。

```bash
rg --files -0 --glob '*.jsonl' --max-depth 2 ~/.claude/projects \
  | xargs -0 -n 1 jq -s -c -f "${CLAUDE_SKILL_DIR}/extract-user-utterances.jq" > ./tmp/retrospective/utterances-all.jsonl
jq -c --arg since "<last_run>" 'select($since == "" or (.timestamp // "") > $since)' \
  ./tmp/retrospective/utterances-all.jsonl > ./tmp/retrospective/conversation.jsonl
```

`--max-depth 2` で subagent の transcript (`<project>/<session>/subagents/`) を外す。利用者の発言は親のセッションにだけ残る。

### Step 2: 分ける

Step 1 の出力と inbox を、`model: "sonnet"` の subagent に分けさせる。crit のコメントは 200 件程度ずつに
`split -l` で割り、割った数だけ subagent を並べて起動する。各 subagent は 1 件ずつ次の区分に分け、
`./tmp/retrospective/classified-<n>.jsonl` へ書く。
会話の発言 (`conversation.jsonl`) も同じく割って渡す。発言の大半は作業の依頼なので、文章への訂正かを先に判定させ、
訂正でないものは作業の指示として捨てさせる。前後を読まないと判定できない発言は、`session_id` の transcript を開かせる。

| 区分 | 判定 | 行き先 |
|----|----|----|
| 作業の指示 | 文章ではなく、作業や設計・コードへの意見 | 捨てる |
| 言い回し | 語・句を差し替えれば直る | 既に prh / writing-gate にあるか確かめ、無ければ [[writing-feedback]] へ回す |
| 構造・文脈 | 段落の順序、節の構成、論証の飛躍、読み手の前提、書く範囲、図や表の置き方のように、文を読み比べないと判断できない | Step 3 へ |
| 文書固有 | その文書の事実・固有名・その場の決定に依存する | 捨てる |
| 事実誤り | 内容が間違っている | 捨てる (規範ではなく知識の問題) |

構造・文脈に分けた指摘には、文書名と固有名を外した 1 文の一般形 (「結論より先に経緯を書いている」) と、
直し方の 1 文を付けさせる。並べた subagent の間で一般形の言い回しがずれるので、Step 3 でまとめ直す。

### Step 3: 繰り返しを見る

一般形が同じ指摘をまとめる。**別々の文書で 2 回以上** 現れたものだけを規範の候補にする。
初回や間が長く空いた後で候補が多すぎるときは、閾値を 3 文書に上げてよい。上げたら PR 本文にそう書く。

理由の無い一言の指摘 (「不要」「書きすぎ」) を寄せた束は、粗いまま規範にしない。束に入った指摘の返信を読み、
書き手が実際にどう直したか (消した、縮めた、別の文書へ移した) で分けてから規範の文を決める。

数えるときは、前回から持ち越した候補 (`<state>/carryover.jsonl`) も足す。
1 回きりのものは同じファイルへ持ち越す。セッションログは消えるので、元の指摘ではなく一般形と文書を残す。
ファイルは Write で丸ごと書き直し、規範へ入れたものは外す。

```json
{"general": "結論より先に経緯を書いている", "fix": "結論を節の冒頭へ移し、経緯は後ろへ回す", "docs": ["docs/adr/0012-storage.md"]}
```

### Step 4: 置き場を決める

候補ごとに、既存の規範がもう同じことを言っていないかを先に確かめる。

- 既にあるなら、規範はあるのに守られていない。新しい節を足さず、その節に今回の型の例を 1 つ足すか、
  Phase 3 の self-check 項目に点検の観点として足す。
- 無いなら、次の表で置き場を決める。

| 指摘の種類 | 置き場 |
|----|----|
| 読み手・文書の立場・scope・重点先行・見出しの骨格 | `japanese-tech-writing/SKILL.md` の Phase 1 |
| 段落と論証、論証の厳密さ、視点と語り、冗長、見出し、読み手の負荷、読者への誠実さ | `japanese-tech-writing/references/phase2-norms.md` の該当節 |
| 設計文書の図・択一案の並置・意思決定の軸・未決事項 | `japanese-tech-writing/references/design-doc-scaffolding.md` |
| 長文の緊張と回収、文のビート | `japanese-tech-writing/references/cognitive-rhythm.md` |
| 制作過程や却下案の痕跡 | [[sanitize-artifacts]] |
| 書き手個人の好み (語尾・句読点・絵文字の指紋) | 書き手の文体 skill。plugin の外なので、Step 6 でユーザーへ渡す |
| ドメインの用語・事実 | domain_check の skill。同じく Step 6 で渡す |

### Step 5: 規範を書く

規範は「何をするか」を先に、「なぜか」を後に書く。避ける形だけを並べず、向かう形を書く。

規範の文には文書名・指摘の引用・社名・識別子・ヒット数を入れない。規範は公開リポジトリに入り、
出所を書くとそのプロジェクトの情報が残る。例が要るなら、元の文書と無関係な題材で作り直す。

書いた節は [[proofread]] を `medium: "docs"` で通す。

### Step 6: 記録して閉じる

1. 規範 1 つにつき 1 commit にする。commit message に一般形と、何件の別文書から来たかを書く。
2. plugin の外へ渡す指摘 (Step 4) を報告に挙げてユーザーへ渡す。渡し方の決めごとがユーザーの指示にあれば、そちらに従う。
3. Draft PR を作る。
4. `<state>/state.json` の `last_run` を、Step 1 を始めた時刻 (ISO 8601) で書き直す。

```json
{"last_run": "2026-11-07T15:00:00+09:00"}
```

規範を書き換えたリポジトリが cwd と違うときは、そのリポジトリを cwd にした別セッションで Step 5 以降を行う。

## 関連スキル

| スキル | 境界 |
|----|----|
| [[writing-feedback]] | 言い回しを 1 件ずつ機械のルールにする。構造・文脈の指摘は inbox へ置き、本スキルが拾う |
| [[japanese-tech-writing]] | 規範の本体。本スキルはそこへ書き足す |
| [[proofread]] | 文書 1 本の添削。proofreader は規範を読むので、規範へ入れた観点は次の添削から効く |
| [[sanitize-artifacts]] | 漏出の型の正本 |
