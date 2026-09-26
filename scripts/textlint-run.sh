#!/bin/bash
set -euo pipefail
# textlint-run.sh
# markdown を textlint で autofix し、残った指摘とあわせて 1 つの JSON にまとめる。
#
# textlint は指摘が残ると終了コード 1 を返す。「lint に失敗した」と「指摘が見つかった」を
# 終了コードでは区別できないため、stdout が空でない JSON 配列かどうかで判定する。
#
# autofix 後にもう一度 lint するのは、--fix の remainingMessages が修正前の位置を持つため。
# 呼び出し元 (proofreader subagent) は修正後のファイルを読むので、行番号を揃える必要がある。
#
# 2 つの pass は別々の config で走る。fix pass は表記ゆれだけを直し、lint pass が
# 残り全部を検出する。語の置換で済ませると文の骨格が歪んだまま残るため、意味に
# 関わる指摘は自動で直さず、文ごと書き直させる。
# fix 用 config は --fix-config で渡す。省略時は --config と同じディレクトリの
# fix.textlintrc.json を使い、それも無ければ --config を両方の pass に使う。
#
# ルールパッケージは npx へ明示的に渡す。対象リポジトリの node_modules に依存しないため、
# textlint を導入していないリポジトリの .md も同じルールで点検できる。

TEXTLINT_PACKAGES=(
  "textlint@15"
  "@textlint-ja/textlint-rule-preset-ai-writing@1"
  "textlint-rule-preset-ja-technical-writing@12"
  "textlint-filter-rule-comments@1"
)

usage() {
  printf 'usage: textlint-run.sh --config CONFIG [--fix-config CONFIG] --output RESULT_JSON FILE...\n' >&2
  printf '  CONFIG      lint pass に使う .textlintrc.json の絶対パス\n' >&2
  printf '  --fix-config  autofix pass に使う config の絶対パス (省略時は CONFIG と同じディレクトリの fix.textlintrc.json)\n' >&2
  printf '  RESULT_JSON 結果 JSON の書き出し先 (絶対パス)\n' >&2
  printf '  FILE        点検対象 .md の絶対パス (複数可)\n' >&2
}

die() {
  printf 'textlint-run: %s\n' "$1" >&2
  exit 1
}

# require_abs LABEL PATH
require_abs() {
  case "$2" in
    /*) return 0 ;;
  esac
  die "$1 は絶対パスで指定してください: $2"
}

CONFIG=""
FIX_CONFIG=""
OUTPUT=""
TARGETS=()

while [ "$#" -gt 0 ]; do
  case "$1" in
    --config)
      [ "$#" -ge 2 ] || die "--config に値がありません"
      CONFIG="$2"
      shift 2
      ;;
    --fix-config)
      [ "$#" -ge 2 ] || die "--fix-config に値がありません"
      FIX_CONFIG="$2"
      shift 2
      ;;
    --output)
      [ "$#" -ge 2 ] || die "--output に値がありません"
      OUTPUT="$2"
      shift 2
      ;;
    --help | -h)
      usage
      exit 0
      ;;
    --)
      shift
      TARGETS+=("$@")
      break
      ;;
    -*)
      usage
      die "不明なオプション: $1"
      ;;
    *)
      TARGETS+=("$1")
      shift
      ;;
  esac
done

[ -n "$CONFIG" ] || { usage; die "--config は必須です"; }
[ -n "$OUTPUT" ] || { usage; die "--output は必須です"; }
[ "${#TARGETS[@]}" -gt 0 ] || { usage; die "点検対象を 1 つ以上指定してください"; }

command -v npx >/dev/null || die "npx が見つかりません"
command -v jq >/dev/null || die "jq が見つかりません"

require_abs "--config" "$CONFIG"
[ -f "$CONFIG" ] || die "config がありません: $CONFIG"

if [ -n "$FIX_CONFIG" ]; then
  require_abs "--fix-config" "$FIX_CONFIG"
  [ -f "$FIX_CONFIG" ] || die "fix config がありません: $FIX_CONFIG"
else
  # 同じディレクトリに fix 用があればそれを使う。無ければ従来どおり lint と同じ config で
  # autofix する (textlint を持ち込んでいないリポジトリの config_root を壊さないため)。
  FIX_CONFIG="$(dirname "$CONFIG")/fix.textlintrc.json"
  [ -f "$FIX_CONFIG" ] || FIX_CONFIG="$CONFIG"
fi

require_abs "--output" "$OUTPUT"
output_dir=$(dirname "$OUTPUT")
[ -d "$output_dir" ] || die "出力先ディレクトリがありません: $output_dir"

for target in "${TARGETS[@]}"; do
  require_abs "点検対象" "$target"
  [ -f "$target" ] || die "点検対象がありません: $target"
done

npx_args=(--yes)
for pkg in "${TEXTLINT_PACKAGES[@]}"; do
  npx_args+=(--package "$pkg")
done
npx_args+=(-- textlint --format json)

# 中間ファイルは出力先と同じディレクトリに置く。sandbox は $TMPDIR への書き込みを
# 拒否することがあり、呼び出し元が用意した出力先だけが確実に書ける。
WORK_DIR=$(mktemp -d "$output_dir/textlint-run-XXXXXX") || die "一時ディレクトリを作成できません: $output_dir"
trap 'rm -rf "$WORK_DIR"' EXIT

# run_pass LABEL STDOUT_FILE CONFIG [EXTRA_FLAG...]
#   textlint を 1 回走らせ、stdout が空でない JSON 配列かどうかで実行の成否を判定する。
#   jq empty は空文字列でも成功してしまい、書き込み権限エラーなどで stdout が
#   空になった失敗を素通りさせるため、jq empty だけでは検知できない。
run_pass() {
  local label="$1" out="$2" config="$3"
  shift 3
  local err="$WORK_DIR/$label.err" status=0 verify_status=0 verify_err

  npx "${npx_args[@]}" --config "$config" "$@" "${TARGETS[@]}" >"$out" 2>"$err" || status=$?

  verify_err=$(jq -e 'type == "array" and length > 0' "$out" 2>&1 >/dev/null) || verify_status=$?
  if [ "$verify_status" -ne 0 ]; then
    printf 'textlint-run: textlint (%s) の実行に失敗しました (exit %s)\n' "$label" "$status" >&2
    printf 'textlint-run: stdout が有効な結果ではありません: %s\n' "${verify_err:-空の出力}" >&2
    # textlint は設定エラーを stdout に書くことがあるので両方そのまま流す
    printf -- '--- textlint stdout ---\n' >&2
    cat "$out" >&2
    printf -- '--- textlint stderr ---\n' >&2
    cat "$err" >&2
    exit 1
  fi
}

run_pass fix "$WORK_DIR/fix.json" "$FIX_CONFIG" --fix
run_pass lint "$WORK_DIR/lint.json" "$CONFIG"

jq -n \
  --slurpfile fix "$WORK_DIR/fix.json" \
  --slurpfile lint "$WORK_DIR/lint.json" \
  '{
     applied_fixes: [
       $fix[0][] | .filePath as $p | (.applyingMessages // [])[]
       | {filePath: $p, ruleId, line, column, message}
     ],
     remaining_issues: [
       $lint[0][] | .filePath as $p | (.messages // [])[]
       | {filePath: $p, ruleId, severity, line, column, message}
     ]
   }' >"$OUTPUT"

applied=$(jq '.applied_fixes | length' "$OUTPUT")
remaining=$(jq '.remaining_issues | length' "$OUTPUT")
printf 'textlint-run: applied_fixes=%s remaining_issues=%s output=%s\n' \
  "$applied" "$remaining" "$OUTPUT"
