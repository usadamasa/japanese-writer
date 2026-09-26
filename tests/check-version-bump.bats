#!/usr/bin/env bats
# check-version-bump.sh のテスト
#
# 配布物に差分があるのに plugin.json の version が base と同じなら落とす。
# 使い捨ての git リポジトリで base と HEAD を作って確かめる。
bats_require_minimum_version 1.5.0

setup() {
  WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/check-version-test.XXXXXX")
  SRC="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  REPO="$WORKDIR/repo"
  mkdir -p "$REPO/.claude-plugin" "$REPO/skills/a" "$REPO/tests" "$REPO/.github/workflows" "$REPO/scripts"
  cp "$SRC/scripts/check-version-bump.sh" "$REPO/scripts/check-version-bump.sh"

  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
  git -C "$REPO" init -q -b main
  git -C "$REPO" config user.email test@example.com
  git -C "$REPO" config user.name test
  set_version "2026.0925.01"
  printf 'skill\n' >"$REPO/skills/a/SKILL.md"
  printf 'test\n' >"$REPO/tests/a.bats"
  printf 'ci\n' >"$REPO/.github/workflows/ci.yaml"
  commit "base"
  git -C "$REPO" branch base
}

teardown() {
  rm -rf "$WORKDIR"
}

set_version() {
  jq -n --arg v "$1" '{name: "japanese-writer", version: $v}' >"$REPO/.claude-plugin/plugin.json"
}

commit() {
  git -C "$REPO" add -A
  git -C "$REPO" commit -q -m "$1"
}

check() {
  run bash -c 'cd "$1" && ./scripts/check-version-bump.sh base' _ "$REPO"
}

@test "配布物を変えて版を上げていれば通す" {
  printf 'changed\n' >>"$REPO/skills/a/SKILL.md"
  set_version "2026.0926.01"
  commit "change"

  check
  [ "$status" -eq 0 ]
}

@test "配布物を変えたのに版が同じなら task bump を案内して落とす" {
  printf 'changed\n' >>"$REPO/skills/a/SKILL.md"
  commit "change"

  check
  [ "$status" -eq 1 ]
  [[ "$output" == *"task bump"* ]]
  [[ "$output" == *"skills/a/SKILL.md"* ]]
}

@test "テストと CI だけの変更なら版が同じでも通す" {
  printf 'changed\n' >>"$REPO/tests/a.bats"
  printf 'changed\n' >>"$REPO/.github/workflows/ci.yaml"
  commit "change"

  check
  [ "$status" -eq 0 ]
}

@test "版が base より古ければ落とす" {
  printf 'changed\n' >>"$REPO/skills/a/SKILL.md"
  set_version "2026.0924.01"
  commit "change"

  check
  [ "$status" -eq 1 ]
}

@test "版が calver の形でなければ、配布物の変更が無くても落とす" {
  printf 'changed\n' >>"$REPO/tests/a.bats"
  set_version "2026.926.1"
  commit "change"

  check
  [ "$status" -eq 1 ]
  [[ "$output" == *"YYYY.MMDD.NN"* ]]
}

@test "base の ref を渡さなければ usage を出して exit 1 する" {
  run bash -c 'cd "$1" && ./scripts/check-version-bump.sh' _ "$REPO"
  [ "$status" -eq 1 ]
  [[ "$output" == *"usage"* ]]
}
