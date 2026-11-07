---
name: writing-retrospective
description: >-
  過去の crit レビューや会話で受けた文章への指摘のうち、正規表現では拾えない構成・論証・文脈の指摘を
  棚卸しして、文章規範へ反映するときに使う。
  「指摘を棚卸しして」「レビューの傾向を規範に入れて」と言われたとき、dream を回すときに使う。
  語・句の言い回しを 1 件ずつルールにする作業は writing-feedback の担当。
---

# writing-retrospective

[[writing-feedback]] はレビューのたびに言い回しを prh と writing-gate へ登録する。
構成・論証・文脈への指摘は置換先と検出パターンを書けないので、そこから漏れて残らない。
本スキルはその漏れた指摘をまとめて読み、繰り返し現れるものを [[japanese-tech-writing]] の規範へ畳む。
レビューのたびには回さず、思い出したときにまとめて回す。

## いつ回すか

- ユーザーが棚卸しを頼んだ
- dream を回すとき (セッションの蓄積が同じ周期になる)
- inbox のファイルが 10 件を超えた
- 前回の `last_run` から 30 日が近い。セッションログは Claude Code の既定 (`cleanupPeriodDays`) で 30 日ほどで消え、
  inbox に書かれていない指摘はそこで失われる

## 入力

次の表の入力を、前回の `last_run` 以降に絞って読む。`last_run` は
`~/.claude/japanese-writer/retrospective/state.json` にある。ファイルが無ければ初回として全期間を読む。

| 入力 | 場所 | 読み方 |
|----|----|----|
| inbox | `~/.claude/japanese-writer/retrospective/inbox/*.md` | writing-feedback が見送った構造・文脈の指摘。1 ファイル 1 レビュー分。形式は [[writing-feedback]] が決める |
| crit の記録 | `~/.crit/reviews/*/review.json`、`~/.crit/plans/*/.crit/review.json` | `jq '[.review_comments[]?, .files[]?.comments[]?] \| map({body, quote, replies: [.replies[]?.body]})'`。approve 後に消える記録もあるので、残っている分だけ読む |
| PR | ルールを持つリポジトリ (本 plugin) の merged PR | writing-feedback の PR 本文の「残した課題」に、登録を見送った構成の指摘が要約で残る。`gh pr list --state merged --search "merged:>=<last_run の日付>" --json number,title,body` |
| セッションログ | `~/.claude/projects/*/<session_id>.jsonl` | crit の approve 後 prompt を受けた tool_result と、文章を直させたユーザーの発言。読む手順は次の節 |

`~/.crit` は `find` で辿らず `rg --files --hidden -g '*.json' ~/.crit` で列挙する。

### セッションログの絞り込み

全 transcript を読むと重いので、文章を扱ったセッションに絞る。

agents-daemon の dream harvest (`~/.local/state/agents-daemon/dream/harvest/*.jsonl`) があるなら、索引として使う。
harvest は 1 セッション 1 行で、`skills_invoked` に呼ばれた skill 名が入る。発言は 200 文字で切られているので、
中身は harvest ではなく transcript から読む。

```bash
jq -r --arg since "<last_run>" \
  'select(.ts_end > $since)
   | select(.skills_invoked | any(test("crit|writing-feedback|proofread|japanese-tech-writing")))
   | .session_id' \
  ~/.local/state/agents-daemon/dream/harvest/*.jsonl | sort -u
```

harvest が無いなら、`rg -l -F '言い回しの指摘をルールへ移す' --glob '*.jsonl' ~/.claude/projects` で
crit の approve 後 prompt を受けたセッションを拾う。

transcript の 1 行は 1 メッセージで、crit のコメントは `type: "user"` の行の `tool_result` の文字列や、
brief ファイルを Read した結果の中に JSON のまま入っている。形が一定しないので、`jq '.. | strings'` で
文字列を全部出してから `"replies"` を含むものを拾う。

## 手順

### Step 1: 採取する

入力ごとに採取用の subagent を起動し、指摘を `./tmp/retrospective/raw-<入力>.md` へ書き出させる。
1 指摘につき、指摘の本文・返信・指摘された文 (`quote` か `anchor`)・文書の種類 (ADR、wiki、README など)・
出所 (ファイルパスか PR 番号) を残す。採取の段では判断させない。

transcript は数が多いので、subagent 1 つあたり 20 ファイル程度に分けて割り当てる。

### Step 2: 分ける

`model: "sonnet"` の subagent に raw を全部渡し、指摘ごとに次の区分へ分けさせる。

| 区分 | 判定 | 行き先 |
|----|----|----|
| 言い回し | 語・句を差し替えれば直る | 既に prh / writing-gate にあるか確かめ、無ければ [[writing-feedback]] へ回す |
| 構造・文脈 | 段落の順序、節の構成、論証の飛躍、読み手の前提、書く範囲、図や表の置き方のように、文を読み比べないと判断できない | Step 3 へ |
| 文書固有 | その文書の事実・固有名・その場の決定に依存する | 捨てる |
| 事実誤り | 内容が間違っている | 捨てる (規範ではなく知識の問題) |

構造・文脈に分けた指摘には、文書名と固有名を外した 1 文の一般形 (「結論より先に経緯を書いている」) を付けさせる。

### Step 3: 繰り返しを見る

一般形が同じ指摘をまとめる。**別々の文書で 2 回以上** 現れたものだけを規範の候補にする。

1 回きりのものは次回へ持ち越す。セッションログや crit の記録から拾った 1 回きりの指摘は、ログが消える前に
inbox へ書いておく (形式は inbox と同じ)。inbox 由来のものはそのまま残る。

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
| 書き手個人の好み (語尾・句読点・絵文字の指紋) | 書き手の文体 skill。plugin の外なので、報告に挙げてユーザーへ渡す |
| ドメインの用語・事実 | domain_check の skill。同じく報告に挙げる |

### Step 5: 規範を書く

規範は「何をするか」を先に、「なぜか」を後に書く。避ける形だけを並べず、向かう形を書く。

規範の文には文書名・指摘の引用・社名・識別子・ヒット数を入れない。規範は公開リポジトリに入り、
出所を書くとそのプロジェクトの情報が残る。例が要るなら、元の文書と無関係な題材で作り直す。

書いた節は [[proofread]] を `medium: "docs"` で通す。

### Step 6: 記録して閉じる

1. 規範 1 つにつき 1 commit にする。commit message に一般形と、何件の別文書から来たかを書く。
2. Draft PR を作る。本文の「残した課題」に、持ち越した候補の一般形と、plugin の外へ渡した指摘を書く。
3. `state.json` の `last_run` を、Step 1 を始めた時刻 (ISO 8601) で書き直す。

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
| dream (agents-daemon) | 記憶階層の consolidation。harvest を本スキルの索引に使う |
