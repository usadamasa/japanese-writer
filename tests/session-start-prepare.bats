#!/usr/bin/env bats
# session-start-prepare.sh のテスト
#
# SessionStart hook として prepare-data.sh --if-stale を呼ぶ。成功時は何も出さず
# (stdout は会話の文脈へ足される)、失敗時は systemMessage で利用者へ知らせてセッションは止めない。
bats_require_minimum_version 1.5.0

load lib/hook-test-helpers
load lib/plugin-tree-helpers

setup() {
  setup_plugin_tree
  SCRIPT_PATH="$ROOT/hooks/session-start-prepare.sh"
  mock_go
}

teardown() {
  teardown_plugin_tree
}

session_start_input() {
  printf '{"hook_event_name":"SessionStart","source":"startup"}'
}

@test "未ビルドなら data へビルドし、何も出さずに exit 0 する" {
  run_hook "$SCRIPT_PATH" <<< "$(session_start_input)"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  [ -x "$DATA/bin/writing-gate" ]
}

@test "2 回目は入力が同じならビルドしない" {
  run_hook "$SCRIPT_PATH" <<< "$(session_start_input)"
  run_hook "$SCRIPT_PATH" <<< "$(session_start_input)"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  [ "$(go_calls)" -eq 1 ]
}

@test "準備に失敗したら systemMessage で理由と setup を案内し、exit 0 する" {
  printf '#!/bin/bash\nprintf "boom: go.mod requires go >= 9.9\\n" >&2\nexit 1\n' >"$MOCK_PATH/go"
  chmod +x "$MOCK_PATH/go"

  run_hook "$SCRIPT_PATH" <<< "$(session_start_input)"
  [ "$status" -eq 0 ]
  msg=$(jq -r '.systemMessage' <<< "$output")
  [[ "$msg" == *"go.mod requires go >= 9.9"* ]]
  [[ "$msg" == *"/japanese-writer:setup"* ]]
}
