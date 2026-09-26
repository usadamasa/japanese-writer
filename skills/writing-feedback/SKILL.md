---
name: writing-feedback
description: >-
  文章の言い回しを直されたとき、その指摘をルールとして登録する｡
  「この言い方やめて」「また同じ言い回しになっている」「AI っぽい」と指摘されたとき、
  同じ直しを繰り返していると気づいたときに使う｡
  NG パターンと良い書き換え例を対にして prh か writing-gate のルールへ足し、
  次のセッションから機械が拾う状態にする｡
  1 回きりの表現の好みや、その文書だけの言い換えには使わない｡
---

<!-- writing-gate-disable -->

# writing-feedback

指摘は、その場で直しても次のセッションには残らない｡書き手への指示は長い会話で薄まり、モデルが変わると癖も変わる｡直した内容を機械が拾える形に落として、点検の層へ移す｡

## 起動条件

- 言い回しを直された｡「この表現やめて」「くどい」「AI っぽい」
- crit のレビューで言い回しにコメントが付いた (同梱の `crit/on_finish_approved.md` を
  crit の approve 後の prompt に置くと、全コメントを渡して本スキルを呼ぶ)
- 同じ種類の直しを 2 回以上受けた
- 自分で書いた文を読み返して、いつもの癖に気づいた

登録しないもの:

- その文書に固有の言い換え (固有名・その場の文脈に依存するもの)
- 1 回きりの好み｡2 回目が来てから登録する
- 事実誤りの訂正｡これはルールではなく知識の問題

## どこへ足すか

置き場は 3 つ｡判定は「置換先が書けるか」で決まる｡

```text
置換先が文脈によらず一意に決まるか
├─ はい → prh-notation.yml       (autofix される)
└─ いいえ
   └─ 語・句として検出できるか
      ├─ はい → prh-prose.yml     (検出のみ｡文ごと書き直させる)
      └─ いいえ → writing-gate    (文書全体の性質・置換先を持たない漏出フレーズ)
```

| 置き場 | パス | 例 |
|----|----|----|
| `prh-notation.yml` | textlint-check の `configs/` | 全角アラビア数字、全角チルダ |
| `prh-prose.yml` | 同上 | 英文略号、「あなた」、空虚なリンクテキスト |
| writing-gate のルール | `~/.claude/writing-gate-rules.json` (端末固有) または `${CLAUDE_PLUGIN_ROOT}/writing-gate/internal/rules/rules.json` (リポジトリ) | 制作過程の痕跡、却下案の痕跡、語尾の連続の閾値 |

端末で試したいだけなら `~/.claude/writing-gate-rules.json` に書く｡定着したら writing-gate の
`internal/rules/rules.json` へ移してリポジトリへ入れる｡

ルールファイルの形・マージの規則・無効化と除外の書式は `${CLAUDE_PLUGIN_ROOT}/writing-gate/README.md` にある｡

## 書き方

### NG パターンと良い書き換え例を対で書く

禁止だけを並べると、それを避けるためだけの回りくどい言い方が出てくる｡向かうべき形を一緒に示す｡

prh の場合、`expected` が書き換えの方向を示す｡`prh` フィールドには「何が問題か」と「どう直すか」を書く｡

```yaml
  - expected: 効果の中身
    pattern: /に効く/
    prh: '「効く」だけでは何がどう変わるか伝わらない｡効果の中身を書く｡語を差し替えず文ごと直す｡'
```

writing-gate の場合、`message` に何が問題かを、`guidance` にどう直すかを書く｡

```json
{
  "id": "vague-contrast",
  "severity": "warn",
  "message": "求められていない対比で書いている｡",
  "guidance": "対比が主張に効いていないなら、後ろ側だけを書く｡",
  "regexps": ["ではなく[^｡。\\n]{1,20}(である|する|だ)"]
}
```

### 語の置換で済ませない

登録するルールは「その語を含む文を書き直せ」と要求する｡検出語だけを別の語へ置き換えても、同じ問題が形を変えて残る｡曖昧な動詞を別の曖昧な動詞に替えても曖昧なままで、削除跡を婉曲にしても跡は残る｡

そのため、意味に関わるルールは autofix されない側 (`prh-prose.yml` か writing-gate) へ置く｡

### 文書の種類で切り替える

同じ表現でも文書の種類によって是非が変わる｡造語は技術文書では避けるが、創作では技法になる｡推量表現は ADR では避けるが、Slack では書き手の指紋になる｡

- 文書の種類ごとに効かせ分けるなら、その文書の種類の `config_root` に別の `.textlintrc.json` を置く｡
- 個別の文書で外すなら、ファイル先頭に無効化コメントを置く｡textlint は `textlint-disable`、writing-gate は `writing-gate-disable` に続けてルール ID を書く｡

## 手順

### Step 1: 指摘を一般化する

その文書だけの話か、癖かを判定する｡癖なら、検出できる形まで抽象化する｡

具体例をそのままパターンにしない｡固有名が入っていたら外す｡逆に抽象化しすぎると誤検出が増える｡

### Step 2: 置き場を決めて書く

上の判定フローで決める｡置換先が文脈によらず一意だと言い切れないものは検出のみの側 (`prh-prose.yml`) へ置く｡autofix は取り返しがつかない｡

ルールのコメントには「何が問題か」「何を拾い、何を拾わないか」だけを書く｡指摘を受けた文書の名前や
レビューの引用、計測したヒット数は書かない｡ルールの是非はそれだけで判断でき、出所を書くと
そのプロジェクトの情報がルールセットに残る｡

### Step 3: 効くことを確かめる

追加したルールが、直したかった文を実際に拾うか見る｡`textlint-run.sh` と config は
[[textlint-check]] が同梱するものを絶対パスで指す｡

```bash
# prh へ足した場合
mkdir -p ./tmp && printf '%s\n' '<直したかった文>' > ./tmp/probe.md
textlint-run.sh \
  --config "${CLAUDE_PLUGIN_ROOT}/skills/textlint-check/configs/base.textlintrc.json" \
  --output "$PWD/tmp/probe.json" "$PWD/tmp/probe.md"

# writing-gate へ足した場合
"${CLAUDE_PLUGIN_DATA}/bin/writing-gate" scan ./tmp/probe.md
```

### Step 4: 誤検出を見積もる

既存の文書へ当てて、ヒット数を見る｡想定より多いならパターンが広すぎる｡

```bash
git ls-files '*.md' > ./tmp/mdlist.txt

# prh へ足した場合
xargs textlint-run.sh \
  --config "${CLAUDE_PLUGIN_ROOT}/skills/textlint-check/configs/base.textlintrc.json" \
  --output "$PWD/tmp/all.json" < ./tmp/mdlist.txt
jq '[.remaining_issues[] | select(.message | startswith("<足したルールの prh 文言の先頭>"))] | length' ./tmp/all.json

# writing-gate へ足した場合
xargs "${CLAUDE_PLUGIN_DATA}/bin/writing-gate" scan --format json < ./tmp/mdlist.txt \
  | jq '[.findings[] | select(.rule_id == "<足した id>")] | length'
```

`textlint-run.sh` は autofix を対象ファイルへ書き込む｡未コミットの変更がある文書へ当てる前に commit しておく｡

正しく直すべき箇所が出るのは成果｡関係ない文が出るなら、パターンを狭めるか、文書の種類を絞る｡

### Step 5: 片付ける

`rm ./tmp/probe.md ./tmp/probe.json ./tmp/mdlist.txt ./tmp/all.json` で probe を消す｡

足したルールは、textlint-check の config と writing-gate を持つリポジトリでコミットする｡
writing-gate の `rules.json` を変えたら再ビルドが要る｡埋め込みなので、ビルドしないと反映されない｡
配布した版へは、利用者の次のセッションの開始時に SessionStart hook が入力の差分を見てビルドし直す｡

そのリポジトリが今の cwd と違うときは、そのリポジトリを cwd にした別セッションを立てて作業する｡
`git -C` や `cd` で書き込み系の git を向ける代用はしない｡`cd` は cwd の字面が合わず sandbox 内に
落ち、`-C` 前置は照合から外れるため、guard hook が拒む｡手動 worktree での代用もしない｡

## 棚卸し

ルールは増える一方なので、たまに見る｡

- ヒットしなくなったルールは、癖が直ったか、パターンが的外れかのどちらか｡後者なら消す｡
- 同じ癖に複数のルールが当たっていたら 1 本にまとめる｡
- 最近の文書に `writing-gate scan` と textlint を当てて、拾えていない癖を探す｡

## 関連スキル

| スキル | 境界 |
|----|----|
| [[textlint-check]] | prh を含む機械点検の実行｡本スキルはそのルールを増やす側 |
| [[proofread]] | 文書 1 本の添削｡本スキルは添削で出た指摘を次へ持ち越す |
| [[japanese-tech-writing]] | 規範の本体｡新しいルールが既存の Phase 2 節に対応するなら、その節番号を `prh` フィールドに書く |
| [[sanitize-artifacts]] | 漏出パターンの正本｡漏出の型を増やしたらこちらの表にも足す |
| 書き手の文体スキル ([[proofread]] の `style_check` で差し込む) | 文書の種類ごとの指紋｡文書の種類で切り替えるルールはその区分に従う |

## 出典

<https://x.com/yugen_matuni/status/2088251220452679951> を翻案した｡指摘の都度登録・NG パターンと良い書き換え例の対・語の置換の禁止・文書の種類ごとの切り替え・文書全体の集計は原典に従う｡
