#!/bin/bash
set -euo pipefail
# bump-version.sh
# .claude-plugin/plugin.json の version を calver (YYYY.MMDD.NN) で 1 つ進める。
#
# Claude Code は manifest の version の文字列が変わったときだけ update を配る。
# 同じ日の版なら NN を上げ、別の日なら今日の 01 にする。semver など calver でない版からは今日の 01 へ移る。
#
#   bump-version.sh [--date YYYY-MM-DD]
#
# --date はテストと日付をまたぐ作業のためにある。省略すると今日のローカル日付を使う。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="$(cd "$SCRIPT_DIR/.." && pwd)/.claude-plugin/plugin.json"

die() {
  printf 'bump-version: %s\n' "$1" >&2
  exit 1
}

usage() {
  printf 'usage: bump-version.sh [--date YYYY-MM-DD]\n' >&2
}

DATE=$(date +%Y-%m-%d)
while [ "$#" -gt 0 ]; do
  case "$1" in
    --date)
      [ "$#" -ge 2 ] || { usage; die "--date に日付がありません"; }
      DATE="$2"
      shift 2
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

[[ "$DATE" =~ ^([0-9]{4})-([0-9]{2})-([0-9]{2})$ ]] || die "--date は YYYY-MM-DD で渡してください: $DATE"
TODAY="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}${BASH_REMATCH[3]}"

[ -f "$MANIFEST" ] || die "manifest がありません: $MANIFEST"
if ! current=$(jq -er '.version' "$MANIFEST"); then
  die "manifest の version を読めません: $MANIFEST"
fi

if [[ "$current" =~ ^([0-9]{4}\.[0-9]{4})\.([0-9]{2})$ ]]; then
  day="${BASH_REMATCH[1]}"
  seq="${BASH_REMATCH[2]}"
  if [[ "$day" > "$TODAY" ]]; then
    die "今の版 $current は今日 ($TODAY) より新しい日付です｡日付を確かめてください"
  fi
  if [ "$day" = "$TODAY" ]; then
    # 先頭ゼロを 8 進と読ませない
    next=$((10#$seq + 1))
    [ "$next" -le 99 ] || die "同じ日の連番が 99 を超えます: $current"
    new=$(printf '%s.%02d' "$TODAY" "$next")
  else
    new="$TODAY.01"
  fi
else
  new="$TODAY.01"
fi

tmp=$(mktemp "$MANIFEST.XXXXXX") || die "一時ファイルを作成できません: $MANIFEST.XXXXXX"
# mktemp は 0600 で作る。置き換えた manifest を他の利用者からも読めるようにする
if ! chmod 644 "$tmp"; then
  rm -f "$tmp"
  die "一時ファイルの権限を変えられません: $tmp"
fi
if ! jq --arg v "$new" '.version = $v' "$MANIFEST" >"$tmp"; then
  rm -f "$tmp"
  die "manifest を書き換えられません: $MANIFEST"
fi
if ! mv -f "$tmp" "$MANIFEST"; then
  rm -f "$tmp"
  die "manifest を置き換えられません: $MANIFEST"
fi
printf '%s -> %s\n' "$current" "$new"
