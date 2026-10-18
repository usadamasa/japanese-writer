#!/bin/bash
set -euo pipefail
# setup.sh
# plugin の準備を手で確かめ直す。通常は SessionStart hook (prepare-data.sh --if-stale) が
# 自動で済ませるので、これを使うのは自動の準備が失敗したときと、crit のリンクを張るとき。
#
#   1. 前提コマンド (go / npx / jq) の確認
#   2. prepare-data.sh で writing-gate を data ディレクトリへビルドし直し、検証する
#   3. --crit を付けたときだけ、crit の approve 後 prompt を data の複製へリンクする
#
# --check を付けると 2 と 3 を書き込み無しの点検 (prepare-data.sh --check と、--crit 時はリンクの確認) に
# 置き換える。hook の準備が済んだ端末なら、sandbox の中からこれだけで完了と判定できる。
#
# data ディレクトリは CLAUDE_PLUGIN_DATA で受け取る。hook には Claude Code が渡すが、
# Bash tool には渡らないので、SKILL.md が置換した値を前置きして呼ぶ。
#
# Claude Code の sandbox は ~/.claude/plugins への書き込みを拒む。go を起動する前に
# data の bin/ へ書けるかを確かめ、書けなければ sandbox の外で打つコマンド (`!` 前置) を案内する。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_PATH="$SCRIPT_DIR/$(basename "${BASH_SOURCE[0]}")"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CRIT_PROMPT_DST="$HOME/.crit/prompts/on_finish_approved.md"

usage() {
  printf 'usage: CLAUDE_PLUGIN_DATA=<data dir> setup.sh [--check] [--crit]\n' >&2
  printf '  --check  何も書かずに、準備が済んでいるかだけを点検する\n' >&2
  printf '  --crit   crit の approve 後 prompt (~/.crit/prompts/on_finish_approved.md) を plugin の data の複製へリンクする\n' >&2
  printf '           --check と併せると、今の版と同じ内容のファイルへリンクが済んでいるかを点検する\n' >&2
}

say() {
  printf 'setup: %s\n' "$1"
}

die() {
  printf 'setup: %s\n' "$1" >&2
  exit 1
}

LINK_CRIT=false
CHECK=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --crit)
      LINK_CRIT=true
      shift
      ;;
    --check)
      CHECK=true
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

DATA="${CLAUDE_PLUGIN_DATA:-}"
[ -n "$DATA" ] || die "CLAUDE_PLUGIN_DATA が設定されていません (/japanese-writer:setup から呼ぶか、plugin の data ディレクトリを前置きしてください)"
case "$DATA" in
  /*) ;;
  *) die "CLAUDE_PLUGIN_DATA は絶対パスで渡してください: $DATA" ;;
esac
BIN_DIR="$DATA/bin"
CRIT_PROMPT_SRC="$DATA/crit/on_finish_approved.md"

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
# --check: 2 と 3 を書き込み無しの点検に置き換える
# ---------------------------------------------------------------------------
# crit_linked -> ~/.crit の prompt が data の複製を指していれば 0 を返す
crit_linked() {
  [ -L "$CRIT_PROMPT_DST" ] && [ "$(readlink "$CRIT_PROMPT_DST")" = "$CRIT_PROMPT_SRC" ]
}

# crit_prompt_current -> ~/.crit の prompt が、指し先を問わず今の版と同じ内容のファイルへのリンクなら 0 を返す。
# 利用者が plugin の clone へ張ったリンクを、--crit で data の複製へ張り替えさせないための判定
crit_prompt_current() {
  [ -L "$CRIT_PROMPT_DST" ] && cmp -s "$CRIT_PROMPT_DST" "$ROOT/crit/on_finish_approved.md"
}

# suggest_crit_link -> --crit 無しのとき、crit があればリンクを提案する
suggest_crit_link() {
  command -v crit >/dev/null || return 0
  if crit_prompt_current; then
    say "crit: リンク済み ($CRIT_PROMPT_DST -> $(readlink "$CRIT_PROMPT_DST"))"
    return 0
  fi
  say "crit: 見つかりました｡approve 後の prompt を plugin のものにするなら --crit を付けて実行してください"
}

if [ "$CHECK" = true ]; then
  ok=true
  if ! "$SCRIPT_DIR/prepare-data.sh" --check; then
    ok=false
  fi
  if [ "$LINK_CRIT" = true ]; then
    if crit_prompt_current; then
      say "crit: リンク済み ($CRIT_PROMPT_DST -> $(readlink "$CRIT_PROMPT_DST"))"
    else
      say "crit: $CRIT_PROMPT_DST は今の版と同じ内容のファイルを指していません"
      ok=false
    fi
  else
    suggest_crit_link
  fi
  if [ "$ok" != true ]; then
    die "点検で欠けが見つかりました｡--check を外して実行すると準備し直します"
  fi
  say "点検 ok: 書き込みの要る作業はありません"
  exit 0
fi

# ---------------------------------------------------------------------------
# 2. ビルドと検証
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
  printf '\n  ! CLAUDE_PLUGIN_DATA="%s" "%s"%s\n\n' "$DATA" "$SCRIPT_PATH" "$rerun_args"
  say "それでも同じなら、$1 の所有者と権限を確認してください"
  exit 1
}

# sandbox に拒まれたときは、go を走らせる前にここで止まる
if ! write_err=$(mkdir -p "$BIN_DIR" 2>&1 && touch "$BIN_DIR/.write-probe" 2>&1); then
  die_unwritable "$BIN_DIR" "$write_err"
fi
rm -f "$BIN_DIR/.write-probe"

# 手で呼ぶのは自動の準備を疑うときなので、入力が同じでもビルドし直す
if ! "$SCRIPT_DIR/prepare-data.sh"; then
  die "writing-gate の準備に失敗しました｡上の出力を確認してください"
fi

# ---------------------------------------------------------------------------
# 3. crit の prompt リンク
# ---------------------------------------------------------------------------
# stale_plugin_link TARGET -> 過去の setup が張ったリンクの残骸なら 0 を返す。
# 以前の setup は版ごとの置き場 (<root の親>/<版>/crit/...) を直接指していた。そうしたリンクと
# 指し先が消えたものは data の複製へ張り替えてよい。それ以外 (利用者が自分で張ったもの) は触らない。
stale_plugin_link() {
  local target="$1"
  [ ! -e "$target" ] && return 0
  case "$target" in
    "$(dirname "$ROOT")"/*/crit/on_finish_approved.md) return 0 ;;
  esac
  return 1
}

# link_crit_prompt -> --crit の処理。状態ごとに報告して抜ける
link_crit_prompt() {
  local current link_err

  if crit_linked; then
    say "crit: リンク済み ($CRIT_PROMPT_DST -> $CRIT_PROMPT_SRC)"
    return 0
  fi

  if [ ! -L "$CRIT_PROMPT_DST" ] && [ ! -e "$CRIT_PROMPT_DST" ]; then
    if ! link_err=$(mkdir -p "$(dirname "$CRIT_PROMPT_DST")" 2>&1 && ln -s "$CRIT_PROMPT_SRC" "$CRIT_PROMPT_DST" 2>&1); then
      die_unwritable "$CRIT_PROMPT_DST" "$link_err"
    fi
    say "crit: リンクしました ($CRIT_PROMPT_DST -> $CRIT_PROMPT_SRC)"
    return 0
  fi

  if [ -L "$CRIT_PROMPT_DST" ] && stale_plugin_link "$(readlink "$CRIT_PROMPT_DST")"; then
    current=$(readlink "$CRIT_PROMPT_DST")
    if ! link_err=$(ln -sfn "$CRIT_PROMPT_SRC" "$CRIT_PROMPT_DST" 2>&1); then
      die_unwritable "$CRIT_PROMPT_DST" "$link_err"
    fi
    say "crit: 版ごとの置き場へのリンクを張り替えました ($current -> $CRIT_PROMPT_SRC)"
    return 0
  fi

  if crit_prompt_current; then
    say "crit: リンク済み ($CRIT_PROMPT_DST -> $(readlink "$CRIT_PROMPT_DST")、今の版と同じ内容)"
    return 0
  fi

  # 利用者が自分で張ったリンクか、symlink ではない実ファイル
  if [ -L "$CRIT_PROMPT_DST" ]; then
    current=$(readlink "$CRIT_PROMPT_DST")
  else
    current="$CRIT_PROMPT_DST (symlink ではない実ファイル)"
  fi
  say "crit: $CRIT_PROMPT_DST は既に別の場所を指しているので触りません: $current"
  say "crit: plugin のファイルへ向けるなら、そのリンクを消してから --crit をもう一度実行してください"
}

if [ "$LINK_CRIT" = true ]; then
  link_crit_prompt
else
  suggest_crit_link
fi

say "完了｡plugin を更新したら、次のセッションの開始時に SessionStart hook が準備し直します"
