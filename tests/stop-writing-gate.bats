#!/usr/bin/env bats
# stop-writing-gate.sh のテスト
#
# hook は自分の ../bin/writing-gate を絶対パスで起動する (PATH は見ない)。
# 実リポジトリの shared/bin/ を巻き込まないよう、隔離ツリーに複製して実行する。
bats_require_minimum_version 1.5.0

load lib/hook-test-helpers

setup_file() { setup_mock_bin; }
teardown_file() { teardown_mock_bin; }

setup() {
  TEST_TMPDIR="$TEST_FILE_TMPDIR/test-$BATS_TEST_NUMBER"
  mkdir -p "$TEST_TMPDIR"

  FAKE_HOOKS=$(setup_fake_hooks_tree "$TEST_TMPDIR/fake")
  FAKE_BIN="$TEST_TMPDIR/fake/bin"
  SCRIPT_PATH="$FAKE_HOOKS/stop-writing-gate.sh"

  TRANSCRIPT="$TEST_TMPDIR/session.jsonl"
  printf '{}\n' > "$TRANSCRIPT"
}

teardown() {
  rm -rf "$TEST_TMPDIR"
}

# writing-gate の代わりに固定の JSON を返す mock を置く
write_mock_gate() {
  local payload="$1"
  cat > "$FAKE_BIN/writing-gate" << MOCKEOF
#!/bin/bash
printf '%s\n' '$payload'
exit 0
MOCKEOF
  chmod +x "$FAKE_BIN/writing-gate"
}

# Stop hook の入力 JSON
make_stop_input() {
  local transcript="$1" active="${2:-false}"
  jq -cn --arg t "$transcript" --argjson a "$active" \
    '{transcript_path:$t, stop_hook_active:$a}'
}

@test "block を返されたら reason へ別セッション案内を追記して stdout へ出す" {
  write_mock_gate '{"decision":"block","reason":"だめだに"}'

  run_hook "$SCRIPT_PATH" <<< "$(make_stop_input "$TRANSCRIPT")"
  [ "$status" -eq 0 ]
  [ "$(jq -r '.decision' <<< "$output")" = "block" ]
  [[ "$(jq -r '.reason' <<< "$output")" == *"だめだに"* ]]
  [[ "$(jq -r '.reason' <<< "$output")" == *"別セッション"* ]]
}

@test "指摘が無ければ何も出さない" {
  write_mock_gate '{}'

  run_hook "$SCRIPT_PATH" <<< "$(make_stop_input "$TRANSCRIPT")"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "stop_hook_active なら判定せず通す" {
  write_mock_gate '{"decision":"block","reason":"だめだに"}'

  run_hook "$SCRIPT_PATH" <<< "$(make_stop_input "$TRANSCRIPT" true)"
  [ "$status" -eq 0 ]
  [[ "$output" != *"block"* ]]
}

@test "transcript が無ければ判定せず通す" {
  write_mock_gate '{"decision":"block","reason":"だめだに"}'

  run_hook "$SCRIPT_PATH" <<< "$(make_stop_input "$TEST_TMPDIR/missing.jsonl")"
  [ "$status" -eq 0 ]
  [[ "$output" != *"block"* ]]
}

@test "writing-gate が未ビルドなら block してその事実を伝える" {
  run_hook "$SCRIPT_PATH" <<< "$(make_stop_input "$TRANSCRIPT")"
  [ "$status" -eq 0 ]
  [ "$(jq -r '.decision' <<< "$output")" = "block" ]
  [[ "$(jq -r '.reason' <<< "$output")" == *"ビルドして"* ]]
}

@test "PATH 上の writing-gate は使わない (../bin/ のみ)" {
  cat > "$MOCK_BIN/writing-gate" << 'MOCKEOF'
#!/bin/bash
printf '%s\n' '{"decision":"block","reason":"PATH 側が動いた"}'
MOCKEOF
  chmod +x "$MOCK_BIN/writing-gate"
  PATH="$MOCK_BIN:$PATH" run_hook "$SCRIPT_PATH" <<< "$(make_stop_input "$TRANSCRIPT")"

  [ "$status" -eq 0 ]
  [[ "$output" != *"PATH 側が動いた"* ]]
}

@test "writing-gate が失敗しても文章の不備として止めない" {
  cat > "$FAKE_BIN/writing-gate" << 'MOCKEOF'
#!/bin/bash
echo "boom" >&2
exit 1
MOCKEOF
  chmod +x "$FAKE_BIN/writing-gate"

  run_hook "$SCRIPT_PATH" <<< "$(make_stop_input "$TRANSCRIPT")"
  [ "$status" -eq 0 ]
  [[ "$output" != *"block"* ]]
}
