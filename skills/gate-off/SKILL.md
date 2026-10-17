---
name: gate-off
description: >-
  このセッションの間だけ writing-gate (Stop hook の文章点検) を止める。
  議事録や他人のファイルのように、無効化コメントや exclude_extra を入れにくい文書を書くときに
  ユーザーが `/japanese-writer:gate-off` で打つ。`/japanese-writer:gate-on` で戻す。
disable-model-invocation: true
---

# gate-off

ユーザーがこのセッションの writing-gate を止めた。次の Stop から、セッションを閉じるときの
文章点検は行わない。`/japanese-writer:gate-on` を打つまで続く。効くのはこのセッションの間だけで、
新しいセッションでは点検する状態に戻る。

判定は writing-gate が transcript から行う。ユーザーが打った slash command だけを読み、
Claude が Skill ツールで呼んだものは数えない。止めるかどうかはユーザーだけが決める。

## すること

1 行で「このセッションの writing-gate を止めた。戻すには `/japanese-writer:gate-on`」と返す。
ファイルは編集しない。コマンドも実行しない。

## やらないこと

- 無効化コメント (`<!-- writing-gate-disable -->`) や `~/.claude/writing-gate.json` を書く。
  この skill はセッション限りの停止で、ファイルや設定には触れない
- 自分で `/japanese-writer:gate-off` を呼ぶ。効かないうえ、止める判断をユーザーから奪う
