#!/usr/bin/env bats
# setup.sh のテスト
#
# スクリプトは自分の親ディレクトリを plugin root として扱う。実リポジトリの bin/ や
# ~/.crit を巻き込まないよう、scripts/ と crit/ を隔離ツリーへ複製し、HOME も差し替える。
# go / npx / jq / crit は PATH 上のモックで置き換え、ビルドと検証は呼び出しの記録だけを見る。
bats_require_minimum_version 1.5.0

load lib/hook-test-helpers

setup() {
  WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/setup-test.XXXXXX")
  export WORKDIR
  ROOT="$WORKDIR/plugin"
  SRC="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  mkdir -p "$ROOT"
  cp -R "$SRC/scripts" "$ROOT/scripts"
  cp -R "$SRC/crit" "$ROOT/crit"
  SCRIPT_PATH="$ROOT/scripts/setup.sh"

  MOCK_PATH="$WORKDIR/bin"
  mkdir -p "$MOCK_PATH"
  HOME="$WORKDIR/home"
  mkdir -p "$HOME"
  export HOME
  # jq は本物を使う (結果 JSON の読み取りに要る)。ディレクトリごと PATH に足すと
  # 同居する npx や crit まで見えてしまうので、jq だけを symlink で持ち込む
  ln -s "$(command -v jq)" "$MOCK_PATH/jq"
  PATH="$MOCK_PATH:/usr/bin:/bin"
  export PATH

  mock_go
  mock_cmd npx
}

teardown() {
  chmod -R u+w "$WORKDIR"
  rm -rf "$WORKDIR"
}

# mock_cmd NAME -> 何もしない実行ファイルを PATH に置く
mock_cmd() {
  printf '#!/bin/bash\nexit 0\n' >"$MOCK_PATH/$1"
  chmod +x "$MOCK_PATH/$1"
}

# mock_go [ERRORS] -> `go build -C ROOT -o OUT ./...` を受け、OUT/writing-gate に
# scan の結果として errors=ERRORS の固定 JSON を返す偽バイナリを置く。呼び出しは go.log に記録する。
mock_go() {
  local errors="${1:-1}"
  {
    printf '#!/bin/bash\n'
    printf "printf '%%s\\\\n' '{\"errors\":%s,\"warnings\":0,\"findings\":[]}'\n" "$errors"
  } >"$WORKDIR/fake-gate"
  cat >"$MOCK_PATH/go" <<MOCK
#!/bin/bash
printf '%s\n' "\$*" >>"$WORKDIR/go.log"
root=""; out=""
while [ "\$#" -gt 0 ]; do
  case "\$1" in
    -C) root="\$2"; shift 2 ;;
    -o) out="\$2"; shift 2 ;;
    *) shift ;;
  esac
done
mkdir -p "\$root/\$out"
cp "$WORKDIR/fake-gate" "\$root/\$out/writing-gate"
chmod +x "\$root/\$out/writing-gate"
MOCK
  chmod +x "$MOCK_PATH/go"
}

# =============================================================================
# 前提コマンド
# =============================================================================

@test "go が無ければその名前を出して exit 1 し、ビルドしない" {
  rm "$MOCK_PATH/go"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"go"* ]]
  [ ! -e "$ROOT/bin/writing-gate" ]
}

@test "npx が無ければその名前を出して exit 1 する" {
  rm "$MOCK_PATH/npx"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [[ "$output" == *"npx"* ]]
  [ ! -f "$WORKDIR/go.log" ]
}

# =============================================================================
# ビルドと検証
# =============================================================================

@test "writing-gate を plugin root の bin/ へビルドし、検証まで通して exit 0 する" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [ -x "$ROOT/bin/writing-gate" ]
  grep -q -- "-C $ROOT" "$WORKDIR/go.log"
  [[ "$output" == *"writing-gate"* ]]
}

@test "検証の probe と一時ファイルを残さない" {
  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  run find "$ROOT/tmp" -mindepth 1
  [ -z "$output" ]
}

@test "bin/ に書けなければ go を起動せず、! 前置のコマンドを出して exit 1 する" {
  mkdir -p "$ROOT/bin"
  chmod a-w "$ROOT/bin"

  run "$SCRIPT_PATH"
  [ "$status" -eq 1 ]
  [ ! -f "$WORKDIR/go.log" ]
  [[ "$output" == *"! \"$SCRIPT_PATH\""* ]]
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

# =============================================================================
# crit の prompt リンク
# =============================================================================

@test "--crit で ~/.crit/prompts/on_finish_approved.md を plugin のファイルへ張る" {
  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ -L "$HOME/.crit/prompts/on_finish_approved.md" ]
  [ "$(readlink "$HOME/.crit/prompts/on_finish_approved.md")" = "$ROOT/crit/on_finish_approved.md" ]
}

@test "--crit でも既存のリンクが別の場所を指していれば触らずに報告する" {
  mkdir -p "$HOME/.crit/prompts" "$WORKDIR/elsewhere"
  printf 'x\n' >"$WORKDIR/elsewhere/on_finish_approved.md"
  ln -s "$WORKDIR/elsewhere/on_finish_approved.md" "$HOME/.crit/prompts/on_finish_approved.md"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ "$(readlink "$HOME/.crit/prompts/on_finish_approved.md")" = "$WORKDIR/elsewhere/on_finish_approved.md" ]
  [[ "$output" == *"$WORKDIR/elsewhere/on_finish_approved.md"* ]]
}

@test "--crit で隣の版 (更新前の plugin) を指すリンクは張り替える" {
  mkdir -p "$HOME/.crit/prompts" "$WORKDIR/0.0.9/crit"
  printf 'old\n' >"$WORKDIR/0.0.9/crit/on_finish_approved.md"
  ln -s "$WORKDIR/0.0.9/crit/on_finish_approved.md" "$HOME/.crit/prompts/on_finish_approved.md"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ "$(readlink "$HOME/.crit/prompts/on_finish_approved.md")" = "$ROOT/crit/on_finish_approved.md" ]
  [[ "$output" == *"張り替え"* ]]
}

@test "--crit で指し先が消えたリンクは張り替える" {
  mkdir -p "$HOME/.crit/prompts"
  ln -s "$WORKDIR/gone/on_finish_approved.md" "$HOME/.crit/prompts/on_finish_approved.md"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 0 ]
  [ "$(readlink "$HOME/.crit/prompts/on_finish_approved.md")" = "$ROOT/crit/on_finish_approved.md" ]
}

@test "--crit で ~/.crit に書けなければ --crit 付きの ! 前置のコマンドを出して exit 1 する" {
  mkdir -p "$HOME/.crit"
  chmod a-w "$HOME/.crit"

  run "$SCRIPT_PATH" --crit
  [ "$status" -eq 1 ]
  [[ "$output" == *"! \"$SCRIPT_PATH\" --crit"* ]]
}

@test "--crit 無しでは crit がありリンクが無くても張らず、--crit を案内する" {
  mock_cmd crit

  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]
  [ ! -e "$HOME/.crit/prompts/on_finish_approved.md" ]
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
