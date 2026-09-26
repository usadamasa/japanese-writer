# plugin-tree-helpers.bash
# scripts/ を隔離ツリーへ複製して実行するテストの共通ヘルパー
#
# スクリプトは自分の親ディレクトリを plugin root として扱う。実リポジトリや本物の
# ~/.claude/plugins/data を巻き込まないよう、root と data と HOME を WORKDIR 配下へ置く。

# setup_plugin_tree -> WORKDIR / ROOT / DATA / HOME / MOCK_PATH を用意して export する。
# go の入力として go.mod と *.go と rules.json を複製する (stamp の計算対象)
setup_plugin_tree() {
  WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/plugin-test.XXXXXX")
  # TMPDIR が / で終わると // を含むパスになる。スクリプトは cd && pwd で正規化した
  # パスを出すので、比べる側も揃えておく
  WORKDIR=$(cd "$WORKDIR" && pwd)
  export WORKDIR
  SRC="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  ROOT="$WORKDIR/plugin"
  mkdir -p "$ROOT/writing-gate/internal/rules"
  cp -R "$SRC/scripts" "$ROOT/scripts"
  cp -R "$SRC/crit" "$ROOT/crit"
  cp -R "$SRC/hooks" "$ROOT/hooks"
  cp "$SRC/go.mod" "$ROOT/go.mod"
  cp "$SRC/writing-gate/main.go" "$ROOT/writing-gate/main.go"
  cp "$SRC/writing-gate/internal/rules/rules.json" "$ROOT/writing-gate/internal/rules/rules.json"

  CLAUDE_PLUGIN_DATA="$WORKDIR/data"
  export CLAUDE_PLUGIN_DATA
  DATA="$CLAUDE_PLUGIN_DATA"
  export DATA

  HOME="$WORKDIR/home"
  mkdir -p "$HOME"
  export HOME

  MOCK_PATH="$WORKDIR/mockbin"
  mkdir -p "$MOCK_PATH"
  # jq は本物を使う (結果 JSON の読み取りに要る)。ディレクトリごと PATH に足すと
  # 同居する npx や crit まで見えてしまうので、jq だけを持ち込む。aqua の shim は
  # 元の PATH が無いと本体を探せないため、symlink ではなく元の PATH で起動する wrapper にする
  printf '#!/bin/bash\nPATH=%q exec jq "$@"\n' "$PATH" >"$MOCK_PATH/jq"
  chmod +x "$MOCK_PATH/jq"
  PATH="$MOCK_PATH:/usr/bin:/bin"
  export PATH
}

teardown_plugin_tree() {
  chmod -R u+w "$WORKDIR"
  rm -rf "$WORKDIR"
}

# mock_cmd NAME -> 何もしない実行ファイルを PATH に置く
mock_cmd() {
  printf '#!/bin/bash\nexit 0\n' >"$MOCK_PATH/$1"
  chmod +x "$MOCK_PATH/$1"
}

# mock_go [ERRORS] -> `go build -C ROOT -o OUT ./...` を受け、OUT/writing-gate に
# scan の結果として errors=ERRORS の固定 JSON を返す偽バイナリを置く。OUT が相対なら ROOT から解決する。
# 呼び出しは go.log に記録する。
# shellcheck disable=SC2016 # 生成するスクリプトの中で展開させるため、$ はそのまま書き出す
mock_go() {
  local errors="${1:-1}"
  {
    printf '#!/bin/bash\n'
    printf "printf '%%s\\\\n' '{\"errors\":%s,\"warnings\":0,\"findings\":[]}'\n" "$errors"
  } >"$WORKDIR/fake-gate"
  {
    printf '#!/bin/bash\n'
    printf 'printf "%%s\\n" "$*" >>"%s/go.log"\n' "$WORKDIR"
    printf 'root=""; out=""\n'
    printf 'while [ "$#" -gt 0 ]; do\n'
    printf '  case "$1" in\n'
    printf '    -C) root="$2"; shift 2 ;;\n'
    printf '    -o) out="$2"; shift 2 ;;\n'
    printf '    *) shift ;;\n'
    printf '  esac\n'
    printf 'done\n'
    printf 'case "$out" in /*) ;; *) out="$root/$out" ;; esac\n'
    printf 'mkdir -p "$out"\n'
    printf 'cp "%s/fake-gate" "$out/writing-gate"\n' "$WORKDIR"
    printf 'chmod +x "$out/writing-gate"\n'
  } >"$MOCK_PATH/go"
  chmod +x "$MOCK_PATH/go"
}

# go_calls -> go が呼ばれた回数
go_calls() {
  if [ -f "$WORKDIR/go.log" ]; then
    wc -l <"$WORKDIR/go.log" | tr -d ' '
  else
    printf '0\n'
  fi
}
