#!/bin/bash
set -euo pipefail
# check-version-bump.sh
# PR で配布物を変えたのに plugin.json の version が上がっていなければ落とす。
#
# Claude Code は manifest の version の文字列が変わったときだけ update を配る。
# 上げ忘れた変更は main に入っても利用者へ届かないので、merge の前に止める。
#
#   check-version-bump.sh BASE_REF
#
# リポジトリの root で実行する。BASE_REF...HEAD の差分を見る。
# テスト・CI・開発用の設定だけの変更は配布物に数えない。

MANIFEST=".claude-plugin/plugin.json"
CALVER_RE='^[0-9]{4}\.[0-9]{4}\.[0-9]{2}$'
# 利用者の手元で動かないファイル。ここに当たる変更だけなら版を上げなくてよい
DEV_ONLY_RE='^(\.github/|tests/|Taskfile\.yml$|aqua\.yaml$|\.golangci\.yml$|\.pinact\.yaml$|\.envrc$|\.gitignore$|scripts/(bump-version|check-version-bump|validate-json-schema)\.sh$)|_test\.go$'

die() {
  printf 'check-version-bump: %s\n' "$1" >&2
  exit 1
}

if [ "$#" -ne 1 ]; then
  printf 'usage: check-version-bump.sh BASE_REF\n' >&2
  exit 1
fi
BASE="$1"

if ! head_manifest=$(git show "HEAD:$MANIFEST"); then
  die "HEAD の $MANIFEST を読めません"
fi
if ! head_version=$(printf '%s' "$head_manifest" | jq -er '.version'); then
  die "HEAD の $MANIFEST に version がありません"
fi
if ! [[ "$head_version" =~ $CALVER_RE ]]; then
  die "version は YYYY.MMDD.NN の形にしてください (今は $head_version)｡task bump で直せます"
fi

if ! base_manifest=$(git show "$BASE:$MANIFEST"); then
  die "$BASE の $MANIFEST を読めません"
fi
if ! base_version=$(printf '%s' "$base_manifest" | jq -er '.version'); then
  die "$BASE の $MANIFEST に version がありません"
fi

if ! changed=$(git diff --name-only "$BASE...HEAD"); then
  die "$BASE との差分を取れません"
fi
distributed=()
while IFS= read -r file; do
  [ -n "$file" ] || continue
  [[ "$file" =~ $DEV_ONLY_RE ]] && continue
  distributed+=("$file")
done < <(printf '%s\n' "$changed")

if [ "${#distributed[@]}" -eq 0 ]; then
  printf 'check-version-bump: 配布物の変更はありません (version %s)\n' "$head_version"
  exit 0
fi

if [ "$head_version" = "$base_version" ]; then
  printf 'check-version-bump: 配布物を変えていますが version が %s のままです｡task bump で上げてください｡変えたファイル:\n' "$head_version" >&2
  printf '  %s\n' "${distributed[@]}" >&2
  exit 1
fi
# calver は桁が固定なので文字列の大小が日付順になる
if ! [[ "$head_version" > "$base_version" ]]; then
  die "version が $base_version から $head_version へ戻っています"
fi
printf 'check-version-bump: ok (%s -> %s)\n' "$base_version" "$head_version"
