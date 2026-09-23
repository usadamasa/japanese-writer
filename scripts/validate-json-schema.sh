#!/bin/bash
set -euo pipefail
# validate-json-schema.sh
# git 管理下の JSON と YAML を､各ファイルが自分で宣言した schema で検証する｡
#
# 対象はカレントディレクトリの git リポジトリで tracked なファイルだけ｡
# - JSON: トップレベルの "$schema" キーが指す schema で検証する
# - YAML: "# yaml-language-server: $schema=<URL>" コメントが指す schema で検証する
# 検証には jv (santhosh-tekuri/jsonschema) を使う｡jv は instance 側の宣言を読まないので､
# ここで URL を取り出して schema 引数へ渡す｡jv は YAML の instance と https の schema URL をそのまま受ける｡
# schema を宣言しない JSON も､tracked なものはすべて jq で構文だけ確かめる｡
#
# 失敗しても次のファイルへ進み､最後に失敗したファイルを stderr へまとめて出して非 0 で終了する｡

for cmd in git jq jv; do
  if ! command -v "$cmd" >/dev/null; then
    printf 'validate-json-schema: %s が見つからない\n' "$cmd" >&2
    exit 1
  fi
done

failed=()
checked=0

# check_schema URL FILE -> schema で検証し､失敗したら failed に積む
check_schema() {
  checked=$((checked + 1))
  if ! jv "$1" "$2"; then
    failed+=("$2 (schema: $1)")
  fi
}

while IFS= read -r -d '' file; do
  case "$file" in
    *.json)
      if ! jq empty "$file"; then
        failed+=("$file (JSON として読めない)")
        continue
      fi
      url=$(jq -r 'if type == "object" then .["$schema"] // empty else empty end' "$file")
      if [ -n "$url" ]; then
        check_schema "$url" "$file"
      fi
      ;;
    *.yaml | *.yml)
      # grep の終了コード 1 は「宣言なし」､2 以上は読み取りの失敗
      # shellcheck disable=SC2016
      line=$(grep -m 1 -E '^# yaml-language-server: \$schema=' "$file" || [ $? -eq 1 ])
      if [ -n "$line" ]; then
        check_schema "${line#*\$schema=}" "$file"
      fi
      ;;
  esac
done < <(git ls-files -z)

if [ "${#failed[@]}" -gt 0 ]; then
  printf 'validate-json-schema: 検証に失敗したファイル:\n' >&2
  printf '  %s\n' "${failed[@]}" >&2
  exit 1
fi

printf 'validate-json-schema: schema 検証 %d 件､すべて通過\n' "$checked"
