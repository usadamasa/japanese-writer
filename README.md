# japanese-writer

日本語の技術文書を書く・校正するための Claude Code plugin｡

## 収録内容

| 種別 | 名前 | 役割 |
| ---- | ---- | ---- |
| skill | japanese-tech-writing | 技術文書の構成・論証・段落の規範｡書き始める前の構成検討から推敲まで扱う |
| skill | proofread | md ドラフトの総合添削｡proofreader subagent へ委譲し、確度別に分類した修正案を返す |
| skill | textlint-check | textlint による機械点検と自動修正｡呼び出し元が `config_root` で config を切り替える |
| skill | sanitize-artifacts | 対話で仕上げた成果物から、制作過程と却下した案の痕跡を取り除く |
| skill | writing-feedback | 言い回しへの指摘を prh のルールとして登録し、次から機械が拾えるようにする |
| agent | proofreader | proofread skill から呼ばれる添削担当の subagent｡直接は呼ばない |
| hook | Stop: writing-gate | セッションを閉じるときに、編集した `.md` を点検して重大な指摘があれば完了を止める |
| crit prompt | `crit/on_finish_approved.md` | crit のレビューが approve されたとき、言い回しへの指摘を writing-feedback へ回す |

writing-gate は文書全体の集計 (語尾の連続・文体の混在) と逐語の漏出を見る Go 製の CLI｡
設計とルールの置き場は [writing-gate/README.md](writing-gate/README.md) にある｡

## 前提

- Node.js (`npx`)｡textlint は `npx` で都度取得するので、`node_modules` を用意する必要はない
- `jq`
- Go (writing-gate のビルドに使う)

## インストール

Claude Code で次を実行する｡

```text
/plugin marketplace add usadamasa/japanese-writer
/plugin install japanese-writer@japanese-writer
```

### writing-gate のビルド

Stop hook は plugin の `bin/writing-gate` を起動する｡install しただけではバイナリが無いので、
install 先 (`~/.claude/plugins/cache/japanese-writer/japanese-writer/<version>/`) で一度ビルドする｡

```sh
go build -o bin/ ./...
```

ビルドしていないあいだは、hook が完了を止めてビルドの手順を示す｡plugin を更新したら再ビルドする｡
`bin/` は plugin が有効なあいだ Bash の PATH に載るので、`writing-gate scan` をそのまま呼べる｡

### crit との連動 (任意)

[crit](https://github.com/tomasz-tomczyk/crit) を使っているなら、approve 後の prompt を plugin の
ファイルへリンクする｡

```sh
mkdir -p ~/.crit/prompts
ln -s ~/.claude/plugins/cache/japanese-writer/japanese-writer/<version>/crit/on_finish_approved.md \
  ~/.crit/prompts/on_finish_approved.md
```

リンク先はバージョンのディレクトリを含むので、plugin を更新したら張り直す｡

## 文体とドメイン用語の差し込み

proofread は汎用の機械点検と文章規範だけを持つ｡書き手ごとの文体とドメイン固有の用語は、
ユーザー側の `~/.claude/skills/` に置いた skill を引数で名指しして差し込む｡

| 引数 | 差し込むもの | 例 |
| ---- | ---- | ---- |
| `style_check` | 書き手の文体 (文書の種類ごとの語尾・句読点・絵文字の指紋) を持つ skill | `{"skill": "my-writing-style"}` |
| `domain_check` | 用語表や事実の正本を持つ skill と、検出結果の Tier 分類表 | `{"skill": "my-domain-terms", "tiering_ref": "references/tiering.md"}` |

`tiering_ref` は絶対パスか、`~/.claude/skills/<skill>/` からの相対パスで指定する｡
どちらも省略でき、省略した側の点検は skip され、結果の `skipped` に記録される｡

## 注意点

同梱の textlint config は、句読点を半角の「｡ ､」へ autofix する｡
コード (インラインとブロック) の中は書き換えない｡

## ライセンス

MIT
