---
name: setup
description: >-
  japanese-writer plugin を install または update した直後、Stop hook が
  「writing-gate が見つかりません」で完了を止めたとき、crit の approve 後 prompt を plugin の
  ものに切り替えたいときに使う｡前提コマンドの確認と writing-gate のビルドと検証を 1 回で済ませる｡
  「セットアップして」「writing-gate をビルドして」「plugin を更新したので直して」と言われたときにも使う｡
---

# setup

plugin を使える状態にする｡実体は `scripts/setup.sh` で、本 skill はそれを走らせて結果を利用者へ返す｡

## Step 1: スクリプトを実行する

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/setup.sh"
```

stdout と stderr をそのまま読む｡終了コード 0 なら Step 3 へ、それ以外は Step 2 へ｡

## Step 2: 失敗の種類ごとに返す

| 出力に含まれる文言 | 意味 | 利用者へ返すこと |
| ---- | ---- | ---- |
| `前提コマンドが不足しています` | go / npx / jq のどれかが PATH に無い | 足りないコマンド名と用途 (出力にある) をそのまま伝える｡インストールは利用者の環境の話なので、代わりに入れない |
| `へ書き込めません` | sandbox が plugin の置き場か `~/.crit` への書き込みを拒んだ | 出力にある `! "…/scripts/setup.sh"` の行をそのまま示し、Claude Code のプロンプトに入力すると sandbox の外で実行できると伝える｡`dangerouslyDisableSandbox` では回避しない｡利用者が `!` で実行しても同じ文言なら、そのディレクトリの所有者と権限の問題として伝える |
| `go build に失敗しました` | Go のビルドが落ちた | 直前に流れた go の出力を要約して伝える｡go.mod の要求より古い Go だと toolchain の取得で落ちる |
| `既知の process-leak パターンを拾いません` | 古いバイナリか埋め込みルールの破損 | plugin の版と `bin/writing-gate` の作成日時を伝え、`bin/` を消してから再実行するよう案内する |

利用者が `!` で実行した後は、その出力を読んで Step 3 へ進む｡

## Step 3: crit のリンクを提案する

出力に `crit: 見つかりました` があれば、`AskUserQuestion` で 1 回だけ尋ねる｡

- `plugin の prompt にリンクする (Recommended)`
- `今はしない`

リンクを選んだら `--crit` を付けて再実行する｡

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/setup.sh" --crit
```

出力に `crit: 見つかりました` が無ければ尋ねない｡`crit: リンク済み` なら済んでいる｡
`既に別の場所を指しているので触りません` が出たら、その指し先を伝えて利用者の判断に任せる｡

## Step 4: 報告する

最後の実行の出力を要約して返す｡

- 前提コマンドの結果
- `bin/writing-gate` のパスと検証の結果
- crit のリンクの状態
- plugin を更新したら再実行が要ること (置き場のディレクトリが版ごとに変わり、`bin/` が無くなる)

## やらないこと

- 前提コマンドのインストール
- `~/.crit/prompts/on_finish_approved.md` が別の場所を指しているときの張り替え
- textlint のパッケージ取得｡初回の `proofread` で `npx` が取りに行く
