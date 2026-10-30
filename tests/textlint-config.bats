#!/usr/bin/env bats
# 同梱の textlint config のテスト
# 句読点を全角へ揃える autofix と、その後の lint が文を句点で区切って数えることを見る。
# 半角の句点は textlint のルールが文の区切りとして扱わず、段落全体を 1 文に数える (#37)。
# 2 つの config の組み合わせは mock では確かめられないため、実物の textlint を起動する。
# npx がルールパッケージを取得するのでネットワークが要る。起動は setup_file の 1 回にまとめる。
bats_require_minimum_version 1.5.0

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"

# prh-prose の語彙ルール: 指摘文に含まれる語|検出する文|検出しない文
# 指摘文は「<一致した文字列> => <expected>\n<prh>」の形で、ルールは prh の先頭の語で見分ける。
# 検出しない文は、比喩でない用法か、prh が示す書き換え後の文を置く。
PROSE_CASES=(
  '「壊れる」|この変更で前提が壊れる。|ディスクが壊れると読み込めない。'
  '「崩す」|互換性を崩す変更は避ける。|体調を崩す。'
  '「破綻する」|この設計は破綻している。|企業の経営破綻が報じられた。'
  '「黙って捨てる」|エラーを黙って捨てる。|一意に決まる修正は黙って直す。'
  '「値を動かす」|閾値を動かして試す。|サーバーを動かして試す。'
  '「から引く」|設定ファイルから引く。|合計から 3 を引く。'
  '「原初」|原初の設計に戻る。|最初の設計に戻る。'
  '「無差別に」|無差別に削除する。|すべての一時ファイルを削除する。'
  '「正典」|この文書を正典とする。|この文書を定義元とする。'
  '「〜に配線する」|hook に配線する。|hook との配線を説明する。'
  '「太る」|コンテキストが太る。|本文が太字になる。'
  '「見張る」|ログを見張る。|進歩に目を見張る。'
  '「原料」|メモを記事の原料にする。|メモを記事の元データにする。'
  '「素通りする」|不正な値が検査を素通りする。|不正な値を検査が検出しない。'
  '「〜という構図」|A が B に依存するという構図だ。|その要素が構図に効く。'
  '「定番」|定番の構成にする。|広く使われている構成にする。'
  '「定番」|定石どおりに書く。|手順どおりに書く。'
  '「走る」|CI でテストが走る。|ロボットが自走する。'
)

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

  local i rest
  for i in "${!PROSE_CASES[@]}"; do
    rest="${PROSE_CASES[$i]#*|}"
    printf '# doc\n\n%s\n' "${rest%%|*}" >"$dir/prose-pos-$i.md"
    printf '# doc\n\n%s\n' "${rest#*|}" >"$dir/prose-neg-$i.md"
  done

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

# count_prh FILE KEY -> FILE に残った prh の指摘のうち、prh の文面に KEY を含むものの件数
count_prh() {
  jq --arg file "$1" --arg key "$2" \
    '[.remaining_issues[] | select((.filePath | endswith("/" + $file)) and .ruleId == "prh"
      and (.message | split("\n")[1:] | join("\n") | contains($key)))] | length' \
    "$RESULT"
}

# check_prose_cases KIND WANT -> KIND (pos / neg) の fixture で、件数が WANT でないケースを列挙して失敗する
check_prose_cases() {
  local kind="$1" want="$2" i key got failed=0
  for i in "${!PROSE_CASES[@]}"; do
    key="${PROSE_CASES[$i]%%|*}"
    got=$(count_prh "prose-$kind-$i.md" "$key")
    if [ "$got" != "$want" ]; then
      printf '%s: %s の件数が %s (期待は %s)\n' "$key" "$kind" "$got" "$want"
      failed=1
    fi
  done
  return "$failed"
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

@test "prh-prose の語彙ルールは比喩の用法を 1 件として検出する" {
  check_prose_cases pos 1
}

@test "prh-prose の語彙ルールは比喩でない用法と書き換え後の文を検出しない" {
  check_prose_cases neg 0
}
