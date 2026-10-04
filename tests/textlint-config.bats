#!/usr/bin/env bats
# 同梱の textlint config のテスト
# 句読点を全角へ揃える autofix と、その後の lint が文を句点で区切って数えることを見る。
# 半角の句点は textlint のルールが文の区切りとして扱わず、段落全体を 1 文に数える (#37)。
# 2 つの config の組み合わせは mock では確かめられないため、実物の textlint を起動する。
# npx がルールパッケージを取得するのでネットワークが要る。起動は setup_file の 1 回にまとめる。
bats_require_minimum_version 1.5.0

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"

setup_file() {
  local dir="$BATS_FILE_TMPDIR"
  export FIXTURES="$dir"
  export RESULT="$dir/result.json"

  # どちらの文も 120 字に収まるが、段落全体では 120 字を超える。句読点は半角で書いてある
  printf '# doc\n\n%s%s\n' \
    '設定ファイルは起動時に一度だけ読み込み､以後はメモリ上の値を参照する構成にしてあるため､変更を反映するにはプロセスを再起動する｡' \
    '再読み込みの最中に届いた要求は古い値のまま処理されるため､反映の完了は応答の本文に含まれる世代番号を見て確かめる必要がある｡' \
    >"$dir/two-sentences.md"

  # 1 文で 120 字を超える
  printf '# doc\n\n%s%s\n' \
    '設定ファイルは起動時に一度だけ読み込み､以後はメモリ上の値を参照する構成にしてあるため､変更を反映するにはプロセスを再起動するか､' \
    '管理用のエンドポイントへ再読み込みの要求を送ったうえで､反映の完了を応答の本文に含まれる世代番号と照らし合わせて確かめる必要がある｡' \
    >"$dir/one-sentence.md"

  # 文をまたいで同じ助詞が並ぶ。文ごとに見れば重複は無い
  printf '# doc\n\n朝に起きる｡夜に寝る｡\n' >"$dir/joshi-across.md"

  "$REPO_ROOT/bin/textlint-run.sh" \
    --config "$REPO_ROOT/skills/textlint-check/configs/base.textlintrc.json" \
    --output "$RESULT" \
    "$dir"/*.md
}

# count_issues FILE RULE_ID -> FILE に残った RULE_ID の指摘の件数
count_issues() {
  jq --arg file "$1" --arg rule "$2" \
    '[.remaining_issues[] | select((.filePath | endswith("/" + $file)) and .ruleId == $rule)] | length' \
    "$RESULT"
}

@test "autofix は半角の句読点を全角へ揃える" {
  # 文字クラスにまとめると、C ロケールでは漢字のバイト列にも一致する
  run grep -cF '｡' "$FIXTURES/two-sentences.md"
  [ "$output" = "0" ]
  run grep -cF '､' "$FIXTURES/two-sentences.md"
  [ "$output" = "0" ]
  run grep -cF '。' "$FIXTURES/two-sentences.md"
  [ "$output" = "1" ]
  run grep -cF '、' "$FIXTURES/two-sentences.md"
  [ "$output" = "1" ]
}

@test "sentence-length は 2 文の段落を 1 文として数えない" {
  run count_issues two-sentences.md ja-technical-writing/sentence-length
  [ "$status" -eq 0 ]
  [ "$output" = "0" ]
}

@test "no-doubled-joshi は句点をまたいだ助詞を重複に数えない" {
  run count_issues joshi-across.md ja-technical-writing/no-doubled-joshi
  [ "$status" -eq 0 ]
  [ "$output" = "0" ]
}

@test "sentence-length は 120 字を超える 1 文を検出する" {
  run count_issues one-sentence.md ja-technical-writing/sentence-length
  [ "$status" -eq 0 ]
  [ "$output" = "1" ]
}
