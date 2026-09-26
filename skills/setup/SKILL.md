---
name: setup
description: >-
  japanese-writer plugin の自動準備 (SessionStart hook による writing-gate のビルド) が失敗したとき、
  Stop hook が「writing-gate が見つかりません」で完了を止めたとき、crit の approve 後 prompt を plugin の
  ものに切り替えたいときに使う｡前提コマンドの確認と writing-gate のビルドと検証を 1 回で済ませる｡
  「セットアップして」「writing-gate をビルドして」「crit をリンクして」と言われたときにも使う｡
---

# setup

plugin の準備を手で確かめ直す｡実体は `scripts/setup.sh` で、本 skill はそれを走らせて結果を利用者へ返す｡

writing-gate のビルドは、普段は SessionStart hook が plugin の data ディレクトリ
(`${CLAUDE_PLUGIN_DATA}`) へ自動で行う｡install と update の後に本 skill を呼ぶ必要はない｡
呼ぶのは、自動の準備が失敗したときと crit のリンクを張るとき｡

## Step 1: スクリプトを実行する

data ディレクトリは Bash の環境変数に入っていないので、前置きして渡す｡

```bash
CLAUDE_PLUGIN_DATA="${CLAUDE_PLUGIN_DATA}" "${CLAUDE_PLUGIN_ROOT}/scripts/setup.sh"
```

stdout と stderr をそのまま読む｡終了コード 0 なら Step 3 へ、それ以外は Step 2 へ｡

## Step 2: 失敗の種類ごとに返す

| 出力に含まれる文言 | 意味 | 利用者へ返すこと |
| ---- | ---- | ---- |
| `前提コマンドが不足しています` | go / npx / jq のどれかが PATH に無い | 足りないコマンド名と用途 (出力にある) をそのまま伝える｡インストールは利用者の環境の話なので、代わりに入れない |
| `へ書き込めません` | sandbox が data ディレクトリか `~/.crit` への書き込みを拒んだ | 出力にある `! CLAUDE_PLUGIN_DATA=… "…/scripts/setup.sh"` の行をそのまま示し、Claude Code のプロンプトに入力すると sandbox の外で実行できると伝える｡`dangerouslyDisableSandbox` では回避しない｡利用者が `!` で実行しても同じ文言なら、そのディレクトリの所有者と権限の問題として伝える |
| `go build に失敗しました` | Go のビルドが落ちた | 直前に流れた go の出力を要約して伝える｡go.mod の要求より古い Go だと toolchain の取得で落ちる |
| `既知の process-leak パターンを拾いません` | 埋め込みルールの破損 | plugin の版を伝え、plugin の不具合として報告するよう案内する｡前回のバイナリは残っている |

利用者が `!` で実行した後は、その出力を読んで Step 3 へ進む｡

## Step 3: crit のリンクを提案する

出力に `crit: 見つかりました` があれば、`AskUserQuestion` で 1 回だけ尋ねる｡

- `plugin の prompt にリンクする (Recommended)`
- `今はしない`

リンクを選んだら `--crit` を付けて再実行する｡

```bash
CLAUDE_PLUGIN_DATA="${CLAUDE_PLUGIN_DATA}" "${CLAUDE_PLUGIN_ROOT}/scripts/setup.sh" --crit
```

出力に `crit: 見つかりました` が無ければ尋ねない｡`crit: リンク済み` なら済んでいる｡
`既に別の場所を指しているので触りません` が出たら、その指し先を伝えて利用者の判断に任せる｡

## Step 4: 報告する

最後の実行の出力を要約して返す｡

- 前提コマンドの結果
- `writing-gate` のパス (`${CLAUDE_PLUGIN_DATA}/bin/writing-gate`) と検証の結果
- crit のリンクの状態
- plugin を更新しても再実行は要らないこと (次のセッションの開始時に hook が準備し直す)

## やらないこと

- 前提コマンドのインストール
- `~/.crit/prompts/on_finish_approved.md` が別の場所を指しているときの張り替え
- textlint のパッケージ取得｡初回の `proofread` で `npx` が取りに行く
