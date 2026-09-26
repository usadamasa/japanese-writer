# japanese-writer

日本語の技術文書を書く・校正するための Claude Code plugin｡

## 収録内容

入口は 2 つ｡書くときは `japanese-tech-writing` が構成から推敲まで導き、書き上がったら
`/japanese-writer:proofread` で添削する｡残りは proofread の内側から呼ばれるか、指摘を受けたときに使う｡

```text
ユーザー
├─ /japanese-writer:setup ── 自動の準備が失敗したとき、crit をリンクするとき
├─ 書く ── japanese-tech-writing (Phase 1 構成 → 2 執筆 → 3 推敲)
│             └─ Phase 3 の最後に proofread を呼ぶ
├─ /japanese-writer:proofread ── 添削の入口
│    ├─ textlint-check (機械点検と autofix)
│    └─ proofreader (subagent)
│         ├─ japanese-tech-writing (規範として読む)
│         ├─ sanitize-artifacts (漏出の観点として読む)
│         ├─ style_check で差し込んだ文体 skill (任意)
│         └─ domain_check で差し込んだ用語 skill (任意)
├─ /japanese-writer:writing-feedback ── 言い回しを直されたとき
│    └─ crit の approve 後の prompt (crit/on_finish_approved.md) からも呼ばれる
├─ (自動) SessionStart hook: writing-gate を data ディレクトリへビルド ── セッションを開くとき
└─ (自動) Stop hook: writing-gate ── セッションを閉じるとき
```

| 種別 | 名前 | 役割 | 呼び方 |
| ---- | ---- | ---- | ---- |
| skill | setup | 前提コマンドの確認、writing-gate のビルドと検証のやり直し、crit の prompt リンク | ユーザーが `/japanese-writer:setup` で呼ぶ｡SessionStart hook の準備が失敗したとき、未ビルドのまま Stop hook に止められたとき、crit をリンクするとき |
| skill | japanese-tech-writing | 技術文書の構成・論証・段落の規範｡書き始める前の構成検討から推敲まで扱う | 文書を書くとき Claude が読み込む｡`/japanese-writer:japanese-tech-writing` で明示的にも呼べる |
| skill | proofread | md ドラフトの総合添削｡proofreader subagent へ委譲し、修正案が一意なものは適用して報告し、論証に関わるものは警告に留める | ユーザーが `/japanese-writer:proofread` で呼ぶ (添削の入口) |
| skill | textlint-check | textlint による機械点検と自動修正｡呼び出し元が `config_root` で config を切り替える | proofread の内側から呼ばれる｡機械点検だけ欲しいときは単体でも呼べる |
| skill | sanitize-artifacts | 対話で仕上げた成果物から、制作過程と却下した案の痕跡を取り除く | proofreader が点検の観点として読み込む｡「これは消して」の後の作り直しでは単体で呼ぶ |
| skill | writing-feedback | 言い回しへの指摘を prh のルールとして登録し、次から機械が拾えるようにする | ユーザーが言い回しを直したときに呼ぶ｡crit の approve 後の prompt からも回る |
| agent | proofreader | proofread skill から呼ばれる添削担当の subagent | proofread が dispatch する｡直接は呼ばない |
| hook | SessionStart: prepare | writing-gate のビルドの入力が前回と違えば data ディレクトリへビルドし直し、crit の prompt を data へ複製する | 自動 (開始時) |
| hook | Stop: writing-gate | セッションを閉じるときに、編集した `.md` を点検して重大な指摘があれば完了を止める | 自動 (完了時) |
| crit prompt | `crit/on_finish_approved.md` | crit のレビューが approve されたとき、言い回しへの指摘を writing-feedback へ回す | crit が approve 時に読む |

writing-gate は文書全体の集計 (語尾の連続・文体の混在) と逐語の漏出を見る Go 製の CLI｡
設計とルールの置き場は [writing-gate/README.md](writing-gate/README.md) にある｡

## 前提

- Node.js (`npx`)｡textlint は `npx` で都度取得するので、`node_modules` を用意する必要はない
- `jq`
- Go (writing-gate のビルドに使う)

## インストール

配布は marketplace `usadamasa` ([usadamasa/agents-marketplace](https://github.com/usadamasa/agents-marketplace))
経由で行う｡Claude Code で次を実行する｡

```text
/plugin marketplace add usadamasa/agents-marketplace
/plugin install japanese-writer@usadamasa
```

### セットアップ

install の後に手で打つものは無い｡セッションを開くと SessionStart hook が、Stop hook の起動する
`writing-gate` を plugin の data ディレクトリ (`~/.claude/plugins/data/japanese-writer-usadamasa/bin/`) へ
ビルドし、既知の漏出パターンを拾えることまで検証する｡初回は Go のビルドの分だけ開始が遅れる｡

install 先 (`~/.claude/plugins/cache/usadamasa/japanese-writer/<version>/`) は版ごとに変わるが、
data ディレクトリは update をまたいで残る｡hook はビルドの入力 (Go のソースと埋め込みルール) の
cksum を前回と比べ、変わったときだけビルドし直す｡update の後に再ビルドやリンクの張り直しは要らない｡

準備に失敗すると、開始時にその理由が表示される｡ビルドできていないあいだは、Stop hook が完了を止めて
`/japanese-writer:setup` を案内する｡setup は前提コマンドを確かめてからビルドをやり直し、失敗の理由を返す｡

```text
/japanese-writer:setup
```

### crit との連動 (任意)

[crit](https://github.com/tomasz-tomczyk/crit) を使っているなら、approve 後の prompt を plugin の
ファイルへリンクする｡`crit` が PATH にあると setup がこれを提案する｡リンク先は data ディレクトリの
複製 (`~/.claude/plugins/data/japanese-writer-usadamasa/crit/on_finish_approved.md`) で、中身は
SessionStart hook が新しい版に揃えるので、張るのは 1 回で済む｡

以前の setup が張った、版ごとの置き場を指すリンクは `--crit` で data の複製へ張り替える｡
指し先が消えたリンクも張り替え、それ以外の場所を指しているときは触らずに報告だけする｡

## バージョン

版は calver の `YYYY.MMDD.NN` (例: `2026.0926.01`) で付ける｡Claude Code は `plugin.json` の
`version` の文字列が変わったときだけ update を配るので、配布物を変える PR では版を上げる｡

```sh
task bump  # 今日の日付の版にする｡同じ日なら NN を 1 つ上げる
```

CI の `version bump` job は、配布物に差分があるのに版が base と同じ PR を落とす｡
テスト・CI・開発用の設定だけの変更なら上げなくてよい｡

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

## 開発

plugin を使うだけなら「前提」節のツールで足りる｡このリポジトリで開発するときは､加えて次を用意する｡

- [aqua](https://aquaproj.github.io/)｡task・golangci-lint・shellcheck・pinact・jv の版を `aqua.yaml` で固定している｡
  `.envrc` が shim を張って `$(aqua root-dir)/bin` を PATH に足すので､direnv を入れて `direnv allow` する｡
  direnv を使わないなら `aqua i -l` を打ち､同じパスを自分で PATH に通す
- [jv](https://github.com/santhosh-tekuri/jsonschema) (`task validate` が JSON / YAML を宣言された schema で検証するのに使う)｡
  aqua で入るので個別のインストールは要らない｡Homebrew には formula が無く､aqua を使わない場合は
  `go install github.com/santhosh-tekuri/jsonschema/cmd/jv@latest` で入れる
- [bats-core](https://github.com/bats-core/bats-core) (`task test` が使う)
- Claude Code CLI (`task validate` が `claude plugin validate` を呼ぶ)

CI と同じ確認は task で手元でも通せる｡

| コマンド | 内容 |
| ---- | ---- |
| `task build` | `bin/writing-gate` をビルドする |
| `task test` | ビルドしてから Go のテストと bats を実行する |
| `task lint` | golangci-lint と shellcheck を実行する |
| `task validate` | `claude plugin validate --strict` と､JSON / YAML が宣言した schema での検証を実行する |
| `task bump` | `plugin.json` の version を calver で 1 つ進める |

## ライセンス

MIT
