#!/bin/bash
set -euo pipefail
# stop-writing-gate.sh
# Stop hook: このセッションで書いた markdown を機械点検し、重大な指摘が残っていれば
# 完了をブロックする。
#
# 書き手への指示 (CLAUDE.md / skill) は長い会話で薄まる。文章規範を守らせる最後の砦を
# 執筆の後段に置き、生成物そのものを見て判定する。
#
# 点検は Go 実装 (cmd/writing-gate) が行う。npx textlint を通すと完了のたびに数秒
# 待たされるため、ここでは文書全体の集計と漏出フレーズだけを見る。語彙・表記の点検は
# proofread skill の担当で、そちらは明示的に呼ぶ。
#
# ブロックのループは stop_hook_active で止める。2 度目の Stop では判定しない。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC2034 # Used by sourced hook-logger.sh
HOOK_NAME="stop-writing-gate"
# shellcheck disable=SC1091 # Dynamically resolved path
source "$SCRIPT_DIR/lib/hook-logger.sh"

INPUT=$(cat)

read -r TRANSCRIPT_PATH STOP_HOOK_ACTIVE < <(
  printf '%s' "$INPUT" | jq -r '[(.transcript_path // ""), (.stop_hook_active // false | tostring)] | @tsv'
)

# 直前の Stop で既にブロックしている。同じ指摘で往復させない。
if [ "$STOP_HOOK_ACTIVE" = "true" ]; then
  log_skip "stop_hook_active"
  exit 0
fi

if [ -z "$TRANSCRIPT_PATH" ] || [ ! -f "$TRANSCRIPT_PATH" ]; then
  log_skip "transcript がありません"
  exit 0
fi

BIN="$SCRIPT_DIR/../bin/writing-gate"
if [ ! -x "$BIN" ]; then
  # 未ビルドだと文章のゲートが丸ごと無効になる。黙って通さず、その事実を報告する。
  jq -cn --arg r "writing-gate が見つかりません｡文章の完了ゲートが無効な状態です｡writing-gate をビルドして hooks の隣の bin/ へ置いてください｡" \
    '{"decision":"block","reason":$r}'
  exit 0
fi

# 点検自体が失敗したときは、その理由を出して通す。文章の不備ではないため止めない。
if ! GATE_OUT=$("$BIN" gate --transcript "$TRANSCRIPT_PATH" 2>&1); then
  log_error "writing-gate の実行に失敗しました: $GATE_OUT"
  exit 0
fi

# 指摘が無いときは {} が返る。そのまま出しても害はないが、Claude Code の
# 出力欄を汚さないよう握っておく。
if [ "$(printf '%s' "$GATE_OUT" | jq -r '.decision // ""')" != "block" ]; then
  exit 0
fi

printf '%s\n' "$GATE_OUT"
exit 0
