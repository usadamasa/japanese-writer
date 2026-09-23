#!/usr/bin/env bats
# validate-json-schema.sh のテスト
# スクリプト自身の責務 (検証対象の選別､schema URL の取り出し､失敗の集約と終了コード) だけを見る｡
# schema による検証そのものは jv (santhosh-tekuri/jsonschema) の責務なので､引数を記録するモックに差し替える｡
# モックの本文とフィクスチャの "$schema" は展開させない文字列なので､単引用符で書く｡
# shellcheck disable=SC2016
bats_require_minimum_version 1.5.0

SCRIPT_PATH="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/scripts/validate-json-schema.sh"

setup() {
  WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/validate-json-schema-test.XXXXXX")
  export WORKDIR
  mkdir -p "$WORKDIR/bin" "$WORKDIR/repo/conf"
  CHECK_LOG="$WORKDIR/check.log"
  export CHECK_LOG
  : >"$CHECK_LOG"

  # 呼ばれた引数を 1 行ずつ記録し､MOCK_CHECK_EXIT の値で終了する
  {
    printf '#!/bin/bash\n'
    printf '%s\n' 'printf "%s\n" "$*" >>"$CHECK_LOG"'
    printf '%s\n' 'exit "${MOCK_CHECK_EXIT:-0}"'
  } >"$WORKDIR/bin/jv"
  chmod +x "$WORKDIR/bin/jv"
  PATH="$WORKDIR/bin:$PATH"
  export PATH

  git init -q "$WORKDIR/repo"
  cd "$WORKDIR/repo" || return 1
}

teardown() {
  [ -n "$WORKDIR" ] && rm -rf "$WORKDIR"
}

# 対象になるファイルと､ならないファイルを一通り並べる
setup_mixed_tree() {
  printf '{"$schema":"https://example.com/a.json","name":"a"}\n' >conf/a.json
  printf '{"name":"b"}\n' >conf/b.json
  printf '["not","an","object"]\n' >conf/list.json
  printf '{"$schema":"https://example.com/untracked.json"}\n' >conf/untracked.json
  printf '# yaml-language-server: $schema=https://example.com/d.json\nkey: value\n' >conf/d.yaml
  printf 'key: value\n' >conf/e.yml
  printf '# yaml-language-server: $schema=https://example.com/untracked-y.json\n' >conf/untracked.yml
  git add conf/a.json conf/b.json conf/list.json conf/d.yaml conf/e.yml
}

@test "schema を宣言した tracked な JSON と YAML だけを検証に回す" {
  setup_mixed_tree

  run "$SCRIPT_PATH"
  [ "$status" -eq 0 ]

  grep -qx -- "https://example.com/a.json conf/a.json" "$CHECK_LOG"
  grep -qx -- "https://example.com/d.json conf/d.yaml" "$CHECK_LOG"
  # schema を宣言しないファイルと､git 管理外のファイルは渡らない
  [ "$(wc -l <"$CHECK_LOG")" -eq 2 ]
}

@test "tracked な JSON が壊れていれば非 0 で終了し､ファイル名を stderr に出す" {
  setup_mixed_tree
  printf '{"broken":\n' >conf/broken.json
  git add conf/broken.json
  # git 管理外の壊れた JSON は見ない
  printf '{"ignored":\n' >conf/ignored.json

  run --separate-stderr "$SCRIPT_PATH"
  [ "$status" -ne 0 ]
  [[ "$stderr" == *"conf/broken.json"* ]]
  [[ "$stderr" != *"conf/ignored.json"* ]]
}

@test "jv が失敗したら非 0 で終了し､残りのファイルも検証する" {
  setup_mixed_tree
  export MOCK_CHECK_EXIT=1

  run --separate-stderr "$SCRIPT_PATH"
  [ "$status" -ne 0 ]
  [[ "$stderr" == *"conf/a.json"* ]]
  [[ "$stderr" == *"conf/d.yaml"* ]]
  # 1 件目の失敗で打ち切らない
  [ "$(wc -l <"$CHECK_LOG")" -eq 2 ]
}
