#!/usr/bin/env bats
# crit の approve prompt (crit/) のテスト
#
# prompt 本文は crit が LLM へ渡す指示であり、文面は推敲で変わる｡grep で
# 手順を固定すると推敲のたびに落ちるだけで、統合手順の正しさは検証できない｡
# ここでは crit が自動検出できる形で置かれているかだけを見る｡
bats_require_minimum_version 1.5.0

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
PROMPT="$REPO_ROOT/crit/on_finish_approved.md"

@test "crit/on_finish_approved.md が存在する" {
  [ -f "$PROMPT" ]
}

@test "crit の conventional path に合う命名になっている (mode suffix は . 区切り)" {
  # crit は ~/.crit/prompts/on_finish_approved[.mode].md しか自動検出しない｡
  # 名前が外れると hook が無言で無視される｡
  for file in "$REPO_ROOT/crit"/*; do
    [ -f "$file" ] || continue
    [[ "$(basename "$file")" =~ ^on_finish_(approved|unresolved)(\.[a-z]+)?\.md$ ]]
  done
}
