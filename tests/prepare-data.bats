#!/usr/bin/env bats
# prepare-data.sh のテスト
#
# plugin の data ディレクトリ (${CLAUDE_PLUGIN_DATA}) へ writing-gate と crit の prompt を用意する。
# go は PATH 上のモックで置き換え、ビルドは呼び出しの記録だけを見る。
bats_require_minimum_version 1.5.0

load lib/plugin-tree-helpers

setup() {
  setup_plugin_tree
  SCRIPT_PATH="$ROOT/scripts/prepare-data.sh"
  mock_go
}

teardown() {
  teardown_plugin_tree
}

# =============================================================================
# 入力の検証
# =============================================================================

@test "CLAUDE_PLUGIN_DATA が無ければ exit 1 し、ビルドしない" {
  unset CLAUDE_PLUGIN_DATA

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"CLAUDE_PLUGIN_DATA"* ]]
  [ "$(go_calls)" -eq 0 ]
}

@test "CLAUDE_PLUGIN_DATA が相対パスなら exit 1 する" {
  CLAUDE_PLUGIN_DATA="relative/data" run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"絶対パス"* ]]
  [ "$(go_calls)" -eq 0 ]
}

@test "go が無ければ exit 1 し、data の bin/ にバイナリを置かない" {
  rm "$MOCK_PATH/go"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"go"* ]]
  [ ! -e "$DATA/bin/writing-gate" ]
}

# =============================================================================
# ビルドと検証
# =============================================================================

@test "data の bin/ へビルドし、検証まで通して exit 0 する" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [ -x "$DATA/bin/writing-gate" ]
  grep -q -- "-C $ROOT" "$WORKDIR/go.log"
}

@test "plugin root には書き込まない" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [ ! -e "$ROOT/bin" ]
  [ ! -e "$ROOT/tmp" ]
}

@test "検証の probe と一時ファイルを残さない" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  run find "$DATA/tmp" -mindepth 1
  [ -z "$output" ]
}

@test "go build が失敗したら stderr をそのまま流して exit 1 する" {
  printf '#!/bin/bash\nprintf "boom: go.mod requires go >= 9.9\\n" >&2\nexit 1\n' >"$MOCK_PATH/go"
  chmod +x "$MOCK_PATH/go"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"go.mod requires go >= 9.9"* ]]
}

@test "ビルドしたバイナリが既知の漏出パターンを拾えなければ exit 1 し、既存のバイナリを残す" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  cp "$DATA/bin/writing-gate" "$WORKDIR/good-gate"
  mock_go 0

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"process-leak"* ]]
  cmp -s "$DATA/bin/writing-gate" "$WORKDIR/good-gate"
}

# =============================================================================
# --if-stale
# =============================================================================

@test "--if-stale は入力が前回のビルドと同じならビルドしない" {
  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 1 ]

  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 1 ]
}

@test "--if-stale は Go のソースが変わったらビルドし直す" {
  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  printf '\n// changed\n' >>"$ROOT/writing-gate/main.go"

  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 2 ]
}

@test "--if-stale は埋め込みルール (rules.json) が変わったらビルドし直す" {
  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  printf '\n' >>"$ROOT/writing-gate/internal/rules/rules.json"

  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 2 ]
}

@test "--if-stale は plugin root の場所が変わっても入力が同じならビルドしない" {
  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  mv "$ROOT" "$WORKDIR/plugin-next"

  run "$WORKDIR/plugin-next/scripts/prepare-data.sh" --if-stale
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 1 ]
}

@test "--if-stale でもバイナリが消えていればビルドする" {
  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  rm "$DATA/bin/writing-gate"

  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 2 ]
  [ -x "$DATA/bin/writing-gate" ]
}

@test "--if-stale 無しなら入力が同じでもビルドする" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]

  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 2 ]
}

@test "検証に失敗したビルドは stamp を残さず、次の --if-stale でビルドし直す" {
  mock_go 0
  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 1 ]
  mock_go 1

  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 2 ]
}

# =============================================================================
# --check
# =============================================================================

# freeze_writes -> data と HOME と TMPDIR を書き込み禁止にする。--check が何か書けば失敗する
freeze_writes() {
  mkdir -p "$WORKDIR/ro-tmp"
  chmod -R a-w "$DATA" "$HOME" "$WORKDIR/ro-tmp"
  export TMPDIR="$WORKDIR/ro-tmp"
}

@test "--check は準備済みなら何も書かずに exit 0 し、ビルドしない" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  freeze_writes

  run "$SCRIPT_PATH" --check
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 1 ]
}

@test "--check はバイナリが無ければその旨を出して exit 1 し、data を作らない" {
  run "$SCRIPT_PATH" --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"writing-gate"* ]]
  [ "$(go_calls)" -eq 0 ]
  [ ! -e "$DATA" ]
}

@test "--check はビルドの入力が前回と違えば exit 1 する" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  printf '\n' >>"$ROOT/writing-gate/internal/rules/rules.json"
  freeze_writes

  run "$SCRIPT_PATH" --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"前回のビルドと違"* ]]
  [ "$(go_calls)" -eq 1 ]
}

@test "--check は置かれたバイナリが既知の漏出パターンを拾えなければ exit 1 する" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  printf '#!/bin/bash\nprintf "%%s\\n" "{\\"errors\\":0}"\n' >"$DATA/bin/writing-gate"
  freeze_writes

  run "$SCRIPT_PATH" --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"process-leak"* ]]
}

@test "--check は data の crit の prompt が古ければ exit 1 する" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  printf '\n追記\n' >>"$ROOT/crit/on_finish_approved.md"
  freeze_writes

  run "$SCRIPT_PATH" --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"crit"* ]]
  [ "$(go_calls)" -eq 1 ]
}

@test "--check は欠けが複数あればすべて出す" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  printf '\n' >>"$ROOT/writing-gate/internal/rules/rules.json"
  printf '\n追記\n' >>"$ROOT/crit/on_finish_approved.md"
  freeze_writes

  run "$SCRIPT_PATH" --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"前回のビルドと違"* ]]
  [[ "$output" == *"crit"* ]]
}

# =============================================================================
# crit の prompt
# =============================================================================

@test "crit の prompt を data の crit/ へ複製する" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  cmp -s "$ROOT/crit/on_finish_approved.md" "$DATA/crit/on_finish_approved.md"
}

@test "--if-stale でビルドしないときも crit の prompt は新しい版の内容に揃える" {
  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  printf '\n追記\n' >>"$ROOT/crit/on_finish_approved.md"

  run "$SCRIPT_PATH" --if-stale
  [ "$status" -eq 0 ]
  [ "$(go_calls)" -eq 1 ]
  cmp -s "$ROOT/crit/on_finish_approved.md" "$DATA/crit/on_finish_approved.md"
}

# =============================================================================
# 引数
# =============================================================================

@test "不明なオプションは usage を出して exit 1 する" {
  run "$SCRIPT_PATH" --bogus
  [ "$status" -eq 1 ]
  [[ "$output" == *"usage"* ]]
}
