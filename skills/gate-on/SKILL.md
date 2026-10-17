---
name: gate-on
description: >-
  `/japanese-writer:gate-off` で止めた writing-gate (Stop hook の文章点検) をこのセッションで戻す。
  ユーザーが `/japanese-writer:gate-on` で打つ。
disable-model-invocation: true
---

# gate-on

ユーザーが writing-gate を戻した。次の Stop から、セッションを閉じるときの文章点検を再び行う。

判定は writing-gate が transcript から行う。gate-off と gate-on のうち、後に打った方が効く。

## すること

1 行で「このセッションの writing-gate を戻した」と返す。ファイルは編集しない。コマンドも実行しない。
