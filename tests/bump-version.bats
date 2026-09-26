#!/usr/bin/env bats
# bump-version.sh のテスト
#
# plugin.json の version を calver (YYYY.MMDD.NN) で 1 つ進める。
# 隔離ツリーの plugin.json を書き換え、日付は --date で固定する。
bats_require_minimum_version 1.5.0

setup() {
  WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/bump-test.XXXXXX")
  SRC="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  ROOT="$WORKDIR/plugin"
  mkdir -p "$ROOT/.claude-plugin"
  cp -R "$SRC/scripts" "$ROOT/scripts"
  MANIFEST="$ROOT/.claude-plugin/plugin.json"
  SCRIPT_PATH="$ROOT/scripts/bump-version.sh"
}

teardown() {
  rm -rf "$WORKDIR"
}

# write_manifest VERSION -> version 以外のフィールドも持つ manifest を置く
write_manifest() {
  jq -n --arg v "$1" '{name: "japanese-writer", version: $v, license: "MIT"}' >"$MANIFEST"
}

version() {
  jq -r '.version' "$MANIFEST"
}

@test "前日の版からは今日の 01 へ進める" {
  write_manifest "2026.0925.03"

  run "$SCRIPT_PATH" --date 2026-09-26
  [ "$status" -eq 0 ]
  [ "$(version)" = "2026.0926.01" ]
  [[ "$output" == *"2026.0926.01"* ]]
}

@test "同じ日の版からは連番を 1 つ上げる" {
  write_manifest "2026.0926.01"

  run "$SCRIPT_PATH" --date 2026-09-26
  [ "$status" -eq 0 ]
  [ "$(version)" = "2026.0926.02" ]
}

@test "連番の繰り上がりは 10 進で数える (08 -> 09 -> 10)" {
  write_manifest "2026.0926.08"
  run "$SCRIPT_PATH" --date 2026-09-26
  [ "$(version)" = "2026.0926.09" ]

  run "$SCRIPT_PATH" --date 2026-09-26
  [ "$status" -eq 0 ]
  [ "$(version)" = "2026.0926.10" ]
}

@test "同じ日に 99 を超えるなら書き換えずに exit 1 する" {
  write_manifest "2026.0926.99"

  run "$SCRIPT_PATH" --date 2026-09-26
  [ "$status" -eq 1 ]
  [ "$(version)" = "2026.0926.99" ]
}

@test "今日より新しい版が入っていたら書き換えずに exit 1 する" {
  write_manifest "2026.0927.01"

  run "$SCRIPT_PATH" --date 2026-09-26
  [ "$status" -eq 1 ]
  [ "$(version)" = "2026.0927.01" ]
}

@test "version 以外のフィールドを保つ" {
  write_manifest "2026.0925.01"

  run "$SCRIPT_PATH" --date 2026-09-26
  [ "$status" -eq 0 ]
  [ "$(jq -r '.name' "$MANIFEST")" = "japanese-writer" ]
  [ "$(jq -r '.license' "$MANIFEST")" = "MIT" ]
}

@test "不正な --date は書き換えずに exit 1 する" {
  write_manifest "2026.0925.01"

  run "$SCRIPT_PATH" --date 2026/09/26
  [ "$status" -eq 1 ]
  [ "$(version)" = "2026.0925.01" ]
}
