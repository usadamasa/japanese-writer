#!/bin/bash
set -euo pipefail
# prepare-data.sh
# plugin の data ディレクトリ (${CLAUDE_PLUGIN_DATA}) に、hook と crit が使うものを用意する。
#
#   1. crit の approve 後 prompt を data の crit/ へ複製する (~/.crit のリンク先を版に依らず固定する)
#   2. writing-gate を一時ディレクトリへビルドし、検証してから data の bin/ へ置く
#
# plugin root (${CLAUDE_PLUGIN_ROOT}) は版ごとに別のディレクトリになり、update のたびに
# 中身が入れ替わる。data は update をまたいで残るので、ここへ置けば張り直しが要らない。
#
# --if-stale を付けると、ビルドの入力 (Go のソースと埋め込みルール) の cksum が前回と同じで
# バイナリが残っていればビルドを省く。SessionStart hook はこちらで呼ぶ。
# 入力の中身で判定するので、版の文字列を上げ忘れたローカルの変更でもビルドし直す。
#
# 検証は exit 0 だけでは足りない。ルールは go:embed でバイナリへ焼き込まれるので、古い
# バイナリでも scan は成功する。既知の process-leak パターンが error として出ることまで見る。
# 検証に通るまで data の bin/ には触れないので、失敗しても前回のバイナリが残る。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# rules.json の process-leak にある逐語パターン。検証用の probe に埋める
LEAK_PROBE='ご要望に従い、この節を書き直しました｡'

usage() {
  printf 'usage: prepare-data.sh [--if-stale]\n' >&2
  printf '  --if-stale  ビルドの入力が前回と同じで、バイナリが残っていればビルドしない\n' >&2
}

say() {
  printf 'prepare-data: %s\n' "$1"
}

die() {
  printf 'prepare-data: %s\n' "$1" >&2
  exit 1
}

IF_STALE=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --if-stale)
      IF_STALE=true
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
[ -n "$DATA" ] || die "CLAUDE_PLUGIN_DATA が設定されていません (plugin の data ディレクトリを渡してください)"
case "$DATA" in
  /*) ;;
  *) die "CLAUDE_PLUGIN_DATA は絶対パスで渡してください: $DATA" ;;
esac

BIN_DIR="$DATA/bin"
GATE="$BIN_DIR/writing-gate"
STAMP="$BIN_DIR/.inputs.cksum"

# ---------------------------------------------------------------------------
# 1. crit の prompt
# ---------------------------------------------------------------------------
CRIT_SRC="$ROOT/crit/on_finish_approved.md"
CRIT_DST="$DATA/crit/on_finish_approved.md"
if ! cmp -s "$CRIT_SRC" "$CRIT_DST"; then
  mkdir -p "$DATA/crit" || die "ディレクトリを作成できません: $DATA/crit"
  # リンク越しに読まれている最中でも欠けないよう、隣へ書いてから置き換える
  cp "$CRIT_SRC" "$CRIT_DST.tmp" || die "crit の prompt を複製できません: $CRIT_DST"
  mv -f "$CRIT_DST.tmp" "$CRIT_DST" || die "crit の prompt を置き換えられません: $CRIT_DST"
  say "crit の prompt を更新しました: $CRIT_DST"
fi

# ---------------------------------------------------------------------------
# 2. writing-gate
# ---------------------------------------------------------------------------
# input_cksum -> ビルドの入力の cksum を 1 行で返す。パスは root からの相対にして、
# 版ごとに置き場が変わっても中身が同じなら同じ値にする
input_cksum() {
  (
    cd "$ROOT"
    find . \( -path ./tmp -o -path ./bin -o -path ./.git \) -prune -o -type f \
      \( -name '*.go' -o -name go.mod -o -name go.sum -o -name rules.json \) -print0 |
      LC_ALL=C sort -z |
      xargs -0 cksum
  ) | cksum
}

if ! inputs=$(input_cksum); then
  die "ビルドの入力の cksum を計算できません: $ROOT"
fi

if [ "$IF_STALE" = true ] && [ -x "$GATE" ] && [ -f "$STAMP" ] && [ "$(cat "$STAMP")" = "$inputs" ]; then
  exit 0
fi

command -v go >/dev/null || die "go が見つかりません (writing-gate のビルドに使います)"
command -v jq >/dev/null || die "jq が見つかりません (writing-gate の検証に使います)"

mkdir -p "$BIN_DIR" "$DATA/tmp" || die "ディレクトリを作成できません: $DATA"
WORK_DIR=$(mktemp -d "$DATA/tmp/build-XXXXXX") || die "一時ディレクトリを作成できません: $DATA/tmp"
trap 'rm -rf "$WORK_DIR"' EXIT

say "writing-gate をビルドします: $GATE"
if ! go build -C "$ROOT" -o "$WORK_DIR/" ./...; then
  die "go build に失敗しました｡上の出力を確認してください (go.mod の要求より古い Go だと、toolchain の取得に network が要ります)"
fi
[ -x "$WORK_DIR/writing-gate" ] || die "ビルドは成功しましたが writing-gate がありません"

PROBE="$WORK_DIR/probe.md"
printf '# probe\n\n%s\n' "$LEAK_PROBE" >"$PROBE"
if ! scan_out=$("$WORK_DIR/writing-gate" scan --format json "$PROBE" 2>&1); then
  printf '%s\n' "$scan_out" >&2
  die "ビルドした writing-gate の実行に失敗しました"
fi
if ! parse_err=$(printf '%s' "$scan_out" | jq empty 2>&1); then
  printf '%s\n' "$scan_out" >&2
  die "writing-gate scan の出力が JSON ではありません: $parse_err"
fi
errors=$(printf '%s' "$scan_out" | jq '.errors // 0')
if [ "$errors" -lt 1 ]; then
  die "ビルドした writing-gate が既知の process-leak パターンを拾いません (errors=$errors)｡埋め込みルールが壊れています"
fi

# 置き換えの途中で Stop hook が古い stamp と新しいバイナリを組み合わせて見ても害はない
# (stamp は再ビルドの判定にだけ使う)。先にバイナリ、後に stamp の順で置く
mv -f "$WORK_DIR/writing-gate" "$GATE" || die "writing-gate を置き換えられません: $GATE"
printf '%s\n' "$inputs" >"$STAMP" || die "stamp を書けません: $STAMP"
say "検証 ok: process-leak を検出しました (errors=$errors): $GATE"
