#!/bin/bash
set -euo pipefail
# session-start-prepare.sh
# SessionStart hook: plugin の data ディレクトリに writing-gate と crit の prompt を用意する。
#
# plugin を update すると置き場のディレクトリが版ごとに変わる。data は update をまたいで
# 残るので、ここで入力の差分を見てビルドし直せば、利用者が setup を打ち直す必要がない。
# hook は Bash tool の sandbox の外で動くため、~/.claude/plugins/data へ書ける。
#
# 成功時は何も出さない (SessionStart の stdout は会話の文脈へ足される)。
# 失敗してもセッションは止めず、systemMessage で理由と setup を案内する。
# ゲートが無いままなら Stop hook が完了を止めるので、黙って通ることはない。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 入力は使わないが、パイプを詰まらせないよう読み捨てる
cat >/dev/null

if ! out=$("$SCRIPT_DIR/../scripts/prepare-data.sh" --if-stale 2>&1); then
  jq -cn --arg o "$out" \
    '{"systemMessage": ("japanese-writer: writing-gate の準備に失敗しました｡/japanese-writer:setup を実行して詳細を確認してください｡\n" + $o)}'
fi
exit 0
