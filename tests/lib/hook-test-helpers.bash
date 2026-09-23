# hook-test-helpers.bash
# フックテスト用の共通ヘルパー

# hook スクリプトを本番と同じ bash で実行する。
# 本番の hook shebang は #!/bin/bash (macOS の /bin/bash 3.2)。
# PATH 経由の `bash` (homebrew の 5.3 等) だと bash 4+ builtin (mapfile 等) の
# 非互換が検出できずすり抜けるため、テストは既定で /bin/bash に固定する。
# 別バージョンで検証したい場合は HOOK_BASH=/path/to/bash を設定する。
run_hook() {
  run "${HOOK_BASH:-/bin/bash}" "$@"
}

# PreToolUse の JSON 入力を生成 (jq -Rs で値エスケープのみ)
make_input() {
  local escaped
  escaped=$(printf '%s' "$1" | jq -Rs .)
  printf '{"tool_name":"Bash","tool_input":{"command":%s}}' "$escaped"
}

# MOCK_BIN + hook-logger.sh の共通セットアップ (setup_file から呼ぶ)
setup_mock_bin() {
  TEST_FILE_TMPDIR=$(mktemp -d "${TMPDIR:-/tmp}/claude-test.XXXXXX")
  export TEST_FILE_TMPDIR
  MOCK_BIN="$TEST_FILE_TMPDIR/bin"
  mkdir -p "$MOCK_BIN"
  local hooks_dir
  hooks_dir="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/hooks"
  mkdir -p "$MOCK_BIN/lib"
  cp "$hooks_dir/lib/hook-logger.sh" "$MOCK_BIN/lib/hook-logger.sh"
  export MOCK_BIN
}

teardown_mock_bin() {
  [ -n "$TEST_FILE_TMPDIR" ] && rm -rf "$TEST_FILE_TMPDIR"
}

# hooks/ と空の bin/ を持つ隔離ツリーを作り、hooks ディレクトリのパスを stdout に返す。
# hook は自分の ../bin/ のバイナリを絶対パスで起動するため、
# 実リポジトリの bin/ を巻き込まずに未ビルド時の挙動をテストするのに使う。
setup_fake_hooks_tree() {
  local root="$1"
  local src
  src="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/hooks"
  mkdir -p "$root/bin"
  cp -R "$src" "$root/hooks"
  printf '%s\n' "$root/hooks"
}
