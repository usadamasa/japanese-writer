#!/usr/bin/env bats
# textlint-run.sh のテスト
# スクリプト自身の責務 (引数の検証、pass ごとの config の振り分け、結果 JSON のまとめ方) だけを見る。
# textlint の振る舞いは検証の対象にしない。npx は固定の JSON を返すモックに差し替える。
bats_require_minimum_version 1.5.0

SCRIPT_PATH="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/bin/textlint-run.sh"

setup() {
  WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/textlint-run-test.XXXXXX")
  export WORKDIR
  mkdir -p "$WORKDIR/bin" "$WORKDIR/out"
  CONFIG="$WORKDIR/.textlintrc.json"
  TARGET="$WORKDIR/doc.md"
  OUTPUT="$WORKDIR/out/result.json"
  export CONFIG TARGET OUTPUT
  printf '{"rules":{}}\n' >"$CONFIG"
  printf '# doc\n' >"$TARGET"
  PATH="$WORKDIR/bin:$PATH"
  export PATH
}

teardown() {
  [ -n "$WORKDIR" ] && rm -rf "$WORKDIR"
}

# mock_npx BODY -> $WORKDIR/bin/npx を BODY で作る
mock_npx() {
  {
    printf '#!/bin/bash\n'
    printf '%s\n' "$1"
  } >"$WORKDIR/bin/npx"
  chmod +x "$WORKDIR/bin/npx"
}

# fix パスと lint パスで別々の JSON を返すモック。
# 終了コードはどちらも 1 にする (textlint は指摘が残ると 1 を返す)。
mock_npx_two_pass() {
  cat >"$WORKDIR/bin/npx" <<'MOCK'
#!/bin/bash
for arg in "$@"; do
  if [ "$arg" = "--fix" ]; then
    printf '%s\n' '[{"filePath":"/x/doc.md","applyingMessages":[{"ruleId":"prh","line":3,"column":1,"message":"fixed"}],"remainingMessages":[]}]'
    exit 1
  fi
done
printf '%s\n' '[{"filePath":"/x/doc.md","messages":[{"ruleId":"sentence-length","severity":2,"line":5,"column":2,"message":"too long"}]}]'
exit 1
MOCK
  chmod +x "$WORKDIR/bin/npx"
}

# =============================================================================
# 入力検証
# =============================================================================

@test "不正な引数はエラー終了する" {
  mock_npx_two_pass
  # ラベル / 引数 の組。いずれも textlint を起動する前に弾かれる
  run "$SCRIPT_PATH"
  [ "$status" -ne 0 ]

  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT"
  [ "$status" -ne 0 ]

  run "$SCRIPT_PATH" --config "./relative.json" --output "$OUTPUT" "$TARGET"
  [ "$status" -ne 0 ]
  [[ "$output" == *"絶対パス"* ]]

  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "./relative.md"
  [ "$status" -ne 0 ]
  [[ "$output" == *"絶対パス"* ]]

  run "$SCRIPT_PATH" --config "$WORKDIR/missing.json" --output "$OUTPUT" "$TARGET"
  [ "$status" -ne 0 ]

  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "$WORKDIR/missing.md"
  [ "$status" -ne 0 ]

  run "$SCRIPT_PATH" --config "$CONFIG" --output "$WORKDIR/no-such-dir/result.json" "$TARGET"
  [ "$status" -ne 0 ]

  [ ! -e "$OUTPUT" ]
}

# =============================================================================
# 結果 JSON
# =============================================================================

@test "fix パスの適用結果と lint パスの残存指摘を 1 つの JSON にまとめる" {
  mock_npx_two_pass
  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "$TARGET"
  [ "$status" -eq 0 ]

  # applied_fixes は --fix の applyingMessages 由来
  run jq -r '.applied_fixes | length' "$OUTPUT"
  [ "$output" = "1" ]
  run jq -r '.applied_fixes[0].ruleId' "$OUTPUT"
  [ "$output" = "prh" ]

  # remaining_issues は再 lint の messages 由来 (修正後のファイルに対する位置)
  run jq -r '.remaining_issues | length' "$OUTPUT"
  [ "$output" = "1" ]
  run jq -r '.remaining_issues[0].ruleId' "$OUTPUT"
  [ "$output" = "sentence-length" ]
  run jq -r '.remaining_issues[0].line' "$OUTPUT"
  [ "$output" = "5" ]
}

# fix パスが書き込み権限エラーで空の stdout を返すモック。
# .claude/ 配下など sandbox が書き込みを拒否するパスに --fix を当てると textlint 自身が
# これと同じ挙動 (stderr にエラー、stdout は空、exit 1) になる。
mock_npx_fix_empty_stdout() {
  cat >"$WORKDIR/bin/npx" <<'MOCK'
#!/bin/bash
for arg in "$@"; do
  if [ "$arg" = "--fix" ]; then
    printf 'Unexpected error during file processing: Error: EACCES: permission denied\n' >&2
    exit 1
  fi
done
printf '%s\n' '[{"filePath":"/x/doc.md","messages":[]}]'
exit 0
MOCK
  chmod +x "$WORKDIR/bin/npx"
}

@test "fix パスの stdout が空ならエラー終了し、結果 JSON を残さない" {
  mock_npx_fix_empty_stdout
  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "$TARGET"

  [ "$status" -ne 0 ]
  [[ "$output" == *"textlint (fix) の実行に失敗しました"* ]]
  [[ "$output" == *"EACCES"* ]]
  [ ! -e "$OUTPUT" ]
}

# 対象が .textlintignore などで無視されると、textlint は指摘 0 件の配列ではなく
# 空配列 `[]` を exit 0 で返す (実測)。これは失敗ではないので run_pass は通す必要がある。
mock_npx_empty_array() {
  cat >"$WORKDIR/bin/npx" <<'MOCK'
#!/bin/bash
printf '%s\n' '[]'
exit 0
MOCK
  chmod +x "$WORKDIR/bin/npx"
}

@test "textlint に無視されて空配列 [] が返っても成功扱いにする" {
  mock_npx_empty_array
  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "$TARGET"

  [ "$status" -eq 0 ]
  run jq -r '.applied_fixes | length' "$OUTPUT"
  [ "$output" = "0" ]
  run jq -r '.remaining_issues | length' "$OUTPUT"
  [ "$output" = "0" ]
}

@test "サマリを stdout に出す" {
  mock_npx_two_pass
  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "$TARGET"

  [ "$status" -eq 0 ]
  [[ "$output" == *"applied_fixes=1"* ]]
  [[ "$output" == *"remaining_issues=1"* ]]
}

@test "中間ファイルを出力先に残さない" {
  mock_npx_two_pass
  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "$TARGET"
  [ "$status" -eq 0 ]

  run find "$WORKDIR/out" -mindepth 1 -maxdepth 1
  [ "$output" = "$OUTPUT" ]
}

# =============================================================================
# fix pass と lint pass の config 振り分け
#
# 語の置換で済ませると文の骨格が歪んだまま残るため、autofix には表記ゆれ用の
# 別 config を当てる。どちらの pass にどの config が渡ったかを記録する mock で見る。
# =============================================================================

# 各 pass に渡った --config を CONFIG_LOG に記録する
mock_npx_log_config() {
  cat >"$WORKDIR/bin/npx" <<MOCK
#!/bin/bash
pass=lint
config=""
prev=""
for arg in "\$@"; do
  [ "\$prev" = "--config" ] && config="\$arg"
  [ "\$arg" = "--fix" ] && pass=fix
  prev="\$arg"
done
printf '%s %s\n' "\$pass" "\$config" >> "$WORKDIR/config.log"
printf '%s\n' '[{"filePath":"/x/doc.md","messages":[]}]'
MOCK
  chmod +x "$WORKDIR/bin/npx"
}

@test "--fix-config を渡すと autofix だけ別 config で走る" {
  mock_npx_log_config
  FIX_CONFIG="$WORKDIR/fix.json"
  printf '{"rules":{}}\n' >"$FIX_CONFIG"

  run "$SCRIPT_PATH" --config "$CONFIG" --fix-config "$FIX_CONFIG" --output "$OUTPUT" "$TARGET"
  [ "$status" -eq 0 ]

  grep -qx "fix $FIX_CONFIG" "$WORKDIR/config.log"
  grep -qx "lint $CONFIG" "$WORKDIR/config.log"
}

@test "--fix-config 省略時は config と同じ場所の fix.textlintrc.json を使う" {
  mock_npx_log_config
  SIBLING="$WORKDIR/fix.textlintrc.json"
  printf '{"rules":{}}\n' >"$SIBLING"

  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "$TARGET"
  [ "$status" -eq 0 ]

  grep -qx "fix $SIBLING" "$WORKDIR/config.log"
  grep -qx "lint $CONFIG" "$WORKDIR/config.log"
}

@test "fix 用 config が無ければ両方の pass で同じ config を使う" {
  mock_npx_log_config

  run "$SCRIPT_PATH" --config "$CONFIG" --output "$OUTPUT" "$TARGET"
  [ "$status" -eq 0 ]

  grep -qx "fix $CONFIG" "$WORKDIR/config.log"
  grep -qx "lint $CONFIG" "$WORKDIR/config.log"
}

@test "--fix-config が存在しなければエラー終了する" {
  mock_npx_log_config

  run "$SCRIPT_PATH" --config "$CONFIG" --fix-config "$WORKDIR/missing.json" --output "$OUTPUT" "$TARGET"
  [ "$status" -ne 0 ]
  [ ! -f "$WORKDIR/config.log" ]
}
