#!/usr/bin/env bats
# setup.sh のテスト
#
# 実リポジトリや本物の ~/.claude/plugins/data と ~/.crit を巻き込まないよう、plugin root と
# data と HOME を隔離ツリーへ置く。go / npx / crit は PATH 上のモックで置き換え、
# ビルドと検証は呼び出しの記録だけを見る。
bats_require_minimum_version 1.5.0

load lib/plugin-tree-helpers

setup() {
  setup_plugin_tree
  SCRIPT_PATH="$ROOT/scripts/setup.sh"
  CRIT_DST="$HOME/.crit/prompts/on_finish_approved.md"
  mock_go
  mock_cmd npx
}

teardown() {
  teardown_plugin_tree
}

# =============================================================================
# 前提
# =============================================================================

@test "go が無ければその名前を出して exit 1 し、ビルドしない" {
  rm "$MOCK_PATH/go"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"go"* ]]
  [ ! -e "$DATA/bin/writing-gate" ]
}

@test "npx が無ければその名前を出して exit 1 する" {
  rm "$MOCK_PATH/npx"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"npx"* ]]
  [ "$(go_calls)" -eq 0 ]
}

@test "CLAUDE_PLUGIN_DATA が無ければ exit 1 し、ビルドしない" {
  unset CLAUDE_PLUGIN_DATA

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"CLAUDE_PLUGIN_DATA"* ]]
  [ "$(go_calls)" -eq 0 ]
}

# =============================================================================
# ビルドと検証
# =============================================================================

@test "writing-gate を data の bin/ へビルドし、検証まで通して exit 0 する" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [ -x "$DATA/bin/writing-gate" ]
  [[ "$output" == *"$DATA/bin/writing-gate"* ]]
}

@test "data の bin/ に書けなければ go を起動せず、data を渡す ! 前置のコマンドを出して exit 1 する" {
  mkdir -p "$DATA/bin"
  chmod a-w "$DATA/bin"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [ "$(go_calls)" -eq 0 ]
  [[ "$output" == *"! CLAUDE_PLUGIN_DATA=\"$DATA\" \"$SCRIPT_PATH\""* ]]
}

@test "go build が失敗したら stderr をそのまま流して exit 1 する" {
  printf '#!/bin/bash\nprintf "boom: go.mod requires go >= 9.9\\n" >&2\nexit 1\n' >"$MOCK_PATH/go"
  chmod +x "$MOCK_PATH/go"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"go.mod requires go >= 9.9"* ]]
}

@test "ビルドしたバイナリが既知の漏出パターンを拾えなければ exit 1 する" {
  mock_go 0

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"process-leak"* ]]
}

@test "入力が前回と同じでもビルドし直す" {
  run "$SCRIPT_PATH"
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 2 ]
}

# =============================================================================
# crit の prompt リンク
# =============================================================================

@test "--crit で ~/.crit/prompts/on_finish_approved.md を data の複製へ張る" {
  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ -L "$CRIT_DST" ]
  [ "$(readlink "$CRIT_DST")" = "$DATA/crit/on_finish_approved.md" ]
  cmp -s "$CRIT_DST" "$ROOT/crit/on_finish_approved.md"
}

@test "--crit でも既存のリンクが別の場所を指していれば触らずに報告する" {
  mkdir -p "$HOME/.crit/prompts" "$WORKDIR/elsewhere"
  printf 'x\n' >"$WORKDIR/elsewhere/on_finish_approved.md"
  ln -s "$WORKDIR/elsewhere/on_finish_approved.md" "$CRIT_DST"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ "$(readlink "$CRIT_DST")" = "$WORKDIR/elsewhere/on_finish_approved.md" ]
  [[ "$output" == *"$WORKDIR/elsewhere/on_finish_approved.md"* ]]
}

@test "--crit で隣の版 (更新前の plugin の置き場) を指すリンクは data の複製へ張り替える" {
  mkdir -p "$HOME/.crit/prompts" "$WORKDIR/0.0.9/crit"
  printf 'old\n' >"$WORKDIR/0.0.9/crit/on_finish_approved.md"
  ln -s "$WORKDIR/0.0.9/crit/on_finish_approved.md" "$CRIT_DST"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ "$(readlink "$CRIT_DST")" = "$DATA/crit/on_finish_approved.md" ]
  [[ "$output" == *"張り替え"* ]]
}

@test "--crit で今の版の置き場を直接指すリンクも data の複製へ張り替える" {
  mkdir -p "$HOME/.crit/prompts"
  ln -s "$ROOT/crit/on_finish_approved.md" "$CRIT_DST"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ "$(readlink "$CRIT_DST")" = "$DATA/crit/on_finish_approved.md" ]
}

@test "--crit で指し先が消えたリンクは張り替える" {
  mkdir -p "$HOME/.crit/prompts"
  ln -s "$WORKDIR/gone/on_finish_approved.md" "$CRIT_DST"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ "$(readlink "$CRIT_DST")" = "$DATA/crit/on_finish_approved.md" ]
}

@test "--crit で ~/.crit に書けなければ --crit 付きの ! 前置のコマンドを出して exit 1 する" {
  mkdir -p "$HOME/.crit"
  chmod a-w "$HOME/.crit"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 1 ]
  [[ "$output" == *"! CLAUDE_PLUGIN_DATA=\"$DATA\" \"$SCRIPT_PATH\" --crit"* ]]
}

@test "--crit 無しでは crit がありリンクが無くても張らず、--crit を案内する" {
  mock_cmd crit

  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [ ! -e "$CRIT_DST" ]
  [[ "$output" == *"--crit"* ]]
}

@test "crit が無ければ --crit を案内しない" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [[ "$output" != *"--crit"* ]]
}

# =============================================================================
# 引数
# =============================================================================

@test "不明なオプションは usage を出して exit 1 する" {
  run "$SCRIPT_PATH" --bogus
  [ "$status" -eq 1 ]
  [[ "$output" == *"usage"* ]]
}
