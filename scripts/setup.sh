#!/bin/bash
set -euo pipefail
# setup.sh
# plugin の install / update 後に要る作業をまとめて行う。
#
#   1. 前提コマンド (go / npx / jq) の確認
#   2. writing-gate を plugin root の bin/ へビルドする (Stop hook がここを絶対パスで起動する)
#   3. ビルドしたバイナリで既知の漏出パターンを拾えることを確かめる
#   4. --crit を付けたときだけ、crit の approve 後 prompt を plugin のファイルへリンクする
#
# plugin root は自分の親ディレクトリ。CLAUDE_PLUGIN_ROOT は SKILL.md の中でだけ置換される
# 文字列で、利用者が `!` から直接このスクリプトを叩くときには存在しない。
#
# Claude Code の sandbox は ~/.claude/plugins/cache への書き込みを拒む。go を起動する前に
# bin/ へ書けるかを確かめ、書けなければ sandbox の外で打つコマンド (`!` 前置) を案内する。
# ビルドし終えてから断られるより、最初に分かる方がよい。
#
# 検証は exit 0 だけでは足りない。ルールは go:embed でバイナリへ焼き込まれるので、古い
# バイナリが残っていても scan は成功する。既知の process-leak パターンが error として
# 出ることまで見る。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/$(basename "${BASH_SOURCE[0]}")"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$ROOT/bin"
GATE="$BIN_DIR/writing-gate"
CRIT_PROMPT_SRC="$ROOT/crit/on_finish_approved.md"
CRIT_PROMPT_DST="$HOME/.crit/prompts/on_finish_approved.md"

# rules.json の process-leak にある逐語パターン。検証用の probe に埋める
LEAK_PROBE='ご要望に従い、この節を書き直しました｡'

usage() {
  printf 'usage: setup.sh [--crit]\n' >&2
  printf '  --crit  crit の approve 後 prompt (~/.crit/prompts/on_finish_approved.md) を plugin のファイルへリンクする\n' >&2
}

say() {
  printf 'setup: %s\n' "$1"
}

die() {
  printf 'setup: %s\n' "$1" >&2
  exit 1
}

LINK_CRIT=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --crit)
      LINK_CRIT=true
      shift
      ;;
    --help | -h)
      usage
      exit 0
      ;;
    *)
      usage
      die "不明なオプション: $1"
      ;;
  esac
done

# ---------------------------------------------------------------------------
# 1. 前提コマンド
# ---------------------------------------------------------------------------
missing=()
for cmd in go npx jq; do
  if command -v "$cmd" >/dev/null; then
    say "$cmd: ok ($(command -v "$cmd"))"
  else
    say "$cmd: 見つかりません"
    missing+=("$cmd")
  fi
done
if [ "${#missing[@]}" -gt 0 ]; then
  die "前提コマンドが不足しています: ${missing[*]} (go は writing-gate のビルド、npx と jq は textlint の点検に使う)"
fi

# ---------------------------------------------------------------------------
# 2. ビルド
# ---------------------------------------------------------------------------
# die_unwritable PATH ERR -> 書き込み拒否の案内を出して exit 1 する。
# Claude Code の Bash から呼ばれたときは sandbox が原因のことが多いので、sandbox の外で
# 同じ引数で打ち直すコマンドを示す。`!` でも同じなら権限そのものの問題になる。
die_unwritable() {
  local rerun_args=""
  [ "$LINK_CRIT" = true ] && rerun_args=" --crit"
  printf '%s\n' "$2" >&2
  say "$1 へ書き込めません｡Claude Code の Bash から実行しているなら、sandbox が拒んでいる可能性があります｡"
  say "次のコマンドを Claude Code のプロンプトにそのまま入力すると、sandbox の外で実行できます:"
  printf '\n  ! "%s"%s\n\n' "$SCRIPT_PATH" "$rerun_args"
  say "それでも同じなら、$1 の所有者と権限を確認してください"
  exit 1
}

# sandbox に拒まれたときは、go を走らせる前にここで止まる
if ! write_err=$(mkdir -p "$BIN_DIR" 2>&1 && touch "$BIN_DIR/.write-probe" 2>&1); then
  die_unwritable "$BIN_DIR" "$write_err"
fi
rm -f "$BIN_DIR/.write-probe"

say "writing-gate をビルドします: $GATE"
if ! go build -C "$ROOT" -o bin/ ./...; then
  die "go build に失敗しました｡上の出力を確認してください (go.mod の要求より古い Go だと、toolchain の取得に network が要ります)"
fi
[ -x "$GATE" ] || die "ビルドは成功しましたが $GATE がありません"

# ---------------------------------------------------------------------------
# 3. 検証
# ---------------------------------------------------------------------------
# probe は plugin root の tmp/ に置く。bin/ に書けた直後なのでここも書ける
# (sandbox は $TMPDIR への書き込みを拒むことがある)
mkdir -p "$ROOT/tmp" || die "一時ディレクトリを作成できません: $ROOT/tmp"
WORK_DIR=$(mktemp -d "$ROOT/tmp/setup-XXXXXX") || die "一時ディレクトリを作成できません: $ROOT/tmp"
trap 'rm -rf "$WORK_DIR"' EXIT

PROBE="$WORK_DIR/probe.md"
printf '# probe\n\n%s\n' "$LEAK_PROBE" >"$PROBE"

if ! scan_out=$("$GATE" scan --format json "$PROBE" 2>&1); then
  printf '%s\n' "$scan_out" >&2
  die "ビルドした writing-gate の実行に失敗しました"
fi
if ! parse_err=$(printf '%s' "$scan_out" | jq empty 2>&1); then
  printf '%s\n' "$scan_out" >&2
  die "writing-gate scan の出力が JSON ではありません: $parse_err"
fi
errors=$(printf '%s' "$scan_out" | jq '.errors // 0')
if [ "$errors" -lt 1 ]; then
  die "ビルドした writing-gate が既知の process-leak パターンを拾いません (errors=$errors)｡古いバイナリか、埋め込みルールが壊れています"
fi
say "検証 ok: process-leak を検出しました (errors=$errors)"

# ---------------------------------------------------------------------------
# 4. crit の prompt リンク
# ---------------------------------------------------------------------------
# stale_plugin_link TARGET -> 過去の setup が張ったリンクの残骸なら 0 を返す。
# plugin を更新すると版ごとのディレクトリが変わるので、隣の版 (<root の親>/<版>/crit/...) を
# 指すものと、指し先が消えたものは張り替えてよい。それ以外 (利用者が自分で張ったもの) は触らない。
stale_plugin_link() {
  local target="$1"
  [ ! -e "$target" ] && return 0
  case "$target" in
    "$(dirname "$ROOT")"/*/crit/on_finish_approved.md) return 0 ;;
  esac
  return 1
}

if [ "$LINK_CRIT" = true ]; then
  if [ -L "$CRIT_PROMPT_DST" ] || [ -e "$CRIT_PROMPT_DST" ]; then
    # symlink ならその先、実ファイルならそのパスを「今の指し先」として扱う
    if [ -L "$CRIT_PROMPT_DST" ]; then
      current=$(readlink "$CRIT_PROMPT_DST")
    else
      current="$CRIT_PROMPT_DST (symlink ではない実ファイル)"
    fi
    if [ "$current" = "$CRIT_PROMPT_SRC" ]; then
      say "crit: リンク済み ($CRIT_PROMPT_DST -> $CRIT_PROMPT_SRC)"
    elif [ -L "$CRIT_PROMPT_DST" ] && stale_plugin_link "$current"; then
      if ! link_err=$(ln -sfn "$CRIT_PROMPT_SRC" "$CRIT_PROMPT_DST" 2>&1); then
        die_unwritable "$CRIT_PROMPT_DST" "$link_err"
      fi
      say "crit: 古い版へのリンクを張り替えました ($current -> $CRIT_PROMPT_SRC)"
    else
      say "crit: $CRIT_PROMPT_DST は既に別の場所を指しているので触りません: $current"
      say "crit: plugin のファイルへ向けるなら、そのリンクを消してから --crit をもう一度実行してください"
    fi
  else
    if ! link_err=$(mkdir -p "$(dirname "$CRIT_PROMPT_DST")" 2>&1 && ln -s "$CRIT_PROMPT_SRC" "$CRIT_PROMPT_DST" 2>&1); then
      die_unwritable "$CRIT_PROMPT_DST" "$link_err"
    fi
    say "crit: リンクしました ($CRIT_PROMPT_DST -> $CRIT_PROMPT_SRC)"
  fi
elif command -v crit >/dev/null; then
  if [ -L "$CRIT_PROMPT_DST" ] && [ "$(readlink "$CRIT_PROMPT_DST")" = "$CRIT_PROMPT_SRC" ]; then
    say "crit: リンク済み ($CRIT_PROMPT_DST)"
  else
    say "crit: 見つかりました｡approve 後の prompt を plugin のものにするなら --crit を付けて実行してください"
  fi
fi

say "完了｡plugin を更新したら (置き場のディレクトリが変わるので) もう一度実行してください"
